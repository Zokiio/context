package factory_test

import (
	"context"
	"os"
	"path/filepath"
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
	merged, err := harness.service.Merge(ctx, harness.admin, mergeRequest(job, "merge", ""))
	if err != nil {
		t.Fatal(err)
	}
	record := merged.Merge
	if record == nil || record.Mode != factory.MergeModeCheckout || record.PreviousTip != previous || record.CandidateRevision != candidate || record.Revision != head(t, harness.repository, "HEAD") {
		t.Fatalf("merge record = %+v", record)
	}
	parents := strings.Fields(runGit(t, harness.repository, "log", "-1", "--format=%P"))
	if len(parents) != 2 || parents[0] != previous || parents[1] != candidate {
		t.Fatalf("merge parents = %v", parents)
	}
	if message := runGit(t, harness.repository, "log", "-1", "--format=%B"); !strings.Contains(message, "Merge Waymark job "+string(job.ID)) || !strings.Contains(message, "Approved-by: admin") {
		t.Fatalf("merge message = %q", message)
	}
	if !hasEvent(merged, "job_merged", record.Revision) || record.TicketNote != "recorded" {
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
	ctx := context.Background()
	harness := setupJob(t)
	runGit(t, harness.repository, "branch", "release", harness.job.Admission.BaseRevision)
	job, candidate := acceptForTest(t, harness, "release")
	checkout := head(t, harness.repository, "HEAD")
	merged, err := harness.service.Merge(ctx, harness.admin, mergeRequest(job, "release", "release"))
	if err != nil {
		t.Fatal(err)
	}
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
	base := head(t, harness.repository, "HEAD")
	submit := func(id, revision string) error {
		_, err := harness.service.Submit(ctx, harness.coordinator, factory.SubmitRequest{RequestID: id, ProjectID: harness.project.ID, WorkItemID: "work-next", BaseRevision: revision, ExecutorKey: "fixture-agent", Deadline: time.Now().Add(time.Hour)})
		return err
	}
	if err := submit("unmerged", base); !isCode(err, factory.CodeNotReady) || !strings.Contains(err.Error(), "not merged") {
		t.Fatalf("dependent admission before merge = %v", err)
	}
	merged, err := harness.service.Merge(ctx, harness.admin, mergeRequest(prerequisite, "merge-first", ""))
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
