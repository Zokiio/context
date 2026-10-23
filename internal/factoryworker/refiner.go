package factoryworker

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/Zokiio/context/internal/factory"
)

// RefinerOptions configures the read-only agent that proposes refined criteria.
type RefinerOptions struct {
	Binary       string
	Model        string
	MaxBudgetUSD float64
}

const maxRefinerContextRunes = 150_000

const refinerInstruction = "Refine the work item described in the input. Reply only with the JSON object the schema asks for."

// RefinerSchema is the structured output the refiner must return.
const RefinerSchema = `{"type":"object","properties":{"criteria":{"type":"array","minItems":1,"maxItems":20,"items":{"type":"string"}},"questions":{"type":"array","maxItems":3,"items":{"type":"object","properties":{"question":{"type":"string"},"unblocks":{"type":"string"},"options":{"type":"array","maxItems":4,"items":{"type":"string"}}},"required":["question","unblocks","options"],"additionalProperties":false}},"notes":{"type":"string"}},"required":["criteria","questions","notes"],"additionalProperties":false}`

// Refiner reads a RefinerInput on in, asks Claude Code for a proposal with read-only
// tools, and writes a RefinerOutput on out. It never edits files: the only tools the
// agent has are Read, Grep, and Glob.
func Refiner(ctx context.Context, in io.Reader, out, log io.Writer, options RefinerOptions) error {
	var input factory.RefinerInput
	if err := json.NewDecoder(io.LimitReader(in, 64<<20)).Decode(&input); err != nil {
		return err
	}
	binary := options.Binary
	if binary == "" {
		binary = "claude"
	}
	argv := []string{binary, "-p", refinerInstruction, "--output-format", "json", "--json-schema", RefinerSchema, "--no-session-persistence"}
	if options.Model != "" {
		argv = append(argv, "--model", options.Model)
	}
	if options.MaxBudgetUSD > 0 {
		argv = append(argv, "--max-budget-usd", fmt.Sprint(options.MaxBudgetUSD))
	}
	argv = append(argv, "--tools", "Read", "Grep", "Glob")
	command := exec.CommandContext(ctx, argv[0], argv[1:]...)
	command.Stdin = strings.NewReader(RefinerPrompt(input))
	var stdout bytes.Buffer
	command.Stdout, command.Stderr = &stdout, log
	if err := command.Run(); err != nil {
		return fmt.Errorf("claude refiner: %w", err)
	}
	output, err := parseRefinerResult(stdout.Bytes(), options.Model)
	if err != nil {
		return err
	}
	return json.NewEncoder(out).Encode(output)
}

func parseRefinerResult(raw []byte, model string) (factory.RefinerOutput, error) {
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
		return factory.RefinerOutput{}, fmt.Errorf("claude refiner returned no result: %w", err)
	}
	if result.IsError {
		return factory.RefinerOutput{}, fmt.Errorf("claude refiner reported an error: %s", result.Result)
	}
	for name := range result.ModelUsage {
		if model == "" || !strings.Contains(name, "haiku") {
			model = name
		}
	}
	output := factory.RefinerOutput{Run: factory.RefinerRun{Model: model, SessionID: result.SessionID, Turns: result.Turns, CostUSD: result.CostUSD}}
	payload := []byte(result.StructuredOutput)
	if len(bytes.TrimSpace(payload)) == 0 || string(bytes.TrimSpace(payload)) == "null" {
		payload = []byte(jsonObject(result.Result))
	}
	if err := json.Unmarshal(payload, &output.Proposal); err != nil {
		return factory.RefinerOutput{}, fmt.Errorf("claude refiner returned no proposal: %w", err)
	}
	if len(output.Proposal.Criteria) == 0 {
		return factory.RefinerOutput{}, errors.New("claude refiner proposed no criteria")
	}
	return output, nil
}

// jsonObject extracts the outermost JSON object from text that may wrap it in prose
// or a code fence.
func jsonObject(text string) string {
	start, end := strings.Index(text, "{"), strings.LastIndex(text, "}")
	if start < 0 || end < start {
		return ""
	}
	return text[start : end+1]
}

// RefinerPrompt renders the refinement rules, the decision-model flags, any earlier
// proposal and answers, the ticket, and its admitted context.
func RefinerPrompt(input factory.RefinerInput) string {
	var b strings.Builder
	b.WriteString("# Refine acceptance criteria before implementation\n\n")
	b.WriteString("An AI coding agent will implement this work item. Admission freezes its acceptance criteria, and independent checks then hold the implementation to them, so vague or untestable criteria get locked in. Propose better criteria now.\n\n")
	b.WriteString("## Rules\n\n")
	b.WriteString("- You can read the repository and the context below. You cannot edit anything.\n")
	b.WriteString("- Rewrite each criterion so it names a concrete result someone could check as met or not met: a command and its outcome, an output, a behaviour, or a file.\n")
	b.WriteString("- Keep the work item's intent and scope. Do not add features, drop requirements, or weaken a criterion. Where intent is unclear, write the most likely concrete reading and ask about it.\n")
	fmt.Fprintf(&b, "- Ask at most %d questions, and only ones whose answer would change the criteria or the implementation. Say what each answer unblocks, and offer two to four likely answers as options, your best guess first. Zero questions is better than filler.\n", input.MaxQuestions)
	b.WriteString("- Record any assumption you made in notes.\n")
	if len(input.ProtectedPaths) > 0 {
		fmt.Fprintf(&b, "- The implementer may not change files matching %s. Criteria may rely on those tests as they are.\n", strings.Join(input.ProtectedPaths, ", "))
	}
	if input.Signals != nil && len(input.Signals.Criteria) > 0 {
		fmt.Fprintf(&b, "\n## Decision-model flags\n\n%s estimated how likely each current criterion is to name a checkable result. The scores are advisory and uncalibrated; flagged ones fell below %.2f.\n\n", input.Signals.Model, input.Signals.Threshold)
		for _, signal := range input.Signals.Criteria {
			mark := ""
			if signal.Flagged {
				mark = " (flagged)"
			}
			score := "unscored"
			if signal.Checkable != nil {
				score = fmt.Sprintf("%.2f", *signal.Checkable)
			}
			fmt.Fprintf(&b, "- %s%s: %s\n", score, mark, signal.Text)
		}
	}
	if input.PreviousProposal != nil {
		b.WriteString("\n## Your previous proposal\n\n")
		for _, criterion := range input.PreviousProposal.Criteria {
			fmt.Fprintf(&b, "- %s\n", criterion)
		}
	}
	if len(input.PreviousAnswers) > 0 {
		b.WriteString("\n## Answers from the person refining this work item\n\nUse these answers. Do not ask these questions again.\n\n")
		for _, answer := range input.PreviousAnswers {
			fmt.Fprintf(&b, "- Q: %s\n  A: %s\n", answer.Question, answer.Answer)
		}
	}
	fmt.Fprintf(&b, "\n## Work item %s (%s)\n\n%s\n", input.WorkItemID, input.TicketPath, input.Ticket)
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
