package factoryworker

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Zokiio/context/internal/factory"
)

func TestExpectationAuthorWritesTestsWithEditAccessOnly(t *testing.T) {
	binary, dir := fakeRefinerClaude(t, `{"type":"result","is_error":false,"result":"TestX covers criterion 1. Criterion 2 is not testable.","session_id":"s-3","num_turns":4,"total_cost_usd":0.06,"modelUsage":{"claude-sonnet-5-5":{}}}`)
	input, _ := json.Marshal(factory.ExpectationInput{ExpectationID: "exp_1", WorkItemID: "work-1", TicketPath: "/records/work.md", Ticket: "## Acceptance criteria\n- Print hello.\n", ProtectedPaths: []string{"**/*_test.go"}, Context: []factory.ContextSource{{Path: "/docs/spec.md", Text: "Spec text."}}})
	var out, log bytes.Buffer
	if err := ExpectationAuthor(context.Background(), bytes.NewReader(input), &out, &log, ExpectationAuthorOptions{Binary: binary, Model: "sonnet", MaxBudgetUSD: 1, AllowedTools: []string{"Bash(go test:*)"}}); err != nil {
		t.Fatal(err)
	}
	var output factory.ExpectationOutput
	if err := json.Unmarshal(out.Bytes(), &output); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.Notes, "Criterion 2 is not testable") || output.Run.CostUSD != 0.06 || output.Run.Model != "claude-sonnet-5-5" {
		t.Fatalf("output = %+v", output)
	}
	args, _ := os.ReadFile(filepath.Join(dir, "args.txt"))
	joined := strings.ReplaceAll(string(args), "\n", " ")
	for _, want := range []string{"--permission-mode acceptEdits", "--allowedTools Bash(go test:*)", "--max-budget-usd 1"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("argv %q lacks %q", joined, want)
		}
	}
	if strings.Contains(joined, "dangerously") {
		t.Fatalf("argv bypasses permissions: %q", joined)
	}
	prompt, _ := os.ReadFile(filepath.Join(dir, "prompt.txt"))
	for _, want := range []string{"**/*_test.go", "never implement", "contradicts its own rule", "Print hello.", "Spec text."} {
		if !strings.Contains(string(prompt), want) {
			t.Fatalf("prompt lacks %q", want)
		}
	}
}
