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

func sleep() { time.Sleep(50 * time.Millisecond) }

// coveringAuthor writes one protected test file and reports that it checks the work
// item's only criterion.
func coveringAuthor(t *testing.T, concern string) []string {
	t.Helper()
	concerns := "[]"
	if concern != "" {
		concerns = `["` + concern + `"]`
	}
	script := "#!/bin/sh\ncat > /dev/null\nmkdir -p tests\nprintf 'expects the change\\n' > tests/pilot_test.txt\n" +
		`echo '{"notes":"One test per criterion.","coverage":[{"criterion":1,"tests":["TestChangeWorks"]}],"concerns":` + concerns + `,"run":{"model":"fake-author","turns":2,"costUsd":0.05}}'` + "\n"
	path := filepath.Join(t.TempDir(), "author")
	if err := os.WriteFile(path, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	return []string{path}
}

// setupPilot opens a service whose autopilot runs only when the test calls PilotOnce.
func setupPilot(t *testing.T, author []string, mutate func(*factory.ProjectPolicy)) jobHarness {
	t.Helper()
	ctx := context.Background()
	dataDir := t.TempDir()
	service, err := factory.Open(dataDir, factory.Options{LeaseDuration: time.Minute, ExpectationCommand: author, Pilot: true, PilotInterval: time.Hour})
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
	if _, err := service.RegisterExecutor(ctx, admin, factory.RegisterExecutorRequest{RequestID: "executor", Key: "fixture-agent", DisplayName: "Fixture agent", Command: []string{"fixture-agent"}, Fixture: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.RegisterCheck(ctx, admin, factory.RegisterCheckRequest{RequestID: "check", Key: "go-test", DisplayName: "Go test", Command: []string{"go", "test", "./..."}, Environment: "local", Fixture: true}); err != nil {
		t.Fatal(err)
	}
	bundle, repository := readyTestProject(t)
	policy := factory.ProjectPolicy{RequiredChecks: []string{"go-test"}, AllowedExecutors: []string{"fixture-agent"}, MaxRepairAttempts: 2, MaxJobDurationSeconds: 3600, Fixture: true, ProtectedPaths: []string{"tests/"}, AllowCoordinatorReview: true}
	if mutate != nil {
		mutate(&policy)
	}
	project, err := service.RegisterProject(ctx, admin, factory.RegisterProjectRequest{RequestID: "project", BundlePath: bundle, RepositoryPath: repository, AllowedSourceRoots: []string{repository}, Policy: policy})
	if err != nil {
		t.Fatal(err)
	}
	worker := func(name string, capabilities []factory.Capability) factory.Principal {
		user, err := service.CreateUser(ctx, admin, factory.CreateUserRequest{RequestID: "user-" + name, Username: name, Kind: factory.UserKindWorker})
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
	return jobHarness{service: service, project: project, admin: admin, coordinator: admin, reviewer: admin, bundle: bundle, repository: repository, dataDir: dataDir,
		implementer: worker("implementer", []factory.Capability{factory.CapabilityRead, factory.CapabilityImplementWorker}),
		verifier:    worker("verifier", []factory.Capability{factory.CapabilityRead, factory.CapabilityVerifyWorker, factory.CapabilityVerify})}
}

// pilotUntil runs autopilot passes until the card reaches the wanted status.
func pilotUntil(t *testing.T, h jobHarness, status string) factory.BoardCard {
	t.Helper()
	var card factory.BoardCard
	for i := 0; i < 300; i++ {
		h.service.PilotOnce(context.Background())
		card = cardFor(t, h.service, h.admin, h.project.ID, "work-test")
		if card.Status == status {
			return card
		}
		sleep()
	}
	t.Fatalf("card never reached %q: %+v", status, card)
	return card
}

func currentJob(t *testing.T, h jobHarness) factory.Job {
	t.Helper()
	card := cardFor(t, h.service, h.admin, h.project.ID, "work-test")
	if card.JobID == "" {
		t.Fatalf("no job on the card: %+v", card)
	}
	job, err := h.service.GetJob(context.Background(), h.admin, card.JobID)
	if err != nil {
		t.Fatal(err)
	}
	return job
}

func delegated(job factory.Job, kind string) bool {
	return hasEvent(job, kind, `"delegated":"autopilot:`)
}

func TestAutopilotLeavesProjectsAloneUntilAPersonSavesAPolicy(t *testing.T) {
	t.Parallel()
	h := setupPilot(t, coveringAuthor(t, ""), nil)
	h.service.PilotOnce(context.Background())
	card := cardFor(t, h.service, h.admin, h.project.ID, "work-test")
	if card.Column != factory.ColumnNext || card.State != factory.CardWaiting || card.Decision != factory.CardDecisionPlan {
		t.Fatalf("without a policy the autopilot must not start work: %+v", card)
	}
}

func TestAutopilotRunsTheLoopAndAsksOnlyWhereThePolicySays(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	h := setupPilot(t, coveringAuthor(t, ""), nil)
	if _, err := h.service.SetAskPolicy(ctx, h.admin, factory.SetAskPolicyRequest{RequestID: "ask", ProjectID: h.project.ID, BeforeBuild: factory.AskBeforeNever, AfterBuild: factory.AskAfterWhenUntested, Autopilot: true}); err != nil {
		t.Fatal(err)
	}
	if card := cardFor(t, h.service, h.admin, h.project.ID, "work-test"); card.Status != "Starts by itself" {
		t.Fatalf("ready card under an autopilot policy = %+v", card)
	}
	pilotUntil(t, h, "Waiting for a builder")
	job := currentJob(t, h)
	if job.Admission.ExpectationsCommit == "" || !delegated(job, "job_admitted") {
		t.Fatalf("the build must start from the locked tests, as a delegated decision: %+v", job.Admission)
	}
	h.job = job
	// The first check fails; the autopilot asks for a fix by itself.
	completeImplementationForTest(t, h, "first")
	job = runChecks(t, h, "first", factory.CheckFailed, false)
	if job.Status != factory.JobVerificationFailed {
		t.Fatalf("job after a failed check = %s", job.Status)
	}
	if card := cardFor(t, h.service, h.admin, h.project.ID, "work-test"); card.Status != "Fixing failed checks" || card.State != factory.CardRunning {
		t.Fatalf("failed check under the autopilot = %+v", card)
	}
	h.service.PilotOnce(ctx)
	job = currentJob(t, h)
	if job.RepairCount != 1 || !delegated(job, "job_action") {
		t.Fatalf("the autopilot did not request the fix: repairs %d, status %s", job.RepairCount, job.Status)
	}
	completeImplementationForTest(t, h, "second")
	job = runChecks(t, h, "second", factory.CheckPassed, false)
	if job.Status != factory.JobReadyForReview {
		t.Fatalf("job after the fix = %s", job.Status)
	}
	// Every requirement has a locked test, so the policy approves and merges.
	if card := cardFor(t, h.service, h.admin, h.project.ID, "work-test"); card.Status != "Approving by your rules" || card.Unchecked != 0 {
		t.Fatalf("covered job under when_untested = %+v", card)
	}
	h.service.PilotOnce(ctx)
	job = currentJob(t, h)
	if job.Status != factory.JobAccepted || job.Merge == nil || !delegated(job, "acceptance_published") {
		t.Fatalf("the policy did not approve: %s, %+v", job.Status, job.Merge)
	}
	runMergeChecks(t, h, "merge", factory.CheckPassed)
	if card := cardFor(t, h.service, h.admin, h.project.ID, "work-test"); card.Column != factory.ColumnDone {
		t.Fatalf("merged card = %+v", card)
	}
	metrics, err := h.service.Metrics(ctx, h.admin, h.project.ID)
	if err != nil {
		t.Fatal(err)
	}
	// Accepting the tests, starting the build, and one approval. Asking for a fix is not a decision.
	if metrics.HumanDecisions != 0 || metrics.DelegatedDecisions != 3 || metrics.MergedJobs != 1 {
		t.Fatalf("decisions: human %d, delegated %d, merged %d", metrics.HumanDecisions, metrics.DelegatedDecisions, metrics.MergedJobs)
	}
}

func TestAutopilotStopsWhenTheTestAuthorRaisesAConcern(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	h := setupPilot(t, coveringAuthor(t, "The example contradicts the rule."), nil)
	if _, err := h.service.SetAskPolicy(ctx, h.admin, factory.SetAskPolicyRequest{RequestID: "ask", ProjectID: h.project.ID, BeforeBuild: factory.AskBeforeNever, AfterBuild: factory.AskAfterAlways, Autopilot: true}); err != nil {
		t.Fatal(err)
	}
	card := pilotUntil(t, h, "Read the locked tests")
	if card.State != factory.CardNeedsYou || !strings.Contains(card.Line, "found 1 problem") {
		t.Fatalf("a concern must stop the autopilot: %+v", card)
	}
	h.service.PilotOnce(ctx)
	if again := cardFor(t, h.service, h.admin, h.project.ID, "work-test"); again.Status != "Read the locked tests" || again.JobID != "" {
		t.Fatalf("the autopilot went past a concern: %+v", again)
	}
}

func TestGoAheadOnThePlanBuildsWhenTheTestsAreClean(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	h := setupPilot(t, coveringAuthor(t, ""), nil)
	result, err := h.service.ApprovePlan(ctx, h.admin, factory.ApprovePlanRequest{RequestID: "agree", ProjectID: h.project.ID, WorkItemID: "work-test", Continue: true})
	if err != nil || result.Expectation == nil {
		t.Fatalf("plan = %+v, %v", result, err)
	}
	pilotUntil(t, h, "Waiting for a builder")
	job := currentJob(t, h)
	if !delegated(job, "job_admitted") || job.Admission.ExpectationsCommit == "" {
		t.Fatalf("the go-ahead did not start the build from the tests: %+v", job.Admission)
	}
	expectation, err := h.service.GetExpectations(ctx, h.admin, result.Expectation.ID)
	if err != nil || expectation.Status != factory.ExpectationAccepted || !strings.HasPrefix(expectation.Delegated, "autopilot:") || len(expectation.Coverage) != 1 {
		t.Fatalf("delegated test acceptance = %+v, %v", expectation, err)
	}
}
