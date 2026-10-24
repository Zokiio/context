package factory

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Merge modes. Every merge commit is built in a temporary worktree and verified before
// a branch moves. Checkout then fast-forwards the project's own working tree, where the
// target branch is checked out. Ref moves a branch that no worktree has checked out.
// Contained records a branch that already contains the candidate, for example after a
// merge by hand.
const (
	MergeModeCheckout  = "checkout"
	MergeModeRef       = "ref"
	MergeModeContained = "contained"
)

// Merge states. A merge is verifying while the required checks run on the merge
// commit; only a merged record lets dependent work through admission.
const (
	MergeStatusVerifying = "verifying"
	MergeStatusMerged    = "merged"
	MergeStatusFailed    = "failed"
)

// AttemptPurposeMerge marks verification attempts that check a merge commit rather
// than the job's candidate. They never change the job's own status.
const AttemptPurposeMerge = "merge"

type MergeRecord struct {
	Status            string     `json:"status"`
	Branch            string     `json:"branch"`
	Revision          string     `json:"revision"`
	PreviousTip       string     `json:"previousTip"`
	CandidateRevision string     `json:"candidateRevision"`
	Mode              string     `json:"mode"`
	Checks            []string   `json:"checks,omitempty"`
	MergedBy          UserID     `json:"mergedBy"`
	RequestedAt       time.Time  `json:"requestedAt"`
	MergedAt          *time.Time `json:"mergedAt,omitempty"`
	Failure           string     `json:"failure,omitempty"`
	TicketNote        string     `json:"ticketNote,omitempty"`
}

// Merged reports whether the branch contains the candidate through this record.
// Records from before merge verification have no status and were merged directly.
func (record *MergeRecord) Merged() bool {
	return record != nil && (record.Status == MergeStatusMerged || record.Status == "")
}

type MergeRequest struct {
	RequestID         string `json:"requestId"`
	JobID             JobID  `json:"jobId"`
	IfVersion         int64  `json:"ifVersion"`
	CandidateRevision string `json:"candidateRevision"`
	// TargetBranch defaults to the branch checked out in the project repository.
	TargetBranch string `json:"targetBranch,omitempty"`
}

// Merge integrates an accepted candidate into a branch of the project repository after
// a human approves it. Acceptance must still be valid, and the merge commit names the
// job, the candidate, the acceptance record, and the approver.
func (s *Service) Merge(ctx context.Context, actor Principal, request MergeRequest) (Job, error) {
	if err := requireRequestID(request.RequestID); err != nil {
		return Job{}, err
	}
	// Reads and the Git work happen before the write transaction: the database has one
	// connection, and the acceptance check opens its own reads. The mutation lock keeps
	// other writers out for the whole operation.
	s.state.mutations.Lock()
	defer s.state.mutations.Unlock()
	job, err := loadJob(ctx, s.state.db, request.JobID)
	if err != nil {
		return Job{}, err
	}
	current, err := authorizeProject(ctx, s.state.db, actor, job.ProjectID, CapabilityMerge)
	if err != nil {
		return Job{}, err
	}
	if current.Kind != UserKindHuman {
		return Job{}, fault(CodeForbidden, "merge requires a human principal", nil)
	}
	if replayed, ok, err := s.readJobReceipt(ctx, current.UserID, "merge", request.RequestID, request); err != nil || ok {
		return replayed, err
	}
	if job.Version != request.IfVersion {
		return Job{}, detailFault(CodeConflict, "job version changed", map[string]any{"currentVersion": job.Version}, nil)
	}
	if job.Status != JobAccepted {
		return Job{}, fault(CodeNotReady, "only an accepted job can be merged", nil)
	}
	if job.Merge != nil && job.Merge.Status != MergeStatusFailed {
		return Job{}, detailFault(CodeConflict, "job is already merged or its merge is being verified", map[string]any{"status": job.Merge.Status, "branch": job.Merge.Branch, "revision": job.Merge.Revision}, nil)
	}
	if request.CandidateRevision != job.CandidateRevision {
		return Job{}, fault(CodeStale, "merge does not match the accepted candidate", nil)
	}
	if status := s.currentAcceptanceStatus(ctx, job); status != "valid" {
		return Job{}, detailFault(CodeNotReady, "acceptance must be valid before merging", map[string]any{"acceptanceStatus": status}, nil)
	}
	project, err := readProject(ctx, s.state.db, job.ProjectID)
	if err != nil {
		return Job{}, err
	}
	repository := project.RepositoryPath
	branch := strings.TrimSpace(request.TargetBranch)
	if branch == "" {
		if branch, err = s.git(ctx, repository, "symbolic-ref", "--quiet", "--short", "HEAD"); err != nil {
			return Job{}, fault(CodeInvalid, "the project repository is on a detached HEAD; name a target branch", err)
		}
	}
	if _, err := s.git(ctx, repository, "check-ref-format", "--branch", branch); err != nil {
		return Job{}, fault(CodeInvalid, "target branch name is invalid", err)
	}
	candidate, err := s.resolveCommit(ctx, repository, job.CandidateRevision)
	if err != nil {
		return Job{}, err
	}
	previous, err := s.git(ctx, repository, "rev-parse", "--verify", "--quiet", "refs/heads/"+branch+"^{commit}")
	if err != nil {
		return Job{}, fault(CodeInvalid, "target branch does not exist", err)
	}
	now := s.state.now().UTC()
	record := MergeRecord{Branch: branch, PreviousTip: previous, CandidateRevision: candidate, MergedBy: current.UserID, RequestedAt: now}
	if s.isAncestor(ctx, repository, candidate, previous) {
		record.Mode, record.Revision, record.Status, record.MergedAt = MergeModeContained, previous, MergeStatusMerged, &now
		record.TicketNote = s.noteMerge(job, record, current.Username)
	} else {
		checkout, err := s.branchCheckout(ctx, repository, branch)
		if err != nil {
			return Job{}, err
		}
		switch {
		case checkout == "":
			record.Mode = MergeModeRef
		case sameDirectory(checkout, repository):
			record.Mode = MergeModeCheckout
		default:
			err = detailFault(CodeConflict, "target branch is checked out in another worktree; merge there or choose another branch", map[string]any{"worktree": checkout}, nil)
		}
		if err == nil {
			record.Revision, err = s.buildMerge(ctx, repository, previous, candidate, mergeMessage(job, current.Username, candidate, repository), current.Username, job.ID)
		}
		if err != nil {
			s.recordMergeFailure(ctx, job.ID, current.UserID, branch, candidate, err, now)
			return Job{}, err
		}
		record.Status, record.Checks = MergeStatusVerifying, append([]string(nil), project.Policy.RequiredChecks...)
	}
	tx, err := s.state.db.BeginTx(ctx, nil)
	if err != nil {
		return Job{}, fault(CodeInternal, "start merge record", err)
	}
	defer tx.Rollback()
	saved, ok, hash, err := receipt[Job](ctx, tx, current.UserID, "merge", request.RequestID, request)
	if err != nil || ok {
		return saved, err
	}
	if record.Status == MergeStatusVerifying {
		if err := s.queueMergeChecks(ctx, tx, job.ID, record.Revision, record.Checks); err != nil {
			return Job{}, err
		}
	}
	event := "job_merged"
	if record.Status == MergeStatusVerifying {
		event = "merge_verifying"
	}
	if err := saveMergeRecord(ctx, tx, job.ID, &current.UserID, record, event, now); err != nil {
		return Job{}, err
	}
	merged, err := loadJob(ctx, tx, job.ID)
	if err != nil {
		return Job{}, err
	}
	merged.AcceptanceStatus = "valid"
	merged.AvailableActions = availableActions(merged)
	if err := saveReceipt(ctx, tx, current.UserID, "merge", request.RequestID, hash, merged, false, now); err != nil {
		return Job{}, err
	}
	if err := tx.Commit(); err != nil {
		return Job{}, fault(CodeInternal, "commit merge", err)
	}
	return merged, nil
}

// queueMergeChecks asks verification workers to run every required check on the merge
// commit, in clean worktrees, before any branch moves.
func (s *Service) queueMergeChecks(ctx context.Context, tx *sql.Tx, jobID JobID, revision string, checks []string) error {
	var ordinal int
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(max(ordinal),0)+1 FROM attempts WHERE job_id=? AND stage=?`, jobID, StageVerification).Scan(&ordinal); err != nil {
		return fault(CodeInternal, "number merge verification", err)
	}
	for _, check := range checks {
		id, err := newID("att_")
		if err != nil {
			return fault(CodeInternal, "create merge check", err)
		}
		dispatch, err := newID("dsp_")
		if err != nil {
			return fault(CodeInternal, "create merge check dispatch", err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO attempts(id,job_id,stage,ordinal,check_key,dispatch_id,workspace_path,input_revision,candidate_revision,status,purpose) VALUES(?,?,?,?,?,?,?,?,?,?,?)`, id, jobID, StageVerification, ordinal, check, dispatch, s.workspacePath(jobID, AttemptID(id)), revision, revision, AttemptPending, AttemptPurposeMerge); err != nil {
			return fault(CodeInternal, "queue merge check", err)
		}
	}
	return nil
}

func saveMergeRecord(ctx context.Context, tx *sql.Tx, jobID JobID, actor *UserID, record MergeRecord, event string, now time.Time) error {
	encoded, err := marshal(record)
	if err != nil {
		return fault(CodeInternal, "encode merge", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE jobs SET merge_json=?, version=version+1, updated_at=? WHERE id=?`, encoded, formatTime(now), jobID); err != nil {
		return fault(CodeInternal, "record merge", err)
	}
	_, err = appendEvent(ctx, tx, jobID, actor, event, record, now)
	return err
}

func loadMergeRecord(ctx context.Context, q queryer, jobID JobID) (*MergeRecord, error) {
	var encoded string
	if err := q.QueryRowContext(ctx, `SELECT merge_json FROM jobs WHERE id=?`, jobID).Scan(&encoded); err != nil {
		return nil, fault(CodeInternal, "read merge", err)
	}
	if encoded == "" {
		return nil, nil
	}
	record := &MergeRecord{}
	if err := json.Unmarshal([]byte(encoded), record); err != nil {
		return nil, fault(CodeInternal, "decode merge", err)
	}
	return record, nil
}

// currentMergeGeneration is the ordinal of the latest queued merge checks. A retried
// merge can rebuild an identical commit, so only this generation decides the merge.
func currentMergeGeneration(ctx context.Context, q queryer, jobID JobID) (int, error) {
	var ordinal int
	if err := q.QueryRowContext(ctx, `SELECT COALESCE(max(ordinal),0) FROM attempts WHERE job_id=? AND purpose=?`, jobID, AttemptPurposeMerge).Scan(&ordinal); err != nil {
		return 0, fault(CodeInternal, "read merge generation", err)
	}
	return ordinal, nil
}

// failMergeVerification ends a verifying merge without moving any branch and stops
// its unclaimed checks. It is used for failed, unknown, expired, and interrupted checks
// of the current generation; older generations no longer decide anything.
func failMergeVerification(ctx context.Context, tx *sql.Tx, jobID JobID, revision string, ordinal int, reason string, now time.Time) error {
	record, err := loadMergeRecord(ctx, tx, jobID)
	if err != nil || record == nil || record.Status != MergeStatusVerifying || record.Revision != revision {
		return err
	}
	if current, err := currentMergeGeneration(ctx, tx, jobID); err != nil || current != ordinal {
		return err
	}
	record.Status, record.Failure = MergeStatusFailed, reason
	if _, err := tx.ExecContext(ctx, `UPDATE attempts SET status=? WHERE job_id=? AND purpose=? AND status=?`, AttemptCancelled, jobID, AttemptPurposeMerge, AttemptPending); err != nil {
		return fault(CodeInternal, "stop merge checks", err)
	}
	return saveMergeRecord(ctx, tx, jobID, nil, *record, "merge_failed", now)
}

// finishMergeVerification records one merge check. When every required check has passed
// on the merge commit, the branch moves; any other outcome fails the merge.
func (s *Service) finishMergeVerification(ctx context.Context, tx *sql.Tx, actor Principal, attempt attemptRecord, request WorkerReportRequest, now time.Time) error {
	if request.CandidateRevision == "" {
		request.CandidateRevision = attempt.CandidateRevision
	}
	if request.CandidateRevision != attempt.CandidateRevision {
		return fault(CodeConflict, "verification report does not match the merge commit", nil)
	}
	if err := s.validateVerificationWorkspace(ctx, attempt); err != nil {
		request.Outcome, request.CheckStatus = OutcomeUnknown, CheckUnknown
	}
	if request.Outcome != OutcomeSucceeded && request.CheckStatus == CheckPassed {
		request.CheckStatus = CheckFailed
	}
	status := AttemptFailed
	switch {
	case request.Outcome == OutcomeSucceeded && request.CheckStatus == CheckPassed:
		status = AttemptSucceeded
	case request.Outcome == OutcomeCancelled:
		status = AttemptCancelled
	case request.Outcome == OutcomeUnknown || request.CheckStatus == CheckUnknown || request.CheckStatus == "":
		status, request.CheckStatus = AttemptUnknown, CheckUnknown
	}
	if err := recordCheckResult(ctx, tx, actor.UserID, attempt, request.CheckStatus, request.ObservedAt, now); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE attempts SET status=?, outcome=?, check_status=?, terminal_receipt=?, finished_at=? WHERE id=?`, status, request.Outcome, request.CheckStatus, request.TerminalReceipt, formatTime(request.ObservedAt), attempt.ID); err != nil {
		return fault(CodeInternal, "finish merge check", err)
	}
	if status != AttemptSucceeded {
		return failMergeVerification(ctx, tx, attempt.JobID, attempt.CandidateRevision, attempt.Ordinal, fmt.Sprintf("required check %s was %s on the merge commit", attempt.CheckKey, request.CheckStatus), now)
	}
	if current, err := currentMergeGeneration(ctx, tx, attempt.JobID); err != nil || current != attempt.Ordinal {
		return err
	}
	var remaining int
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM attempts WHERE job_id=? AND purpose=? AND ordinal=? AND status<>?`, attempt.JobID, AttemptPurposeMerge, attempt.Ordinal, AttemptSucceeded).Scan(&remaining); err != nil {
		return fault(CodeInternal, "inspect merge checks", err)
	}
	record, err := loadMergeRecord(ctx, tx, attempt.JobID)
	if err != nil || record == nil || record.Status != MergeStatusVerifying || record.Revision != attempt.CandidateRevision {
		return err
	}
	if remaining > 0 {
		_, err := tx.ExecContext(ctx, `UPDATE jobs SET version=version+1, updated_at=? WHERE id=?`, formatTime(now), attempt.JobID)
		return err
	}
	if err := s.applyMerge(ctx, attempt.Repository, attempt.JobID, *record); err != nil {
		return failMergeVerification(ctx, tx, attempt.JobID, attempt.CandidateRevision, attempt.Ordinal, "verified, but the branch could not move: "+truncateRunes(err.Error(), maxRefinementRunes), now)
	}
	record.Status, record.MergedAt = MergeStatusMerged, &now
	job, err := loadJob(ctx, tx, attempt.JobID)
	if err != nil {
		return err
	}
	approver, err := readUser(ctx, tx, record.MergedBy)
	if err != nil {
		return err
	}
	record.TicketNote = s.noteMerge(job, *record, approver.Username)
	return saveMergeRecord(ctx, tx, attempt.JobID, nil, *record, "job_merged", now)
}

// applyMerge moves the branch to a verified merge commit. A checkout is fast-forwarded
// only from the tip the merge was built on; a ref moves only by compare-and-swap.
func (s *Service) applyMerge(ctx context.Context, repository string, jobID JobID, record MergeRecord) error {
	switch record.Mode {
	case MergeModeRef:
		if _, err := s.git(ctx, repository, "update-ref", "-m", "waymark: merge job "+string(jobID), "refs/heads/"+record.Branch, record.Revision, record.PreviousTip); err != nil {
			return fmt.Errorf("the branch moved during verification: %w", err)
		}
	case MergeModeCheckout:
		checkout, err := s.branchCheckout(ctx, repository, record.Branch)
		if err != nil || !sameDirectory(checkout, repository) {
			return errors.New("the branch is no longer checked out in the project repository")
		}
		if head, err := s.git(ctx, repository, "rev-parse", "HEAD"); err != nil || head != record.PreviousTip {
			return errors.New("the branch moved during verification")
		}
		if _, err := s.git(ctx, repository, "merge", "--ff-only", "--quiet", record.Revision); err != nil {
			return fmt.Errorf("local changes block the fast-forward: %w", err)
		}
	default:
		return fmt.Errorf("unknown merge mode %q", record.Mode)
	}
	_, _ = s.git(ctx, repository, "update-ref", "-d", mergePin(jobID))
	return nil
}

func mergePin(jobID JobID) string { return "refs/waymark/merges/" + string(jobID) }

func (s *Service) recordMergeFailure(ctx context.Context, jobID JobID, actor UserID, branch, candidate string, cause error, now time.Time) {
	tx, err := s.state.db.BeginTx(ctx, nil)
	if err != nil {
		return
	}
	defer tx.Rollback()
	if _, err := appendEvent(ctx, tx, jobID, &actor, "merge_failed", map[string]any{"branch": branch, "candidateRevision": candidate, "error": truncateRunes(cause.Error(), maxRefinementRunes)}, now); err == nil {
		_ = tx.Commit()
	}
}

// buildMerge creates the merge commit in a temporary detached worktree and pins it
// under refs/waymark/merges until verification finishes. No branch moves here.
func (s *Service) buildMerge(ctx context.Context, repository, previous, candidate, message, username string, jobID JobID) (string, error) {
	directory := filepath.Join(s.state.workspaceRoot, "merges", fmt.Sprintf("%s-%d", jobID, time.Now().UnixNano()))
	if err := os.MkdirAll(filepath.Dir(directory), 0o700); err != nil {
		return "", fault(CodeInternal, "create merge workspace", err)
	}
	if _, err := s.git(ctx, repository, "worktree", "add", "--detach", directory, previous); err != nil {
		return "", fault(CodeInternal, "create merge worktree", err)
	}
	defer func() { _, _ = s.git(context.Background(), repository, "worktree", "remove", "--force", directory) }()
	if _, err := s.git(ctx, directory, append(s.identity(ctx, directory, username), "merge", "--no-ff", "-m", message, candidate)...); err != nil {
		return "", detailFault(CodeConflict, "the candidate does not merge cleanly into the target branch", map[string]any{"git": truncateRunes(err.Error(), maxRefinementRunes)}, err)
	}
	merged, err := s.git(ctx, directory, "rev-parse", "HEAD")
	if err != nil {
		return "", fault(CodeInternal, "read merge commit", err)
	}
	if _, err := s.git(ctx, repository, "update-ref", mergePin(jobID), merged); err != nil {
		return "", fault(CodeInternal, "pin merge commit", err)
	}
	return merged, nil
}

// branchCheckout returns the worktree that has the branch checked out, or "".
func (s *Service) branchCheckout(ctx context.Context, repository, branch string) (string, error) {
	output, err := s.git(ctx, repository, "worktree", "list", "--porcelain")
	if err != nil {
		return "", fault(CodeInternal, "list repository worktrees", err)
	}
	var worktree string
	for _, line := range strings.Split(output, "\n") {
		if path, ok := strings.CutPrefix(line, "worktree "); ok {
			worktree = path
		} else if line == "branch refs/heads/"+branch {
			return worktree, nil
		}
	}
	return "", nil
}

// identity supplies a committer only when the repository has none configured, so a
// user's own identity is kept when present.
func (s *Service) identity(ctx context.Context, directory, username string) []string {
	if email, err := s.git(ctx, directory, "config", "user.email"); err == nil && email != "" {
		return nil
	}
	return []string{"-c", "user.name=" + username, "-c", "user.email=" + username + "@waymark.invalid"}
}

func (s *Service) isAncestor(ctx context.Context, repository, ancestor, descendant string) bool {
	_, err := s.git(ctx, repository, "merge-base", "--is-ancestor", ancestor, descendant)
	return err == nil
}

// mergeMessage names the acceptance record relative to the repository when it lives
// inside it, so shared history carries no machine-specific paths.
func mergeMessage(job Job, username, candidate, repository string) string {
	acceptance := filepath.Base(job.AcceptanceRecordPath)
	if relative, err := filepath.Rel(repository, job.AcceptanceRecordPath); err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		acceptance = filepath.ToSlash(relative)
	}
	return fmt.Sprintf("Merge Waymark job %s for %s\n\nCandidate: %s\nRequirements: %s\nAcceptance: %s\nApproved-by: %s\n", job.ID, job.WorkItemID, candidate, job.Admission.TicketSHA256, acceptance, username)
}

// noteMerge appends the merged revision to the ticket's Comments. The merge has already
// happened, so a ticket that cannot be updated is reported rather than undoing it.
func (s *Service) noteMerge(job Job, record MergeRecord, username string) string {
	path := job.Admission.TicketPath
	raw, err := os.ReadFile(path)
	if err != nil {
		return "skipped: " + err.Error()
	}
	comment := fmt.Sprintf("- %s: %s merged job %s into %s at %s (%s).\n", record.MergedAt.Format("2006-01-02"), username, job.ID, record.Branch, record.Revision, record.Mode)
	updated, err := appendComment(raw, path, comment)
	if err == nil {
		err = writeFileCAS(path, raw, updated)
	}
	if err != nil {
		return "skipped: " + err.Error()
	}
	return "recorded"
}

// requireMergedPrerequisites refuses admission when a prerequisite was accepted by a
// factory job that is not merged, or when the base revision does not contain that
// merge. Prerequisites accepted outside the factory are left to the reader's checks.
func (s *Service) requireMergedPrerequisites(ctx context.Context, q queryer, project Project, dependencies []string, base string) error {
	for _, dependency := range dependencies {
		var jobID JobID
		var encoded string
		err := q.QueryRowContext(ctx, `SELECT id, merge_json FROM jobs WHERE project_id=? AND work_item_id=? AND status=? ORDER BY updated_at DESC LIMIT 1`, project.ID, dependency, JobAccepted).Scan(&jobID, &encoded)
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			return fault(CodeInternal, "inspect prerequisite job", err)
		}
		var record MergeRecord
		if encoded != "" {
			if err := json.Unmarshal([]byte(encoded), &record); err != nil {
				return fault(CodeInternal, "decode prerequisite merge", err)
			}
		}
		if encoded == "" || !record.Merged() {
			return detailFault(CodeNotReady, "a prerequisite was accepted but its candidate is not merged; merge it before admitting dependent work", map[string]any{"workItemId": dependency, "jobId": jobID, "mergeStatus": record.Status}, nil)
		}
		if !s.isAncestor(ctx, project.RepositoryPath, record.Revision, base) {
			return detailFault(CodeNotReady, "the base revision does not contain a merged prerequisite; submit from a base that includes it", map[string]any{"workItemId": dependency, "jobId": jobID, "branch": record.Branch, "revision": record.Revision}, nil)
		}
	}
	return nil
}

func sameDirectory(left, right string) bool {
	resolve := func(path string) string {
		if resolved, err := filepath.EvalSymlinks(path); err == nil {
			return filepath.Clean(resolved)
		}
		return filepath.Clean(path)
	}
	return resolve(left) == resolve(right)
}
