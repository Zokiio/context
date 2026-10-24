package factory_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/Zokiio/context/internal/factory"
)

func TestTheRunGraphFollowsAnItemThroughItsSteps(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	h := setupExpectations(t, fakeAuthor(t, map[string]string{"tests/plan_test.txt": "expects the change\n"}), func(policy *factory.ProjectPolicy) { policy.AllowCoordinatorReview = true })
	run := func() factory.ItemRun {
		t.Helper()
		run, err := h.service.WorkItemRun(ctx, h.admin, h.project.ID, "work-test")
		if err != nil {
			t.Fatal(err)
		}
		return run
	}
	statuses := func(run factory.ItemRun) string {
		parts := []string{}
		for _, step := range run.Steps {
			parts = append(parts, step.Key+"="+step.Status)
		}
		return strings.Join(parts, " ")
	}
	if got := statuses(run()); got != "sharpen=skipped tests=next agree=next build=next checks=next approve=next merge=next repair=idle" {
		t.Fatalf("before anything = %s", got)
	}
	started, err := h.service.ApprovePlan(ctx, h.admin, factory.ApprovePlanRequest{RequestID: "agree", ProjectID: h.project.ID, WorkItemID: "work-test"})
	if err != nil {
		t.Fatal(err)
	}
	expectation := h.waitExpectation(t, started.Expectation.ID)
	written := run()
	if got := statuses(written); !strings.HasPrefix(got, "sharpen=skipped tests=done agree=needs_you build=next") {
		t.Fatalf("with tests written = %s", got)
	}
	if tests := written.Steps[1]; tests.Model != "fake-author" || tests.Turns != 3 || tests.CostUSD != 0.02 || tests.Summary != "0 of 1 tested" {
		t.Fatalf("tests step = %+v", tests)
	}
	built, err := h.service.ApprovePlan(ctx, h.admin, factory.ApprovePlanRequest{RequestID: "tests", ProjectID: h.project.ID, WorkItemID: "work-test", ExpectationID: expectation.ID})
	if err != nil {
		t.Fatal(err)
	}
	h.job = *built.Job
	if got := statuses(run()); !strings.Contains(got, "agree=done build=running checks=next") {
		t.Fatalf("queued = %s", got)
	}
	job, _ := passVerificationForTest(t, h.jobHarness, "run")
	if got := statuses(run()); !strings.Contains(got, "build=done checks=done approve=needs_you merge=next repair=idle") {
		t.Fatalf("checked = %s", got)
	}
	accepted, err := h.service.Approve(ctx, h.admin, approveRequest(job, "approve", false))
	if err != nil {
		t.Fatal(err)
	}
	approved := run()
	if approve := approved.Steps[5]; approve.Status != factory.RunDone || approve.By != "admin" {
		t.Fatalf("approve step = %+v", approve)
	}
	if got := statuses(approved); !strings.HasSuffix(got, "merge=needs_you repair=idle") {
		t.Fatalf("approved = %s", got)
	}
	mergeAndVerify(t, h.jobHarness, accepted, "merge", "")
	if merge := run().Steps[6]; merge.Status != factory.RunDone || !strings.HasPrefix(merge.Title, "Merge into ") {
		t.Fatalf("merge step = %+v", merge)
	}
}

func TestRunGraphRetainsStoppedExecution(t *testing.T) {
	t.Parallel()
	for _, phase := range []string{"queued", "building", "checked"} {
		t.Run(phase, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			h := setupJob(t)
			job := h.job
			var assignment factory.Assignment
			if phase == "building" {
				assignment = startActivityAttempt(t, h, "stopped-graph")
				var err error
				job, err = h.service.GetJob(ctx, h.coordinator, job.ID)
				if err != nil {
					t.Fatal(err)
				}
			} else if phase == "checked" {
				job, _ = passVerificationForTest(t, h, "stopped-graph")
			}
			job, err := h.service.Action(ctx, h.coordinator, factory.JobActionRequest{RequestID: "stop-graph", JobID: job.ID, IfVersion: job.Version, Action: factory.ActionCancel})
			if err != nil {
				t.Fatal(err)
			}
			if phase == "building" {
				pending, err := h.service.WorkItemRun(ctx, h.coordinator, h.project.ID, "work-test")
				if err != nil || pending.JobID != job.ID || pending.Steps[3].Status != factory.RunRunning || !strings.Contains(pending.Steps[3].Summary, "Stop requested") {
					t.Fatalf("stop request lost live execution: %+v, %v", pending, err)
				}
				job, err = h.service.Report(ctx, h.implementer, factory.WorkerReportRequest{RequestID: "stopped-graph-terminal", AttemptID: assignment.AttemptID, DispatchID: assignment.DispatchID, Kind: factory.ReportTerminal, ObservedAt: time.Now(), NativeRunID: "stopped-graph", Outcome: factory.OutcomeCancelled, TerminalReceipt: "process exited after stop"})
				if err != nil {
					t.Fatal(err)
				}
			}
			if job.Status != factory.JobCancelled {
				t.Fatalf("stop was not confirmed: %+v", job)
			}
			run, err := h.service.WorkItemRun(ctx, h.coordinator, h.project.ID, "work-test")
			if err != nil {
				t.Fatal(err)
			}
			if run.JobID != job.ID || run.Steps[2].Status != factory.RunDone {
				t.Fatalf("stopped run lost its admission and agreement: %+v", run)
			}
			buildStatus, checkStatus := factory.RunSkipped, factory.RunSkipped
			if phase == "building" {
				buildStatus = factory.RunFailed
			} else if phase == "checked" {
				buildStatus, checkStatus = factory.RunDone, factory.RunDone
			}
			if run.Steps[3].Status != buildStatus || run.Steps[4].Status != checkStatus || run.Steps[5].Status != factory.RunSkipped || run.Steps[6].Status != factory.RunSkipped {
				t.Fatalf("stopped run rewrote completed or skipped stages: %+v", run.Steps)
			}
			for _, step := range run.Steps {
				if step.Status == factory.RunRunning || step.Status == factory.RunNeedsYou {
					t.Fatalf("confirmed stop still advertises active work: %+v", step)
				}
			}
		})
	}
}
