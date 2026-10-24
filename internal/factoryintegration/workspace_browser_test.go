package factoryintegration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/Zokiio/context/internal/factory"
	"github.com/Zokiio/context/internal/factoryhttp"
	"github.com/Zokiio/context/internal/factoryworker"
)

// TestWorkspaceBrowserProcess is an offline subprocess, never a model provider.
// The browser fixture controls one failed check through a private marker file.
func TestWorkspaceBrowserProcess(t *testing.T) {
	for _, arg := range os.Args {
		if arg != "workspace-build" && arg != "workspace-check" {
			continue
		}
		raw, err := io.ReadAll(io.LimitReader(os.Stdin, 16<<20))
		if err == nil && arg == "workspace-build" {
			if path := os.Getenv("CTX_FACTORY_PROGRESS"); path != "" {
				err = os.WriteFile(path, []byte("Reading the admitted requirements\nPreparing the fixture change\n"), 0600)
			}
			fmt.Fprintln(os.Stdout, "Preparing a real candidate in the isolated fixture worktree")
			if err == nil {
				err = factoryworker.Fixture(context.Background(), bytes.NewReader(raw), os.Stdout, 6*time.Second, false)
			}
		} else if err == nil {
			marker := filepath.Join(os.Getenv("WAYMARK_BROWSER_FIXTURE_DIR"), "fail-next-check")
			if _, statErr := os.Stat(marker); statErr == nil {
				_ = os.Remove(marker)
				err = fmt.Errorf("controlled browser fixture failure: the first check needs a repair")
			} else {
				err = factoryworker.FixtureCheck(bytes.NewReader(raw), os.Stdout)
			}
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(0)
	}
}

// Start with WAYMARK_BROWSER_FIXTURE_DIR set to a private scratch directory.
// The browser runner reads fixture.json, then writes done to shut down cleanly.
// The normal Go suite skips this manual/browser fixture.
func TestWorkspaceBrowserFixture(t *testing.T) {
	dir := os.Getenv("WAYMARK_BROWSER_FIXTURE_DIR")
	if dir == "" {
		t.Skip("set WAYMARK_BROWSER_FIXTURE_DIR for the browser acceptance fixture")
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	service, err := factory.Open(filepath.Join(dir, "state"), factory.Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer service.Close()
	boot, err := service.Bootstrap(ctx, factory.BootstrapRequest{Username: "workspace-operator"})
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(factoryhttp.NewHandler(service))
	defer server.Close()
	admin := client(t, server.URL, boot.Token)
	repo, bundle, _ := repositoryFixture(t)
	if err := os.WriteFile(filepath.Join(repo, ".gitignore"), []byte(".factory-evidence/\nrecords/acceptances/\n"), 0600); err != nil {
		t.Fatal(err)
	}
	git(t, repo, "add", ".gitignore")
	git(t, repo, "commit", "-m", "Ignore generated acceptance evidence")
	call[factory.ExecutorProfile](t, admin, "POST", "/v1/executors", factory.RegisterExecutorRequest{
		RequestID: "browser-executor", Key: "offline-browser", DisplayName: "Offline browser fixture", Fixture: true,
		Command: []string{os.Args[0], "-test.run=^TestWorkspaceBrowserProcess$", "--", "workspace-build"},
	})
	call[factory.CheckProfile](t, admin, "POST", "/v1/checks", factory.RegisterCheckRequest{
		RequestID: "browser-check", Key: "offline-browser-check", DisplayName: "Check the candidate", Fixture: true, Environment: "disposable browser fixture",
		Command: []string{os.Args[0], "-test.run=^TestWorkspaceBrowserProcess$", "--", "workspace-check"},
	})
	project := call[factory.Project](t, admin, "POST", "/v1/projects", factory.RegisterProjectRequest{
		RequestID: "browser-project", BundlePath: bundle, RepositoryPath: repo, AllowedSourceRoots: []string{repo},
		Policy: factory.ProjectPolicy{RequiredChecks: []string{"offline-browser-check"}, AllowedExecutors: []string{"offline-browser"}, MaxRepairAttempts: 2, MaxJobDurationSeconds: 600, Fixture: true, AllowCoordinatorReview: true},
	})
	var workers sync.WaitGroup
	for _, role := range []struct {
		name       string
		stage      factory.Stage
		capability factory.Capability
	}{{"builder", factory.StageImplementation, factory.CapabilityImplementWorker}, {"verifier", factory.StageVerification, factory.CapabilityVerifyWorker}} {
		user := call[factory.User](t, admin, "POST", "/v1/users", factory.CreateUserRequest{RequestID: role.name, Username: role.name, Kind: factory.UserKindWorker})
		token := call[factory.IssuedToken](t, admin, "POST", "/v1/users/"+string(user.ID)+"/tokens", factory.IssueTokenRequest{RequestID: role.name + "-token", UserID: user.ID, Label: "browser fixture"})
		call[factory.Grant](t, admin, "PUT", "/v1/projects/"+string(project.ID)+"/grants/"+string(user.ID), factory.GrantProjectRequest{RequestID: role.name + "-grant", ProjectID: project.ID, UserID: user.ID, Capabilities: []factory.Capability{factory.CapabilityRead, role.capability}})
		worker := &factoryworker.Worker{Client: client(t, server.URL, token.Token), Process: &factoryworker.Process{ReceiptDir: filepath.Join(dir, role.name)}, Stage: role.stage, ProjectID: &project.ID, PollInterval: 200 * time.Millisecond, HeartbeatInterval: 500 * time.Millisecond}
		workers.Add(1)
		go func() { defer workers.Done(); _ = worker.Loop(ctx, nil) }()
	}
	defer func() { cancel(); workers.Wait() }()
	tokenPath := filepath.Join(dir, "operator.token")
	if err := os.WriteFile(tokenPath, []byte(boot.Token), 0600); err != nil {
		t.Fatal(err)
	}
	config, _ := json.Marshal(map[string]string{"url": server.URL, "tokenFile": tokenPath, "projectId": string(project.ID), "workItemId": "ec352583-bd7e-4a59-9576-46e129c76c6b", "repository": repo})
	if err := os.WriteFile(filepath.Join(dir, "fixture.json"), config, 0600); err != nil {
		t.Fatal(err)
	}
	deadline := time.NewTimer(25 * time.Minute)
	defer deadline.Stop()
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-deadline.C:
			t.Fatal("browser fixture timed out waiting for done")
		case <-ticker.C:
			if _, err := os.Stat(filepath.Join(dir, "done")); err == nil {
				return
			}
		}
	}
}
