package factoryworker

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/Zokiio/context/internal/factory"
	"github.com/Zokiio/context/internal/factoryclient"
	"github.com/Zokiio/context/internal/factoryhttp"
)

func TestLostTerminalReportCanBeReplayedAfterServiceRestart(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	dataDir := t.TempDir()
	service, err := factory.Open(dataDir, factory.Options{})
	if err != nil {
		t.Fatal(err)
	}
	var server *httptest.Server
	t.Cleanup(func() {
		if server != nil {
			server.Close()
		}
		if service != nil {
			_ = service.Close()
		}
	})
	boot, err := service.Bootstrap(ctx, factory.BootstrapRequest{Username: "admin"})
	if err != nil {
		t.Fatal(err)
	}
	admin, err := service.Authenticate(ctx, boot.Token)
	if err != nil {
		t.Fatal(err)
	}
	dir, _ := repo(t)
	bundle := filepath.Join(dir, "records")
	if err = os.MkdirAll(bundle, 0700); err != nil {
		t.Fatal(err)
	}
	projectText := "---\ntype: Project\nid: fixture-project\ntitle: Offline fixture project\n---\n\n## Goals\nExercise offline worker execution.\n\n## Current commitments\n- [Fixture](ticket.md)\n\n## Open decisions\nNone\n"
	ticketText := "---\ntype: WorkItem\nid: fixture-work\ntitle: Produce offline fixture candidate\ntriage: ready-for-agent\nexecution: unstarted\n---\n\n## Acceptance criteria\n- Produce the offline fixture file in a real candidate commit.\n\n## Blocked by\nNone\n\n## Blocked by decisions\nNone\n\n## Acceptance\n\n"
	for name, text := range map[string]string{"project.md": projectText, "ticket.md": ticketText} {
		if err = os.WriteFile(filepath.Join(bundle, name), []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, args := range [][]string{{"add", "records"}, {"commit", "-m", "fixture requirements"}} {
		if _, err = git(dir, args...); err != nil {
			t.Fatal(err)
		}
	}
	base, err := git(dir, "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = service.RegisterExecutor(ctx, admin, factory.RegisterExecutorRequest{RequestID: "executor", Key: "fixture-agent", DisplayName: "Offline fixture agent", Command: helper("standard-fixture").Argv, Fixture: true}); err != nil {
		t.Fatal(err)
	}
	if _, err = service.RegisterCheck(ctx, admin, factory.RegisterCheckRequest{RequestID: "check", Key: "fixture-check", DisplayName: "Offline fixture check", Command: helper("standard-check").Argv, Environment: "offline fixture", Fixture: true}); err != nil {
		t.Fatal(err)
	}
	project, err := service.RegisterProject(ctx, admin, factory.RegisterProjectRequest{RequestID: "project", BundlePath: bundle, RepositoryPath: dir, AllowedSourceRoots: []string{dir}, ReaderLimits: factory.ReaderLimits{MaxFiles: 200, MaxBytes: 2 << 20}, Policy: factory.ProjectPolicy{RequiredChecks: []string{"fixture-check"}, AllowedExecutors: []string{"fixture-agent"}, MaxRepairAttempts: 1, MaxJobDurationSeconds: 3600, Fixture: true}})
	if err != nil {
		t.Fatal(err)
	}
	user, err := service.CreateUser(ctx, admin, factory.CreateUserRequest{RequestID: "implementer", Username: "implementer", Kind: factory.UserKindWorker})
	if err != nil {
		t.Fatal(err)
	}
	credential, err := service.IssueToken(ctx, admin, factory.IssueTokenRequest{RequestID: "implementer-token", UserID: user.ID, Label: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = service.GrantProject(ctx, admin, factory.GrantProjectRequest{RequestID: "implementer-grant", ProjectID: project.ID, UserID: user.ID, Capabilities: []factory.Capability{factory.CapabilityRead, factory.CapabilityImplementWorker}}); err != nil {
		t.Fatal(err)
	}
	job, err := service.Submit(ctx, admin, factory.SubmitRequest{RequestID: "submit", ProjectID: project.ID, WorkItemID: "fixture-work", BaseRevision: base, ExecutorKey: "fixture-agent", Deadline: time.Now().UTC().Add(30 * time.Minute), RepairLimit: 1})
	if err != nil {
		t.Fatal(err)
	}

	var mu sync.Mutex
	var captured factory.WorkerReportRequest
	var rejected int
	handler := factoryhttp.NewHandler(service)
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/worker/reports" {
			raw, readErr := io.ReadAll(r.Body)
			_ = r.Body.Close()
			if readErr != nil {
				http.Error(w, "cannot read report", http.StatusBadRequest)
				return
			}
			r.Body = io.NopCloser(bytes.NewReader(raw))
			var report factory.WorkerReportRequest
			if json.Unmarshal(raw, &report) == nil && report.Kind == factory.ReportTerminal {
				mu.Lock()
				captured = report
				rejected++
				mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusServiceUnavailable)
				_, _ = w.Write([]byte(`{"code":"internal","message":"injected terminal transport outage"}`))
				return
			}
		}
		handler.ServeHTTP(w, r)
	}))
	client, err := factoryclient.New(server.URL, credential.Token)
	if err != nil {
		t.Fatal(err)
	}
	receipts := t.TempDir()
	worker := Worker{Client: client, Process: &Process{ReceiptDir: receipts}, Stage: factory.StageImplementation}
	actual, err := worker.Once(ctx)
	mu.Lock()
	terminalReport, rejectedCount := captured, rejected
	mu.Unlock()
	if err == nil || actual.State != "succeeded" || rejectedCount == 0 {
		t.Fatalf("lost terminal delivery: receipt=%+v error=%v rejected=%d", actual, err, rejectedCount)
	}
	saved, err := worker.Process.Read(actual.AttemptID)
	if err != nil {
		t.Fatal(err)
	}
	if saved.State != "succeeded" || saved.Candidate == "" || saved.Candidate == base {
		t.Fatalf("missing durable candidate receipt: %+v", saved)
	}
	server.Close()
	server = nil
	if err = service.Close(); err != nil {
		t.Fatal(err)
	}
	service = nil
	service, err = factory.Open(dataDir, factory.Options{})
	if err != nil {
		t.Fatal(err)
	}
	server = httptest.NewServer(factoryhttp.NewHandler(service))
	client, err = factoryclient.New(server.URL, credential.Token)
	if err != nil {
		t.Fatal(err)
	}
	worker = Worker{Client: client, Process: &Process{ReceiptDir: receipts}, Stage: factory.StageImplementation}
	_, err = worker.Once(ctx)
	var apiError *factoryclient.APIError
	if !errors.As(err, &apiError) || apiError.Code != string(factory.CodeUncertain) {
		t.Fatalf("restarted worker must reconcile before claiming: %v", err)
	}
	adminClient, err := factoryclient.New(server.URL, boot.Token)
	if err != nil {
		t.Fatal(err)
	}
	retained, err := adminClient.GetJob(ctx, job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if retained.Status != factory.JobImplementationUnknown {
		t.Fatalf("restart status = %s", retained.Status)
	}
	retained, err = adminClient.Action(ctx, factory.JobActionRequest{RequestID: "reconcile", JobID: job.ID, IfVersion: retained.Version, Action: factory.ActionReconcile})
	if err != nil {
		t.Fatal(err)
	}
	if retained.Status != factory.JobImplementationUnknown {
		t.Fatalf("reconciliation request inferred an outcome: %s", retained.Status)
	}

	// Replay the actual saved observation without resolving success as a failure
	// or authorizing another invocation of the implementation command.
	raw, err := json.Marshal(saved)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != terminalReport.TerminalReceipt {
		t.Fatal("retained receipt differs from the undelivered terminal observation")
	}
	recovered, err := client.Report(ctx, terminalReport)
	if err != nil {
		t.Fatal(err)
	}
	if recovered.Status != factory.JobVerifying || recovered.CandidateRevision != saved.Candidate {
		t.Fatalf("terminal replay did not recover the exact candidate: %+v", recovered)
	}
	var implementations []factory.Attempt
	for _, attempt := range recovered.Attempts {
		if attempt.Stage == factory.StageImplementation {
			implementations = append(implementations, attempt)
		}
	}
	if len(implementations) != 1 {
		t.Fatalf("implementation attempts = %d, want one", len(implementations))
	}
	head, err := git(implementations[0].WorkspacePath, "rev-parse", "HEAD")
	if err != nil || head != saved.Candidate {
		t.Fatalf("candidate changed during recovery: %s, %v", head, err)
	}
}
