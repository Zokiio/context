package discoveryconfig_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Zokiio/context/internal/discoveryconfig"
)

func TestConfigSymlinkUsesEncounteredDirectory(t *testing.T) {
	checkout := directory(t)
	elsewhere := directory(t)
	records := filepath.Join(checkout, "records")
	mkdir(t, records)
	// Both candidate bases exist, so a mistaken rebase would look usable.
	mkdir(t, filepath.Join(elsewhere, "records"))
	sourceFile := filepath.Join(elsewhere, ".context", "source.md")
	write(t, sourceFile, markdown("project: {records: ../records}\n"))
	encountered := filepath.Join(checkout, ".context", "config.md")
	symlink(t, sourceFile, encountered)
	doc, err := discoveryconfig.Read(encountered, discoveryconfig.SharedProfile)
	if err != nil {
		t.Fatal(err)
	}
	assertTarget(t, doc.Shared.Project.Records, encountered, "project.records", "../records", records, records)
	if doc.File != encountered {
		t.Fatal("the encountered source filename was replaced by its target")
	}
}

func TestTargetAliasesAndLiteralPaths(t *testing.T) {
	root := directory(t)
	file := filepath.Join(root, ".context", "config.md")
	records := directory(t)
	alias := filepath.Join(root, "records-link")
	symlink(t, records, alias)
	for _, name := range []string{"~", "$CTX_CONFIG_TEST", "docs,notes", "docs%20notes"} {
		mkdir(t, filepath.Join(root, ".context", name))
	}
	t.Setenv("CTX_CONFIG_TEST", directory(t))
	write(t, file, markdown("project:\n  records: ../records-link\n  allowSources: ['~', '$CTX_CONFIG_TEST', 'docs,notes', 'docs%20notes']\n"))
	doc, err := discoveryconfig.Read(file, discoveryconfig.SharedProfile)
	if err != nil {
		t.Fatal(err)
	}
	assertTarget(t, doc.Shared.Project.Records, file, "project.records", "../records-link", alias, records)
	for _, path := range doc.Shared.Project.AllowSources {
		want := filepath.Join(filepath.Dir(file), path.Authored)
		if path.Resolved != want || path.Canonical != want || path.Failure != nil {
			t.Fatalf("a literal filesystem value was interpreted: %+v", path)
		}
	}
}

func TestTargetFailuresRemainAttributable(t *testing.T) {
	root := directory(t)
	file := filepath.Join(root, "config.md")
	symlink(t, "loop", filepath.Join(root, "loop"))
	write(t, filepath.Join(root, "regular-file"), "not a directory")
	write(t, file, markdown("projects:\n  - {key: good, directory: ., records: .}\n  - {key: absent, directory: missing, records: loop, allowSources: [regular-file/child]}\n"))
	doc, err := discoveryconfig.Read(file, discoveryconfig.PersonalProfile)
	if err != nil {
		t.Fatalf("unrelated unavailable paths invalidated structure: %v", err)
	}
	assertTarget(t, doc.Personal.Projects[0].Records, file, "projects[0].records", ".", root, root)
	missing := doc.Personal.Projects[1]
	for _, path := range []discoveryconfig.TargetPath{missing.Directory, missing.Records, missing.AllowSources[0]} {
		if path.Canonical != "" || path.Failure == nil {
			t.Fatalf("unresolved path reported as canonical: %+v", path)
		}
		failure := path.Failure
		if failure.File != file || failure.Field != path.Field || failure.Value != path.Authored || failure.Resolved != path.Resolved {
			t.Fatalf("path failure lost provenance: %+v", path)
		}
		if !strings.Contains(failure.Error(), path.Field) || !strings.Contains(failure.Error(), path.Resolved) || errors.Unwrap(failure) == nil {
			t.Fatalf("path error lost its cause or location: %v", failure)
		}
	}
	if !errors.Is(missing.Directory.Failure, os.ErrNotExist) || errors.Is(missing.Records.Failure, os.ErrNotExist) {
		t.Fatal("missing targets and symlink loops became indistinguishable")
	}
	// Identity observation is separate from the selected-directory access check.
	write(t, file, markdown("project: {records: regular-file}\n"))
	doc, err = discoveryconfig.Read(file, discoveryconfig.SharedProfile)
	if err != nil || doc.Shared.Project.Records.Failure != nil || doc.Shared.Project.Records.Canonical != filepath.Join(root, "regular-file") {
		t.Fatalf("canonical identity was conflated with directory validation: %+v, %v", doc, err)
	}
}

func TestUnknownYAMLSyntaxRemainsInOriginalSource(t *testing.T) {
	file := filepath.Join(directory(t), "config.md")
	source := markdown("# Retained YAML comment\ncustom: &custom\n  17: tagged\n  retained: !!str 001\nother: *custom\nproject: {records: .}\n") + "\nKeep **Markdown** and [links](../notes.md).\n"
	write(t, file, source)
	doc, err := discoveryconfig.Read(file, discoveryconfig.SharedProfile)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Source != source || doc.Metadata["custom"] == nil || doc.Metadata["other"] == nil || doc.Body != "\nKeep **Markdown** and [links](../notes.md).\n" {
		t.Fatal("unknown YAML syntax or Markdown was lost")
	}
}
