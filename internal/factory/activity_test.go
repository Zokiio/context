package factory_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/Zokiio/context/internal/factory"
)

func startActivityAttempt(t *testing.T, h jobHarness, native string) factory.Assignment {
	t.Helper()
	ctx := context.Background()
	a, err := h.service.Claim(ctx, h.implementer, factory.WorkerClaimRequest{RequestID: native + "-claim", Stages: []factory.Stage{factory.StageImplementation}, ProjectID: &h.project.ID})
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range []factory.WorkerReportKind{factory.ReportStartRegistered, factory.ReportStarted} {
		if _, err := h.service.Report(ctx, h.implementer, factory.WorkerReportRequest{RequestID: native + "-" + string(kind), AttemptID: a.AttemptID, DispatchID: a.DispatchID, Kind: kind, ObservedAt: time.Now(), NativeRunID: native}); err != nil {
			t.Fatal(err)
		}
	}
	return a
}

func assertActivityBoardStatus(t *testing.T, h jobHarness, status, decision string) {
	t.Helper()
	board, err := h.service.Board(context.Background(), h.coordinator, h.project.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, card := range board.Cards {
		if card.WorkItemID == "work-test" {
			if card.Status != status || card.Decision != decision {
				t.Fatalf("board obscured stop state: %+v; want %q, %q", card, status, decision)
			}
			return
		}
	}
	t.Fatal("stopped task disappeared from board")
}

func TestWorkItemActivityRetainsAttemptsProgressAndOriginalEvidence(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	h := setupJob(t)
	a := startActivityAttempt(t, h, "activity-first")
	if _, err := h.service.Heartbeat(ctx, h.implementer, factory.HeartbeatRequest{RequestID: "activity-progress", AttemptID: a.AttemptID, DispatchID: a.DispatchID, Progress: "Checking the implementation"}); err != nil {
		t.Fatal(err)
	}
	live, err := h.service.WorkItemActivity(ctx, h.coordinator, h.project.ID, "work-test")
	if err != nil {
		t.Fatal(err)
	}
	attempt := live.Jobs[0].Attempts[0]
	if live.GeneratedAt.IsZero() || live.Truncated || attempt.Progress != "Checking the implementation" || attempt.LastHeartbeatAt == nil || attempt.FinishedAt != nil || len(attempt.Artifacts) != 0 {
		t.Fatalf("live observation invented or lost output: %+v", live)
	}
	if !hasEvent(live.Jobs[0], "worker_progress", "Checking the implementation") {
		t.Fatalf("progress history is missing: %+v", live.Jobs[0].Events)
	}
	const receipt = `{"state":"failed","commands":[{"log":"raw receipt output"}]}`
	first, err := h.service.Report(ctx, h.implementer, factory.WorkerReportRequest{RequestID: "activity-failed", AttemptID: a.AttemptID, DispatchID: a.DispatchID, Kind: factory.ReportTerminal, ObservedAt: time.Now(), NativeRunID: "activity-first", Outcome: factory.OutcomeFailed, TerminalReceipt: receipt, Artifacts: []factory.ArtifactInput{{Name: "command-1.log", MediaType: "text/plain", Content: "original failure output"}}})
	if err != nil {
		t.Fatal(err)
	}
	repaired, err := h.service.Action(ctx, h.coordinator, factory.JobActionRequest{RequestID: "activity-repair", JobID: first.ID, IfVersion: first.Version, Action: factory.ActionRepair})
	if err != nil {
		t.Fatal(err)
	}
	history, err := h.service.WorkItemActivity(ctx, h.coordinator, h.project.ID, "work-test")
	if err != nil {
		t.Fatal(err)
	}
	job := history.Jobs[0]
	if job.ID != repaired.ID || len(job.Attempts) != 2 || job.Attempts[0].Outcome != factory.OutcomeFailed || job.Attempts[1].Status != factory.AttemptPending {
		t.Fatalf("repair replaced original attempt: %+v", job)
	}
	payload, err := json.Marshal(history)
	if err != nil || job.Attempts[0].TerminalReceipt != "" || strings.Contains(string(payload), "terminalReceipt") || strings.Contains(string(payload), "raw receipt output") {
		t.Fatalf("polling copied raw terminal receipt output: %s, %v", payload, err)
	}
	if job.Admission.TicketSHA256 != h.job.Admission.TicketSHA256 || len(job.Admission.Sources) == 0 || job.Admission.Sources[0].Text == "" {
		t.Fatalf("original requirements were lost: %+v", job.Admission)
	}
	artifact := job.Attempts[0].Artifacts[0]
	if artifact.Content != "" || artifact.Bytes != len("original failure output") || artifact.SHA256 == "" {
		t.Fatalf("activity must retain artifact metadata without repeatedly copying logs: %+v", artifact)
	}
	retained, err := h.service.GetJob(ctx, h.coordinator, job.ID)
	if err != nil || retained.Attempts[0].TerminalReceipt != receipt || retained.Attempts[0].Artifacts[0].Content != "original failure output" {
		t.Fatalf("polling lost retained artifact: %+v, %v", retained, err)
	}
	actors := map[factory.UserID]string{}
	for _, actor := range history.Actors {
		actors[actor.ID] = actor.Username
	}
	if actors[h.coordinator.UserID] != "coordinator" || actors[h.implementer.UserID] != "implementer" {
		t.Fatalf("participant identities missing: %+v", history.Actors)
	}
}

func TestWorkItemActivityEnforcesProjectAccessAndActionCapabilities(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	h := setupJob(t)
	user, err := h.service.CreateUser(ctx, h.admin, factory.CreateUserRequest{RequestID: "activity-outsider", Username: "outsider", Kind: factory.UserKindHuman})
	if err != nil {
		t.Fatal(err)
	}
	outsider := factory.Principal{UserID: user.ID}
	if _, err := h.service.WorkItemActivity(ctx, outsider, h.project.ID, "work-test"); factory.ErrorCodeOf(err) != factory.CodeNotFound {
		t.Fatalf("activity leaked across projects: %v", err)
	}
	if _, err := h.service.GrantProject(ctx, h.admin, factory.GrantProjectRequest{RequestID: "activity-reader", ProjectID: h.project.ID, UserID: user.ID, Capabilities: []factory.Capability{factory.CapabilityRead}}); err != nil {
		t.Fatal(err)
	}
	history, err := h.service.WorkItemActivity(ctx, outsider, h.project.ID, "work-test")
	if err != nil || len(history.Jobs) != 1 || len(history.Jobs[0].AvailableActions) != 0 {
		t.Fatalf("read-only activity advertised mutations: %+v, %v", history, err)
	}
	coordinated, err := h.service.WorkItemActivity(ctx, h.coordinator, h.project.ID, "work-test")
	if err != nil || len(coordinated.Jobs[0].AvailableActions) != 1 || coordinated.Jobs[0].AvailableActions[0] != factory.ActionCancel {
		t.Fatalf("coordinator cannot see stop: %+v, %v", coordinated, err)
	}
	if _, err := h.service.WorkItemActivity(ctx, h.admin, h.project.ID, "missing-task"); factory.ErrorCodeOf(err) != factory.CodeNotFound {
		t.Fatalf("unknown task activity: %v", err)
	}
}

func TestWorkItemActivityKeepsCancelledJobsAfterExplicitRequeue(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	h := setupJob(t)
	old, err := h.service.Action(ctx, h.coordinator, factory.JobActionRequest{RequestID: "history-cancel", JobID: h.job.ID, IfVersion: h.job.Version, Action: factory.ActionCancel})
	if err != nil {
		t.Fatal(err)
	}
	record, err := h.service.GetWorkItem(ctx, h.coordinator, h.project.ID, "work-test")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.service.SetWorkItemDisposition(ctx, h.coordinator, factory.SetWorkItemDispositionRequest{RequestID: "history-requeue", ProjectID: h.project.ID, WorkItemID: "work-test", IfTicketSHA256: record.TicketSHA256, Action: "queue"}); err != nil {
		t.Fatal(err)
	}
	newer, err := h.service.Submit(ctx, h.coordinator, factory.SubmitRequest{RequestID: "history-resubmit", ProjectID: h.project.ID, WorkItemID: "work-test", BaseRevision: "HEAD", ExecutorKey: "fixture-agent", RepairLimit: 1, Deadline: time.Now().Add(30 * time.Minute)})
	if err != nil {
		t.Fatal(err)
	}
	activity, err := h.service.WorkItemActivity(ctx, h.coordinator, h.project.ID, "work-test")
	if err != nil {
		t.Fatal(err)
	}
	if activity.Held || len(activity.Jobs) != 2 || activity.Jobs[0].ID != newer.ID || activity.Jobs[1].ID != old.ID || activity.Jobs[1].Status != factory.JobCancelled {
		t.Fatalf("requeue lost the cancelled run: %+v", activity)
	}
	if activity.Jobs[1].Attempts[0].Status != factory.AttemptCancelled || activity.Jobs[1].Admission.ContextSHA256 != old.Admission.ContextSHA256 {
		t.Fatalf("older attempt or requirement identity changed: %+v", activity.Jobs[1])
	}
}

func TestExplicitStopSurvivesRestartAndPreventsAutopilotPickup(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	h := setupJob(t)
	if _, err := h.service.SetAskPolicy(ctx, h.admin, factory.SetAskPolicyRequest{RequestID: "activity-ask", ProjectID: h.project.ID, BeforeBuild: factory.AskBeforeNever, AfterBuild: factory.AskAfterAlways, Autopilot: true}); err != nil {
		t.Fatal(err)
	}
	request := factory.JobActionRequest{RequestID: "activity-stop", JobID: h.job.ID, IfVersion: h.job.Version, Action: factory.ActionCancel}
	stopped, err := h.service.Action(ctx, h.coordinator, request)
	if err != nil || stopped.Status != factory.JobCancelled {
		t.Fatalf("queued stop: %+v, %v", stopped, err)
	}
	if replay, err := h.service.Action(ctx, h.coordinator, request); err != nil || replay.Version != stopped.Version {
		t.Fatalf("stop replay: %+v, %v", replay, err)
	}
	for pass := 0; pass < 2; pass++ {
		h.service.PilotOnce(ctx)
		activity, err := h.service.WorkItemActivity(ctx, h.coordinator, h.project.ID, "work-test")
		if err != nil || !activity.Held || len(activity.Jobs) != 1 || activity.Jobs[0].Status != factory.JobCancelled {
			t.Fatalf("explicit stop was restarted, pass %d: %+v, %v", pass, activity, err)
		}
		assertActivityBoardStatus(t, h, "Stopped", "")
		if pass == 0 {
			if err := h.service.Close(); err != nil {
				t.Fatal(err)
			}
			h.service, err = factory.Open(h.dataDir, factory.Options{})
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = h.service.Close() })
		}
	}
}

func TestStopOfLostContactPreservesUncertaintyUntilEvidence(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	h := setupJob(t)
	a := startActivityAttempt(t, h, "lost-contact")
	job, err := h.service.Report(ctx, h.implementer, factory.WorkerReportRequest{RequestID: "lost-contact-unknown", AttemptID: a.AttemptID, DispatchID: a.DispatchID, Kind: factory.ReportUnknown, ObservedAt: time.Now(), NativeRunID: "lost-contact", Outcome: factory.OutcomeUnknown, TerminalReceipt: "wrapper lost contact; native execution may continue"})
	if err != nil {
		t.Fatal(err)
	}
	job, err = h.service.Action(ctx, h.coordinator, factory.JobActionRequest{RequestID: "lost-contact-stop", JobID: job.ID, IfVersion: job.Version, Action: factory.ActionCancel})
	if err != nil || job.Status != factory.JobCancellationUnknown || !job.Attempts[0].CancellationRequested || job.Attempts[0].Status != factory.AttemptUnknown {
		t.Fatalf("stop falsely confirmed termination: %+v, %v", job, err)
	}
	assertActivityBoardStatus(t, h, "Stop outcome unknown", factory.CardDecisionFix)
	history, err := h.service.WorkItemActivity(ctx, h.coordinator, h.project.ID, "work-test")
	if err != nil || !history.Held || history.Jobs[0].Attempts[0].Status != factory.AttemptUnknown || history.Jobs[0].Attempts[0].TerminalReceipt != "" {
		t.Fatalf("lost-contact facts were erased: %+v, %v", history, err)
	}
	retained, err := h.service.GetJob(ctx, h.coordinator, job.ID)
	if err != nil || !strings.Contains(retained.Attempts[0].TerminalReceipt, "may continue") {
		t.Fatalf("lost-contact receipt was erased: %+v, %v", retained, err)
	}
	resolved, err := h.service.Action(ctx, h.admin, factory.JobActionRequest{RequestID: "lost-contact-resolved", JobID: job.ID, IfVersion: job.Version, Action: factory.ActionResolve, AttemptID: &a.AttemptID, Resolution: factory.ResolutionNotStarted, Evidence: "Native process registry and receipt confirm execution never started."})
	if err != nil || resolved.Status != factory.JobCancelled || len(resolved.Attempts) != 1 {
		t.Fatalf("resolution restarted a stopped run: %+v, %v", resolved, err)
	}
	assertActivityBoardStatus(t, h, "Stopped", "")
}

func TestWorkItemActivityRetainsRefinementAndExpectationDecisions(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	t.Run("refinements", func(t *testing.T) {
		command, _ := fakeRefiner(t, "")
		h := setupRefine(t, command)
		first := h.start(t, factory.StartRefinementRequest{RequestID: "first"})
		if _, err := h.service.RejectRefinement(ctx, h.admin, factory.RejectRefinementRequest{RequestID: "reject", RefinementID: first.ID}); err != nil {
			t.Fatal(err)
		}
		second := h.start(t, factory.StartRefinementRequest{RequestID: "second"})
		activity, err := h.service.WorkItemActivity(ctx, h.admin, h.project.ID, "work-test")
		if err != nil || len(activity.Refinements) != 2 || activity.Refinements[0].ID != second.ID || activity.Refinements[1].Status != factory.RefinementRejected || activity.Refinements[1].Proposal == nil || activity.Refinements[1].DecidedBy == nil {
			t.Fatalf("refinement history lost its earlier proposal/decision: %+v, %v", activity, err)
		}
	})
	t.Run("expectations", func(t *testing.T) {
		h := setupExpectations(t, fakeAuthor(t, map[string]string{"tests/expected.txt": "expected"}), nil)
		first := h.write(t, "first")
		if _, err := h.service.RejectExpectations(ctx, h.admin, factory.DecideExpectationsRequest{RequestID: "reject", ExpectationID: first.ID}); err != nil {
			t.Fatal(err)
		}
		second := h.write(t, "second")
		activity, err := h.service.WorkItemActivity(ctx, h.admin, h.project.ID, "work-test")
		if err != nil || len(activity.Expectations) != 2 || activity.Expectations[0].ID != second.ID || activity.Expectations[1].Status != factory.ExpectationRejected || activity.Expectations[1].Patch == "" || activity.Expectations[1].DecidedBy == nil {
			t.Fatalf("expectation history lost its earlier tests/decision: %+v, %v", activity, err)
		}
	})
}
