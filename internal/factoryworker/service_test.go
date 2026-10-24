package factoryworker

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Zokiio/context/internal/factory"
)

type serviceClient struct {
	s     *factory.Service
	actor factory.Principal
}

func (c serviceClient) Claim(ctx context.Context, r factory.WorkerClaimRequest) (factory.Assignment, error) {
	return c.s.Claim(ctx, c.actor, r)
}
func (c serviceClient) Heartbeat(ctx context.Context, r factory.HeartbeatRequest) (factory.HeartbeatResult, error) {
	return c.s.Heartbeat(ctx, c.actor, r)
}
func (c serviceClient) Report(ctx context.Context, r factory.WorkerReportRequest) (factory.Job, error) {
	return c.s.Report(ctx, c.actor, r)
}
func TestServiceWorkerRunsStandardFixtureAndIndependentCheck(t *testing.T) {
	ctx := context.Background()
	service, e := factory.Open(t.TempDir(), factory.Options{})
	if e != nil {
		t.Fatal(e)
	}
	defer service.Close()
	boot, e := service.Bootstrap(ctx, factory.BootstrapRequest{Username: "admin"})
	if e != nil {
		t.Fatal(e)
	}
	admin, e := service.Authenticate(ctx, boot.Token)
	if e != nil {
		t.Fatal(e)
	}
	dir, _ := repo(t)
	bundle := filepath.Join(dir, "records")
	if e = os.MkdirAll(bundle, 0700); e != nil {
		t.Fatal(e)
	}
	projectText := "---\ntype: Project\nid: fixture-project\ntitle: Offline fixture project\n---\n\n## Goals\nExercise offline worker execution.\n\n## Current commitments\n- [Fixture](ticket.md)\n\n## Open decisions\nNone\n"
	ticketText := "---\ntype: WorkItem\nid: fixture-work\ntitle: Produce offline fixture candidate\ntriage: ready-for-agent\nexecution: unstarted\n---\n\n## Acceptance criteria\n- Produce the offline fixture file in a real candidate commit.\n\n## Blocked by\nNone\n\n## Blocked by decisions\nNone\n\n## Acceptance\n\n"
	for name, text := range map[string]string{"project.md": projectText, "ticket.md": ticketText} {
		if e = os.WriteFile(filepath.Join(bundle, name), []byte(text), 0600); e != nil {
			t.Fatal(e)
		}
	}
	for _, args := range [][]string{{"add", "records"}, {"commit", "-m", "fixture requirements"}} {
		if _, e = git(dir, args...); e != nil {
			t.Fatal(e)
		}
	}
	base, e := git(dir, "rev-parse", "HEAD")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = service.RegisterExecutor(ctx, admin, factory.RegisterExecutorRequest{RequestID: "executor", Key: "fixture-agent", DisplayName: "Offline fixture agent", Command: helper("standard-fixture").Argv, Fixture: true}); e != nil {
		t.Fatal(e)
	}
	if _, e = service.RegisterCheck(ctx, admin, factory.RegisterCheckRequest{RequestID: "check", Key: "fixture-check", DisplayName: "Offline fixture check", Command: helper("standard-check").Argv, Environment: "offline fixture", Fixture: true}); e != nil {
		t.Fatal(e)
	}
	project, e := service.RegisterProject(ctx, admin, factory.RegisterProjectRequest{RequestID: "project", BundlePath: bundle, RepositoryPath: dir, AllowedSourceRoots: []string{dir}, ReaderLimits: factory.ReaderLimits{MaxFiles: 200, MaxBytes: 2 << 20}, Policy: factory.ProjectPolicy{RequiredChecks: []string{"fixture-check"}, AllowedExecutors: []string{"fixture-agent"}, MaxRepairAttempts: 1, MaxJobDurationSeconds: 3600, Fixture: true}})
	if e != nil {
		t.Fatal(e)
	}
	workerUser := func(name string, cap factory.Capability) factory.Principal {
		t.Helper()
		u, e := service.CreateUser(ctx, admin, factory.CreateUserRequest{RequestID: "user-" + name, Username: name, Kind: factory.UserKindWorker})
		if e != nil {
			t.Fatal(e)
		}
		token, e := service.IssueToken(ctx, admin, factory.IssueTokenRequest{RequestID: "token-" + name, UserID: u.ID, Label: "test"})
		if e != nil {
			t.Fatal(e)
		}
		p, e := service.Authenticate(ctx, token.Token)
		if e != nil {
			t.Fatal(e)
		}
		if _, e = service.GrantProject(ctx, admin, factory.GrantProjectRequest{RequestID: "grant-" + name, ProjectID: project.ID, UserID: u.ID, Capabilities: []factory.Capability{factory.CapabilityRead, cap}}); e != nil {
			t.Fatal(e)
		}
		return p
	}
	implementer := workerUser("implementer", factory.CapabilityImplementWorker)
	verifier := workerUser("verifier", factory.CapabilityVerifyWorker)
	job, e := service.Submit(ctx, admin, factory.SubmitRequest{RequestID: "submit", ProjectID: project.ID, WorkItemID: "fixture-work", BaseRevision: base, ExecutorKey: "fixture-agent", Deadline: time.Now().UTC().Add(30 * time.Minute), RepairLimit: 1})
	if e != nil {
		t.Fatal(e)
	}
	implementation := Worker{Client: serviceClient{service, implementer}, Process: &Process{ReceiptDir: t.TempDir()}, Stage: factory.StageImplementation}
	receipt, e := implementation.Once(ctx)
	if e != nil || receipt.State != "succeeded" || receipt.Candidate == base {
		t.Fatalf("implementation %+v %v", receipt, e)
	}
	verification := Worker{Client: serviceClient{service, verifier}, Process: &Process{ReceiptDir: t.TempDir()}, Stage: factory.StageVerification}
	checked, e := verification.Once(ctx)
	if e != nil || checked.State != "succeeded" || checked.Candidate != receipt.Candidate {
		t.Fatalf("verification %+v %v", checked, e)
	}
	actual, e := service.GetJob(ctx, admin, job.ID)
	if e != nil {
		t.Fatal(e)
	}
	if actual.Status != factory.JobReadyForReview || actual.CandidateRevision != receipt.Candidate || len(actual.Checks) != 1 || actual.Checks[0].Status != factory.CheckPassed || actual.Checks[0].ObserverID == nil || *actual.Checks[0].ObserverID != verifier.UserID {
		t.Fatalf("actual job %+v", actual)
	}
	if actual.Attempts[0].WorkspacePath == actual.Attempts[1].WorkspacePath {
		t.Fatal("verification reused implementation workspace")
	}
	if head, _ := git(dir, "rev-parse", "HEAD"); head != base {
		t.Fatal("source checkout revision changed")
	}
	if _, e = os.Stat(filepath.Join(dir, "waymark-fixture.txt")); !os.IsNotExist(e) {
		t.Fatal("worker changed source checkout")
	}
	if len(actual.Checks[0].Artifacts) == 0 {
		t.Fatal("actual verification evidence not retained")
	}
}
