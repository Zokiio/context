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

func TestCurrentSources(t *testing.T) {
	for _, projectName := range []string{"first", "independent"} {
		t.Run(projectName, func(t *testing.T) {
			project := t.TempDir()
			project, _ = filepath.EvalSymlinks(project)
			path := filepath.Join(project, "ticket.md")
			for _, source := range []string{"---\r\ntype: Unknown\r\ncustom: {nested: [one, 2]}\r\n---\r\n# Café\r\n<!-- comment -->\r\n", "# Current uncommitted text\n\n"} {
				if err := os.WriteFile(path, []byte(source), 0600); err != nil {
					t.Fatal(err)
				}
				got, err := taskcontext.Assemble(context.Background(), taskcontext.Request{ProjectDir: project, TicketPath: "ticket.md"})
				if err != nil {
					t.Fatal(err)
				}
				want := taskcontext.Result{SchemaVersion: 1, Complete: true, TraversalComplete: true, Sources: []taskcontext.Source{{Path: path, Text: source, SHA256: fmt.Sprintf("%x", sha256.Sum256([]byte(source))), Reasons: []taskcontext.Reason{{Kind: "root"}}}}, Diagnostics: []taskcontext.Diagnostic{}}
				if diff := cmp.Diff(want, got); diff != "" {
					t.Fatal(diff)
				}
				unchanged, err := os.ReadFile(path)
				if err != nil || string(unchanged) != source {
					t.Fatalf("source changed: %q, %v", unchanged, err)
				}
				entries, _ := os.ReadDir(project)
				if len(entries) != 1 {
					t.Fatalf("reader created files: %v", entries)
				}
			}
		})
	}
}

func TestInvalidFrontmatter(t *testing.T) {
	for name, source := range map[string]string{
		"malformed":    "---\ntitle: [broken\n---\n# Body\n",
		"unterminated": "---\ntitle: x\n# Body\n",
		"scalar":       "---\nhello\n---\n# Body\n",
		"sequence":     "---\n- hello\n---\n# Body\n",
		"null":         "---\nnull\n---\n# Body\n",
		"empty":        "---\n---\n# Body\n",
	} {
		t.Run(name, func(t *testing.T) {
			project := t.TempDir()
			path := filepath.Join(project, "ticket.md")
			if err := os.WriteFile(path, []byte(source), 0600); err != nil {
				t.Fatal(err)
			}
			got, err := taskcontext.Assemble(context.Background(), taskcontext.Request{ProjectDir: project, TicketPath: "ticket.md"})
			if err != nil {
				t.Fatal(err)
			}
			if got.Complete || got.TraversalComplete || len(got.Sources) != 1 || len(got.Diagnostics) != 1 || got.Diagnostics[0].Code != "invalid_frontmatter" {
				t.Fatalf("unexpected result: %+v", got)
			}
			if got.Sources[0].Text != source || got.Sources[0].SHA256 != fmt.Sprintf("%x", sha256.Sum256([]byte(source))) {
				t.Fatal("source/digest changed")
			}
			unchanged, _ := os.ReadFile(path)
			if string(unchanged) != source {
				t.Fatal("reader wrote source")
			}
		})
	}
}

func TestUnavailableAndOutsideSources(t *testing.T) {
	project := t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "external.md"), []byte("secret"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(outside, "external.md"), filepath.Join(project, "escape.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(project, "directory.md"), 0700); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ ticket, code string }{
		{"missing.md", "source_missing"}, {"directory.md", "source_unreadable"}, {"../outside.md", "source_outside_scope"}, {"escape.md", "source_outside_scope"}, {filepath.Join(outside, "external.md"), "source_outside_scope"},
	} {
		t.Run(tc.ticket, func(t *testing.T) {
			got, err := taskcontext.Assemble(context.Background(), taskcontext.Request{ProjectDir: project, TicketPath: tc.ticket})
			if err != nil {
				t.Fatal(err)
			}
			if got.Complete || got.TraversalComplete || len(got.Sources) != 0 || len(got.Diagnostics) != 1 || got.Diagnostics[0].Code != tc.code {
				t.Fatalf("unexpected result: %+v", got)
			}
		})
	}
}

func TestCanonicalInternalSymlink(t *testing.T) {
	project := t.TempDir()
	project, _ = filepath.EvalSymlinks(project)
	path := filepath.Join(project, "ticket.md")
	if err := os.WriteFile(path, []byte("# Ticket\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("ticket.md", filepath.Join(project, "alias.md")); err != nil {
		t.Fatal(err)
	}
	got, err := taskcontext.Assemble(context.Background(), taskcontext.Request{ProjectDir: project, TicketPath: "alias.md"})
	if err != nil || !got.Complete || got.Sources[0].Path != path {
		t.Fatalf("%+v %v", got, err)
	}
}

func TestOperationErrors(t *testing.T) {
	for _, request := range []taskcontext.Request{{}, {ProjectDir: filepath.Join(t.TempDir(), "absent"), TicketPath: "x"}} {
		if _, err := taskcontext.Assemble(context.Background(), request); err == nil {
			t.Fatal("expected operation error")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := taskcontext.Assemble(ctx, taskcontext.Request{ProjectDir: t.TempDir(), TicketPath: "x"}); err == nil {
		t.Fatal("expected cancellation")
	}
}

func TestMappingWrappers(t *testing.T) {
	for _, metadata := range []string{"&record\ntype: WorkItem", "!!map {type: WorkItem}", "{}"} {
		project := t.TempDir()
		if err := os.WriteFile(filepath.Join(project, "ticket.md"), []byte("---\n"+metadata+"\n---\n# Ticket\n"), 0600); err != nil {
			t.Fatal(err)
		}
		got, err := taskcontext.Assemble(context.Background(), taskcontext.Request{ProjectDir: project, TicketPath: "ticket.md"})
		if err != nil || !got.Complete {
			t.Fatalf("metadata=%q result=%+v error=%v", metadata, got, err)
		}
	}
}

func TestAbsoluteTicketThroughProjectAlias(t *testing.T) {
	parent := t.TempDir()
	project := filepath.Join(parent, "bundle")
	if err := os.Mkdir(project, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "ticket.md"), []byte("# Ticket\n"), 0600); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(parent, "alias")
	if err := os.Symlink(project, alias); err != nil {
		t.Fatal(err)
	}
	got, err := taskcontext.Assemble(context.Background(), taskcontext.Request{ProjectDir: alias, TicketPath: filepath.Join(alias, "ticket.md")})
	if err != nil || !got.Complete {
		t.Fatalf("%+v %v", got, err)
	}
}

func TestInvalidUTF8(t *testing.T) {
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, "ticket.md"), []byte{0xff}, 0600); err != nil {
		t.Fatal(err)
	}
	got, err := taskcontext.Assemble(context.Background(), taskcontext.Request{ProjectDir: project, TicketPath: "ticket.md"})
	if err != nil || got.Complete || got.TraversalComplete || len(got.Sources) != 0 || got.Diagnostics[0].Code != "invalid_source_encoding" {
		t.Fatalf("%+v %v", got, err)
	}
}
