package factory

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
)

func (s *Service) Action(ctx context.Context, actor Principal, request JobActionRequest) (Job, error) {
	if request.JobID == "" || request.IfVersion <= 0 || request.Action == "" {
		return Job{}, fault(CodeInvalid, "jobId, ifVersion, and action are required", nil)
	}
	s.state.mutations.Lock()
	defer s.state.mutations.Unlock()
	tx, err := s.state.db.BeginTx(ctx, nil)
	if err != nil {
		return Job{}, fault(CodeInternal, "start job action", err)
	}
	defer tx.Rollback()
	projectID, err := jobProject(ctx, tx, request.JobID)
	if err != nil {
		return Job{}, err
	}
	capability := CapabilityCoordinate
	if request.Action == ActionResolve {
		capability = CapabilityProjectAdmin
	}
	current, err := authorizeProject(ctx, tx, actor, projectID, capability)
	if err != nil {
		return Job{}, err
	}
	if saved, ok, hash, err := receipt[Job](ctx, tx, current.UserID, "job_action", request.RequestID, request); err != nil || ok {
		return saved, err
	} else {
		job, err := loadJob(ctx, tx, request.JobID)
		if err != nil {
			return Job{}, err
		}
		if job.Version != request.IfVersion {
			return Job{}, detailFault(CodeConflict, "job version changed", map[string]any{"currentVersion": job.Version}, nil)
		}
		now := s.state.now().UTC()
		switch request.Action {
		case ActionCancel:
			err = s.cancelJob(ctx, tx, job, now)
		case ActionRepair:
			err = s.repairJob(ctx, tx, job, now)
		case ActionReconcile:
			if job.Status != JobImplementationUnknown && job.Status != JobCancellationUnknown {
				err = fault(CodeConflict, "job has no uncertain execution to reconcile", nil)
			} else {
				_, err = appendEvent(ctx, tx, job.ID, &current.UserID, "reconciliation_requested", map[string]any{}, now)
			}
		case ActionResolve:
			err = s.resolveExecution(ctx, tx, job, request, now)
		default:
			err = fault(CodeInvalid, "unknown job action", nil)
		}
		if err != nil {
			return Job{}, err
		}
		if _, err := appendEvent(ctx, tx, job.ID, &current.UserID, "job_action", map[string]any{"action": request.Action, "attemptId": request.AttemptID, "resolution": request.Resolution}, now); err != nil {
			return Job{}, err
		}
		job, err = loadJob(ctx, tx, job.ID)
		if err != nil {
			return Job{}, err
		}
		job.AvailableActions = availableActions(job)
		if err := saveReceipt(ctx, tx, current.UserID, "job_action", request.RequestID, hash, job, false, now); err != nil {
			return Job{}, err
		}
		if err := tx.Commit(); err != nil {
			return Job{}, fault(CodeInternal, "commit job action", err)
		}
		return job, nil
	}
}

func (s *Service) cancelJob(ctx context.Context, tx *sql.Tx, job Job, now time.Time) error {
	if job.Status == JobAccepting {
		return fault(CodeConflict, "acceptance publication must be reconciled before cancellation", nil)
	}
	if job.Status == JobAccepted || job.Status == JobCancelled {
		return fault(CodeConflict, "job is already terminal", nil)
	}
	live := []AttemptID{}
	for _, attempt := range job.Attempts {
		switch attempt.Status {
		case AttemptUnknown:
			return fault(CodeUncertain, "uncertain execution must be resolved before cancellation can complete", nil)
		case AttemptAssigned, AttemptStartRegistered, AttemptRunning, AttemptCancelAcknowledged:
			live = append(live, attempt.ID)
		}
	}
	if len(live) == 0 {
		if _, err := tx.ExecContext(ctx, `UPDATE attempts SET status=?,outcome=?,finished_at=? WHERE job_id=? AND status=?`, AttemptCancelled, OutcomeCancelled, formatTime(now), job.ID, AttemptPending); err != nil {
			return fault(CodeInternal, "cancel pending attempts", err)
		}
		if _, err := tx.ExecContext(ctx, `UPDATE jobs SET status=?,active_claim=0,version=version+1,updated_at=? WHERE id=?`, JobCancelled, formatTime(now), job.ID); err != nil {
			return fault(CodeInternal, "cancel queued job", err)
		}
		return nil
	}
	if _, err := tx.ExecContext(ctx, `UPDATE attempts SET status=?,outcome=?,finished_at=? WHERE job_id=? AND status=?`, AttemptCancelled, OutcomeCancelled, formatTime(now), job.ID, AttemptPending); err != nil {
		return fault(CodeInternal, "cancel unclaimed attempts", err)
	}
	for _, attemptID := range live {
		if _, err := tx.ExecContext(ctx, `UPDATE attempts SET cancellation_requested=1 WHERE id=?`, attemptID); err != nil {
			return fault(CodeInternal, "request attempt cancellation", err)
		}
	}
	if _, err := tx.ExecContext(ctx, `UPDATE jobs SET status=?,version=version+1,updated_at=? WHERE id=?`, JobCancelRequested, formatTime(now), job.ID); err != nil {
		return fault(CodeInternal, "request job cancellation", err)
	}
	return nil
}

func (s *Service) repairJob(ctx context.Context, tx *sql.Tx, job Job, now time.Time) error {
	if job.Status != JobFailed && job.Status != JobVerificationFailed {
		return fault(CodeConflict, "only a failed implementation or verification can be repaired", nil)
	}
	if !now.Before(job.Admission.Deadline) {
		return fault(CodeDeadline, "job deadline has elapsed", nil)
	}
	if job.RepairCount >= job.Admission.RepairLimit {
		return fault(CodeExhausted, "job repair limit is exhausted", nil)
	}
	for _, attempt := range job.Attempts {
		if attempt.Status == AttemptUnknown || attempt.Status == AttemptStartRegistered || attempt.Status == AttemptRunning || attempt.Status == AttemptCancelAcknowledged {
			return fault(CodeUncertain, "unresolved execution blocks repair", nil)
		}
	}
	if err := checkSourcesUnchanged(job.Admission.Sources); err != nil {
		return err
	}
	input := job.Admission.BaseRevision
	if job.CandidateRevision != "" {
		input = job.CandidateRevision
	}
	id, err := newID("att_")
	if err != nil {
		return fault(CodeInternal, "create repair attempt", err)
	}
	dispatch, err := newID("dsp_")
	if err != nil {
		return fault(CodeInternal, "create repair dispatch", err)
	}
	var ordinal int
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(max(ordinal),0)+1 FROM attempts WHERE job_id=? AND stage=?`, job.ID, StageImplementation).Scan(&ordinal); err != nil {
		return fault(CodeInternal, "number repair attempt", err)
	}
	path := s.workspacePath(job.ID, AttemptID(id))
	if _, err := tx.ExecContext(ctx, `INSERT INTO attempts(id,job_id,stage,ordinal,dispatch_id,workspace_path,input_revision,status) VALUES(?,?,?,?,?,?,?,?)`, id, job.ID, StageImplementation, ordinal, dispatch, path, input, AttemptPending); err != nil {
		return fault(CodeInternal, "queue repair attempt", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE jobs SET status=?,candidate_revision='',repair_count=repair_count+1,version=version+1,updated_at=? WHERE id=?`, JobQueued, formatTime(now), job.ID); err != nil {
		return fault(CodeInternal, "queue job repair", err)
	}
	return nil
}

func (s *Service) resolveExecution(ctx context.Context, tx *sql.Tx, job Job, request JobActionRequest, now time.Time) error {
	if request.AttemptID == nil || strings.TrimSpace(request.Evidence) == "" {
		return fault(CodeInvalid, "attemptId and evidence are required to resolve execution", nil)
	}
	attempt, err := readAttempt(ctx, tx, *request.AttemptID)
	if err != nil {
		return err
	}
	if attempt.JobID != job.ID || attempt.Status != AttemptUnknown {
		return fault(CodeConflict, "resolution must identify this job's uncertain attempt", nil)
	}
	switch request.Resolution {
	case ResolutionStillRunning:
		if attempt.NativeRunID == "" {
			return fault(CodeConflict, "attempt has no registered native run", nil)
		}
		lease := now.Add(s.state.leaseDuration)
		if _, err := tx.ExecContext(ctx, `UPDATE attempts SET status=?,outcome='',lease_expires_at=?,last_heartbeat_at=? WHERE id=?`, AttemptRunning, formatTime(lease), formatTime(now), attempt.ID); err != nil {
			return fault(CodeInternal, "restore running attempt", err)
		}
		status := JobImplementing
		if attempt.Stage == StageVerification {
			status = JobVerifying
		}
		if attempt.CancellationRequested {
			status = JobCancelRequested
		}
		_, err = tx.ExecContext(ctx, `UPDATE jobs SET status=?,version=version+1,updated_at=? WHERE id=?`, status, formatTime(now), job.ID)
	case ResolutionNotStarted:
		newAttempt, idErr := newID("att_")
		if idErr != nil {
			return fault(CodeInternal, "create replacement attempt", idErr)
		}
		dispatch, idErr := newID("dsp_")
		if idErr != nil {
			return fault(CodeInternal, "rotate dispatch identity", idErr)
		}
		var ordinal int
		if err := tx.QueryRowContext(ctx, `SELECT COALESCE(max(ordinal),0)+1 FROM attempts WHERE job_id=? AND stage=?`, job.ID, attempt.Stage).Scan(&ordinal); err != nil {
			return fault(CodeInternal, "number replacement attempt", err)
		}
		if _, err := tx.ExecContext(ctx, `UPDATE attempts SET status=?,outcome=?,finished_at=? WHERE id=?`, AttemptCancelled, OutcomeCancelled, formatTime(now), attempt.ID); err != nil {
			return fault(CodeInternal, "seal unstarted attempt", err)
		}
		workspace := s.workspacePath(job.ID, AttemptID(newAttempt))
		if _, err := tx.ExecContext(ctx, `INSERT INTO attempts(id,job_id,stage,ordinal,check_key,dispatch_id,workspace_path,input_revision,candidate_revision,status) VALUES(?,?,?,?,?,?,?,?,?,?)`, newAttempt, job.ID, attempt.Stage, ordinal, attempt.CheckKey, dispatch, workspace, attempt.InputRevision, attempt.CandidateRevision, AttemptPending); err != nil {
			return fault(CodeInternal, "queue replacement attempt", err)
		}
		if _, err := appendEvent(ctx, tx, job.ID, nil, "replacement_attempt_created", map[string]any{"priorAttemptId": attempt.ID, "attemptId": newAttempt, "stage": attempt.Stage}, now); err != nil {
			return err
		}
		status := JobQueued
		if attempt.Stage == StageVerification {
			status = JobVerifying
		}
		_, err = tx.ExecContext(ctx, `UPDATE jobs SET status=?,version=version+1,updated_at=? WHERE id=?`, status, formatTime(now), job.ID)
	case ResolutionFailed:
		if _, err := tx.ExecContext(ctx, `UPDATE attempts SET status=?,outcome=?,finished_at=? WHERE id=?`, AttemptFailed, OutcomeFailed, formatTime(now), attempt.ID); err != nil {
			return fault(CodeInternal, "resolve failed attempt", err)
		}
		status := JobFailed
		if attempt.Stage == StageVerification {
			status = JobVerificationFailed
		}
		_, err = tx.ExecContext(ctx, `UPDATE jobs SET status=?,version=version+1,updated_at=? WHERE id=?`, status, formatTime(now), job.ID)
	case ResolutionCancelled:
		if _, err := tx.ExecContext(ctx, `UPDATE attempts SET status=?,outcome=?,finished_at=? WHERE id=?`, AttemptCancelled, OutcomeCancelled, formatTime(now), attempt.ID); err != nil {
			return fault(CodeInternal, "resolve cancelled attempt", err)
		}
		err = s.updateJobAfterResolvedCancellation(ctx, tx, job, attempt, now)
	default:
		return fault(CodeInvalid, "unknown execution resolution", nil)
	}
	if err != nil {
		return fault(CodeInternal, "update resolved job", err)
	}
	if _, err := appendEvent(ctx, tx, job.ID, nil, "execution_resolved", map[string]any{"attemptId": attempt.ID, "resolution": request.Resolution, "evidence": strings.TrimSpace(request.Evidence)}, now); err != nil {
		return err
	}
	return nil
}

func (s *Service) updateJobAfterResolvedCancellation(ctx context.Context, tx *sql.Tx, job Job, resolved attemptRecord, now time.Time) error {
	var pending, active, uncertain, verificationOutstanding, cancellationOutstanding int
	err := tx.QueryRowContext(ctx, `SELECT
		COALESCE(sum(CASE WHEN status=? THEN 1 ELSE 0 END),0),
		COALESCE(sum(CASE WHEN status IN (?,?,?,?) THEN 1 ELSE 0 END),0),
		COALESCE(sum(CASE WHEN status=? THEN 1 ELSE 0 END),0),
		COALESCE(sum(CASE WHEN stage=? AND status IN (?,?,?,?,?,?) THEN 1 ELSE 0 END),0),
		COALESCE(sum(CASE WHEN cancellation_requested=1 AND status IN (?,?,?,?,?,?) THEN 1 ELSE 0 END),0)
		FROM attempts WHERE job_id=?`,
		AttemptPending,
		AttemptAssigned, AttemptStartRegistered, AttemptRunning, AttemptCancelAcknowledged,
		AttemptUnknown,
		StageVerification, AttemptPending, AttemptAssigned, AttemptStartRegistered, AttemptRunning, AttemptCancelAcknowledged, AttemptUnknown,
		AttemptPending, AttemptAssigned, AttemptStartRegistered, AttemptRunning, AttemptCancelAcknowledged, AttemptUnknown,
		job.ID,
	).Scan(&pending, &active, &uncertain, &verificationOutstanding, &cancellationOutstanding)
	if err != nil {
		return fault(CodeInternal, "inspect attempts after cancellation resolution", err)
	}

	status := JobCancelled
	activeClaim := 0
	globalCancellation := resolved.CancellationRequested || job.Status == JobCancelRequested || job.Status == JobCancellationUnknown || cancellationOutstanding > 0
	if globalCancellation {
		activeClaim = 1
		switch {
		case uncertain > 0:
			status = JobCancellationUnknown
		case pending+active > 0:
			status = JobCancelRequested
		default:
			status, activeClaim = JobCancelled, 0
		}
	} else {
		activeClaim = 1
		switch {
		case uncertain > 0:
			status = JobImplementationUnknown
		case verificationOutstanding > 0:
			status = JobVerifying
		case active > 0:
			status = JobImplementing
		case pending > 0:
			status = JobQueued
		case resolved.Stage == StageVerification:
			status = JobVerificationFailed
		default:
			status, activeClaim = JobCancelled, 0
		}
	}
	if _, err := tx.ExecContext(ctx, `UPDATE jobs SET status=?,active_claim=?,version=version+1,updated_at=? WHERE id=?`, status, activeClaim, formatTime(now), job.ID); err != nil {
		return fault(CodeInternal, "update job after cancellation resolution", err)
	}
	return nil
}

func latestUncertainAttempt(ctx context.Context, q queryer, jobID JobID) (AttemptID, error) {
	var id AttemptID
	err := q.QueryRowContext(ctx, `SELECT id FROM attempts WHERE job_id=? AND status=? ORDER BY ordinal DESC LIMIT 1`, jobID, AttemptUnknown).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return "", fault(CodeNotFound, "uncertain attempt not found", nil)
	}
	return id, err
}
