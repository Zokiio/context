package factory

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

func loadDispute(ctx context.Context, q queryer, id ID) (Dispute, error) {
	var raw string
	if err := q.QueryRowContext(ctx, `SELECT data_json FROM disputes WHERE id=?`, id).Scan(&raw); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Dispute{}, fault(CodeNotFound, "dispute not found", nil)
		}
		return Dispute{}, fault(CodeInternal, "read dispute", err)
	}
	var d Dispute
	if err := json.Unmarshal([]byte(raw), &d); err != nil {
		return d, fault(CodeInternal, "decode dispute", err)
	}
	return d, nil
}
func latestJobDispute(ctx context.Context, q queryer, jobID JobID) (*Dispute, error) {
	var id ID
	err := q.QueryRowContext(ctx, `SELECT id FROM disputes WHERE job_id=? ORDER BY created_at DESC,id DESC LIMIT 1`, jobID).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	d, err := loadDispute(ctx, q, id)
	return &d, err
}
func saveDispute(ctx context.Context, tx *sql.Tx, d Dispute, insert bool) error {
	d.AvailableActions = nil
	raw, err := marshal(d)
	if err != nil {
		return err
	}
	if insert {
		_, err = tx.ExecContext(ctx, `INSERT INTO disputes(id,project_id,work_item_id,job_id,lineage_id,status,data_json,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?)`, d.ID, d.ProjectID, d.WorkItemID, d.JobID, d.LineageID, d.Status, raw, formatTime(d.CreatedAt), formatTime(d.UpdatedAt))
	} else {
		_, err = tx.ExecContext(ctx, `UPDATE disputes SET status=?,data_json=?,updated_at=? WHERE id=?`, d.Status, raw, formatTime(d.UpdatedAt), d.ID)
	}
	if err != nil {
		return fault(CodeInternal, "save dispute", err)
	}
	return nil
}
func (s *Service) disputeActions(ctx context.Context, q queryer, actor Principal, d Dispute) Dispute {
	d.AvailableActions = []string{}
	current, err := authorizeProject(ctx, q, actor, d.ProjectID, CapabilityCoordinate)
	if err != nil || current.Kind != UserKindHuman {
		return d
	}
	if held, err := workItemHeld(ctx, q, d.ProjectID, d.WorkItemID); err != nil || held {
		return d
	}
	switch d.Status {
	case "needs_human":
		d.AvailableActions = []string{"repair_implementation", "return_to_planning", "dismiss"}
		if d.ExpectationID != "" {
			if proposal, err := loadExpectation(ctx, q, d.ExpectationID); err == nil && proposal.Status == ExpectationProposed {
				d.AvailableActions = append(d.AvailableActions, "accept_tests")
				if len(s.state.judge) > 0 {
					d.AvailableActions = append(d.AvailableActions, "review_tests")
				}
			}
		} else if len(s.state.judge) > 0 {
			d.AvailableActions = append(d.AvailableActions, "retry_judge")
		}
		if len(s.state.expectationAuthor) > 0 && d.TestProposalCount < maxDisputeTestProposals {
			d.AvailableActions = append(d.AvailableActions, "revise_tests", "add_tests")
		}
	case "tests_proposed":
		d.AvailableActions = []string{"accept_tests", "return_to_planning", "dismiss"}
		if len(s.state.judge) > 0 {
			d.AvailableActions = append(d.AvailableActions, "review_tests")
		}
	}
	return d
}
func (s *Service) GetDispute(ctx context.Context, actor Principal, id ID) (Dispute, error) {
	d, err := loadDispute(ctx, s.state.db, id)
	if err != nil {
		return d, err
	}
	if _, err = authorizeProject(ctx, s.state.db, actor, d.ProjectID, CapabilityRead); err != nil {
		return Dispute{}, err
	}
	return s.disputeActions(ctx, s.state.db, actor, d), nil
}
func (s *Service) ListDisputes(ctx context.Context, actor Principal, projectID ProjectID, workItemID string) ([]Dispute, error) {
	if _, err := authorizeProject(ctx, s.state.db, actor, projectID, CapabilityRead); err != nil {
		return nil, err
	}
	records, _, err := activityRecords[Dispute](ctx, s.state.db, "disputes", projectID, workItemID)
	if err != nil {
		return nil, err
	}
	for i := range records {
		records[i] = s.disputeActions(ctx, s.state.db, actor, records[i])
	}
	return records, nil
}
func noLiveDisputeAttempts(ctx context.Context, q queryer, jobID JobID) error {
	var n int
	err := q.QueryRowContext(ctx, `SELECT count(*) FROM attempts WHERE job_id=? AND status IN (?,?,?,?,?,?)`, jobID, AttemptPending, AttemptAssigned, AttemptStartRegistered, AttemptRunning, AttemptCancelAcknowledged, AttemptUnknown).Scan(&n)
	if err != nil {
		return fault(CodeInternal, "inspect disputed execution", err)
	}
	if n > 0 {
		return fault(CodeUncertain, "execution must be conclusively stopped before a dispute can route work", nil)
	}
	return nil
}
func (s *Service) newDispute(ctx context.Context, tx *sql.Tx, actor UserID, job Job, input DisputeInput, now time.Time) (Dispute, error) {
	if err := ValidateDisputeInput(input); err != nil {
		return Dispute{}, err
	}
	if err := noLiveDisputeAttempts(ctx, tx, job.ID); err != nil {
		return Dispute{}, err
	}
	held, err := workItemHeld(ctx, tx, job.ProjectID, job.WorkItemID)
	if err != nil {
		return Dispute{}, err
	}
	if held {
		return Dispute{}, fault(CodeConflict, "stopped work cannot start a dispute", nil)
	}
	lineage := job.Admission.LineageID
	if lineage == "" {
		lineage = job.ID
	}
	var count int
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM disputes WHERE lineage_id=?`, lineage).Scan(&count); err != nil {
		return Dispute{}, err
	}
	id, err := newID("dspte_")
	if err != nil {
		return Dispute{}, err
	}
	d := Dispute{ID: ID(id), ProjectID: job.ProjectID, WorkItemID: job.WorkItemID, JobID: job.ID, LineageID: lineage, Version: 1, Status: "judging", Input: input, NextOwner: "judge", RequestedBy: actor, CreatedAt: now, UpdatedAt: now}
	if count > 0 {
		d.Status, d.NextOwner, d.Error = "needs_human", "you", "This lineage already used its automatic dispute cycle."
	} else if len(s.state.judge) == 0 {
		d.Status, d.NextOwner, d.Error = "needs_human", "you", "No independent judge is configured."
	}
	if _, err = tx.ExecContext(ctx, `UPDATE jobs SET status=?,active_claim=1,version=version+1,updated_at=? WHERE id=?`, JobDisputed, formatTime(now), job.ID); err != nil {
		return d, err
	}
	if err = dropConsent(ctx, tx, job.ProjectID, job.WorkItemID); err != nil {
		return d, err
	}
	if err = saveDispute(ctx, tx, d, true); err != nil {
		return d, err
	}
	_, err = appendEvent(ctx, tx, job.ID, &actor, "dispute_reported", map[string]any{"disputeId": d.ID, "summary": input.Summary, "nextOwner": d.NextOwner}, now)
	return d, err
}
func (s *Service) ReportDispute(ctx context.Context, actor Principal, r ReportDisputeRequest) (Dispute, error) {
	if r.JobID == "" || r.IfVersion <= 0 {
		return Dispute{}, fault(CodeInvalid, "jobId and ifVersion are required", nil)
	}
	if err := requireRequestID(r.RequestID); err != nil {
		return Dispute{}, err
	}
	if err := ValidateDisputeInput(r.Dispute); err != nil {
		return Dispute{}, err
	}
	s.state.mutations.Lock()
	defer s.state.mutations.Unlock()
	tx, err := s.state.db.BeginTx(ctx, nil)
	if err != nil {
		return Dispute{}, err
	}
	defer tx.Rollback()
	job, err := loadJob(ctx, tx, r.JobID)
	if err != nil {
		return Dispute{}, err
	}
	current, err := authorizeProject(ctx, tx, actor, job.ProjectID, CapabilityCoordinate)
	if err != nil {
		return Dispute{}, err
	}
	if current.Kind != UserKindHuman {
		return Dispute{}, fault(CodeForbidden, "a coordinator must request a dispute; workers report one with terminal execution", nil)
	}
	saved, ok, hash, err := receipt[Dispute](ctx, tx, current.UserID, "report_dispute", r.RequestID, r)
	if err != nil || ok {
		return saved, err
	}
	if job.Version != r.IfVersion {
		return Dispute{}, fault(CodeConflict, "job version changed", nil)
	}
	switch job.Status {
	case JobFailed, JobVerificationFailed, JobReadyForReview, JobReadyForAcceptance:
	default:
		return Dispute{}, fault(CodeConflict, "only failed or reviewable jobs can be disputed", nil)
	}
	d, err := s.newDispute(ctx, tx, current.UserID, job, r.Dispute, s.state.now().UTC())
	if err != nil {
		return d, err
	}
	d = s.disputeActions(ctx, tx, current, d)
	if err = saveReceipt(ctx, tx, current.UserID, "report_dispute", r.RequestID, hash, d, false, d.CreatedAt); err != nil {
		return d, err
	}
	if err = tx.Commit(); err != nil {
		return d, err
	}
	s.launchJudge(d)
	return d, nil
}
func (s *Service) launchJudge(d Dispute) {
	if d.Status == "judging" || d.Status == "reviewing_tests" {
		s.state.refiners.Add(1)
		go s.runDisputeJudge(d.ID)
	}
}

func (s *Service) ResolveDispute(ctx context.Context, actor Principal, r ResolveDisputeRequest) (Dispute, error) {
	if r.DisputeID == "" || r.IfVersion <= 0 || strings.TrimSpace(r.Rationale) == "" || len(r.Rationale) > 16000 {
		return Dispute{}, fault(CodeInvalid, "disputeId, ifVersion and bounded rationale are required", nil)
	}
	if err := requireRequestID(r.RequestID); err != nil {
		return Dispute{}, err
	}
	s.state.mutations.Lock()
	defer s.state.mutations.Unlock()
	tx, err := s.state.db.BeginTx(ctx, nil)
	if err != nil {
		return Dispute{}, err
	}
	defer tx.Rollback()
	d, err := loadDispute(ctx, tx, r.DisputeID)
	if err != nil {
		return d, err
	}
	current, err := authorizeProject(ctx, tx, actor, d.ProjectID, CapabilityCoordinate)
	if err != nil {
		return Dispute{}, err
	}
	if current.Kind != UserKindHuman {
		return Dispute{}, fault(CodeForbidden, "resolving a dispute requires a human principal", nil)
	}
	saved, ok, hash, err := receipt[Dispute](ctx, tx, current.UserID, "resolve_dispute", r.RequestID, r)
	if err != nil || ok {
		return saved, err
	}
	if d.Version != r.IfVersion {
		return Dispute{}, fault(CodeConflict, "dispute version changed", nil)
	}
	allowed := s.disputeActions(ctx, tx, current, d)
	if !contains(allowed.AvailableActions, r.Action) {
		return Dispute{}, fault(CodeConflict, "this dispute action is unavailable", nil)
	}
	job, project, err := s.disputeRouteReady(ctx, tx, d)
	if err != nil {
		return d, err
	}
	now := s.state.now().UTC()
	d.ResolvedBy, d.Resolution, d.Rationale = &current.UserID, r.Action, r.Rationale
	d.Error, d.Delegated = "", ""
	var input *ExpectationInput
	switch r.Action {
	case "retry_judge":
		d.Status, d.NextOwner = "judging", "judge"
	case "review_tests":
		err = s.prepareDisputeReview(ctx, tx, &d, now)
	case "repair_implementation":
		err = s.repairDispute(ctx, tx, &d, job, now)
	case "revise_tests", "add_tests":
		input, err = s.prepareDisputeTests(ctx, tx, &d, job, project, current.UserID, now)
	case "accept_tests":
		err = s.replaceDisputedJob(ctx, tx, &d, job, project, current.UserID, now)
	case "dismiss", "return_to_planning":
		status := JobCancelled
		if r.Action == "return_to_planning" {
			status = JobSuperseded
		}
		_, err = tx.ExecContext(ctx, `UPDATE jobs SET status=?,active_claim=0,version=version+1,updated_at=? WHERE id=?`, status, formatTime(now), job.ID)
		if err == nil {
			err = holdWorkItem(ctx, tx, job.ProjectID, job.WorkItemID, now)
		}
		d.Status, d.NextOwner = "resolved", "you"
	}
	if err != nil {
		return d, err
	}
	d.Version++
	d.UpdatedAt = now
	if err = saveDispute(ctx, tx, d, false); err != nil {
		return d, err
	}
	if _, err = appendEvent(ctx, tx, d.JobID, &current.UserID, "dispute_resolved", map[string]any{"disputeId": d.ID, "action": r.Action, "rationale": r.Rationale, "nextOwner": d.NextOwner, "replacementJobId": d.ReplacementJobID}, now); err != nil {
		return d, err
	}
	d = s.disputeActions(ctx, tx, current, d)
	if err = saveReceipt(ctx, tx, current.UserID, "resolve_dispute", r.RequestID, hash, d, false, now); err != nil {
		return d, err
	}
	if err = tx.Commit(); err != nil {
		return d, err
	}
	s.launchJudge(d)
	if input != nil {
		s.state.refiners.Add(1)
		go s.runExpectationAuthor(input.ExpectationID, *input, project)
	}
	return d, nil
}
func (s *Service) disputeRouteReady(ctx context.Context, q queryer, d Dispute) (Job, Project, error) {
	job, err := loadJob(ctx, q, d.JobID)
	if err != nil {
		return job, Project{}, err
	}
	project, err := readProject(ctx, q, d.ProjectID)
	if err != nil {
		return job, project, err
	}
	if job.Status != JobDisputed {
		return job, project, fault(CodeConflict, "the disputed job is no longer awaiting a decision", nil)
	}
	if err = noLiveDisputeAttempts(ctx, q, job.ID); err != nil {
		return job, project, err
	}
	if held, e := workItemHeld(ctx, q, d.ProjectID, d.WorkItemID); e != nil || held {
		if e != nil {
			return job, project, e
		}
		return job, project, fault(CodeConflict, "the task is stopped", nil)
	}
	return job, project, nil
}
func (s *Service) repairDispute(ctx context.Context, tx *sql.Tx, d *Dispute, job Job, now time.Time) error {
	job.Status = JobFailed
	if err := s.repairJob(ctx, tx, job, now); err != nil {
		return err
	}
	d.Status, d.NextOwner = "resolved", "implementer"
	return nil
}
func recoverDisputes(ctx context.Context, db *sql.DB, now time.Time) error {
	rows, err := db.QueryContext(ctx, `SELECT data_json FROM disputes WHERE status IN ('judging','reviewing_tests','authoring_tests','routing')`)
	if err != nil {
		return err
	}
	var records []Dispute
	for rows.Next() {
		var raw string
		if err = rows.Scan(&raw); err != nil {
			rows.Close()
			return err
		}
		var d Dispute
		if err = json.Unmarshal([]byte(raw), &d); err != nil {
			rows.Close()
			return err
		}
		records = append(records, d)
	}
	if err = rows.Close(); err != nil {
		return err
	}
	for _, d := range records {
		d.Version++
		if d.Status == "reviewing_tests" && len(d.TestReviews) > 0 {
			review := &d.TestReviews[len(d.TestReviews)-1]
			review.NextOwner, review.Error = "you", "The service stopped during test proposal review."
		}
		d.Status, d.NextOwner, d.Error, d.UpdatedAt = "needs_human", "you", "The service stopped during this handoff. Review before retrying.", now
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		if err = saveDispute(ctx, tx, d, false); err == nil {
			err = tx.Commit()
		} else {
			_ = tx.Rollback()
		}
		if err != nil {
			return err
		}
	}
	return nil
}
