package factory_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Zokiio/context/internal/factory"
)

const fakeProposal = `{"proposal":{"criteria":["Reverse returns its input with code points in reverse order.","go test passes without editing the holdout tests."],"questions":[{"question":"Should invalid UTF-8 be preserved?","unblocks":"Byte handling"},{"question":"Q2?","unblocks":"u2"},{"question":"Q3?","unblocks":"u3"},{"question":"Q4?","unblocks":"u4"}],"notes":"Assumed code points, not grapheme clusters."},"run":{"model":"fake-refiner","sessionId":"sess-r","turns":2,"costUsd":0.01}}`

// fakeRefiner writes a refiner that records its input and prints a fixed proposal,
// optionally after a delay.
func fakeRefiner(t *testing.T, delay string) ([]string, string) {
	t.Helper()
	dir := t.TempDir()
	dump := filepath.Join(dir, "input.json")
	script := "#!/bin/sh\ncat > " + dump + "\n"
	if delay != "" {
		script += "sleep " + delay + "\n"
	}
	script += "cat <<'JSON'\n" + fakeProposal + "\nJSON\n"
	path := filepath.Join(dir, "refiner")
	if err := os.WriteFile(path, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	return []string{path}, dump
}

type refineHarness struct {
	dataDir string
	service *factory.Service
	admin   factory.Principal
	project factory.Project
	ticket  string
}

func setupRefine(t *testing.T, refiner []string) refineHarness {
	t.Helper()
	return setupRefineConfigured(t, refiner, nil)
}

func setupRefineConfigured(t *testing.T, refiner []string, configure func(*factory.Options, *factory.ProjectPolicy)) refineHarness {
	t.Helper()
	ctx := context.Background()
	var calls atomic.Int64
	server := fakeDecisionModel(t, &calls)
	dataDir := t.TempDir()
	options := factory.Options{DecisionModel: "kev:0.8b-mxfp8", DecisionURL: server.URL, RefinerCommand: refiner}
	policy := factory.ProjectPolicy{RequiredChecks: []string{"go-test"}, AllowedExecutors: []string{"fixture-agent"}, MaxRepairAttempts: 1, MaxJobDurationSeconds: 7200, Fixture: true, ProtectedPaths: []string{"**/*_test.go"}}
	if configure != nil {
		configure(&options, &policy)
	}
	service, err := factory.Open(dataDir, options)
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
	ticket := filepath.Join(bundle, "work.md")
	work, err := os.ReadFile(ticket)
	if err != nil {
		t.Fatal(err)
	}
	updated := strings.Replace(string(work), "- The change works.\n", "- Reverse works.\n- Make it vague and nice.\n", 1) + "\n## Comments\n\n- Earlier note.\n"
	if err := os.WriteFile(ticket, []byte(updated), 0o600); err != nil {
		t.Fatal(err)
	}
	project, err := service.RegisterProject(ctx, admin, factory.RegisterProjectRequest{RequestID: "project", BundlePath: bundle, RepositoryPath: repository, AllowedSourceRoots: []string{repository}, Policy: policy})
	if err != nil {
		t.Fatal(err)
	}
	return refineHarness{dataDir: dataDir, service: service, admin: admin, project: project, ticket: ticket}
}

func (h refineHarness) start(t *testing.T, request factory.StartRefinementRequest) factory.Refinement {
	t.Helper()
	request.ProjectID, request.WorkItemID = h.project.ID, "work-test"
	started, err := h.service.StartRefinement(context.Background(), h.admin, request)
	if err != nil {
		t.Fatal(err)
	}
	return h.waitProposed(t, started.ID)
}

func (h refineHarness) waitProposed(t *testing.T, id factory.ID) factory.Refinement {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for {
		refinement, err := h.service.GetRefinement(context.Background(), h.admin, id)
		if err != nil {
			t.Fatal(err)
		}
		if refinement.Status != factory.RefinementRunning {
			return refinement
		}
		if time.Now().After(deadline) {
			t.Fatalf("refinement still running: %+v", refinement)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func TestRefinementProposesScoresAndRewritesTheTicket(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	refiner, dump := fakeRefiner(t, "")
	h := setupRefine(t, refiner)
	refinement := h.start(t, factory.StartRefinementRequest{RequestID: "refine"})
	if refinement.Status != factory.RefinementProposed || refinement.Run == nil || refinement.Run.Model != "fake-refiner" {
		t.Fatalf("refinement = %+v", refinement)
	}
	if refinement.Before == nil || refinement.Before.FlaggedCount != 1 || refinement.After == nil || refinement.After.FlaggedCount != 0 {
		t.Fatalf("Kev did not score before and after: before=%+v after=%+v", refinement.Before, refinement.After)
	}
	if len(refinement.Proposal.Questions) != factory.MaxRefinementQuestions || !strings.Contains(refinement.Proposal.Notes, "1 further questions were dropped") {
		t.Fatalf("questions were not capped transparently: %+v", refinement.Proposal)
	}
	var input factory.RefinerInput
	raw, err := os.ReadFile(dump)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &input); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(input.Ticket, "Make it vague and nice.") || len(input.Criteria) != 2 || input.Signals == nil || strings.Join(input.ProtectedPaths, ",") != "**/*_test.go" || input.MaxQuestions != 3 {
		t.Fatalf("refiner input = %+v", input)
	}

	before, err := h.service.Orientation(ctx, h.admin, h.project.ID)
	if err != nil {
		t.Fatal(err)
	}
	criteria := append([]string{}, refinement.Proposal.Criteria...)
	criteria[1] = "go test ./... passes without editing tests."
	accepted, err := h.service.AcceptRefinement(ctx, h.admin, factory.AcceptRefinementRequest{RequestID: "accept", RefinementID: refinement.ID, Criteria: criteria, Answers: []factory.RefinementAnswer{{Question: "Should invalid UTF-8 be preserved?", Answer: "No, replace it."}}})
	if err != nil {
		t.Fatal(err)
	}
	if accepted.Status != factory.RefinementAccepted || accepted.UpdatedTicketSHA256 == "" || accepted.UpdatedTicketSHA256 == accepted.TicketSHA256 {
		t.Fatalf("accepted = %+v", accepted)
	}
	ticket, err := os.ReadFile(h.ticket)
	if err != nil {
		t.Fatal(err)
	}
	text := string(ticket)
	for _, want := range []string{"## Acceptance criteria\n\n- Reverse returns its input with code points in reverse order.\n- go test ./... passes without editing tests.\n\n## Spec", "- Earlier note.", "accepted refined acceptance criteria proposed by fake-refiner", "Q: Should invalid UTF-8 be preserved? A: No, replace it."} {
		if !strings.Contains(text, want) {
			t.Fatalf("ticket lacks %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "vague and nice") {
		t.Fatalf("old criterion survived:\n%s", text)
	}
	after, err := h.service.Orientation(ctx, h.admin, h.project.ID)
	if err != nil {
		t.Fatal(err)
	}
	if readiness(after, "work-test") != "ready" || criteriaDigest(after, "work-test") == criteriaDigest(before, "work-test") {
		t.Fatalf("rewritten ticket readiness=%s, criteria digest unchanged=%v", readiness(after, "work-test"), criteriaDigest(after, "work-test") == criteriaDigest(before, "work-test"))
	}
	if _, err := h.service.AcceptRefinement(ctx, h.admin, factory.AcceptRefinementRequest{RequestID: "accept-again", RefinementID: refinement.ID, Criteria: criteria}); !isCode(err, factory.CodeConflict) {
		t.Fatalf("second acceptance = %v", err)
	}
}

func TestRefinementReceivesConfiguredTestAuthorAvailability(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name                              string
		author, protected, required, want bool
	}{
		{name: "optional tests with author", author: true, protected: true, want: true},
		{name: "required tests with author", author: true, protected: true, required: true, want: true},
		{name: "protected paths without author", protected: true},
		{name: "author without protected paths", author: true},
		{name: "neither author nor protected paths"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			refiner, dump := fakeRefiner(t, "")
			h := setupRefineConfigured(t, refiner, func(options *factory.Options, policy *factory.ProjectPolicy) {
				if tc.author {
					options.ExpectationCommand = fakeAuthor(t, nil)
				}
				if !tc.protected {
					policy.ProtectedPaths = nil
				}
				policy.RequireExpectations = tc.required
			})
			if result := h.start(t, factory.StartRefinementRequest{RequestID: "ownership"}); result.Status != factory.RefinementProposed {
				t.Fatalf("refinement = %+v", result)
			}
			raw, err := os.ReadFile(dump)
			if err != nil {
				t.Fatal(err)
			}
			var input struct {
				TestAuthorAvailable *bool `json:"testAuthorAvailable"`
			}
			if err := json.Unmarshal(raw, &input); err != nil {
				t.Fatal(err)
			}
			if input.TestAuthorAvailable == nil || *input.TestAuthorAvailable != tc.want {
				t.Fatalf("refiner subprocess testAuthorAvailable = %v, want explicit %t", input.TestAuthorAvailable, tc.want)
			}
		})
	}
}

func TestRefinementGoesStaleWhenTheTicketChanges(t *testing.T) {
	t.Parallel()
	refiner, _ := fakeRefiner(t, "")
	h := setupRefine(t, refiner)
	refinement := h.start(t, factory.StartRefinementRequest{RequestID: "refine"})
	f, err := os.OpenFile(h.ticket, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.WriteString("- Edited by hand meanwhile.\n")
	_ = f.Close()
	_, err = h.service.AcceptRefinement(context.Background(), h.admin, factory.AcceptRefinementRequest{RequestID: "accept", RefinementID: refinement.ID, Criteria: refinement.Proposal.Criteria})
	if !isCode(err, factory.CodeStale) {
		t.Fatalf("accept after edit = %v", err)
	}
	current, _ := h.service.GetRefinement(context.Background(), h.admin, refinement.ID)
	if current.Status != factory.RefinementStale {
		t.Fatalf("status = %s", current.Status)
	}
}

func TestRefiningAgainCarriesAnswersAndSupersedes(t *testing.T) {
	t.Parallel()
	refiner, dump := fakeRefiner(t, "")
	h := setupRefine(t, refiner)
	first := h.start(t, factory.StartRefinementRequest{RequestID: "first"})
	second := h.start(t, factory.StartRefinementRequest{RequestID: "second", PreviousID: first.ID, Answers: []factory.RefinementAnswer{{Question: first.Proposal.Questions[0].Question, Answer: "Replace invalid bytes."}}})
	if second.Status != factory.RefinementProposed || second.PreviousID != first.ID {
		t.Fatalf("second = %+v", second)
	}
	previous, _ := h.service.GetRefinement(context.Background(), h.admin, first.ID)
	if previous.Status != factory.RefinementSuperseded {
		t.Fatalf("first status = %s", previous.Status)
	}
	raw, _ := os.ReadFile(dump)
	var input factory.RefinerInput
	_ = json.Unmarshal(raw, &input)
	if input.PreviousProposal == nil || len(input.PreviousAnswers) != 1 || input.PreviousAnswers[0].Answer != "Replace invalid bytes." {
		t.Fatalf("second input did not carry the answers: %+v", input)
	}
	rejected, err := h.service.RejectRefinement(context.Background(), h.admin, factory.RejectRefinementRequest{RequestID: "reject", RefinementID: second.ID})
	if err != nil || rejected.Status != factory.RefinementRejected {
		t.Fatalf("reject = %+v, %v", rejected, err)
	}
	listed, err := h.service.Refinements(context.Background(), h.admin, h.project.ID, "work-test")
	if err != nil || len(listed) != 2 || listed[0].ID != second.ID {
		t.Fatalf("listed = %d, %v", len(listed), err)
	}
}

func TestRefinementRefusesWorkFrozenByAnActiveJob(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	refiner, _ := fakeRefiner(t, "")
	h := setupRefine(t, refiner)
	base := strings.TrimSpace(runGit(t, h.project.RepositoryPath, "rev-parse", "HEAD"))
	if _, err := h.service.Submit(ctx, h.admin, factory.SubmitRequest{RequestID: "submit", ProjectID: h.project.ID, WorkItemID: "work-test", BaseRevision: base, ExecutorKey: "fixture-agent", Deadline: time.Now().Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	_, err := h.service.StartRefinement(ctx, h.admin, factory.StartRefinementRequest{RequestID: "refine", ProjectID: h.project.ID, WorkItemID: "work-test"})
	if !isCode(err, factory.CodeConflict) {
		t.Fatalf("refinement during an active job = %v", err)
	}
}

func TestRefinementWithoutRefinerAndStoppedRuns(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	unconfigured := setupRefine(t, nil)
	if _, err := unconfigured.service.StartRefinement(ctx, unconfigured.admin, factory.StartRefinementRequest{RequestID: "refine", ProjectID: unconfigured.project.ID, WorkItemID: "work-test"}); !isCode(err, factory.CodeNotReady) {
		t.Fatalf("unconfigured refinement = %v", err)
	}

	refiner, _ := fakeRefiner(t, "30")
	h := setupRefine(t, refiner)
	started, err := h.service.StartRefinement(ctx, h.admin, factory.StartRefinementRequest{RequestID: "slow", ProjectID: h.project.ID, WorkItemID: "work-test"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.service.StartRefinement(ctx, h.admin, factory.StartRefinementRequest{RequestID: "parallel", ProjectID: h.project.ID, WorkItemID: "work-test"}); !isCode(err, factory.CodeConflict) {
		t.Fatalf("second running refinement = %v", err)
	}
	stopped := make(chan error, 1)
	go func() { stopped <- h.service.Close() }()
	select {
	case err := <-stopped:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(15 * time.Second):
		t.Fatal("close did not stop the running refiner")
	}
	reopened, err := factory.Open(h.dataDir, factory.Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	recorded, err := reopened.GetRefinement(ctx, h.admin, started.ID)
	if err != nil {
		t.Fatal(err)
	}
	if recorded.Status != factory.RefinementFailed || !strings.Contains(recorded.Error, "stopped") {
		t.Fatalf("stopped refinement = %s %q", recorded.Status, recorded.Error)
	}
}

func readiness(view factory.ProjectOrientation, id string) string {
	for _, work := range view.Result.WorkItems {
		if work.ID != nil && *work.ID == id {
			return work.Readiness
		}
	}
	return ""
}

func criteriaDigest(view factory.ProjectOrientation, id string) string {
	for _, work := range view.Result.WorkItems {
		if work.ID != nil && *work.ID == id && work.CriteriaSHA256 != nil {
			return *work.CriteriaSHA256
		}
	}
	return ""
}

func isCode(err error, code factory.ErrorCode) bool {
	var fault *factory.Error
	return errors.As(err, &fault) && fault.Code == code
}
