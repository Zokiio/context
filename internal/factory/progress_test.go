package factory_test

import (
	"context"
	"testing"
	"time"

	"github.com/Zokiio/context/internal/factory"
)

func TestHeartbeatProgressBecomesOneEventPerChange(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	harness := setupJob(t)
	assignment, err := harness.service.Claim(ctx, harness.implementer, factory.WorkerClaimRequest{RequestID: "claim", Stages: []factory.Stage{factory.StageImplementation}, ProjectID: &harness.project.ID})
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range []factory.WorkerReportKind{factory.ReportStartRegistered, factory.ReportStarted} {
		if _, err := harness.service.Report(ctx, harness.implementer, factory.WorkerReportRequest{RequestID: string(kind), AttemptID: assignment.AttemptID, DispatchID: assignment.DispatchID, Kind: kind, ObservedAt: time.Now(), NativeRunID: "native"}); err != nil {
			t.Fatal(err)
		}
	}
	for i, note := range []string{"checkpoint 1 committed", "checkpoint 1 committed", "", "checkpoint 2 committed"} {
		if _, err := harness.service.Heartbeat(ctx, harness.implementer, factory.HeartbeatRequest{RequestID: "beat-" + string(rune('a'+i)), AttemptID: assignment.AttemptID, DispatchID: assignment.DispatchID, Progress: note}); err != nil {
			t.Fatal(err)
		}
	}
	job, err := harness.service.GetJob(ctx, harness.coordinator, harness.job.ID)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, event := range job.Events {
		if event.Kind == "worker_progress" {
			count++
		}
	}
	if count != 2 || !hasEvent(job, "worker_progress", "checkpoint 2 committed") {
		t.Fatalf("progress events = %d: %+v", count, job.Events)
	}
}
