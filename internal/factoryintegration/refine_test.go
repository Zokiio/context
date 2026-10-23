package factoryintegration

import (
	"context"
	"encoding/json"
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Zokiio/context/internal/factory"
	"github.com/Zokiio/context/internal/factoryhttp"
)

// fixtureRefine is an offline refiner: it echoes a proposal built from the input so
// the HTTP flow can be checked without a model.
func fixtureRefine(in io.Reader, out io.Writer) error {
	var input factory.RefinerInput
	if err := json.NewDecoder(in).Decode(&input); err != nil {
		return err
	}
	output := factory.RefinerOutput{Proposal: factory.RefinementProposal{Criteria: []string{"A committed fixture marker file exists at the repository root.", "The fixture check reads non-empty marker content."}, Questions: []factory.RefinementQuestion{{Question: "Should the marker name the job?", Unblocks: "marker format"}}}, Run: factory.RefinerRun{Model: "offline-refiner"}}
	return json.NewEncoder(out).Encode(output)
}

func TestRefinementOverHTTPRewritesTheTicket(t *testing.T) {
	ctx := context.Background()
	service, err := factory.Open(t.TempDir(), factory.Options{RefinerCommand: []string{os.Args[0], "-test.run=^TestFactoryFixtureProcess$", "--", "factory-fixture-refine"}})
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
	repository, bundle, _ := repositoryFixture(t)
	call[factory.ExecutorProfile](t, admin, "POST", "/v1/executors", factory.RegisterExecutorRequest{RequestID: "executor", Key: "offline", DisplayName: "Offline", Fixture: true, Command: []string{"offline"}})
	call[factory.CheckProfile](t, admin, "POST", "/v1/checks", factory.RegisterCheckRequest{RequestID: "check", Key: "check", DisplayName: "Check", Environment: "local", Fixture: true, Command: []string{"check"}})
	project := call[factory.Project](t, admin, "POST", "/v1/projects", factory.RegisterProjectRequest{RequestID: "project", BundlePath: bundle, RepositoryPath: repository, AllowedSourceRoots: []string{repository}, Policy: factory.ProjectPolicy{RequiredChecks: []string{"check"}, AllowedExecutors: []string{"offline"}, MaxRepairAttempts: 1, MaxJobDurationSeconds: 3600, Fixture: true}})
	const workID = "ec352583-bd7e-4a59-9576-46e129c76c6b"

	refinement := call[factory.Refinement](t, admin, "POST", "/v1/projects/"+string(project.ID)+"/refinements", factory.StartRefinementRequest{RequestID: "refine", WorkItemID: workID})
	waitFor(t, 15*time.Second, func() bool {
		refinement = call[factory.Refinement](t, admin, "GET", "/v1/refinements/"+string(refinement.ID), nil)
		return refinement.Status != factory.RefinementRunning
	}, func() string { return string(refinement.Status) + " " + refinement.Error })
	if refinement.Status != factory.RefinementProposed || refinement.Run.Model != "offline-refiner" {
		t.Fatalf("refinement = %+v", refinement)
	}
	listed := call[[]factory.Refinement](t, admin, "GET", "/v1/projects/"+string(project.ID)+"/refinements?workItemId="+workID, nil)
	if len(listed) != 1 {
		t.Fatalf("listed %d refinements", len(listed))
	}
	accepted := call[factory.Refinement](t, admin, "POST", "/v1/refinements/"+string(refinement.ID)+"/accept", factory.AcceptRefinementRequest{RequestID: "accept", Criteria: refinement.Proposal.Criteria, Answers: []factory.RefinementAnswer{{Question: "Should the marker name the job?", Answer: "Yes."}}})
	if accepted.Status != factory.RefinementAccepted {
		t.Fatalf("accepted = %+v", accepted)
	}
	ticket, err := os.ReadFile(filepath.Join(bundle, "issues", "change.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(ticket), "- A committed fixture marker file exists at the repository root.") || !strings.Contains(string(ticket), "A: Yes.") {
		t.Fatalf("ticket was not rewritten:\n%s", ticket)
	}
	orientation := call[factory.ProjectOrientation](t, admin, "GET", "/v1/projects/"+string(project.ID)+"/orientation", nil)
	if orientation.Result.WorkItems[0].Readiness != "ready" {
		t.Fatalf("rewritten ticket readiness = %s", orientation.Result.WorkItems[0].Readiness)
	}
}
