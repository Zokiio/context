package orientation_test

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Zokiio/context/internal/orientation"
	"github.com/Zokiio/context/internal/taskcontext"
)

func TestDecisionStateControlsOnlyExplicitBlockers(t *testing.T) {
	manifest := strings.Replace(committedManifest("work.md", "unrelated.md"), "## Open decisions\nNone", "## Open decisions\n[Blocking choice](decision.md)\n[Project choice](project-choice.md)", 1)
	project := writeProject(t, map[string]string{
		"project.md":        manifest,
		"work.md":           strings.Replace(workItem("work", "unstarted", ""), "## Blocked by decisions\nNone", "## Blocked by decisions\n[Choice](decision.md)", 1),
		"unrelated.md":      workItem("unrelated", "unstarted", ""),
		"decision.md":       decisionRecord("choice", "open", ""),
		"project-choice.md": decisionRecord("project-choice", "open", ""),
	})
	for _, state := range []string{"open", "resolved"} {
		t.Run(state, func(t *testing.T) {
			answer := ""
			if state == "resolved" {
				answer = "## Resolution\nUse the authored answer.\n"
			}
			if err := os.WriteFile(filepath.Join(project, "decision.md"), []byte(decisionRecord("choice", state, answer)), 0600); err != nil {
				t.Fatal(err)
			}
			got, err := orientation.Orient(context.Background(), orientation.Request{ProjectDir: project})
			if err != nil || !got.Complete {
				t.Fatalf("complete=%v diagnostics=%+v error=%v", got.Complete, got.Diagnostics, err)
			}
			work := findWork(t, got, "work.md")
			unrelated := findWork(t, got, "unrelated.md")
			wantStatus, wantReadiness := "fail", "blocked"
			if state == "resolved" {
				wantStatus, wantReadiness = "pass", "ready"
			}
			if checkStatus(work, "blocking_decisions") != wantStatus || work.Readiness != wantReadiness || work.Eligible != (state == "resolved") {
				t.Fatalf("blocked work: %+v", work)
			}
			if unrelated.Readiness != "ready" || !unrelated.Eligible {
				t.Fatalf("unrelated work blocked: %+v", unrelated)
			}
			choice := findDecision(t, got, "choice")
			if choice.State == nil || *choice.State != state || choice.CheckStatus != wantStatus || !reflect.DeepEqual(referenceIDs(choice.AffectedWork), []string{"work"}) || len(choice.References) != 2 {
				t.Fatalf("decision: %+v", choice)
			}
			if state == "resolved" && (choice.Resolution == nil || *choice.Resolution != "Use the authored answer.") {
				t.Fatalf("answer: %+v", choice)
			}
		})
	}
}

func decisionRecord(id, state, body string) string {
	return "---\ntype: Decision\nid: " + id + "\ntitle: Decision " + id + "\ndecisionState: " + state + "\n---\n" + body
}

func findDecision(t *testing.T, result orientation.Result, id string) orientation.Decision {
	t.Helper()
	for _, decision := range result.Decisions {
		if decision.ID != nil && *decision.ID == id {
			return decision
		}
	}
	t.Fatalf("decision %s absent", id)
	return orientation.Decision{}
}

func TestUnavailableDecisionEffectsStayUnknown(t *testing.T) {
	for _, tc := range []struct{ name, body, code string }{
		{"missing state", strings.Replace(decisionRecord("choice", "open", ""), "decisionState: open\n", "", 1), "invalid_profile"},
		{"invalid state", decisionRecord("choice", "done", ""), "invalid_profile"},
		{"missing identity", strings.Replace(decisionRecord("choice", "open", ""), "id: choice\n", "", 1), "invalid_profile"},
		{"missing title", strings.Replace(decisionRecord("choice", "open", ""), "title: Decision choice\n", "", 1), "invalid_profile"},
		{"missing answer", decisionRecord("choice", "resolved", ""), "missing_resolution"},
		{"empty answer", decisionRecord("choice", "resolved", "## Resolution\n"), "missing_resolution"},
		{"comment-only answer", decisionRecord("choice", "resolved", "## Resolution\n<!-- not answered yet -->\n"), "missing_resolution"},
		{"fenced answer heading", decisionRecord("choice", "resolved", "```md\n## Resolution\nExample answer\n```\n"), "missing_resolution"},
		{"wrong type", workItem("choice", "completed", ""), "invalid_profile"},
		{"missing source", "", "source_missing"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			files := map[string]string{"project.md": committedManifest("work.md"), "work.md": strings.Replace(workItem("work", "unstarted", ""), "## Blocked by decisions\nNone", "## Blocked by decisions\n[Choice](decision.md)", 1)}
			if tc.body != "" {
				files["decision.md"] = tc.body
			}
			got, err := orientation.Orient(context.Background(), orientation.Request{ProjectDir: writeProject(t, files)})
			if err != nil {
				t.Fatal(err)
			}
			work := findWork(t, got, "work.md")
			if got.Complete || work.Readiness != "unknown" || checkStatus(work, "blocking_decisions") != "unknown" || work.Eligible || !hasCode(got, tc.code) {
				t.Fatalf("work=%+v complete=%v diagnostics=%+v", work, got.Complete, got.Diagnostics)
			}
		})
	}
}

func TestDecisionChecksRetainOpenAndUnknownReasons(t *testing.T) {
	work := strings.Replace(workItem("work", "unstarted", ""), "## Blocked by decisions\nNone", "## Blocked by decisions\n[Open](open.md)\n[Repeated](open.md)\n[Fragment](open.md#question)\n[Unknown](missing.md)\n[Ambiguous](duplicate-a.md)\n", 1)
	project := writeProject(t, map[string]string{
		"project.md": committedManifest("work.md", "safe.md"), "work.md": work, "safe.md": workItem("safe", "unstarted", ""),
		"open.md":        decisionRecord("open-choice", "open", ""),
		"duplicate-a.md": decisionRecord("duplicate", "resolved", "## Resolution\nAn answer.\n"),
		"duplicate-b.md": decisionRecord("duplicate", "open", ""),
	})
	got, err := orientation.Orient(context.Background(), orientation.Request{ProjectDir: project})
	if err != nil {
		t.Fatal(err)
	}
	blocked := findWork(t, got, "work.md")
	if got.Complete || !got.InventoryComplete || blocked.Readiness != "blocked" || checkStatus(blocked, "blocking_decisions") != "fail" {
		t.Fatalf("work=%+v complete=%v inventory=%v diagnostics=%+v", blocked, got.Complete, got.InventoryComplete, got.Diagnostics)
	}
	codes := map[string]bool{}
	for _, check := range blocked.Checks {
		if check.Name == "blocking_decisions" {
			for _, reason := range check.Reasons {
				codes[reason.Code] = true
			}
		}
	}
	for _, code := range []string{"open_decision", "decision_unavailable", "duplicate_identity"} {
		if !codes[code] {
			t.Fatalf("reason %s lost: %+v", code, blocked.Checks)
		}
	}
	if safe := findWork(t, got, "safe.md"); !safe.Eligible || safe.Readiness != "ready" {
		t.Fatalf("unrelated work affected: %+v", safe)
	}
	open := findDecision(t, got, "open-choice")
	if len(open.References) != 2 || !reflect.DeepEqual(referenceIDs(open.AffectedWork), []string{"work"}) {
		t.Fatalf("repeated references: %+v", open)
	}
	for _, source := range got.Sources {
		if filepath.Base(source.Path) == "open.md" && len(source.Reasons) != 3 {
			t.Fatalf("source provenance: %+v", source)
		}
	}
}

func TestMalformedDecisionDeclarationPreservesKnownOpenBlocker(t *testing.T) {
	work := strings.Replace(workItem("work", "unstarted", ""), "## Blocked by decisions\nNone", "## Blocked by decisions\n[Open](open.md)\n## Blocked by decisions\nNeed another answer later.\n", 1)
	project := writeProject(t, map[string]string{"project.md": committedManifest("work.md"), "work.md": work, "open.md": decisionRecord("choice", "open", "")})
	got, err := orientation.Orient(context.Background(), orientation.Request{ProjectDir: project})
	if err != nil {
		t.Fatal(err)
	}
	blocked := findWork(t, got, "work.md")
	if got.Complete || blocked.Readiness != "blocked" || checkStatus(blocked, "blocking_decisions") != "fail" || !hasCode(got, "invalid_relationship_section") {
		t.Fatalf("work=%+v diagnostics=%+v", blocked, got.Diagnostics)
	}
}

func TestExternalDecisionCannotBeAuthorizedAsARecord(t *testing.T) {
	parent := t.TempDir()
	project := filepath.Join(parent, "bundle")
	if err := os.Mkdir(project, 0700); err != nil {
		t.Fatal(err)
	}
	manifest := strings.Replace(committedManifest("work.md"), "## Goals\n", "## Goals\n[Document view](../decision.md)\n", 1)
	work := strings.Replace(workItem("work", "unstarted", ""), "## Blocked by decisions\nNone", "## Blocked by decisions\n[External answer](../decision.md)", 1)
	for path, content := range map[string]string{"bundle/project.md": manifest, "bundle/work.md": work, "decision.md": decisionRecord("outside", "resolved", "## Resolution\nUse this answer.\n")} {
		if err := os.WriteFile(filepath.Join(parent, path), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	got, err := orientation.Orient(context.Background(), orientation.Request{ProjectDir: project, AllowedSourceDirs: []string{parent}})
	if err != nil {
		t.Fatal(err)
	}
	blocked := findWork(t, got, "work.md")
	if got.Complete || !got.InventoryComplete || blocked.Readiness != "unknown" || checkStatus(blocked, "blocking_decisions") != "unknown" || len(got.Decisions) != 0 || len(got.Sources) != 3 || !hasCode(got, "source_outside_scope") {
		t.Fatalf("external decision: work=%+v sources=%+v diagnostics=%+v", blocked, got.Sources, got.Diagnostics)
	}
}

func TestUnreadableDecisionTargetIsUnknown(t *testing.T) {
	work := strings.Replace(workItem("work", "unstarted", ""), "## Blocked by decisions\nNone", "## Blocked by decisions\n[Choice](decision.md)", 1)
	project := writeProject(t, map[string]string{"project.md": committedManifest("work.md"), "work.md": work})
	if err := os.Mkdir(filepath.Join(project, "decision.md"), 0700); err != nil {
		t.Fatal(err)
	}
	got, err := orientation.Orient(context.Background(), orientation.Request{ProjectDir: project})
	if err != nil {
		t.Fatal(err)
	}
	if got.Complete || findWork(t, got, "work.md").Readiness != "unknown" || !hasCode(got, "source_unreadable") {
		t.Fatalf("diagnostics=%+v", got.Diagnostics)
	}
}

func TestDecisionAnswersRequireExplicitTaskContextLinks(t *testing.T) {
	work := strings.Replace(workItem("work", "unstarted", ""), "## Blocked by decisions\nNone", "## Blocked by decisions\n[Choice](decision.md)", 1)
	project := writeProject(t, map[string]string{"project.md": committedManifest("work.md"), "work.md": work, "decision.md": decisionRecord("choice", "resolved", "## Resolution\nUse the local format.\n")})
	request := taskcontext.Request{ProjectDir: project, TicketPath: "work.md"}
	got, err := taskcontext.Assemble(context.Background(), request)
	if err != nil || !got.Complete || len(got.Sources) != 1 {
		t.Fatalf("decision blocker implicitly selected: %+v %v", got, err)
	}
	if err := os.WriteFile(filepath.Join(project, "work.md"), []byte(work+"## Context\n[Implementation decision](decision.md)\n"), 0600); err != nil {
		t.Fatal(err)
	}
	got, err = taskcontext.Assemble(context.Background(), request)
	if err != nil || !got.Complete || len(got.Sources) != 2 || filepath.Base(got.Sources[1].Path) != "decision.md" {
		t.Fatalf("explicit decision context missing: %+v %v", got, err)
	}
}
