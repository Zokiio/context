package factory_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Zokiio/context/internal/factory"
)

func disputeReport() factory.DisputeInput {
	return factory.DisputeInput{Summary: "The frozen test contradicts criterion one", CriterionRefs: []int{1}, TestPaths: []string{"tests/expected.txt"}, ObservedFailure: "Expected two, observed three", EvidenceRefs: []string{"criterion:1", "check:go-test"}, ProposedChange: "Check the stated result without changing the requirement"}
}
func judgeCommand(t *testing.T, kind factory.DisputeDecisionKind, extra string) ([]string, string) {
	t.Helper()
	dir := t.TempDir()
	dump := filepath.Join(dir, "input.json")
	output, _ := json.Marshal(factory.JudgeOutput{Decision: factory.JudgeDecision{Kind: kind, Rationale: "The admitted requirement establishes the intended result", EvidenceRefs: []string{"criterion:1"}, PreservesIntent: true, Directive: "Follow criterion one exactly; preserve all other behavior"}, Run: factory.RefinerRun{Model: "independent-fixture", Turns: 1}})
	path := filepath.Join(dir, "judge")
	script := "#!/bin/sh\ncat > '" + dump + "'\n" + extra + "\n"
	if kind == factory.DisputeReviseTests || kind == factory.DisputeAddTests {
		approved := strings.Replace(string(output), `"kind":"`+string(kind)+`"`, `"kind":"accept_tests"`, 1)
		script += "if grep -q 'proposedTests' '" + dump + "'; then\ncat <<'APPROVED'\n" + approved + "\nAPPROVED\nexit 0\nfi\n"
	}
	script += "cat <<'RESULT'\n" + string(output) + "\nRESULT\n"
	if err := os.WriteFile(path, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	return []string{path}, dump
}
func reopenDisputeService(t *testing.T, h *jobHarness, judge, author []string) {
	t.Helper()
	if err := h.service.Close(); err != nil {
		t.Fatal(err)
	}
	service, err := factory.Open(h.dataDir, factory.Options{JudgeCommand: judge, ExpectationCommand: author, LeaseDuration: time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	h.service = service
	t.Cleanup(func() { _ = service.Close() })
}
func reportWorkerDispute(t *testing.T, h jobHarness, key string) factory.Job {
	t.Helper()
	a := startActivityAttempt(t, h, key)
	input := disputeReport()
	request := factory.WorkerReportRequest{RequestID: key + "-dispute", AttemptID: a.AttemptID, DispatchID: a.DispatchID, Kind: factory.ReportTerminal, ObservedAt: time.Now(), NativeRunID: key, Outcome: factory.OutcomeDisputed, TerminalReceipt: "process exited without committing", Dispute: &input, Artifacts: []factory.ArtifactInput{{Name: "failure.log", Content: "Expected two, observed three", MediaType: "text/plain"}}}
	job, err := h.service.Report(context.Background(), h.implementer, request)
	if err != nil {
		t.Fatal(err)
	}
	replay, err := h.service.Report(context.Background(), h.implementer, request)
	if err != nil || !reflect.DeepEqual(job, replay) {
		t.Fatalf("terminal dispute replay changed: %v", err)
	}
	if job.Status != factory.JobDisputed || job.CandidateRevision != "" {
		t.Fatalf("dispute accepted candidate or wrong state: %+v", job)
	}
	return job
}
func waitDispute(t *testing.T, h jobHarness, status string) factory.Dispute {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		records, err := h.service.ListDisputes(context.Background(), h.coordinator, h.project.ID, "work-test")
		if err != nil {
			t.Fatal(err)
		}
		if len(records) > 0 {
			if records[0].Status == status {
				return records[0]
			}
			if records[0].Status == "needs_human" && status != "needs_human" {
				t.Fatalf("unexpected escalation: %+v", records[0])
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("dispute did not reach %s", status)
	return factory.Dispute{}
}

func TestDisputeRoutesImplementationRepairAndBoundsRepeatedDisagreement(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	h := setupJob(t)
	judge, dump := judgeCommand(t, factory.DisputeRepair, "")
	reopenDisputeService(t, &h, judge, nil)
	original := h.job.Admission
	reportWorkerDispute(t, h, "disputed-build")
	d := waitDispute(t, h, "resolved")
	job, err := h.service.GetJob(ctx, h.coordinator, h.job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if job.Status != factory.JobQueued || job.RepairCount != 1 || len(job.Attempts) != 2 || !reflect.DeepEqual(job.Admission, original) || d.NextOwner != "implementer" {
		t.Fatalf("repair changed frozen contract or failed handoff: %+v %+v", job, d)
	}
	raw, err := os.ReadFile(dump)
	if err != nil {
		t.Fatal(err)
	}
	var input factory.JudgeInput
	if err = json.Unmarshal(raw, &input); err != nil {
		t.Fatal(err)
	}
	if len(input.Job.Admission.Sources) == 0 || input.Job.Attempts[0].Artifacts[0].Content != "Expected two, observed three" {
		t.Fatalf("judge lacks retained evidence: %+v", input)
	}
	assignment, err := h.service.Claim(ctx, h.implementer, factory.WorkerClaimRequest{RequestID: "repair-claim", Stages: []factory.Stage{factory.StageImplementation}, ProjectID: &h.project.ID})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(assignment.Instructions, d.Decision.Directive) {
		t.Fatalf("repair lacks judge directive: %s", assignment.Instructions)
	}
	for _, kind := range []factory.WorkerReportKind{factory.ReportStartRegistered, factory.ReportStarted} {
		if _, err = h.service.Report(ctx, h.implementer, factory.WorkerReportRequest{RequestID: string(kind) + "-second", AttemptID: assignment.AttemptID, DispatchID: assignment.DispatchID, Kind: kind, ObservedAt: time.Now(), NativeRunID: "second"}); err != nil {
			t.Fatal(err)
		}
	}
	report := disputeReport()
	if _, err = h.service.Report(ctx, h.implementer, factory.WorkerReportRequest{RequestID: "second-dispute", AttemptID: assignment.AttemptID, DispatchID: assignment.DispatchID, Kind: factory.ReportTerminal, ObservedAt: time.Now(), NativeRunID: "second", Outcome: factory.OutcomeDisputed, TerminalReceipt: "stopped", Dispute: &report}); err != nil {
		t.Fatal(err)
	}
	repeated := waitDispute(t, h, "needs_human")
	if repeated.Decision != nil || !strings.Contains(repeated.Error, "already used") {
		t.Fatalf("second cycle was not bounded: %+v", repeated)
	}
	request := factory.ResolveDisputeRequest{RequestID: "back-to-planning", DisputeID: repeated.ID, IfVersion: repeated.Version, Action: "return_to_planning", Rationale: "The intent needs to be clarified"}
	if _, err = h.service.ResolveDispute(ctx, h.reviewer, request); factory.ErrorCodeOf(err) != factory.CodeForbidden {
		t.Fatalf("reviewer routed work: %v", err)
	}
	stale := request
	stale.IfVersion--
	if _, err = h.service.ResolveDispute(ctx, h.coordinator, stale); err == nil {
		t.Fatal("stale decision routed work")
	}
	resolved, err := h.service.ResolveDispute(ctx, h.coordinator, request)
	if err != nil {
		t.Fatal(err)
	}
	replay, err := h.service.ResolveDispute(ctx, h.coordinator, request)
	if err != nil || !reflect.DeepEqual(resolved, replay) {
		t.Fatalf("resolution replay: %+v %+v %v", resolved, replay, err)
	}
	record, err := h.service.GetWorkItem(ctx, h.coordinator, h.project.ID, "work-test")
	if err != nil || record.Frozen || !record.Stopped {
		t.Fatalf("return to planning left editor frozen or dispatch enabled: %+v %v", record, err)
	}
	if _, err = h.service.SetWorkItemDisposition(ctx, h.coordinator, factory.SetWorkItemDispositionRequest{RequestID: "resume-planning", ProjectID: h.project.ID, WorkItemID: "work-test", IfTicketSHA256: record.TicketSHA256, Action: "queue"}); err != nil {
		t.Fatal(err)
	}
	board, err := h.service.Board(ctx, h.coordinator, h.project.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(board.Cards) != 1 || board.Cards[0].JobID != "" || board.Cards[0].Decision != factory.CardDecisionPlan {
		t.Fatalf("superseded run blocks planning: %+v", board.Cards)
	}
}

func TestDisputeTestRevisionJudgeApprovesAndCreatesFreshReplacement(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	oldAuthor := fakeAuthor(t, map[string]string{"tests/expected.txt": "original expectation\n"})
	h := setupExpectations(t, oldAuthor, nil)
	original := h.write(t, "original-tests")
	accepted, err := h.service.AcceptExpectations(ctx, h.admin, factory.DecideExpectationsRequest{RequestID: "accept-original", ExpectationID: original.ID})
	if err != nil {
		t.Fatal(err)
	}
	h.job, err = h.service.Submit(ctx, h.coordinator, factory.SubmitRequest{RequestID: "initial", ProjectID: h.project.ID, WorkItemID: "work-test", ExpectationsID: accepted.ID, ExecutorKey: "fixture-agent", RepairLimit: 2, Deadline: time.Now().Add(20 * time.Minute)})
	if err != nil {
		t.Fatal(err)
	}
	admitted := h.job.Admission
	judge, dump := judgeCommand(t, factory.DisputeReviseTests, "")
	newAuthor := fakeAuthor(t, map[string]string{"tests/expected.txt": "corrected expectation\n"})
	reopenDisputeService(t, &h.jobHarness, judge, newAuthor)
	reportWorkerDispute(t, h.jobHarness, "test-dispute")
	d := waitDispute(t, h.jobHarness, "resolved")
	if d.NextOwner != "implementer" || d.ReplacementJobID == "" || len(d.TestReviews) != 1 || d.TestReviews[0].Decision.Kind != factory.DisputeAcceptTests || d.Decision.Kind != factory.DisputeReviseTests {
		t.Fatalf("test revision did not receive independent review: %+v", d)
	}
	raw, err := os.ReadFile(dump)
	if err != nil {
		t.Fatal(err)
	}
	var input factory.JudgeInput
	if err = json.Unmarshal(raw, &input); err != nil {
		t.Fatal(err)
	}
	if input.OriginalTests == nil || input.OriginalTests.ID != original.ID || len(input.TestSources) != 1 || input.TestSources[0].Text != "original expectation\n" {
		t.Fatalf("judge did not receive original protected tests: %+v", input)
	}
	proposal, err := h.service.GetExpectations(ctx, h.coordinator, d.ExpectationID)
	if err != nil || proposal.DisputeID != d.ID || proposal.BaseRevision != admitted.BaseRevision || !strings.Contains(proposal.Patch, "corrected expectation") {
		t.Fatalf("bad test proposal: %+v %v", proposal, err)
	}
	if _, err = h.service.AcceptExpectations(ctx, h.coordinator, factory.DecideExpectationsRequest{RequestID: "bypass", ExpectationID: proposal.ID}); factory.ErrorCodeOf(err) != factory.CodeConflict {
		t.Fatalf("ordinary accept bypassed dispute: %v", err)
	}
	resolved := d
	old, err := h.service.GetJob(ctx, h.coordinator, h.job.ID)
	if err != nil || old.Status != factory.JobSuperseded || !reflect.DeepEqual(old.Admission, admitted) {
		t.Fatalf("old run changed: %+v %v", old, err)
	}
	replacement, err := h.service.GetJob(ctx, h.coordinator, resolved.ReplacementJobID)
	if err != nil {
		t.Fatal(err)
	}
	if replacement.Admission.ParentJobID != old.ID || replacement.Admission.LineageID != old.ID || replacement.Admission.ExpectationsID != proposal.ID || len(replacement.Checks) != 1 || replacement.Checks[0].Status != factory.CheckPending || replacement.Checks[0].ObservedAt != nil || len(replacement.Reviews) != 0 || replacement.ID == old.ID {
		t.Fatalf("replacement reused stale evidence: %+v", replacement)
	}
	h.job = replacement
	verified, _ := passVerificationForTest(t, h.jobHarness, "replacement")
	if verified.Status != factory.JobReadyForReview || len(verified.Checks) != 1 || verified.Checks[0].RequirementFingerprint != admitted.TicketSHA256 {
		t.Fatalf("replacement did not freshly verify: %+v", verified)
	}
	history, err := h.service.WorkItemActivity(ctx, h.coordinator, h.project.ID, "work-test")
	if err != nil || len(history.Jobs) != 2 || len(history.Disputes) != 1 {
		t.Fatalf("history lost original: %+v %v", history, err)
	}
	run, err := h.service.WorkItemRun(ctx, h.coordinator, h.project.ID, "work-test")
	if err != nil || run.Steps[len(run.Steps)-1].Key != "judge" {
		t.Fatalf("replacement lost judge decision: %+v %v", run, err)
	}
}

func TestDisputeStopPreventsLateJudgeRouteAndRestartDoesNotRepeat(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	h := setupJob(t)
	gate := filepath.Join(t.TempDir(), "continue")
	judge, dump := judgeCommand(t, factory.DisputeRepair, "while [ ! -f '"+gate+"' ]; do sleep 0.02; done")
	reopenDisputeService(t, &h, judge, nil)
	reported := reportWorkerDispute(t, h, "stop-judge")
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(dump); err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	stopped, err := h.service.Action(ctx, h.coordinator, factory.JobActionRequest{RequestID: "stop-during-judge", JobID: reported.ID, IfVersion: reported.Version, Action: factory.ActionCancel})
	if err != nil || stopped.Status != factory.JobCancelled {
		t.Fatalf("stop dispute: %+v %v", stopped, err)
	}
	if err = os.WriteFile(gate, []byte("go"), 0600); err != nil {
		t.Fatal(err)
	}
	d := waitDispute(t, h, "stopped")
	reopenDisputeService(t, &h, judge, nil)
	retained, err := h.service.GetDispute(ctx, h.coordinator, d.ID)
	if err != nil || retained.Status != "stopped" || len(retained.AvailableActions) != 0 {
		t.Fatalf("stop lost on restart: %+v %v", retained, err)
	}
	current, err := h.service.GetJob(ctx, h.coordinator, h.job.ID)
	if err != nil || len(current.Attempts) != 1 || current.Status != factory.JobCancelled {
		t.Fatalf("late judge restarted: %+v %v", current, err)
	}
}

func TestDisputeMalformedJudgeEscalatesAndHumanReportChecksVersion(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	h := setupJob(t)
	judge, dump := judgeCommand(t, factory.DisputeRepair, "printf '{}\\n'")
	reopenDisputeService(t, &h, judge, nil)
	job, _ := passVerificationForTest(t, h, "human-dispute")
	request := factory.ReportDisputeRequest{RequestID: "human-report", JobID: job.ID, IfVersion: job.Version, Dispute: disputeReport()}
	stale := request
	stale.IfVersion--
	if _, err := h.service.ReportDispute(ctx, h.coordinator, stale); factory.ErrorCodeOf(err) != factory.CodeConflict {
		t.Fatalf("stale human report: %v", err)
	}
	if _, err := h.service.ReportDispute(ctx, h.reviewer, request); factory.ErrorCodeOf(err) != factory.CodeForbidden {
		t.Fatalf("reviewer reported dispute: %v", err)
	}
	initial, err := h.service.ReportDispute(ctx, h.coordinator, request)
	if err != nil {
		t.Fatal(err)
	}
	replay, err := h.service.ReportDispute(ctx, h.coordinator, request)
	if err != nil || !reflect.DeepEqual(initial, replay) {
		t.Fatalf("human report replay: %+v %+v %v", initial, replay, err)
	}
	d := waitDispute(t, h, "needs_human")
	if d.Error == "" || d.Decision != nil {
		t.Fatalf("invalid judge output routed work: %+v", d)
	}
	raw, err := os.ReadFile(dump)
	if err != nil {
		t.Fatal(err)
	}
	var input factory.JudgeInput
	if err = json.Unmarshal(raw, &input); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(input.CandidatePatch, "+human-dispute") {
		t.Fatalf("judge lacked candidate contents: %q", input.CandidatePatch)
	}
	reader, err := h.service.GetDispute(ctx, h.reviewer, d.ID)
	if err != nil || len(reader.AvailableActions) != 0 {
		t.Fatalf("read actions not filtered: %+v %v", reader, err)
	}
}

func gatedDisputeAuthor(t *testing.T) ([]string, string, string) {
	t.Helper()
	command := fakeAuthor(t, map[string]string{"tests/new.txt": "check original intent\n"})
	dir := t.TempDir()
	started, gate := filepath.Join(dir, "started"), filepath.Join(dir, "continue")
	raw, err := os.ReadFile(command[0])
	if err != nil {
		t.Fatal(err)
	}
	script := strings.Replace(string(raw), "cat > /dev/null\n", "cat > /dev/null\ntouch '"+started+"'\nwhile [ ! -f '"+gate+"' ]; do sleep 0.02; done\n", 1)
	if err = os.WriteFile(command[0], []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	return command, started, gate
}
func waitDisputeFile(t *testing.T, path string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("subprocess did not reach gate")
}

func TestDisputeJudgeTestApprovalRechecksCurrentRunOwner(t *testing.T) {
	t.Parallel()
	for _, revoked := range []bool{false, true} {
		name := "authorized"
		if revoked {
			name = "revoked"
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			h := setupJobWith(t, []string{"go-test"}, 2, nil, func(p *factory.ProjectPolicy) { p.ProtectedPaths = []string{"tests/"} })
			judge, _ := judgeCommand(t, factory.DisputeAddTests, "")
			author, started, gate := gatedDisputeAuthor(t)
			reopenDisputeService(t, &h, judge, author)
			reportWorkerDispute(t, h, "delegated-test-dispute")
			waitDisputeFile(t, started)
			if revoked {
				if _, err := h.service.GrantProject(ctx, h.admin, factory.GrantProjectRequest{RequestID: "revoke-owner", ProjectID: h.project.ID, UserID: h.coordinator.UserID, Capabilities: []factory.Capability{factory.CapabilityRead}}); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.WriteFile(gate, []byte("go"), 0600); err != nil {
				t.Fatal(err)
			}
			status := "resolved"
			if revoked {
				status = "needs_human"
			}
			d := waitDispute(t, h, status)
			proposal, err := h.service.GetExpectations(ctx, h.admin, d.ExpectationID)
			if err != nil {
				t.Fatal(err)
			}
			if revoked {
				if d.NextOwner != "you" || d.ReplacementJobID != "" || proposal.Status != factory.ExpectationProposed {
					t.Fatalf("revoked coordinator dispatched: %+v %+v", d, proposal)
				}
			} else {
				replacement, err := h.service.GetJob(ctx, h.admin, d.ReplacementJobID)
				if err != nil {
					t.Fatal(err)
				}
				if proposal.DecidedBy == nil || *proposal.DecidedBy != h.coordinator.UserID || !strings.Contains(proposal.Delegated, "independent judge") || d.ResolvedBy == nil || *d.ResolvedBy != h.coordinator.UserID || replacement.CoordinatorID != h.coordinator.UserID {
					t.Fatalf("judge continuation misattributed: %+v %+v", d, proposal)
				}
			}
		})
	}
}

func TestDisputeStopDuringAuthorPreventsReplacement(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	h := setupJobWith(t, []string{"go-test"}, 2, nil, func(p *factory.ProjectPolicy) { p.ProtectedPaths = []string{"tests/"} })
	judge, _ := judgeCommand(t, factory.DisputeAddTests, "")
	author, started, gate := gatedDisputeAuthor(t)
	reopenDisputeService(t, &h, judge, author)
	reportWorkerDispute(t, h, "stop-author")
	waitDisputeFile(t, started)
	d := waitDispute(t, h, "authoring_tests")
	job, err := h.service.GetJob(ctx, h.coordinator, h.job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = h.service.Action(ctx, h.coordinator, factory.JobActionRequest{RequestID: "stop-author-action", JobID: job.ID, IfVersion: job.Version, Action: factory.ActionCancel}); err != nil {
		t.Fatal(err)
	}
	// Do not release the gate until Close confirms cancellation terminates the author.
	closed := make(chan error, 1)
	go func() { closed <- h.service.Close() }()
	select {
	case err = <-closed:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		_ = os.WriteFile(gate, []byte("go"), 0600)
		t.Fatal("stop did not terminate the test author")
	}
	service, err := factory.Open(h.dataDir, factory.Options{JudgeCommand: judge, ExpectationCommand: author})
	if err != nil {
		t.Fatal(err)
	}
	h.service = service
	t.Cleanup(func() { _ = service.Close() })
	d, err = h.service.GetDispute(ctx, h.coordinator, d.ID)
	if err != nil || d.Status != "stopped" || d.ReplacementJobID != "" {
		t.Fatalf("author stop created replacement: %+v %v", d, err)
	}
	proposal, err := h.service.GetExpectations(ctx, h.coordinator, d.ExpectationID)
	if err != nil || proposal.Status != factory.ExpectationFailed {
		t.Fatalf("author did not terminate: %+v %v", proposal, err)
	}
}

func TestDisputeRunGraphPreservesFailedChecksAfterReturningToPlanning(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	h := setupJob(t)
	completeImplementationForTest(t, h, "check-dispute")
	a := startVerificationForTest(t, h, "check-dispute")
	job, err := h.service.Report(ctx, h.verifier, factory.WorkerReportRequest{RequestID: "failed-check", AttemptID: a.AttemptID, DispatchID: a.DispatchID, Kind: factory.ReportTerminal, ObservedAt: time.Now(), NativeRunID: "check-dispute-verify-native", Outcome: factory.OutcomeFailed, CheckStatus: factory.CheckFailed, TerminalReceipt: "exit 1"})
	if err != nil {
		t.Fatal(err)
	}
	d, err := h.service.ReportDispute(ctx, h.coordinator, factory.ReportDisputeRequest{RequestID: "dispute-check", JobID: job.ID, IfVersion: job.Version, Dispute: disputeReport()})
	if err != nil {
		t.Fatal(err)
	}
	for _, superseded := range []bool{false, true} {
		if superseded {
			if _, err = h.service.ResolveDispute(ctx, h.coordinator, factory.ResolveDisputeRequest{RequestID: "plan-check", DisputeID: d.ID, IfVersion: d.Version, Action: "return_to_planning", Rationale: "Clarify what this check should assert"}); err != nil {
				t.Fatal(err)
			}
		}
		run, err := h.service.WorkItemRun(ctx, h.coordinator, h.project.ID, "work-test")
		if err != nil {
			t.Fatal(err)
		}
		if run.JobID != job.ID || run.Steps[3].Status != factory.RunDone || run.Steps[4].Status != factory.RunFailed || run.Steps[4].Summary != "A check failed" {
			t.Fatalf("dispute erased verification, superseded=%t: %+v", superseded, run)
		}
	}
}
