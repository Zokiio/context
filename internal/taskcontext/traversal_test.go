package taskcontext_test

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/Zokiio/context/internal/taskcontext"
	"github.com/google/go-cmp/cmp"
)

func selectedNames(t *testing.T, project string, result taskcontext.Result) []string {
	t.Helper()
	names := []string{}
	for _, source := range result.Sources {
		name, err := filepath.Rel(project, source.Path)
		if err != nil {
			t.Fatal(err)
		}
		names = append(names, name)
	}
	return names
}

func TestBreadthFirstBlockersAndSharedReasons(t *testing.T) {
	project := t.TempDir()
	project, _ = filepath.EvalSymlinks(project)
	files := map[string]string{
		"root.md":         "## Context\n[c](root-context.md)\n## Blocked by\n[a](a.md)\n[a](a.md)\n### Nested\n[b][second]\n## Spec\n[s](root-spec.md)\n## Blocked by\n[alias](alias.md#part)\n\n[second]: b.md\n",
		"a.md":            "## Context\n[c](a-context.md)\n## Spec\n[s](a-spec.md)\n## Blocked by\n[shared](shared.md)\n[deep](deep.md)\n",
		"b.md":            "## Blocked by\n[shared](shared.md#section)\n## Spec\n[s](b-spec.md)\n",
		"shared.md":       "## Context\n[c](shared-context.md)\n",
		"deep.md":         "## Blocked by\n[leaf](leaf.md)\n",
		"leaf.md":         "leaf",
		"root-context.md": "root context", "root-spec.md": "root spec",
		"a-context.md": "a context", "a-spec.md": "a spec", "b-spec.md": "b spec", "shared-context.md": "shared context",
	}
	writeSources(t, project, files)
	if err := os.Symlink("a.md", filepath.Join(project, "alias.md")); err != nil {
		t.Fatal(err)
	}
	result := readContext(t, project, "root.md")
	if !result.Complete || !result.TraversalComplete || len(result.Diagnostics) != 0 {
		t.Fatalf("%+v", result)
	}
	want := []string{"root.md", "root-spec.md", "root-context.md", "a.md", "a-spec.md", "a-context.md", "b.md", "b-spec.md", "shared.md", "shared-context.md", "deep.md", "leaf.md"}
	if diff := cmp.Diff(want, selectedNames(t, project, result)); diff != "" {
		t.Fatal(diff)
	}
	for _, source := range result.Sources {
		name, _ := filepath.Rel(project, source.Path)
		if source.Text != files[name] || source.SHA256 != fmt.Sprintf("%x", sha256.Sum256([]byte(files[name]))) {
			t.Fatalf("source changed: %+v", source)
		}
		unchanged, err := os.ReadFile(source.Path)
		if err != nil || string(unchanged) != files[name] {
			t.Fatalf("source modified: %s", name)
		}
	}
	if diff := cmp.Diff([]taskcontext.Reason{
		{Kind: "blocked_by", From: filepath.Join(project, "root.md"), Link: "a.md"},
		{Kind: "blocked_by", From: filepath.Join(project, "root.md"), Link: "alias.md#part"},
	}, result.Sources[3].Reasons); diff != "" {
		t.Fatal(diff)
	}
	if diff := cmp.Diff([]taskcontext.Reason{
		{Kind: "blocked_by", From: filepath.Join(project, "a.md"), Link: "shared.md"},
		{Kind: "blocked_by", From: filepath.Join(project, "b.md"), Link: "shared.md#section"},
	}, result.Sources[8].Reasons); diff != "" {
		t.Fatal(diff)
	}
}

func TestDependencyCycles(t *testing.T) {
	for _, tc := range []struct {
		name         string
		files        map[string]string
		from, target string
		count        int
	}{
		{"self", map[string]string{"root.md": "## Blocked by\n[self](root.md)\n[self](root.md#again)"}, "root.md", "root.md", 1},
		{"chain", map[string]string{"root.md": "## Blocked by\n[a](a.md)", "a.md": "## Blocked by\n[b](b.md)", "b.md": "## Blocked by\n[root](root.md)"}, "b.md", "root.md", 3},
		{"cross branch", map[string]string{"root.md": "## Blocked by\n[a](a.md)\n[b](b.md)", "a.md": "## Blocked by\n[b](b.md)", "b.md": "## Blocked by\n[a](a.md)"}, "b.md", "a.md", 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			project := t.TempDir()
			project, _ = filepath.EvalSymlinks(project)
			writeSources(t, project, tc.files)
			result := readContext(t, project, "root.md")
			if !result.Complete || !result.TraversalComplete || len(result.Sources) != tc.count || len(result.Diagnostics) != 1 {
				t.Fatalf("%+v", result)
			}
			d := result.Diagnostics[0]
			if d.Code != "dependency_cycle" || d.Severity != "warning" || d.From != filepath.Join(project, tc.from) || d.Path != filepath.Join(project, tc.target) || d.Link == "" {
				t.Fatalf("%+v", d)
			}
		})
	}
}

func TestUnavailableBlockersContinueOtherBranches(t *testing.T) {
	project := t.TempDir()
	project, _ = filepath.EvalSymlinks(project)
	files := map[string]string{
		"root.md": "## Blocked by\n[missing](missing.md)\n[unreadable](directory.md)\n[a](a.md)\n[good](good.md)",
		"a.md":    "## Blocked by\n[deep](deep-missing.md)\n## Spec\n[s](missing-spec.md)",
		"good.md": "## Blocked by\n[leaf](leaf.md)", "leaf.md": "leaf",
	}
	writeSources(t, project, files)
	if err := os.Mkdir(filepath.Join(project, "directory.md"), 0700); err != nil {
		t.Fatal(err)
	}
	result := readContext(t, project, "root.md")
	if result.Complete || result.TraversalComplete {
		t.Fatalf("%+v", result)
	}
	if diff := cmp.Diff([]string{"root.md", "a.md", "good.md", "leaf.md"}, selectedNames(t, project, result)); diff != "" {
		t.Fatal(diff)
	}
	codes := []string{}
	for _, d := range result.Diagnostics {
		codes = append(codes, d.Code)
		if d.From == "" || d.Link == "" {
			t.Fatalf("missing provenance: %+v", d)
		}
	}
	if diff := cmp.Diff([]string{"source_missing", "source_unreadable", "source_missing", "source_missing"}, codes); diff != "" {
		t.Fatal(diff)
	}
	for name, want := range files {
		content, err := os.ReadFile(filepath.Join(project, name))
		if err != nil || string(content) != want {
			t.Fatalf("source modified: %s", name)
		}
	}
}

func TestMalformedBlockerRetainsSourceAndSkipsRelationships(t *testing.T) {
	for _, metadata := range []string{"---\ntitle: [broken\n---\n", "---\ntype: WorkItem\n", "---\n- sequence\n---\n"} {
		project := t.TempDir()
		project, _ = filepath.EvalSymlinks(project)
		broken := metadata + "## Blocked by\n[not discovered](hidden.md)\n"
		writeSources(t, project, map[string]string{"root.md": "## Blocked by\n[bad](bad.md)\n[good](good.md)", "bad.md": broken, "good.md": "good", "hidden.md": "hidden"})
		result := readContext(t, project, "root.md")
		if result.Complete || result.TraversalComplete || len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "invalid_frontmatter" {
			t.Fatalf("%+v", result)
		}
		if diff := cmp.Diff([]string{"root.md", "bad.md", "good.md"}, selectedNames(t, project, result)); diff != "" {
			t.Fatal(diff)
		}
		if result.Sources[1].Text != broken || result.Sources[1].SHA256 != fmt.Sprintf("%x", sha256.Sum256([]byte(broken))) {
			t.Fatal("malformed source was not preserved")
		}
	}
}

func TestMissingBlockerDocumentDoesNotPreventTraversal(t *testing.T) {
	project := t.TempDir()
	project, _ = filepath.EvalSymlinks(project)
	writeSources(t, project, map[string]string{"root.md": "## Blocked by\n[a](a.md)", "a.md": "## Spec\n[s](missing.md)\n## Context\n[c](absent.md)\n## Blocked by\n[b](b.md)", "b.md": "b"})
	result := readContext(t, project, "root.md")
	if result.Complete || !result.TraversalComplete || len(result.Diagnostics) != 2 {
		t.Fatalf("%+v", result)
	}
	if diff := cmp.Diff([]string{"root.md", "a.md", "b.md"}, selectedNames(t, project, result)); diff != "" {
		t.Fatal(diff)
	}
}

func TestDocumentLaterTraversedAsTicket(t *testing.T) {
	project := t.TempDir()
	project, _ = filepath.EvalSymlinks(project)
	writeSources(t, project, map[string]string{
		"root.md": "## Context\n[future blocker](b.md)\n## Blocked by\n[a](a.md)",
		"a.md":    "## Blocked by\n[b](b.md)",
		"b.md":    "## Spec\n[s](b-spec.md)\n## Blocked by\n[leaf](leaf.md)", "b-spec.md": "spec", "leaf.md": "leaf",
	})
	result := readContext(t, project, "root.md")
	if !result.Complete || !result.TraversalComplete {
		t.Fatalf("%+v", result)
	}
	if diff := cmp.Diff([]string{"root.md", "b.md", "a.md", "b-spec.md", "leaf.md"}, selectedNames(t, project, result)); diff != "" {
		t.Fatal(diff)
	}
	if diff := cmp.Diff([]taskcontext.Reason{
		{Kind: "context", From: filepath.Join(project, "root.md"), Link: "b.md"},
		{Kind: "blocked_by", From: filepath.Join(project, "a.md"), Link: "b.md"},
	}, result.Sources[1].Reasons); diff != "" {
		t.Fatal(diff)
	}
}

func TestBlockerScopeCheckedForPreviouslyIncludedDocument(t *testing.T) {
	parent := t.TempDir()
	parent, _ = filepath.EvalSymlinks(parent)
	project := filepath.Join(parent, "bundle")
	writeSources(t, project, map[string]string{"root.md": "## Context\n[doc](../external.md)\n## Blocked by\n[outside](../external.md)\n[alias](escape.md)\n[remote](https://example.invalid/ticket)\n[good](good.md)", "good.md": "good"})
	writeSources(t, parent, map[string]string{"external.md": "## Blocked by\n[hidden](hidden.md)", "hidden.md": "hidden"})
	if err := os.Symlink(filepath.Join(parent, "external.md"), filepath.Join(project, "escape.md")); err != nil {
		t.Fatal(err)
	}
	result, err := taskcontext.Assemble(context.Background(), taskcontext.Request{ProjectDir: project, TicketPath: "root.md", AllowedSourceDirs: []string{parent}})
	if err != nil {
		t.Fatal(err)
	}
	if result.Complete || result.TraversalComplete {
		t.Fatalf("%+v", result)
	}
	if diff := cmp.Diff([]string{"root.md", "../external.md", "good.md"}, selectedNames(t, project, result)); diff != "" {
		t.Fatal(diff)
	}
	codes := []string{}
	for _, d := range result.Diagnostics {
		codes = append(codes, d.Code)
	}
	if diff := cmp.Diff([]string{"source_outside_scope", "source_outside_scope", "unsupported_source"}, codes); diff != "" {
		t.Fatal(diff)
	}
	if len(result.Sources[1].Reasons) != 1 || result.Sources[1].Reasons[0].Kind != "context" {
		t.Fatalf("unauthorized ticket reason: %+v", result.Sources[1])
	}
}
