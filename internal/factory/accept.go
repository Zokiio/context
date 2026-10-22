package factory

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Zokiio/context/internal/factoryrecords"
	"github.com/Zokiio/context/internal/orientation"
)

type acceptanceOperation struct {
	ActorID     UserID              `json:"actorId"`
	RequestHash string              `json:"requestHash"`
	Request     AcceptRequest       `json:"request"`
	Plan        factoryrecords.Plan `json:"plan"`
}

type retainedAcceptanceEvidence struct {
	SchemaVersion          int           `json:"schemaVersion"`
	JobID                  JobID         `json:"jobId"`
	ProjectID              ProjectID     `json:"projectId"`
	WorkItemID             string        `json:"workItemId"`
	CandidateRevision      string        `json:"candidateRevision"`
	RequirementFingerprint string        `json:"requirementFingerprint"`
	CriteriaFingerprint    string        `json:"criteriaFingerprint"`
	AdmissionContext       string        `json:"admissionContextSHA256"`
	BaseRevision           string        `json:"baseRevision"`
	Checks                 []CheckResult `json:"checks"`
	Review                 Review        `json:"review"`
	AcceptedBy             UserID        `json:"acceptedBy"`
	Summary                string        `json:"summary"`
	ObservedAt             time.Time     `json:"observedAt"`
}

func (s *Service) Accept(ctx context.Context, actor Principal, request AcceptRequest) (Job, error) {
	if request.JobID == "" || request.IfVersion <= 0 || request.CandidateRevision == "" || request.RequirementFingerprint == "" || strings.TrimSpace(request.Summary) == "" {
		return Job{}, fault(CodeInvalid, "jobId, ifVersion, candidateRevision, requirementFingerprint, and summary are required", nil)
	}
	s.state.mutations.Lock()
	defer s.state.mutations.Unlock()

	tx, err := s.state.db.BeginTx(ctx, nil)
	if err != nil {
		return Job{}, fault(CodeInternal, "start acceptance", err)
	}
	defer tx.Rollback()
	projectID, err := jobProject(ctx, tx, request.JobID)
	if err != nil {
		return Job{}, err
	}
	current, err := authorizeProject(ctx, tx, actor, projectID, CapabilityAccept)
	if err != nil {
		return Job{}, err
	}
	if current.Kind != UserKindHuman {
		return Job{}, fault(CodeForbidden, "acceptance requires a human principal", nil)
	}
	if saved, ok, hash, err := receipt[Job](ctx, tx, current.UserID, "accept", request.RequestID, request); err != nil || ok {
		return saved, err
	} else {
		operation, resumed, err := s.prepareAcceptance(ctx, tx, current, request, hash)
		if err != nil {
			return Job{}, err
		}
		if err := tx.Commit(); err != nil {
			return Job{}, fault(CodeInternal, "commit acceptance intent", err)
		}

		var publication factoryrecords.Result
		if resumed {
			publication, err = factoryrecords.Reconcile(ctx, operation.Plan)
		} else {
			publication, err = factoryrecords.Publish(ctx, operation.Plan)
		}
		if err != nil {
			s.recordAcceptanceFailure(ctx, request.JobID, err)
			code := CodeUncertain
			if errors.Is(err, factoryrecords.ErrConflict) {
				code = CodeStale
			}
			return Job{}, detailFault(code, "acceptance publication requires reconciliation", map[string]any{"jobId": request.JobID}, err)
		}

		project, projectErr := readProject(ctx, s.state.db, ProjectID(operation.Plan.ProjectID))
		if projectErr != nil {
			s.recordAcceptanceFailure(ctx, request.JobID, projectErr)
			return Job{}, projectErr
		}
		observed, err := orientation.Orient(ctx, orientation.Request{ProjectDir: operation.Plan.BundlePath, AllowedSourceDirs: operation.Plan.AllowedSourceRoots, MaxFiles: project.ReaderLimits.MaxFiles, MaxBytes: project.ReaderLimits.MaxBytes})
		if err != nil {
			s.recordAcceptanceFailure(ctx, request.JobID, err)
			return Job{}, fault(CodeUncertain, "read acceptance after publication", err)
		}
		accepted, err := selectWork(observed, operation.Plan.WorkItemID)
		if err != nil || accepted.Acceptance == nil || accepted.Acceptance.Status != "valid" || accepted.Acceptance.Record == nil || accepted.Acceptance.Record.Path != publication.AcceptancePath {
			if err == nil {
				err = errors.New("authoritative reader did not validate the published acceptance")
			}
			s.recordAcceptanceFailure(ctx, request.JobID, err)
			return Job{}, detailFault(CodeUncertain, "published acceptance is not authoritative", map[string]any{"acceptancePath": publication.AcceptancePath}, err)
		}
		return s.finishAcceptance(ctx, current, request, hash, publication)
	}
}

func (s *Service) prepareAcceptance(ctx context.Context, tx *sql.Tx, actor Principal, request AcceptRequest, hash string) (acceptanceOperation, bool, error) {
	var encoded string
	err := tx.QueryRowContext(ctx, `SELECT plan_json FROM acceptance_writes WHERE job_id=?`, request.JobID).Scan(&encoded)
	if err == nil {
		var operation acceptanceOperation
		if err := json.Unmarshal([]byte(encoded), &operation); err != nil {
			return acceptanceOperation{}, false, fault(CodeInternal, "decode acceptance intent", err)
		}
		if operation.ActorID != actor.UserID || operation.RequestHash != hash {
			return acceptanceOperation{}, false, fault(CodeConflict, "job already has a different acceptance publication intent", nil)
		}
		return operation, true, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return acceptanceOperation{}, false, fault(CodeInternal, "read acceptance intent", err)
	}
	job, err := loadJob(ctx, tx, request.JobID)
	if err != nil {
		return acceptanceOperation{}, false, err
	}
	if job.Version != request.IfVersion {
		return acceptanceOperation{}, false, detailFault(CodeConflict, "job version changed", map[string]any{"currentVersion": job.Version}, nil)
	}
	if job.Status != JobReadyForAcceptance {
		return acceptanceOperation{}, false, fault(CodeNotReady, "job is not ready for acceptance", nil)
	}
	if job.CandidateRevision != request.CandidateRevision || job.Admission.TicketSHA256 != request.RequirementFingerprint {
		return acceptanceOperation{}, false, fault(CodeStale, "acceptance does not match the current candidate and requirements", nil)
	}
	if err := checkSourcesUnchanged(job.Admission.Sources); err != nil {
		return acceptanceOperation{}, false, err
	}
	if !allChecksPassed(job) {
		return acceptanceOperation{}, false, fault(CodeNotReady, "current required checks are missing, skipped, unknown, stale, or failed", nil)
	}
	review, ok := currentApprovedReview(job)
	if !ok {
		return acceptanceOperation{}, false, fault(CodeNotReady, "an independent current approval is required", nil)
	}
	project, err := readProject(ctx, tx, job.ProjectID)
	if err != nil {
		return acceptanceOperation{}, false, err
	}
	resolved, err := s.resolveCommit(ctx, project.RepositoryPath, job.CandidateRevision)
	if err != nil || resolved != job.CandidateRevision {
		return acceptanceOperation{}, false, fault(CodeStale, "candidate commit is unavailable", err)
	}
	result, err := orientation.Orient(ctx, orientation.Request{ProjectDir: project.BundlePath, AllowedSourceDirs: project.AllowedSourceRoots, MaxFiles: project.ReaderLimits.MaxFiles, MaxBytes: project.ReaderLimits.MaxBytes})
	if err != nil {
		return acceptanceOperation{}, false, fault(CodeInternal, "evaluate acceptance requirements", err)
	}
	work, err := selectWork(result, job.WorkItemID)
	if err != nil {
		return acceptanceOperation{}, false, err
	}
	if work.FingerprintVersion == nil || *work.FingerprintVersion != job.Admission.FingerprintVersion || work.TicketSHA256 == nil || *work.TicketSHA256 != job.Admission.TicketSHA256 || work.CriteriaSHA256 == nil || *work.CriteriaSHA256 != job.Admission.CriteriaSHA256 || work.Source != job.Admission.TicketPath {
		return acceptanceOperation{}, false, fault(CodeStale, "authoritative requirements changed after admission", nil)
	}
	if work.Readiness != "ready" || !work.Eligible {
		return acceptanceOperation{}, false, fault(CodeNotReady, "work item is no longer ready and eligible", nil)
	}
	expectedTicket, err := admissionTicket(job.Admission)
	if err != nil {
		return acceptanceOperation{}, false, err
	}
	now := s.state.now().UTC()
	acceptancePath := filepath.Join(project.BundlePath, "acceptances", "factory-"+string(job.ID)+".md")
	evidencePath, err := acceptanceEvidencePath(project, job.ID)
	if err != nil {
		return acceptanceOperation{}, false, err
	}
	evidence := retainedAcceptanceEvidence{SchemaVersion: 1, JobID: job.ID, ProjectID: job.ProjectID, WorkItemID: job.WorkItemID, CandidateRevision: job.CandidateRevision, RequirementFingerprint: job.Admission.TicketSHA256, CriteriaFingerprint: job.Admission.CriteriaSHA256, AdmissionContext: job.Admission.ContextSHA256, BaseRevision: job.Admission.BaseRevision, Checks: currentChecks(job), Review: review, AcceptedBy: actor.UserID, Summary: strings.TrimSpace(request.Summary), ObservedAt: now}
	evidenceBytes, err := json.MarshalIndent(evidence, "", "  ")
	if err != nil {
		return acceptanceOperation{}, false, fault(CodeInternal, "encode acceptance evidence", err)
	}
	evidenceBytes = append(evidenceBytes, '\n')
	acceptanceBytes, err := renderAcceptance(job, actor, now, acceptancePath, evidencePath, evidenceBytes)
	if err != nil {
		return acceptanceOperation{}, false, err
	}
	lockDigest := sha256.Sum256([]byte(job.Admission.TicketPath))
	plan := factoryrecords.Plan{BundlePath: project.BundlePath, AllowedSourceRoots: project.AllowedSourceRoots, ProjectID: string(project.ID), WorkItemID: job.WorkItemID, TicketPath: job.Admission.TicketPath, ExpectedTicket: expectedTicket, AcceptancePath: acceptancePath, AcceptanceBytes: acceptanceBytes, EvidencePath: evidencePath, EvidenceBytes: evidenceBytes, LockPath: filepath.Join(s.state.dataDir, "record-locks", hex.EncodeToString(lockDigest[:])+".lock")}
	operation := acceptanceOperation{ActorID: actor.UserID, RequestHash: hash, Request: request, Plan: plan}
	operationJSON, err := marshal(operation)
	if err != nil {
		return acceptanceOperation{}, false, fault(CodeInternal, "encode acceptance intent", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO acceptance_writes(job_id,plan_json,state,updated_at) VALUES(?,?,?,?)`, job.ID, operationJSON, "prepared", formatTime(now)); err != nil {
		return acceptanceOperation{}, false, fault(CodeInternal, "persist acceptance intent", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE jobs SET status=?,version=version+1,updated_at=? WHERE id=?`, JobAccepting, formatTime(now), job.ID); err != nil {
		return acceptanceOperation{}, false, fault(CodeInternal, "mark acceptance in progress", err)
	}
	if _, err := appendEvent(ctx, tx, job.ID, &actor.UserID, "acceptance_prepared", map[string]any{"acceptancePath": acceptancePath, "evidencePath": evidencePath}, now); err != nil {
		return acceptanceOperation{}, false, err
	}
	return operation, false, nil
}

func (s *Service) finishAcceptance(ctx context.Context, actor Principal, request AcceptRequest, hash string, publication factoryrecords.Result) (Job, error) {
	tx, err := s.state.db.BeginTx(ctx, nil)
	if err != nil {
		return Job{}, fault(CodeInternal, "start acceptance completion", err)
	}
	defer tx.Rollback()
	current, err := authorizeProject(ctx, tx, actor, mustJobProject(ctx, tx, request.JobID), CapabilityAccept)
	if err != nil {
		return Job{}, err
	}
	job, err := loadJob(ctx, tx, request.JobID)
	if err != nil {
		return Job{}, err
	}
	if job.Status != JobAccepting || job.CandidateRevision != request.CandidateRevision || job.Admission.TicketSHA256 != request.RequirementFingerprint {
		return Job{}, fault(CodeUncertain, "job changed while acceptance was published", nil)
	}
	now := s.state.now().UTC()
	if _, err := tx.ExecContext(ctx, `UPDATE acceptance_writes SET state='completed',error='',updated_at=? WHERE job_id=?`, formatTime(now), job.ID); err != nil {
		return Job{}, fault(CodeInternal, "complete acceptance intent", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE jobs SET status=?,active_claim=0,acceptance_record_path=?,acceptance_status='valid',version=version+1,updated_at=? WHERE id=?`, JobAccepted, publication.AcceptancePath, formatTime(now), job.ID); err != nil {
		return Job{}, fault(CodeInternal, "complete accepted job", err)
	}
	if _, err := appendEvent(ctx, tx, job.ID, &current.UserID, "acceptance_published", publication, now); err != nil {
		return Job{}, err
	}
	job, err = loadJob(ctx, tx, job.ID)
	if err != nil {
		return Job{}, err
	}
	job.AvailableActions = availableActions(job)
	if err := saveReceipt(ctx, tx, current.UserID, "accept", request.RequestID, hash, job, false, now); err != nil {
		return Job{}, err
	}
	if err := tx.Commit(); err != nil {
		return Job{}, fault(CodeInternal, "commit accepted job", err)
	}
	return job, nil
}

func (s *Service) recordAcceptanceFailure(ctx context.Context, jobID JobID, cause error) {
	_, _ = s.state.db.ExecContext(ctx, `UPDATE acceptance_writes SET state='uncertain',error=?,updated_at=? WHERE job_id=?`, cause.Error(), formatTime(s.state.now().UTC()), jobID)
}

func renderAcceptance(job Job, actor Principal, now time.Time, acceptancePath, evidencePath string, evidence []byte) ([]byte, error) {
	requirements := []string{}
	seen := map[string]bool{}
	for _, source := range job.Admission.Sources {
		if source.Path == job.Admission.TicketPath || seen[source.Path] || !directRequirement(source, job.Admission.TicketPath) {
			continue
		}
		seen[source.Path] = true
		requirements = append(requirements, fmt.Sprintf("- [Requirement](%s) `%s`", relativeMarkdown(acceptancePath, source.Path), source.SHA256))
	}
	if len(requirements) == 0 {
		requirements = append(requirements, "None")
	}
	digest := sha256.Sum256(evidence)
	text := "---\n" +
		"type: Acceptance\n" +
		"id: " + strconv.Quote("factory-acceptance-"+string(job.ID)) + "\n" +
		"title: " + strconv.Quote("Factory acceptance for "+job.WorkItemID) + "\n" +
		"projectId: " + strconv.Quote(string(job.ProjectID)) + "\n" +
		"workItemId: " + strconv.Quote(job.WorkItemID) + "\n" +
		"actor: {kind: workflow, identity: " + strconv.Quote(actor.Username) + "}\n" +
		"decidedAt: " + strconv.Quote(now.Format(time.RFC3339Nano)) + "\n" +
		"testedRevision: {origin: " + strconv.Quote("factory-job-"+string(job.ID)) + ", revision: " + strconv.Quote(job.CandidateRevision) + "}\n" +
		"fingerprintVersion: 1\n" +
		"ticketSHA256: " + job.Admission.TicketSHA256 + "\n" +
		"criteriaSHA256: " + job.Admission.CriteriaSHA256 + "\n" +
		"---\n\n## Requirements\n\n" + strings.Join(requirements, "\n") + "\n\n## Evidence\n\n" +
		fmt.Sprintf("- [Retained factory evidence](%s) `%s`\n", relativeMarkdown(acceptancePath, evidencePath), hex.EncodeToString(digest[:]))
	return []byte(text), nil
}

func directRequirement(source ContextSource, ticket string) bool {
	var reasons []struct {
		Kind string `json:"kind"`
		From string `json:"from"`
	}
	if json.Unmarshal(source.Reasons, &reasons) != nil {
		return false
	}
	for _, reason := range reasons {
		if reason.From == ticket && (reason.Kind == "spec" || reason.Kind == "context" || reason.Kind == "blocked_by_decision") {
			return true
		}
	}
	return false
}

func admissionTicket(admission Admission) ([]byte, error) {
	for _, source := range admission.Sources {
		if source.Path == admission.TicketPath {
			return []byte(source.Text), nil
		}
	}
	return nil, fault(CodeInternal, "admission ticket snapshot is unavailable", nil)
}

func acceptanceEvidencePath(project Project, jobID JobID) (string, error) {
	for _, root := range project.AllowedSourceRoots {
		candidate := filepath.Join(root, ".factory-evidence", string(project.ID), string(jobID)+".json")
		if !pathWithin(project.BundlePath, candidate) {
			return candidate, nil
		}
	}
	return "", fault(CodeNotReady, "project requires an allowed source root outside the record bundle for immutable evidence", nil)
}

func relativeMarkdown(from, target string) string {
	relative, err := filepath.Rel(filepath.Dir(from), target)
	if err != nil {
		relative = target
	}
	return (&url.URL{Path: filepath.ToSlash(relative)}).EscapedPath()
}

func pathWithin(root, path string) bool {
	relative, err := filepath.Rel(root, path)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) && !filepath.IsAbs(relative)
}

func currentApprovedReview(job Job) (Review, bool) {
	for index := len(job.Reviews) - 1; index >= 0; index-- {
		review := job.Reviews[index]
		if review.Decision == ReviewApproved && review.CandidateRevision == job.CandidateRevision && review.RequirementFingerprint == job.Admission.TicketSHA256 && review.ReviewerID != job.CoordinatorID && (job.ImplementerID == nil || review.ReviewerID != *job.ImplementerID) {
			return review, true
		}
	}
	return Review{}, false
}

func currentChecks(job Job) []CheckResult {
	result := []CheckResult{}
	for _, check := range job.Checks {
		if check.CandidateRevision == job.CandidateRevision && check.RequirementFingerprint == job.Admission.TicketSHA256 && check.Status == CheckPassed {
			result = append(result, check)
		}
	}
	return result
}

func mustJobProject(ctx context.Context, q queryer, jobID JobID) ProjectID {
	projectID, _ := jobProject(ctx, q, jobID)
	return projectID
}
