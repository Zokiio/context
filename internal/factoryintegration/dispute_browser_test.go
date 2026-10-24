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
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Zokiio/context/internal/factory"
	"github.com/Zokiio/context/internal/factoryhttp"
	"github.com/Zokiio/context/internal/factoryworker"
)

// Each fixture role runs in its own subprocess and makes no provider calls.
func TestDisputeBrowserProcess(t *testing.T) {
	for _, role := range os.Args {
		if role != "dispute-build" && role != "dispute-check" && role != "dispute-judge" && role != "dispute-tests" {
			continue
		}
		raw, err := io.ReadAll(io.LimitReader(os.Stdin, 16<<20))
		if err == nil {
			switch role {
			case "dispute-build":
				err = factoryworker.Fixture(context.Background(), bytes.NewReader(raw), os.Stdout, 2*time.Second, false)
			case "dispute-check":
				var assignment factory.Assignment
				err = json.Unmarshal(raw, &assignment)
				if err == nil {
					var content []byte
					content, err = os.ReadFile(filepath.Join(assignment.WorkspacePath, "tests", "marker.expect"))
					if err == nil && string(content) != "nonempty\n" {
						err = fmt.Errorf("protected expectation incorrectly requires empty content; approved criterion requires a committed marker with actual content")
					}
				}
				if err == nil {
					err = factoryworker.FixtureCheck(bytes.NewReader(raw), os.Stdout)
				}
			case "dispute-judge":
				var input factory.JudgeInput
				err = json.Unmarshal(raw, &input)
				if err == nil {
					time.Sleep(4 * time.Second)
					decision := factory.JudgeDecision{Kind: factory.DisputeReviseTests, Rationale: "The protected expectation contradicts the approved requirement for actual marker content.", EvidenceRefs: []string{"criterion:1", "tests/marker.expect", string(input.Job.ID)}, PreservesIntent: true, WeakensAcceptance: false, Directive: "Change tests/marker.expect to require nonempty content. Keep the approved requirement unchanged."}
					if proposal := input.ProposedTests; proposal != nil {
						if proposal.Commit == "" || !strings.Contains(proposal.Patch, "\n-empty") {
							err = fmt.Errorf("proposal review lacks a pinned cumulative patch against the original tests")
						}
						decision.EvidenceRefs = append(decision.EvidenceRefs, proposal.Commit)
						switch {
						case strings.Contains(proposal.Patch, "\n+present"):
							decision.Rationale = "The first proposal checks that the marker exists, but still permits empty content."
							decision.Directive = "Require nonempty marker content, not only presence. Preserve the existing criterion and valid coverage."
						case strings.Contains(proposal.Patch, "\n+nonempty"):
							decision.Kind = factory.DisputeAcceptTests
							decision.Rationale = "The improved proposal now checks actual marker content and preserves the approved requirement."
							decision.Directive = "Build against this reviewed test revision and run fresh independent verification."
						default:
							err = fmt.Errorf("judge did not receive the exact proposal patch: %+v", proposal)
						}
					}
					if err == nil {
						err = json.NewEncoder(os.Stdout).Encode(factory.JudgeOutput{Decision: decision, Run: factory.RefinerRun{Model: "offline-dispute-judge", Turns: 1}})
					}
				}
			case "dispute-tests":
				var input factory.ExpectationInput
				err = json.Unmarshal(raw, &input)
				if err == nil && input.Dispute == nil {
					err = fmt.Errorf("fixture expects a judge-directed test revision")
				}
				content := "present\n"
				notes := "The first proposal checks marker presence but does not yet require content."
				if err == nil && input.PreviousProposal != nil {
					if len(input.Dispute.TestReviews) != 1 || input.Dispute.TestReviews[0].Decision == nil || !strings.Contains(input.Dispute.TestReviews[0].Decision.Directive, "not only presence") {
						err = fmt.Errorf("improvement lacks the judge's specific feedback")
					}
					if input.BaseRevision != input.PreviousProposal.Commit {
						err = fmt.Errorf("improvement did not start from the previous proposal")
					} else if previous, readErr := os.ReadFile(filepath.Join("tests", "marker.expect")); readErr != nil || string(previous) != "present\n" {
						err = fmt.Errorf("previous test proposal missing from improvement workspace: %q, %v", previous, readErr)
					}
					content = "nonempty\n"
					notes = "Improved the protected expectation to require nonempty marker content."
				}
				if err == nil {
					time.Sleep(4 * time.Second)
					err = os.WriteFile(filepath.Join("tests", "marker.expect"), []byte(content), 0600)
				}
				if err == nil {
					err = json.NewEncoder(os.Stdout).Encode(factory.ExpectationOutput{Notes: notes, Coverage: []factory.CriterionCoverage{{Criterion: 1, Tests: []string{"tests/marker.expect"}}}, Run: factory.RefinerRun{Model: "offline-dispute-test-author", Turns: 1}})
				}
			}
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(0)
	}
}

// The external browser driver writes done after checking the real HTTP workflow.
// The ordinary Go suite skips this opt-in fixture.
func TestDisputeBrowserFixture(t *testing.T) {
	dir := os.Getenv("WAYMARK_DISPUTE_BROWSER_DIR")
	if dir == "" {
		t.Skip("set WAYMARK_DISPUTE_BROWSER_DIR for the dispute browser fixture")
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	process := func(role string) []string {
		return []string{os.Args[0], "-test.run=^TestDisputeBrowserProcess$", "--", role}
	}
	service, err := factory.Open(filepath.Join(dir, "state"), factory.Options{JudgeCommand: process("dispute-judge"), ExpectationCommand: process("dispute-tests")})
	if err != nil {
		t.Fatal(err)
	}
	defer service.Close()
	boot, err := service.Bootstrap(ctx, factory.BootstrapRequest{Username: "dispute-operator"})
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(factoryhttp.NewHandler(service))
	defer server.Close()
	admin := client(t, server.URL, boot.Token)
	repo, bundle, _ := repositoryFixture(t)
	if err := os.Mkdir(filepath.Join(repo, "tests"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "tests", "marker.expect"), []byte("empty\n"), 0600); err != nil {
		t.Fatal(err)
	}
	git(t, repo, "add", "tests")
	git(t, repo, "commit", "-m", "Add the deliberately incorrect protected expectation")
	base := git(t, repo, "rev-parse", "HEAD")
	call[factory.ExecutorProfile](t, admin, "POST", "/v1/executors", factory.RegisterExecutorRequest{RequestID: "dispute-builder", Key: "offline-dispute", DisplayName: "Offline dispute builder", Fixture: true, Command: process("dispute-build")})
	call[factory.CheckProfile](t, admin, "POST", "/v1/checks", factory.RegisterCheckRequest{RequestID: "dispute-check", Key: "protected-marker", DisplayName: "Protected marker expectation", Fixture: true, Environment: "isolated browser fixture", Command: process("dispute-check")})
	project := call[factory.Project](t, admin, "POST", "/v1/projects", factory.RegisterProjectRequest{RequestID: "dispute-project", BundlePath: bundle, RepositoryPath: repo, AllowedSourceRoots: []string{repo}, Policy: factory.ProjectPolicy{RequiredChecks: []string{"protected-marker"}, AllowedExecutors: []string{"offline-dispute"}, MaxRepairAttempts: 2, MaxJobDurationSeconds: 600, Fixture: true, ProtectedPaths: []string{"tests/"}, AllowCoordinatorReview: true}})
	var workers sync.WaitGroup
	for _, role := range []struct {
		name       string
		stage      factory.Stage
		capability factory.Capability
	}{{"builder", factory.StageImplementation, factory.CapabilityImplementWorker}, {"verifier", factory.StageVerification, factory.CapabilityVerifyWorker}} {
		user := call[factory.User](t, admin, "POST", "/v1/users", factory.CreateUserRequest{RequestID: role.name, Username: role.name, Kind: factory.UserKindWorker})
		token := call[factory.IssuedToken](t, admin, "POST", "/v1/users/"+string(user.ID)+"/tokens", factory.IssueTokenRequest{RequestID: role.name + "-token", UserID: user.ID, Label: "dispute fixture"})
		call[factory.Grant](t, admin, "PUT", "/v1/projects/"+string(project.ID)+"/grants/"+string(user.ID), factory.GrantProjectRequest{RequestID: role.name + "-grant", ProjectID: project.ID, UserID: user.ID, Capabilities: []factory.Capability{factory.CapabilityRead, role.capability}})
		worker := &factoryworker.Worker{Client: client(t, server.URL, token.Token), Process: &factoryworker.Process{ReceiptDir: filepath.Join(dir, role.name)}, Stage: role.stage, ProjectID: &project.ID, PollInterval: 100 * time.Millisecond, HeartbeatInterval: 500 * time.Millisecond}
		workers.Add(1)
		go func() { defer workers.Done(); _ = worker.Loop(ctx, nil) }()
	}
	defer func() { cancel(); workers.Wait() }()
	const workID = "ec352583-bd7e-4a59-9576-46e129c76c6b"
	job := call[factory.Job](t, admin, "POST", "/v1/jobs", factory.SubmitRequest{RequestID: "original-run", ProjectID: project.ID, WorkItemID: workID, BaseRevision: base, ExecutorKey: "offline-dispute", RepairLimit: 2, Deadline: time.Now().Add(10 * time.Minute)})
	tokenPath := filepath.Join(dir, "operator.token")
	if err := os.WriteFile(tokenPath, []byte(boot.Token), 0600); err != nil {
		t.Fatal(err)
	}
	config, _ := json.Marshal(map[string]string{"url": server.URL, "tokenFile": tokenPath, "projectId": string(project.ID), "workItemId": workID, "jobId": string(job.ID), "repository": repo})
	if err := os.WriteFile(filepath.Join(dir, "fixture.json"), config, 0600); err != nil {
		t.Fatal(err)
	}
	deadline, ticker := time.NewTimer(5*time.Minute), time.NewTicker(200*time.Millisecond)
	defer deadline.Stop()
	defer ticker.Stop()
	for {
		select {
		case <-deadline.C:
			t.Fatal("dispute browser fixture timed out waiting for done")
		case <-ticker.C:
			if _, err := os.Stat(filepath.Join(dir, "done")); err == nil {
				return
			}
		}
	}
}
