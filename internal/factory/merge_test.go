package factory_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Zokiio/context/internal/factory"
)

func acceptForTest(t *testing.T, harness jobHarness, prefix string) (factory.Job, string) {
	t.Helper()
	ctx := context.Background()
	job, candidate := passVerificationForTest(t, harness, prefix)
	job, err := harness.service.Review(ctx, harness.reviewer, factory.ReviewRequest{RequestID: prefix + "-review", JobID: job.ID, IfVersion: job.Version, CandidateRevision: candidate, RequirementFingerprint: job.Admission.TicketSHA256, Decision: factory.ReviewApproved, Summary: "Reviewed."})
	if err != nil {
		t.Fatal(err)
	}
	job, err = harness.service.Accept(ctx, harness.reviewer, factory.AcceptRequest{RequestID: prefix + "-accept", JobID: job.ID, IfVersion: job.Version, CandidateRevision: candidate, RequirementFingerprint: job.Admission.TicketSHA256, Summary: "Accepted."})
	if err != nil {
		t.Fatal(err)
	}
	if job.Status != factory.JobAccepted {
		t.Fatalf("job after acceptance = %s", job.Status)
	}
	return job, candidate
}

// runMergeChecks lets the verifier run every queued merge check with the given status.
func runMergeChecks(t *testing.T, harness jobHarness, prefix string, status factory.CheckStatus) factory.Job {
	t.Helper()
	return runChecks(t, harness, prefix, status, true)
}

// runChecks lets the verifier claim and report every pending verification attempt.
func runChecks(t *testing.T, harness jobHarness, prefix string, status factory.CheckStatus, merge bool) factory.Job {
	t.Helper()
	ctx := context.Background()
	var job factory.Job
	for i := 0; ; i++ {
		assignment, err := harness.service.Claim(ctx, harness.verifier, factory.WorkerClaimRequest{RequestID: fmt.Sprintf("%s-merge-claim-%d", prefix, i), Stages: []factory.Stage{factory.StageVerification}, ProjectID: &harness.project.ID})
		if isCode(err, factory.CodeNoWork) {
			return job
		}
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(assignment.Instructions, "merge commit") != merge {
			t.Fatalf("claimed the wrong kind of check: %+v", assignment)
		}
		native := fmt.Sprintf("%s-merge-native-%d", prefix, i)
		for _, kind := range []factory.WorkerReportKind{factory.ReportStartRegistered, factory.ReportStarted} {
			if _, err := harness.service.Report(ctx, harness.verifier, factory.WorkerReportRequest{RequestID: fmt.Sprintf("%s-%s-%d", prefix, kind, i), AttemptID: assignment.AttemptID, DispatchID: assignment.DispatchID, Kind: kind, ObservedAt: time.Now(), NativeRunID: native}); err != nil {
				t.Fatal(err)
			}
		}
		outcome := factory.OutcomeSucceeded
		if status != factory.CheckPassed {
			outcome = factory.OutcomeFailed
		}
		job, err = harness.service.Report(ctx, harness.verifier, factory.WorkerReportRequest{RequestID: fmt.Sprintf("%s-terminal-%d", prefix, i), AttemptID: assignment.AttemptID, DispatchID: assignment.DispatchID, Kind: factory.ReportTerminal, ObservedAt: time.Now(), NativeRunID: native, Outcome: outcome, CheckStatus: status, TerminalReceipt: "merge check receipt"})
		if err != nil {
			t.Fatal(err)
		}
	}
}

func mergeAndVerify(t *testing.T, harness jobHarness, job factory.Job, id, branch string) factory.Job {
	t.Helper()
	started, err := harness.service.Merge(context.Background(), harness.admin, mergeRequest(job, id, branch))
	if err != nil {
		t.Fatal(err)
	}
	if started.Merge == nil || started.Merge.Status != factory.MergeStatusVerifying || started.Status != factory.JobAccepted {
		t.Fatalf("merge did not start verifying: %+v", started.Merge)
	}
	runMergeChecks(t, harness, id, factory.CheckPassed)
	merged, err := harness.service.GetJob(context.Background(), harness.admin, job.ID)
	if err != nil {
		t.Fatal(err)
	}
	return merged
}

func mergeRequest(job factory.Job, id, branch string) factory.MergeRequest {
	return factory.MergeRequest{RequestID: id, JobID: job.ID, IfVersion: job.Version, CandidateRevision: job.CandidateRevision, TargetBranch: branch}
}

func head(t *testing.T, repository, ref string) string {
	t.Helper()
	return strings.TrimSpace(runGit(t, repository, "rev-parse", ref))
}

func TestMergeIntegratesTheAcceptedCandidateInTheCheckout(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	harness := setupJob(t)
	job, candidate := acceptForTest(t, harness, "merge")
	previous := head(t, harness.repository, "HEAD")
	started, err := harness.service.Merge(ctx, harness.admin, mergeRequest(job, "merge", ""))
	if err != nil {
		t.Fatal(err)
	}
	if head(t, harness.repository, "HEAD") != previous {
		t.Fatal("the branch moved before the merge commit was verified")
	}
	if _, err := harness.service.Merge(ctx, harness.admin, mergeRequest(started, "merge-during-verification", "")); !isCode(err, factory.CodeConflict) {
		t.Fatalf("merge during verification = %v", err)
	}
	runMergeChecks(t, harness, "merge", factory.CheckPassed)
	merged, err := harness.service.GetJob(ctx, harness.admin, job.ID)
	if err != nil {
		t.Fatal(err)
	}
	record := merged.Merge
	if record == nil || record.Status != factory.MergeStatusMerged || record.Mode != factory.MergeModeCheckout || record.PreviousTip != previous || record.CandidateRevision != candidate || record.Revision != head(t, harness.repository, "HEAD") {
		t.Fatalf("merge record = %+v", record)
	}
	parents := strings.Fields(runGit(t, harness.repository, "log", "-1", "--format=%P"))
	if len(parents) != 2 || parents[0] != previous || parents[1] != candidate {
		t.Fatalf("merge parents = %v", parents)
	}
	if message := runGit(t, harness.repository, "log", "-1", "--format=%B"); !strings.Contains(message, "Merge Waymark job "+string(job.ID)) || !strings.Contains(message, "Approved-by: admin") || strings.Contains(message, string(filepath.Separator)+"private") || strings.Contains(message, os.TempDir()) {
		t.Fatalf("merge message = %q", message)
	}
	if !hasEvent(merged, "merge_verifying", record.Revision) || !hasEvent(merged, "job_merged", record.Revision) || record.TicketNote != "recorded" {
		t.Fatalf("merge event or ticket note missing: note=%q events=%+v", record.TicketNote, merged.Events)
	}
	ticket, _ := os.ReadFile(merged.Admission.TicketPath)
	if !strings.Contains(string(ticket), "merged job "+string(job.ID)) {
		t.Fatalf("ticket lacks merge note:\n%s", ticket)
	}
	refreshed, err := harness.service.GetJob(ctx, harness.admin, job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if refreshed.AcceptanceStatus != "valid" || refreshed.Merge == nil {
		t.Fatalf("after merge acceptance=%s merge=%v", refreshed.AcceptanceStatus, refreshed.Merge)
	}
	if _, err := harness.service.Merge(ctx, harness.admin, mergeRequest(refreshed, "merge-again", "")); !isCode(err, factory.CodeConflict) {
		t.Fatalf("second merge = %v", err)
	}
	replayed, err := harness.service.Merge(ctx, harness.admin, mergeRequest(job, "merge", ""))
	if err != nil || replayed.Merge == nil || replayed.Merge.Revision != record.Revision {
		t.Fatalf("replayed merge = %+v, %v", replayed.Merge, err)
	}
	if strings.Contains(runGit(t, harness.repository, "for-each-ref", "refs/waymark"), string(job.ID)) {
		t.Fatal("the merge pin was not removed")
	}
}

func TestMergeRequiresAnAcceptedCandidateAndAHuman(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	harness := setupJob(t)
	reviewable, _ := passVerificationForTest(t, harness, "early")
	if _, err := harness.service.Merge(ctx, harness.admin, mergeRequest(reviewable, "early", "")); !isCode(err, factory.CodeNotReady) {
		t.Fatalf("merge before acceptance = %v", err)
	}
	job, err := harness.service.GetJob(ctx, harness.admin, reviewable.ID)
	if err != nil {
		t.Fatal(err)
	}
	job, err = harness.service.Review(ctx, harness.reviewer, factory.ReviewRequest{RequestID: "review", JobID: job.ID, IfVersion: job.Version, CandidateRevision: job.CandidateRevision, RequirementFingerprint: job.Admission.TicketSHA256, Decision: factory.ReviewApproved, Summary: "ok"})
	if err != nil {
		t.Fatal(err)
	}
	job, err = harness.service.Accept(ctx, harness.reviewer, factory.AcceptRequest{RequestID: "accept", JobID: job.ID, IfVersion: job.Version, CandidateRevision: job.CandidateRevision, RequirementFingerprint: job.Admission.TicketSHA256, Summary: "ok"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := harness.service.GrantProject(ctx, harness.admin, factory.GrantProjectRequest{RequestID: "worker-merge", ProjectID: harness.project.ID, UserID: harness.verifier.UserID, Capabilities: []factory.Capability{factory.CapabilityRead, factory.CapabilityMerge}}); err != nil {
		t.Fatal(err)
	}
	if _, err := harness.service.Merge(ctx, harness.verifier, mergeRequest(job, "worker", "")); !isCode(err, factory.CodeForbidden) {
		t.Fatalf("worker merge = %v", err)
	}
	if _, err := harness.service.Merge(ctx, harness.reviewer, mergeRequest(job, "no-capability", "")); !isCode(err, factory.CodeForbidden) {
		t.Fatalf("merge without capability = %v", err)
	}
	stale := mergeRequest(job, "stale-version", "")
	stale.IfVersion--
	if _, err := harness.service.Merge(ctx, harness.admin, stale); !isCode(err, factory.CodeConflict) {
		t.Fatalf("stale version merge = %v", err)
	}
	wrong := mergeRequest(job, "wrong-candidate", "")
	wrong.CandidateRevision = job.Admission.BaseRevision
	if _, err := harness.service.Merge(ctx, harness.admin, wrong); !isCode(err, factory.CodeStale) {
		t.Fatalf("wrong candidate merge = %v", err)
	}
}

func TestConflictingMergeIsAbortedAndRecorded(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	harness := setupJob(t)
	job, _ := acceptForTest(t, harness, "clash")
	if err := os.WriteFile(filepath.Join(harness.repository, "clash.txt"), []byte("mainline content\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runGit(t, harness.repository, "add", "clash.txt")
	runGit(t, harness.repository, "commit", "-qm", "conflicting mainline change")
	before := head(t, harness.repository, "HEAD")
	_, err := harness.service.Merge(ctx, harness.admin, mergeRequest(job, "clash", ""))
	if !isCode(err, factory.CodeConflict) {
		t.Fatalf("conflicting merge = %v", err)
	}
	if head(t, harness.repository, "HEAD") != before {
		t.Fatal("conflicting merge moved the branch")
	}
	if _, statErr := os.Stat(filepath.Join(harness.repository, ".git", "MERGE_HEAD")); !os.IsNotExist(statErr) {
		t.Fatal("conflicting merge was left in progress")
	}
	if status := strings.TrimSpace(runGit(t, harness.repository, "status", "--porcelain", "--untracked-files=no")); status != "" {
		t.Fatalf("conflict left tracked changes: %q", status)
	}
	after, err := harness.service.GetJob(ctx, harness.admin, job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if after.Merge != nil || !hasEvent(after, "merge_failed", "does not merge cleanly") {
		t.Fatalf("conflict was not recorded: merge=%v events=%+v", after.Merge, after.Events)
	}
}

func TestMergeIntoAnUncheckedOutBranchLeavesTheCheckoutAlone(t *testing.T) {
	t.Parallel()
	harness := setupJob(t)
	runGit(t, harness.repository, "branch", "release", harness.job.Admission.BaseRevision)
	job, candidate := acceptForTest(t, harness, "release")
	checkout := head(t, harness.repository, "HEAD")
	merged := mergeAndVerify(t, harness, job, "release", "release")
	if merged.Merge.Mode != factory.MergeModeRef || head(t, harness.repository, "refs/heads/release") != merged.Merge.Revision {
		t.Fatalf("ref merge = %+v", merged.Merge)
	}
	if head(t, harness.repository, "HEAD") != checkout {
		t.Fatal("ref merge moved the checked-out branch")
	}
	if !strings.Contains(runGit(t, harness.repository, "branch", "--contains", candidate), "release") {
		t.Fatal("release does not contain the candidate")
	}
	if strings.Contains(runGit(t, harness.repository, "worktree", "list"), "merges") {
		t.Fatal("temporary merge worktree was not removed")
	}
}

func TestMergeRecordsABranchThatAlreadyContainsTheCandidate(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	harness := setupJob(t)
	job, candidate := acceptForTest(t, harness, "byhand")
	runGit(t, harness.repository, "merge", "-q", "--no-ff", "-m", "merged by hand", candidate)
	tip := head(t, harness.repository, "HEAD")
	merged, err := harness.service.Merge(ctx, harness.admin, mergeRequest(job, "byhand", ""))
	if err != nil {
		t.Fatal(err)
	}
	if merged.Merge.Mode != factory.MergeModeContained || merged.Merge.Revision != tip || head(t, harness.repository, "HEAD") != tip {
		t.Fatalf("contained merge = %+v", merged.Merge)
	}
}

func TestDependentAdmissionWaitsForTheMergedPrerequisite(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	harness := setupJob(t)
	prerequisite, _ := acceptForTest(t, harness, "first")
	manifest := "---\ntype: Project\nid: project-test\ntitle: Test project\n---\n## Goals\nTest.\n## Current commitments\n- [Work](work.md)\n- [Next](next.md)\n## Open decisions\nNone\n"
	next := "---\ntype: WorkItem\nid: work-next\ntitle: Dependent work\ntriage: ready-for-agent\nexecution: unstarted\n---\n## Acceptance criteria\n- The next change works.\n\n## Spec\nNone\n\n## Context\nNone\n\n## Blocked by\n- [Ready work](work.md)\n\n## Blocked by decisions\nNone\n"
	for name, content := range map[string]string{"project.md": manifest, "next.md": next} {
		if err := os.WriteFile(filepath.Join(harness.bundle, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	// The board draws the work graph from each card's dependencies.
	if card := cardFor(t, harness.service, harness.admin, harness.project.ID, "work-next"); !slices.Equal(card.BlockedBy, []string{"work-test"}) || card.Status != "Waits for Ready work" {
		t.Fatalf("dependent card = %+v", card)
	}
	base := head(t, harness.repository, "HEAD")
	submit := func(id, revision string) error {
		_, err := harness.service.Submit(ctx, harness.coordinator, factory.SubmitRequest{RequestID: id, ProjectID: harness.project.ID, WorkItemID: "work-next", BaseRevision: revision, ExecutorKey: "fixture-agent", Deadline: time.Now().Add(time.Hour)})
		return err
	}
	if err := submit("unmerged", base); !isCode(err, factory.CodeNotReady) || !strings.Contains(err.Error(), "not merged") {
		t.Fatalf("dependent admission before merge = %v", err)
	}
	if _, err := harness.service.Merge(ctx, harness.admin, mergeRequest(prerequisite, "merge-first", "")); err != nil {
		t.Fatal(err)
	}
	if err := submit("verifying", base); !isCode(err, factory.CodeNotReady) || !strings.Contains(err.Error(), "not merged") {
		t.Fatalf("dependent admission while the merge verifies = %v", err)
	}
	runMergeChecks(t, harness, "merge-first", factory.CheckPassed)
	merged, err := harness.service.GetJob(ctx, harness.admin, prerequisite.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := submit("old-base", base); !isCode(err, factory.CodeNotReady) || !strings.Contains(err.Error(), "does not contain") {
		t.Fatalf("dependent admission from a base without the merge = %v", err)
	}
	if err := submit("merged-base", merged.Merge.Revision); err != nil {
		t.Fatalf("dependent admission from the merged base = %v", err)
	}
}

func TestFailedMergeCheckLeavesTheBranchAndAllowsRetry(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	harness := setupJobWithChecks(t, []string{"go-test", "lint"})
	_, candidate := completeImplementationForTest(t, harness, "failing")
	runChecks(t, harness, "failing-candidate", factory.CheckPassed, false)
	job, err := harness.service.GetJob(ctx, harness.admin, harness.job.ID)
	if err != nil {
		t.Fatal(err)
	}
	job, err = harness.service.Review(ctx, harness.reviewer, factory.ReviewRequest{RequestID: "failing-review", JobID: job.ID, IfVersion: job.Version, CandidateRevision: candidate, RequirementFingerprint: job.Admission.TicketSHA256, Decision: factory.ReviewApproved, Summary: "ok"})
	if err != nil {
		t.Fatal(err)
	}
	job, err = harness.service.Accept(ctx, harness.reviewer, factory.AcceptRequest{RequestID: "failing-accept", JobID: job.ID, IfVersion: job.Version, CandidateRevision: candidate, RequirementFingerprint: job.Admission.TicketSHA256, Summary: "ok"})
	if err != nil {
		t.Fatal(err)
	}
	before := head(t, harness.repository, "HEAD")
	if _, err := harness.service.Merge(ctx, harness.admin, mergeRequest(job, "failing", "")); err != nil {
		t.Fatal(err)
	}
	failed := runMergeChecks(t, harness, "failing", factory.CheckFailed)
	if failed.Merge == nil || failed.Merge.Status != factory.MergeStatusFailed || !strings.Contains(failed.Merge.Failure, "on the merge commit") {
		t.Fatalf("failed merge = %+v", failed.Merge)
	}
	if head(t, harness.repository, "HEAD") != before || failed.Status != factory.JobAccepted {
		t.Fatalf("a failed merge check moved the branch or the job: status=%s", failed.Status)
	}
	cancelled := 0
	for _, attempt := range failed.Attempts {
		if attempt.Purpose == factory.AttemptPurposeMerge && attempt.Status == factory.AttemptCancelled {
			cancelled++
		}
	}
	if cancelled != 1 {
		t.Fatalf("the second merge check was not stopped: %d cancelled", cancelled)
	}
	merged := mergeAndVerify(t, harness, failed, "retry", "")
	if merged.Merge.Status != factory.MergeStatusMerged || head(t, harness.repository, "HEAD") != merged.Merge.Revision {
		t.Fatalf("retried merge = %+v", merged.Merge)
	}
}

func TestMergeFailsWhenTheCheckoutMovesDuringVerification(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	harness := setupJob(t)
	job, _ := acceptForTest(t, harness, "moving")
	if _, err := harness.service.Merge(ctx, harness.admin, mergeRequest(job, "moving", "")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(harness.repository, "elsewhere.txt"), []byte("moved\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runGit(t, harness.repository, "add", "elsewhere.txt")
	runGit(t, harness.repository, "commit", "-qm", "someone else committed")
	moved := head(t, harness.repository, "HEAD")
	failed := runMergeChecks(t, harness, "moving", factory.CheckPassed)
	if failed.Merge.Status != factory.MergeStatusFailed || !strings.Contains(failed.Merge.Failure, "moved") || head(t, harness.repository, "HEAD") != moved {
		t.Fatalf("merge after the checkout moved = %+v", failed.Merge)
	}
}

func TestMergeChecksRunAfterTheAdmissionDeadline(t *testing.T) {
	t.Parallel()
	clock := time.Now()
	harness := setupJobConfig(t, []string{"go-test"}, 2, func() time.Time { return clock })
	job, _ := acceptForTest(t, harness, "late")
	clock = clock.Add(2 * time.Hour)
	merged := mergeAndVerify(t, harness, job, "late", "")
	if merged.Merge.Status != factory.MergeStatusMerged {
		t.Fatalf("merge after the admission deadline = %+v", merged.Merge)
	}
}

func TestRestartDuringAMergeCheckFailsTheMerge(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	harness := setupJob(t)
	job, _ := acceptForTest(t, harness, "restart")
	if _, err := harness.service.Merge(ctx, harness.admin, mergeRequest(job, "restart", "")); err != nil {
		t.Fatal(err)
	}
	assignment, err := harness.service.Claim(ctx, harness.verifier, factory.WorkerClaimRequest{RequestID: "restart-claim", Stages: []factory.Stage{factory.StageVerification}, ProjectID: &harness.project.ID})
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range []factory.WorkerReportKind{factory.ReportStartRegistered, factory.ReportStarted} {
		if _, err := harness.service.Report(ctx, harness.verifier, factory.WorkerReportRequest{RequestID: "restart-" + string(kind), AttemptID: assignment.AttemptID, DispatchID: assignment.DispatchID, Kind: kind, ObservedAt: time.Now(), NativeRunID: "restart-native"}); err != nil {
			t.Fatal(err)
		}
	}
	if err := harness.service.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := factory.Open(harness.dataDir, factory.Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	after, err := reopened.GetJob(ctx, harness.admin, job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if after.Merge.Status != factory.MergeStatusFailed || after.Status != factory.JobAccepted || !strings.Contains(after.Merge.Failure, "restarted") {
		t.Fatalf("after restart: status=%s merge=%+v", after.Status, after.Merge)
	}
}
