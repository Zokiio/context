package factoryworker

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Zokiio/context/internal/factory"
)

// fakeClaude writes a script that behaves like `claude -p --output-format
// stream-json`: it records its input, optionally edits the workspace, and prints
// stream events.
func fakeClaude(t *testing.T, edit bool, isError bool) (string, string) {
	t.Helper()
	dir := t.TempDir()
	prompt := filepath.Join(dir, "prompt.txt")
	script := "#!/bin/sh\ncat > " + prompt + "\n"
	if edit {
		script += "printf 'hello\\n' > hello.txt\n"
	}
	script += `echo '{"type":"system","subtype":"init","session_id":"sess-1","model":"fake-model"}'` + "\n"
	script += `echo '{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Write","input":{"file_path":"hello.txt"}}]}}'` + "\n"
	script += `echo '{"type":"result","subtype":"success","is_error":` + map[bool]string{true: "true", false: "false"}[isError] + `,"total_cost_usd":0.01,"num_turns":2,"session_id":"sess-1","result":"done"}'` + "\n"
	path := filepath.Join(dir, "claude")
	if err := os.WriteFile(path, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	return path, prompt
}

func agentWorkspace(t *testing.T) (string, factory.Assignment) {
	t.Helper()
	workspace := t.TempDir()
	for _, args := range [][]string{{"init", "-q"}, {"config", "user.name", "Test"}, {"config", "user.email", "test@example.invalid"}} {
		if out, err := exec.Command("git", append([]string{"-C", workspace}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	if err := os.WriteFile(filepath.Join(workspace, "README.md"), []byte("base\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("sh", "-c", "git -C "+workspace+" add -A && git -C "+workspace+" commit -qm base").CombinedOutput(); err != nil {
		t.Fatalf("commit: %v %s", err, out)
	}
	base, _ := git(workspace, "rev-parse", "HEAD")
	return workspace, factory.Assignment{AttemptID: "att_1", JobID: "job_1", WorkItemID: "work-1", Stage: factory.StageImplementation, WorkspacePath: workspace, BaseRevision: base, RequirementFingerprint: "fp", Instructions: "Implement the captured work item.", ProtectedPaths: []string{"tests/**"}, ContextSources: []factory.ContextSource{{Path: "/records/work.md", Text: "## Acceptance criteria\n- Print hello.\n", SHA256: "abc", Reasons: json.RawMessage(`[{"kind":"selected"}]`)}}}
}

func runAgent(t *testing.T, binary string, assignment factory.Assignment) (string, error) {
	t.Helper()
	input, err := json.Marshal(assignment)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	err = Agent(context.Background(), bytes.NewReader(input), &out, AgentOptions{Harness: "claude", Binary: binary})
	return out.String(), err
}

func TestAgentCommitsTheAgentsChangesAsTheCandidate(t *testing.T) {
	progress := filepath.Join(t.TempDir(), "progress.txt")
	t.Setenv("CTX_FACTORY_PROGRESS", progress)
	resultPath := filepath.Join(t.TempDir(), "result.json")
	t.Setenv("CTX_FACTORY_RESULT", resultPath)
	binary, prompt := fakeClaude(t, true, false)
	workspace, assignment := agentWorkspace(t)
	out, err := runAgent(t, binary, assignment)
	if err != nil {
		t.Fatalf("agent failed: %v\n%s", err, out)
	}
	head, _ := git(workspace, "rev-parse", "HEAD")
	if head == assignment.BaseRevision {
		t.Fatal("no candidate commit")
	}
	message, _ := git(workspace, "log", "-1", "--format=%B")
	if !strings.Contains(message, "Session: sess-1") || !strings.Contains(message, "Job: job_1") {
		t.Fatalf("commit message lacks provenance: %q", message)
	}
	if status, _ := git(workspace, "status", "--porcelain"); status != "" {
		t.Fatalf("workspace not clean: %q", status)
	}
	notes, _ := os.ReadFile(progress)
	if !strings.Contains(string(notes), "Write: hello.txt") || !strings.Contains(string(notes), "committed candidate "+head) {
		t.Fatalf("progress notes = %q", notes)
	}
	if got := latestProgress(progress); got != "committed candidate "+head {
		t.Fatalf("latest progress = %q", got)
	}
	input, _ := os.ReadFile(prompt)
	for _, want := range []string{"Print hello.", "tests/**", "Do not commit", "/records/work.md"} {
		if !strings.Contains(string(input), want) {
			t.Fatalf("prompt lacks %q:\n%s", want, input)
		}
	}
	if result, err := os.ReadFile(resultPath); err != nil || !strings.Contains(string(result), `"costUsd":0.01`) {
		t.Fatalf("agent result file = %q, %v", result, err)
	}
	if !strings.Contains(out, `"costUsd":0.01`) || !strings.Contains(out, `"turns":2`) {
		t.Fatalf("result summary missing: %s", out)
	}
}

func TestAgentFailsWithoutChangesOrOnAgentError(t *testing.T) {
	for _, tc := range []struct {
		name          string
		edit, isError bool
		want          string
	}{
		{"no changes", false, false, "without changing"},
		{"agent error", true, true, "reported an error"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			binary, _ := fakeClaude(t, tc.edit, tc.isError)
			workspace, assignment := agentWorkspace(t)
			_, err := runAgent(t, binary, assignment)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v, want %q", err, tc.want)
			}
			if head, _ := git(workspace, "rev-parse", "HEAD"); head != assignment.BaseRevision {
				t.Fatal("a failed agent run must not commit")
			}
		})
	}
}

func TestAgentCommandConfinesEachHarness(t *testing.T) {
	assignment := factory.Assignment{WorkspacePath: "/w"}
	claude, err := agentCommand(assignment, AgentOptions{Harness: "claude", Model: "sonnet", MaxBudgetUSD: 1.5, AllowedTools: []string{"Bash(go test:*)"}})
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(claude, " ")
	for _, want := range []string{"--permission-mode acceptEdits", "--max-budget-usd 1.5", "--allowedTools Bash(go test:*)", "--model sonnet"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("claude argv %q lacks %q", joined, want)
		}
	}
	if strings.Contains(joined, "dangerously") {
		t.Fatalf("claude argv bypasses permissions: %q", joined)
	}
	codex, err := agentCommand(assignment, AgentOptions{Harness: "codex"})
	if err != nil {
		t.Fatal(err)
	}
	if joined := strings.Join(codex, " "); !strings.Contains(joined, "--sandbox workspace-write") || !strings.Contains(joined, "--cd /w") {
		t.Fatalf("codex argv = %q", joined)
	}
	if _, err := agentCommand(assignment, AgentOptions{Harness: "other"}); err == nil {
		t.Fatal("unknown harness accepted")
	}
}

func TestPromptsAddTheProjectsInstructionsAfterTheRules(t *testing.T) {
	const instructions = "Keep functions short."
	prompts := map[string]string{
		"builder":     AgentPrompt(factory.Assignment{ProjectInstructions: instructions}),
		"refiner":     RefinerPrompt(factory.RefinerInput{Instructions: instructions, MaxQuestions: 3}),
		"test author": ExpectationPrompt(factory.ExpectationInput{Instructions: instructions, ProtectedPaths: []string{"tests/"}}),
	}
	for name, prompt := range prompts {
		rules, added := strings.Index(prompt, "## Rules"), strings.Index(prompt, instructions)
		if rules < 0 || added < rules || !strings.Contains(prompt, "the rules above win") {
			t.Errorf("%s prompt does not add the instructions after its rules:\n%s", name, prompt)
		}
	}
	if strings.Contains(AgentPrompt(factory.Assignment{}), "## The project's instructions") {
		t.Error("a project without instructions got an instructions section")
	}
}
