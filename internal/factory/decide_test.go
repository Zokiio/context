package factory_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/Zokiio/context/internal/factory"
)

func approveRequest(job factory.Job, id string, merge bool, findings ...string) factory.ApproveRequest {
	return factory.ApproveRequest{RequestID: id, JobID: job.ID, IfVersion: job.Version, CandidateRevision: job.CandidateRevision, RequirementFingerprint: job.Admission.TicketSHA256, Findings: findings, Merge: merge}
}

func decisionsOf(job factory.Job, kinds ...string) map[string]string {
	result := map[string]string{}
	for _, event := range job.Events {
		for _, kind := range kinds {
			if event.Kind != kind {
				continue
			}
			var data struct {
				Decision string `json:"decision"`
			}
			_ = json.Unmarshal(event.Data, &data)
			result[kind] = data.Decision
		}
	}
	return result
}

func TestApproveReviewsAcceptsAndMergesAsOneDecision(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	h := setupJob(t)
	job, _ := passVerificationForTest(t, h, "approve")
	approved, err := h.service.Approve(ctx, h.admin, approveRequest(job, "approve", true, factory.FindingWrongRequirement))
	if err != nil {
		t.Fatal(err)
	}
	if approved.Status != factory.JobAccepted || approved.Merge == nil || approved.Merge.Status != factory.MergeStatusVerifying {
		t.Fatalf("approval = %s, merge %+v", approved.Status, approved.Merge)
	}
	decisions := decisionsOf(approved, "review_recorded", "acceptance_published", "merge_verifying")
	if len(decisions) != 3 || decisions["review_recorded"] == "" || decisions["review_recorded"] != decisions["acceptance_published"] || decisions["acceptance_published"] != decisions["merge_verifying"] {
		t.Fatalf("one approval must carry one decision across its records: %v", decisions)
	}
	if len(approved.Reviews) != 1 || len(approved.Reviews[0].Findings) != 1 || approved.Reviews[0].Findings[0] != factory.FindingWrongRequirement {
		t.Fatalf("review findings = %+v", approved.Reviews)
	}
	// Retrying the same approval continues instead of failing on the old version.
	again, err := h.service.Approve(ctx, h.admin, approveRequest(job, "approve", true, factory.FindingWrongRequirement))
	if err != nil || again.Merge == nil || again.Merge.Status != factory.MergeStatusVerifying {
		t.Fatalf("retry = %+v, %v", again.Merge, err)
	}
	runMergeChecks(t, h, "approve", factory.CheckPassed)
	metrics, err := h.service.Metrics(ctx, h.admin, h.project.ID)
	if err != nil {
		t.Fatal(err)
	}
	// Admission and one approval: two decisions, not four.
	if metrics.MergedJobs != 1 || metrics.HumanDecisions != 2 || metrics.DelegatedDecisions != 0 || metrics.Findings[factory.FindingWrongRequirement] != 1 || metrics.TrustedMerges != 1 {
		t.Fatalf("metrics = %+v", metrics)
	}
}

func TestApproveChecksEveryCapabilityBeforeRecordingAnything(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	h := setupJob(t)
	job, _ := passVerificationForTest(t, h, "refuse")
	if _, err := h.service.Approve(ctx, h.reviewer, approveRequest(job, "merge", true)); !isCode(err, factory.CodeForbidden) {
		t.Fatalf("approving with a merge but no merge capability = %v", err)
	}
	if _, err := h.service.Approve(ctx, h.admin, approveRequest(job, "finding", false, "looks fine")); !isCode(err, factory.CodeInvalid) {
		t.Fatalf("unknown finding = %v", err)
	}
	unchanged, err := h.service.GetJob(ctx, h.admin, job.ID)
	if err != nil || unchanged.Status != factory.JobReadyForReview || len(unchanged.Reviews) != 0 {
		t.Fatalf("a refused approval recorded something: %s, %d reviews, %v", unchanged.Status, len(unchanged.Reviews), err)
	}
	accepted, err := h.service.Approve(ctx, h.reviewer, approveRequest(job, "accept-only", false))
	if err != nil || accepted.Status != factory.JobAccepted || accepted.Merge != nil {
		t.Fatalf("approving without merging = %s, %+v, %v", accepted.Status, accepted.Merge, err)
	}
	// A new approval of the same candidate, from a stale page, only merges.
	merging, err := h.service.Approve(ctx, h.admin, approveRequest(job, "merge-later", true))
	if err != nil || merging.Merge == nil || merging.Merge.Status != factory.MergeStatusVerifying || len(merging.Reviews) != 1 {
		t.Fatalf("merging an accepted job = %+v, %d reviews, %v", merging.Merge, len(merging.Reviews), err)
	}
}

func TestAskPolicyDelegatesOnlyWhatThePersonMayDecide(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	h := setupJob(t)
	request := factory.SetAskPolicyRequest{RequestID: "ask", ProjectID: h.project.ID, BeforeBuild: factory.AskBeforeWhenUnclear, AfterBuild: factory.AskAfterWhenUntested, Autopilot: true}
	if _, err := h.service.SetAskPolicy(ctx, h.coordinator, request); !isCode(err, factory.CodeForbidden) {
		t.Fatalf("a coordinator without project administration = %v", err)
	}
	if _, err := h.service.GrantProject(ctx, h.admin, factory.GrantProjectRequest{RequestID: "grant-admin", ProjectID: h.project.ID, UserID: h.coordinator.UserID, Capabilities: []factory.Capability{factory.CapabilityRead, factory.CapabilityCoordinate, factory.CapabilityProjectAdmin}}); err != nil {
		t.Fatal(err)
	}
	if _, err := h.service.SetAskPolicy(ctx, h.coordinator, request); !isCode(err, factory.CodeForbidden) {
		t.Fatalf("delegating approvals without review, accept, and merge = %v", err)
	}
	never := request
	never.RequestID, never.AfterBuild = "never", factory.AskAfterNever
	if _, err := h.service.SetAskPolicy(ctx, h.admin, never); !isCode(err, factory.CodeNotReady) {
		t.Fatalf("merging without asking before trust is earned = %v", err)
	}
	bad := request
	bad.RequestID, bad.BeforeBuild = "bad", "sometimes"
	if _, err := h.service.SetAskPolicy(ctx, h.admin, bad); !isCode(err, factory.CodeInvalid) {
		t.Fatalf("unknown policy = %v", err)
	}
	project, err := h.service.SetAskPolicy(ctx, h.admin, request)
	if err != nil {
		t.Fatal(err)
	}
	if project.Policy.Ask.SetBy == nil || *project.Policy.Ask.SetBy != h.admin.UserID || project.Policy.Ask.After() != factory.AskAfterWhenUntested || !project.Policy.Ask.Autopilot {
		t.Fatalf("saved policy = %+v", project.Policy.Ask)
	}
}

func TestReportedProblemLowersTheAskPolicyAndResetsTrust(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	h := setupJob(t)
	job, _ := acceptForTest(t, h, "problem")
	if _, err := h.service.ReportProblem(ctx, h.coordinator, factory.ReportProblemRequest{RequestID: "early", JobID: job.ID, Summary: "Broken."}); !isCode(err, factory.CodeNotReady) {
		t.Fatalf("a problem on unmerged work = %v", err)
	}
	mergeAndVerify(t, h, job, "merge", "")
	if _, err := h.service.SetAskPolicy(ctx, h.admin, factory.SetAskPolicyRequest{RequestID: "ask", ProjectID: h.project.ID, BeforeBuild: factory.AskBeforeAlways, AfterBuild: factory.AskAfterWhenUntested}); err != nil {
		t.Fatal(err)
	}
	reported, err := h.service.ReportProblem(ctx, h.coordinator, factory.ReportProblemRequest{RequestID: "report", JobID: job.ID, Summary: "The change breaks empty input."})
	if err != nil {
		t.Fatal(err)
	}
	if !hasEvent(reported, "problem_reported", "empty input") || !hasEvent(reported, "ask_policy_lowered", "always") {
		t.Fatalf("problem events missing: %+v", reported.Events)
	}
	board, err := h.service.Board(ctx, h.admin, h.project.ID)
	if err != nil {
		t.Fatal(err)
	}
	if board.Project.Policy.Ask.After() != factory.AskAfterAlways || board.TrustedMerges != 0 {
		t.Fatalf("after a problem: policy %+v, trusted merges %d", board.Project.Policy.Ask, board.TrustedMerges)
	}
	metrics, err := h.service.Metrics(ctx, h.admin, h.project.ID)
	if err != nil || metrics.ProblemsReported != 1 {
		t.Fatalf("problems = %d, %v", metrics.ProblemsReported, err)
	}
}

func cardFor(t *testing.T, service *factory.Service, actor factory.Principal, project factory.ProjectID, workItemID string) factory.BoardCard {
	t.Helper()
	board, err := service.Board(context.Background(), actor, project)
	if err != nil {
		t.Fatal(err)
	}
	for _, card := range board.Cards {
		if card.WorkItemID == workItemID {
			return card
		}
	}
	t.Fatalf("no card for %s in %+v", workItemID, board.Cards)
	return factory.BoardCard{}
}

func TestBoardFollowsAJobFromBuildToDone(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	h := setupJob(t)
	card := cardFor(t, h.service, h.admin, h.project.ID, "work-test")
	if card.Column != factory.ColumnBuild || card.State != factory.CardRunning || card.Status != "Waiting for a builder" || card.Since == nil {
		t.Fatalf("queued card = %+v", card)
	}
	job, _ := passVerificationForTest(t, h, "board")
	card = cardFor(t, h.service, h.admin, h.project.ID, "work-test")
	if card.Column != factory.ColumnApprove || card.State != factory.CardNeedsYou || card.Decision != factory.CardDecisionApprove || card.JobVersion != job.Version {
		t.Fatalf("card ready for approval = %+v", card)
	}
	accepted, err := h.service.Approve(ctx, h.admin, approveRequest(job, "accept", false))
	if err != nil {
		t.Fatal(err)
	}
	card = cardFor(t, h.service, h.admin, h.project.ID, "work-test")
	if card.Column != factory.ColumnMerge || card.Decision != factory.CardDecisionMerge || card.Status != "Ready to merge" {
		t.Fatalf("accepted card = %+v", card)
	}
	mergeAndVerify(t, h, accepted, "merge", "")
	board, err := h.service.Board(ctx, h.admin, h.project.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(board.Cards) != 1 || board.Cards[0].Column != factory.ColumnDone || board.MergedToday != 1 || board.NeedsYou != 0 {
		t.Fatalf("merged board = %+v", board)
	}
}

func TestPlanDecisionStartsTestsThenBuildsFromThem(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	h := setupExpectations(t, fakeAuthor(t, map[string]string{"tests/plan_test.txt": "expects the change\n"}), nil)
	card := cardFor(t, h.service, h.admin, h.project.ID, "work-test")
	if card.Column != factory.ColumnNext || card.Decision != factory.CardDecisionPlan {
		t.Fatalf("ready card = %+v", card)
	}
	started, err := h.service.ApprovePlan(ctx, h.admin, factory.ApprovePlanRequest{RequestID: "agree", ProjectID: h.project.ID, WorkItemID: "work-test"})
	if err != nil {
		t.Fatal(err)
	}
	if started.Expectation == nil || started.Job != nil || started.Waiting == "" {
		t.Fatalf("agreeing the plan should start locked tests first: %+v", started)
	}
	expectation := h.waitExpectation(t, started.Expectation.ID)
	card = cardFor(t, h.service, h.admin, h.project.ID, "work-test")
	if card.Column != factory.ColumnPlan || card.State != factory.CardNeedsYou || card.Status != "Read the locked tests" || card.ExpectationID != expectation.ID {
		t.Fatalf("proposed tests card = %+v", card)
	}
	built, err := h.service.ApprovePlan(ctx, h.admin, factory.ApprovePlanRequest{RequestID: "tests", ProjectID: h.project.ID, WorkItemID: "work-test", ExpectationID: expectation.ID})
	if err != nil {
		t.Fatal(err)
	}
	if built.Job == nil || built.Job.Admission.ExpectationsCommit != expectation.Commit || built.Expectation.Status != factory.ExpectationAccepted {
		t.Fatalf("accepting the tests should start the build from them: %+v", built)
	}
	again, err := h.service.ApprovePlan(ctx, h.admin, factory.ApprovePlanRequest{RequestID: "tests", ProjectID: h.project.ID, WorkItemID: "work-test", ExpectationID: expectation.ID})
	if err != nil || again.Job == nil || again.Job.ID != built.Job.ID {
		t.Fatalf("a retried plan decision started another build: %+v, %v", again.Job, err)
	}
}

func TestPlanDecisionBuildsDirectlyWithoutProtectedTests(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	h := setupExpectations(t, nil, func(policy *factory.ProjectPolicy) { policy.ProtectedPaths = nil })
	result, err := h.service.ApprovePlan(ctx, h.admin, factory.ApprovePlanRequest{RequestID: "agree", ProjectID: h.project.ID, WorkItemID: "work-test"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Job == nil || result.Expectation != nil || result.Job.Admission.RepairLimit != 2 {
		t.Fatalf("a plan without locked tests should start the build: %+v", result)
	}
}

func (h expectationHarness) waitExpectation(t *testing.T, id factory.ID) factory.Expectation {
	t.Helper()
	for i := 0; i < 300; i++ {
		current, err := h.service.GetExpectations(context.Background(), h.admin, id)
		if err != nil {
			t.Fatal(err)
		}
		if current.Status != factory.ExpectationRunning {
			return current
		}
		sleep()
	}
	t.Fatal("expectations still running")
	return factory.Expectation{}
}

func TestChangeShowsTheCandidateAgainstItsBase(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	h := setupJob(t)
	if _, err := h.service.Change(ctx, h.admin, h.job.ID); !isCode(err, factory.CodeNotReady) {
		t.Fatalf("change before a candidate = %v", err)
	}
	job, _ := passVerificationForTest(t, h, "change")
	change, err := h.service.Change(ctx, h.admin, job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(change.Files) != 1 || change.Files[0].Path != "change.txt" || change.Files[0].Added != 1 || !strings.Contains(change.Patch, "+change") || change.Truncated {
		t.Fatalf("change = %+v", change)
	}
}

func TestBoardShowsRefinementsUntilTheTicketChanges(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	refiner, _ := fakeRefiner(t, "0.3")
	h := setupRefine(t, refiner)
	started, err := h.service.StartRefinement(ctx, h.admin, factory.StartRefinementRequest{RequestID: "sharpen", ProjectID: h.project.ID, WorkItemID: "work-test"})
	if err != nil {
		t.Fatal(err)
	}
	if card := cardFor(t, h.service, h.admin, h.project.ID, "work-test"); card.Column != factory.ColumnPlan || card.Status != "Sharpening requirements" || card.RefinementID != started.ID {
		t.Fatalf("running refinement card = %+v", card)
	}
	proposed := h.waitProposed(t, started.ID)
	card := cardFor(t, h.service, h.admin, h.project.ID, "work-test")
	if card.State != factory.CardNeedsYou || card.Decision != factory.CardDecisionPlan || card.Questions != len(proposed.Proposal.Questions) || !strings.HasPrefix(card.Status, "Answer") {
		t.Fatalf("proposed refinement card = %+v", card)
	}
	result, err := h.service.ApprovePlan(ctx, h.admin, factory.ApprovePlanRequest{RequestID: "agree", ProjectID: h.project.ID, WorkItemID: "work-test", RefinementID: proposed.ID})
	if err != nil {
		t.Fatal(err)
	}
	if result.Refinement == nil || result.Refinement.Status != factory.RefinementAccepted || result.Job == nil {
		t.Fatalf("agreeing the sharpened requirements = %+v", result)
	}
}
