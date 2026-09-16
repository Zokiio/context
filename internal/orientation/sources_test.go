package orientation_test

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Zokiio/context/internal/orientation"
)

func TestSourceScopeAliasesAndReadOnlySnapshots(t *testing.T) {
	parent := t.TempDir()
	parent, _ = filepath.EvalSymlinks(parent)
	external := filepath.Join(parent, "external.md")
	if err := os.WriteFile(external, []byte(workItem("outside", "unstarted", "")), 0600); err != nil {
		t.Fatal(err)
	}
	project := filepath.Join(parent, "bundle")
	if err := os.Mkdir(project, 0700); err != nil {
		t.Fatal(err)
	}
	manifest := "---\ntype: Project\nid: project\ntitle: Project\n---\n## Goals\n[External document](../external.md)\n## Current commitments\n[Internal alias](alias.md)\n[External record](../external.md)\n## Open decisions\nNone\n"
	files := map[string]string{"project.md": manifest, "work.md": workItem("inside", "unstarted", "## Context\n[Shared](../external.md#requirements)\n"), "notes/note.md": "No nested prose expansion. [not selected](missing.md)\n"}
	for name, content := range files {
		path := filepath.Join(project, name)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink("work.md", filepath.Join(project, "alias.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("notes", filepath.Join(project, "alias-directory")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(parent, filepath.Join(project, "outside-directory")); err != nil {
		t.Fatal(err)
	}
	got, err := orientation.Orient(context.Background(), orientation.Request{ProjectDir: project, AllowedSourceDirs: []string{parent}, MaxFiles: 4})
	if err != nil || got.Complete || !got.InventoryComplete || len(got.WorkItems) != 1 || len(got.Sources) != 4 || !hasCode(got, "source_outside_scope") || hasCode(got, "source_limit_exceeded") {
		t.Fatalf("sources=%d work=%d complete=%v inventory=%v diagnostics=%+v error=%v", len(got.Sources), len(got.WorkItems), got.Complete, got.InventoryComplete, got.Diagnostics, err)
	}
	if *got.WorkItems[0].ID != "inside" || got.WorkItems[0].Committed == nil || !*got.WorkItems[0].Committed || got.Project.CommitmentsKnown {
		t.Fatalf("membership: %+v", got.WorkItems)
	}
	for _, source := range got.Sources {
		content, err := os.ReadFile(source.Path)
		if err != nil {
			t.Fatal(err)
		}
		if source.SHA256 != fmt.Sprintf("%x", sha256.Sum256(content)) {
			t.Fatalf("digest differs from captured bytes: %+v", source)
		}
		if source.Path == filepath.Join(project, "work.md") && len(source.Reasons) != 3 {
			t.Fatalf("alias reasons: %+v", source)
		}
		if source.Path == external && len(source.Reasons) != 2 {
			t.Fatalf("external reasons: %+v", source)
		}
	}
	for name, want := range files {
		got, err := os.ReadFile(filepath.Join(project, name))
		if err != nil || string(got) != want {
			t.Fatalf("modified source %s", name)
		}
	}
	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "No nested prose expansion") {
		t.Fatal("orientation returned full source bodies")
	}
	var decoded map[string]any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	for _, source := range decoded["sources"].([]any) {
		if _, exists := source.(map[string]any)["text"]; exists {
			t.Fatal("source body included")
		}
	}
}

func TestInventoryIsOrderedByRelativePath(t *testing.T) {
	project := writeProject(t, map[string]string{"project.md": emptyManifest(), "a/z.md": "z", "a.md": "a", "b.md": "b"})
	got, err := orientation.Orient(context.Background(), orientation.Request{ProjectDir: project})
	if err != nil {
		t.Fatal(err)
	}
	names := []string{}
	for _, source := range got.Sources {
		name, _ := filepath.Rel(project, source.Path)
		names = append(names, filepath.ToSlash(name))
	}
	if !reflect.DeepEqual(names, []string{"project.md", "a.md", "a/z.md", "b.md"}) {
		t.Fatalf("order: %v", names)
	}
}

func emptyManifest() string {
	return "---\ntype: Project\nid: project\ntitle: Project\n---\n## Goals\n## Current commitments\nNone\n## Open decisions\nNone\n"
}

func TestUnknownMetadataRemainsRenderable(t *testing.T) {
	manifest := strings.Replace(emptyManifest(), "title: Project\n", "title: Project\ncustom: {limits: [.nan, .inf, -.inf], nested: {1: one, two: 2}}\n", 1)
	project := writeProject(t, map[string]string{"project.md": manifest, "work.md": workItem("work", "unstarted", "")})
	got, err := orientation.Orient(context.Background(), orientation.Request{ProjectDir: project})
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("valid unknown YAML metadata cannot be rendered: %v", err)
	}
	for _, want := range []string{"custom", "limits", "nested", "one", "two"} {
		if !strings.Contains(string(encoded), want) {
			t.Fatalf("metadata %s lost", want)
		}
	}
	if len(got.WorkItems) != 1 || got.WorkItems[0].Lifecycle == nil || *got.WorkItems[0].Lifecycle != "stable" {
		t.Fatalf("default document lifecycle: %+v", got.WorkItems)
	}
}
