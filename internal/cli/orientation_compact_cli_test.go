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
		"Record store: /bundle", "Keep this exact authored goal.\n\nAnd its second paragraph.",
		"execution: unstarted; readiness: blocked", "Needs attention:", "In progress:", "Active work [active]",
		"New pickup shortlist:", "Next work [next]", "ctx resume --ticket <work-item-source> with the same scope",
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
		strings.Count(shared, "source: decision.md") != 1 || strings.Contains(shared, "link ") || strings.Contains(shared, "from ") {
		t.Fatalf("group lost affected order or repeated routine provenance: %s", shared)
	}
	for _, clutter := range []string{"source: /bundle/project.md", "/docs/goal.md", "Project references:", "source: /bundle/z.md", "source: /bundle/a.md"} {
		if strings.Contains(text, clutter) {
			t.Fatalf("compact output retained routine provenance %q: %s", clutter, text)
		}
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

func TestCompactDecisionAffectedWorkUsesCommitmentOrder(t *testing.T) {
	first := orientation.Reference{Path: "/bundle/z.md", ID: str("z"), Title: str("First commitment")}
	second := orientation.Reference{Path: "/bundle/a.md", ID: str("a"), Title: str("Second commitment")}
	result := orientation.Result{
		SchemaVersion: 1, Complete: true, InventoryComplete: true,
		Project:            &orientation.Project{ID: "project", Title: "Project", Source: "/bundle/project.md", GoalsKnown: true, CommitmentsKnown: true, OpenDecisionsKnown: true},
		CurrentCommitments: []orientation.Reference{first, second},
		Decisions: []orientation.Decision{{
			ID: str("choice"), Title: str("Shared choice"), Source: "/bundle/choice.md", State: str("open"), CheckStatus: "fail",
			AffectedWork: []orientation.Reference{second, first},
			Reasons:      []orientation.Finding{{Code: "open_decision", Message: "Decision is open", Path: "/bundle/choice.md"}},
		}},
	}
	var stdout, stderr bytes.Buffer
	status := cli.Run(context.Background(), []string{"ctx", "orient", "--bundle", "."}, &stdout, &stderr, cli.Operations{Orient: func(context.Context, orientation.Request) (orientation.Result, error) { return result, nil }})
	text := stdout.String()
	start := strings.Index(text, "decision/open_decision:")
	if status != 0 || stderr.Len() != 0 || start < 0 {
		t.Fatalf("status=%d stderr=%q output=%s", status, stderr.String(), text)
	}
	row := text[start:]
	firstIndex, secondIndex := strings.Index(row, "First commitment [z]"), strings.Index(row, "Second commitment [a]")
	if firstIndex < 0 || secondIndex < firstIndex {
		t.Fatalf("decision row lost authored commitment order: %s", row)
	}
}

func TestCompactOrientationShortensOnlyPathsInsideTheRecordStore(t *testing.T) {
	missingID := orientation.Reference{Path: "/records/alpha/task.md", Title: str("Same filename")}
	missingTitle := orientation.Reference{Path: "/records/beta/task.md", ID: str("beta")}
	ambiguous := orientation.Reference{Path: "/records/gamma/task.md", ID: str("duplicate"), Title: str("Ambiguous task")}
	normal := orientation.Reference{Path: "/records/work/normal.md", ID: str("normal"), Title: str("Normal task"), From: "/records/project.md", Link: "work/normal.md"}
	result := orientation.Result{
		SchemaVersion: 1, Complete: true, InventoryComplete: true,
		Project: &orientation.Project{ID: "paths", Title: "Path project", Source: "/records/project.md", GoalsKnown: true, CommitmentsKnown: true, OpenDecisionsKnown: true},
		Goals: []orientation.Goal{{
			Text:       "  Preserve these authored spaces.\nAnd this line.\n",
			Source:     "/records/project.md",
			References: []orientation.Reference{{Path: "/external/goal.md", From: "/records/project.md", Link: "../external/goal.md"}},
		}},
		CurrentCommitments: []orientation.Reference{missingID, missingTitle, ambiguous, normal},
		WorkItems: []orientation.WorkItem{
			{Title: missingID.Title, Source: missingID.Path, Execution: str("unstarted"), Readiness: "unknown"},
			{ID: missingTitle.ID, Source: missingTitle.Path, Execution: str("unstarted"), Readiness: "unknown"},
			{ID: ambiguous.ID, Title: ambiguous.Title, Source: ambiguous.Path, IdentityAmbiguous: true, Execution: str("unstarted"), Readiness: "unknown"},
			{ID: normal.ID, Title: normal.Title, Source: normal.Path, Execution: str("unstarted"), Readiness: "blocked"},
		},
		InProgress: []orientation.Reference{normal},
		Shortlist:  []orientation.Reference{normal},
		Sources: []orientation.Source{{Path: "/records/project.md", Reasons: []orientation.Reason{{Kind: "project"}}}, {
			Path: "/external/goal.md", Reasons: []orientation.Reason{{Kind: "goal", From: "/records/project.md", Link: "../external/goal.md"}},
		}},
		Diagnostics: []orientation.Diagnostic{
			{Code: "internal_problem", Severity: "warning", Message: "Internal problem.", Path: "/records/findings/problem.md", From: normal.Path, Link: "../findings/problem.md"},
			{Code: "ambiguous_problem", Severity: "warning", Message: "Ambiguous work problem.", Path: "/records/findings/ambiguous.md", From: ambiguous.Path, Link: "../findings/ambiguous.md"},
			{Code: "sibling_problem", Severity: "warning", Message: "Sibling problem.", Path: "/records-copy/problem.md"},
			{Code: "external_problem", Severity: "warning", Message: "External problem.", Path: "/external/problem.md"},
			{Code: "unresolved_reference", Severity: "error", Message: "Relationship is unresolved.", Path: normal.Path, From: normal.Path, Link: "[missing]"},
		},
	}
	original, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}

	compact := runOrientationResult(t, result)
	for _, wanted := range []string{
		"Record store: /records", "  Preserve these authored spaces.\nAnd this line.\n",
		"Same filename [unknown] alpha/task.md", "unknown [beta] beta/task.md", "Ambiguous task [duplicate] gamma/task.md",
		"source: findings/problem.md", "source: /records-copy/problem.md", "source: /external/problem.md",
		"source: work/normal.md (from work/normal.md; link \"[missing]\")",
	} {
		if !strings.Contains(compact, wanted) {
			t.Errorf("compact output missing %q:\n%s", wanted, compact)
		}
	}
	if strings.Count(compact, "Record store: /records") != 1 {
		t.Errorf("compact output did not declare the record store exactly once:\n%s", compact)
	}
	for _, unwanted := range []string{
		"Normal task [normal] /records/work/normal.md", "source: /records/project.md", "/external/goal.md",
		"link \"../findings/problem.md\"", "Project references:",
	} {
		if strings.Contains(compact, unwanted) {
			t.Errorf("compact output retained %q:\n%s", unwanted, compact)
		}
	}
	normalStart := strings.Index(compact, "warning internal_problem:")
	ambiguousStart := strings.Index(compact, "warning ambiguous_problem:")
	siblingStart := strings.Index(compact, "warning sibling_problem:")
	if normalStart < 0 || ambiguousStart <= normalStart || siblingStart <= ambiguousStart {
		t.Fatalf("attention causes are missing or out of order:\n%s", compact)
	}
	normalCause := compact[normalStart:ambiguousStart]
	if !strings.Contains(normalCause, "      Normal task [normal]\n") || strings.Contains(normalCause, "Normal task [normal] work/normal.md") {
		t.Errorf("affected work repeated an unambiguous task path:\n%s", normalCause)
	}
	ambiguousCause := compact[ambiguousStart:siblingStart]
	if !strings.Contains(ambiguousCause, "      Ambiguous task [duplicate] gamma/task.md\n") {
		t.Errorf("affected work omitted the path needed for ambiguous identity:\n%s", ambiguousCause)
	}

	detail := runOrientationResult(t, result, "--detail")
	for _, retained := range []string{"source: /records/project.md", "/records/alpha/task.md", "/external/goal.md", "from /records/project.md; link \"../external/goal.md\""} {
		if !strings.Contains(detail, retained) {
			t.Errorf("detail output lost %q:\n%s", retained, detail)
		}
	}
	jsonOutput := runOrientationResult(t, result, "--json")
	if jsonOutput != string(original)+"\n" {
		t.Fatalf("JSON changed:\n got %s\nwant %s", jsonOutput, original)
	}
	after, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(after, original) {
		t.Fatalf("compact rendering mutated the result:\n before %s\n after %s", original, after)
	}
}

func TestCompactOrientationPreservesPathsWhenProjectRootIsUnknown(t *testing.T) {
	result := orientation.Result{
		SchemaVersion: 1, Complete: false, InventoryComplete: false,
		CurrentCommitments: []orientation.Reference{{Path: "/records/one/task.md", Title: str("Unknown identity")}},
		Diagnostics:        []orientation.Diagnostic{{Code: "missing_project", Severity: "error", Message: "Project identity is unavailable.", Path: "/records/project.md"}},
	}
	text := runOrientationResult(t, result)
	for _, wanted := range []string{"Project: unknown", "Record store: unknown", "Unknown identity [unknown] /records/one/task.md", "source: /records/project.md"} {
		if !strings.Contains(text, wanted) {
			t.Errorf("unknown-root output missing %q:\n%s", wanted, text)
		}
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

func runOrientationResult(t *testing.T, result orientation.Result, flags ...string) string {
	t.Helper()
	args := append([]string{"ctx", "orient", "--bundle", "."}, flags...)
	var stdout, stderr bytes.Buffer
	status := cli.Run(context.Background(), args, &stdout, &stderr, cli.Operations{Orient: func(context.Context, orientation.Request) (orientation.Result, error) {
		return result, nil
	}})
	wantStatus := 0
	if !result.Complete {
		wantStatus = 1
	}
	if status != wantStatus || stderr.Len() != 0 {
		t.Fatalf("status=%d want=%d stderr=%q stdout=%q", status, wantStatus, stderr.String(), stdout.String())
	}
	return stdout.String()
}
