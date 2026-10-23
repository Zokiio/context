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

// fakeAuthor writes files into its working directory (the expectation worktree) and
// reports notes.
func fakeAuthor(t *testing.T, files map[string]string) []string {
	t.Helper()
	dir := t.TempDir()
	script := "#!/bin/sh\ncat > /dev/null\n"
	for name, content := range files {
		script += "mkdir -p \"$(dirname '" + name + "')\"\nprintf '%s' '" + content + "' > '" + name + "'\n"
	}
	script += `echo '{"notes":"Covered every criterion.","run":{"model":"fake-author","sessionId":"sess-e","turns":3,"costUsd":0.02}}'` + "\n"
	path := filepath.Join(dir, "author")
	if err := os.WriteFile(path, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	return []string{path}
}

type expectationHarness struct {
	jobHarness
}

func setupExpectations(t *testing.T, author []string, mutate func(*factory.ProjectPolicy)) expectationHarness {
	t.Helper()
	ctx := context.Background()
	dataDir := t.TempDir()
	service, err := factory.Open(dataDir, factory.Options{LeaseDuration: time.Minute, ExpectationCommand: author})
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
	policy := factory.ProjectPolicy{RequiredChecks: []string{"go-test"}, AllowedExecutors: []string{"fixture-agent"}, MaxRepairAttempts: 2, MaxJobDurationSeconds: 3600, Fixture: true, ProtectedPaths: []string{"tests/"}}
	if mutate != nil {
		mutate(&policy)
	}
	project, err := service.RegisterProject(ctx, admin, factory.RegisterProjectRequest{RequestID: "project", BundlePath: bundle, RepositoryPath: repository, AllowedSourceRoots: []string{repository}, Policy: policy})
	if err != nil {
		t.Fatal(err)
	}
	principal := func(name string, kind factory.UserKind, capabilities []factory.Capability) factory.Principal {
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
		p, err := service.Authenticate(ctx, token.Token)
		if err != nil {
			t.Fatal(err)
		}
		return p
	}
	return expectationHarness{jobHarness{service: service, project: project, admin: admin, coordinator: admin, bundle: bundle, repository: repository, dataDir: dataDir,
		implementer: principal("implementer", factory.UserKindWorker, []factory.Capability{factory.CapabilityRead, factory.CapabilityImplementWorker}),
		verifier:    principal("verifier", factory.UserKindWorker, []factory.Capability{factory.CapabilityRead, factory.CapabilityVerifyWorker, factory.CapabilityVerify}),
		reviewer:    principal("reviewer", factory.UserKindHuman, []factory.Capability{factory.CapabilityRead, factory.CapabilityReview, factory.CapabilityAccept})}}
}

func (h expectationHarness) write(t *testing.T, id string) factory.Expectation {
	t.Helper()
	started, err := h.service.StartExpectations(context.Background(), h.admin, factory.StartExpectationsRequest{RequestID: id, ProjectID: h.project.ID, WorkItemID: "work-test"})
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(15 * time.Second)
	for {
		current, err := h.service.GetExpectations(context.Background(), h.admin, started.ID)
		if err != nil {
			t.Fatal(err)
		}
		if current.Status != factory.ExpectationRunning {
			return current
		}
		if time.Now().After(deadline) {
			t.Fatalf("expectations still running: %+v", current)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func TestHoldoutExpectationsBecomeTheJobBaseAndStayOutOfReach(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	h := setupExpectations(t, fakeAuthor(t, map[string]string{"tests/holdout_test.txt": "expects the change\n"}), nil)
	expectation := h.write(t, "write")
	if expectation.Status != factory.ExpectationProposed || len(expectation.Files) != 1 || expectation.Files[0] != "tests/holdout_test.txt" || !strings.Contains(expectation.Patch, "expects the change") || expectation.Run.Model != "fake-author" {
		t.Fatalf("expectations = %+v", expectation)
	}
	if !strings.Contains(runGit(t, h.repository, "for-each-ref", "refs/waymark/expectations"), expectation.Commit) {
		t.Fatal("the expectation commit is not pinned")
	}
	if head(t, h.repository, "HEAD") == expectation.Commit {
		t.Fatal("writing expectations moved the project branch")
	}
	submit := factory.SubmitRequest{RequestID: "early", ProjectID: h.project.ID, WorkItemID: "work-test", ExecutorKey: "fixture-agent", Deadline: time.Now().Add(time.Hour), ExpectationsID: expectation.ID}
	if _, err := h.service.Submit(ctx, h.admin, submit); !isCode(err, factory.CodeNotReady) {
		t.Fatalf("submit with unaccepted expectations = %v", err)
	}
	if _, err := h.service.AcceptExpectations(ctx, h.implementer, factory.DecideExpectationsRequest{RequestID: "worker-accept", ExpectationID: expectation.ID}); err == nil {
		t.Fatal("a worker accepted expectations")
	}
	accepted, err := h.service.AcceptExpectations(ctx, h.admin, factory.DecideExpectationsRequest{RequestID: "accept", ExpectationID: expectation.ID})
	if err != nil || accepted.Status != factory.ExpectationAccepted {
		t.Fatalf("accept = %+v, %v", accepted, err)
	}
	submit.RequestID = "submit"
	job, err := h.service.Submit(ctx, h.admin, submit)
	if err != nil {
		t.Fatal(err)
	}
	if job.Admission.BaseRevision != expectation.Commit || job.Admission.ExpectationsCommit != expectation.Commit {
		t.Fatalf("job base = %s, expectations = %s", job.Admission.BaseRevision, job.Admission.ExpectationsCommit)
	}
	h.job = job
	assignment, err := h.service.Claim(ctx, h.implementer, factory.WorkerClaimRequest{RequestID: "peek", Stages: []factory.Stage{factory.StageImplementation}, ProjectID: &h.project.ID})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(assignment.Instructions, "Holdout tests") {
		t.Fatalf("implementer was not told about the holdout tests: %q", assignment.Instructions)
	}
	if _, err := os.Stat(filepath.Join(assignment.WorkspacePath, "tests", "holdout_test.txt")); err != nil {
		t.Fatalf("the implementation workspace lacks the holdout tests: %v", err)
	}
	rejected, _ := implementFilesForTest(t, h.jobHarness, "weaken", map[string]string{"tests/holdout_test.txt": "expects nothing\n", "change.txt": "done\n"})
	if rejected.Status != factory.JobFailed || !hasEvent(rejected, "protected_paths_changed", "tests/holdout_test.txt") {
		t.Fatalf("weakening the holdout tests was not rejected: %s", rejected.Status)
	}
}

func TestExpectationAuthorsMayOnlyWriteProtectedTests(t *testing.T) {
	t.Parallel()
	outside := setupExpectations(t, fakeAuthor(t, map[string]string{"tests/ok_test.txt": "ok\n", "src/impl.go": "package src\n"}), nil)
	if failed := outside.write(t, "outside"); failed.Status != factory.ExpectationFailed || !strings.Contains(failed.Error, "src/impl.go") {
		t.Fatalf("author writing outside protected paths = %+v", failed)
	}
	nothing := setupExpectations(t, fakeAuthor(t, nil), nil)
	if failed := nothing.write(t, "nothing"); failed.Status != factory.ExpectationFailed || !strings.Contains(failed.Error, "no tests") {
		t.Fatalf("author writing nothing = %+v", failed)
	}
	unprotected := setupExpectations(t, fakeAuthor(t, nil), func(policy *factory.ProjectPolicy) { policy.ProtectedPaths = nil })
	if _, err := unprotected.service.StartExpectations(context.Background(), unprotected.admin, factory.StartExpectationsRequest{RequestID: "x", ProjectID: unprotected.project.ID, WorkItemID: "work-test"}); !isCode(err, factory.CodeNotReady) {
		t.Fatalf("expectations without protected paths = %v", err)
	}
}

func TestPolicyCanRequireExpectationsAndCriteriaChangesStaleThem(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	h := setupExpectations(t, fakeAuthor(t, map[string]string{"tests/a_test.txt": "a\n"}), func(policy *factory.ProjectPolicy) { policy.RequireExpectations = true })
	base := head(t, h.repository, "HEAD")
	if _, err := h.service.Submit(ctx, h.admin, factory.SubmitRequest{RequestID: "bare", ProjectID: h.project.ID, WorkItemID: "work-test", BaseRevision: base, ExecutorKey: "fixture-agent", Deadline: time.Now().Add(time.Hour)}); !isCode(err, factory.CodeNotReady) {
		t.Fatalf("submit without required expectations = %v", err)
	}
	expectation := h.write(t, "write")
	ticket := filepath.Join(h.bundle, "work.md")
	raw, _ := os.ReadFile(ticket)
	if err := os.WriteFile(ticket, []byte(strings.Replace(string(raw), "- The change works.", "- The change works quickly.", 1)), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := h.service.AcceptExpectations(ctx, h.admin, factory.DecideExpectationsRequest{RequestID: "accept", ExpectationID: expectation.ID}); !isCode(err, factory.CodeStale) {
		t.Fatalf("accepting expectations for changed criteria = %v", err)
	}
	rejected, err := h.service.RejectExpectations(ctx, h.admin, factory.DecideExpectationsRequest{RequestID: "reject", ExpectationID: expectation.ID})
	if err != nil || rejected.Status != factory.ExpectationRejected {
		t.Fatalf("reject = %+v, %v", rejected, err)
	}
	invalid := factory.RegisterProjectRequest{RequestID: "invalid", BundlePath: h.bundle, RepositoryPath: h.repository, AllowedSourceRoots: []string{h.repository}, Policy: factory.ProjectPolicy{RequiredChecks: []string{"go-test"}, AllowedExecutors: []string{"fixture-agent"}, MaxRepairAttempts: 1, MaxJobDurationSeconds: 60, Fixture: true, RequireExpectations: true}}
	if _, err := h.service.RegisterProject(ctx, h.admin, invalid); !isCode(err, factory.CodeInvalid) {
		t.Fatalf("requiring expectations without protected paths = %v", err)
	}
}
