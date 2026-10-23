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

const expectationInstruction = "Write the holdout tests described in the input. Do not implement the feature. Reply only with the JSON object the schema asks for."

// ExpectationSchema is the structured report the author returns after writing tests:
// which tests check which criterion, counted from 1, and any problems it found.
const ExpectationSchema = `{"type":"object","properties":{"notes":{"type":"string"},"coverage":{"type":"array","items":{"type":"object","properties":{"criterion":{"type":"integer","minimum":1},"tests":{"type":"array","items":{"type":"string"}}},"required":["criterion","tests"],"additionalProperties":false}},"concerns":{"type":"array","items":{"type":"string"}}},"required":["notes","coverage","concerns"],"additionalProperties":false}`

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
	argv := []string{binary, "-p", expectationInstruction, "--output-format", "json", "--json-schema", ExpectationSchema, "--permission-mode", "acceptEdits", "--no-session-persistence"}
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
	output, err := parseExpectationResult(stdout.Bytes(), options.Model)
	if err != nil {
		return err
	}
	return json.NewEncoder(out).Encode(output)
}

// parseExpectationResult reads Claude Code's JSON result. The structured report is
// optional: without it the reply text becomes the notes and coverage stays unknown.
func parseExpectationResult(raw []byte, model string) (factory.ExpectationOutput, error) {
	var result struct {
		IsError          bool            `json:"is_error"`
		Result           string          `json:"result"`
		StructuredOutput json.RawMessage `json:"structured_output"`
		SessionID        string          `json:"session_id"`
		Turns            int             `json:"num_turns"`
		CostUSD          float64         `json:"total_cost_usd"`
		ModelUsage       map[string]any  `json:"modelUsage"`
	}
	if err := json.Unmarshal(bytes.TrimSpace(raw), &result); err != nil {
		return factory.ExpectationOutput{}, fmt.Errorf("claude expectation author returned no result: %w", err)
	}
	if result.IsError {
		return factory.ExpectationOutput{}, fmt.Errorf("claude expectation author reported an error: %s", result.Result)
	}
	for name := range result.ModelUsage {
		if model == "" || !strings.Contains(name, "haiku") {
			model = name
		}
	}
	output := factory.ExpectationOutput{Notes: result.Result, Run: factory.RefinerRun{Model: model, SessionID: result.SessionID, Turns: result.Turns, CostUSD: result.CostUSD}}
	payload := bytes.TrimSpace(result.StructuredOutput)
	if len(payload) == 0 || string(payload) == "null" {
		payload = []byte(jsonObject(result.Result))
	}
	var report struct {
		Notes    string                      `json:"notes"`
		Coverage []factory.CriterionCoverage `json:"coverage"`
		Concerns []string                    `json:"concerns"`
	}
	if len(payload) > 0 && json.Unmarshal(payload, &report) == nil {
		output.Coverage, output.Concerns = report.Coverage, report.Concerns
		if strings.TrimSpace(report.Notes) != "" {
			output.Notes = report.Notes
		}
	}
	return output, nil
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
	b.WriteString("- Finish with the JSON report: notes, a short summary for the person who reads your tests; coverage, the names of the tests that check each criterion, using the criterion numbers below, with an empty list for a criterion no test can check; and concerns, each contradiction or problem you found in the criteria.\n\n")
	b.WriteString(projectInstructions(input.Instructions))
	if input.Dispute != nil {
		b.WriteString("## Judge-directed revision\n\nRevise only the protected tests identified by this independent decision, against the captured criteria and base revision. Preserve valid existing coverage; do not weaken assertions to fit the implementation. Do not change production code or requirements. If the directive needs changed intent, weaker acceptance, or unsupported assumptions, leave the affected tests unchanged and report concerns for a person. The old suite and run remain retained; your patch needs approval under project policy.\n\n")
		decision, _ := json.MarshalIndent(input.Dispute, "", "  ")
		b.Write(decision)
		if input.OriginalTests != nil {
			original, _ := json.MarshalIndent(input.OriginalTests, "", "  ")
			b.WriteString("\n\n### Original test suite and coverage\n\n")
			b.Write(original)
		}
		fmt.Fprintf(&b, "\n\nRevision base: %s\n\n", input.BaseRevision)
	}
	if len(input.Criteria) > 0 {
		b.WriteString("## Criteria by number\n\n")
		for index, criterion := range input.Criteria {
			fmt.Fprintf(&b, "%d. %s\n", index+1, criterion)
		}
		b.WriteString("\n")
	}
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
