package factory_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Zokiio/context/internal/factory"
)

func TestBootstrapAuthenticationPersistsAcrossRestart(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dataDir := t.TempDir()

	service, err := factory.Open(dataDir, factory.Options{})
	if err != nil {
		t.Fatal(err)
	}
	bootstrap, err := service.Bootstrap(ctx, factory.BootstrapRequest{Username: "admin"})
	if err != nil {
		t.Fatal(err)
	}
	if bootstrap.Token == "" || !bootstrap.User.InstallationAdmin {
		t.Fatalf("unexpected bootstrap: %+v", bootstrap)
	}
	principal, err := service.Authenticate(ctx, bootstrap.Token)
	if err != nil {
		t.Fatal(err)
	}
	if principal.UserID != bootstrap.User.ID || principal.Username != "admin" || !principal.InstallationAdmin {
		t.Fatalf("unexpected principal: %+v", principal)
	}
	if _, err := service.Bootstrap(ctx, factory.BootstrapRequest{Username: "second"}); factory.ErrorCodeOf(err) != factory.CodeConflict {
		t.Fatalf("second bootstrap error = %v, want conflict", err)
	}
	if err := service.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := factory.Open(dataDir, factory.Options{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = reopened.Close() })
	principal, err = reopened.Authenticate(ctx, bootstrap.Token)
	if err != nil {
		t.Fatal(err)
	}
	if principal.UserID != bootstrap.User.ID {
		t.Fatalf("reopened principal = %+v", principal)
	}
}

func TestProjectPolicyAndAccessAreServiceEnforced(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	service, err := factory.Open(t.TempDir(), factory.Options{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = service.Close() })
	bootstrap, err := service.Bootstrap(ctx, factory.BootstrapRequest{Username: "admin"})
	if err != nil {
		t.Fatal(err)
	}
	admin, err := service.Authenticate(ctx, bootstrap.Token)
	if err != nil {
		t.Fatal(err)
	}
	reader, err := service.CreateUser(ctx, admin, factory.CreateUserRequest{RequestID: "user-reader", Username: "reader", Kind: factory.UserKindHuman})
	if err != nil {
		t.Fatal(err)
	}
	readerToken, err := service.IssueToken(ctx, admin, factory.IssueTokenRequest{RequestID: "token-reader", UserID: reader.ID, Label: "test"})
	if err != nil {
		t.Fatal(err)
	}
	readerPrincipal, err := service.Authenticate(ctx, readerToken.Token)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.RegisterExecutor(ctx, admin, factory.RegisterExecutorRequest{RequestID: "executor", Key: "fixture-agent", DisplayName: "Fixture agent", Command: []string{"fixture-agent", "run"}, Fixture: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.RegisterCheck(ctx, admin, factory.RegisterCheckRequest{RequestID: "check", Key: "go-test", DisplayName: "Go test", Command: []string{"go", "test", "./..."}, Environment: "local", Fixture: true}); err != nil {
		t.Fatal(err)
	}
	bundle, repository := testProject(t)
	project, err := service.RegisterProject(ctx, admin, factory.RegisterProjectRequest{
		RequestID: "project", BundlePath: bundle, RepositoryPath: repository,
		ReaderLimits: factory.ReaderLimits{MaxFiles: 200, MaxBytes: 2 << 20},
		Policy:       factory.ProjectPolicy{RequiredChecks: []string{"go-test"}, AllowedExecutors: []string{"fixture-agent"}, MaxRepairAttempts: 2, MaxJobDurationSeconds: 3600, Fixture: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if project.ID != "project-test" || !project.Policy.Fixture {
		t.Fatalf("unexpected project: %+v", project)
	}
	if projects, err := service.ListProjects(ctx, readerPrincipal); err != nil || len(projects) != 0 {
		t.Fatalf("projects before grant = %+v, %v", projects, err)
	}
	grant, err := service.GrantProject(ctx, admin, factory.GrantProjectRequest{RequestID: "grant", ProjectID: project.ID, UserID: reader.ID, Capabilities: []factory.Capability{factory.CapabilityRead}})
	if err != nil {
		t.Fatal(err)
	}
	if len(grant.Capabilities) != 1 || grant.Capabilities[0] != factory.CapabilityRead {
		t.Fatalf("unexpected grant: %+v", grant)
	}
	projects, err := service.ListProjects(ctx, readerPrincipal)
	if err != nil || len(projects) != 1 || projects[0].ID != project.ID {
		t.Fatalf("projects after grant = %+v, %v", projects, err)
	}
	capabilities, err := service.Capabilities(ctx, readerPrincipal, project.ID)
	if err != nil || len(capabilities) != 1 || capabilities[0] != factory.CapabilityRead {
		t.Fatalf("capabilities = %+v, %v", capabilities, err)
	}
	if _, err := service.Submit(ctx, readerPrincipal, factory.SubmitRequest{RequestID: "denied", ProjectID: project.ID, WorkItemID: "work-test", BaseRevision: "HEAD", ExecutorKey: "fixture-agent", Deadline: time.Now().Add(time.Hour)}); factory.ErrorCodeOf(err) != factory.CodeForbidden {
		t.Fatalf("submit error = %v, want forbidden", err)
	}
}

func TestSubmitCapturesReadyWorkAndKeepsOneClaimAcrossRestart(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dataDir := t.TempDir()
	service, err := factory.Open(dataDir, factory.Options{})
	if err != nil {
		t.Fatal(err)
	}
	bootstrap, err := service.Bootstrap(ctx, factory.BootstrapRequest{Username: "admin"})
	if err != nil {
		t.Fatal(err)
	}
	admin, err := service.Authenticate(ctx, bootstrap.Token)
	if err != nil {
		t.Fatal(err)
	}
	coordinator, err := service.CreateUser(ctx, admin, factory.CreateUserRequest{RequestID: "coordinator", Username: "coordinator", Kind: factory.UserKindHuman})
	if err != nil {
		t.Fatal(err)
	}
	coordinatorToken, err := service.IssueToken(ctx, admin, factory.IssueTokenRequest{RequestID: "coordinator-token", UserID: coordinator.ID, Label: "test"})
	if err != nil {
		t.Fatal(err)
	}
	coordinatorPrincipal, err := service.Authenticate(ctx, coordinatorToken.Token)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.RegisterExecutor(ctx, admin, factory.RegisterExecutorRequest{RequestID: "executor", Key: "fixture-agent", DisplayName: "Fixture agent", Command: []string{"fixture-agent", "run"}, Fixture: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.RegisterCheck(ctx, admin, factory.RegisterCheckRequest{RequestID: "check", Key: "go-test", DisplayName: "Go test", Command: []string{"go", "test", "./..."}, Environment: "local", Fixture: true}); err != nil {
		t.Fatal(err)
	}
	bundle, repository := readyTestProject(t)
	project, err := service.RegisterProject(ctx, admin, factory.RegisterProjectRequest{
		RequestID: "project", BundlePath: bundle, RepositoryPath: repository,
		Policy: factory.ProjectPolicy{RequiredChecks: []string{"go-test"}, AllowedExecutors: []string{"fixture-agent"}, MaxRepairAttempts: 1, MaxJobDurationSeconds: 3600, Fixture: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.GrantProject(ctx, admin, factory.GrantProjectRequest{RequestID: "grant", ProjectID: project.ID, UserID: coordinator.ID, Capabilities: []factory.Capability{factory.CapabilityRead, factory.CapabilityCoordinate}}); err != nil {
		t.Fatal(err)
	}
	base := strings.TrimSpace(runGit(t, repository, "rev-parse", "HEAD"))
	request := factory.SubmitRequest{RequestID: "submit-one", ProjectID: project.ID, WorkItemID: "work-test", BaseRevision: base, ExecutorKey: "fixture-agent", RepairLimit: 1, Deadline: time.Now().Add(30 * time.Minute)}
	job, err := service.Submit(ctx, coordinatorPrincipal, request)
	if err != nil {
		t.Fatal(err)
	}
	if job.Status != factory.JobQueued || job.Admission.BaseRevision != base || len(job.Admission.RequiredChecks) != 1 || !job.Admission.ContextComplete {
		t.Fatalf("unexpected job: %+v", job)
	}
	ticket, err := os.ReadFile(filepath.Join(bundle, "work.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(ticket), "## Acceptance\n\nNone") {
		t.Fatalf("ticket did not receive prepared Acceptance section:\n%s", ticket)
	}
	replayed, err := service.Submit(ctx, coordinatorPrincipal, request)
	if err != nil || replayed.ID != job.ID {
		t.Fatalf("replayed submit = %+v, %v", replayed, err)
	}
	request.RequestID = "submit-two"
	if _, err := service.Submit(ctx, coordinatorPrincipal, request); factory.ErrorCodeOf(err) != factory.CodeConflict {
		t.Fatalf("competing submit error = %v, want conflict", err)
	}
	if err := service.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := factory.Open(dataDir, factory.Options{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = reopened.Close() })
	restored, err := reopened.GetJob(ctx, coordinatorPrincipal, job.ID)
	if err != nil || restored.ID != job.ID || restored.Admission.TicketSHA256 == "" {
		t.Fatalf("restored job = %+v, %v", restored, err)
	}
}

func TestWorkerAssignmentsSeparateImplementationVerificationAndReview(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	harness := setupJob(t)

	assignment, err := harness.service.Claim(ctx, harness.implementer, factory.WorkerClaimRequest{RequestID: "claim-implementation", Stages: []factory.Stage{factory.StageImplementation}, ProjectID: &harness.project.ID})
	if err != nil {
		t.Fatal(err)
	}
	if assignment.Stage != factory.StageImplementation || assignment.WorkspacePath == "" || assignment.BaseRevision != harness.job.Admission.BaseRevision || !assignment.Fixture {
		t.Fatalf("unexpected implementation assignment: %+v", assignment)
	}
	if got := strings.TrimSpace(runGit(t, assignment.WorkspacePath, "rev-parse", "HEAD")); got != assignment.BaseRevision {
		t.Fatalf("implementation workspace HEAD = %s, want %s", got, assignment.BaseRevision)
	}
	registration := factory.WorkerReportRequest{RequestID: "registered", AttemptID: assignment.AttemptID, DispatchID: assignment.DispatchID, Kind: factory.ReportStartRegistered, ObservedAt: time.Now(), NativeRunID: "native-1"}
	if _, err := harness.service.Report(ctx, harness.implementer, registration); err != nil {
		t.Fatal(err)
	}
	if _, err := harness.service.Report(ctx, harness.implementer, registration); err != nil {
		t.Fatal(err)
	}
	if _, err := harness.service.Report(ctx, harness.implementer, factory.WorkerReportRequest{RequestID: "started", AttemptID: assignment.AttemptID, DispatchID: assignment.DispatchID, Kind: factory.ReportStarted, ObservedAt: time.Now(), NativeRunID: "native-1"}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(assignment.WorkspacePath, "candidate.txt"), []byte("candidate\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runGit(t, assignment.WorkspacePath, "add", "candidate.txt")
	runGit(t, assignment.WorkspacePath, "commit", "-qm", "candidate")
	candidate := strings.TrimSpace(runGit(t, assignment.WorkspacePath, "rev-parse", "HEAD"))
	job, err := harness.service.Report(ctx, harness.implementer, factory.WorkerReportRequest{RequestID: "implementation-terminal", AttemptID: assignment.AttemptID, DispatchID: assignment.DispatchID, Kind: factory.ReportTerminal, ObservedAt: time.Now(), NativeRunID: "native-1", Outcome: factory.OutcomeSucceeded, CandidateRevision: candidate, Summary: "implementation finished", TerminalReceipt: "native terminal succeeded", Artifacts: []factory.ArtifactInput{{Name: "implementation.log", MediaType: "text/plain", Content: "fixture completed\n"}}})
	if err != nil {
		t.Fatal(err)
	}
	if job.Status != factory.JobVerifying || job.CandidateRevision != candidate {
		t.Fatalf("job after implementation = %+v", job)
	}
	if len(job.Attempts) == 0 || job.Attempts[0].Summary != "implementation finished" || job.Attempts[0].TerminalReceipt != "native terminal succeeded" || len(job.Attempts[0].Artifacts) != 1 || job.Attempts[0].Artifacts[0].Content != "fixture completed\n" {
		t.Fatalf("implementation evidence was not inspectable on its attempt: %+v", job.Attempts)
	}

	verification, err := harness.service.Claim(ctx, harness.verifier, factory.WorkerClaimRequest{RequestID: "claim-verification", Stages: []factory.Stage{factory.StageVerification}, ProjectID: &harness.project.ID})
	if err != nil {
		t.Fatal(err)
	}
	if verification.Stage != factory.StageVerification || verification.CandidateRevision != candidate || verification.WorkspacePath == assignment.WorkspacePath {
		t.Fatalf("unexpected verification assignment: %+v", verification)
	}
	if status := strings.TrimSpace(runGit(t, verification.WorkspacePath, "status", "--porcelain")); status != "" {
		t.Fatalf("verification workspace is dirty: %q", status)
	}
	if _, err := harness.service.Report(ctx, harness.verifier, factory.WorkerReportRequest{RequestID: "verify-registered", AttemptID: verification.AttemptID, DispatchID: verification.DispatchID, Kind: factory.ReportStartRegistered, ObservedAt: time.Now(), NativeRunID: "verify-native"}); err != nil {
		t.Fatal(err)
	}
	if _, err := harness.service.Report(ctx, harness.verifier, factory.WorkerReportRequest{RequestID: "verify-started", AttemptID: verification.AttemptID, DispatchID: verification.DispatchID, Kind: factory.ReportStarted, ObservedAt: time.Now(), NativeRunID: "verify-native"}); err != nil {
		t.Fatal(err)
	}
	job, err = harness.service.Report(ctx, harness.verifier, factory.WorkerReportRequest{RequestID: "verify-terminal", AttemptID: verification.AttemptID, DispatchID: verification.DispatchID, Kind: factory.ReportTerminal, ObservedAt: time.Now(), NativeRunID: "verify-native", Outcome: factory.OutcomeSucceeded, CheckStatus: factory.CheckPassed, TerminalReceipt: "native check terminal", Artifacts: []factory.ArtifactInput{{Name: "go-test.txt", MediaType: "text/plain", Content: "PASS\n"}}})
	if err != nil {
		t.Fatal(err)
	}
	if job.Status != factory.JobReadyForReview || len(job.Checks) == 0 || job.Checks[len(job.Checks)-1].Status != factory.CheckPassed {
		t.Fatalf("job after verification = %+v", job)
	}
	job, err = harness.service.Review(ctx, harness.reviewer, factory.ReviewRequest{RequestID: "review", JobID: job.ID, IfVersion: job.Version, CandidateRevision: candidate, RequirementFingerprint: job.Admission.TicketSHA256, Decision: factory.ReviewApproved, Summary: "Independent fixture review."})
	if err != nil {
		t.Fatal(err)
	}
	if job.Status != factory.JobReadyForAcceptance {
		t.Fatalf("job after review = %+v", job)
	}
	acceptRequest := factory.AcceptRequest{RequestID: "accept", JobID: job.ID, IfVersion: job.Version, CandidateRevision: candidate, RequirementFingerprint: job.Admission.TicketSHA256, Summary: "Fixture candidate meets the captured requirements."}
	job, err = harness.service.Accept(ctx, harness.reviewer, acceptRequest)
	if err != nil {
		t.Fatal(err)
	}
	if job.Status != factory.JobAccepted || job.AcceptanceStatus != "valid" || job.AcceptanceRecordPath == "" {
		t.Fatalf("job after acceptance = %+v", job)
	}
	replayed, err := harness.service.Accept(ctx, harness.reviewer, acceptRequest)
	if err != nil || replayed.ID != job.ID || replayed.Status != factory.JobAccepted {
		t.Fatalf("acceptance replay = %+v, %v", replayed, err)
	}
	oriented, err := harness.service.Orientation(ctx, harness.reviewer, harness.project.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, work := range oriented.Result.WorkItems {
		if work.ID != nil && *work.ID == job.WorkItemID {
			if work.Acceptance == nil || work.Acceptance.Status != "valid" {
				t.Fatalf("reader rejected service acceptance: %+v", work.Acceptance)
			}
			if work.Acceptance.Actor == nil || work.Acceptance.Actor.Kind != "workflow" || len(work.Acceptance.HumanApprovals) != 0 {
				t.Fatalf("service invented human approval provenance: %+v", work.Acceptance)
			}
		}
	}
	evidencePath := filepath.Join(harness.repository, ".factory-evidence", string(harness.project.ID), string(job.ID)+".json")
	evidence, err := os.ReadFile(evidencePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(evidencePath, append(evidence, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	refreshed, err := harness.service.GetJob(ctx, harness.reviewer, job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if refreshed.Status != factory.JobAccepted || refreshed.AcceptanceStatus != "stale" {
		t.Fatalf("tampered evidence did not stale the accepted job view: %+v", refreshed)
	}
	if err := os.Remove(evidencePath); err != nil {
		t.Fatal(err)
	}
	refreshed, err = harness.service.GetJob(ctx, harness.reviewer, job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if refreshed.Status != factory.JobAccepted || refreshed.AcceptanceStatus != "unknown" {
		t.Fatalf("missing evidence did not make the accepted job view unknown: %+v", refreshed)
	}
	if err := os.WriteFile(evidencePath, evidence, 0o600); err != nil {
		t.Fatal(err)
	}
	refreshed, err = harness.service.GetJob(ctx, harness.reviewer, job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if refreshed.Status != factory.JobAccepted || refreshed.AcceptanceStatus != "valid" {
		t.Fatalf("restored evidence did not revalidate the accepted job view: %+v", refreshed)
	}
}

func TestCancellationAcknowledgementAndRestartRemainUncertainUntilResolved(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	harness := setupJob(t)
	assignment, err := harness.service.Claim(ctx, harness.implementer, factory.WorkerClaimRequest{RequestID: "claim", Stages: []factory.Stage{factory.StageImplementation}, ProjectID: &harness.project.ID})
	if err != nil {
		t.Fatal(err)
	}
	job, err := harness.service.Report(ctx, harness.implementer, factory.WorkerReportRequest{RequestID: "registered", AttemptID: assignment.AttemptID, DispatchID: assignment.DispatchID, Kind: factory.ReportStartRegistered, ObservedAt: time.Now(), NativeRunID: "native-cancel"})
	if err != nil {
		t.Fatal(err)
	}
	job, err = harness.service.Report(ctx, harness.implementer, factory.WorkerReportRequest{RequestID: "started", AttemptID: assignment.AttemptID, DispatchID: assignment.DispatchID, Kind: factory.ReportStarted, ObservedAt: time.Now(), NativeRunID: "native-cancel"})
	if err != nil {
		t.Fatal(err)
	}
	job, err = harness.service.Action(ctx, harness.coordinator, factory.JobActionRequest{RequestID: "cancel", JobID: job.ID, IfVersion: job.Version, Action: factory.ActionCancel})
	if err != nil || job.Status != factory.JobCancelRequested {
		t.Fatalf("cancel = %+v, %v", job, err)
	}
	heartbeat, err := harness.service.Heartbeat(ctx, harness.implementer, factory.HeartbeatRequest{RequestID: "heartbeat", AttemptID: assignment.AttemptID, DispatchID: assignment.DispatchID})
	if err != nil || heartbeat.Directive != factory.DirectiveCancel {
		t.Fatalf("heartbeat = %+v, %v", heartbeat, err)
	}
	job, err = harness.service.Report(ctx, harness.implementer, factory.WorkerReportRequest{RequestID: "cancel-ack", AttemptID: assignment.AttemptID, DispatchID: assignment.DispatchID, Kind: factory.ReportCancelAcknowledged, ObservedAt: time.Now(), NativeRunID: "native-cancel"})
	if err != nil {
		t.Fatal(err)
	}
	if job.Status != factory.JobCancelRequested || job.Attempts[len(job.Attempts)-1].Status != factory.AttemptCancelAcknowledged {
		t.Fatalf("acknowledgement was treated as terminal: %+v", job)
	}
	if err := harness.service.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := factory.Open(harness.dataDir, factory.Options{LeaseDuration: time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = reopened.Close() })
	harness.service = reopened
	job, err = reopened.GetJob(ctx, harness.coordinator, job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if job.Status != factory.JobCancellationUnknown || job.Attempts[len(job.Attempts)-1].Status != factory.AttemptUnknown {
		t.Fatalf("restart did not preserve uncertainty: %+v", job)
	}
	if _, err := reopened.Claim(ctx, harness.implementer, factory.WorkerClaimRequest{RequestID: "claim-after-restart", Stages: []factory.Stage{factory.StageImplementation}, ProjectID: &harness.project.ID}); factory.ErrorCodeOf(err) != factory.CodeUncertain {
		t.Fatalf("claim after restart error = %v, want uncertain", err)
	}
	job, err = reopened.Action(ctx, harness.admin, factory.JobActionRequest{RequestID: "resolve", JobID: job.ID, IfVersion: job.Version, Action: factory.ActionResolve, AttemptID: &assignment.AttemptID, Resolution: factory.ResolutionNotStarted, Evidence: "The registered process was independently proven not to have started."})
	if err != nil {
		t.Fatal(err)
	}
	if job.Status != factory.JobQueued || len(job.Attempts) != 2 || job.Attempts[0].Status != factory.AttemptCancelled || job.Attempts[1].Status != factory.AttemptPending || job.Attempts[1].ID == assignment.AttemptID {
		t.Fatalf("resolved job = %+v", job)
	}
	replacement, err := reopened.Claim(ctx, harness.implementer, factory.WorkerClaimRequest{RequestID: "replacement-claim", Stages: []factory.Stage{factory.StageImplementation}, ProjectID: &harness.project.ID})
	if err != nil {
		t.Fatal(err)
	}
	if replacement.AttemptID == assignment.AttemptID || replacement.DispatchID == assignment.DispatchID {
		t.Fatalf("replacement reused immutable worker receipt identity: %+v", replacement)
	}
}

func TestRepairUsesFreshRequirementsAndOnlyCurrentCandidateChecks(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	harness := setupJob(t)
	implement := func(prefix, filename string) (factory.Job, string) {
		t.Helper()
		assignment, err := harness.service.Claim(ctx, harness.implementer, factory.WorkerClaimRequest{RequestID: prefix + "-claim", Stages: []factory.Stage{factory.StageImplementation}, ProjectID: &harness.project.ID})
		if err != nil {
			t.Fatal(err)
		}
		native := prefix + "-native"
		if _, err := harness.service.Report(ctx, harness.implementer, factory.WorkerReportRequest{RequestID: prefix + "-registered", AttemptID: assignment.AttemptID, DispatchID: assignment.DispatchID, Kind: factory.ReportStartRegistered, ObservedAt: time.Now(), NativeRunID: native}); err != nil {
			t.Fatal(err)
		}
		if _, err := harness.service.Report(ctx, harness.implementer, factory.WorkerReportRequest{RequestID: prefix + "-started", AttemptID: assignment.AttemptID, DispatchID: assignment.DispatchID, Kind: factory.ReportStarted, ObservedAt: time.Now(), NativeRunID: native}); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(assignment.WorkspacePath, filename), []byte(prefix+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		runGit(t, assignment.WorkspacePath, "add", filename)
		runGit(t, assignment.WorkspacePath, "commit", "-qm", prefix)
		candidate := strings.TrimSpace(runGit(t, assignment.WorkspacePath, "rev-parse", "HEAD"))
		if _, err := harness.service.Report(ctx, harness.implementer, factory.WorkerReportRequest{RequestID: prefix + "-wrong-candidate", AttemptID: assignment.AttemptID, DispatchID: assignment.DispatchID, Kind: factory.ReportTerminal, ObservedAt: time.Now(), NativeRunID: native, Outcome: factory.OutcomeSucceeded, CandidateRevision: assignment.BaseRevision, TerminalReceipt: "wrong candidate receipt"}); factory.ErrorCodeOf(err) != factory.CodeConflict {
			t.Fatalf("wrong candidate error = %v, want conflict", err)
		}
		job, err := harness.service.Report(ctx, harness.implementer, factory.WorkerReportRequest{RequestID: prefix + "-terminal", AttemptID: assignment.AttemptID, DispatchID: assignment.DispatchID, Kind: factory.ReportTerminal, ObservedAt: time.Now(), NativeRunID: native, Outcome: factory.OutcomeSucceeded, CandidateRevision: candidate, TerminalReceipt: "implementation receipt"})
		if err != nil {
			t.Fatal(err)
		}
		return job, candidate
	}
	verify := func(prefix string, status factory.CheckStatus, outcome factory.WorkerOutcome) factory.Job {
		t.Helper()
		assignment, err := harness.service.Claim(ctx, harness.verifier, factory.WorkerClaimRequest{RequestID: prefix + "-claim", Stages: []factory.Stage{factory.StageVerification}, ProjectID: &harness.project.ID})
		if err != nil {
			t.Fatal(err)
		}
		native := prefix + "-native"
		if _, err := harness.service.Report(ctx, harness.verifier, factory.WorkerReportRequest{RequestID: prefix + "-registered", AttemptID: assignment.AttemptID, DispatchID: assignment.DispatchID, Kind: factory.ReportStartRegistered, ObservedAt: time.Now(), NativeRunID: native}); err != nil {
			t.Fatal(err)
		}
		if _, err := harness.service.Report(ctx, harness.verifier, factory.WorkerReportRequest{RequestID: prefix + "-started", AttemptID: assignment.AttemptID, DispatchID: assignment.DispatchID, Kind: factory.ReportStarted, ObservedAt: time.Now(), NativeRunID: native}); err != nil {
			t.Fatal(err)
		}
		job, err := harness.service.Report(ctx, harness.verifier, factory.WorkerReportRequest{RequestID: prefix + "-terminal", AttemptID: assignment.AttemptID, DispatchID: assignment.DispatchID, Kind: factory.ReportTerminal, ObservedAt: time.Now(), NativeRunID: native, Outcome: outcome, CheckStatus: status, CandidateRevision: assignment.CandidateRevision, TerminalReceipt: "verification receipt"})
		if err != nil {
			t.Fatal(err)
		}
		return job
	}

	job, firstCandidate := implement("first", "first.txt")
	job = verify("first-check", factory.CheckFailed, factory.OutcomeFailed)
	if job.Status != factory.JobVerificationFailed {
		t.Fatalf("failed verification job = %+v", job)
	}
	ticketPath := job.Admission.TicketPath
	original, err := os.ReadFile(ticketPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ticketPath, append(append([]byte{}, original...), '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := harness.service.Action(ctx, harness.coordinator, factory.JobActionRequest{RequestID: "stale-repair", JobID: job.ID, IfVersion: job.Version, Action: factory.ActionRepair}); factory.ErrorCodeOf(err) != factory.CodeStale {
		t.Fatalf("stale repair error = %v, want stale", err)
	}
	if err := os.WriteFile(ticketPath, original, 0o600); err != nil {
		t.Fatal(err)
	}
	job, err = harness.service.Action(ctx, harness.coordinator, factory.JobActionRequest{RequestID: "repair", JobID: job.ID, IfVersion: job.Version, Action: factory.ActionRepair})
	if err != nil {
		t.Fatal(err)
	}
	if job.Status != factory.JobQueued || job.RepairCount != 1 {
		t.Fatalf("queued repair = %+v", job)
	}
	job, secondCandidate := implement("second", "second.txt")
	if secondCandidate == firstCandidate {
		t.Fatal("repair reused the failed candidate")
	}
	job = verify("second-check", factory.CheckPassed, factory.OutcomeSucceeded)
	if job.Status != factory.JobReadyForReview {
		t.Fatalf("historical failed check blocked repaired candidate: %+v", job)
	}
}

func TestCancellingMultipleChecksRetainsClaimUntilRunningVerifierStops(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	harness := setupJobWithChecks(t, []string{"check-a", "check-b"})
	implementation, err := harness.service.Claim(ctx, harness.implementer, factory.WorkerClaimRequest{RequestID: "implement-claim", Stages: []factory.Stage{factory.StageImplementation}, ProjectID: &harness.project.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := harness.service.Report(ctx, harness.implementer, factory.WorkerReportRequest{RequestID: "implement-register", AttemptID: implementation.AttemptID, DispatchID: implementation.DispatchID, Kind: factory.ReportStartRegistered, ObservedAt: time.Now(), NativeRunID: "implement-native"}); err != nil {
		t.Fatal(err)
	}
	if _, err := harness.service.Report(ctx, harness.implementer, factory.WorkerReportRequest{RequestID: "implement-start", AttemptID: implementation.AttemptID, DispatchID: implementation.DispatchID, Kind: factory.ReportStarted, ObservedAt: time.Now(), NativeRunID: "implement-native"}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(implementation.WorkspacePath, "candidate.txt"), []byte("candidate\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runGit(t, implementation.WorkspacePath, "add", "candidate.txt")
	runGit(t, implementation.WorkspacePath, "commit", "-qm", "candidate")
	candidate := strings.TrimSpace(runGit(t, implementation.WorkspacePath, "rev-parse", "HEAD"))
	job, err := harness.service.Report(ctx, harness.implementer, factory.WorkerReportRequest{RequestID: "implement-terminal", AttemptID: implementation.AttemptID, DispatchID: implementation.DispatchID, Kind: factory.ReportTerminal, ObservedAt: time.Now(), NativeRunID: "implement-native", Outcome: factory.OutcomeSucceeded, CandidateRevision: candidate, TerminalReceipt: "implementation receipt"})
	if err != nil {
		t.Fatal(err)
	}
	verification, err := harness.service.Claim(ctx, harness.verifier, factory.WorkerClaimRequest{RequestID: "verify-claim", Stages: []factory.Stage{factory.StageVerification}, ProjectID: &harness.project.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := harness.service.Report(ctx, harness.verifier, factory.WorkerReportRequest{RequestID: "verify-register", AttemptID: verification.AttemptID, DispatchID: verification.DispatchID, Kind: factory.ReportStartRegistered, ObservedAt: time.Now(), NativeRunID: "verify-native"}); err != nil {
		t.Fatal(err)
	}
	job, err = harness.service.Report(ctx, harness.verifier, factory.WorkerReportRequest{RequestID: "verify-start", AttemptID: verification.AttemptID, DispatchID: verification.DispatchID, Kind: factory.ReportStarted, ObservedAt: time.Now(), NativeRunID: "verify-native"})
	if err != nil {
		t.Fatal(err)
	}
	job, err = harness.service.Action(ctx, harness.coordinator, factory.JobActionRequest{RequestID: "cancel", JobID: job.ID, IfVersion: job.Version, Action: factory.ActionCancel})
	if err != nil {
		t.Fatal(err)
	}
	if job.Status != factory.JobCancelRequested {
		t.Fatalf("cancel released a still-running verifier: %+v", job)
	}
	var running, cancelled int
	for _, attempt := range job.Attempts {
		if attempt.Stage != factory.StageVerification {
			continue
		}
		if attempt.Status == factory.AttemptRunning && attempt.CancellationRequested {
			running++
		}
		if attempt.Status == factory.AttemptCancelled {
			cancelled++
		}
	}
	if running != 1 || cancelled != 1 {
		t.Fatalf("cancel did not address every verification attempt: %+v", job.Attempts)
	}
	if _, err := harness.service.Report(ctx, harness.verifier, factory.WorkerReportRequest{RequestID: "verify-ack", AttemptID: verification.AttemptID, DispatchID: verification.DispatchID, Kind: factory.ReportCancelAcknowledged, ObservedAt: time.Now(), NativeRunID: "verify-native"}); err != nil {
		t.Fatal(err)
	}
	job, err = harness.service.Report(ctx, harness.verifier, factory.WorkerReportRequest{RequestID: "verify-terminal", AttemptID: verification.AttemptID, DispatchID: verification.DispatchID, Kind: factory.ReportTerminal, ObservedAt: time.Now(), NativeRunID: "verify-native", Outcome: factory.OutcomeCancelled, CheckStatus: factory.CheckStatus("cancelled"), CandidateRevision: candidate, TerminalReceipt: "cancelled verification receipt"})
	if err != nil {
		t.Fatal(err)
	}
	if job.Status != factory.JobCancelled {
		t.Fatalf("terminal verifier did not complete cancellation: %+v", job)
	}
}

func TestRequiredCheckObservationsCannotFalseAccept(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		status  factory.CheckStatus
		outcome factory.WorkerOutcome
		kind    factory.WorkerReportKind
	}{
		{name: "missing"},
		{name: "passed_without_review", status: factory.CheckPassed, outcome: factory.OutcomeSucceeded, kind: factory.ReportTerminal},
		{name: "failed", status: factory.CheckFailed, outcome: factory.OutcomeFailed, kind: factory.ReportTerminal},
		{name: "skipped", status: factory.CheckSkipped, outcome: factory.OutcomeSucceeded, kind: factory.ReportTerminal},
		{name: "unavailable", status: factory.CheckUnavailable, outcome: factory.OutcomeFailed, kind: factory.ReportTerminal},
		{name: "unknown", status: factory.CheckUnknown, outcome: factory.OutcomeUnknown, kind: factory.ReportUnknown},
	}
	for _, test := range cases {
		test := test
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			harness := setupJob(t)
			job, candidate := completeImplementationForTest(t, harness, "gate-"+test.name)
			if test.kind != "" {
				assignment := startVerificationForTest(t, harness, "gate-"+test.name)
				var err error
				job, err = harness.service.Report(ctx, harness.verifier, factory.WorkerReportRequest{RequestID: "gate-report-" + test.name, AttemptID: assignment.AttemptID, DispatchID: assignment.DispatchID, Kind: test.kind, ObservedAt: time.Now(), NativeRunID: "gate-" + test.name + "-verify-native", Outcome: test.outcome, CheckStatus: test.status, CandidateRevision: assignment.CandidateRevision, TerminalReceipt: "retained gate receipt"})
				if err != nil {
					t.Fatal(err)
				}
				if job.ID == "" {
					t.Fatal("worker observation was not retained")
				}
			}
			_, err := harness.service.Accept(ctx, harness.reviewer, factory.AcceptRequest{RequestID: "accept-" + test.name, JobID: job.ID, IfVersion: job.Version, CandidateRevision: candidate, RequirementFingerprint: job.Admission.TicketSHA256, Summary: "must remain blocked"})
			if factory.ErrorCodeOf(err) != factory.CodeNotReady {
				t.Fatalf("accept with %s check error = %v, want not ready", test.name, err)
			}
		})
	}
}

func TestRepairBoundsRejectZeroBudgetAndExpiredDeadline(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	t.Run("zero budget", func(t *testing.T) {
		harness := setupJobConfig(t, []string{"go-test"}, 0, nil)
		job := failImplementationForTest(t, harness, "zero")
		_, err := harness.service.Action(ctx, harness.coordinator, factory.JobActionRequest{RequestID: "repair-zero", JobID: job.ID, IfVersion: job.Version, Action: factory.ActionRepair})
		if factory.ErrorCodeOf(err) != factory.CodeExhausted {
			t.Fatalf("zero-budget repair error = %v, want exhausted", err)
		}
	})
	t.Run("expired deadline", func(t *testing.T) {
		clock := time.Date(2026, 10, 23, 10, 0, 0, 0, time.UTC)
		harness := setupJobConfig(t, []string{"go-test"}, 1, func() time.Time { return clock })
		job := failImplementationForTest(t, harness, "expired")
		clock = clock.Add(31 * time.Minute)
		_, err := harness.service.Action(ctx, harness.coordinator, factory.JobActionRequest{RequestID: "repair-expired", JobID: job.ID, IfVersion: job.Version, Action: factory.ActionRepair})
		if factory.ErrorCodeOf(err) != factory.CodeDeadline {
			t.Fatalf("expired repair error = %v, want deadline", err)
		}
	})
}

func TestClaimAndStartRecheckSourcesAndRevokedGrant(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	harness := setupJob(t)
	ticket := harness.job.Admission.TicketPath
	original, err := os.ReadFile(ticket)
	if err != nil {
		t.Fatal(err)
	}
	changed := append(append([]byte{}, original...), []byte("\nChanged requirement.\n")...)
	if err := os.WriteFile(ticket, changed, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := harness.service.Claim(ctx, harness.implementer, factory.WorkerClaimRequest{RequestID: "stale-claim", Stages: []factory.Stage{factory.StageImplementation}, ProjectID: &harness.project.ID}); factory.ErrorCodeOf(err) != factory.CodeStale {
		t.Fatalf("claim after source change error = %v, want stale", err)
	}
	if err := os.WriteFile(ticket, original, 0o600); err != nil {
		t.Fatal(err)
	}
	assignment, err := harness.service.Claim(ctx, harness.implementer, factory.WorkerClaimRequest{RequestID: "fresh-claim", Stages: []factory.Stage{factory.StageImplementation}, ProjectID: &harness.project.ID})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ticket, changed, 0o600); err != nil {
		t.Fatal(err)
	}
	start := factory.WorkerReportRequest{RequestID: "stale-start", AttemptID: assignment.AttemptID, DispatchID: assignment.DispatchID, Kind: factory.ReportStartRegistered, ObservedAt: time.Now(), NativeRunID: "never-started"}
	if _, err := harness.service.Report(ctx, harness.implementer, start); factory.ErrorCodeOf(err) != factory.CodeStale {
		t.Fatalf("start after source change error = %v, want stale", err)
	}
	if err := os.WriteFile(ticket, original, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := harness.service.GrantProject(ctx, harness.admin, factory.GrantProjectRequest{RequestID: "revoke-worker-project", ProjectID: harness.project.ID, UserID: harness.implementer.UserID, Capabilities: []factory.Capability{}}); err != nil {
		t.Fatal(err)
	}
	start.RequestID = "revoked-start"
	_, err = harness.service.Report(ctx, harness.implementer, start)
	if code := factory.ErrorCodeOf(err); code != factory.CodeNotFound && code != factory.CodeForbidden {
		t.Fatalf("start after grant revocation error = %v, want access denial", err)
	}
	_, err = harness.service.Claim(ctx, harness.implementer, factory.WorkerClaimRequest{RequestID: "revoked-reclaim", Stages: []factory.Stage{factory.StageImplementation}, ProjectID: &harness.project.ID})
	if code := factory.ErrorCodeOf(err); code != factory.CodeNotFound && code != factory.CodeForbidden {
		t.Fatalf("claim after grant revocation error = %v, want access denial", err)
	}
	ack, err := harness.service.Report(ctx, harness.implementer, factory.WorkerReportRequest{RequestID: "drain-prepared", AttemptID: assignment.AttemptID, DispatchID: assignment.DispatchID, Kind: factory.ReportUnknown, ObservedAt: time.Now(), NativeRunID: "prepared-receipt", Outcome: factory.OutcomeUnknown, TerminalReceipt: "prepared receipt proves no terminal process outcome"})
	if err != nil {
		t.Fatal(err)
	}
	if ack.ID == "" || ack.Status != factory.JobImplementationUnknown || ack.Admission.TicketPath != "" || len(ack.Attempts) != 0 || len(ack.Events) != 0 {
		t.Fatalf("revoked worker received unredacted terminal acknowledgement: %+v", ack)
	}
}

func TestUnknownVerificationCannotCompleteCancellationOrReleaseClaim(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	harness := setupJob(t)
	job, _ := completeImplementationForTest(t, harness, "cancel-unknown")
	assignment := startVerificationForTest(t, harness, "cancel-unknown")
	job, err := harness.service.GetJob(ctx, harness.coordinator, job.ID)
	if err != nil {
		t.Fatal(err)
	}
	job, err = harness.service.Action(ctx, harness.coordinator, factory.JobActionRequest{RequestID: "cancel-unknown-action", JobID: job.ID, IfVersion: job.Version, Action: factory.ActionCancel})
	if err != nil {
		t.Fatal(err)
	}
	job, err = harness.service.Report(ctx, harness.verifier, factory.WorkerReportRequest{
		RequestID: "cancel-unknown-terminal", AttemptID: assignment.AttemptID, DispatchID: assignment.DispatchID,
		Kind: factory.ReportTerminal, ObservedAt: time.Now(), NativeRunID: "cancel-unknown-verify-native",
		Outcome: factory.OutcomeSucceeded, CheckStatus: factory.CheckUnknown, TerminalReceipt: "check outcome remains unknown",
	})
	if err != nil {
		t.Fatal(err)
	}
	if job.Status != factory.JobCancellationUnknown {
		t.Fatalf("unknown cancelling verifier released the claim: %+v", job)
	}
	_, err = harness.service.Submit(ctx, harness.coordinator, factory.SubmitRequest{RequestID: "cancel-unknown-second", ProjectID: harness.project.ID, WorkItemID: job.WorkItemID, BaseRevision: "HEAD", ExecutorKey: "fixture-agent", Deadline: time.Now().Add(time.Hour)})
	if factory.ErrorCodeOf(err) != factory.CodeConflict {
		t.Fatalf("second submit with uncertain verifier error = %v, want conflict", err)
	}
}

func TestResolvingOneVerifierRetainsOtherActiveVerifier(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	harness := setupJobWithChecks(t, []string{"first", "second"})
	job, _ := completeImplementationForTest(t, harness, "resolve-one")
	first := startVerificationForTest(t, harness, "resolve-first")
	user, err := harness.service.CreateUser(ctx, harness.admin, factory.CreateUserRequest{RequestID: "resolve-second-user", Username: "second-verifier", Kind: factory.UserKindWorker})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := harness.service.GrantProject(ctx, harness.admin, factory.GrantProjectRequest{RequestID: "resolve-second-grant", ProjectID: harness.project.ID, UserID: user.ID, Capabilities: []factory.Capability{factory.CapabilityRead, factory.CapabilityVerifyWorker}}); err != nil {
		t.Fatal(err)
	}
	token, err := harness.service.IssueToken(ctx, harness.admin, factory.IssueTokenRequest{RequestID: "resolve-second-token", UserID: user.ID, Label: "test"})
	if err != nil {
		t.Fatal(err)
	}
	secondVerifier, err := harness.service.Authenticate(ctx, token.Token)
	if err != nil {
		t.Fatal(err)
	}
	second, err := harness.service.Claim(ctx, secondVerifier, factory.WorkerClaimRequest{RequestID: "resolve-second-claim", Stages: []factory.Stage{factory.StageVerification}, ProjectID: &harness.project.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := harness.service.Report(ctx, secondVerifier, factory.WorkerReportRequest{RequestID: "resolve-second-registered", AttemptID: second.AttemptID, DispatchID: second.DispatchID, Kind: factory.ReportStartRegistered, ObservedAt: time.Now(), NativeRunID: "resolve-second-native"}); err != nil {
		t.Fatal(err)
	}
	job, err = harness.service.Report(ctx, harness.verifier, factory.WorkerReportRequest{RequestID: "resolve-first-unknown", AttemptID: first.AttemptID, DispatchID: first.DispatchID, Kind: factory.ReportUnknown, ObservedAt: time.Now(), NativeRunID: "resolve-first-verify-native", Outcome: factory.OutcomeUnknown, TerminalReceipt: "first verifier outcome unknown"})
	if err != nil {
		t.Fatal(err)
	}
	job, err = harness.service.Action(ctx, harness.admin, factory.JobActionRequest{RequestID: "resolve-first-cancelled", JobID: job.ID, IfVersion: job.Version, Action: factory.ActionResolve, AttemptID: &first.AttemptID, Resolution: factory.ResolutionCancelled, Evidence: "first verifier was independently proven stopped"})
	if err != nil {
		t.Fatal(err)
	}
	if job.Status != factory.JobVerifying {
		t.Fatalf("resolving one verifier released another active verifier: %+v", job)
	}
	_, err = harness.service.Submit(ctx, harness.coordinator, factory.SubmitRequest{RequestID: "resolve-one-second-job", ProjectID: harness.project.ID, WorkItemID: job.WorkItemID, BaseRevision: "HEAD", ExecutorKey: "fixture-agent", Deadline: time.Now().Add(time.Hour)})
	if factory.ErrorCodeOf(err) != factory.CodeConflict {
		t.Fatalf("second submit with active verifier error = %v, want conflict", err)
	}
}

func TestStartRegistrationAfterDeadlineRemainsUncertain(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	clock := time.Date(2026, 10, 23, 10, 0, 0, 0, time.UTC)
	harness := setupJobConfig(t, []string{"go-test"}, 1, func() time.Time { return clock })
	assignment, err := harness.service.Claim(ctx, harness.implementer, factory.WorkerClaimRequest{RequestID: "late-claim", Stages: []factory.Stage{factory.StageImplementation}, ProjectID: &harness.project.ID})
	if err != nil {
		t.Fatal(err)
	}
	clock = clock.Add(31 * time.Minute)
	_, err = harness.service.Report(ctx, harness.implementer, factory.WorkerReportRequest{RequestID: "late-start", AttemptID: assignment.AttemptID, DispatchID: assignment.DispatchID, Kind: factory.ReportStartRegistered, ObservedAt: clock, NativeRunID: "late-native"})
	if factory.ErrorCodeOf(err) != factory.CodeDeadline {
		t.Fatalf("late start error = %v, want deadline", err)
	}
	job, err := harness.service.GetJob(ctx, harness.coordinator, harness.job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if job.Status != factory.JobCancellationUnknown || job.Attempts[0].Status != factory.AttemptUnknown {
		t.Fatalf("late assignment was not retained for reconciliation: %+v", job)
	}
}

func TestAcceptanceInProgressCannotBeCancelledBeforeReconciliation(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	harness := setupJob(t)
	job, candidate := completeImplementationForTest(t, harness, "accept-reconcile")
	verification := startVerificationForTest(t, harness, "accept-reconcile")
	job, err := harness.service.Report(ctx, harness.verifier, factory.WorkerReportRequest{RequestID: "accept-reconcile-check", AttemptID: verification.AttemptID, DispatchID: verification.DispatchID, Kind: factory.ReportTerminal, ObservedAt: time.Now(), NativeRunID: "accept-reconcile-verify-native", Outcome: factory.OutcomeSucceeded, CheckStatus: factory.CheckPassed, TerminalReceipt: "check passed"})
	if err != nil {
		t.Fatal(err)
	}
	job, err = harness.service.Review(ctx, harness.reviewer, factory.ReviewRequest{RequestID: "accept-reconcile-review", JobID: job.ID, IfVersion: job.Version, CandidateRevision: candidate, RequirementFingerprint: job.Admission.TicketSHA256, Decision: factory.ReviewApproved, Summary: "independent approval"})
	if err != nil {
		t.Fatal(err)
	}
	evidencePath := filepath.Join(harness.repository, ".factory-evidence", string(harness.project.ID), string(job.ID)+".json")
	if err := os.MkdirAll(filepath.Dir(evidencePath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(evidencePath, []byte("conflicting destination"), 0o600); err != nil {
		t.Fatal(err)
	}
	request := factory.AcceptRequest{RequestID: "accept-reconcile", JobID: job.ID, IfVersion: job.Version, CandidateRevision: candidate, RequirementFingerprint: job.Admission.TicketSHA256, Summary: "reconciled acceptance"}
	if _, err := harness.service.Accept(ctx, harness.reviewer, request); factory.ErrorCodeOf(err) != factory.CodeStale {
		t.Fatalf("blocked publication error = %v, want stale", err)
	}
	job, err = harness.service.GetJob(ctx, harness.coordinator, job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := harness.service.Action(ctx, harness.coordinator, factory.JobActionRequest{RequestID: "cancel-accepting", JobID: job.ID, IfVersion: job.Version, Action: factory.ActionCancel}); factory.ErrorCodeOf(err) != factory.CodeConflict {
		t.Fatalf("cancel accepting error = %v, want conflict", err)
	}
	if err := os.Remove(evidencePath); err != nil {
		t.Fatal(err)
	}
	job, err = harness.service.Accept(ctx, harness.reviewer, request)
	if err != nil {
		t.Fatal(err)
	}
	if job.Status != factory.JobAccepted {
		t.Fatalf("reconciled acceptance = %+v", job)
	}
}

func TestReplacementVerificationSupersedesUnknownAttempt(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	harness := setupJob(t)
	job, _ := completeImplementationForTest(t, harness, "replacement")
	first := startVerificationForTest(t, harness, "replacement-first")
	job, err := harness.service.Report(ctx, harness.verifier, factory.WorkerReportRequest{RequestID: "replacement-unknown", AttemptID: first.AttemptID, DispatchID: first.DispatchID, Kind: factory.ReportUnknown, ObservedAt: time.Now(), NativeRunID: "replacement-first-verify-native", Outcome: factory.OutcomeUnknown, TerminalReceipt: "first verifier did not start"})
	if err != nil {
		t.Fatal(err)
	}
	job, err = harness.service.Action(ctx, harness.admin, factory.JobActionRequest{RequestID: "replacement-resolve", JobID: job.ID, IfVersion: job.Version, Action: factory.ActionResolve, AttemptID: &first.AttemptID, Resolution: factory.ResolutionNotStarted, Evidence: "receipt proves the first verifier did not start"})
	if err != nil {
		t.Fatal(err)
	}
	replacement := startVerificationForTest(t, harness, "replacement-second")
	job, err = harness.service.Report(ctx, harness.verifier, factory.WorkerReportRequest{RequestID: "replacement-pass", AttemptID: replacement.AttemptID, DispatchID: replacement.DispatchID, Kind: factory.ReportTerminal, ObservedAt: time.Now(), NativeRunID: "replacement-second-verify-native", Outcome: factory.OutcomeSucceeded, CheckStatus: factory.CheckPassed, TerminalReceipt: "replacement passed"})
	if err != nil {
		t.Fatal(err)
	}
	if job.Status != factory.JobReadyForReview {
		t.Fatalf("superseded unknown check blocked replacement: %+v", job)
	}
}

func TestRepairAfterReplacementUsesNewVerificationGeneration(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	harness := setupJob(t)
	job, _ := completeImplementationForTest(t, harness, "generation-first")
	first := startVerificationForTest(t, harness, "generation-first")
	job, err := harness.service.Report(ctx, harness.verifier, factory.WorkerReportRequest{RequestID: "generation-unknown", AttemptID: first.AttemptID, DispatchID: first.DispatchID, Kind: factory.ReportUnknown, ObservedAt: time.Now(), NativeRunID: "generation-first-verify-native", Outcome: factory.OutcomeUnknown, TerminalReceipt: "first verifier did not start"})
	if err != nil {
		t.Fatal(err)
	}
	job, err = harness.service.Action(ctx, harness.admin, factory.JobActionRequest{RequestID: "generation-resolve", JobID: job.ID, IfVersion: job.Version, Action: factory.ActionResolve, AttemptID: &first.AttemptID, Resolution: factory.ResolutionNotStarted, Evidence: "receipt proves the first verifier did not start"})
	if err != nil {
		t.Fatal(err)
	}
	replacement := startVerificationForTest(t, harness, "generation-replacement")
	job, err = harness.service.Report(ctx, harness.verifier, factory.WorkerReportRequest{RequestID: "generation-fail", AttemptID: replacement.AttemptID, DispatchID: replacement.DispatchID, Kind: factory.ReportTerminal, ObservedAt: time.Now(), NativeRunID: "generation-replacement-verify-native", Outcome: factory.OutcomeFailed, CheckStatus: factory.CheckFailed, TerminalReceipt: "replacement failed"})
	if err != nil {
		t.Fatal(err)
	}
	job, err = harness.service.Action(ctx, harness.coordinator, factory.JobActionRequest{RequestID: "generation-repair", JobID: job.ID, IfVersion: job.Version, Action: factory.ActionRepair})
	if err != nil {
		t.Fatal(err)
	}
	job, _ = completeImplementationForTest(t, harness, "generation-repaired")
	verification := startVerificationForTest(t, harness, "generation-repaired")
	job, err = harness.service.Report(ctx, harness.verifier, factory.WorkerReportRequest{RequestID: "generation-repaired-pass", AttemptID: verification.AttemptID, DispatchID: verification.DispatchID, Kind: factory.ReportTerminal, ObservedAt: time.Now(), NativeRunID: "generation-repaired-verify-native", Outcome: factory.OutcomeSucceeded, CheckStatus: factory.CheckPassed, TerminalReceipt: "repaired candidate passed"})
	if err != nil {
		t.Fatal(err)
	}
	if job.Status != factory.JobReadyForReview {
		t.Fatalf("new verification generation did not pass: %+v", job)
	}
}

func completeImplementationForTest(t *testing.T, harness jobHarness, prefix string) (factory.Job, string) {
	t.Helper()
	ctx := context.Background()
	assignment, err := harness.service.Claim(ctx, harness.implementer, factory.WorkerClaimRequest{
		RequestID: prefix + "-implement-claim",
		Stages:    []factory.Stage{factory.StageImplementation},
		ProjectID: &harness.project.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	nativeRunID := prefix + "-implement-native"
	if _, err := harness.service.Report(ctx, harness.implementer, factory.WorkerReportRequest{
		RequestID: prefix + "-implement-registered", AttemptID: assignment.AttemptID, DispatchID: assignment.DispatchID,
		Kind: factory.ReportStartRegistered, ObservedAt: time.Now(), NativeRunID: nativeRunID,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := harness.service.Report(ctx, harness.implementer, factory.WorkerReportRequest{
		RequestID: prefix + "-implement-started", AttemptID: assignment.AttemptID, DispatchID: assignment.DispatchID,
		Kind: factory.ReportStarted, ObservedAt: time.Now(), NativeRunID: nativeRunID,
	}); err != nil {
		t.Fatal(err)
	}
	filename := prefix + ".txt"
	if err := os.WriteFile(filepath.Join(assignment.WorkspacePath, filename), []byte(prefix+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runGit(t, assignment.WorkspacePath, "add", filename)
	runGit(t, assignment.WorkspacePath, "commit", "-qm", prefix)
	candidate := strings.TrimSpace(runGit(t, assignment.WorkspacePath, "rev-parse", "HEAD"))
	job, err := harness.service.Report(ctx, harness.implementer, factory.WorkerReportRequest{
		RequestID: prefix + "-implement-terminal", AttemptID: assignment.AttemptID, DispatchID: assignment.DispatchID,
		Kind: factory.ReportTerminal, ObservedAt: time.Now(), NativeRunID: nativeRunID,
		Outcome: factory.OutcomeSucceeded, CandidateRevision: candidate, TerminalReceipt: "retained implementation receipt",
	})
	if err != nil {
		t.Fatal(err)
	}
	return job, candidate
}

func startVerificationForTest(t *testing.T, harness jobHarness, prefix string) factory.Assignment {
	t.Helper()
	ctx := context.Background()
	assignment, err := harness.service.Claim(ctx, harness.verifier, factory.WorkerClaimRequest{
		RequestID: prefix + "-verify-claim",
		Stages:    []factory.Stage{factory.StageVerification},
		ProjectID: &harness.project.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	nativeRunID := prefix + "-verify-native"
	if _, err := harness.service.Report(ctx, harness.verifier, factory.WorkerReportRequest{
		RequestID: prefix + "-verify-registered", AttemptID: assignment.AttemptID, DispatchID: assignment.DispatchID,
		Kind: factory.ReportStartRegistered, ObservedAt: time.Now(), NativeRunID: nativeRunID,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := harness.service.Report(ctx, harness.verifier, factory.WorkerReportRequest{
		RequestID: prefix + "-verify-started", AttemptID: assignment.AttemptID, DispatchID: assignment.DispatchID,
		Kind: factory.ReportStarted, ObservedAt: time.Now(), NativeRunID: nativeRunID,
	}); err != nil {
		t.Fatal(err)
	}
	return assignment
}

func failImplementationForTest(t *testing.T, harness jobHarness, prefix string) factory.Job {
	t.Helper()
	ctx := context.Background()
	assignment, err := harness.service.Claim(ctx, harness.implementer, factory.WorkerClaimRequest{
		RequestID: prefix + "-claim",
		Stages:    []factory.Stage{factory.StageImplementation},
		ProjectID: &harness.project.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	nativeRunID := prefix + "-native"
	if _, err := harness.service.Report(ctx, harness.implementer, factory.WorkerReportRequest{
		RequestID: prefix + "-registered", AttemptID: assignment.AttemptID, DispatchID: assignment.DispatchID,
		Kind: factory.ReportStartRegistered, ObservedAt: time.Now(), NativeRunID: nativeRunID,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := harness.service.Report(ctx, harness.implementer, factory.WorkerReportRequest{
		RequestID: prefix + "-started", AttemptID: assignment.AttemptID, DispatchID: assignment.DispatchID,
		Kind: factory.ReportStarted, ObservedAt: time.Now(), NativeRunID: nativeRunID,
	}); err != nil {
		t.Fatal(err)
	}
	job, err := harness.service.Report(ctx, harness.implementer, factory.WorkerReportRequest{
		RequestID: prefix + "-terminal", AttemptID: assignment.AttemptID, DispatchID: assignment.DispatchID,
		Kind: factory.ReportTerminal, ObservedAt: time.Now(), NativeRunID: nativeRunID,
		Outcome: factory.OutcomeFailed, TerminalReceipt: "retained failure receipt",
	})
	if err != nil {
		t.Fatal(err)
	}
	return job
}

type jobHarness struct {
	service     *factory.Service
	project     factory.Project
	job         factory.Job
	coordinator factory.Principal
	implementer factory.Principal
	verifier    factory.Principal
	reviewer    factory.Principal
	admin       factory.Principal
	bundle      string
	repository  string
	dataDir     string
}

func setupJob(t *testing.T) jobHarness {
	return setupJobWithChecks(t, []string{"go-test"})
}

func setupJobWithChecks(t *testing.T, checks []string) jobHarness {
	return setupJobConfig(t, checks, 2, nil)
}

func setupJobConfig(t *testing.T, checks []string, repairLimit int, now func() time.Time) jobHarness {
	return setupJobWith(t, checks, repairLimit, now, nil)
}

func setupJobWith(t *testing.T, checks []string, repairLimit int, now func() time.Time, mutate func(*factory.ProjectPolicy)) jobHarness {
	t.Helper()
	ctx := context.Background()
	dataDir := t.TempDir()
	service, err := factory.Open(dataDir, factory.Options{LeaseDuration: time.Minute, Now: now})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = service.Close() })
	bootstrap, err := service.Bootstrap(ctx, factory.BootstrapRequest{Username: "admin"})
	if err != nil {
		t.Fatal(err)
	}
	admin, err := service.Authenticate(ctx, bootstrap.Token)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.RegisterExecutor(ctx, admin, factory.RegisterExecutorRequest{RequestID: "executor", Key: "fixture-agent", DisplayName: "Fixture agent", Command: []string{"fixture-agent", "run"}, Fixture: true}); err != nil {
		t.Fatal(err)
	}
	for _, check := range checks {
		if _, err := service.RegisterCheck(ctx, admin, factory.RegisterCheckRequest{RequestID: "check-" + check, Key: check, DisplayName: "Go test", Command: []string{"go", "test", "./..."}, Environment: "local", Fixture: true}); err != nil {
			t.Fatal(err)
		}
	}
	bundle, repository := readyTestProject(t)
	policy := factory.ProjectPolicy{RequiredChecks: checks, AllowedExecutors: []string{"fixture-agent"}, MaxRepairAttempts: 2, MaxJobDurationSeconds: 3600, Fixture: true}
	if mutate != nil {
		mutate(&policy)
	}
	project, err := service.RegisterProject(ctx, admin, factory.RegisterProjectRequest{RequestID: "project", BundlePath: bundle, RepositoryPath: repository, AllowedSourceRoots: []string{repository}, Policy: policy})
	if err != nil {
		t.Fatal(err)
	}
	makePrincipal := func(name string, kind factory.UserKind, capabilities []factory.Capability) factory.Principal {
		user, err := service.CreateUser(ctx, admin, factory.CreateUserRequest{RequestID: "user-" + name, Username: name, Kind: kind})
		if err != nil {
			t.Fatal(err)
		}
		token, err := service.IssueToken(ctx, admin, factory.IssueTokenRequest{RequestID: "token-" + name, UserID: user.ID, Label: "test"})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := service.GrantProject(ctx, admin, factory.GrantProjectRequest{RequestID: "grant-" + name, ProjectID: project.ID, UserID: user.ID, Capabilities: capabilities}); err != nil {
			t.Fatal(err)
		}
		principal, err := service.Authenticate(ctx, token.Token)
		if err != nil {
			t.Fatal(err)
		}
		return principal
	}
	coordinator := makePrincipal("coordinator", factory.UserKindHuman, []factory.Capability{factory.CapabilityRead, factory.CapabilityCoordinate})
	implementer := makePrincipal("implementer", factory.UserKindWorker, []factory.Capability{factory.CapabilityRead, factory.CapabilityImplementWorker})
	verifier := makePrincipal("verifier", factory.UserKindWorker, []factory.Capability{factory.CapabilityRead, factory.CapabilityVerifyWorker, factory.CapabilityVerify})
	reviewer := makePrincipal("reviewer", factory.UserKindHuman, []factory.Capability{factory.CapabilityRead, factory.CapabilityReview, factory.CapabilityAccept})
	base := strings.TrimSpace(runGit(t, repository, "rev-parse", "HEAD"))
	current := time.Now()
	if now != nil {
		current = now()
	}
	job, err := service.Submit(ctx, coordinator, factory.SubmitRequest{RequestID: "submit", ProjectID: project.ID, WorkItemID: "work-test", BaseRevision: base, ExecutorKey: "fixture-agent", RepairLimit: repairLimit, Deadline: current.Add(30 * time.Minute)})
	if err != nil {
		t.Fatal(err)
	}
	return jobHarness{service: service, project: project, job: job, coordinator: coordinator, implementer: implementer, verifier: verifier, reviewer: reviewer, admin: admin, bundle: bundle, repository: repository, dataDir: dataDir}
}

func testProject(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	bundle := filepath.Join(root, "records")
	repository := filepath.Join(root, "repo")
	if err := os.MkdirAll(bundle, 0o700); err != nil {
		t.Fatal(err)
	}
	manifest := "---\ntype: Project\nid: project-test\ntitle: Test project\n---\n## Goals\nTest.\n## Current commitments\nNone\n## Open decisions\nNone\n"
	if err := os.WriteFile(filepath.Join(bundle, "project.md"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(repository, 0o700); err != nil {
		t.Fatal(err)
	}
	runGit(t, repository, "init", "-q")
	runGit(t, repository, "config", "user.name", "Test")
	runGit(t, repository, "config", "user.email", "test@example.com")
	if err := os.WriteFile(filepath.Join(repository, "README.md"), []byte("test\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runGit(t, repository, "add", "README.md")
	runGit(t, repository, "commit", "-qm", "initial")
	return bundle, repository
}

func readyTestProject(t *testing.T) (string, string) {
	t.Helper()
	bundle, repository := testProject(t)
	manifest := "---\ntype: Project\nid: project-test\ntitle: Test project\n---\n## Goals\nTest.\n## Current commitments\n- [Work](work.md)\n## Open decisions\nNone\n"
	if err := os.WriteFile(filepath.Join(bundle, "project.md"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	work := "---\ntype: WorkItem\nid: work-test\ntitle: Ready work\ntriage: ready-for-agent\nexecution: unstarted\n---\n## Acceptance criteria\n- The change works.\n\n## Spec\nNone\n\n## Context\nNone\n\n## Blocked by\nNone\n\n## Blocked by decisions\nNone\n"
	if err := os.WriteFile(filepath.Join(bundle, "work.md"), []byte(work), 0o600); err != nil {
		t.Fatal(err)
	}
	return bundle, repository
}

func runGit(t *testing.T, directory string, args ...string) string {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", directory}, args...)...)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, output)
	}
	return string(output)
}
