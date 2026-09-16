package taskcontext_test

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Zokiio/context/internal/taskcontext"
	"github.com/google/go-cmp/cmp"
)

func limitedContext(t *testing.T, project string, maxFiles int, maxBytes int64) taskcontext.Result {
	t.Helper()
	result, err := taskcontext.Assemble(context.Background(), taskcontext.Request{ProjectDir: project, TicketPath: "root.md", MaxFiles: maxFiles, MaxBytes: maxBytes})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestLimitsExactFitAndDeduplication(t *testing.T) {
	project := t.TempDir()
	project, _ = filepath.EvalSymlinks(project)
	files := map[string]string{
		"root.md":    "## Spec\n[doc](doc.md)\n[again](doc.md#part)\n## Blocked by\n[blocker](blocker.md)\n",
		"doc.md":     "Café\n\t\"quotes\" <>&\\",
		"blocker.md": "## Context\n[shared](doc.md)\n## Blocked by\n[root](root.md)",
	}
	writeSources(t, project, files)
	var bytes int64
	for _, value := range files {
		bytes += int64(len(value))
	}
	result := limitedContext(t, project, 3, bytes)
	if !result.Complete || !result.TraversalComplete || len(result.Sources) != 3 || len(result.Sources[1].Reasons) != 3 || len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "dependency_cycle" {
		t.Fatalf("%+v", result)
	}
	for _, source := range result.Sources {
		name, _ := filepath.Rel(project, source.Path)
		if source.Text != files[name] {
			t.Fatalf("truncated %s", name)
		}
	}
	for _, limits := range []struct {
		files int
		bytes int64
	}{{2, bytes}, {3, bytes - 1}} {
		result = limitedContext(t, project, limits.files, limits.bytes)
		if result.Complete || result.TraversalComplete || len(result.Sources) != 2 || result.Diagnostics[0].Code != "source_limit_exceeded" || filepath.Base(result.Diagnostics[0].Path) != "blocker.md" {
			t.Fatalf("%+v", result)
		}
	}
}

func TestFirstBreachReportsKnownDocumentsAndBlockers(t *testing.T) {
	project := t.TempDir()
	project, _ = filepath.EvalSymlinks(project)
	files := map[string]string{
		"root.md":  "## Spec\n[large](large.md)\n[small](small.md)\n## Context\n[context](context.md)\n## Blocked by\n[a](a.md)\n[b](b.md)\n",
		"large.md": strings.Repeat("x", 200), "small.md": "s", "context.md": "c",
		"a.md": "## Blocked by\n[undiscovered](hidden.md)", "b.md": "b", "hidden.md": "hidden",
	}
	writeSources(t, project, files)
	for _, limits := range []struct {
		files int
		bytes int64
	}{{1, 0}, {10, int64(len(files["root.md"]) + 1)}} {
		result := limitedContext(t, project, limits.files, limits.bytes)
		if result.Complete || result.TraversalComplete || len(result.Sources) != 1 {
			t.Fatalf("%+v", result)
		}
		paths := []string{}
		for i, d := range result.Diagnostics {
			paths = append(paths, filepath.Base(d.Path))
			code := "source_omitted"
			if i == 0 {
				code = "source_limit_exceeded"
			}
			if d.Code != code || d.From == "" || d.Link == "" {
				t.Fatalf("%+v", d)
			}
		}
		if diff := cmp.Diff([]string{"large.md", "small.md", "context.md", "a.md", "b.md"}, paths); diff != "" {
			t.Fatal(diff)
		}
	}
}

func TestLimitInsideBlockerReportsBreadthFirstQueue(t *testing.T) {
	project := t.TempDir()
	project, _ = filepath.EvalSymlinks(project)
	writeSources(t, project, map[string]string{
		"root.md": "## Blocked by\n[a](a.md)\n[b](b.md)",
		"a.md":    "## Spec\n[spec](spec.md)\n## Blocked by\n[child](child.md)",
		"b.md":    "## Blocked by\n[unknown](unknown.md)", "spec.md": "spec", "child.md": "child", "unknown.md": "unknown",
	})
	result := limitedContext(t, project, 2, 0)
	if diff := cmp.Diff([]string{"root.md", "a.md"}, selectedNames(t, project, result)); diff != "" {
		t.Fatal(diff)
	}
	paths := []string{}
	for _, d := range result.Diagnostics {
		paths = append(paths, filepath.Base(d.Path))
	}
	if diff := cmp.Diff([]string{"spec.md", "b.md", "child.md"}, paths); diff != "" {
		t.Fatal(diff)
	}
	if result.Complete || result.TraversalComplete {
		t.Fatalf("%+v", result)
	}
}

func TestOversizedStartingTicketDoesNotDiscoverRelationships(t *testing.T) {
	project := t.TempDir()
	writeSources(t, project, map[string]string{"root.md": "## Spec\n[unknown](unknown.md)"})
	result := limitedContext(t, project, 0, 1)
	if result.Complete || result.TraversalComplete || len(result.Sources) != 0 || len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "source_limit_exceeded" || filepath.Base(result.Diagnostics[0].Path) != "root.md" {
		t.Fatalf("%+v", result)
	}
}

func TestLimitsPreserveEarlierErrorsAndSkipDuplicateOmissions(t *testing.T) {
	project := t.TempDir()
	writeSources(t, project, map[string]string{"root.md": "## Spec\n[missing](missing.md)\n[limit](large.md)\n[again](large.md)\n[self](root.md)\n[last](last.md)\n[again](last.md)", "large.md": "large", "last.md": "last"})
	result := limitedContext(t, project, 1, 0)
	codes := []string{}
	for _, d := range result.Diagnostics {
		codes = append(codes, d.Code)
	}
	if diff := cmp.Diff([]string{"source_missing", "source_limit_exceeded", "source_omitted"}, codes); diff != "" {
		t.Fatal(diff)
	}
	if filepath.Base(result.Diagnostics[2].Path) != "last.md" {
		t.Fatalf("%+v", result)
	}
}

func TestDefaultLimits(t *testing.T) {
	t.Run("files", func(t *testing.T) {
		project := t.TempDir()
		files := map[string]string{}
		var root strings.Builder
		root.WriteString("## Spec\n")
		for i := 0; i < 100; i++ {
			name := fmt.Sprintf("doc-%03d.md", i)
			fmt.Fprintf(&root, "[doc](%s)\n", name)
			files[name] = "x"
		}
		files["root.md"] = root.String()
		writeSources(t, project, files)
		result := limitedContext(t, project, 0, 0)
		if result.Complete || len(result.Sources) != 100 || filepath.Base(result.Diagnostics[0].Path) != "doc-099.md" {
			t.Fatalf("unexpected default file limit: %+v", result.Diagnostics)
		}
	})
	t.Run("bytes", func(t *testing.T) {
		project := t.TempDir()
		writeSources(t, project, map[string]string{"root.md": strings.Repeat("x", int(taskcontext.DefaultMaxBytes))})
		if result := limitedContext(t, project, 0, 0); !result.Complete {
			t.Fatalf("exact default failed: %+v", result.Diagnostics)
		}
		writeSources(t, project, map[string]string{"root.md": strings.Repeat("x", int(taskcontext.DefaultMaxBytes)+1)})
		if result := limitedContext(t, project, 0, 0); result.Complete || len(result.Sources) != 0 {
			t.Fatalf("default overflow: %+v", result.Diagnostics)
		}
	})
}

func TestNegativeApplicationLimits(t *testing.T) {
	for _, request := range []taskcontext.Request{{ProjectDir: t.TempDir(), TicketPath: "root.md", MaxFiles: -1}, {ProjectDir: t.TempDir(), TicketPath: "root.md", MaxBytes: -1}} {
		if _, err := taskcontext.Assemble(context.Background(), request); err == nil {
			t.Fatal("negative limit accepted")
		}
	}
}
