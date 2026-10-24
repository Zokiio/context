package factoryworker

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/Zokiio/context/internal/factory"
)

// AgentOptions configures a native coding-agent CLI run as an implementation command.
type AgentOptions struct {
	Harness      string // "claude" or "codex"
	Binary       string
	Model        string
	MaxBudgetUSD float64  // Claude Code only
	AllowedTools []string // Claude Code only; for example "Bash(go test:*)"
}

// AgentResult is what the adapter observed from the native agent, not acceptance
// evidence.
type AgentResult struct {
	Harness   string  `json:"harness"`
	Model     string  `json:"model,omitempty"`
	SessionID string  `json:"sessionId,omitempty"`
	Turns     int     `json:"turns,omitempty"`
	CostUSD   float64 `json:"costUsd,omitempty"`
	IsError   bool    `json:"isError"`
	Result    string  `json:"result,omitempty"`
	Candidate string  `json:"candidate,omitempty"`
}

const agentInstruction = "Implement the Waymark work item described in the input. Follow its rules exactly."

// Agent reads an implementation assignment from in, runs the native agent in the
// assigned worktree with the admitted context, and commits the agent's changes as the
// candidate. The agent never commits itself, so the candidate is exactly what the
// agent left in the worktree.
func Agent(ctx context.Context, in io.Reader, out io.Writer, options AgentOptions) error {
	var a factory.Assignment
	if err := json.NewDecoder(io.LimitReader(in, 64<<20)).Decode(&a); err != nil {
		return err
	}
	if a.Stage != factory.StageImplementation || a.WorkspacePath == "" {
		return errors.New("agent adapter requires an implementation assignment")
	}
	if !a.Deadline.IsZero() {
		var cancel context.CancelFunc
		ctx, cancel = context.WithDeadline(ctx, a.Deadline)
		defer cancel()
	}
	// The agent's stderr is copied by exec on its own goroutine while this one
	// writes progress, so both share one serialized writer.
	out = &lockedWriter{w: out}
	progress := progressWriter(os.Getenv("CTX_FACTORY_PROGRESS"))
	defer progress.Close()
	argv, err := agentCommand(a, options)
	if err != nil {
		return err
	}
	command := exec.CommandContext(ctx, argv[0], argv[1:]...)
	command.Dir = a.WorkspacePath
	prompt := AgentPrompt(a)
	// The native agent can write only inside its workspace. It never receives
	// access to the worker's durable receipt directory through this handoff.
	var nativeDispute string
	if os.Getenv("CTX_FACTORY_DISPUTE") != "" {
		directory, err := os.MkdirTemp(a.WorkspacePath, ".waymark-dispute-")
		if err != nil {
			return err
		}
		defer os.RemoveAll(directory)
		nativeDispute = filepath.Join(directory, "dispute.json")
		command.Env = append(os.Environ(), "CTX_FACTORY_DISPUTE="+nativeDispute)
		prompt += "\n## Dispute output\n\nCTX_FACTORY_DISPUTE for this attempt is " + nativeDispute + ". Use your file-writing tool to write the JSON report at this exact path, then stop.\n"
	}
	command.Stdin = strings.NewReader(prompt)
	command.Stderr = out
	stdout, err := command.StdoutPipe()
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "waymark agent: starting %s in %s\n", options.Harness, a.WorkspacePath)
	if err := command.Start(); err != nil {
		return fmt.Errorf("start %s: %w", options.Harness, err)
	}
	result := AgentResult{Harness: options.Harness, Model: options.Model}
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 64<<10), 16<<20)
	for scanner.Scan() {
		if note := agentEvent(options.Harness, scanner.Bytes(), &result); note != "" {
			progress.note(note)
			fmt.Fprintln(out, "progress:", note)
		}
	}
	waitErr := command.Wait()
	summary, _ := json.Marshal(result)
	fmt.Fprintf(out, "waymark agent result: %s\n", summary)
	if path := os.Getenv("CTX_FACTORY_RESULT"); path != "" {
		_ = os.WriteFile(path, summary, 0o600)
	}
	if waitErr != nil {
		return fmt.Errorf("%s exited: %w", options.Harness, waitErr)
	}
	if result.IsError {
		return fmt.Errorf("%s reported an error: %s", options.Harness, result.Result)
	}
	if nativeDispute != "" {
		dispute, err := readDispute(nativeDispute)
		if err != nil {
			return err
		}
		if dispute != nil {
			if err := publishAgentDispute(os.Getenv("CTX_FACTORY_DISPUTE"), dispute); err != nil {
				return err
			}
			progress.note("stopped for independent judgment")
			fmt.Fprintln(out, "waymark agent: reported dispute without committing a candidate")
			return nil
		}
		if err := os.RemoveAll(filepath.Dir(nativeDispute)); err != nil {
			return err
		}
	}
	status, err := git(a.WorkspacePath, "status", "--porcelain")
	if err != nil {
		return err
	}
	if strings.TrimSpace(status) == "" {
		return errors.New("agent finished without changing the workspace")
	}
	if _, err := git(a.WorkspacePath, "add", "-A"); err != nil {
		return err
	}
	if changed, err := git(a.WorkspacePath, "diff", "--cached", "--name-only", "--no-renames"); err == nil {
		for _, file := range strings.Split(changed, "\n") {
			for _, pattern := range a.ProtectedPaths {
				if file != "" && factory.MatchProtectedPath(pattern, file) {
					fmt.Fprintf(out, "warning: agent changed protected path %s; the controller will reject this candidate\n", file)
				}
			}
		}
	}
	message := fmt.Sprintf("Agent candidate for %s\n\nJob: %s\nAttempt: %s\nHarness: %s\nModel: %s\nSession: %s\n", a.WorkItemID, a.JobID, a.AttemptID, result.Harness, result.Model, result.SessionID)
	if _, err := git(a.WorkspacePath, "-c", "user.name=Waymark agent ("+options.Harness+")", "-c", "user.email=agent@waymark.invalid", "commit", "-q", "-m", message); err != nil {
		return err
	}
	result.Candidate, err = git(a.WorkspacePath, "rev-parse", "HEAD")
	if err != nil {
		return err
	}
	progress.note("committed candidate " + result.Candidate)
	fmt.Fprintf(out, "waymark agent: %s produced candidate %s\n", options.Harness, result.Candidate)
	return nil
}

func agentCommand(a factory.Assignment, options AgentOptions) ([]string, error) {
	binary := options.Binary
	switch options.Harness {
	case "claude":
		if binary == "" {
			binary = "claude"
		}
		argv := []string{binary, "-p", agentInstruction, "--output-format", "stream-json", "--verbose", "--permission-mode", "acceptEdits", "--no-session-persistence"}
		if options.Model != "" {
			argv = append(argv, "--model", options.Model)
		}
		if options.MaxBudgetUSD > 0 {
			argv = append(argv, "--max-budget-usd", fmt.Sprint(options.MaxBudgetUSD))
		}
		if len(options.AllowedTools) > 0 {
			argv = append(argv, "--allowedTools")
			argv = append(argv, options.AllowedTools...)
		}
		return argv, nil
	case "codex":
		if binary == "" {
			binary = "codex"
		}
		argv := []string{binary, "exec", "--json", "--sandbox", "workspace-write", "--cd", a.WorkspacePath}
		if options.Model != "" {
			argv = append(argv, "--model", options.Model)
		}
		return append(argv, agentInstruction), nil
	default:
		return nil, fmt.Errorf("unsupported agent harness %q; use claude or codex", options.Harness)
	}
}

// AgentPrompt renders the assignment as the agent's input: the rules, then every
// admitted context source with its inclusion reasons and digest.
func AgentPrompt(a factory.Assignment) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Waymark implementation assignment\n\nWork item: %s\nJob: %s\nAttempt: %s\nBase revision: %s\nRequirement fingerprint: %s\n\n", a.WorkItemID, a.JobID, a.AttemptID, a.BaseRevision, a.RequirementFingerprint)
	b.WriteString("## Rules\n\n")
	fmt.Fprintf(&b, "- %s\n", a.Instructions)
	b.WriteString("- The current directory is an isolated Git worktree at the base revision. Change files only here.\n")
	b.WriteString("- Satisfy every acceptance criterion of the work item below. The admitted requirements are frozen; do not edit the work item or other records.\n")
	b.WriteString("- Do not commit. Waymark commits what you leave in the worktree as the candidate.\n")
	b.WriteString("- Independent checks run after you finish. Passing tests you wrote yourself does not accept the work.\n")
	if len(a.ProtectedPaths) > 0 {
		fmt.Fprintf(&b, "- Do not create, edit, rename, or delete files matching these protected paths; verification depends on them and a candidate that touches them is rejected: %s\n", strings.Join(a.ProtectedPaths, ", "))
	}
	b.WriteString("- If a criterion cannot be met without changing requirements or protected tests, stop instead of working around it. When CTX_FACTORY_DISPUTE is set, write exactly one JSON object to that output file: {\"summary\":\"the conflict\",\"criterionRefs\":[1],\"testPaths\":[\"path/to/test\"],\"observedFailure\":\"actual failure or contradiction\",\"evidenceRefs\":[\"criterion 1; test path and failing assertion\"],\"proposedChange\":\"the correction requested\"}. Identify at least one actual criterion number or test path, cite concrete evidence, and omit references that do not apply. Then finish without committing or changing requirements or protected tests. An independent judge decides the next step; a dispute does not authorize your proposed correction. If no dispute output is configured, stop and explain the problem.\n\n")
	b.WriteString(projectInstructions(a.ProjectInstructions))
	b.WriteString("## Admitted context\n\n")
	for _, source := range a.ContextSources {
		fmt.Fprintf(&b, "### %s\n\nSHA-256: %s\nIncluded because: %s\n\n%s\n\n", source.Path, source.SHA256, strings.TrimSpace(string(source.Reasons)), source.Text)
	}
	return b.String()
}

// projectInstructions renders a project's own instructions as a section after the
// built-in rules, or nothing when the project has none.
func projectInstructions(text string) string {
	if text = strings.TrimSpace(text); text == "" {
		return ""
	}
	return "## The project's instructions\n\nA project administrator added these for you. Follow them as well; where they conflict with the rules above, the rules above win.\n\n" + text + "\n\n"
}

// agentEvent reads one JSON line of agent output, updates the result, and returns a
// short progress note for tool use, if any.
func agentEvent(harness string, line []byte, result *AgentResult) string {
	var event map[string]any
	if json.Unmarshal(line, &event) != nil {
		return ""
	}
	kind, _ := event["type"].(string)
	if harness == "codex" {
		if id, ok := event["thread_id"].(string); ok && result.SessionID == "" {
			result.SessionID = id
		}
		item, _ := event["item"].(map[string]any)
		if kind == "item.completed" && item != nil {
			itemType, _ := item["type"].(string)
			switch itemType {
			case "command_execution":
				command, _ := item["command"].(string)
				return "command: " + command
			case "file_change":
				return "edited files"
			case "agent_message":
				text, _ := item["text"].(string)
				result.Result = text
			}
		}
		if kind == "turn.failed" || kind == "error" {
			result.IsError = true
		}
		return ""
	}
	switch kind {
	case "system":
		if id, ok := event["session_id"].(string); ok {
			result.SessionID = id
		}
		if model, ok := event["model"].(string); ok {
			result.Model = model
		}
	case "assistant":
		message, _ := event["message"].(map[string]any)
		content, _ := message["content"].([]any)
		for _, part := range content {
			block, _ := part.(map[string]any)
			if block["type"] != "tool_use" {
				continue
			}
			name, _ := block["name"].(string)
			input, _ := block["input"].(map[string]any)
			for _, key := range []string{"file_path", "path", "command", "pattern"} {
				if value, ok := input[key].(string); ok {
					return name + ": " + value
				}
			}
			return name
		}
	case "result":
		result.IsError, _ = event["is_error"].(bool)
		result.Result, _ = event["result"].(string)
		if cost, ok := event["total_cost_usd"].(float64); ok {
			result.CostUSD = cost
		}
		if turns, ok := event["num_turns"].(float64); ok {
			result.Turns = int(turns)
		}
		if id, ok := event["session_id"].(string); ok {
			result.SessionID = id
		}
	}
	return ""
}

type lockedWriter struct {
	mu sync.Mutex
	w  io.Writer
}

func (l *lockedWriter) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.w.Write(p)
}

type progressFile struct{ file *os.File }

func progressWriter(path string) *progressFile {
	if path == "" {
		return &progressFile{}
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o600)
	if err != nil {
		return &progressFile{}
	}
	return &progressFile{file: file}
}

func (p *progressFile) note(text string) {
	if p.file != nil {
		fmt.Fprintln(p.file, strings.ReplaceAll(text, "\n", " "))
	}
}

func (p *progressFile) Close() {
	if p.file != nil {
		_ = p.file.Close()
	}
}
