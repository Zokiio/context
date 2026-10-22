package factoryintegration

import (
	"context"
	"errors"
	"fmt"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Zokiio/context/internal/factory"
	"github.com/Zokiio/context/internal/factoryclient"
	"github.com/Zokiio/context/internal/factoryhttp"
	"github.com/Zokiio/context/internal/factoryworker"
)

// The subprocess uses the real offline adapter. Its command, commit and check
// receipt cross the same HTTP boundary as an enrolled external worker.
func TestFactoryFixtureProcess(t *testing.T) {
	for _, arg := range os.Args {
		var err error
		switch arg {
		case "factory-fixture-implement":
			err = factoryworker.Fixture(context.Background(), os.Stdin, os.Stdout, 0, false)
		case "factory-fixture-verify":
			err = factoryworker.FixtureCheck(os.Stdin, os.Stdout)
		default:
			continue
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(0)
	}
}

func TestSharedPlatformProducesReaderCompatibleAcceptance(t *testing.T) {
	ctx := context.Background()
	data := t.TempDir()
	service, err := factory.Open(data, factory.Options{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = service.Close() })
	bootstrap, err := service.Bootstrap(ctx, factory.BootstrapRequest{Username: "admin"})
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(factoryhttp.NewHandler(service))
	t.Cleanup(server.Close)
	admin := client(t, server.URL, bootstrap.Token)

	type account struct {
		user   factory.User
		client *factoryclient.Client
	}
	newAccount := func(name string, kind factory.UserKind) account {
		user := call[factory.User](t, admin, "POST", "/v1/users", factory.CreateUserRequest{RequestID: "user-" + name, Username: name, Kind: kind})
		token := call[factory.IssuedToken](t, admin, "POST", "/v1/users/"+string(user.ID)+"/tokens", factory.IssueTokenRequest{RequestID: "token-" + name, UserID: user.ID, Label: "offline integration"})
		return account{user, client(t, server.URL, token.Token)}
	}
	coordinator := newAccount("coordinator", factory.UserKindHuman)
	reviewer := newAccount("reviewer", factory.UserKindHuman)
	implementer := newAccount("implementer", factory.UserKindWorker)
	verifier := newAccount("verifier", factory.UserKindWorker)
	outsider := newAccount("outsider", factory.UserKindHuman)

	call[factory.ExecutorProfile](t, admin, "POST", "/v1/executors", factory.RegisterExecutorRequest{
		RequestID: "executor", Key: "offline", DisplayName: "Offline fixture", Fixture: true,
		Command: []string{os.Args[0], "-test.run=^TestFactoryFixtureProcess$", "--", "factory-fixture-implement"},
	})
	call[factory.CheckProfile](t, admin, "POST", "/v1/checks", factory.RegisterCheckRequest{
		RequestID: "check", Key: "candidate-content", DisplayName: "Actual candidate content", Fixture: true, Environment: "offline temporary repository",
		Command: []string{os.Args[0], "-test.run=^TestFactoryFixtureProcess$", "--", "factory-fixture-verify"},
	})
	repository, bundle, base := repositoryFixture(t)
	project := call[factory.Project](t, admin, "POST", "/v1/projects", factory.RegisterProjectRequest{
		RequestID: "project", BundlePath: bundle, RepositoryPath: repository, AllowedSourceRoots: []string{repository},
		Policy: factory.ProjectPolicy{RequiredChecks: []string{"candidate-content"}, AllowedExecutors: []string{"offline"}, MaxRepairAttempts: 2, MaxJobDurationSeconds: 600, Fixture: true},
	})
	grant := func(user factory.User, capabilities ...factory.Capability) {
		call[factory.Grant](t, admin, "PUT", "/v1/projects/"+string(project.ID)+"/grants/"+string(user.ID), factory.GrantProjectRequest{
			RequestID: "grant-" + string(user.ID), ProjectID: project.ID, UserID: user.ID, Capabilities: capabilities,
		})
	}
	grant(coordinator.user, factory.CapabilityRead, factory.CapabilityCoordinate, factory.CapabilityReview, factory.CapabilityAccept)
	grant(reviewer.user, factory.CapabilityRead, factory.CapabilityReview, factory.CapabilityAccept)
	grant(implementer.user, factory.CapabilityRead, factory.CapabilityImplementWorker)
	grant(verifier.user, factory.CapabilityRead, factory.CapabilityVerifyWorker)
	if projects := call[[]factory.Project](t, outsider.client, "GET", "/v1/projects", nil); len(projects) != 0 {
		t.Fatal("an ungranted user can enumerate another project's metadata")
	}
	request := factory.SubmitRequest{RequestID: "submit", ProjectID: project.ID, WorkItemID: "ec352583-bd7e-4a59-9576-46e129c76c6b", BaseRevision: base, ExecutorKey: "offline", RepairLimit: 1, Deadline: time.Now().Add(5 * time.Minute).UTC()}
	job := call[factory.Job](t, coordinator.client, "POST", "/v1/jobs", request)
	duplicate := call[factory.Job](t, coordinator.client, "POST", "/v1/jobs", request)
	if duplicate.ID != job.ID {
		t.Fatal("an idempotent submission produced a second job")
	}
	jobPath := "/v1/jobs/" + string(job.ID)
	var inaccessible factory.Job
	if err := outsider.client.DoJSON(ctx, "GET", jobPath, nil, &inaccessible); err == nil {
		t.Fatal("an ungranted user can read the job directly")
	}
	runWorker := func(account account, stage factory.Stage) {
		worker := factoryworker.Worker{Client: account.client, Process: &factoryworker.Process{ReceiptDir: t.TempDir()}, Stage: stage, ProjectID: &project.ID}
		receipt, err := worker.Once(ctx)
		if err != nil {
			t.Fatalf("%s worker: %v", stage, err)
		}
		if receipt.State != "succeeded" {
			t.Fatalf("%s receipt: %+v", stage, receipt)
		}
	}
	runWorker(implementer, factory.StageImplementation)
	job = call[factory.Job](t, reviewer.client, "GET", jobPath, nil)
	if job.Status == factory.JobAccepted || job.CandidateRevision == "" {
		t.Fatalf("worker success did not retain an unaccepted candidate: %+v", job)
	}
	premature := factory.AcceptRequest{RequestID: "premature-acceptance", JobID: job.ID, IfVersion: job.Version, CandidateRevision: job.CandidateRevision, RequirementFingerprint: job.Admission.TicketSHA256}
	if err := reviewer.client.DoJSON(ctx, "POST", jobPath+"/acceptance", premature, &factory.Job{}); err == nil {
		t.Fatal("a candidate without required verification and review was accepted")
	}
	runWorker(verifier, factory.StageVerification)
	job = call[factory.Job](t, reviewer.client, "GET", jobPath, nil)
	if job.Status != factory.JobReadyForReview {
		t.Fatalf("after verification: %s", job.Status)
	}
	selfReview := factory.ReviewRequest{RequestID: "self-review", JobID: job.ID, IfVersion: job.Version, CandidateRevision: job.CandidateRevision, RequirementFingerprint: job.Admission.TicketSHA256, Decision: factory.ReviewApproved, Summary: "not independent"}
	if err := coordinator.client.DoJSON(ctx, "POST", jobPath+"/reviews", selfReview, &factory.Job{}); err == nil {
		t.Fatal("the coordinating actor supplied its own independent review")
	}
	selfReview.RequestID = "independent-review"
	selfReview.Summary = "Verified the fixture candidate and independently observed check receipt"
	job = call[factory.Job](t, reviewer.client, "POST", jobPath+"/reviews", selfReview)
	job = call[factory.Job](t, reviewer.client, "POST", jobPath+"/acceptance", factory.AcceptRequest{RequestID: "accept", JobID: job.ID, IfVersion: job.Version, CandidateRevision: job.CandidateRevision, RequirementFingerprint: job.Admission.TicketSHA256, Summary: "Offline fixture platform acceptance"})
	if job.Status != factory.JobAccepted || job.AcceptanceRecordPath == "" {
		t.Fatalf("expected published acceptance, got %s %q", job.Status, job.AcceptanceRecordPath)
	}
	observed := call[factory.ProjectOrientation](t, reviewer.client, "GET", "/v1/projects/"+string(project.ID)+"/orientation", nil)
	var evidencePath string
	for _, item := range observed.Result.WorkItems {
		if item.ID == nil || *item.ID != request.WorkItemID {
			continue
		}
		if item.Acceptance == nil || item.Acceptance.Status != "valid" {
			t.Fatalf("reader rejected the published acceptance: %+v", item.Acceptance)
		}
		if len(item.Acceptance.Evidence) == 0 {
			t.Fatal("acceptance has no retained evidence")
		}
		evidencePath = item.Acceptance.Evidence[0].Reference.Path
	}
	if evidencePath == "" {
		t.Fatal("accepted work item was not found by the original reader")
	}
	original, err := os.ReadFile(evidencePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(evidencePath, append(append([]byte{}, original...), []byte("\nchanged evidence\n")...), 0600); err != nil {
		t.Fatal(err)
	}
	changed := call[factory.ProjectOrientation](t, reviewer.client, "GET", "/v1/projects/"+string(project.ID)+"/orientation", nil)
	for _, item := range changed.Result.WorkItems {
		if item.ID != nil && *item.ID == request.WorkItemID && (item.Acceptance == nil || item.Acceptance.Status != "stale") {
			t.Fatal("changed retained evidence did not stale historical acceptance")
		}
	}
	if err := os.WriteFile(evidencePath, original, 0600); err != nil {
		t.Fatal(err)
	}
	if head := git(t, repository, "rev-parse", "HEAD"); head != base {
		t.Fatal("worker changed the primary checkout revision")
	}
	if _, err := os.Stat(filepath.Join(repository, "waymark-fixture.txt")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("worker wrote its candidate into the primary checkout")
	}
}

func client(t *testing.T, endpoint, token string) *factoryclient.Client {
	t.Helper()
	c, err := factoryclient.New(endpoint, token)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func call[T any](t *testing.T, c *factoryclient.Client, method, path string, input any) T {
	t.Helper()
	var result T
	if err := c.DoJSON(context.Background(), method, path, input, &result); err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	return result
}

func repositoryFixture(t *testing.T) (string, string, string) {
	t.Helper()
	repo := t.TempDir()
	git(t, repo, "init", "-b", "main")
	git(t, repo, "config", "user.name", "Factory integration")
	git(t, repo, "config", "user.email", "fixture@example.invalid")
	bundle := filepath.Join(repo, "records")
	if err := os.MkdirAll(filepath.Join(bundle, "issues"), 0700); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"project.md":       "---\ntype: Project\nid: 9be2ce8f-138e-4da5-b677-8221d9da93ca\ntitle: Platform fixture\n---\n\n## Goals\n\nExercise the offline factory platform.\n\n## Current commitments\n\n- [Change](issues/change.md)\n\n## Open decisions\n\nNone\n",
		"issues/change.md": "---\ntype: WorkItem\nid: ec352583-bd7e-4a59-9576-46e129c76c6b\ntitle: Produce the offline candidate\ntriage: ready-for-agent\nexecution: unstarted\n---\n\n## Acceptance criteria\n\n- Produce a committed fixture marker and verify its actual candidate content.\n\n## Blocked by\n\nNone\n\n## Blocked by decisions\n\nNone\n\n## Acceptance\n\n",
	}
	for path, text := range files {
		if err := os.WriteFile(filepath.Join(bundle, path), []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("Offline platform fixture\n"), 0600); err != nil {
		t.Fatal(err)
	}
	git(t, repo, "add", ".")
	git(t, repo, "commit", "-m", "Create offline fixture")
	return repo, bundle, git(t, repo, "rev-parse", "HEAD")
}

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", dir}, args...)...)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, output)
	}
	return strings.TrimSpace(string(output))
}

