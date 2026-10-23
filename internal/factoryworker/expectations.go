package factoryworker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/Zokiio/context/internal/factory"
)

// ExpectationAuthorOptions configures the agent that writes holdout tests.
type ExpectationAuthorOptions struct {
	Binary       string
	Model        string
	MaxBudgetUSD float64
	AllowedTools []string
}

const expectationInstruction = "Write the holdout tests described in the input. Do not implement the feature."

// ExpectationAuthor reads an ExpectationInput on in and lets Claude Code write tests in
// the current directory, a temporary worktree at the base revision. The service keeps
// the result only if every changed file is protected, so the author cannot also change
// production code.
func ExpectationAuthor(ctx context.Context, in io.Reader, out, log io.Writer, options ExpectationAuthorOptions) error {
	var input factory.ExpectationInput
	if err := json.NewDecoder(io.LimitReader(in, 64<<20)).Decode(&input); err != nil {
		return err
	}
	binary := options.Binary
	if binary == "" {
		binary = "claude"
	}
	argv := []string{binary, "-p", expectationInstruction, "--output-format", "json", "--permission-mode", "acceptEdits", "--no-session-persistence"}
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
	command := exec.CommandContext(ctx, argv[0], argv[1:]...)
	command.Stdin = strings.NewReader(ExpectationPrompt(input))
	var stdout bytes.Buffer
	command.Stdout, command.Stderr = &stdout, log
	if err := command.Run(); err != nil {
		return fmt.Errorf("claude expectation author: %w", err)
	}
	var result struct {
		IsError    bool           `json:"is_error"`
		Result     string         `json:"result"`
		SessionID  string         `json:"session_id"`
		Turns      int            `json:"num_turns"`
		CostUSD    float64        `json:"total_cost_usd"`
		ModelUsage map[string]any `json:"modelUsage"`
	}
	if err := json.Unmarshal(bytes.TrimSpace(stdout.Bytes()), &result); err != nil {
		return fmt.Errorf("claude expectation author returned no result: %w", err)
	}
	if result.IsError {
		return fmt.Errorf("claude expectation author reported an error: %s", result.Result)
	}
	model := options.Model
	for name := range result.ModelUsage {
		if model == "" || !strings.Contains(name, "haiku") {
			model = name
		}
	}
	return json.NewEncoder(out).Encode(factory.ExpectationOutput{Notes: result.Result, Run: factory.RefinerRun{Model: model, SessionID: result.SessionID, Turns: result.Turns, CostUSD: result.CostUSD}})
}

// ExpectationPrompt renders the rules for an independent test author, then the work item
// and its context.
func ExpectationPrompt(input factory.ExpectationInput) string {
	var b strings.Builder
	b.WriteString("# Write holdout expectations\n\n")
	b.WriteString("You write the tests an AI implementer will be held to. You never implement the work. The implementer will not be able to change your tests, and independent checks will run them against its result.\n\n")
	b.WriteString("## Rules\n\n")
	fmt.Fprintf(&b, "- Create or edit files only under these protected paths: %s. Any other change rejects your whole suite.\n", strings.Join(input.ProtectedPaths, ", "))
	b.WriteString("- Follow the repository's existing test conventions: language, package, framework, and file naming.\n")
	b.WriteString("- Write one focused test per acceptance criterion that a test can check. Keep the suite small: a person must read every line, and the implementer cannot fix a wrong test.\n")
	b.WriteString("- The tests must compile and pass once the criteria are met. They are expected to fail now, because the work does not exist yet. Do not add production code, stubs, or placeholders.\n")
	b.WriteString("- Derive every expected value from the criteria. If a criterion's example contradicts its own rule, do not encode the contradiction; leave it out and report it.\n")
	b.WriteString("- Do not edit the work item or any record.\n")
	b.WriteString("- Finish with a short summary: which test covers which criterion, which criteria no test covers and why, and any contradictions you found.\n\n")
	fmt.Fprintf(&b, "## Work item %s (%s)\n\n%s\n", input.WorkItemID, input.TicketPath, input.Ticket)
	b.WriteString("\n## Context\n\n")
	budget := maxRefinerContextRunes
	for _, source := range input.Context {
		if source.Path == input.TicketPath {
			continue
		}
		text := source.Text
		if runes := []rune(text); len(runes) > budget {
			text = string(runes[:budget]) + "\n[truncated]"
		}
		budget -= len([]rune(text))
		fmt.Fprintf(&b, "### %s\n\n%s\n\n", source.Path, text)
		if budget <= 0 {
			b.WriteString("[further context omitted]\n")
			break
		}
	}
	return b.String()
}
