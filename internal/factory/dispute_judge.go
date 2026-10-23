package factory

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"
)

type disputeCancellation struct{ cancel context.CancelFunc }

func (s *Service) registerDisputeCancel(id ID, cancel context.CancelFunc) func() {
	registration := &disputeCancellation{cancel: cancel}
	s.state.disputeCancels.Store(id, registration)
	return func() { s.state.disputeCancels.CompareAndDelete(id, registration) }
}

func (s *Service) runDisputeJudge(id ID) {
	defer s.state.refiners.Done()
	d, err := loadDispute(s.state.background, s.state.db, id)
	if err != nil || d.Status != "judging" {
		return
	}
	job, err := loadJob(s.state.background, s.state.db, d.JobID)
	if err != nil {
		return
	}
	deadline := s.state.now().Add(5 * time.Minute)
	if job.Admission.Deadline.Before(deadline) {
		deadline = job.Admission.Deadline
	}
	ctx, cancel := context.WithDeadline(s.state.background, deadline)
	defer cancel()
	defer s.registerDisputeCancel(id, cancel)()
	if latest, err := loadDispute(ctx, s.state.db, id); err != nil || latest.Status != "judging" || latest.Version != d.Version {
		return
	}
	project, err := readProject(ctx, s.state.db, d.ProjectID)
	input := JudgeInput{Dispute: d, Job: job, ProtectedPaths: job.Admission.ProtectedPaths, TestAuthorAvailable: s.canWriteExpectations(project), Instructions: "Assess this frozen contract independently. Never edit it. Intent changes and weaker acceptance require the person."}
	if err == nil && job.Admission.ExpectationsID != "" {
		var e Expectation
		e, err = loadExpectation(ctx, s.state.db, job.Admission.ExpectationsID)
		input.OriginalTests = &e
	}
	if err == nil {
		input.TestSources, err = s.disputeTestSources(ctx, project, job.Admission)
	}
	if err == nil && job.CandidateRevision != "" {
		input.CandidatePatch, err = s.git(ctx, project.RepositoryPath, "diff", "--no-ext-diff", "--no-textconv", "--no-renames", job.Admission.BaseRevision, job.CandidateRevision, "--")
		if len(input.CandidatePatch) > maxExpectationPatch {
			err = fmt.Errorf("candidate patch exceeds the judge input limit; human review required")
		}
	}
	var output JudgeOutput
	var log string
	if err == nil {
		output, log, err = s.execDisputeJudge(ctx, project.RepositoryPath, input)
	}
	s.state.mutations.Lock()
	defer s.state.mutations.Unlock()
	tx, e := s.state.db.BeginTx(context.Background(), nil)
	if e != nil {
		return
	}
	defer tx.Rollback()
	current, e := loadDispute(context.Background(), tx, id)
	if e != nil || current.Status != "judging" || current.Version != d.Version {
		return
	}
	now := s.state.now().UTC()
	current.Version++
	current.UpdatedAt = now
	current.Log = log
	job, project, e = s.disputeRouteReady(context.Background(), tx, current)
	var authorInput *ExpectationInput
	if e != nil {
		current.Status, current.NextOwner, current.Error = "stopped", "none", e.Error()
	} else if err != nil {
		current.Status, current.NextOwner, current.Error = "needs_human", "you", truncateRunes(err.Error(), maxRefinementRunes)
	} else {
		current.Decision, current.Run = &output.Decision, &output.Run
		decision := output.Decision
		var cycles int
		if e = tx.QueryRowContext(context.Background(), `SELECT count(*) FROM disputes WHERE lineage_id=?`, current.LineageID).Scan(&cycles); e != nil {
			return
		}
		if !decision.PreservesIntent || decision.WeakensAcceptance || decision.Kind == DisputeNeedsHuman || cycles > 1 {
			current.Status, current.NextOwner = "needs_human", "you"
		} else {
			// Routing is still attributed to the original coordinator and rechecks their
			// current access. A worker's report never grants coordination authority.
			coordinator, e := authorizeProject(context.Background(), tx, Principal{UserID: job.CoordinatorID}, project.ID, CapabilityCoordinate)
			if e == nil {
				if _, e = tx.ExecContext(context.Background(), "SAVEPOINT dispute_route"); e != nil {
					return
				}
				switch decision.Kind {
				case DisputeRepair:
					e = s.repairDispute(context.Background(), tx, &current, job, now)
				case DisputeReviseTests, DisputeAddTests:
					authorInput, e = s.prepareDisputeTests(context.Background(), tx, &current, job, project, coordinator.UserID, now)
				}
				if e != nil {
					_, _ = tx.ExecContext(context.Background(), "ROLLBACK TO dispute_route")
					authorInput = nil
				}
				if _, releaseErr := tx.ExecContext(context.Background(), "RELEASE dispute_route"); releaseErr != nil {
					return
				}
			}
			if e != nil {
				current.Status, current.NextOwner, current.Error = "needs_human", "you", e.Error()
			}
		}
	}
	if saveDispute(context.Background(), tx, current, false) != nil {
		return
	}
	if _, e = appendEvent(context.Background(), tx, current.JobID, nil, "dispute_judged", map[string]any{"disputeId": id, "decision": current.Decision, "nextOwner": current.NextOwner, "error": current.Error}, now); e != nil {
		return
	}
	if tx.Commit() != nil {
		return
	}
	if authorInput != nil {
		s.state.refiners.Add(1)
		go s.runExpectationAuthor(authorInput.ExpectationID, *authorInput, project)
	}
}

func (s *Service) execDisputeJudge(ctx context.Context, directory string, input JudgeInput) (JudgeOutput, string, error) {
	if len(s.state.judge) == 0 {
		return JudgeOutput{}, "", fmt.Errorf("no independent judge is configured")
	}
	encoded, err := json.Marshal(input)
	if err != nil {
		return JudgeOutput{}, "", err
	}
	command := exec.CommandContext(ctx, s.state.judge[0], s.state.judge[1:]...)
	command.Dir = directory
	command.Stdin = bytes.NewReader(encoded)
	command.WaitDelay = 5 * time.Second
	out, log := &limitedBuffer{limit: maxRefinerOutput}, &limitedBuffer{limit: maxRefinerLog}
	command.Stdout, command.Stderr = out, log
	if err = command.Run(); err != nil {
		return JudgeOutput{}, log.String(), fmt.Errorf("judge stopped or failed: %w", err)
	}
	if out.truncated {
		return JudgeOutput{}, log.String(), fmt.Errorf("judge response exceeds the output limit")
	}
	var output JudgeOutput
	decoder := json.NewDecoder(bytes.NewReader(out.Bytes()))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&output); err != nil {
		return output, log.String(), fmt.Errorf("invalid judge response: %w", err)
	}
	if err = decoder.Decode(new(any)); err != io.EOF {
		return output, log.String(), fmt.Errorf("judge response contains trailing data")
	}
	var flags struct {
		Decision struct {
			PreservesIntent   *bool `json:"preservesIntent"`
			WeakensAcceptance *bool `json:"weakensAcceptance"`
		} `json:"decision"`
	}
	if err = json.Unmarshal(out.Bytes(), &flags); err != nil || flags.Decision.PreservesIntent == nil || flags.Decision.WeakensAcceptance == nil {
		return output, log.String(), fmt.Errorf("judge must explicitly assess intent preservation and weakened acceptance")
	}
	switch output.Decision.Kind {
	case DisputeRepair, DisputeReviseTests, DisputeAddTests, DisputeNeedsHuman:
	default:
		return output, log.String(), fmt.Errorf("unknown judge decision")
	}
	if strings.TrimSpace(output.Decision.Rationale) == "" || len(output.Decision.Rationale) > 16000 || len(output.Decision.Directive) > 16000 || len(output.Decision.EvidenceRefs) == 0 || len(output.Decision.EvidenceRefs) > 20 {
		return output, log.String(), fmt.Errorf("judge decision needs bounded rationale and evidence")
	}
	if output.Decision.Kind != DisputeNeedsHuman && strings.TrimSpace(output.Decision.Directive) == "" {
		return output, log.String(), fmt.Errorf("routed judge decisions require a directive")
	}
	for _, ref := range output.Decision.EvidenceRefs {
		if strings.TrimSpace(ref) == "" || len(ref) > 16000 {
			return output, log.String(), fmt.Errorf("invalid judge evidence reference")
		}
	}
	return output, log.String(), nil
}

func (s *Service) disputeTestSources(ctx context.Context, project Project, admission Admission) ([]ContextSource, error) {
	paths, err := s.git(ctx, project.RepositoryPath, "ls-tree", "-r", "--name-only", "-z", admission.BaseRevision)
	if err != nil {
		return nil, err
	}
	result := []ContextSource{}
	total := 0
	for _, path := range strings.Split(paths, "\x00") {
		if path == "" {
			continue
		}
		protected := false
		for _, pattern := range admission.ProtectedPaths {
			if protectedMatch(pattern, path) {
				protected = true
				break
			}
		}
		if !protected {
			continue
		}
		command := exec.CommandContext(ctx, s.state.gitBinary, "-C", project.RepositoryPath, "show", admission.BaseRevision+":"+path)
		buffer := &limitedBuffer{limit: 2 << 20}
		command.Stdout = buffer
		err := command.Run()
		raw := buffer.String()
		if err != nil {
			return nil, err
		}
		if buffer.truncated {
			return nil, fmt.Errorf("protected test snapshot exceeds the judge input limit; human review required")
		}
		total += len(raw)
		if len(result) >= 128 || total > 2<<20 {
			return nil, fmt.Errorf("protected test snapshot exceeds the judge input limit; human review required")
		}
		result = append(result, ContextSource{Path: path, Text: raw, SHA256: digestBytes([]byte(raw))})
	}
	return result, nil
}

func stopDisputes(ctx context.Context, tx *sql.Tx, jobID JobID, now time.Time) ([]ID, error) {
	rows, err := tx.QueryContext(ctx, `SELECT data_json FROM disputes WHERE job_id=? AND status NOT IN ('resolved','stopped')`, jobID)
	if err != nil {
		return nil, err
	}
	var records []Dispute
	for rows.Next() {
		var raw string
		if err = rows.Scan(&raw); err != nil {
			rows.Close()
			return nil, err
		}
		var d Dispute
		if err = json.Unmarshal([]byte(raw), &d); err != nil {
			rows.Close()
			return nil, err
		}
		records = append(records, d)
	}
	rows.Close()
	ids := []ID{}
	for _, d := range records {
		d.Version++
		d.Status, d.NextOwner, d.UpdatedAt = "stopped", "none", now
		if err = saveDispute(ctx, tx, d, false); err != nil {
			return nil, err
		}
		ids = append(ids, d.ID)
	}
	return ids, nil
}
