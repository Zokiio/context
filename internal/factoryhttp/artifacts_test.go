package factoryhttp_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/Zokiio/context/internal/factory"
)

func TestFailedImplementationEvidenceIsInspectable(t *testing.T) {
	i := start(t)
	ctx := context.Background()
	if _, err := i.service.RegisterExecutor(ctx, i.admin, factory.RegisterExecutorRequest{RequestID: "executor", Key: "fixture-executor", DisplayName: "Fixture", Command: []string{"fixture"}, Fixture: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := i.service.RegisterCheck(ctx, i.admin, factory.RegisterCheckRequest{RequestID: "check", Key: "fixture-check", DisplayName: "Fixture check", Command: []string{"fixture"}, Environment: "offline", Fixture: true}); err != nil {
		t.Fatal(err)
	}
	p := project(t, i, "artifact-project")
	u, err := i.service.CreateUser(ctx, i.admin, factory.CreateUserRequest{RequestID: "worker", Username: "worker", Kind: factory.UserKindWorker})
	if err != nil {
		t.Fatal(err)
	}
	token, err := i.service.IssueToken(ctx, i.admin, factory.IssueTokenRequest{RequestID: "worker-token", UserID: u.ID, Label: "fixture"})
	if err != nil {
		t.Fatal(err)
	}
	worker, err := i.service.Authenticate(ctx, token.Token)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := i.service.GrantProject(ctx, i.admin, factory.GrantProjectRequest{RequestID: "worker-grant", ProjectID: p.ID, UserID: u.ID, Capabilities: []factory.Capability{factory.CapabilityRead, factory.CapabilityImplementWorker}}); err != nil {
		t.Fatal(err)
	}
	job, err := i.service.Submit(ctx, i.admin, factory.SubmitRequest{RequestID: "submit", ProjectID: p.ID, WorkItemID: "work-1", BaseRevision: "HEAD", ExecutorKey: "fixture-executor", RepairLimit: 1, Deadline: time.Now().Add(time.Minute)})
	if err != nil {
		t.Fatal(err)
	}
	a, err := i.service.Claim(ctx, worker, factory.WorkerClaimRequest{RequestID: "claim", Stages: []factory.Stage{factory.StageImplementation}, ProjectID: &p.ID})
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range []factory.WorkerReportKind{factory.ReportStartRegistered, factory.ReportStarted} {
		if _, err := i.service.Report(ctx, worker, factory.WorkerReportRequest{RequestID: string(kind), AttemptID: a.AttemptID, DispatchID: a.DispatchID, Kind: kind, ObservedAt: time.Now(), NativeRunID: "fixture-process"}); err != nil {
			t.Fatal(err)
		}
	}
	const observation = "Fixture implementation failed. <script>window.evidenceExecuted=true</script>"
	if _, err := i.service.Report(ctx, worker, factory.WorkerReportRequest{RequestID: "failed", AttemptID: a.AttemptID, DispatchID: a.DispatchID, Kind: factory.ReportTerminal, ObservedAt: time.Now(), NativeRunID: "fixture-process", Outcome: factory.OutcomeFailed, TerminalReceipt: observation, Artifacts: []factory.ArtifactInput{{Name: "command-1.log", MediaType: "text/plain", Content: observation}}}); err != nil {
		t.Fatal(err)
	}
	path := "/v1/jobs/" + string(job.ID)
	status, _, raw := call(t, i.server.URL, i.token, "GET", path, "", nil)
	if status != 200 {
		t.Fatalf("read failed job: %d %s", status, raw)
	}
	if err := json.Unmarshal(raw, &job); err != nil {
		t.Fatal(err)
	}
	if job.Status != factory.JobFailed || len(job.Attempts) != 1 || job.Attempts[0].TerminalReceipt != observation || len(job.Attempts[0].Artifacts) != 1 {
		t.Fatalf("failed execution lost its inspectable observations: %+v", job)
	}
	artifact := job.Attempts[0].Artifacts[0]
	artifactPath := path + "/artifacts/" + string(artifact.ID)
	status, _, raw = call(t, i.server.URL, i.token, "GET", artifactPath, "", nil)
	var fetched factory.Artifact
	if err := json.Unmarshal(raw, &fetched); err != nil {
		t.Fatal(err)
	}
	if status != 200 || fetched.Content != observation || fetched.SHA256 != artifact.SHA256 {
		t.Fatalf("retained failure artifact not accessible: %d %+v", status, fetched)
	}
	_, _, outsider := user(t, i, "outsider")
	status, _, _ = call(t, i.server.URL, outsider, "GET", artifactPath, "", nil)
	if status != 404 {
		t.Fatalf("ungranted user accessed execution evidence: %d", status)
	}
}
