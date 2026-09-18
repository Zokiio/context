package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/Zokiio/context/internal/cli"
	"github.com/Zokiio/context/internal/orientation"
)

func TestCompactOrientationGroupsEquivalentCausesInCommitmentOrder(t *testing.T) {
	result := orientation.Result{
		SchemaVersion: 1, Complete: true, InventoryComplete: true,
		Project: &orientation.Project{ID: "project", Title: "Compact project", Source: "/bundle/project.md", GoalsKnown: true, CommitmentsKnown: true, OpenDecisionsKnown: true},
		Goals:   []orientation.Goal{{Text: "Keep this exact authored goal.\n\nAnd its second paragraph.", Source: "/bundle/project.md", References: []orientation.Reference{{Path: "/docs/goal.md", From: "/bundle/project.md", Link: "../docs/goal.md"}}}},
		CurrentCommitments: []orientation.Reference{
			{Path: "/bundle/z.md", ID: str("z"), Title: str("First authored"), From: "/bundle/project.md", Link: "z.md"},
			{Path: "/bundle/a.md", ID: str("a"), Title: str("Second authored"), From: "/bundle/project.md", Link: "a.md"},
		},
		WorkItems: []orientation.WorkItem{
			compactWork("a", "Second authored", "/bundle/a.md", "/bundle/decision.md", "/bundle/a.md", "decision.md"),
			compactWork("z", "First authored", "/bundle/z.md", "/bundle/decision.md", "/bundle/z.md", "./decision.md"),
			compactWork("other", "Other work", "/bundle/other.md", "/bundle/other-decision.md", "/bundle/other.md", "decision.md"),
		},
		InProgress: []orientation.Reference{{Path: "/bundle/active.md", ID: str("active"), Title: str("Active work")}},
		Shortlist:  []orientation.Reference{{Path: "/bundle/next.md", ID: str("next"), Title: str("Next work")}},
		Sources: []orientation.Source{
			{Path: "/bundle/project.md", Reasons: []orientation.Reason{{Kind: "project"}}},
			{Path: "/docs/goal.md", Reasons: []orientation.Reason{{Kind: "goal", From: "/bundle/project.md", Link: "../docs/goal.md"}}},
		},
		Diagnostics: []orientation.Diagnostic{
			{Code: "same_words", Severity: "warning", Message: "Same prose.", Path: "/one.md"},
			{Code: "same_words", Severity: "warning", Message: "Same prose.", Path: "/two.md"},
		},
	}

	var stdout, stderr bytes.Buffer
	status := cli.Run(context.Background(), []string{"ctx", "orient", "--bundle", "."}, &stdout, &stderr, cli.Operations{Orient: func(context.Context, orientation.Request) (orientation.Result, error) { return result, nil }})
	if status != 0 || stderr.Len() != 0 {
		t.Fatalf("status=%d stderr=%q", status, stderr.String())
	}
	text := stdout.String()
	for _, fact := range []string{
		"Keep this exact authored goal.\n\nAnd its second paragraph.", "source: /bundle/project.md",
		"execution: unstarted; readiness: blocked", "Needs attention:", "In progress:", "Active work [active]",
		"New pickup shortlist:", "Next work [next]", "/docs/goal.md", "ctx resume --ticket <work-item-source> is forthcoming",
	} {
		if !strings.Contains(text, fact) {
			t.Errorf("compact output missing %q:\n%s", fact, text)
		}
	}
	first, second := strings.Index(text, "First authored [z]"), strings.Index(text, "Second authored [a]")
	if first < 0 || second <= first {
		t.Fatalf("commitment order changed: %s", text)
	}
	if strings.Count(text, "blocking_decisions/open_decision:") != 1 {
		t.Fatalf("the shared commitment cause was not grouped: %s", text)
	}
	sharedStart := strings.Index(text, "blocking_decisions/open_decision:")
	sharedEnd := strings.Index(text[sharedStart:], "warning same_words:") + sharedStart
	shared := text[sharedStart:sharedEnd]
	if strings.Index(shared, "First authored [z]") > strings.Index(shared, "Second authored [a]") ||
		!strings.Contains(shared, "from /bundle/z.md; link \"./decision.md\"") ||
		!strings.Contains(shared, "from /bundle/a.md; link \"decision.md\"") {
		t.Fatalf("group lost affected order or relationships: %s", shared)
	}
	if strings.Count(text, "warning same_words:") != 2 {
		t.Fatalf("equal prose merged distinct diagnostic identities: %s", text)
	}
	if strings.Contains(text, "routine pass") {
		t.Fatalf("compact output included routine pass detail: %s", text)
	}
}

func TestCompactOrientationKeepsUnknownAndPartialFacts(t *testing.T) {
	result := orientation.Result{
		SchemaVersion: 1, Complete: false, InventoryComplete: false,
		Project:            &orientation.Project{ID: "partial", Title: "Partial", Source: "/bundle/project.md", GoalsKnown: false, CommitmentsKnown: false},
		CurrentCommitments: []orientation.Reference{{Path: "/bundle/unknown.md", ID: str("unknown"), Title: str("Unknown work")}},
		WorkItems:          []orientation.WorkItem{{ID: str("unknown"), Title: str("Unknown work"), Source: "/bundle/unknown.md", Checks: []orientation.Check{{Name: "dependencies", Status: "unknown"}}, Readiness: "unknown", ExclusionReasons: []orientation.Finding{{Code: "execution_unknown", Message: "execution state is unknown", Path: "/bundle/unknown.md"}}}},
		Shortlist:          []orientation.Reference{{Path: "/bundle/unsafe.md", ID: str("unsafe"), Title: str("Must not be shown")}},
		Diagnostics:        []orientation.Diagnostic{{Code: "source_omitted", Severity: "error", Message: "A source was omitted.", Path: "/bundle/later.md"}, {Code: "inventory_warning", Severity: "warning", Message: "Inventory warning.", Path: "/bundle/project.md"}},
	}
	var stdout, stderr bytes.Buffer
	status := cli.Run(context.Background(), []string{"ctx", "orient", "--bundle", "."}, &stdout, &stderr, cli.Operations{Orient: func(context.Context, orientation.Request) (orientation.Result, error) { return result, nil }})
	text := stdout.String()
	if status != 1 || stderr.Len() != 0 {
		t.Fatalf("status=%d stderr=%q output=%s", status, stderr.String(), text)
	}
	for _, fact := range []string{"Evaluation: partial; inventory: partial", "Goals: unknown", "Current commitments: unknown", "dependencies/check_unknown", "eligibility/execution_unknown", "error source_omitted", "warning inventory_warning", "New pickup shortlist: suppressed because inventory is partial"} {
		if !strings.Contains(text, fact) {
			t.Errorf("partial output missing %q:\n%s", fact, text)
		}
	}
	if strings.Contains(text, "Must not be shown") {
		t.Fatalf("partial inventory exposed a pickup candidate: %s", text)
	}
}

func TestOrientationJSONIsUnchangedAndDetailConflicts(t *testing.T) {
	result := orientation.Result{SchemaVersion: 1, Complete: true, InventoryComplete: true, Goals: []orientation.Goal{}, CurrentCommitments: []orientation.Reference{}, WorkItems: []orientation.WorkItem{}, Shortlist: []orientation.Reference{}, InProgress: []orientation.Reference{}, Backlog: []orientation.Reference{}, Decisions: []orientation.Decision{}, Sources: []orientation.Source{}, Diagnostics: []orientation.Diagnostic{}}
	operation := func(context.Context, orientation.Request) (orientation.Result, error) { return result, nil }
	var stdout, stderr bytes.Buffer
	status := cli.Run(context.Background(), []string{"ctx", "orient", "--bundle", ".", "--json"}, &stdout, &stderr, cli.Operations{Orient: operation})
	want, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if status != 0 || stderr.Len() != 0 || stdout.String() != string(want)+"\n" {
		t.Fatalf("JSON changed: status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	called := false
	status = cli.Run(context.Background(), []string{"ctx", "orient", "--bundle", ".", "--detail", "--json"}, &stdout, &stderr, cli.Operations{Orient: func(context.Context, orientation.Request) (orientation.Result, error) {
		called = true
		return result, nil
	}})
	if status != 2 || called || stdout.Len() != 0 || !strings.Contains(stderr.String(), "--detail") || !strings.Contains(stderr.String(), "--json") {
		t.Fatalf("detail/json conflict: status=%d called=%t stdout=%q stderr=%q", status, called, stdout.String(), stderr.String())
	}
}

func compactWork(id, title, path, causePath, from, link string) orientation.WorkItem {
	committed := true
	return orientation.WorkItem{
		ID: str(id), Title: str(title), Source: path, Execution: str("unstarted"), Committed: &committed, Readiness: "blocked",
		Checks: []orientation.Check{{Name: "blocking_decisions", Status: "fail", Reasons: []orientation.Finding{
			{Code: "acceptance_criteria_present", Message: "routine pass", Path: path},
			{Code: "open_decision", Message: "The same words can describe different causes.", Path: causePath, From: from, Link: link},
		}}},
	}
}
