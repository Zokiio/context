package discoveryconfig_test

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Zokiio/context/internal/discoveryconfig"
)

func TestReadSharedConfiguration(t *testing.T) {
	root := directory(t)
	records := directory(t) // Records live outside the checkout, without Git.
	docs := filepath.Join(root, "docs")
	mkdir(t, docs)
	file := filepath.Join(root, ".context", "config.md")
	body := "\r\n# Configuration café\r\n\r\nKeep this prose.\r\n"
	source := fmt.Sprintf("---\r\ntype: ContextConfig\r\nversion: 1\r\ncustom: {nested: [one, 2]}\r\nproject:\r\n  records: %q\r\n  allowSources: [../docs]\r\n  owner: keep me\r\nworkspace:\r\n  id: workspace-one\r\n  title: Workspace one\r\n  members:\r\n    - {key: second, records: ../other, title: Second, directory: ../checkout, custom: {value: kept}}\r\n    - {key: first, records: %q, allowSources: [../docs]}\r\n---\r\n%s", records, records, body)
	write(t, file, source)
	got, err := discoveryconfig.Read(file, discoveryconfig.SharedProfile)
	if err != nil {
		t.Fatal(err)
	}
	if got.File != file || got.Source != source || got.Body != body || got.Shared == nil || got.Personal != nil {
		t.Fatalf("unexpected document: %+v", got)
	}
	project := got.Shared.Project
	assertTarget(t, project.Records, file, "project.records", records, records, records)
	if len(project.AllowSources) != 1 {
		t.Fatalf("allowed sources: %+v", project.AllowSources)
	}
	assertTarget(t, project.AllowSources[0], file, "project.allowSources[0]", "../docs", docs, docs)
	workspace := got.Shared.Workspace
	if workspace.ID != "workspace-one" || workspace.Title != "Workspace one" || len(workspace.Members) != 2 {
		t.Fatalf("workspace: %+v", workspace)
	}
	if workspace.Members[0].Key != "second" || workspace.Members[1].Key != "first" || workspace.Members[1].Directory != nil {
		t.Fatalf("member order or optional directory changed: %+v", workspace.Members)
	}
	if workspace.Members[0].Title == nil || *workspace.Members[0].Title != "Second" || workspace.Members[1].Title != nil {
		t.Fatalf("member titles changed: %+v", workspace.Members)
	}
	if workspace.Members[0].AllowSources == nil || len(workspace.Members[0].AllowSources) != 0 {
		t.Fatal("workspace member inherited another declaration's allowed sources")
	}
	if got.Metadata["project"].(map[string]any)["owner"] != "keep me" || got.Metadata["custom"] == nil {
		t.Fatal("unknown metadata was discarded")
	}
	members := got.Metadata["workspace"].(map[string]any)["members"].([]any)
	if members[0].(map[string]any)["custom"].(map[string]any)["value"] != "kept" {
		t.Fatal("nested unknown metadata was discarded")
	}
	unchanged, err := os.ReadFile(file)
	if err != nil || string(unchanged) != source {
		t.Fatalf("source changed: %q, %v", unchanged, err)
	}
	entries, err := os.ReadDir(filepath.Dir(file))
	if err != nil || len(entries) != 1 {
		t.Fatalf("configuration read created files: %v, %v", entries, err)
	}
	if _, err := os.Stat(filepath.Join(root, "other")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("read created a missing record location: %v", err)
	}
}

func TestReadPersonalConfiguration(t *testing.T) {
	root := directory(t)
	records := directory(t)
	file := filepath.Join(root, ".context", "config.md")
	write(t, file, markdown(fmt.Sprintf(`projects:
  - key: same-key
    alias: same-alias
    directory: ../missing-checkout
    records: %q
    custom: retain
  - key: second
    directory: ../second-checkout
    records: ../missing-records
workspaces:
  - key: same-key
    alias: same-alias
    id: same-id
    title: Unbound
    members: []
  - key: bound
    directory: ../bound
    id: same-id
    title: Bound
    members: [{key: same-key, records: ../member}]
`, records)))
	got, err := discoveryconfig.Read(file, discoveryconfig.PersonalProfile)
	if err != nil {
		t.Fatal(err)
	}
	if got.Shared != nil || got.Personal == nil || len(got.Personal.Projects) != 2 || len(got.Personal.Workspaces) != 2 {
		t.Fatalf("personal profile: %+v", got)
	}
	project := got.Personal.Projects[0]
	if project.Key != "same-key" || project.Alias == nil || *project.Alias != "same-alias" || got.Personal.Projects[1].Alias != nil {
		t.Fatalf("project registration: %+v", got.Personal.Projects)
	}
	assertTarget(t, project.Records, file, "projects[0].records", records, records, records)
	if !errors.Is(project.Directory.Failure, os.ErrNotExist) || project.Directory.Canonical != "" {
		t.Fatalf("missing checkout failure lost: %+v", project.Directory)
	}
	if got.Personal.Workspaces[0].Directory != nil || got.Personal.Workspaces[1].Directory == nil {
		t.Fatalf("workspace bindings: %+v", got.Personal.Workspaces)
	}
	if got.Personal.Workspaces[0].Members == nil || len(got.Personal.Workspaces[0].Members) != 0 {
		t.Fatal("explicitly empty membership was not retained")
	}
	if got.Personal.Projects[1].Records.Failure == nil {
		t.Fatal("unavailable unrelated records were lost")
	}
}

func TestEmptyPersonalRegistryAndOptionalValues(t *testing.T) {
	file := filepath.Join(directory(t), "config.md")
	for _, fields := range []string{"", "projects: []\nworkspaces: []\n"} {
		write(t, file, markdown(fields))
		doc, err := discoveryconfig.Read(file, discoveryconfig.PersonalProfile)
		if err != nil {
			t.Fatal(err)
		}
		if doc.Personal.Projects == nil || doc.Personal.Workspaces == nil || len(doc.Personal.Projects)+len(doc.Personal.Workspaces) != 0 {
			t.Fatalf("empty personal registry: %+v", doc.Personal)
		}
	}
	write(t, file, markdown("projects: [{key: p, alias: '', directory: ., records: .}]\nworkspaces: [{key: w, id: w, title: W, members: [{key: m, title: '', records: .}]}]\n"))
	doc, err := discoveryconfig.Read(file, discoveryconfig.PersonalProfile)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Personal.Projects[0].Alias == nil || *doc.Personal.Projects[0].Alias != "" || doc.Personal.Workspaces[0].Members[0].Title == nil || *doc.Personal.Workspaces[0].Members[0].Title != "" {
		t.Fatal("empty optional strings became absent values")
	}
}

func TestSharedDeclarationsWorkIndependently(t *testing.T) {
	file := filepath.Join(directory(t), "config.md")
	write(t, file, markdown("project: {records: .}\n"))
	doc, err := discoveryconfig.Read(file, discoveryconfig.SharedProfile)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Shared.Workspace != nil || doc.Shared.Project.AllowSources == nil || len(doc.Shared.Project.AllowSources) != 0 {
		t.Fatalf("project-only defaults: %+v", doc.Shared)
	}
	write(t, file, markdown("workspace: {id: w, title: W, members: []}\n"))
	doc, err = discoveryconfig.Read(file, discoveryconfig.SharedProfile)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Shared.Project != nil || doc.Shared.Workspace.Members == nil || len(doc.Shared.Workspace.Members) != 0 {
		t.Fatalf("workspace-only defaults: %+v", doc.Shared)
	}
}

func TestConfigurationFieldErrors(t *testing.T) {
	for _, tc := range []struct {
		name    string
		profile discoveryconfig.Profile
		fields  string
		field   string
	}{
		{"empty shared", discoveryconfig.SharedProfile, "", "$"},
		{"mixed shared", discoveryconfig.SharedProfile, "project: {records: .}\nprojects: []", "projects"},
		{"mixed personal", discoveryconfig.PersonalProfile, "workspace: null", "workspace"},
		{"project scalar", discoveryconfig.SharedProfile, "project: false", "project"},
		{"project null", discoveryconfig.SharedProfile, "project: null", "project"},
		{"records missing", discoveryconfig.SharedProfile, "project: {}", "project.records"},
		{"records numeric", discoveryconfig.SharedProfile, "project: {records: 17}", "project.records"},
		{"records null", discoveryconfig.SharedProfile, "project: {records: null}", "project.records"},
		{"records empty", discoveryconfig.SharedProfile, "project: {records: '  '}", "project.records"},
		{"roots scalar", discoveryconfig.SharedProfile, "project: {records: ., allowSources: docs}", "project.allowSources"},
		{"roots null", discoveryconfig.SharedProfile, "project: {records: ., allowSources: null}", "project.allowSources"},
		{"root type", discoveryconfig.SharedProfile, "project: {records: ., allowSources: [true]}", "project.allowSources[0]"},
		{"root empty", discoveryconfig.SharedProfile, "project: {records: ., allowSources: ['']}", "project.allowSources[0]"},
		{"workspace scalar", discoveryconfig.SharedProfile, "workspace: []", "workspace"},
		{"workspace id", discoveryconfig.SharedProfile, "workspace: {id: '', title: W, members: []}", "workspace.id"},
		{"workspace title", discoveryconfig.SharedProfile, "workspace: {id: w, title: true, members: []}", "workspace.title"},
		{"members missing", discoveryconfig.SharedProfile, "workspace: {id: w, title: W}", "workspace.members"},
		{"members type", discoveryconfig.SharedProfile, "workspace: {id: w, title: W, members: {}}", "workspace.members"},
		{"member scalar", discoveryconfig.SharedProfile, "workspace: {id: w, title: W, members: [true]}", "workspace.members[0]"},
		{"member key", discoveryconfig.SharedProfile, "workspace: {id: w, title: W, members: [{records: .}]}", "workspace.members[0].key"},
		{"member title", discoveryconfig.SharedProfile, "workspace: {id: w, title: W, members: [{key: m, title: 2, records: .}]}", "workspace.members[0].title"},
		{"member directory", discoveryconfig.SharedProfile, "workspace: {id: w, title: W, members: [{key: m, directory: null, records: .}]}", "workspace.members[0].directory"},
		{"member records", discoveryconfig.SharedProfile, "workspace: {id: w, title: W, members: [{key: m}]}", "workspace.members[0].records"},
		{"duplicate members", discoveryconfig.SharedProfile, "workspace: {id: w, title: W, members: [{key: m, records: a}, {key: m, records: b}]}", "workspace.members[1].key"},
		{"projects type", discoveryconfig.PersonalProfile, "projects: {}", "projects"},
		{"projects null", discoveryconfig.PersonalProfile, "projects: null", "projects"},
		{"project entry", discoveryconfig.PersonalProfile, "projects: [bad]", "projects[0]"},
		{"registration key", discoveryconfig.PersonalProfile, "projects: [{key: '', directory: ., records: .}]", "projects[0].key"},
		{"registration directory", discoveryconfig.PersonalProfile, "projects: [{key: p, alias: p, records: .}]", "projects[0].directory"},
		{"alias type", discoveryconfig.PersonalProfile, "projects: [{key: p, alias: 2, directory: ., records: .}]", "projects[0].alias"},
		{"duplicate project keys", discoveryconfig.PersonalProfile, "projects: [{key: p, directory: a, records: a}, {key: p, directory: b, records: b}]", "projects[1].key"},
		{"duplicate project aliases", discoveryconfig.PersonalProfile, "projects: [{key: p, alias: dup, directory: a, records: a}, {key: q, alias: dup, directory: b, records: b}]", "projects[1].alias"},
		{"workspaces type", discoveryconfig.PersonalProfile, "workspaces: true", "workspaces"},
		{"workspace registration", discoveryconfig.PersonalProfile, "workspaces: [null]", "workspaces[0]"},
		{"workspace alias", discoveryconfig.PersonalProfile, "workspaces: [{key: w, alias: null, id: w, title: W, members: []}]", "workspaces[0].alias"},
		{"workspace directory", discoveryconfig.PersonalProfile, "workspaces: [{key: w, directory: '', id: w, title: W, members: []}]", "workspaces[0].directory"},
		{"duplicate workspace keys", discoveryconfig.PersonalProfile, "workspaces: [{key: w, id: w, title: W, members: []}, {key: w, id: q, title: Q, members: []}]", "workspaces[1].key"},
		{"duplicate workspace aliases", discoveryconfig.PersonalProfile, "workspaces: [{key: w, alias: dup, id: w, title: W, members: []}, {key: q, alias: dup, id: q, title: Q, members: []}]", "workspaces[1].alias"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			file := filepath.Join(directory(t), "config.md")
			write(t, file, markdown(tc.fields+"\n"))
			got, err := discoveryconfig.Read(file, tc.profile)
			var fieldErr *discoveryconfig.FieldError
			if !errors.As(err, &fieldErr) || fieldErr.File != file || fieldErr.Field != tc.field || !strings.Contains(err.Error(), tc.field) {
				t.Fatalf("want field %s, got %v", tc.field, err)
			}
			if got.Shared != nil || got.Personal != nil {
				t.Fatal("invalid configuration was returned for use")
			}
			if strings.HasPrefix(tc.name, "duplicate") && !strings.Contains(err.Error(), "first declared at") {
				t.Fatal("duplicate diagnostic omitted the first declaration")
			}
		})
	}
}

func TestFrontmatterAndVersionErrors(t *testing.T) {
	for name, source := range map[string]string{
		"no frontmatter":    "# config\n",
		"empty":             "---\n---\n",
		"unterminated":      "---\ntype: ContextConfig\n",
		"malformed":         "---\ntype: [broken\n---\n",
		"sequence":          "---\n- ContextConfig\n---\n",
		"duplicate root":    markdown("version: 1\n"),
		"duplicate nested":  markdown("project: {records: first, records: second}\n"),
		"duplicate unknown": markdown("custom: {same: first, same: second}\n"),
		"invalid encoding":  markdown("custom: \xff\n"),
	} {
		t.Run(name, func(t *testing.T) {
			file := filepath.Join(directory(t), "config.md")
			write(t, file, source)
			_, err := discoveryconfig.Read(file, discoveryconfig.PersonalProfile)
			var fieldErr *discoveryconfig.FieldError
			if !errors.As(err, &fieldErr) || fieldErr.File != file || fieldErr.Field != "$" {
				t.Fatalf("expected a source diagnostic: %v", err)
			}
		})
	}
	for _, version := range []string{"", "0", "2", "-1", "1.0", "'1'", "true", "null", "[1]", "{}"} {
		t.Run("version="+version, func(t *testing.T) {
			file := filepath.Join(directory(t), "config.md")
			write(t, file, "---\ntype: ContextConfig\nversion: "+version+"\n---\n")
			_, err := discoveryconfig.Read(file, discoveryconfig.PersonalProfile)
			var fieldErr *discoveryconfig.FieldError
			if !errors.As(err, &fieldErr) || fieldErr.Field != "version" {
				t.Fatalf("invalid version accepted or misidentified: %v", err)
			}
		})
	}
	for _, fields := range []string{"version: 1", "type: Project\nversion: 1", "type: true\nversion: 1", "type: ' '\nversion: 1"} {
		file := filepath.Join(directory(t), "config.md")
		write(t, file, "---\n"+fields+"\n---\n")
		_, err := discoveryconfig.Read(file, discoveryconfig.PersonalProfile)
		var fieldErr *discoveryconfig.FieldError
		if !errors.As(err, &fieldErr) || fieldErr.Field != "type" {
			t.Fatalf("invalid type: %v", err)
		}
	}
}

func TestReadFailuresAndExplicitInputs(t *testing.T) {
	for _, tc := range []struct {
		file    string
		profile discoveryconfig.Profile
	}{
		{"config.md", discoveryconfig.SharedProfile},
		{"", discoveryconfig.PersonalProfile},
		{filepath.Join(directory(t), "config.md"), 0},
	} {
		_, err := discoveryconfig.Read(tc.file, tc.profile)
		var fieldErr *discoveryconfig.FieldError
		if !errors.As(err, &fieldErr) || fieldErr.File != tc.file {
			t.Fatalf("expected explicit input error, got %v", err)
		}
	}
	file := filepath.Join(directory(t), "missing.md")
	if _, err := discoveryconfig.Read(file, discoveryconfig.PersonalProfile); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing configuration read cause lost: %v", err)
	}
	if _, err := discoveryconfig.Read(directory(t), discoveryconfig.SharedProfile); err == nil {
		t.Fatal("a directory was read as configuration")
	}
}

func markdown(fields string) string {
	return "---\ntype: ContextConfig\nversion: 1\n" + fields + "---\n"
}

func directory(t *testing.T) string {
	t.Helper()
	path, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return path
}

func mkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0700); err != nil {
		t.Fatal(err)
	}
}

func write(t *testing.T, file, source string) {
	t.Helper()
	mkdir(t, filepath.Dir(file))
	if err := os.WriteFile(file, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
}

func symlink(t *testing.T, target, link string) {
	t.Helper()
	mkdir(t, filepath.Dir(link))
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
}

func assertTarget(t *testing.T, got discoveryconfig.TargetPath, file, field, authored, resolved, canonical string) {
	t.Helper()
	if got.File != file || got.Field != field || got.Authored != authored || got.Resolved != resolved || got.Canonical != canonical || got.Failure != nil {
		t.Fatalf("%s: got %+v; want %q -> %q -> %q", field, got, authored, resolved, canonical)
	}
}
