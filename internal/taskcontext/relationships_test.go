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

func writeSources(t *testing.T, project string, files map[string]string) {
	t.Helper()
	for name, content := range files {
		path := filepath.Join(project, name)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
}

func readContext(t *testing.T, project, ticket string) taskcontext.Result {
	t.Helper()
	result, err := taskcontext.Assemble(context.Background(), taskcontext.Request{ProjectDir: project, TicketPath: ticket})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestRelationshipSectionsAndOrder(t *testing.T) {
	project := t.TempDir()
	project, _ = filepath.EvalSymlinks(project)
	ticket := "---\ntype: Strange\nexample: |\n  ## Spec\n  [metadata](missing.md)\n---\n# Ticket\n[ordinary](missing.md)\n\n## Context\n[context](../context.md)\n\n## Spec\n[first][one]\n### Nested\n[second](../second.md#absent-heading)\n\n## Other\n[ignored](missing.md)\n\n## Spec\n[third][]\n[shortcut]\n\n# End\n[ignored](missing.md)\n\n[one]: ../first.md\n[third]: /third.md\n[shortcut]: ../fourth.md\n"
	files := map[string]string{"issues/ticket.md": ticket, "context.md": "# Context\n## Spec\n[not selected](missing.md)", "first.md": "same", "second.md": "# Résumé\r\n", "third.md": "same", "fourth.md": "four"}
	writeSources(t, project, files)
	result := readContext(t, project, "issues/ticket.md")
	if !result.Complete || !result.TraversalComplete || len(result.Diagnostics) != 0 {
		t.Fatalf("%+v", result)
	}
	wantNames := []string{"issues/ticket.md", "first.md", "second.md", "third.md", "fourth.md", "context.md"}
	gotNames := []string{}
	for _, source := range result.Sources {
		name, _ := filepath.Rel(project, source.Path)
		gotNames = append(gotNames, name)
		if source.Text != files[name] || source.SHA256 != fmt.Sprintf("%x", sha256.Sum256([]byte(files[name]))) {
			t.Fatalf("changed source: %+v", source)
		}
		unchanged, _ := os.ReadFile(source.Path)
		if string(unchanged) != files[name] {
			t.Fatalf("source modified: %s", name)
		}
	}
	if diff := cmp.Diff(wantNames, gotNames); diff != "" {
		t.Fatal(diff)
	}
	wantReason := []taskcontext.Reason{{Kind: "spec", From: filepath.Join(project, "issues/ticket.md"), Link: "../second.md#absent-heading"}}
	if diff := cmp.Diff(wantReason, result.Sources[2].Reasons); diff != "" {
		t.Fatal(diff)
	}
}

func TestRelationshipExclusionsAndReferences(t *testing.T) {
	for name, content := range map[string]string{
		"ignored": "## Spec\n![image](missing.md) ![image][absent] ` [code][absent] ` [prose]\n\n    [indented](missing.md)\n\n```markdown\n## Context\n[fenced](missing.md)\n```\n\n## Other\n[unresolved][missing]\n",
		"case":    "## spec\n[not recognized](missing.md)\n### Spec\n[not recognized](missing.md)\n",
		"escaped": "## Spec\n\\[escaped][missing] ![image][]\n",
	} {
		t.Run(name, func(t *testing.T) {
			project := t.TempDir()
			writeSources(t, project, map[string]string{"ticket.md": content})
			result := readContext(t, project, "ticket.md")
			if !result.Complete || len(result.Sources) != 1 || len(result.Diagnostics) != 0 {
				t.Fatalf("%+v", result)
			}
		})
	}
	for _, reference := range []string{"[label][missing]", "[missing][]", "[**styled label**][missing]", "[label][missing\n reference]", "[doc][missing\\]label]", "[outer [inner]][missing]"} {
		t.Run(reference, func(t *testing.T) {
			project := t.TempDir()
			writeSources(t, project, map[string]string{"ticket.md": "## Spec\n" + reference + "\n[ok](ok.md)", "ok.md": "ok"})
			result := readContext(t, project, "ticket.md")
			if result.Complete || !result.TraversalComplete || len(result.Sources) != 2 || len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "unresolved_reference" || result.Diagnostics[0].Link == "" {
				t.Fatalf("%+v", result)
			}
		})
	}
}

func TestLinkedSourceAliasesAndReasons(t *testing.T) {
	project := t.TempDir()
	project, _ = filepath.EvalSymlinks(project)
	writeSources(t, project, map[string]string{"ticket.md": "## Context\n[a](alias.md#one)\n## Spec\n[a](shared.md#one)\n[a](shared.md#one)\n[b](shared.md#two)\n[self](#ticket)\n", "shared.md": "shared"})
	if err := os.Symlink("shared.md", filepath.Join(project, "alias.md")); err != nil {
		t.Fatal(err)
	}
	result := readContext(t, project, "ticket.md")
	if !result.Complete || len(result.Sources) != 2 {
		t.Fatalf("%+v", result)
	}
	want := []taskcontext.Reason{{Kind: "spec", From: filepath.Join(project, "ticket.md"), Link: "shared.md#one"}, {Kind: "spec", From: filepath.Join(project, "ticket.md"), Link: "shared.md#two"}, {Kind: "context", From: filepath.Join(project, "ticket.md"), Link: "alias.md#one"}}
	if diff := cmp.Diff(want, result.Sources[1].Reasons); diff != "" {
		t.Fatal(diff)
	}
	if len(result.Sources[0].Reasons) != 2 {
		t.Fatalf("root reasons: %+v", result.Sources[0].Reasons)
	}
}

func TestLinkedSourceFailuresPreserveAvailableContext(t *testing.T) {
	project := t.TempDir()
	outside := t.TempDir()
	writeSources(t, outside, map[string]string{"external.md": "outside"})
	if err := os.Symlink(filepath.Join(outside, "external.md"), filepath.Join(project, "escape.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(project, "directory.md"), 0700); err != nil {
		t.Fatal(err)
	}
	writeSources(t, project, map[string]string{"ticket.md": "## Spec\n[missing](missing.md)\n[dir](directory.md)\n[escape](escape.md)\n[outside](../outside.md)\n[remote](https://example.invalid/doc)\n<https://example.invalid/auto>\n[good](good.md)\n## Context\n[raw](raw.md)", "good.md": "available", "raw.md": "---\nmalformed: [\n---\n## Spec\n[ignored](not-selected.md)"})
	result := readContext(t, project, "ticket.md")
	if result.Complete || !result.TraversalComplete || len(result.Sources) != 3 {
		t.Fatalf("%+v", result)
	}
	codes := []string{}
	for _, diagnostic := range result.Diagnostics {
		codes = append(codes, diagnostic.Code)
		if diagnostic.From == "" || diagnostic.Link == "" {
			t.Fatalf("missing provenance: %+v", diagnostic)
		}
	}
	want := []string{"source_missing", "source_unreadable", "source_outside_scope", "source_outside_scope", "unsupported_source", "unsupported_source"}
	if diff := cmp.Diff(want, codes); diff != "" {
		t.Fatal(diff)
	}
}

func TestEncodedAndEscapedDestinations(t *testing.T) {
	project := t.TempDir()
	writeSources(t, project, map[string]string{"ticket.md": "## Spec\n[a](space%20name.md#heading)\n[b](a\\(b\\).md)\n[c](a&amp;b.md)", "space name.md": "space", "a(b).md": "parens", "a&b.md": "amp"})
	result := readContext(t, project, "ticket.md")
	if !result.Complete || len(result.Sources) != 4 {
		t.Fatalf("%+v", result)
	}
	if result.Sources[2].Reasons[0].Link != "a\\(b\\).md" {
		t.Fatalf("lost authored link: %+v", result.Sources[2])
	}
}

func TestReferenceDefinitionsWinOverMissingLabels(t *testing.T) {
	project := t.TempDir()
	writeSources(t, project, map[string]string{
		"ticket.md": "## Spec\n[good][escaped\\]label]\n[good][KNOWN]\n[bad][missing]\n\n[escaped\\]label]: good.md\n[known]: good.md#section\n[known]: absent.md\n",
		"good.md":   "current source",
	})
	result := readContext(t, project, "ticket.md")
	if result.Complete || !result.TraversalComplete || len(result.Sources) != 2 || len(result.Sources[1].Reasons) != 2 || len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "unresolved_reference" {
		t.Fatalf("%+v", result)
	}
}

func TestNestedBracketLabelsPreserveNormalLinks(t *testing.T) {
	for _, label := range []string{"outer [inner]", "outer [inner][missing]", "outer [inner][]", "outer `code`"} {
		t.Run(label, func(t *testing.T) {
			project := t.TempDir()
			writeSources(t, project, map[string]string{
				"ticket.md": "## Spec\n[" + label + "](doc.md)\n![" + label + "](ignored.md)\n",
				"doc.md":    "required source",
			})
			result := readContext(t, project, "ticket.md")
			if !result.Complete || !result.TraversalComplete || len(result.Sources) != 2 || len(result.Diagnostics) != 0 || result.Sources[1].Text != "required source" {
				t.Fatalf("%+v", result)
			}
		})
	}
}
