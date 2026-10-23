package factory

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Zokiio/context/internal/orientation"
)

type attemptRecord struct {
	Attempt
	ProjectID     ProjectID
	CoordinatorID UserID
	ImplementerID sql.NullString
	JobStatus     JobStatus
	Admission     Admission
	Repository    string
}

func (s *Service) Claim(ctx context.Context, actor Principal, request WorkerClaimRequest) (Assignment, error) {
	if err := requireRequestID(request.RequestID); err != nil {
		return Assignment{}, err
	}
	stages, err := normalizedStages(request.Stages)
	if err != nil {
		return Assignment{}, err
	}
	s.state.mutations.Lock()
	defer s.state.mutations.Unlock()
	tx, err := s.state.db.BeginTx(ctx, nil)
	if err != nil {
		return Assignment{}, fault(CodeInternal, "start worker claim", err)
	}
	defer tx.Rollback()
	current, err := requirePrincipal(ctx, tx, actor)
	if err != nil {
		return Assignment{}, err
	}
	if current.Kind != UserKindWorker {
		return Assignment{}, fault(CodeForbidden, "worker principal is required", nil)
	}
	now := s.state.now().UTC()
	if err := expireLeases(ctx, tx, now); err != nil {
		return Assignment{}, err
	}
	if saved, ok, hash, err := receipt[Assignment](ctx, tx, current.UserID, "worker_claim", request.RequestID, request); err != nil {
		return Assignment{}, err
	} else if ok {
		attempt, readErr := readAttempt(ctx, tx, saved.AttemptID)
		if readErr != nil {
			return Assignment{}, readErr
		}
		if attempt.Status == AttemptUnknown {
			return Assignment{}, detailFault(CodeUncertain, "worker claim expired and requires reconciliation", map[string]any{"attemptId": attempt.ID}, nil)
		}
		if err := authorizeWorkerStage(ctx, tx, current, attempt); err != nil {
			return Assignment{}, err
		}
		if err := claimFresh(ctx, tx, attempt); err != nil {
			return Assignment{}, err
		}
		return saved, nil
	} else {
		active, found, err := activeWorkerAttempt(ctx, tx, current.UserID, stages, request.ProjectID)
		if err != nil {
			return Assignment{}, err
		}
		if found && active.Status == AttemptUnknown {
			return Assignment{}, detailFault(CodeUncertain, "worker has an unresolved attempt", map[string]any{"attemptId": active.ID}, nil)
		}
		var selected attemptRecord
		if found {
			selected = active
			if err := authorizeWorkerStage(ctx, tx, current, selected); err != nil {
				return Assignment{}, err
			}
			if err := claimFresh(ctx, tx, selected); err != nil {
				return Assignment{}, err
			}
		} else {
			selected, err = selectPendingAttempt(ctx, tx, current, stages, request.ProjectID, now)
			if err != nil {
				return Assignment{}, err
			}
			if err := claimFresh(ctx, tx, selected); err != nil {
				return Assignment{}, err
			}
			if err := s.ensureWorkspace(ctx, selected); err != nil {
				return Assignment{}, err
			}
		}
		lease := now.Add(s.state.leaseDuration)
		if found {
			if _, err := tx.ExecContext(ctx, `UPDATE attempts SET lease_expires_at=?, last_heartbeat_at=? WHERE id=?`, formatTime(lease), formatTime(now), selected.ID); err != nil {
				return Assignment{}, fault(CodeInternal, "renew worker claim", err)
			}
		} else {
			jobStatus := JobImplementing
			if selected.Stage == StageVerification {
				jobStatus = JobVerifying
			}
			if _, err := tx.ExecContext(ctx, `UPDATE attempts SET worker_id=?, status=?, lease_expires_at=?, last_heartbeat_at=? WHERE id=? AND status=?`, current.UserID, AttemptAssigned, formatTime(lease), formatTime(now), selected.ID, AttemptPending); err != nil {
				return Assignment{}, fault(CodeInternal, "assign worker attempt", err)
			}
			if selected.Stage == StageImplementation {
				if _, err := tx.ExecContext(ctx, `UPDATE jobs SET implementer_id=?, status=?, version=version+1, updated_at=? WHERE id=?`, current.UserID, jobStatus, formatTime(now), selected.JobID); err != nil {
					return Assignment{}, fault(CodeInternal, "assign implementation", err)
				}
			} else if selected.mergeCheck() {
				if _, err := tx.ExecContext(ctx, `UPDATE jobs SET version=version+1, updated_at=? WHERE id=?`, formatTime(now), selected.JobID); err != nil {
					return Assignment{}, fault(CodeInternal, "assign merge check", err)
				}
			} else if _, err := tx.ExecContext(ctx, `UPDATE jobs SET status=?, version=version+1, updated_at=? WHERE id=?`, jobStatus, formatTime(now), selected.JobID); err != nil {
				return Assignment{}, fault(CodeInternal, "assign verification", err)
			}
			if _, err := appendEvent(ctx, tx, selected.JobID, &current.UserID, "attempt_assigned", map[string]any{"attemptId": selected.ID, "dispatchId": selected.DispatchID, "stage": selected.Stage, "checkKey": selected.CheckKey}, now); err != nil {
				return Assignment{}, err
			}
		}
		selected.WorkerID = &current.UserID
		selected.LeaseExpiresAt = &lease
		assignment, err := assignmentFor(ctx, tx, selected, current.UserID, lease)
		if err != nil {
			return Assignment{}, err
		}
		if err := saveReceipt(ctx, tx, current.UserID, "worker_claim", request.RequestID, hash, assignment, false, now); err != nil {
			return Assignment{}, err
		}
		if err := tx.Commit(); err != nil {
			return Assignment{}, fault(CodeInternal, "commit worker claim", err)
		}
		return assignment, nil
	}
}

func (s *Service) Heartbeat(ctx context.Context, actor Principal, request HeartbeatRequest) (HeartbeatResult, error) {
	if request.AttemptID == "" || request.DispatchID == "" {
		return HeartbeatResult{}, fault(CodeInvalid, "attemptId and dispatchId are required", nil)
	}
	s.state.mutations.Lock()
	defer s.state.mutations.Unlock()
	tx, err := s.state.db.BeginTx(ctx, nil)
	if err != nil {
		return HeartbeatResult{}, fault(CodeInternal, "start heartbeat", err)
	}
	defer tx.Rollback()
	current, attempt, err := authorizeWorkerAttempt(ctx, tx, actor, request.AttemptID, request.DispatchID)
	if err != nil {
		return HeartbeatResult{}, err
	}
	if err := authorizeWorkerStage(ctx, tx, current, attempt); err != nil {
		return HeartbeatResult{}, err
	}
	now := s.state.now().UTC()
	if attempt.LeaseExpiresAt != nil && !now.Before(*attempt.LeaseExpiresAt) && attempt.Status != AttemptUnknown && !terminalAttempt(attempt.Status) && attempt.mergeCheck() {
		if _, err := tx.ExecContext(ctx, `UPDATE attempts SET status=?,outcome=?,check_status=? WHERE id=?`, AttemptUnknown, OutcomeUnknown, CheckUnknown, attempt.ID); err != nil {
			return HeartbeatResult{}, fault(CodeInternal, "mark missed merge heartbeat uncertain", err)
		}
		if err := failMergeVerification(ctx, tx, attempt.JobID, attempt.CandidateRevision, attempt.Ordinal, "a merge check missed its heartbeat", now); err != nil {
			return HeartbeatResult{}, err
		}
		if err := tx.Commit(); err != nil {
			return HeartbeatResult{}, fault(CodeInternal, "commit missed merge heartbeat", err)
		}
		return HeartbeatResult{}, fault(CodeUncertain, "worker lease expired and requires reconciliation", nil)
	}
	if attempt.LeaseExpiresAt != nil && !now.Before(*attempt.LeaseExpiresAt) && attempt.Status != AttemptUnknown && !terminalAttempt(attempt.Status) {
		check := attempt.CheckStatus
		if attempt.Stage == StageVerification {
			check = CheckUnknown
		}
		if _, err := tx.ExecContext(ctx, `UPDATE attempts SET status=?,outcome=?,check_status=? WHERE id=?`, AttemptUnknown, OutcomeUnknown, check, attempt.ID); err != nil {
			return HeartbeatResult{}, fault(CodeInternal, "mark missed heartbeat uncertain", err)
		}
		jobStatus := JobImplementationUnknown
		if attempt.CancellationRequested {
			jobStatus = JobCancellationUnknown
		}
		if _, err := tx.ExecContext(ctx, `UPDATE jobs SET status=?,version=version+1,updated_at=? WHERE id=?`, jobStatus, formatTime(now), attempt.JobID); err != nil {
			return HeartbeatResult{}, fault(CodeInternal, "mark missed-heartbeat job uncertain", err)
		}
		if _, err := appendEvent(ctx, tx, attempt.JobID, nil, "worker_lease_expired", map[string]any{"attemptId": attempt.ID}, now); err != nil {
			return HeartbeatResult{}, err
		}
		if err := tx.Commit(); err != nil {
			return HeartbeatResult{}, fault(CodeInternal, "commit missed heartbeat", err)
		}
		return HeartbeatResult{}, fault(CodeUncertain, "worker lease expired and requires reconciliation", nil)
	}
	if saved, ok, hash, err := receipt[HeartbeatResult](ctx, tx, current.UserID, "heartbeat", request.RequestID, request); err != nil || ok {
		return saved, err
	} else {
		if attempt.Status == AttemptUnknown {
			return HeartbeatResult{}, fault(CodeUncertain, "attempt requires reconciliation", nil)
		}
		if terminalAttempt(attempt.Status) {
			return HeartbeatResult{}, fault(CodeConflict, "attempt is already terminal", nil)
		}
		lease := now.Add(s.state.leaseDuration)
		if _, err := tx.ExecContext(ctx, `UPDATE attempts SET lease_expires_at=?, last_heartbeat_at=? WHERE id=?`, formatTime(lease), formatTime(now), attempt.ID); err != nil {
			return HeartbeatResult{}, fault(CodeInternal, "save heartbeat", err)
		}
		directive := DirectiveContinue
		if !attempt.mergeCheck() && !now.Before(attempt.Admission.Deadline) && !attempt.CancellationRequested {
			attempt.CancellationRequested = true
			if _, err := tx.ExecContext(ctx, `UPDATE attempts SET cancellation_requested=1 WHERE id=?`, attempt.ID); err != nil {
				return HeartbeatResult{}, fault(CodeInternal, "request deadline cancellation", err)
			}
			if _, err := tx.ExecContext(ctx, `UPDATE jobs SET status=?,version=version+1,updated_at=? WHERE id=?`, JobCancelRequested, formatTime(now), attempt.JobID); err != nil {
				return HeartbeatResult{}, fault(CodeInternal, "mark deadline cancellation", err)
			}
		}
		if attempt.CancellationRequested {
			directive = DirectiveCancel
		}
		if err := recordProgress(ctx, tx, attempt, request.Progress, now); err != nil {
			return HeartbeatResult{}, err
		}
		result := HeartbeatResult{AttemptID: attempt.ID, Directive: directive, LeaseExpiresAt: lease}
		if err := saveReceipt(ctx, tx, current.UserID, "heartbeat", request.RequestID, hash, result, false, now); err != nil {
			return HeartbeatResult{}, err
		}
		if err := tx.Commit(); err != nil {
			return HeartbeatResult{}, fault(CodeInternal, "commit heartbeat", err)
		}
		return result, nil
	}
}

func (s *Service) Report(ctx context.Context, actor Principal, request WorkerReportRequest) (Job, error) {
	if request.AttemptID == "" || request.DispatchID == "" || request.Kind == "" || request.ObservedAt.IsZero() {
		return Job{}, fault(CodeInvalid, "attemptId, dispatchId, kind, and observedAt are required", nil)
	}
	s.state.mutations.Lock()
	defer s.state.mutations.Unlock()
	tx, err := s.state.db.BeginTx(ctx, nil)
	if err != nil {
		return Job{}, fault(CodeInternal, "start worker report", err)
	}
	defer tx.Rollback()
	current, attempt, err := authorizeWorkerAttempt(ctx, tx, actor, request.AttemptID, request.DispatchID)
	if err != nil {
		return Job{}, err
	}
	canRead, err := principalCanReadProject(ctx, tx, current, attempt.ProjectID)
	if err != nil {
		return Job{}, err
	}
	if request.Kind == ReportStartRegistered {
		if err := authorizeWorkerStage(ctx, tx, current, attempt); err != nil {
			return Job{}, err
		}
		if request.NativeRunID == "" {
			return Job{}, fault(CodeInvalid, "start registration requires nativeRunId", nil)
		}
		if attempt.Status == AttemptUnknown && attempt.NativeRunID == request.NativeRunID {
			if attempt.CancellationRequested {
				return Job{}, fault(CodeDeadline, "job deadline elapsed before start registration", nil)
			}
			return Job{}, fault(CodeUncertain, "worker lease expired before start registration", nil)
		}
		if attempt.Status == AttemptAssigned {
			now := s.state.now().UTC()
			deadlineElapsed := !attempt.mergeCheck() && !now.Before(attempt.Admission.Deadline)
			leaseExpired := attempt.LeaseExpiresAt == nil || !now.Before(*attempt.LeaseExpiresAt)
			if attempt.mergeCheck() && leaseExpired {
				if _, err := tx.ExecContext(ctx, `UPDATE attempts SET status=?,outcome=?,check_status=?,native_run_id=? WHERE id=?`, AttemptUnknown, OutcomeUnknown, CheckUnknown, request.NativeRunID, attempt.ID); err != nil {
					return Job{}, fault(CodeInternal, "retain expired merge check", err)
				}
				if err := failMergeVerification(ctx, tx, attempt.JobID, attempt.CandidateRevision, attempt.Ordinal, "a merge check lease expired before it started", now); err != nil {
					return Job{}, err
				}
				if err := tx.Commit(); err != nil {
					return Job{}, fault(CodeInternal, "commit expired merge check", err)
				}
				return Job{}, fault(CodeUncertain, "worker lease expired before start registration", nil)
			}
			if deadlineElapsed || leaseExpired {
				checkStatus := attempt.CheckStatus
				if attempt.Stage == StageVerification {
					checkStatus = CheckUnknown
				}
				cancellationRequested := attempt.CancellationRequested || deadlineElapsed
				if _, err := tx.ExecContext(ctx, `UPDATE attempts SET status=?,outcome=?,check_status=?,native_run_id=?,cancellation_requested=? WHERE id=?`, AttemptUnknown, OutcomeUnknown, checkStatus, request.NativeRunID, boolInt(cancellationRequested), attempt.ID); err != nil {
					return Job{}, fault(CodeInternal, "retain expired pre-start assignment", err)
				}
				jobStatus := JobImplementationUnknown
				if cancellationRequested {
					jobStatus = JobCancellationUnknown
				}
				if _, err := tx.ExecContext(ctx, `UPDATE jobs SET status=?,version=version+1,updated_at=? WHERE id=?`, jobStatus, formatTime(now), attempt.JobID); err != nil {
					return Job{}, fault(CodeInternal, "retain expired pre-start job", err)
				}
				if _, err := appendEvent(ctx, tx, attempt.JobID, &current.UserID, "worker_start_rejected", map[string]any{"attemptId": attempt.ID, "deadlineElapsed": deadlineElapsed, "leaseExpired": leaseExpired}, now); err != nil {
					return Job{}, err
				}
				if err := tx.Commit(); err != nil {
					return Job{}, fault(CodeInternal, "commit rejected worker start", err)
				}
				if deadlineElapsed {
					return Job{}, fault(CodeDeadline, "job deadline elapsed before start registration", nil)
				}
				return Job{}, fault(CodeUncertain, "worker lease expired before start registration", nil)
			}
		}
		if !attempt.mergeCheck() {
			if err := validateAdmissionFresh(ctx, tx, attempt.ProjectID, mustWorkItem(ctx, tx, attempt.JobID), attempt.Admission); err != nil {
				return Job{}, err
			}
		}
	}
	if saved, ok, hash, err := receipt[Job](ctx, tx, current.UserID, "worker_report", request.RequestID, request); err != nil {
		return Job{}, err
	} else if ok {
		if !canRead {
			saved = redactJobAcknowledgement(saved)
		}
		return saved, nil
	} else {
		now := s.state.now().UTC()
		switch request.Kind {
		case ReportStartRegistered:
			if request.NativeRunID == "" || attempt.Status != AttemptAssigned {
				return Job{}, fault(CodeConflict, "start registration requires an assigned attempt and nativeRunId", nil)
			}
			if _, err := tx.ExecContext(ctx, `UPDATE attempts SET status=?, native_run_id=?, started_at=COALESCE(started_at,?) WHERE id=?`, AttemptStartRegistered, request.NativeRunID, formatTime(request.ObservedAt), attempt.ID); err != nil {
				return Job{}, fault(CodeInternal, "register worker start", err)
			}
		case ReportStarted:
			if request.NativeRunID == "" || request.NativeRunID != attempt.NativeRunID || attempt.Status != AttemptStartRegistered {
				return Job{}, fault(CodeConflict, "started report does not match the registered native run", nil)
			}
			if _, err := tx.ExecContext(ctx, `UPDATE attempts SET status=? WHERE id=?`, AttemptRunning, attempt.ID); err != nil {
				return Job{}, fault(CodeInternal, "mark worker running", err)
			}
		case ReportCancelAcknowledged:
			if request.NativeRunID == "" || request.NativeRunID != attempt.NativeRunID || !attempt.CancellationRequested || (attempt.Status != AttemptStartRegistered && attempt.Status != AttemptRunning) {
				return Job{}, fault(CodeConflict, "cancellation acknowledgement does not match an active requested run", nil)
			}
			if _, err := tx.ExecContext(ctx, `UPDATE attempts SET status=? WHERE id=?`, AttemptCancelAcknowledged, attempt.ID); err != nil {
				return Job{}, fault(CodeInternal, "record cancellation acknowledgement", err)
			}
		case ReportUnknown:
			if attempt.Status == AttemptAssigned {
				if request.NativeRunID == "" {
					return Job{}, fault(CodeInvalid, "unknown prepared execution requires nativeRunId", nil)
				}
				if _, err := tx.ExecContext(ctx, `UPDATE attempts SET native_run_id=? WHERE id=?`, request.NativeRunID, attempt.ID); err != nil {
					return Job{}, fault(CodeInternal, "retain prepared execution identity", err)
				}
			} else if err := validateRunCorrelation(attempt, request); err != nil {
				return Job{}, err
			}
			if request.TerminalReceipt == "" {
				return Job{}, fault(CodeInvalid, "unknown execution requires a retained receipt", nil)
			}
			if err := s.finishUnknown(ctx, tx, current, attempt, request, now); err != nil {
				return Job{}, err
			}
		case ReportTerminal:
			if err := validateRunCorrelation(attempt, request); err != nil {
				return Job{}, err
			}
			if request.TerminalReceipt == "" {
				return Job{}, fault(CodeInvalid, "terminal report requires terminalReceipt", nil)
			}
			if err := s.finishAttempt(ctx, tx, current, attempt, request, now); err != nil {
				return Job{}, err
			}
		default:
			return Job{}, fault(CodeInvalid, "unknown worker report kind", nil)
		}
		if summary := strings.TrimSpace(request.Summary); summary != "" {
			if _, err := tx.ExecContext(ctx, `UPDATE attempts SET summary=? WHERE id=?`, summary, attempt.ID); err != nil {
				return Job{}, fault(CodeInternal, "retain worker report summary", err)
			}
		}
		if _, err := appendEvent(ctx, tx, attempt.JobID, &current.UserID, "worker_reported", map[string]any{"attemptId": attempt.ID, "kind": request.Kind, "outcome": request.Outcome, "checkStatus": request.CheckStatus}, now); err != nil {
			return Job{}, err
		}
		job, err := loadJob(ctx, tx, attempt.JobID)
		if err != nil {
			return Job{}, err
		}
		job.AvailableActions = availableActions(job)
		response := job
		if !canRead {
			response = redactJobAcknowledgement(job)
		}
		if err := saveReceipt(ctx, tx, current.UserID, "worker_report", request.RequestID, hash, response, false, now); err != nil {
			return Job{}, err
		}
		if err := tx.Commit(); err != nil {
			return Job{}, fault(CodeInternal, "commit worker report", err)
		}
		return response, nil
	}
}

func (s *Service) finishUnknown(ctx context.Context, tx *sql.Tx, actor Principal, attempt attemptRecord, request WorkerReportRequest, now time.Time) error {
	if err := storeArtifacts(ctx, tx, attempt.JobID, attempt.ID, request.Artifacts, now); err != nil {
		return err
	}
	if attempt.mergeCheck() {
		if err := recordCheckResult(ctx, tx, actor.UserID, attempt, CheckUnknown, request.ObservedAt, now); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE attempts SET status=?, outcome=?, check_status=?, terminal_receipt=?, finished_at=? WHERE id=?`, AttemptUnknown, OutcomeUnknown, CheckUnknown, request.TerminalReceipt, formatTime(request.ObservedAt), attempt.ID); err != nil {
			return fault(CodeInternal, "mark merge check uncertain", err)
		}
		return failMergeVerification(ctx, tx, attempt.JobID, attempt.CandidateRevision, attempt.Ordinal, "a merge check outcome is unknown", now)
	}
	status := JobImplementationUnknown
	if attempt.CancellationRequested {
		status = JobCancellationUnknown
	}
	checkStatus := attempt.CheckStatus
	if attempt.Stage == StageVerification {
		checkStatus = CheckUnknown
		if err := recordCheckResult(ctx, tx, actor.UserID, attempt, CheckUnknown, request.ObservedAt, now); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, `UPDATE attempts SET status=?, outcome=?, check_status=?, terminal_receipt=?, finished_at=? WHERE id=?`, AttemptUnknown, OutcomeUnknown, checkStatus, request.TerminalReceipt, formatTime(request.ObservedAt), attempt.ID); err != nil {
		return fault(CodeInternal, "mark attempt uncertain", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE jobs SET status=?, version=version+1, updated_at=? WHERE id=?`, status, formatTime(now), attempt.JobID); err != nil {
		return fault(CodeInternal, "mark job uncertain", err)
	}
	return nil
}

func (s *Service) finishAttempt(ctx context.Context, tx *sql.Tx, actor Principal, attempt attemptRecord, request WorkerReportRequest, now time.Time) error {
	if request.Outcome != OutcomeSucceeded && request.Outcome != OutcomeFailed && request.Outcome != OutcomeCancelled {
		return fault(CodeInvalid, "terminal outcome must be succeeded, failed, or cancelled", nil)
	}
	if err := storeArtifacts(ctx, tx, attempt.JobID, attempt.ID, request.Artifacts, now); err != nil {
		return err
	}
	if attempt.Stage == StageImplementation {
		return s.finishImplementation(ctx, tx, attempt, request, now)
	}
	return s.finishVerification(ctx, tx, actor, attempt, request, now)
}

func (s *Service) finishImplementation(ctx context.Context, tx *sql.Tx, attempt attemptRecord, request WorkerReportRequest, now time.Time) error {
	status := AttemptFailed
	jobStatus := JobFailed
	active := 1
	if request.Outcome == OutcomeCancelled {
		status, jobStatus, active = AttemptCancelled, JobCancelled, 0
	} else if request.Outcome == OutcomeSucceeded {
		candidate, err := s.validateImplementationCandidate(ctx, attempt, request.CandidateRevision)
		if err != nil {
			return err
		}
		protected, err := s.protectedChanges(ctx, attempt.WorkspacePath, attempt.Admission.BaseRevision, candidate, attempt.Admission.ProtectedPaths)
		if err != nil {
			return err
		}
		if len(protected) > 0 {
			return s.rejectProtectedCandidate(ctx, tx, attempt, request, candidate, protected, now)
		}
		status, jobStatus = AttemptSucceeded, JobVerifying
		request.CandidateRevision = candidate
		if attempt.CancellationRequested {
			jobStatus, active = JobCancelled, 0
		} else {
			var verificationOrdinal int
			if err := tx.QueryRowContext(ctx, `SELECT COALESCE(max(ordinal),0)+1 FROM attempts WHERE job_id=? AND stage=?`, attempt.JobID, StageVerification).Scan(&verificationOrdinal); err != nil {
				return fault(CodeInternal, "number verification generation", err)
			}
			for _, check := range attempt.Admission.RequiredChecks {
				id, err := newID("att_")
				if err != nil {
					return fault(CodeInternal, "create verification attempt", err)
				}
				dispatch, err := newID("dsp_")
				if err != nil {
					return fault(CodeInternal, "create verification dispatch", err)
				}
				path := s.workspacePath(attempt.JobID, AttemptID(id))
				if _, err := tx.ExecContext(ctx, `INSERT INTO attempts(id,job_id,stage,ordinal,check_key,dispatch_id,workspace_path,input_revision,candidate_revision,status) VALUES(?,?,?,?,?,?,?,?,?,?)`, id, attempt.JobID, StageVerification, verificationOrdinal, check, dispatch, path, candidate, candidate, AttemptPending); err != nil {
					return fault(CodeInternal, "queue verification", err)
				}
			}
		}
	}
	if attempt.CancellationRequested && request.Outcome == OutcomeFailed {
		jobStatus, active = JobCancelled, 0
	}
	if _, err := tx.ExecContext(ctx, `UPDATE attempts SET status=?, outcome=?, candidate_revision=?, terminal_receipt=?, finished_at=? WHERE id=?`, status, request.Outcome, request.CandidateRevision, request.TerminalReceipt, formatTime(request.ObservedAt), attempt.ID); err != nil {
		return fault(CodeInternal, "finish implementation attempt", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE jobs SET status=?, candidate_revision=?, active_claim=?, version=version+1, updated_at=? WHERE id=?`, jobStatus, request.CandidateRevision, active, formatTime(now), attempt.JobID); err != nil {
		return fault(CodeInternal, "finish implementation job", err)
	}
	return nil
}

func (s *Service) finishVerification(ctx context.Context, tx *sql.Tx, actor Principal, attempt attemptRecord, request WorkerReportRequest, now time.Time) error {
	if attempt.mergeCheck() {
		return s.finishMergeVerification(ctx, tx, actor, attempt, request, now)
	}
	if request.CandidateRevision == "" {
		request.CandidateRevision = attempt.CandidateRevision
	}
	if request.CandidateRevision != attempt.CandidateRevision {
		return fault(CodeConflict, "verification report does not match the sealed candidate", nil)
	}
	if err := s.validateVerificationWorkspace(ctx, attempt); err != nil {
		request.Outcome, request.CheckStatus = OutcomeUnknown, CheckUnknown
	}
	if request.Outcome == OutcomeCancelled {
		request.CheckStatus = CheckUnknown
	}
	if request.CheckStatus != CheckPassed && request.CheckStatus != CheckFailed && request.CheckStatus != CheckSkipped && request.CheckStatus != CheckUnavailable && request.CheckStatus != CheckUnknown {
		return fault(CodeInvalid, "verification report has an invalid checkStatus", nil)
	}
	if request.Outcome != OutcomeSucceeded && request.CheckStatus == CheckPassed {
		request.CheckStatus = CheckFailed
	}
	attemptStatus := AttemptFailed
	if request.Outcome == OutcomeSucceeded && request.CheckStatus == CheckPassed {
		attemptStatus = AttemptSucceeded
	} else if request.Outcome == OutcomeCancelled {
		attemptStatus = AttemptCancelled
	} else if request.Outcome == OutcomeUnknown || request.CheckStatus == CheckUnknown {
		attemptStatus = AttemptUnknown
	}
	if err := recordCheckResult(ctx, tx, actor.UserID, attempt, request.CheckStatus, request.ObservedAt, now); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE attempts SET status=?, outcome=?, check_status=?, terminal_receipt=?, finished_at=? WHERE id=?`, attemptStatus, request.Outcome, request.CheckStatus, request.TerminalReceipt, formatTime(request.ObservedAt), attempt.ID); err != nil {
		return fault(CodeInternal, "finish verification attempt", err)
	}
	jobStatus := JobVerificationFailed
	var incomplete int
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM attempts current
		WHERE current.job_id=? AND current.stage=? AND current.candidate_revision=?
		AND current.ordinal=(SELECT max(latest.ordinal) FROM attempts latest WHERE latest.job_id=current.job_id AND latest.stage=current.stage AND latest.candidate_revision=current.candidate_revision AND latest.check_key=current.check_key)
		AND (current.status<>? OR current.check_status<>?)`, attempt.JobID, StageVerification, attempt.CandidateRevision, AttemptSucceeded, CheckPassed).Scan(&incomplete); err != nil {
		return fault(CodeInternal, "inspect required checks", err)
	}
	if incomplete == 0 {
		jobStatus = JobReadyForReview
	} else {
		var pending int
		if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM attempts current
			WHERE current.job_id=? AND current.stage=? AND current.candidate_revision=?
			AND current.ordinal=(SELECT max(latest.ordinal) FROM attempts latest WHERE latest.job_id=current.job_id AND latest.stage=current.stage AND latest.candidate_revision=current.candidate_revision AND latest.check_key=current.check_key)
			AND current.status IN (?,?,?,?,?)`, attempt.JobID, StageVerification, attempt.CandidateRevision, AttemptPending, AttemptAssigned, AttemptStartRegistered, AttemptRunning, AttemptCancelAcknowledged).Scan(&pending); err != nil {
			return fault(CodeInternal, "inspect pending checks", err)
		}
		if pending > 0 {
			jobStatus = JobVerifying
		}
	}
	if attemptStatus == AttemptUnknown {
		jobStatus = JobImplementationUnknown
	}
	activeClaim := 1
	if attempt.CancellationRequested {
		var remaining, uncertain int
		if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM attempts WHERE job_id=? AND status IN (?,?,?,?,?,?)`, attempt.JobID, AttemptPending, AttemptAssigned, AttemptStartRegistered, AttemptRunning, AttemptCancelAcknowledged, AttemptUnknown).Scan(&remaining); err != nil {
			return fault(CodeInternal, "inspect cancelling verification attempts", err)
		}
		if remaining == 0 {
			jobStatus, activeClaim = JobCancelled, 0
		} else {
			if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM attempts WHERE job_id=? AND status=?`, attempt.JobID, AttemptUnknown).Scan(&uncertain); err != nil {
				return fault(CodeInternal, "inspect uncertain cancelling attempts", err)
			}
			jobStatus = JobCancelRequested
			if uncertain > 0 {
				jobStatus = JobCancellationUnknown
			}
		}
	}
	if _, err := tx.ExecContext(ctx, `UPDATE jobs SET status=?,active_claim=?, version=version+1, updated_at=? WHERE id=?`, jobStatus, activeClaim, formatTime(now), attempt.JobID); err != nil {
		return fault(CodeInternal, "update verification job", err)
	}
	return nil
}

func normalizedStages(input []Stage) ([]Stage, error) {
	seen := map[Stage]bool{}
	result := []Stage{}
	for _, stage := range input {
		if stage != StageImplementation && stage != StageVerification {
			return nil, fault(CodeInvalid, "worker stages must be implementation or verification", nil)
		}
		if !seen[stage] {
			seen[stage] = true
			result = append(result, stage)
		}
	}
	if len(result) == 0 {
		return nil, fault(CodeInvalid, "at least one worker stage is required", nil)
	}
	return result, nil
}

func activeWorkerAttempt(ctx context.Context, q queryer, worker UserID, stages []Stage, projectID *ProjectID) (attemptRecord, bool, error) {
	query := `SELECT a.id FROM attempts a JOIN jobs j ON j.id=a.job_id WHERE a.worker_id=? AND a.status IN (?,?,?,?,?)`
	args := []any{worker, AttemptAssigned, AttemptStartRegistered, AttemptRunning, AttemptCancelAcknowledged, AttemptUnknown}
	query += ` AND a.stage IN (` + placeholders(len(stages)) + `)`
	for _, stage := range stages {
		args = append(args, stage)
	}
	if projectID != nil {
		query += ` AND j.project_id=?`
		args = append(args, *projectID)
	}
	query += ` ORDER BY j.created_at,a.ordinal LIMIT 1`
	var id AttemptID
	err := q.QueryRowContext(ctx, query, args...).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return attemptRecord{}, false, nil
	}
	if err != nil {
		return attemptRecord{}, false, fault(CodeInternal, "find active worker attempt", err)
	}
	attempt, err := readAttempt(ctx, q, id)
	return attempt, err == nil, err
}

func selectPendingAttempt(ctx context.Context, q queryer, actor Principal, stages []Stage, projectID *ProjectID, now time.Time) (attemptRecord, error) {
	query := `SELECT a.id FROM attempts a JOIN jobs j ON j.id=a.job_id WHERE a.status=? AND a.stage IN (` + placeholders(len(stages)) + `)`
	args := []any{AttemptPending}
	for _, stage := range stages {
		args = append(args, stage)
	}
	if projectID != nil {
		query += ` AND j.project_id=?`
		args = append(args, *projectID)
	}
	query += ` ORDER BY j.created_at,a.ordinal,a.check_key`
	rows, err := q.QueryContext(ctx, query, args...)
	if err != nil {
		return attemptRecord{}, fault(CodeInternal, "find pending worker attempt", err)
	}
	ids := []AttemptID{}
	for rows.Next() {
		var id AttemptID
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return attemptRecord{}, fault(CodeInternal, "read pending attempt", err)
		}
		ids = append(ids, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return attemptRecord{}, fault(CodeInternal, "read pending attempts", err)
	}
	for _, id := range ids {
		attempt, err := readAttempt(ctx, q, id)
		if err != nil {
			return attemptRecord{}, err
		}
		capability := CapabilityImplementWorker
		if attempt.Stage == StageVerification {
			capability = CapabilityVerifyWorker
			if actor.UserID == attempt.CoordinatorID || (attempt.ImplementerID.Valid && actor.UserID == UserID(attempt.ImplementerID.String)) {
				continue
			}
		}
		caps, err := readCapabilities(ctx, q, actor.UserID, attempt.ProjectID)
		if err != nil {
			return attemptRecord{}, err
		}
		if !actor.InstallationAdmin && (!hasCapability(caps, CapabilityRead) || !hasCapability(caps, capability)) {
			continue
		}
		if !attempt.mergeCheck() && !now.Before(attempt.Admission.Deadline) {
			continue
		}
		return attempt, nil
	}
	return attemptRecord{}, fault(CodeNoWork, "no eligible worker assignment is available", nil)
}

func readAttempt(ctx context.Context, q queryer, attemptID AttemptID) (attemptRecord, error) {
	var result attemptRecord
	var worker, lease, heartbeat, started, finished sql.NullString
	var admissionJSON string
	err := q.QueryRowContext(ctx, `SELECT a.id,a.job_id,a.stage,a.ordinal,a.check_key,a.dispatch_id,a.worker_id,a.native_run_id,a.workspace_path,a.input_revision,a.candidate_revision,a.status,a.outcome,a.check_status,a.lease_expires_at,a.last_heartbeat_at,a.started_at,a.finished_at,a.cancellation_requested,a.purpose,j.project_id,j.coordinator_id,j.implementer_id,j.status,j.admission_json,p.repository_path FROM attempts a JOIN jobs j ON j.id=a.job_id JOIN projects p ON p.id=j.project_id WHERE a.id=?`, attemptID).Scan(
		&result.ID, &result.JobID, &result.Stage, &result.Ordinal, &result.CheckKey, &result.DispatchID, &worker, &result.NativeRunID, &result.WorkspacePath, &result.InputRevision, &result.CandidateRevision, &result.Status, &result.Outcome, &result.CheckStatus, &lease, &heartbeat, &started, &finished, &result.CancellationRequested, &result.Purpose, &result.ProjectID, &result.CoordinatorID, &result.ImplementerID, &result.JobStatus, &admissionJSON, &result.Repository,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return attemptRecord{}, fault(CodeNotFound, "attempt not found", nil)
	}
	if err != nil {
		return attemptRecord{}, fault(CodeInternal, "read attempt", err)
	}
	if worker.Valid {
		value := UserID(worker.String)
		result.WorkerID = &value
	}
	result.LeaseExpiresAt, result.LastHeartbeatAt = nullableTime(lease), nullableTime(heartbeat)
	result.StartedAt, result.FinishedAt = nullableTime(started), nullableTime(finished)
	if err := json.Unmarshal([]byte(admissionJSON), &result.Admission); err != nil {
		return attemptRecord{}, fault(CodeInternal, "decode attempt admission", err)
	}
	return result, nil
}

func authorizeWorkerAttempt(ctx context.Context, q queryer, actor Principal, attemptID AttemptID, dispatchID string) (Principal, attemptRecord, error) {
	current, err := requirePrincipal(ctx, q, actor)
	if err != nil {
		return Principal{}, attemptRecord{}, err
	}
	if current.Kind != UserKindWorker {
		return Principal{}, attemptRecord{}, fault(CodeForbidden, "worker principal is required", nil)
	}
	attempt, err := readAttempt(ctx, q, attemptID)
	if err != nil {
		return Principal{}, attemptRecord{}, err
	}
	if attempt.DispatchID != dispatchID || attempt.WorkerID == nil || *attempt.WorkerID != current.UserID {
		return Principal{}, attemptRecord{}, fault(CodeForbidden, "attempt does not belong to this worker dispatch", nil)
	}
	return current, attempt, nil
}

func authorizeWorkerStage(ctx context.Context, q queryer, actor Principal, attempt attemptRecord) error {
	capability := CapabilityImplementWorker
	if attempt.Stage == StageVerification {
		capability = CapabilityVerifyWorker
	}
	_, err := authorizeProject(ctx, q, actor, attempt.ProjectID, capability)
	return err
}

func principalCanReadProject(ctx context.Context, q queryer, actor Principal, projectID ProjectID) (bool, error) {
	if actor.InstallationAdmin {
		return true, nil
	}
	capabilities, err := readCapabilities(ctx, q, actor.UserID, projectID)
	if err != nil {
		return false, err
	}
	return hasCapability(capabilities, CapabilityRead), nil
}

func redactJobAcknowledgement(job Job) Job {
	return Job{ID: job.ID, Version: job.Version, ProjectID: job.ProjectID, Status: job.Status, UpdatedAt: job.UpdatedAt}
}

func validateAdmissionFresh(ctx context.Context, q queryer, projectID ProjectID, workItemID string, admission Admission) error {
	if err := checkSourcesUnchanged(admission.Sources); err != nil {
		return err
	}
	project, err := readProject(ctx, q, projectID)
	if err != nil {
		return err
	}
	result, err := orientation.Orient(ctx, orientation.Request{ProjectDir: project.BundlePath, AllowedSourceDirs: project.AllowedSourceRoots, MaxFiles: project.ReaderLimits.MaxFiles, MaxBytes: project.ReaderLimits.MaxBytes})
	if err != nil {
		return fault(CodeInternal, "re-evaluate admitted work item", err)
	}
	work, err := selectWork(result, workItemID)
	if err != nil {
		return err
	}
	if !result.InventoryComplete || work.Source != admission.TicketPath || work.Readiness != "ready" || !work.Eligible || work.FingerprintVersion == nil || *work.FingerprintVersion != admission.FingerprintVersion || work.TicketSHA256 == nil || *work.TicketSHA256 != admission.TicketSHA256 || work.CriteriaSHA256 == nil || *work.CriteriaSHA256 != admission.CriteriaSHA256 {
		return fault(CodeStale, "admitted work item is no longer ready with the captured requirements", nil)
	}
	return nil
}

func assignmentFor(ctx context.Context, q queryer, attempt attemptRecord, worker UserID, lease time.Time) (Assignment, error) {
	project, err := readProject(ctx, q, attempt.ProjectID)
	if err != nil {
		return Assignment{}, err
	}
	var key, environment string
	var command []string
	var fixture bool
	if attempt.Stage == StageImplementation {
		key = attempt.Admission.ExecutorKey
		var encoded string
		if err := q.QueryRowContext(ctx, `SELECT command_json,fixture FROM executor_profiles WHERE key=?`, key).Scan(&encoded, &fixture); err != nil {
			return Assignment{}, fault(CodeInternal, "read executor assignment", err)
		}
		_ = json.Unmarshal([]byte(encoded), &command)
	} else {
		key = attempt.CheckKey
		var encoded string
		if err := q.QueryRowContext(ctx, `SELECT command_json,environment,fixture FROM check_profiles WHERE key=?`, key).Scan(&encoded, &environment, &fixture); err != nil {
			return Assignment{}, fault(CodeInternal, "read check assignment", err)
		}
		_ = json.Unmarshal([]byte(encoded), &command)
	}
	instructions := "Implement the captured work item and commit the complete candidate in the assigned workspace."
	if attempt.Stage == StageVerification {
		instructions = "Run only the configured required check in this clean candidate workspace; do not modify the candidate."
	}
	base := attempt.InputRevision
	deadline := attempt.Admission.Deadline
	if attempt.mergeCheck() {
		instructions = "Run only the configured required check on this clean merge commit; do not modify it."
		deadline = lease.Add(time.Duration(project.Policy.MaxJobDurationSeconds) * time.Second)
	}
	return Assignment{AttemptID: attempt.ID, JobID: attempt.JobID, ProjectID: attempt.ProjectID, WorkItemID: mustWorkItem(ctx, q, attempt.JobID), Stage: attempt.Stage, WorkerID: worker, DispatchID: attempt.DispatchID, WorkspacePath: attempt.WorkspacePath, BaseRevision: base, CandidateRevision: attempt.CandidateRevision, RequirementFingerprint: attempt.Admission.TicketSHA256, ContextSHA256: attempt.Admission.ContextSHA256, ContextSources: append([]ContextSource(nil), attempt.Admission.Sources...), Instructions: instructions, ProfileKey: key, Command: command, CheckEnvironment: environment, Fixture: project.Policy.Fixture && fixture, Deadline: deadline, LeaseExpiresAt: lease, ProtectedPaths: append([]string(nil), attempt.Admission.ProtectedPaths...)}, nil
}

func mustWorkItem(ctx context.Context, q queryer, jobID JobID) string {
	var value string
	_ = q.QueryRowContext(ctx, `SELECT work_item_id FROM jobs WHERE id=?`, jobID).Scan(&value)
	return value
}

func checkProfile(ctx context.Context, q queryer, key string) ([]string, string, error) {
	var encoded, environment string
	if err := q.QueryRowContext(ctx, `SELECT command_json,environment FROM check_profiles WHERE key=?`, key).Scan(&encoded, &environment); err != nil {
		return nil, "", fault(CodeInternal, "read check profile", err)
	}
	var command []string
	if err := json.Unmarshal([]byte(encoded), &command); err != nil {
		return nil, "", fault(CodeInternal, "decode check command", err)
	}
	return command, environment, nil
}

func recordCheckResult(ctx context.Context, tx *sql.Tx, observer UserID, attempt attemptRecord, status CheckStatus, observed, now time.Time) error {
	command, environment, err := checkProfile(ctx, tx, attempt.CheckKey)
	if err != nil {
		return err
	}
	commandJSON, _ := marshal(command)
	resultID, err := newID("chk_")
	if err != nil {
		return fault(CodeInternal, "create check result", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO check_results(id,job_id,check_key,status,candidate_revision,requirement_fingerprint,observer_id,attempt_id,command_json,environment,observed_at,created_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`, resultID, attempt.JobID, attempt.CheckKey, status, attempt.CandidateRevision, attempt.Admission.TicketSHA256, observer, attempt.ID, commandJSON, environment, formatTime(observed), formatTime(now)); err != nil {
		return fault(CodeInternal, "record verification result", err)
	}
	return nil
}

func (s *Service) ensureWorkspace(ctx context.Context, attempt attemptRecord) error {
	revision := attempt.InputRevision
	if attempt.Stage == StageVerification {
		revision = attempt.CandidateRevision
	}
	if _, err := os.Stat(attempt.WorkspacePath); err == nil {
		return s.verifyWorkspace(ctx, attempt.WorkspacePath, revision)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fault(CodeInternal, "inspect assigned workspace", err)
	}
	if err := os.MkdirAll(filepath.Dir(attempt.WorkspacePath), 0o700); err != nil {
		return fault(CodeInternal, "create workspace parent", err)
	}
	command := exec.CommandContext(ctx, s.state.gitBinary, "-C", attempt.Repository, "worktree", "add", "--detach", attempt.WorkspacePath, revision)
	if output, err := command.CombinedOutput(); err != nil {
		return detailFault(CodeInternal, "allocate Git worktree", map[string]any{"output": strings.TrimSpace(string(output))}, err)
	}
	if err := s.verifyWorkspace(ctx, attempt.WorkspacePath, revision); err != nil {
		_ = os.RemoveAll(filepath.Dir(attempt.WorkspacePath))
		return err
	}
	return nil
}

func (s *Service) verifyWorkspace(ctx context.Context, path, revision string) error {
	head, err := s.git(ctx, path, "rev-parse", "HEAD")
	if err != nil || head != revision {
		return fault(CodeConflict, "workspace does not match its recorded revision", err)
	}
	status, err := s.git(ctx, path, "status", "--porcelain")
	if err != nil || status != "" {
		return fault(CodeConflict, "workspace is not clean", err)
	}
	return nil
}

func (s *Service) validateImplementationCandidate(ctx context.Context, attempt attemptRecord, candidate string) (string, error) {
	if candidate == "" {
		return "", fault(CodeInvalid, "successful implementation requires candidateRevision", nil)
	}
	resolved, err := s.resolveCommit(ctx, attempt.WorkspacePath, candidate)
	if err != nil {
		return "", err
	}
	if err := s.verifyWorkspace(ctx, attempt.WorkspacePath, resolved); err != nil {
		return "", err
	}
	command := exec.CommandContext(ctx, s.state.gitBinary, "-C", attempt.WorkspacePath, "merge-base", "--is-ancestor", attempt.InputRevision, resolved)
	if err := command.Run(); err != nil || resolved == attempt.InputRevision {
		return "", fault(CodeConflict, "candidate must be a new descendant of the recorded base", err)
	}
	return resolved, nil
}

func (s *Service) validateVerificationWorkspace(ctx context.Context, attempt attemptRecord) error {
	return s.verifyWorkspace(ctx, attempt.WorkspacePath, attempt.CandidateRevision)
}

func (s *Service) git(ctx context.Context, directory string, args ...string) (string, error) {
	command := exec.CommandContext(ctx, s.state.gitBinary, append([]string{"-C", directory}, args...)...)
	output, err := command.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, output)
	}
	return strings.TrimSpace(string(output)), nil
}

func validateRunCorrelation(attempt attemptRecord, request WorkerReportRequest) error {
	if attempt.NativeRunID == "" || request.NativeRunID != attempt.NativeRunID {
		return fault(CodeConflict, "terminal report does not match the registered native run", nil)
	}
	if attempt.Status != AttemptStartRegistered && attempt.Status != AttemptRunning && attempt.Status != AttemptCancelAcknowledged && attempt.Status != AttemptUnknown {
		return fault(CodeConflict, "attempt cannot accept a terminal report in its current state", nil)
	}
	return nil
}

func storeArtifacts(ctx context.Context, tx *sql.Tx, jobID JobID, attemptID AttemptID, input []ArtifactInput, now time.Time) error {
	if len(input) > 32 {
		return fault(CodeInvalid, "too many worker artifacts", nil)
	}
	for _, item := range input {
		if strings.TrimSpace(item.Name) == "" || strings.TrimSpace(item.MediaType) == "" || len(item.Content) > 2<<20 {
			return fault(CodeInvalid, "worker artifact is missing metadata or exceeds 2 MiB", nil)
		}
		id, err := newID("art_")
		if err != nil {
			return fault(CodeInternal, "create artifact identity", err)
		}
		digest := sha256.Sum256([]byte(item.Content))
		if _, err := tx.ExecContext(ctx, `INSERT INTO artifacts(id,job_id,attempt_id,name,media_type,sha256,content,byte_count,created_at) VALUES(?,?,?,?,?,?,?,?,?)`, id, jobID, attemptID, item.Name, item.MediaType, hex.EncodeToString(digest[:]), item.Content, len(item.Content), formatTime(now)); err != nil {
			return fault(CodeInternal, "retain worker artifact", err)
		}
	}
	return nil
}

func terminalAttempt(status AttemptStatus) bool {
	return status == AttemptSucceeded || status == AttemptFailed || status == AttemptCancelled
}

func placeholders(count int) string {
	values := make([]string, count)
	for index := range values {
		values[index] = "?"
	}
	return strings.Join(values, ",")
}

func expireLeases(ctx context.Context, tx *sql.Tx, now time.Time) error {
	rows, err := tx.QueryContext(ctx, `SELECT id,job_id,stage,cancellation_requested,lease_expires_at,purpose,candidate_revision,ordinal FROM attempts WHERE status IN (?,?,?,?) AND lease_expires_at IS NOT NULL`, AttemptAssigned, AttemptStartRegistered, AttemptRunning, AttemptCancelAcknowledged)
	if err != nil {
		return fault(CodeInternal, "find expired worker leases", err)
	}
	type expired struct {
		attempt AttemptID
		job     JobID
		stage   Stage
		cancel  bool
		lease   string
		purpose string
		commit  string
		ordinal int
	}
	values := []expired{}
	for rows.Next() {
		var value expired
		if err := rows.Scan(&value.attempt, &value.job, &value.stage, &value.cancel, &value.lease, &value.purpose, &value.commit, &value.ordinal); err != nil {
			rows.Close()
			return fault(CodeInternal, "read worker lease", err)
		}
		deadline, err := parseTime(value.lease)
		if err != nil || !now.Before(deadline) {
			values = append(values, value)
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return fault(CodeInternal, "read worker leases", err)
	}
	for _, value := range values {
		check := CheckStatus("")
		if value.stage == StageVerification {
			check = CheckUnknown
		}
		if _, err := tx.ExecContext(ctx, `UPDATE attempts SET status=?,outcome=?,check_status=? WHERE id=?`, AttemptUnknown, OutcomeUnknown, check, value.attempt); err != nil {
			return fault(CodeInternal, "mark expired worker lease uncertain", err)
		}
		if value.purpose == AttemptPurposeMerge {
			if err := failMergeVerification(ctx, tx, value.job, value.commit, value.ordinal, "a merge check lease expired", now); err != nil {
				return err
			}
			continue
		}
		jobStatus := JobImplementationUnknown
		if value.cancel {
			jobStatus = JobCancellationUnknown
		}
		if _, err := tx.ExecContext(ctx, `UPDATE jobs SET status=?,version=version+1,updated_at=? WHERE id=?`, jobStatus, formatTime(now), value.job); err != nil {
			return fault(CodeInternal, "mark expired job uncertain", err)
		}
		if _, err := appendEvent(ctx, tx, value.job, nil, "worker_lease_expired", map[string]any{"attemptId": value.attempt}, now); err != nil {
			return err
		}
	}
	return nil
}

func (state *serviceState) recoverInterruptedAttempts(ctx context.Context) error {
	tx, err := state.db.BeginTx(ctx, nil)
	if err != nil {
		return fault(CodeInternal, "start interrupted-attempt recovery", err)
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx, `SELECT id,job_id,cancellation_requested,purpose,candidate_revision,ordinal FROM attempts WHERE status IN (?,?,?)`, AttemptStartRegistered, AttemptRunning, AttemptCancelAcknowledged)
	if err != nil {
		return fault(CodeInternal, "find interrupted attempts", err)
	}
	type interrupted struct {
		attempt AttemptID
		job     JobID
		cancel  bool
		purpose string
		commit  string
		ordinal int
	}
	var found []interrupted
	for rows.Next() {
		var value interrupted
		if err := rows.Scan(&value.attempt, &value.job, &value.cancel, &value.purpose, &value.commit, &value.ordinal); err != nil {
			rows.Close()
			return fault(CodeInternal, "read interrupted attempt", err)
		}
		found = append(found, value)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return fault(CodeInternal, "read interrupted attempts", err)
	}
	now := state.now().UTC()
	for _, value := range found {
		if _, err := tx.ExecContext(ctx, `UPDATE attempts SET status=?,outcome=?,check_status=CASE WHEN stage=? THEN ? ELSE check_status END WHERE id=?`, AttemptUnknown, OutcomeUnknown, StageVerification, CheckUnknown, value.attempt); err != nil {
			return fault(CodeInternal, "retain interrupted attempt", err)
		}
		if value.purpose == AttemptPurposeMerge {
			if err := failMergeVerification(ctx, tx, value.job, value.commit, value.ordinal, "the controller restarted while a merge check ran", now); err != nil {
				return err
			}
			continue
		}
		jobStatus := JobImplementationUnknown
		if value.cancel {
			jobStatus = JobCancellationUnknown
		}
		if _, err := tx.ExecContext(ctx, `UPDATE jobs SET status=?,version=version+1,updated_at=? WHERE id=?`, jobStatus, formatTime(now), value.job); err != nil {
			return fault(CodeInternal, "retain interrupted job", err)
		}
		if _, err := appendEvent(ctx, tx, value.job, nil, "controller_restart_uncertain", map[string]any{"attemptId": value.attempt}, now); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return fault(CodeInternal, "commit interrupted-attempt recovery", err)
	}
	return nil
}

const maxProgressRunes = 300

// recordProgress appends a worker_progress event when the running command reports a
// note that differs from its previous one.
func recordProgress(ctx context.Context, tx *sql.Tx, attempt attemptRecord, progress string, now time.Time) error {
	progress = strings.TrimSpace(progress)
	if progress == "" {
		return nil
	}
	if runes := []rune(progress); len(runes) > maxProgressRunes {
		progress = string(runes[:maxProgressRunes])
	}
	var last string
	if err := tx.QueryRowContext(ctx, `SELECT progress FROM attempts WHERE id=?`, attempt.ID).Scan(&last); err != nil {
		return fault(CodeInternal, "read attempt progress", err)
	}
	if last == progress {
		return nil
	}
	if _, err := tx.ExecContext(ctx, `UPDATE attempts SET progress=? WHERE id=?`, progress, attempt.ID); err != nil {
		return fault(CodeInternal, "save attempt progress", err)
	}
	_, err := appendEvent(ctx, tx, attempt.JobID, nil, "worker_progress", map[string]any{"attemptId": attempt.ID, "progress": progress}, now)
	return err
}

// mergeCheck reports whether the attempt verifies a merge commit rather than the job's
// candidate. Merge checks never change the job's own status or deadline handling.
func (attempt attemptRecord) mergeCheck() bool { return attempt.Purpose == AttemptPurposeMerge }

// claimFresh requires unchanged admitted sources for candidate work. A merge check runs
// after acceptance, which legitimately rewrote the ticket; Merge checked that the
// acceptance is still valid instead.
func claimFresh(ctx context.Context, q queryer, attempt attemptRecord) error {
	if attempt.mergeCheck() {
		return nil
	}
	return validateAdmissionFresh(ctx, q, attempt.ProjectID, mustWorkItem(ctx, q, attempt.JobID), attempt.Admission)
}
