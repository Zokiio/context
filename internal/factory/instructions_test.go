package factory_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Zokiio/context/internal/factory"
)

func TestEachAgentGetsTheProjectsInstructionsAsTheyWereWhenItStarted(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	author := fakeAuthor(t, map[string]string{"tests/plan_test.txt": "expects the change\n"})
	capture := filepath.Join(t.TempDir(), "author-input.json")
	// Keep what the author was given instead of discarding it.
	script, err := os.ReadFile(author[0])
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(author[0], []byte(strings.Replace(string(script), "cat > /dev/null", "cat > '"+capture+"'", 1)), 0o700); err != nil {
		t.Fatal(err)
	}
	h := setupExpectations(t, author, nil)
	request := factory.SetAgentInstructionsRequest{RequestID: "instructions", ProjectID: h.project.ID, Refiner: "Name the Go package.", TestAuthor: "Use table tests.", Builder: "Keep functions short."}
	for name, actor := range map[string]factory.Principal{"reviewer": h.reviewer, "worker": h.implementer} {
		if _, err := h.service.SetAgentInstructions(ctx, actor, request); !isCode(err, factory.CodeForbidden) {
			t.Fatalf("%s set the instructions: %v", name, err)
		}
	}
	tooLong := request
	tooLong.RequestID, tooLong.Builder = "too-long", strings.Repeat("x", factory.MaxAgentInstructionRunes+1)
	if _, err := h.service.SetAgentInstructions(ctx, h.admin, tooLong); !isCode(err, factory.CodeInvalid) {
		t.Fatalf("overlong instructions = %v", err)
	}
	project, err := h.service.SetAgentInstructions(ctx, h.admin, request)
	if err != nil {
		t.Fatal(err)
	}
	if saved := project.Policy.Instructions; saved.Builder != "Keep functions short." || saved.TestAuthor != "Use table tests." || saved.SetBy == nil || *saved.SetBy != h.admin.UserID {
		t.Fatalf("saved instructions = %+v", saved)
	}

	expectation := h.write(t, "tests")
	encoded, err := os.ReadFile(capture)
	if err != nil {
		t.Fatal(err)
	}
	var given factory.ExpectationInput
	if err := json.Unmarshal(encoded, &given); err != nil {
		t.Fatal(err)
	}
	if expectation.Instructions != "Use table tests." || given.Instructions != "Use table tests." {
		t.Fatalf("test author was given %q and recorded %q", given.Instructions, expectation.Instructions)
	}

	job, err := h.service.Submit(ctx, h.coordinator, factory.SubmitRequest{RequestID: "job", ProjectID: h.project.ID, WorkItemID: "work-test", BaseRevision: head(t, h.repository, "HEAD"), ExecutorKey: "fixture-agent", Deadline: time.Now().Add(time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	changed := request
	changed.RequestID, changed.Builder = "changed", "Changed after admission."
	if _, err := h.service.SetAgentInstructions(ctx, h.admin, changed); err != nil {
		t.Fatal(err)
	}
	assignment, err := h.service.Claim(ctx, h.implementer, factory.WorkerClaimRequest{RequestID: "claim", Stages: []factory.Stage{factory.StageImplementation}, ProjectID: &h.project.ID})
	if err != nil {
		t.Fatal(err)
	}
	if job.Admission.Instructions != "Keep functions short." || assignment.ProjectInstructions != "Keep functions short." {
		t.Fatalf("builder admitted with %q and assigned %q", job.Admission.Instructions, assignment.ProjectInstructions)
	}
}

func TestTheRefinerGetsTheProjectsInstructions(t *testing.T) {
	t.Parallel()
	refiner, dump := fakeRefiner(t, "")
	h := setupRefine(t, refiner)
	if _, err := h.service.SetAgentInstructions(context.Background(), h.admin, factory.SetAgentInstructionsRequest{RequestID: "instructions", ProjectID: h.project.ID, Refiner: "Name the Go package."}); err != nil {
		t.Fatal(err)
	}
	refinement := h.start(t, factory.StartRefinementRequest{RequestID: "refine"})
	encoded, err := os.ReadFile(dump)
	if err != nil {
		t.Fatal(err)
	}
	var given factory.RefinerInput
	if err := json.Unmarshal(encoded, &given); err != nil {
		t.Fatal(err)
	}
	if refinement.Instructions != "Name the Go package." || given.Instructions != "Name the Go package." {
		t.Fatalf("refiner was given %q and recorded %q", given.Instructions, refinement.Instructions)
	}
}
