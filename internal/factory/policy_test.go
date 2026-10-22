package factory_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Zokiio/context/internal/factory"
)

func TestProtectedPathChangesFailImplementationBeforeVerification(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	harness := setupJobWith(t, []string{"go-test"}, 2, nil, func(policy *factory.ProjectPolicy) {
		policy.ProtectedPaths = []string{"tests/", "**/*_holdout.go"}
	})
	if got := harness.job.Admission.ProtectedPaths; strings.Join(got, ",") != "**/*_holdout.go,tests/**" {
		t.Fatalf("admission protected paths = %v", got)
	}
	job, candidate := implementFilesForTest(t, harness, "protected", map[string]string{"tests/expect.txt": "weakened\n", "src.go": "package src\n"})
	if job.Status != factory.JobFailed || job.CandidateRevision != "" {
		t.Fatalf("protected change was not rejected: status=%s candidate=%q", job.Status, job.CandidateRevision)
	}
	if !hasEvent(job, "protected_paths_changed", "tests/expect.txt") {
		t.Fatalf("missing protected path event: %+v", job.Events)
	}
	for _, attempt := range job.Attempts {
		if attempt.Stage == factory.StageVerification {
			t.Fatalf("verification was queued for a rejected candidate: %+v", attempt)
		}
		if attempt.Stage == factory.StageImplementation && attempt.CandidateRevision != candidate {
			t.Fatalf("rejected candidate was not retained on the attempt: %+v", attempt)
		}
	}
	if !hasAction(job, factory.ActionRepair) {
		t.Fatalf("repair is not offered after a protected path rejection: %v", job.AvailableActions)
	}
	repaired, err := harness.service.Action(ctx, harness.coordinator, factory.JobActionRequest{RequestID: "repair", JobID: job.ID, IfVersion: job.Version, Action: factory.ActionRepair})
	if err != nil {
		t.Fatal(err)
	}
	latest := repaired.Attempts[len(repaired.Attempts)-1]
	if latest.Stage != factory.StageImplementation || latest.InputRevision != harness.job.Admission.BaseRevision {
		t.Fatalf("repair did not restart from the admitted base: %+v", latest)
	}
}

func TestUnprotectedChangesProceedToVerification(t *testing.T) {
	t.Parallel()
	harness := setupJobWith(t, []string{"go-test"}, 2, nil, func(policy *factory.ProjectPolicy) {
		policy.ProtectedPaths = []string{"tests/**", "*_holdout.go"}
	})
	job, candidate := implementFilesForTest(t, harness, "unprotected", map[string]string{"src/tests_helper.go": "package src\n", "src/case_holdout.go": "package src\n"})
	if job.Status != factory.JobVerifying || job.CandidateRevision != candidate {
		t.Fatalf("unprotected change did not reach verification: status=%s", job.Status)
	}
}

func TestProtectedPathPatternsAreValidated(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	harness := setupJob(t)
	for _, pattern := range []string{"/abs/path", "../outside", "tests//x", "[bad"} {
		_, err := harness.service.RegisterProject(ctx, harness.admin, factory.RegisterProjectRequest{RequestID: "invalid-" + pattern, BundlePath: harness.bundle, RepositoryPath: harness.repository, AllowedSourceRoots: []string{harness.repository}, Policy: factory.ProjectPolicy{RequiredChecks: []string{"go-test"}, AllowedExecutors: []string{"fixture-agent"}, MaxRepairAttempts: 1, MaxJobDurationSeconds: 60, Fixture: true, ProtectedPaths: []string{pattern}}})
		var fault *factory.Error
		if !errors.As(err, &fault) || fault.Code != factory.CodeInvalid {
			t.Fatalf("pattern %q was accepted: %v", pattern, err)
		}
	}
}

func TestCoordinatorReviewRequiresAdmittedPolicy(t *testing.T) {
	t.Parallel()
	for _, allowed := range []bool{false, true} {
		harness := setupJobWith(t, []string{"go-test"}, 2, nil, func(policy *factory.ProjectPolicy) {
			policy.AllowCoordinatorReview = allowed
		})
		ctx := context.Background()
		if _, err := harness.service.GrantProject(ctx, harness.admin, factory.GrantProjectRequest{RequestID: "coordinator-review", ProjectID: harness.project.ID, UserID: harness.coordinator.UserID, Capabilities: []factory.Capability{factory.CapabilityRead, factory.CapabilityCoordinate, factory.CapabilityReview, factory.CapabilityAccept}}); err != nil {
			t.Fatal(err)
		}
		job, candidate := passVerificationForTest(t, harness, "coord")
		reviewed, err := harness.service.Review(ctx, harness.coordinator, factory.ReviewRequest{RequestID: "self-review", JobID: job.ID, IfVersion: job.Version, CandidateRevision: candidate, RequirementFingerprint: job.Admission.TicketSHA256, Decision: factory.ReviewApproved, Summary: "Solo review."})
		var fault *factory.Error
		if !allowed {
			if !errors.As(err, &fault) || fault.Code != factory.CodeForbidden {
				t.Fatalf("coordinator reviewed without policy: %v", err)
			}
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		accepted, err := harness.service.Accept(ctx, harness.coordinator, factory.AcceptRequest{RequestID: "self-accept", JobID: reviewed.ID, IfVersion: reviewed.Version, CandidateRevision: candidate, RequirementFingerprint: reviewed.Admission.TicketSHA256, Summary: "Solo acceptance."})
		if err != nil {
			t.Fatal(err)
		}
		if accepted.Status != factory.JobAccepted {
			t.Fatalf("solo acceptance status = %s", accepted.Status)
		}
	}
}

func implementFilesForTest(t *testing.T, harness jobHarness, prefix string, files map[string]string) (factory.Job, string) {
	t.Helper()
	ctx := context.Background()
	assignment, err := harness.service.Claim(ctx, harness.implementer, factory.WorkerClaimRequest{RequestID: prefix + "-claim", Stages: []factory.Stage{factory.StageImplementation}, ProjectID: &harness.project.ID})
	if err != nil {
		t.Fatal(err)
	}
	if len(harness.job.Admission.ProtectedPaths) > 0 && len(assignment.ProtectedPaths) == 0 {
		t.Fatalf("assignment omitted protected paths: %+v", assignment)
	}
	for _, kind := range []factory.WorkerReportKind{factory.ReportStartRegistered, factory.ReportStarted} {
		if _, err := harness.service.Report(ctx, harness.implementer, factory.WorkerReportRequest{RequestID: prefix + "-" + string(kind), AttemptID: assignment.AttemptID, DispatchID: assignment.DispatchID, Kind: kind, ObservedAt: time.Now(), NativeRunID: prefix + "-native"}); err != nil {
			t.Fatal(err)
		}
	}
	for name, content := range files {
		path := filepath.Join(assignment.WorkspacePath, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	runGit(t, assignment.WorkspacePath, "add", "-A")
	runGit(t, assignment.WorkspacePath, "commit", "-qm", prefix)
	candidate := strings.TrimSpace(runGit(t, assignment.WorkspacePath, "rev-parse", "HEAD"))
	job, err := harness.service.Report(ctx, harness.implementer, factory.WorkerReportRequest{RequestID: prefix + "-terminal", AttemptID: assignment.AttemptID, DispatchID: assignment.DispatchID, Kind: factory.ReportTerminal, ObservedAt: time.Now(), NativeRunID: prefix + "-native", Outcome: factory.OutcomeSucceeded, CandidateRevision: candidate, TerminalReceipt: "retained implementation receipt"})
	if err != nil {
		t.Fatal(err)
	}
	return job, candidate
}

func passVerificationForTest(t *testing.T, harness jobHarness, prefix string) (factory.Job, string) {
	t.Helper()
	_, candidate := completeImplementationForTest(t, harness, prefix)
	verification := startVerificationForTest(t, harness, prefix)
	job, err := harness.service.Report(context.Background(), harness.verifier, factory.WorkerReportRequest{RequestID: prefix + "-verify-terminal", AttemptID: verification.AttemptID, DispatchID: verification.DispatchID, Kind: factory.ReportTerminal, ObservedAt: time.Now(), NativeRunID: prefix + "-verify-native", Outcome: factory.OutcomeSucceeded, CheckStatus: factory.CheckPassed, TerminalReceipt: "check terminal"})
	if err != nil {
		t.Fatal(err)
	}
	if job.Status != factory.JobReadyForReview {
		t.Fatalf("job after verification = %s", job.Status)
	}
	return job, candidate
}

func hasEvent(job factory.Job, kind, contains string) bool {
	for _, event := range job.Events {
		if event.Kind == kind && strings.Contains(string(event.Data), contains) {
			return true
		}
	}
	return false
}

func hasAction(job factory.Job, wanted factory.ActionKind) bool {
	for _, action := range job.AvailableActions {
		if action == wanted {
			return true
		}
	}
	return false
}
