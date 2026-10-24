package factory_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Zokiio/context/internal/factory"
)

// Each invocation retains its exact input; optional gates let the service receive
// a stop, revocation, or stale snapshot while the external judge is running.
func proposalJudge(t *testing.T, decisions []factory.JudgeDecision, gateCall int) ([]string, string, string) {
	t.Helper()
	dir := t.TempDir()
	gate := filepath.Join(dir, "continue")
	script := "#!/bin/sh\nset -eu\nn=$(cat '" + dir + "/count' 2>/dev/null || echo 0)\nn=$((n+1))\necho \"$n\" > '" + dir + "/count'\ncat > '" + dir + "/input.'\"$n\"'.json'\n"
	if gateCall > 0 {
		script += fmt.Sprintf("if [ \"$n\" = %d ]; then while [ ! -f '%s' ]; do sleep 0.02; done; fi\n", gateCall, gate)
	}
	script += "case \"$n\" in\n"
	for i, decision := range decisions {
		output, err := json.Marshal(factory.JudgeOutput{Decision: decision, Run: factory.RefinerRun{Model: "judge-fixture", SessionID: fmt.Sprint(i + 1), CostUSD: 0.01, Turns: 1}})
		if err != nil {
			t.Fatal(err)
		}
		script += fmt.Sprintf("%d) cat <<'RESULT'\n%s\nRESULT\n;;\n", i+1, output)
	}
	script += "*) exit 22;;\nesac\n"
	path := filepath.Join(dir, "judge")
	if err := os.WriteFile(path, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	return []string{path}, dir, gate
}

func proposalDecision(kind factory.DisputeDecisionKind) factory.JudgeDecision {
	return factory.JudgeDecision{Kind: kind, Rationale: "The concrete proposal must cover criterion one without changing its meaning", EvidenceRefs: []string{"criterion:1", "tests/expected.txt"}, PreservesIntent: true, Directive: "Preserve the existing assertion and add the missing boundary case"}
}

func improvingProposalAuthor(t *testing.T) ([]string, string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "author")
	script := "#!/bin/sh\nset -eu\nn=$(cat '" + dir + "/count' 2>/dev/null || echo 0)\nn=$((n+1))\necho \"$n\" > '" + dir + "/count'\ncat > '" + dir + "/input.'\"$n\"'.json'\nmkdir -p tests\nprintf 'assertion %s\\n' \"$n\" >> tests/expected.txt\necho '{\"notes\":\"Added boundary coverage\",\"concerns\":[\"Please check this boundary interpretation\"],\"coverage\":[{\"criterion\":1,\"tests\":[\"tests/expected.txt\"]}],\"run\":{\"model\":\"author-fixture\",\"turns\":1}}'\n"
	if err := os.WriteFile(path, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	return []string{path}, dir
}

func readProposalInput[T any](t *testing.T, directory string, call int) T {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(directory, fmt.Sprintf("input.%d.json", call)))
	if err != nil {
		t.Fatal(err)
	}
	var input T
	if err = json.Unmarshal(raw, &input); err != nil {
		t.Fatal(err)
	}
	return input
}

func TestDisputeProposalJudgeImprovesThenApprovesExactCumulativeTests(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	h := setupJobWith(t, []string{"go-test"}, 2, nil, func(p *factory.ProjectPolicy) { p.ProtectedPaths = []string{"tests/"} })
	judge, judgeInputs, _ := proposalJudge(t, []factory.JudgeDecision{proposalDecision(factory.DisputeAddTests), proposalDecision(factory.DisputeReviseTests), proposalDecision(factory.DisputeAcceptTests)}, 0)
	author, authorInputs := improvingProposalAuthor(t)
	reopenDisputeService(t, &h, judge, author)
	admission := h.job.Admission
	reportWorkerDispute(t, h, "proposal-loop")
	d := waitDispute(t, h, "resolved")
	if d.TestProposalCount != 2 || len(d.TestReviews) != 2 || d.Decision.Kind != factory.DisputeAddTests || d.Run.SessionID != "1" || d.NextOwner != "implementer" || d.Delegated == "" {
		t.Fatalf("lost initial judgment or review history: %+v", d)
	}
	first, second := d.TestReviews[0], d.TestReviews[1]
	if first.Decision.Kind != factory.DisputeReviseTests || first.NextOwner != "test_author" || first.Round != 1 || second.Decision.Kind != factory.DisputeAcceptTests || second.Round != 2 || second.NextOwner != "implementer" || first.Commit == second.Commit || first.ExpectationID == second.ExpectationID {
		t.Fatalf("wrong review routing: %+v", d.TestReviews)
	}
	authored := readProposalInput[factory.ExpectationInput](t, authorInputs, 2)
	if authored.PreviousProposal == nil || authored.BaseRevision != first.Commit || authored.PreviousProposal.Commit != first.Commit || authored.Dispute.TestReviews[0].Decision.Directive != first.Decision.Directive {
		t.Fatalf("improvement did not receive prior proposal and feedback: %+v", authored)
	}
	reviewed := readProposalInput[factory.JudgeInput](t, judgeInputs, 3)
	if reviewed.ProposedTests == nil || reviewed.ProposedTests.Commit != second.Commit || !strings.Contains(reviewed.ProposedTests.Patch, "+assertion 1") || !strings.Contains(reviewed.ProposedTests.Patch, "+assertion 2") || len(reviewed.ProposedTests.Concerns) != 1 || !reflect.DeepEqual(reviewed.Job.Admission, admission) {
		t.Fatalf("judge did not receive exact cumulative proposal and concerns: %+v", reviewed)
	}
	old, err := h.service.GetJob(ctx, h.coordinator, h.job.ID)
	if err != nil || old.Status != factory.JobSuperseded || !reflect.DeepEqual(old.Admission, admission) {
		t.Fatalf("original admission changed: %+v %v", old, err)
	}
	replacement, err := h.service.GetJob(ctx, h.coordinator, d.ReplacementJobID)
	if err != nil || replacement.Admission.ExpectationsCommit != second.Commit || replacement.Admission.ParentJobID != old.ID || len(replacement.Reviews) != 0 || replacement.Checks[0].Status != factory.CheckPending {
		t.Fatalf("replacement did not require fresh evidence: %+v %v", replacement, err)
	}
	retained, err := h.service.GetExpectations(ctx, h.coordinator, first.ExpectationID)
	if err != nil || retained.Commit != first.Commit || retained.Status != factory.ExpectationProposed {
		t.Fatalf("previous proposal was overwritten: %+v %v", retained, err)
	}
}

func TestDisputeProposalJudgeBoundsThreeProposals(t *testing.T) {
	t.Parallel()
	h := setupJobWith(t, []string{"go-test"}, 2, nil, func(p *factory.ProjectPolicy) { p.ProtectedPaths = []string{"tests/"} })
	judge, _, _ := proposalJudge(t, []factory.JudgeDecision{proposalDecision(factory.DisputeAddTests), proposalDecision(factory.DisputeReviseTests), proposalDecision(factory.DisputeReviseTests), proposalDecision(factory.DisputeReviseTests)}, 0)
	author, _ := improvingProposalAuthor(t)
	reopenDisputeService(t, &h, judge, author)
	reportWorkerDispute(t, h, "proposal-bound")
	d := waitDispute(t, h, "needs_human")
	if d.TestProposalCount != 3 || len(d.TestReviews) != 3 || d.NextOwner != "you" || d.ReplacementJobID != "" || !strings.Contains(d.Error, "three-proposal limit") || d.TestReviews[2].NextOwner != "you" {
		t.Fatalf("unbounded improvement loop: %+v", d)
	}
	for _, action := range d.AvailableActions {
		if action == "revise_tests" || action == "add_tests" {
			t.Fatalf("bound exposed further automatic authoring: %+v", d)
		}
	}
}

func TestDisputeProposalReviewStopRevocationAndStaleCommit(t *testing.T) {
	t.Parallel()
	for _, action := range []string{"stop", "revoke", "stale_commit", "restart"} {
		t.Run(action, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			h := setupJobWith(t, []string{"go-test"}, 2, nil, func(p *factory.ProjectPolicy) { p.ProtectedPaths = []string{"tests/"} })
			judge, inputs, gate := proposalJudge(t, []factory.JudgeDecision{proposalDecision(factory.DisputeAddTests), proposalDecision(factory.DisputeAcceptTests)}, 2)
			author, _ := improvingProposalAuthor(t)
			reopenDisputeService(t, &h, judge, author)
			reported := reportWorkerDispute(t, h, "review-"+action)
			waitDisputeFile(t, filepath.Join(inputs, "input.2.json"))
			d := waitDispute(t, h, "reviewing_tests")
			switch action {
			case "stop":
				if _, err := h.service.Action(ctx, h.coordinator, factory.JobActionRequest{RequestID: "stop-review", JobID: reported.ID, IfVersion: reported.Version, Action: factory.ActionCancel}); err != nil {
					t.Fatal(err)
				}
			case "revoke":
				if _, err := h.service.GrantProject(ctx, h.admin, factory.GrantProjectRequest{RequestID: "revoke-owner", ProjectID: h.project.ID, UserID: h.coordinator.UserID, Capabilities: []factory.Capability{factory.CapabilityRead}}); err != nil {
					t.Fatal(err)
				}
			case "stale_commit":
				// Simulate restoration of an independently changed proposal record.
				// The running judge still holds the original exact proposal.
				proposal, err := h.service.GetExpectations(ctx, h.coordinator, d.ExpectationID)
				if err != nil {
					t.Fatal(err)
				}
				proposal.Commit = h.job.Admission.BaseRevision
				raw, _ := json.Marshal(proposal)
				db, err := sql.Open("sqlite", filepath.Join(h.dataDir, "factory.db"))
				if err != nil {
					t.Fatal(err)
				}
				_, err = db.Exec(`UPDATE expectations SET data_json=? WHERE id=?`, string(raw), proposal.ID)
				_ = db.Close()
				if err != nil {
					t.Fatal(err)
				}
			case "restart":
				reopenDisputeService(t, &h, judge, author)
			}
			if err := os.WriteFile(gate, []byte("go"), 0600); err != nil {
				t.Fatal(err)
			}
			status := "needs_human"
			if action == "stop" {
				status = "stopped"
			}
			d = waitDispute(t, h, status)
			if d.ReplacementJobID != "" || len(d.TestReviews) != 1 || d.TestReviews[0].Error == "" {
				t.Fatalf("late review routed new work: %+v", d)
			}
			if action == "restart" {
				count, err := os.ReadFile(filepath.Join(inputs, "count"))
				if err != nil || strings.TrimSpace(string(count)) != "2" {
					t.Fatalf("restart repeated external review: %s %v", count, err)
				}
			}
		})
	}
}

func TestDisputeProposalJudgmentCannotWeakenOrChangeIntent(t *testing.T) {
	t.Parallel()
	for _, unsafe := range []string{"weakening", "intent", "repair_phase", "initial_accept"} {
		t.Run(unsafe, func(t *testing.T) {
			t.Parallel()
			h := setupJobWith(t, []string{"go-test"}, 2, nil, func(p *factory.ProjectPolicy) { p.ProtectedPaths = []string{"tests/"} })
			decision := proposalDecision(factory.DisputeAcceptTests)
			if unsafe == "weakening" {
				decision.WeakensAcceptance = true
			}
			if unsafe == "intent" {
				decision.PreservesIntent = false
			}
			if unsafe == "repair_phase" {
				decision.Kind = factory.DisputeRepair
			}
			decisions := []factory.JudgeDecision{proposalDecision(factory.DisputeAddTests), decision}
			if unsafe == "initial_accept" {
				decisions = []factory.JudgeDecision{decision}
			}
			judge, _, _ := proposalJudge(t, decisions, 0)
			author, _ := improvingProposalAuthor(t)
			reopenDisputeService(t, &h, judge, author)
			reportWorkerDispute(t, h, "unsafe-"+unsafe)
			d := waitDispute(t, h, "needs_human")
			if d.NextOwner != "you" || d.ReplacementJobID != "" {
				t.Fatalf("unsafe judgment dispatched work: %+v", d)
			}
		})
	}
}

func TestDisputeLegacyProposalCanRequestJudgeReviewWithVersionAndReplay(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	h := setupJobWith(t, []string{"go-test"}, 2, nil, func(p *factory.ProjectPolicy) { p.ProtectedPaths = []string{"tests/"} })
	author, _ := improvingProposalAuthor(t)
	reopenDisputeService(t, &h, nil, author)
	reportWorkerDispute(t, h, "legacy-proposal")
	d := waitDispute(t, h, "needs_human")
	if _, err := h.service.ResolveDispute(ctx, h.coordinator, factory.ResolveDisputeRequest{RequestID: "write-proposal", DisputeID: d.ID, IfVersion: d.Version, Action: "add_tests", Rationale: "Write a correction for independent review"}); err != nil {
		t.Fatal(err)
	}
	d = waitDispute(t, h, "needs_human")
	if d.ExpectationID == "" {
		t.Fatal("no concrete legacy proposal")
	}
	if err := h.service.Close(); err != nil {
		t.Fatal(err)
	}
	// The prior release stored a human-pending proposal with no review ledger.
	d.Status, d.NextOwner, d.Error = "tests_proposed", "you", ""
	d.TestProposalCount, d.TestReviews = 0, nil
	raw, _ := json.Marshal(d)
	db, err := sql.Open("sqlite", filepath.Join(h.dataDir, "factory.db"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`UPDATE disputes SET status=?,data_json=? WHERE id=?`, d.Status, string(raw), d.ID)
	_ = db.Close()
	if err != nil {
		t.Fatal(err)
	}
	judge, inputs, gate := proposalJudge(t, []factory.JudgeDecision{proposalDecision(factory.DisputeAcceptTests)}, 1)
	reopenDisputeService(t, &h, judge, author)
	request := factory.ResolveDisputeRequest{RequestID: "review-legacy", DisputeID: d.ID, IfVersion: d.Version, Action: "review_tests", Rationale: "Ask the independent judge to assess this proposal"}
	if _, err = h.service.ResolveDispute(ctx, h.reviewer, request); factory.ErrorCodeOf(err) != factory.CodeForbidden {
		t.Fatalf("read-only reviewer started judgment: %v", err)
	}
	stale := request
	stale.IfVersion--
	if _, err = h.service.ResolveDispute(ctx, h.coordinator, stale); factory.ErrorCodeOf(err) != factory.CodeConflict {
		t.Fatalf("stale decision started judgment: %v", err)
	}
	started, err := h.service.ResolveDispute(ctx, h.coordinator, request)
	if err != nil || started.Status != "reviewing_tests" || started.TestProposalCount != 1 {
		t.Fatalf("legacy review not started: %+v %v", started, err)
	}
	replay, err := h.service.ResolveDispute(ctx, h.coordinator, request)
	if err != nil || !reflect.DeepEqual(started, replay) {
		t.Fatalf("review replay changed response: %+v %+v %v", started, replay, err)
	}
	waitDisputeFile(t, filepath.Join(inputs, "input.1.json"))
	if err = os.WriteFile(gate, []byte("go"), 0600); err != nil {
		t.Fatal(err)
	}
	resolved := waitDispute(t, h, "resolved")
	if len(resolved.TestReviews) != 1 || resolved.TestReviews[0].ExpectationID != d.ExpectationID || resolved.ReplacementJobID == "" {
		t.Fatalf("legacy review lost proposal identity: %+v", resolved)
	}
	replay, err = h.service.ResolveDispute(ctx, h.coordinator, request)
	if err != nil || !reflect.DeepEqual(started, replay) {
		t.Fatalf("completed review replay changed response: %+v %v", replay, err)
	}
	count, err := os.ReadFile(filepath.Join(inputs, "count"))
	if err != nil || strings.TrimSpace(string(count)) != "1" {
		t.Fatalf("replay repeated judge invocation: %s %v", count, err)
	}
}
