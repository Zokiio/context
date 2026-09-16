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

func TestAuthorizedExternalSources(t *testing.T) {
	parent := t.TempDir()
	parent, _ = filepath.EvalSymlinks(parent)
	project := filepath.Join(parent, "bundle")
	docs := filepath.Join(parent, "docs")
	other := filepath.Join(parent, "other")
	text := "# External café\r\n[ordinary](https://example.invalid)\r\n## Spec\r\n[not followed](absent.md)\r\n"
	files := map[string]string{
		"bundle/ticket.md": "## Spec\n[external](../docs/guide.md#one)\n[alias](alias.md#two)\n## Context\n[second](../other/design.md)\n[bundle root](/local.md)\n",
		"bundle/local.md":  "bundle scoped", "docs/guide.md": text, "other/design.md": "# Plain Markdown\n",
	}
	writeSources(t, parent, files)
	if err := os.Symlink(filepath.Join(docs, "guide.md"), filepath.Join(project, "alias.md")); err != nil {
		t.Fatal(err)
	}
	result, err := taskcontext.Assemble(context.Background(), taskcontext.Request{ProjectDir: project, TicketPath: "ticket.md", AllowedSourceDirs: []string{docs, parent, other, docs}})
	if err != nil || !result.Complete || !result.TraversalComplete || len(result.Diagnostics) != 0 || len(result.Sources) != 4 {
		t.Fatalf("%+v %v", result, err)
	}
	external := result.Sources[1]
	if external.Path != filepath.Join(docs, "guide.md") || external.Text != text || external.SHA256 != fmt.Sprintf("%x", sha256.Sum256([]byte(text))) {
		t.Fatalf("%+v", external)
	}
	want := []taskcontext.Reason{{Kind: "spec", From: filepath.Join(project, "ticket.md"), Link: "../docs/guide.md#one"}, {Kind: "spec", From: filepath.Join(project, "ticket.md"), Link: "alias.md#two"}}
	if diff := cmp.Diff(want, external.Reasons); diff != "" {
		t.Fatal(diff)
	}
	for name, expected := range files {
		got, err := os.ReadFile(filepath.Join(parent, name))
		if err != nil || string(got) != expected {
			t.Fatalf("changed source %s: %v", name, err)
		}
	}
}

func TestExternalSourceScopeAndFailures(t *testing.T) {
	parent := t.TempDir()
	project := filepath.Join(parent, "bundle")
	docs := filepath.Join(parent, "docs")
	writeSources(t, parent, map[string]string{
		"bundle/ticket.md": "## Context\n[good](../docs/good.md)\n[missing](../docs/missing.md)\n[unreadable](../docs/directory.md)\n[prefix escape](../docs-neighbor/secret.md)\n[symlink escape](../docs/escape.md)\n",
		"docs/good.md":     "good", "docs-neighbor/secret.md": "secret",
	})
	if err := os.Mkdir(filepath.Join(docs, "directory.md"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("../docs-neighbor/secret.md", filepath.Join(docs, "escape.md")); err != nil {
		t.Fatal(err)
	}
	result, err := taskcontext.Assemble(context.Background(), taskcontext.Request{ProjectDir: project, TicketPath: "ticket.md", AllowedSourceDirs: []string{docs}})
	if err != nil || result.Complete || !result.TraversalComplete || len(result.Sources) != 2 {
		t.Fatalf("%+v %v", result, err)
	}
	codes := []string{}
	for _, diagnostic := range result.Diagnostics {
		codes = append(codes, diagnostic.Code)
		if diagnostic.Path == "" || diagnostic.From == "" || diagnostic.Link == "" {
			t.Fatalf("missing provenance: %+v", diagnostic)
		}
	}
	if diff := cmp.Diff([]string{"source_missing", "source_unreadable", "source_outside_scope", "source_outside_scope"}, codes); diff != "" {
		t.Fatal(diff)
	}
}

func TestAllowedDirectoryDoesNotAuthorizeTickets(t *testing.T) {
	parent := t.TempDir()
	project := filepath.Join(parent, "bundle")
	docs := filepath.Join(parent, "docs")
	writeSources(t, parent, map[string]string{"bundle/ticket.md": "## Blocked by\n[external](../docs/blocker.md)\n## Context\n[good](../docs/context.md)", "docs/blocker.md": "# Blocker", "docs/context.md": "# Context"})
	for _, ticket := range []string{"../docs/blocker.md", "ticket.md"} {
		t.Run(ticket, func(t *testing.T) {
			result, err := taskcontext.Assemble(context.Background(), taskcontext.Request{ProjectDir: project, TicketPath: ticket, AllowedSourceDirs: []string{docs}})
			if err != nil || result.Complete || result.TraversalComplete || len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "source_outside_scope" {
				t.Fatalf("%+v %v", result, err)
			}
			for _, source := range result.Sources {
				if filepath.Base(source.Path) == "blocker.md" {
					t.Fatal("external ticket included")
				}
			}
		})
	}
}

func TestInvalidAllowedDirectories(t *testing.T) {
	project := t.TempDir()
	writeSources(t, project, map[string]string{"ticket.md": "# Ticket"})
	for _, directory := range []string{"", filepath.Join(project, "missing"), filepath.Join(project, "ticket.md")} {
		if _, err := taskcontext.Assemble(context.Background(), taskcontext.Request{ProjectDir: project, TicketPath: "ticket.md", AllowedSourceDirs: []string{directory}}); err == nil {
			t.Fatalf("accepted invalid root %q", directory)
		}
	}
}

func TestAllowedRootAliasAndMissingChild(t *testing.T) {
	parent := t.TempDir()
	project := filepath.Join(parent, "bundle")
	writeSources(t, parent, map[string]string{"bundle/ticket.md": "## Context\n[present](../alias/present.md)\n[missing](../alias/missing.md)", "docs/present.md": "present"})
	if err := os.Symlink("docs", filepath.Join(parent, "alias")); err != nil {
		t.Fatal(err)
	}
	result, err := taskcontext.Assemble(context.Background(), taskcontext.Request{ProjectDir: project, TicketPath: "ticket.md", AllowedSourceDirs: []string{filepath.Join(parent, "alias")}})
	if err != nil || result.Complete || !result.TraversalComplete || len(result.Sources) != 2 || len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "source_missing" {
		t.Fatalf("%+v %v", result, err)
	}
}
