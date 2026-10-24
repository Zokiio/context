package factory

import (
	"context"
	"database/sql"
	"path"
	"sort"
	"strings"
	"time"
)

// normalizeProtectedPaths validates repository-relative patterns. A pattern is a
// slash-separated glob in which "**" matches any number of directories, and a
// trailing "/" protects everything beneath a directory.
func normalizeProtectedPaths(patterns []string) ([]string, error) {
	normalized := make([]string, 0, len(patterns))
	for _, pattern := range patterns {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}
		if strings.HasSuffix(pattern, "/") {
			pattern += "**"
		}
		if strings.HasPrefix(pattern, "/") || strings.Contains(pattern, "\\") {
			return nil, fault(CodeInvalid, "protected paths must be repository-relative slash patterns", nil)
		}
		for _, segment := range strings.Split(pattern, "/") {
			if segment == "" || segment == "." || segment == ".." {
				return nil, fault(CodeInvalid, "protected path patterns cannot contain empty, '.', or '..' segments", nil)
			}
			if _, err := path.Match(segment, ""); err != nil {
				return nil, fault(CodeInvalid, "protected path pattern is malformed", err)
			}
		}
		normalized = append(normalized, pattern)
	}
	return uniqueSorted(normalized), nil
}

func protectedMatch(pattern, file string) bool {
	return matchSegments(strings.Split(pattern, "/"), strings.Split(file, "/"))
}

func matchSegments(pattern, file []string) bool {
	for len(pattern) > 0 {
		if pattern[0] == "**" {
			for skip := 0; skip <= len(file); skip++ {
				if matchSegments(pattern[1:], file[skip:]) {
					return true
				}
			}
			return false
		}
		if len(file) == 0 {
			return false
		}
		if ok, _ := path.Match(pattern[0], file[0]); !ok {
			return false
		}
		pattern, file = pattern[1:], file[1:]
	}
	return len(file) == 0
}

// protectedChanges lists the files between base and candidate that match a protected
// pattern. Renames are reported as a deletion and an addition so that moving a
// protected file out of a protected directory still counts as changing it.
func (s *Service) protectedChanges(ctx context.Context, workspace, base, candidate string, patterns []string) ([]string, error) {
	if len(patterns) == 0 {
		return nil, nil
	}
	output, err := s.git(ctx, workspace, "diff", "--name-only", "--no-renames", "-z", base, candidate)
	if err != nil {
		return nil, fault(CodeInternal, "compare candidate with protected paths", err)
	}
	var changed []string
	for _, file := range strings.Split(output, "\x00") {
		if file == "" {
			continue
		}
		for _, pattern := range patterns {
			if protectedMatch(pattern, file) {
				changed = append(changed, file)
				break
			}
		}
	}
	sort.Strings(changed)
	return changed, nil
}

// rejectProtectedCandidate fails an implementation whose candidate changed protected
// paths. The attempt keeps the rejected candidate for inspection, but the job does not,
// so a repair starts again from the admitted base.
func (s *Service) rejectProtectedCandidate(ctx context.Context, tx *sql.Tx, attempt attemptRecord, request WorkerReportRequest, candidate string, paths []string, now time.Time) error {
	jobStatus, active := JobFailed, 1
	if attempt.CancellationRequested {
		jobStatus, active = JobCancelled, 0
	}
	if _, err := tx.ExecContext(ctx, `UPDATE attempts SET status=?, outcome=?, candidate_revision=?, terminal_receipt=?, finished_at=? WHERE id=?`, AttemptFailed, request.Outcome, candidate, request.TerminalReceipt, formatTime(request.ObservedAt), attempt.ID); err != nil {
		return fault(CodeInternal, "reject protected candidate", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE jobs SET status=?, candidate_revision='', active_claim=?, version=version+1, updated_at=? WHERE id=?`, jobStatus, active, formatTime(now), attempt.JobID); err != nil {
		return fault(CodeInternal, "fail job with protected changes", err)
	}
	_, err := appendEvent(ctx, tx, attempt.JobID, nil, "protected_paths_changed", map[string]any{"attemptId": attempt.ID, "candidateRevision": candidate, "paths": paths}, now)
	return err
}

// MatchProtectedPath reports whether a repository-relative file matches a normalized
// protected path pattern. Adapters use it to warn before the controller rejects.
func MatchProtectedPath(pattern, file string) bool {
	return protectedMatch(pattern, file)
}
