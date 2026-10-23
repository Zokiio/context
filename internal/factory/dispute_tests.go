package factory

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

func (s *Service) prepareDisputeTests(ctx context.Context, tx *sql.Tx, d *Dispute, job Job, project Project, actor UserID, now time.Time) (*ExpectationInput, error) {
	if !s.canWriteExpectations(project) {
		return nil, fault(CodeNotReady, "a configured independent test author and protected paths are required", nil)
	}
	if !now.Before(job.Admission.Deadline) {
		return nil, fault(CodeDeadline, "the original run deadline has elapsed", nil)
	}
	if err := validateAdmissionFresh(ctx, tx, job.ProjectID, job.WorkItemID, job.Admission); err != nil {
		return nil, err
	}
	criteria, err := acceptanceCriteria(job.Admission.TicketPath)
	if err != nil {
		return nil, err
	}
	id, err := newID("exp_")
	if err != nil {
		return nil, err
	}
	expectation := Expectation{ID: ID(id), DisputeID: d.ID, ProjectID: d.ProjectID, WorkItemID: d.WorkItemID, Title: d.Input.Summary, TicketPath: job.Admission.TicketPath, CriteriaSHA256: job.Admission.CriteriaSHA256, Status: ExpectationRunning, RequestedBy: actor, BaseRevision: job.Admission.BaseRevision, Criteria: criteria, Instructions: project.Policy.Instructions.TestAuthor, CreatedAt: now, UpdatedAt: now}
	input := ExpectationInput{ExpectationID: expectation.ID, WorkItemID: d.WorkItemID, Title: expectation.Title, TicketPath: expectation.TicketPath, Criteria: criteria, Context: job.Admission.Sources, ProtectedPaths: job.Admission.ProtectedPaths, BaseRevision: expectation.BaseRevision, Instructions: expectation.Instructions}
	for _, source := range job.Admission.Sources {
		if source.Path == job.Admission.TicketPath {
			input.Ticket = source.Text
		}
	}
	if job.Admission.ExpectationsID != "" {
		original, err := loadExpectation(ctx, tx, job.Admission.ExpectationsID)
		if err != nil {
			return nil, err
		}
		input.OriginalTests = &original
	}
	if err = saveExpectation(ctx, tx, expectation, true); err != nil {
		return nil, err
	}
	d.Status, d.NextOwner, d.ExpectationID = "authoring_tests", "test_author", expectation.ID
	snapshot := *d
	input.Dispute = &snapshot
	return &input, nil
}

// replaceDisputedJob publishes the new admission and relinquishes the old claim in
// one SQLite transaction. Existing attempts, checks and approvals are never copied.
func (s *Service) replaceDisputedJob(ctx context.Context, tx *sql.Tx, d *Dispute, job Job, project Project, actor UserID, now time.Time) error {
	if !now.Before(job.Admission.Deadline) {
		return fault(CodeDeadline, "the original run deadline has elapsed; return to planning", nil)
	}
	if err := validateAdmissionFresh(ctx, tx, project.ID, job.WorkItemID, job.Admission); err != nil {
		return err
	}
	expectation, err := loadExpectation(ctx, tx, d.ExpectationID)
	if err != nil {
		return err
	}
	if expectation.DisputeID != d.ID || expectation.Status != ExpectationProposed || expectation.CriteriaSHA256 != job.Admission.CriteriaSHA256 {
		return fault(CodeConflict, "the disputed test proposal is not ready", nil)
	}
	if _, err = s.resolveCommit(ctx, project.RepositoryPath, expectation.Commit); err != nil {
		return err
	}
	expectation.Status, expectation.DecidedBy, expectation.UpdatedAt = ExpectationAccepted, &actor, now
	stampDecision(ctx, &expectation.DecisionID, &expectation.Delegated)
	if err = saveExpectation(ctx, tx, expectation, false); err != nil {
		return err
	}
	jobID, err := newID("job_")
	if err != nil {
		return err
	}
	attemptID, err := newID("att_")
	if err != nil {
		return err
	}
	dispatchID, err := newID("dsp_")
	if err != nil {
		return err
	}
	admission := job.Admission
	admission.ParentJobID, admission.LineageID, admission.DisputeID = job.ID, d.LineageID, d.ID
	admission.BaseRevision, admission.ExpectationsID, admission.ExpectationsCommit, admission.AdmittedAt = expectation.Commit, expectation.ID, expectation.Commit, now
	encoded, err := marshal(admission)
	if err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE jobs SET status=?,active_claim=0,version=version+1,updated_at=? WHERE id=?`, JobSuperseded, formatTime(now), job.ID); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO jobs(id,project_id,work_item_id,coordinator_id,status,version,admission_json,repair_count,created_at,updated_at) VALUES(?,?,?,?,?,1,?,?,?,?)`, jobID, project.ID, job.WorkItemID, actor, JobQueued, encoded, job.RepairCount, formatTime(now), formatTime(now)); err != nil {
		return fault(CodeInternal, "admit replacement job", err)
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO attempts(id,job_id,stage,ordinal,dispatch_id,workspace_path,input_revision,status) VALUES(?,?,?,?,?,?,?,?)`, attemptID, jobID, StageImplementation, 1, dispatchID, s.workspacePath(JobID(jobID), AttemptID(attemptID)), admission.BaseRevision, AttemptPending); err != nil {
		return err
	}
	if _, err = appendEvent(ctx, tx, JobID(jobID), &actor, "dispute_replacement_admitted", map[string]any{"disputeId": d.ID, "parentJobId": job.ID, "expectationId": expectation.ID, "baseRevision": admission.BaseRevision}, now); err != nil {
		return err
	}
	d.Status, d.NextOwner, d.ReplacementJobID = "resolved", "implementer", JobID(jobID)
	return nil
}

// finishDisputeTests runs in the same transaction that records the author result.
// A late result after Stop remains evidence and cannot authorize another run.
func (s *Service) finishDisputeTests(ctx context.Context, tx *sql.Tx, expectation Expectation, now time.Time) error {
	if expectation.DisputeID == "" {
		return nil
	}
	d, err := loadDispute(ctx, tx, expectation.DisputeID)
	if err != nil {
		return err
	}
	if d.Status != "authoring_tests" || d.ExpectationID != expectation.ID {
		return nil
	}
	d.Version++
	d.UpdatedAt = now
	job, project, err := s.disputeRouteReady(ctx, tx, d)
	if err != nil {
		d.Status, d.NextOwner, d.Error = "stopped", "none", err.Error()
	} else if expectation.Status != ExpectationProposed {
		d.Status, d.NextOwner, d.Error = "needs_human", "you", expectation.Error
	} else {
		d.Status, d.NextOwner = "tests_proposed", "you"
		// Existing deliberate delegation is narrower than a judge's recommendation.
		// Intent changes, weakening, author concerns, and repeated cycles never auto-accept.
		var count int
		if err = tx.QueryRowContext(ctx, `SELECT count(*) FROM disputes WHERE lineage_id=?`, d.LineageID).Scan(&count); err != nil {
			return err
		}
		if s.state.pilot && count == 1 && project.Policy.Ask.SetBy != nil && project.Policy.Ask.Autopilot && project.Policy.Ask.BeforeBuild == AskBeforeNever && d.Decision != nil && d.Decision.PreservesIntent && !d.Decision.WeakensAcceptance && len(expectation.Concerns) == 0 {
			current, e := authorizeProject(ctx, tx, Principal{UserID: *project.Policy.Ask.SetBy}, project.ID, CapabilityCoordinate)
			if e == nil {
				if _, e = tx.ExecContext(ctx, "SAVEPOINT dispute_replace"); e != nil {
					return e
				}
				delegated := withDecision(ctx, decisionNote{ID: "dispute/" + string(d.ID) + "/tests", Delegated: "autopilot: the project's ask policy permits this scope-preserving test revision"})
				e = s.replaceDisputedJob(delegated, tx, &d, job, project, current.UserID, now)
				if e == nil {
					d.ResolvedBy, d.Resolution, d.Rationale = &current.UserID, "accept_tests", "Delegated by the project's ask policy"
				}
				if e != nil {
					_, _ = tx.ExecContext(ctx, "ROLLBACK TO dispute_replace")
				}
				if _, releaseErr := tx.ExecContext(ctx, "RELEASE dispute_replace"); releaseErr != nil {
					return releaseErr
				}
			}
			if e != nil {
				d.Status, d.NextOwner, d.Error = "needs_human", "you", e.Error()
			}
		}
	}
	if err = saveDispute(ctx, tx, d, false); err != nil {
		return err
	}
	_, err = appendEvent(ctx, tx, d.JobID, nil, "dispute_tests_ready", map[string]any{"disputeId": d.ID, "expectationId": expectation.ID, "status": d.Status, "nextOwner": d.NextOwner, "replacementJobId": d.ReplacementJobID}, now)
	return err
}

func (s *Service) finishDisputedAttempt(ctx context.Context, tx *sql.Tx, actor Principal, attempt attemptRecord, request WorkerReportRequest, now time.Time) error {
	if attempt.Stage != StageImplementation || attempt.mergeCheck() || request.Dispute == nil || request.CandidateRevision != "" {
		return fault(CodeInvalid, "a dispute requires a stopped implementation, structured report, and no candidate", nil)
	}
	if attempt.Status == AttemptUnknown || attempt.JobStatus == JobImplementationUnknown || attempt.JobStatus == JobCancellationUnknown {
		return fault(CodeUncertain, "uncertain execution must be reconciled before disputing", nil)
	}
	if err := ValidateDisputeInput(*request.Dispute); err != nil {
		return err
	}
	if attempt.CancellationRequested || attempt.JobStatus == JobCancelRequested {
		request.Outcome = OutcomeCancelled
		request.Dispute = nil
		return s.finishImplementation(ctx, tx, attempt, request, now)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE attempts SET status=?,outcome=?,candidate_revision='',terminal_receipt=?,finished_at=? WHERE id=?`, AttemptFailed, OutcomeDisputed, request.TerminalReceipt, formatTime(request.ObservedAt), attempt.ID); err != nil {
		return err
	}
	job, err := loadJob(ctx, tx, attempt.JobID)
	if err != nil {
		return err
	}
	if _, err = s.newDispute(ctx, tx, actor.UserID, job, *request.Dispute, now); err != nil {
		return fmt.Errorf("record implementation dispute: %w", err)
	}
	return nil
}
