package factory_test

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/Zokiio/context/internal/factory"
)

func TestDisputeTerminalProtocolPreservesExecutionAuthority(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name      string
		candidate bool
		unknown   bool
		stop      bool
		wantError factory.ErrorCode
	}{
		{name: "candidate is incompatible with dispute", candidate: true, wantError: factory.CodeInvalid},
		{name: "lost contact needs reconciliation", unknown: true, wantError: factory.CodeUncertain},
		{name: "uncertain cancellation needs reconciliation", unknown: true, stop: true, wantError: factory.CodeUncertain},
		{name: "confirmed exit honors requested stop", stop: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			h := setupJob(t)
			assignment := startActivityAttempt(t, h, "dispute-protocol")
			if test.unknown {
				_, err := h.service.Report(ctx, h.implementer, factory.WorkerReportRequest{
					RequestID: "lost-contact", AttemptID: assignment.AttemptID, DispatchID: assignment.DispatchID,
					Kind: factory.ReportUnknown, ObservedAt: time.Now(), NativeRunID: "dispute-protocol",
					Outcome: factory.OutcomeUnknown, TerminalReceipt: "External execution may still be running.",
				})
				if err != nil {
					t.Fatal(err)
				}
			}
			before, err := h.service.GetJob(ctx, h.coordinator, h.job.ID)
			if err != nil {
				t.Fatal(err)
			}
			if test.stop {
				before, err = h.service.Action(ctx, h.coordinator, factory.JobActionRequest{
					RequestID: "stop", JobID: before.ID, IfVersion: before.Version, Action: factory.ActionCancel,
				})
				if err != nil {
					t.Fatal(err)
				}
			}
			dispute := disputeReport()
			request := factory.WorkerReportRequest{
				RequestID: "disputed-terminal", AttemptID: assignment.AttemptID, DispatchID: assignment.DispatchID,
				Kind: factory.ReportTerminal, ObservedAt: time.Now(), NativeRunID: "dispute-protocol",
				Outcome: factory.OutcomeDisputed, TerminalReceipt: "Process exited and requested independent review.", Dispute: &dispute,
				Artifacts: []factory.ArtifactInput{{Name: "dispute-evidence.txt", MediaType: "text/plain", Content: "Expected two, observed three"}},
			}
			if test.candidate {
				request.CandidateRevision = h.job.Admission.BaseRevision
			}
			_, err = h.service.Report(ctx, h.implementer, request)
			if (test.wantError == "" && err != nil) || (test.wantError != "" && factory.ErrorCodeOf(err) != test.wantError) {
				t.Fatalf("disputed report error = %v; want %q", err, test.wantError)
			}
			after, err := h.service.GetJob(ctx, h.coordinator, h.job.ID)
			if err != nil {
				t.Fatal(err)
			}
			if test.wantError != "" {
				if !reflect.DeepEqual(before, after) {
					t.Fatalf("rejected dispute changed execution or retained artifacts:\nbefore %+v\nafter %+v", before, after)
				}
			} else {
				attempt := after.Attempts[0]
				if after.Status != factory.JobCancelled || len(after.Attempts) != 1 || after.CandidateRevision != "" || attempt.Status != factory.AttemptCancelled || attempt.Outcome != factory.OutcomeCancelled || attempt.TerminalReceipt != request.TerminalReceipt || len(attempt.Artifacts) != 1 {
					t.Fatalf("dispute overrode Stop or lost terminal evidence: %+v", after)
				}
				activity, err := h.service.WorkItemActivity(ctx, h.coordinator, h.project.ID, "work-test")
				if err != nil || !activity.Held {
					t.Fatalf("terminal dispute cleared stop hold: %+v, %v", activity, err)
				}
			}
			disputes, err := h.service.ListDisputes(ctx, h.coordinator, h.project.ID, "work-test")
			if err != nil || len(disputes) != 0 {
				t.Fatalf("terminal report created an unauthorized dispute: %+v, %v", disputes, err)
			}
		})
	}
}
