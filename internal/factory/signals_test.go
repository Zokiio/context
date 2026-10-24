package factory_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Zokiio/context/internal/factory"
)

// fakeDecisionModel answers like Ollama's System One endpoint: criteria containing
// "vague" are judged uncheckable.
func fakeDecisionModel(t *testing.T, calls *atomic.Int64) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			_ = json.NewEncoder(w).Encode(map[string]any{"models": []map[string]string{{"name": "kev:0.8b-mxfp8", "model": "kev:0.8b-mxfp8", "digest": "sha256:fake"}}})
		case "/v1/systemone":
			calls.Add(1)
			var request struct {
				State map[string]string `json:"state"`
			}
			_ = json.NewDecoder(r.Body).Decode(&request)
			probability := 0.8
			if strings.Contains(request.State["criterion"], "vague") {
				probability = 0.1
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"answers": map[string]any{"checkable": map[string]any{"type": "noul", "noul": probability}}})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func TestDecisionSignalsAreAdvisoryAndCached(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	var calls atomic.Int64
	server := fakeDecisionModel(t, &calls)
	service, admin, project := decisionProject(t, server.URL, "- The command exits 0 for valid input.\n- Make it vague and nice.\n")

	oriented, err := service.Orientation(ctx, admin, project.ID)
	if err != nil {
		t.Fatal(err)
	}
	signals := oriented.CriterionSignals["work-test"]
	if signals == nil || signals.Status != factory.SignalsObserved || signals.FlaggedCount != 1 || !signals.AdvisoryOnly || signals.ModelDigest != "sha256:fake" {
		t.Fatalf("orientation signals = %+v", signals)
	}
	if len(signals.Criteria) != 2 || signals.Criteria[0].Flagged || !signals.Criteria[1].Flagged {
		t.Fatalf("criterion flags = %+v", signals.Criteria)
	}
	if _, err := service.Orientation(ctx, admin, project.ID); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 2 {
		t.Fatalf("cached answers were requested again: %d calls", calls.Load())
	}

	base := strings.TrimSpace(runGit(t, project.RepositoryPath, "rev-parse", "HEAD"))
	job, err := service.Submit(ctx, admin, factory.SubmitRequest{RequestID: "submit", ProjectID: project.ID, WorkItemID: "work-test", BaseRevision: base, ExecutorKey: "fixture-agent", Deadline: time.Now().Add(time.Hour)})
	if err != nil {
		t.Fatalf("a flagged criterion must not block admission: %v", err)
	}
	if job.Admission.CriterionSignals == nil || job.Admission.CriterionSignals.FlaggedCount != 1 {
		t.Fatalf("admission did not copy cached signals: %+v", job.Admission.CriterionSignals)
	}
}

func TestUnreachableDecisionModelIsReportedNotFatal(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	service, admin, project := decisionProject(t, "http://127.0.0.1:1", "- The change works.\n")
	oriented, err := service.Orientation(ctx, admin, project.ID)
	if err != nil {
		t.Fatal(err)
	}
	if signals := oriented.CriterionSignals["work-test"]; signals == nil || signals.Status != factory.SignalsUnavailable {
		t.Fatalf("unreachable model signals = %+v", signals)
	}
}

func decisionProject(t *testing.T, url, criteria string) (*factory.Service, factory.Principal, factory.Project) {
	t.Helper()
	ctx := context.Background()
	service, err := factory.Open(t.TempDir(), factory.Options{DecisionModel: "kev:0.8b-mxfp8", DecisionURL: url})
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
	work, err := os.ReadFile(filepath.Join(bundle, "work.md"))
	if err != nil {
		t.Fatal(err)
	}
	updated := strings.Replace(string(work), "- The change works.\n", criteria, 1)
	if err := os.WriteFile(filepath.Join(bundle, "work.md"), []byte(updated), 0o600); err != nil {
		t.Fatal(err)
	}
	project, err := service.RegisterProject(ctx, admin, factory.RegisterProjectRequest{RequestID: "project", BundlePath: bundle, RepositoryPath: repository, AllowedSourceRoots: []string{repository}, Policy: factory.ProjectPolicy{RequiredChecks: []string{"go-test"}, AllowedExecutors: []string{"fixture-agent"}, MaxRepairAttempts: 1, MaxJobDurationSeconds: 7200, Fixture: true}})
	if err != nil {
		t.Fatal(err)
	}
	return service, admin, project
}
