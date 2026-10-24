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

func fakeRefinerClaude(t *testing.T, result string) (string, string) {
	t.Helper()
	dir := t.TempDir()
	args := filepath.Join(dir, "args.txt")
	prompt := filepath.Join(dir, "prompt.txt")
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > " + args + "\ncat > " + prompt + "\ncat <<'JSON'\n" + result + "\nJSON\n"
	path := filepath.Join(dir, "claude")
	if err := os.WriteFile(path, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	return path, dir
}

func refinerInput() factory.RefinerInput {
	low := 0.2
	return factory.RefinerInput{RefinementID: "ref_1", WorkItemID: "work-1", Title: "Reverse", TicketPath: "/records/work.md", Ticket: "## Acceptance criteria\n- Make it nice.\n", Criteria: []string{"Make it nice."}, Signals: &factory.CriterionSignals{Model: "kev:0.8b-mxfp8", Threshold: 0.25, Criteria: []factory.CriterionSignal{{Text: "Make it nice.", Checkable: &low, Flagged: true}}}, ProtectedPaths: []string{"**/*_test.go"}, PreviousAnswers: []factory.RefinementAnswer{{Question: "Bytes?", Answer: "Code points."}}, MaxQuestions: 3, Context: []factory.ContextSource{{Path: "/docs/spec.md", Text: "Spec text."}}}
}

func runRefiner(t *testing.T, binary string) (factory.RefinerOutput, error) {
	t.Helper()
	encoded, _ := json.Marshal(refinerInput())
	var out, log bytes.Buffer
	err := Refiner(context.Background(), bytes.NewReader(encoded), &out, &log, RefinerOptions{Binary: binary, Model: "sonnet", MaxBudgetUSD: 0.5})
	var output factory.RefinerOutput
	if err == nil {
		if decodeErr := json.Unmarshal(out.Bytes(), &output); decodeErr != nil {
			t.Fatal(decodeErr)
		}
	}
	return output, err
}

func TestRefinerUsesStructuredOutputAndReadOnlyTools(t *testing.T) {
	binary, dir := fakeRefinerClaude(t, `{"type":"result","is_error":false,"result":"","structured_output":{"criteria":["Reverse returns code points in reverse order."],"questions":[{"question":"Invalid UTF-8?","unblocks":"byte handling"}],"notes":"n"},"session_id":"s-1","num_turns":3,"total_cost_usd":0.04,"modelUsage":{"claude-sonnet-5-5":{}}}`)
	output, err := runRefiner(t, binary)
	if err != nil {
		t.Fatal(err)
	}
	if output.Proposal.Criteria[0] != "Reverse returns code points in reverse order." || len(output.Proposal.Questions) != 1 || output.Run.SessionID != "s-1" || output.Run.CostUSD != 0.04 || output.Run.Model != "claude-sonnet-5-5" {
		t.Fatalf("output = %+v", output)
	}
	args, _ := os.ReadFile(filepath.Join(dir, "args.txt"))
	joined := strings.ReplaceAll(string(args), "\n", " ")
	for _, want := range []string{"--tools Read Grep Glob", "--json-schema", "--max-budget-usd 0.5", "--model sonnet", "--no-session-persistence"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("argv %q lacks %q", joined, want)
		}
	}
	for _, forbidden := range []string{"Edit", "Write", "Bash", "dangerously", "acceptEdits"} {
		if strings.Contains(joined, forbidden) {
			t.Fatalf("refiner argv grants %q: %q", forbidden, joined)
		}
	}
	prompt, _ := os.ReadFile(filepath.Join(dir, "prompt.txt"))
	for _, want := range []string{"0.20 (flagged): Make it nice.", "Q: Bytes?", "A: Code points.", "**/*_test.go", "Spec text.", "at most 3 questions"} {
		if !strings.Contains(string(prompt), want) {
			t.Fatalf("prompt lacks %q:\n%s", want, prompt)
		}
	}
}

func TestRefinerFallsBackToResultTextAndReportsErrors(t *testing.T) {
	binary, _ := fakeRefinerClaude(t, `{"type":"result","is_error":false,"result":"Here you go:\n`+"```json"+`\n{\"criteria\":[\"Prints hello.\"],\"questions\":[],\"notes\":\"\"}\n`+"```"+`","session_id":"s-2"}`)
	output, err := runRefiner(t, binary)
	if err != nil || output.Proposal.Criteria[0] != "Prints hello." {
		t.Fatalf("fenced result = %+v, %v", output, err)
	}
	binary, _ = fakeRefinerClaude(t, `{"type":"result","is_error":true,"result":"budget exceeded"}`)
	if _, err := runRefiner(t, binary); err == nil || !strings.Contains(err.Error(), "budget exceeded") {
		t.Fatalf("error result = %v", err)
	}
}

func TestRefinerAdapterExplainsProtectedTestOwnership(t *testing.T) {
	for _, available := range []bool{true, false} {
		name := "unavailable"
		if available {
			name = "available"
		}
		t.Run(name, func(t *testing.T) {
			binary, dir := fakeRefinerClaude(t, `{"structured_output":{"criteria":["Reverse returns code points in reverse order."],"questions":[],"notes":""}}`)
			encoded, _ := json.Marshal(refinerInput())
			var input map[string]any
			if err := json.Unmarshal(encoded, &input); err != nil {
				t.Fatal(err)
			}
			input["testAuthorAvailable"] = available
			encoded, _ = json.Marshal(input)
			var output, log bytes.Buffer
			if err := Refiner(context.Background(), bytes.NewReader(encoded), &output, &log, RefinerOptions{Binary: binary}); err != nil {
				t.Fatal(err)
			}
			raw, err := os.ReadFile(filepath.Join(dir, "prompt.txt"))
			if err != nil {
				t.Fatal(err)
			}
			prompt := string(raw)
			for _, want := range []string{"must not create, edit, rename, or delete", "**/*_test.go", "do not ask for permission to bypass this policy", "Keep the work item's intent and scope", "Q: Bytes?", "A: Code points."} {
				if !strings.Contains(prompt, want) {
					t.Errorf("refiner subprocess prompt lacks %q", want)
				}
			}
			if available {
				for _, want := range []string{"A separate test author is available", "That author owns new protected tests", "Do not assign that work to the implementer", "without adding requirements about who writes tests"} {
					if !strings.Contains(prompt, want) {
						t.Errorf("refiner subprocess prompt lacks %q", want)
					}
				}
			} else if !strings.Contains(prompt, "No separate test author is available") || strings.Contains(prompt, "A separate test author is available") {
				t.Error("refiner subprocess prompt misrepresented unavailable test author")
			}
		})
	}
}
