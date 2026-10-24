package factory

import (
	"math"
	"strings"
	"testing"
)

func TestJudgeStepRetainsProposalReviewEvidenceAcrossHandoffs(t *testing.T) {
	dispute := &Dispute{Status: "reviewing_tests", NextOwner: "judge", Decision: &JudgeDecision{Kind: DisputeReviseTests, Rationale: "Original frozen test contradicts the requirement"}, Run: &RefinerRun{Model: "original-judge", Turns: 2, CostUSD: .01}}
	step := judgeStep(dispute)
	if step.Status != RunRunning || step.Title != "Review the test proposal" || !strings.Contains(step.Summary, "reviewing the test proposal") {
		t.Fatalf("proposal review is not shown as active: %+v", step)
	}
	dispute.TestReviews = []DisputeTestReview{{ExpectationID: "proposal-1", Commit: "reviewed-revision-1", Round: 1, Decision: &JudgeDecision{Kind: DisputeReviseTests, Rationale: "Keep the existing case and add the missing boundary check", Directive: "Test the empty input"}, Run: &RefinerRun{Model: "proposal-judge", Turns: 3, CostUSD: .02}, NextOwner: "test_author"}}
	dispute.Status, dispute.NextOwner = "authoring_tests", "test_author"
	step = judgeStep(dispute)
	if step.Status != RunRunning || !strings.Contains(step.Summary, "improvement") || !strings.Contains(strings.Join(step.Notes, "\n"), "reviewed-revision-1") {
		t.Fatalf("returned tests lost the reviewed revision or handoff: %+v", step)
	}
	dispute.TestReviews = append(dispute.TestReviews, DisputeTestReview{ExpectationID: "proposal-2", Commit: "reviewed-revision-2", Round: 2, Decision: &JudgeDecision{Kind: DisputeAcceptTests, Rationale: "The corrected tests preserve the intended behavior"}, Run: &RefinerRun{Model: "final-judge", Turns: 1, CostUSD: .03}, NextOwner: "implementer"})
	dispute.Status, dispute.NextOwner = "resolved", "implementer"
	step = judgeStep(dispute)
	if step.Status != RunDone || !strings.Contains(step.Summary, "accepted") || step.Model != "final-judge" || step.Turns != 6 || math.Abs(step.CostUSD-.06) > 1e-9 {
		t.Fatalf("accepted proposal has wrong state or recorded judge usage: %+v", step)
	}
	notes := strings.Join(step.Notes, "\n")
	for _, evidence := range []string{"Original frozen test", "reviewed-revision-1", "Test the empty input", "reviewed-revision-2", "corrected tests preserve"} {
		if !strings.Contains(notes, evidence) {
			t.Fatalf("judge history lost %q: %s", evidence, notes)
		}
	}
	// A later hold overrides a successful old review, without removing its evidence.
	dispute.Status, dispute.NextOwner, dispute.Error = "needs_human", "you", "Automatic review limit reached"
	step = judgeStep(dispute)
	if step.Status != RunNeedsYou || !strings.Contains(strings.Join(step.Notes, "\n"), dispute.Error) {
		t.Fatalf("escalation is hidden by an older verdict: %+v", step)
	}
	dispute.Status, dispute.NextOwner = "stopped", "none"
	if stopped := judgeStep(dispute); stopped.Status != RunSkipped || stopped.Turns != 6 {
		t.Fatalf("stop erased history or still advertises active work: %+v", stopped)
	}
}
