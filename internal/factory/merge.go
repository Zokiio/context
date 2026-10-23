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

// Merge modes. Checkout merges in the project's own working tree because the target
// branch is checked out there. Ref builds the merge in a temporary worktree and moves a
// branch that no worktree has checked out. Contained records a branch that already
// contains the candidate, for example after a merge by hand.
const (
	MergeModeCheckout  = "checkout"
	MergeModeRef       = "ref"
	MergeModeContained = "contained"
)

type MergeRecord struct {
	Branch            string    `json:"branch"`
	Revision          string    `json:"revision"`
	PreviousTip       string    `json:"previousTip"`
	CandidateRevision string    `json:"candidateRevision"`
	Mode              string    `json:"mode"`
	MergedBy          UserID    `json:"mergedBy"`
	MergedAt          time.Time `json:"mergedAt"`
	TicketNote        string    `json:"ticketNote,omitempty"`
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
	if job.Merge != nil {
		return Job{}, detailFault(CodeConflict, "job is already merged", map[string]any{"branch": job.Merge.Branch, "revision": job.Merge.Revision}, nil)
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
	record := MergeRecord{Branch: branch, PreviousTip: previous, CandidateRevision: candidate, MergedBy: current.UserID, MergedAt: now}
	if s.isAncestor(ctx, repository, candidate, previous) {
		record.Mode, record.Revision = MergeModeContained, previous
	} else {
		checkout, err := s.branchCheckout(ctx, repository, branch)
		if err != nil {
			return Job{}, err
		}
		message := mergeMessage(job, current.Username, candidate)
		switch {
		case checkout == "":
			record.Mode = MergeModeRef
			record.Revision, err = s.mergeByRef(ctx, repository, branch, previous, candidate, message, current.Username, job.ID)
		case sameDirectory(checkout, repository):
			record.Mode = MergeModeCheckout
			record.Revision, err = s.mergeInCheckout(ctx, repository, candidate, message, current.Username)
		default:
			err = detailFault(CodeConflict, "target branch is checked out in another worktree; merge there or choose another branch", map[string]any{"worktree": checkout}, nil)
		}
		if err != nil {
			s.recordMergeFailure(ctx, job.ID, current.UserID, branch, candidate, err, now)
			return Job{}, err
		}
	}
	record.TicketNote = s.noteMerge(job, record, current.Username)
	tx, err := s.state.db.BeginTx(ctx, nil)
	if err != nil {
		return Job{}, fault(CodeInternal, "start merge record", err)
	}
	defer tx.Rollback()
	saved, ok, hash, err := receipt[Job](ctx, tx, current.UserID, "merge", request.RequestID, request)
	if err != nil || ok {
		return saved, err
	}
	encoded, err := marshal(record)
	if err != nil {
		return Job{}, fault(CodeInternal, "encode merge", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE jobs SET merge_json=?, version=version+1, updated_at=? WHERE id=?`, encoded, formatTime(now), job.ID); err != nil {
		return Job{}, fault(CodeInternal, "record merge", err)
	}
	if _, err := appendEvent(ctx, tx, job.ID, &current.UserID, "job_merged", record, now); err != nil {
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

// mergeInCheckout merges in the working tree that has the branch checked out. Git
// refuses when uncommitted changes would be overwritten; a conflicted merge is aborted
// so the working tree is left as it was.
func (s *Service) mergeInCheckout(ctx context.Context, repository, candidate, message, username string) (string, error) {
	if _, err := s.git(ctx, repository, "rev-parse", "--quiet", "--verify", "MERGE_HEAD"); err == nil {
		return "", fault(CodeConflict, "a merge is already in progress in the project repository", nil)
	}
	if _, err := s.git(ctx, repository, append(s.identity(ctx, repository, username), "merge", "--no-ff", "-m", message, candidate)...); err != nil {
		if _, pending := s.git(ctx, repository, "rev-parse", "--quiet", "--verify", "MERGE_HEAD"); pending == nil {
			_, _ = s.git(ctx, repository, "merge", "--abort")
		}
		return "", detailFault(CodeConflict, "the candidate does not merge cleanly; the merge was aborted", map[string]any{"git": truncateRunes(err.Error(), maxRefinementRunes)}, err)
	}
	return s.git(ctx, repository, "rev-parse", "HEAD")
}

// mergeByRef merges in a temporary detached worktree, then moves the branch only if it
// still points at the tip the merge was built on.
func (s *Service) mergeByRef(ctx context.Context, repository, branch, previous, candidate, message, username string, jobID JobID) (string, error) {
	directory := filepath.Join(s.state.workspaceRoot, "merges", fmt.Sprintf("%s-%d", jobID, time.Now().UnixNano()))
	if err := os.MkdirAll(filepath.Dir(directory), 0o700); err != nil {
		return "", fault(CodeInternal, "create merge workspace", err)
	}
	if _, err := s.git(ctx, repository, "worktree", "add", "--detach", directory, previous); err != nil {
		return "", fault(CodeInternal, "create merge worktree", err)
	}
	defer func() { _, _ = s.git(context.Background(), repository, "worktree", "remove", "--force", directory) }()
	if _, err := s.git(ctx, directory, append(s.identity(ctx, directory, username), "merge", "--no-ff", "-m", message, candidate)...); err != nil {
		return "", detailFault(CodeConflict, "the candidate does not merge cleanly into "+branch, map[string]any{"git": truncateRunes(err.Error(), maxRefinementRunes)}, err)
	}
	merged, err := s.git(ctx, directory, "rev-parse", "HEAD")
	if err != nil {
		return "", fault(CodeInternal, "read merge commit", err)
	}
	if _, err := s.git(ctx, repository, "update-ref", "-m", "waymark: merge job "+string(jobID), "refs/heads/"+branch, merged, previous); err != nil {
		return "", fault(CodeStale, "the target branch moved during the merge; retry", err)
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

func mergeMessage(job Job, username, candidate string) string {
	return fmt.Sprintf("Merge Waymark job %s for %s\n\nCandidate: %s\nRequirements: %s\nAcceptance: %s\nApproved-by: %s\n", job.ID, job.WorkItemID, candidate, job.Admission.TicketSHA256, job.AcceptanceRecordPath, username)
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
		if encoded == "" {
			return detailFault(CodeNotReady, "a prerequisite was accepted but its candidate is not merged; merge it before admitting dependent work", map[string]any{"workItemId": dependency, "jobId": jobID}, nil)
		}
		var record MergeRecord
		if err := json.Unmarshal([]byte(encoded), &record); err != nil {
			return fault(CodeInternal, "decode prerequisite merge", err)
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
