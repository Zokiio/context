package factory

import (
	"context"
	"strings"
)

func (s *Service) Review(ctx context.Context, actor Principal, request ReviewRequest) (Job, error) {
	if request.JobID == "" || request.IfVersion <= 0 || request.CandidateRevision == "" || request.RequirementFingerprint == "" || strings.TrimSpace(request.Summary) == "" {
		return Job{}, fault(CodeInvalid, "jobId, ifVersion, candidateRevision, requirementFingerprint, and summary are required", nil)
	}
	if request.Decision != ReviewApproved && request.Decision != ReviewChangesRequested {
		return Job{}, fault(CodeInvalid, "review decision must be approved or changes_requested", nil)
	}
	findings, err := normalizeFindings(request.Findings)
	if err != nil {
		return Job{}, err
	}
	findingsJSON, err := marshal(findings)
	if err != nil {
		return Job{}, fault(CodeInternal, "encode review findings", err)
	}
	s.state.mutations.Lock()
	defer s.state.mutations.Unlock()
	tx, err := s.state.db.BeginTx(ctx, nil)
	if err != nil {
		return Job{}, fault(CodeInternal, "start review", err)
	}
	defer tx.Rollback()
	projectID, err := jobProject(ctx, tx, request.JobID)
	if err != nil {
		return Job{}, err
	}
	current, err := authorizeProject(ctx, tx, actor, projectID, CapabilityReview)
	if err != nil {
		return Job{}, err
	}
	if current.Kind != UserKindHuman {
		return Job{}, fault(CodeForbidden, "review requires a human principal", nil)
	}
	if saved, ok, hash, err := receipt[Job](ctx, tx, current.UserID, "review", request.RequestID, request); err != nil || ok {
		return saved, err
	} else {
		job, err := loadJob(ctx, tx, request.JobID)
		if err != nil {
			return Job{}, err
		}
		if job.Version != request.IfVersion {
			return Job{}, detailFault(CodeConflict, "job version changed", map[string]any{"currentVersion": job.Version}, nil)
		}
		if job.Status != JobReadyForReview {
			return Job{}, fault(CodeNotReady, "job is not ready for review", nil)
		}
		if job.CandidateRevision != request.CandidateRevision || job.Admission.TicketSHA256 != request.RequirementFingerprint {
			return Job{}, fault(CodeStale, "review does not match the current candidate and requirements", nil)
		}
		if !independentReviewer(job, current.UserID) {
			return Job{}, fault(CodeForbidden, "reviewer must be independent from coordinator and implementer", nil)
		}
		if err := checkSourcesUnchanged(job.Admission.Sources); err != nil {
			return Job{}, err
		}
		if !allChecksPassed(job) {
			return Job{}, fault(CodeNotReady, "all required checks must pass for the current candidate and requirements", nil)
		}
		id, err := newID("rev_")
		if err != nil {
			return Job{}, fault(CodeInternal, "create review identity", err)
		}
		now := s.state.now().UTC()
		if _, err := tx.ExecContext(ctx, `INSERT INTO reviews(id,job_id,reviewer_id,candidate_revision,requirement_fingerprint,decision,summary,findings_json,created_at) VALUES(?,?,?,?,?,?,?,?,?)`, id, job.ID, current.UserID, request.CandidateRevision, request.RequirementFingerprint, request.Decision, strings.TrimSpace(request.Summary), findingsJSON, formatTime(now)); err != nil {
			return Job{}, fault(CodeInternal, "record review", err)
		}
		status := JobReadyForAcceptance
		if request.Decision == ReviewChangesRequested {
			status = JobVerificationFailed
		}
		if _, err := tx.ExecContext(ctx, `UPDATE jobs SET status=?,version=version+1,updated_at=? WHERE id=?`, status, formatTime(now), job.ID); err != nil {
			return Job{}, fault(CodeInternal, "update reviewed job", err)
		}
		if _, err := appendEvent(ctx, tx, job.ID, &current.UserID, "review_recorded", map[string]any{"reviewId": id, "decision": request.Decision, "candidateRevision": request.CandidateRevision, "findings": findings}, now); err != nil {
			return Job{}, err
		}
		job, err = loadJob(ctx, tx, job.ID)
		if err != nil {
			return Job{}, err
		}
		job.AvailableActions = availableActions(job)
		if err := saveReceipt(ctx, tx, current.UserID, "review", request.RequestID, hash, job, false, now); err != nil {
			return Job{}, err
		}
		if err := tx.Commit(); err != nil {
			return Job{}, fault(CodeInternal, "commit review", err)
		}
		return job, nil
	}
}

func allChecksPassed(job Job) bool {
	if len(job.Admission.RequiredChecks) == 0 {
		return false
	}
	latest := latestCurrentCheckAttempts(job)
	passedAttempts := map[AttemptID]bool{}
	for _, result := range job.Checks {
		if result.CandidateRevision == job.CandidateRevision && result.RequirementFingerprint == job.Admission.TicketSHA256 && result.Status == CheckPassed && result.ObserverID != nil && result.AttemptID != nil {
			passedAttempts[*result.AttemptID] = true
		}
	}
	for _, required := range job.Admission.RequiredChecks {
		attempt, ok := latest[required]
		if !ok || attempt.Status != AttemptSucceeded || attempt.CheckStatus != CheckPassed || !passedAttempts[attempt.ID] {
			return false
		}
	}
	return true
}

func latestCurrentCheckAttempts(job Job) map[string]Attempt {
	latest := map[string]Attempt{}
	for _, attempt := range job.Attempts {
		if attempt.Stage != StageVerification || attempt.CandidateRevision != job.CandidateRevision {
			continue
		}
		current, ok := latest[attempt.CheckKey]
		if !ok || attempt.Ordinal > current.Ordinal {
			latest[attempt.CheckKey] = attempt
		}
	}
	return latest
}
