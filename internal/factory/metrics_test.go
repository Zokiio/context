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

func TestMetricsMeasureOutcomesDecisionsAndCost(t *testing.T) {
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
	if err := os.WriteFile(filepath.Join(assignment.WorkspacePath, "change.txt"), []byte("done\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runGit(t, assignment.WorkspacePath, "add", "change.txt")
	runGit(t, assignment.WorkspacePath, "commit", "-qm", "change")
	candidate := strings.TrimSpace(runGit(t, assignment.WorkspacePath, "rev-parse", "HEAD"))
	if _, err := harness.service.Report(ctx, harness.implementer, factory.WorkerReportRequest{RequestID: "terminal", AttemptID: assignment.AttemptID, DispatchID: assignment.DispatchID, Kind: factory.ReportTerminal, ObservedAt: time.Now(), NativeRunID: "native", Outcome: factory.OutcomeSucceeded, CandidateRevision: candidate, TerminalReceipt: "receipt", Artifacts: []factory.ArtifactInput{{Name: "agent-result.json", MediaType: "application/json", Content: `{"harness":"claude","costUsd":0.12,"turns":5}`}}}); err != nil {
		t.Fatal(err)
	}
	job := runChecks(t, harness, "candidate", factory.CheckPassed, false)
	job, err = harness.service.Review(ctx, harness.reviewer, factory.ReviewRequest{RequestID: "review", JobID: job.ID, IfVersion: job.Version, CandidateRevision: candidate, RequirementFingerprint: job.Admission.TicketSHA256, Decision: factory.ReviewApproved, Summary: "ok"})
	if err != nil {
		t.Fatal(err)
	}
	job, err = harness.service.Accept(ctx, harness.reviewer, factory.AcceptRequest{RequestID: "accept", JobID: job.ID, IfVersion: job.Version, CandidateRevision: candidate, RequirementFingerprint: job.Admission.TicketSHA256, Summary: "ok"})
	if err != nil {
		t.Fatal(err)
	}
	mergeAndVerify(t, harness, job, "merge", "")

	metrics, err := harness.service.Metrics(ctx, harness.reviewer, harness.project.ID)
	if err != nil {
		t.Fatal(err)
	}
	if metrics.Jobs != 1 || metrics.AcceptedJobs != 1 || metrics.MergedJobs != 1 || metrics.FirstPassRate != 1 || metrics.Repairs != 0 {
		t.Fatalf("outcomes = %+v", metrics)
	}
	if metrics.MergeChecksPassed != 1 || metrics.MergeChecksFailed != 0 || metrics.ImplementationCostUSD != 0.12 || metrics.TotalCostUSD != 0.12 || metrics.CostPerMergedJobUSD != 0.12 {
		t.Fatalf("verification and cost = %+v", metrics)
	}
	// Admission, review, acceptance, and the merge request were each a person's decision.
	if metrics.HumanDecisions != 4 || metrics.HumanDecisionsPerMerge != 4 {
		t.Fatalf("human decisions = %d (%.1f per merge)", metrics.HumanDecisions, metrics.HumanDecisionsPerMerge)
	}
	if metrics.MedianAdmitToAcceptSeconds <= 0 || metrics.MedianAcceptToMergeSeconds < 0 {
		t.Fatalf("lead times = %+v", metrics)
	}
}
