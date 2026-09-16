package cli_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Zokiio/context/internal/cli"
	"github.com/Zokiio/context/internal/orientation"
)

func TestOrientationTextRetainsPartialFacts(t *testing.T) {
	var stdout, stderr bytes.Buffer
	result := orientation.Result{
		SchemaVersion: 1, InventoryComplete: true,
		Project: &orientation.Project{ID: "project-1", Title: "Known project", Source: "/bundle/project.md"},
		WorkItems: []orientation.WorkItem{{
			ID: str("work-1"), Title: str("Known task"), Source: "/bundle/task.md",
			Triage: str("ready-for-agent"), Execution: str("completed"), Lifecycle: str("stable"), Readiness: "unknown",
			Checks:           []orientation.Check{{Name: "acceptance_criteria", Status: "unknown", Reasons: []orientation.Finding{{Code: "unsupported_check", Message: "This check is unavailable."}}}},
			ExclusionReasons: []orientation.Finding{{Code: "completed_work", Message: "Completed work is not eligible."}},
			Specifications:   []orientation.Reference{{Path: "/docs/spec.md", From: "/bundle/task.md", Link: "../docs/spec.md"}},
		}},
		Decisions:   []orientation.Decision{{ID: str("decision-1"), Title: str("Choice"), Source: "/bundle/choice.md"}},
		Sources:     []orientation.Source{{Path: "/bundle/project.md", SHA256: strings.Repeat("a", 64), Reasons: []orientation.Reason{{Kind: "manifest"}}}},
		Diagnostics: []orientation.Diagnostic{{Code: "missing_required_section", Severity: "error", Message: "Goals is missing.", Path: "/bundle/project.md"}},
	}
	status := cli.Run(context.Background(), []string{"ctx", "orient", "--project", "."}, &stdout, &stderr, cli.Operations{Orient: func(context.Context, orientation.Request) (orientation.Result, error) {
		return result, nil
	}})
	if status != 1 || stderr.Len() != 0 {
		t.Fatalf("status=%d stderr=%q", status, stderr.String())
	}
	for _, fact := range []string{
		"Project: Known project [project-1]", "Evaluation: partial; inventory: complete",
		"Goals: unknown", "Current commitments: unknown", "Open decisions: unknown",
		"Known task [work-1]", "triage: ready-for-agent", "execution: completed", "lifecycle: stable",
		"committed: unknown", "readiness: unknown", "eligible: false", "acceptance_criteria: unknown",
		"unsupported_check", "completed_work", "acceptance: unknown", "fingerprint: unknown",
		"/docs/spec.md", "../docs/spec.md", "Choice [decision-1]", "state: unknown", "resolution: unknown",
		"missing_required_section", "Goals is missing.", "/bundle/project.md", strings.Repeat("a", 64),
	} {
		if !strings.Contains(stdout.String(), fact) {
			t.Errorf("missing %q in text report:\n%s", fact, stdout.String())
		}
	}
}

func str(value string) *string { return &value }

func TestOrientationTextShowsAmbiguousIdentities(t *testing.T) {
	var stdout, stderr bytes.Buffer
	result := orientation.Result{
		SchemaVersion: 1,
		WorkItems:     []orientation.WorkItem{{ID: str("shared"), Title: str("Task"), Source: "/bundle/task.md", IdentityAmbiguous: true, Readiness: "unknown"}},
		Decisions:     []orientation.Decision{{ID: str("shared"), Title: str("Choice"), Source: "/bundle/choice.md", IdentityAmbiguous: true}},
	}
	status := cli.Run(context.Background(), []string{"ctx", "orient", "--project", "."}, &stdout, &stderr, cli.Operations{Orient: func(context.Context, orientation.Request) (orientation.Result, error) {
		return result, nil
	}})
	if status != 1 || stderr.Len() != 0 || strings.Count(stdout.String(), "identity ambiguous: true") != 2 {
		t.Fatalf("ambiguous work and decision identities must both be visible: status=%d stderr=%q stdout=%q", status, stderr.String(), stdout.String())
	}
}

func TestOrientationTextUsesAuthoritativeDecisionState(t *testing.T) {
	var stdout, stderr bytes.Buffer
	project := &orientation.Project{ID: "project", Title: "Project", Source: "/bundle/project.md", GoalsKnown: true, CommitmentsKnown: true, OpenDecisionsKnown: true}
	result := orientation.Result{
		SchemaVersion: 1, Complete: true, InventoryComplete: true, Project: project,
		Decisions: []orientation.Decision{
			{ID: str("resolved"), Title: str("Answered choice"), Source: "/bundle/resolved.md", State: str("resolved"), Resolution: str("Use the local format."), CheckStatus: "pass",
				References: []orientation.Reference{{Path: "/bundle/resolved.md", ID: str("resolved"), Title: str("Answered choice"), From: project.Source, Link: "resolved.md"}}},
			{ID: str("open"), Title: str("Open choice"), Source: "/bundle/open.md", State: str("open"), CheckStatus: "fail",
				Reasons:      []orientation.Finding{{Code: "open_decision", Message: "The choice remains open."}},
				AffectedWork: []orientation.Reference{{Path: "/bundle/blocked.md", ID: str("blocked"), Title: str("Blocked task")}},
				References:   []orientation.Reference{{Path: "/bundle/open.md", ID: str("open"), Title: str("Open choice"), From: project.Source, Link: "open.md"}}},
		},
	}
	status := cli.Run(context.Background(), []string{"ctx", "orient", "--project", "."}, &stdout, &stderr, cli.Operations{Orient: func(context.Context, orientation.Request) (orientation.Result, error) {
		return result, nil
	}})
	text := stdout.String()
	start, end := strings.Index(text, "Open decisions:"), strings.Index(text, "Decision inventory:")
	if status != 0 || stderr.Len() != 0 || start < 0 || end <= start {
		t.Fatalf("decision report: status=%d stderr=%q stdout=%q", status, stderr.String(), text)
	}
	if open := text[start:end]; !strings.Contains(open, "Open choice") || strings.Contains(open, "Answered choice") {
		t.Fatalf("stale index must not override the resolved state: %s", open)
	}
	if !strings.Contains(text[end:], "state: resolved") || !strings.Contains(text[end:], "resolution: Use the local format.") ||
		!strings.Contains(text[end:], "link \"resolved.md\"") {
		t.Fatalf("resolved answer and authored index provenance must remain visible: %s", text[end:])
	}
	for _, fact := range []string{"decision check: pass", "decision check: fail", "The choice remains open.", "Affected work:", "Blocked task [blocked]", "/bundle/blocked.md"} {
		if !strings.Contains(text[end:], fact) {
			t.Errorf("missing decision effect %q in %s", fact, text[end:])
		}
	}
}

func TestOrientationExecutionFailures(t *testing.T) {
	for _, format := range []string{"text", "json"} {
		for _, failure := range []string{"operation", "writer", "cancellation"} {
			t.Run(format+"/"+failure, func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				var stdout, stderr bytes.Buffer
				var writer io.Writer = &stdout
				operation := func(ctx context.Context, _ orientation.Request) (orientation.Result, error) {
					if err := ctx.Err(); err != nil {
						return orientation.Result{}, err
					}
					if failure == "operation" {
						return orientation.Result{}, errors.New("orientation operation failed")
					}
					return orientation.Result{Complete: true}, nil
				}
				if failure == "writer" {
					writer = failingWriter{}
				}
				if failure == "cancellation" {
					cancel()
				}
				args := []string{"ctx", "orient", "--project", "."}
				if format == "json" {
					args = append(args, "--json")
				}
				status := cli.Run(ctx, args, writer, &stderr, cli.Operations{Orient: operation})
				if status != 2 || stderr.Len() == 0 || stdout.Len() != 0 {
					t.Fatalf("status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
				}
			})
		}
	}
}

func TestOrientationReceivesExplicitScopeAndLimits(t *testing.T) {
	caller := t.TempDir()
	t.Chdir(caller)
	for _, tc := range []struct {
		name  string
		flags []string
		files int
		bytes int64
	}{
		{"defaults", nil, 100, 1_048_576},
		{"overrides", []string{"--max-files", "7", "--max-bytes", "1234"}, 7, 1234},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			called := false
			args := append([]string{"ctx", "orient", "--project", "bundle", "--allow-source", "docs,extra", "--allow-source", "other"}, tc.flags...)
			status := cli.Run(context.Background(), args, &stdout, &stderr, cli.Operations{Orient: func(_ context.Context, request orientation.Request) (orientation.Result, error) {
				called = true
				if request.ProjectDir != filepath.Join(caller, "bundle") ||
					len(request.AllowedSourceDirs) != 2 ||
					request.AllowedSourceDirs[0] != filepath.Join(caller, "docs,extra") ||
					request.AllowedSourceDirs[1] != filepath.Join(caller, "other") ||
					request.MaxFiles != tc.files || request.MaxBytes != tc.bytes {
					t.Fatalf("unexpected scope or limits: %+v", request)
				}
				return orientation.Result{Complete: true}, nil
			}})
			if status != 0 || !called || stderr.Len() != 0 {
				t.Fatalf("status=%d called=%v stderr=%q", status, called, stderr.String())
			}
		})
	}
}
