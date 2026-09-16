package discovery_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Zokiio/context/internal/discovery"
)

func configDocument(fields string) string {
	return "---\ntype: ContextConfig\nversion: 1\n" + fields + "\n---\n"
}

func writeConfig(t *testing.T, path, text string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

func physicalTempDir(t *testing.T) string {
	t.Helper()
	path, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return path
}

func TestReadSharedConfigPreservesDocumentAndResolvesSeparateRecords(t *testing.T) {
	root := physicalTempDir(t)
	path := filepath.Join(root, "checkout", ".context", "config.md")
	records := filepath.Join(root, "records-repository", "bundle")
	for _, directory := range []string{records, filepath.Join(root, "docs")} {
		if err := os.MkdirAll(directory, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	body := "\r\n# Checkout notes\r\n\r\nPreserve these **authored** bytes.\r\n"
	source := configDocument(`owner:
  team: tools
project:
  records: ../../records-repository/bundle
  allowSources: [../../docs]
  custom:
    labels: [one, two]
workspace:
  id: shared-workspace
  title: Shared workspace
  members:
    - key: independent
      title: Independent project
      directory: ../../other-checkout
      records: ../../other-records
      allowSources: [../../other-docs]
      custom: keep me`) + body
	writeConfig(t, path, source)
	config, err := discovery.ReadConfig(path, discovery.SharedConfig)
	if err != nil {
		t.Fatal(err)
	}
	if config.Path != path || config.Profile != discovery.SharedConfig || config.Project == nil || config.Workspace == nil {
		t.Fatalf("unexpected declarations: %#v", config)
	}
	if !bytes.Equal(config.Raw, []byte(source)) || !bytes.Equal(config.Body, []byte(body)) {
		t.Fatal("document bytes were not retained")
	}
	if config.Metadata["owner"].(map[string]any)["team"] != "tools" {
		t.Fatal("unknown top-level metadata was not retained")
	}
	projectMetadata := config.Metadata["project"].(map[string]any)
	if projectMetadata["custom"].(map[string]any)["labels"].([]any)[1] != "two" {
		t.Fatal("unknown nested project metadata was not retained")
	}
	memberMetadata := config.Metadata["workspace"].(map[string]any)["members"].([]any)[0].(map[string]any)
	if memberMetadata["custom"] != "keep me" {
		t.Fatal("unknown member metadata was not retained")
	}
	binding := config.Project
	if binding.Records.Authored != "../../records-repository/bundle" || binding.Records.Canonical != records || binding.Records.Err != nil {
		t.Fatalf("records path = %#v", binding.Records)
	}
	if binding.Records.Origin.Path != path || binding.Records.Origin.Field != "project.records" {
		t.Fatalf("records origin = %#v", binding.Records.Origin)
	}
	if len(binding.AllowSources) != 1 || binding.AllowSources[0].Canonical != filepath.Join(root, "docs") {
		t.Fatalf("allowed roots = %#v", binding.AllowSources)
	}
	workspace := config.Workspace
	if workspace.ID != "shared-workspace" || workspace.Title != "Shared workspace" || len(workspace.Members) != 1 {
		t.Fatalf("workspace = %#v", workspace)
	}
	member := workspace.Members[0]
	if member.Key != "independent" || member.Title != "Independent project" || member.Directory == nil || member.Directory.Canonical != filepath.Join(root, "other-checkout") {
		t.Fatalf("member = %#v", member)
	}
	if member.Records.Canonical != filepath.Join(root, "other-records") || len(member.AllowSources) != 1 || member.AllowSources[0].Canonical != filepath.Join(root, "other-docs") {
		t.Fatalf("member paths = %#v", member)
	}
}

func TestReadPersonalConfigAllowsUnavailableRegistrations(t *testing.T) {
	root := physicalTempDir(t)
	path := filepath.Join(root, "home", ".context", "config.md")
	source := configDocument(`projects:
  - key: project-one
    alias: shared-name
    directory: ../../moved-checkout
    records: ../../records
  - key: project-two
    directory: ../../another-checkout
    records: ../../other-records
    allowSources: [../../missing-docs]
workspaces:
  - key: workspace-one
    alias: shared-name
    id: workspace-one-id
    title: Alias-only workspace
    members: []
  - key: workspace-two
    directory: ../../team
    id: workspace-two-id
    title: Directory workspace
    members:
      - key: same-project-first-checkout
        records: ../../records
        directory: ../../first-checkout
      - key: same-project-second-checkout
        records: ../../records
        directory: ../../second-checkout`)
	writeConfig(t, path, source)
	config, err := discovery.ReadConfig(path, discovery.PersonalConfig)
	if err != nil {
		t.Fatal(err)
	}
	if config.Project != nil || config.Workspace != nil || len(config.Projects) != 2 || len(config.Workspaces) != 2 {
		t.Fatalf("config = %#v", config)
	}
	project := config.Projects[0]
	if project.Key != "project-one" || project.Alias != "shared-name" || project.Directory.Canonical != filepath.Join(root, "moved-checkout") || project.Directory.Err != nil {
		t.Fatalf("project = %#v", project)
	}
	if project.AllowSources == nil || len(project.AllowSources) != 0 {
		t.Fatalf("default allowSources = %#v", project.AllowSources)
	}
	if config.Workspaces[0].Directory != nil || config.Workspaces[0].Alias != "shared-name" || len(config.Workspaces[0].Members) != 0 {
		t.Fatalf("alias-only workspace = %#v", config.Workspaces[0])
	}
	workspace := config.Workspaces[1]
	if workspace.Directory == nil || workspace.Directory.Canonical != filepath.Join(root, "team") || len(workspace.Members) != 2 {
		t.Fatalf("workspace = %#v", workspace)
	}
	if workspace.Members[0].Records.Canonical != workspace.Members[1].Records.Canonical {
		t.Fatal("identical records were not canonicalized equally")
	}
}

func TestPersonalArraysDefaultToEmpty(t *testing.T) {
	config, err := discovery.ParseConfig(filepath.Join(t.TempDir(), "config.md"), []byte(configDocument("notes: no registrations")), discovery.PersonalConfig)
	if err != nil {
		t.Fatal(err)
	}
	if config.Projects == nil || config.Workspaces == nil || len(config.Projects) != 0 || len(config.Workspaces) != 0 {
		t.Fatalf("default arrays = %#v, %#v", config.Projects, config.Workspaces)
	}
}

func TestConfigFieldDiagnostics(t *testing.T) {
	tests := []struct {
		name    string
		profile discovery.ConfigProfile
		fields  string
		field   string
	}{
		{"shared empty", discovery.SharedConfig, "notes: none", "project/workspace"},
		{"shared personal projects", discovery.SharedConfig, "projects: []", "projects"},
		{"shared personal workspaces", discovery.SharedConfig, "project: {records: ../records}\nworkspaces: []", "workspaces"},
		{"personal shared project", discovery.PersonalConfig, "project: {records: ../records}", "project"},
		{"personal shared workspace", discovery.PersonalConfig, "workspace: {id: w, title: W, members: []}", "workspace"},
		{"project scalar", discovery.SharedConfig, "project: wrong", "project"},
		{"project null", discovery.SharedConfig, "project: null", "project"},
		{"records missing", discovery.SharedConfig, "project: {}", "project.records"},
		{"records integer", discovery.SharedConfig, "project: {records: 1}", "project.records"},
		{"records blank", discovery.SharedConfig, "project: {records: '  '}", "project.records"},
		{"roots scalar", discovery.SharedConfig, "project: {records: ../records, allowSources: ../docs}", "project.allowSources"},
		{"roots null", discovery.SharedConfig, "project: {records: ../records, allowSources: null}", "project.allowSources"},
		{"root bool", discovery.SharedConfig, "project: {records: ../records, allowSources: [true]}", "project.allowSources[0]"},
		{"root blank", discovery.SharedConfig, "project: {records: ../records, allowSources: ['']}", "project.allowSources[0]"},
		{"workspace scalar", discovery.SharedConfig, "workspace: []", "workspace"},
		{"workspace id missing", discovery.SharedConfig, "workspace: {title: W, members: []}", "workspace.id"},
		{"workspace id blank", discovery.SharedConfig, "workspace: {id: '', title: W, members: []}", "workspace.id"},
		{"workspace title missing", discovery.SharedConfig, "workspace: {id: w, members: []}", "workspace.title"},
		{"workspace title bool", discovery.SharedConfig, "workspace: {id: w, title: false, members: []}", "workspace.title"},
		{"members missing", discovery.SharedConfig, "workspace: {id: w, title: W}", "workspace.members"},
		{"members scalar", discovery.SharedConfig, "workspace: {id: w, title: W, members: wrong}", "workspace.members"},
		{"member scalar", discovery.SharedConfig, "workspace: {id: w, title: W, members: [wrong]}", "workspace.members[0]"},
		{"member key missing", discovery.SharedConfig, "workspace: {id: w, title: W, members: [{records: ../r}]}", "workspace.members[0].key"},
		{"member title bool", discovery.SharedConfig, "workspace: {id: w, title: W, members: [{key: m, records: ../r, title: false}]}", "workspace.members[0].title"},
		{"member directory null", discovery.SharedConfig, "workspace: {id: w, title: W, members: [{key: m, records: ../r, directory: null}]}", "workspace.members[0].directory"},
		{"member records missing", discovery.SharedConfig, "workspace: {id: w, title: W, members: [{key: m}]}", "workspace.members[0].records"},
		{"member roots invalid", discovery.SharedConfig, "workspace: {id: w, title: W, members: [{key: m, records: ../r, allowSources: [{}]}]}", "workspace.members[0].allowSources[0]"},
		{"project registrations mapping", discovery.PersonalConfig, "projects: {}", "projects"},
		{"project registration scalar", discovery.PersonalConfig, "projects: [wrong]", "projects[0]"},
		{"project key missing", discovery.PersonalConfig, "projects: [{directory: ../d, records: ../r}]", "projects[0].key"},
		{"project alias number", discovery.PersonalConfig, "projects: [{key: p, directory: ../d, records: ../r, alias: 12}]", "projects[0].alias"},
		{"project directory missing", discovery.PersonalConfig, "projects: [{key: p, records: ../r}]", "projects[0].directory"},
		{"workspace registrations null", discovery.PersonalConfig, "workspaces: null", "workspaces"},
		{"workspace registration scalar", discovery.PersonalConfig, "workspaces: [false]", "workspaces[0]"},
		{"workspace registration key missing", discovery.PersonalConfig, "workspaces: [{id: w, title: W, members: []}]", "workspaces[0].key"},
		{"workspace registration alias null", discovery.PersonalConfig, "workspaces: [{key: w, alias: null, id: w, title: W, members: []}]", "workspaces[0].alias"},
		{"workspace registration directory blank", discovery.PersonalConfig, "workspaces: [{key: w, directory: '', id: w, title: W, members: []}]", "workspaces[0].directory"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), ".context", "config.md")
			writeConfig(t, path, configDocument(test.fields))
			_, err := discovery.ReadConfig(path, test.profile)
			var diagnostic *discovery.ConfigError
			if !errors.As(err, &diagnostic) || diagnostic.Field != test.field || diagnostic.Path != path {
				t.Fatalf("error = %v, want field %q at %q", err, test.field, path)
			}
			if !strings.Contains(err.Error(), "correct") {
				t.Fatalf("error lacks corrective guidance: %v", err)
			}
		})
	}
}

func TestConfigRejectsMalformedAndUnsupportedDocuments(t *testing.T) {
	tests := []struct {
		name   string
		source string
		field  string
	}{
		{"no frontmatter", "# Plain Markdown\n", "frontmatter"},
		{"unterminated", "---\ntype: ContextConfig\n", "frontmatter"},
		{"malformed", "---\ntype: [unterminated\n---\n", "frontmatter"},
		{"frontmatter array", "---\n- ContextConfig\n---\n", "frontmatter"},
		{"duplicate root key", configDocument("project: {records: ../r}\nproject: {records: ../r}"), "frontmatter"},
		{"duplicate known nested key", configDocument("project: {records: ../r, records: ../other}"), "frontmatter"},
		{"duplicate unknown nested key", configDocument("project: {records: ../r}\ncustom: {name: first, name: second}"), "frontmatter"},
		{"duplicate merge key", configDocument("defaults: &defaults {records: ../r}\nproject: {<<: *defaults, <<: *defaults}"), "frontmatter"},
		{"type wrong", "---\ntype: Project\nversion: 1\nproject: {records: ../r}\n---\n", "type"},
		{"type missing", "---\nversion: 1\nproject: {records: ../r}\n---\n", "type"},
		{"version missing", "---\ntype: ContextConfig\nproject: {records: ../r}\n---\n", "version"},
		{"version unsupported", strings.Replace(configDocument("project: {records: ../r}"), "version: 1", "version: 2", 1), "version"},
		{"version string", strings.Replace(configDocument("project: {records: ../r}"), "version: 1", "version: '1'", 1), "version"},
		{"version float", strings.Replace(configDocument("project: {records: ../r}"), "version: 1", "version: 1.0", 1), "version"},
		{"version bool", strings.Replace(configDocument("project: {records: ../r}"), "version: 1", "version: true", 1), "version"},
		{"invalid UTF-8", configDocument("project: {records: ../r}") + "\xff", "document"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.md")
			writeConfig(t, path, test.source)
			_, err := discovery.ReadConfig(path, discovery.SharedConfig)
			var diagnostic *discovery.ConfigError
			if !errors.As(err, &diagnostic) || diagnostic.Field != test.field {
				t.Fatalf("error = %v, want field %q", err, test.field)
			}
			if test.name == "version unsupported" && !strings.Contains(err.Error(), "unsupported version") {
				t.Fatalf("version error = %v", err)
			}
		})
	}
}

func TestConfigRejectsDuplicateDeclarationKeysAndAliases(t *testing.T) {
	tests := []struct {
		name    string
		profile discovery.ConfigProfile
		fields  string
		field   string
		other   string
	}{
		{"project keys", discovery.PersonalConfig, `projects:
  - {key: repeated, directory: ../one, records: ../r}
  - {key: repeated, directory: ../two, records: ../s}`, "projects[1].key", "projects[0].key"},
		{"project aliases", discovery.PersonalConfig, `projects:
  - {key: one, alias: repeated, directory: ../one, records: ../r}
  - {key: two, alias: repeated, directory: ../two, records: ../s}`, "projects[1].alias", "projects[0].alias"},
		{"authored empty aliases", discovery.PersonalConfig, `projects:
  - {key: one, alias: '', directory: ../one, records: ../r}
  - {key: two, alias: '', directory: ../two, records: ../s}`, "projects[1].alias", "projects[0].alias"},
		{"workspace keys", discovery.PersonalConfig, `workspaces:
  - {key: repeated, id: one, title: One, members: []}
  - {key: repeated, id: two, title: Two, members: []}`, "workspaces[1].key", "workspaces[0].key"},
		{"workspace aliases", discovery.PersonalConfig, `workspaces:
  - {key: one, alias: repeated, id: one, title: One, members: []}
  - {key: two, alias: repeated, id: two, title: Two, members: []}`, "workspaces[1].alias", "workspaces[0].alias"},
		{"member keys", discovery.SharedConfig, `workspace:
  id: w
  title: W
  members:
    - {key: repeated, records: ../r}
    - {key: repeated, records: ../s}`, "workspace.members[1].key", "workspace.members[0].key"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.md")
			writeConfig(t, path, configDocument(test.fields))
			_, err := discovery.ReadConfig(path, test.profile)
			var diagnostic *discovery.ConfigError
			if !errors.As(err, &diagnostic) || diagnostic.Field != test.field || !strings.Contains(err.Error(), test.other) {
				t.Fatalf("error = %v, want %s to reference %s", err, test.field, test.other)
			}
		})
	}
}

func TestConfigSymlinkKeepsEncounteredRelativeBase(t *testing.T) {
	root := physicalTempDir(t)
	path := filepath.Join(root, "checkout", ".context", "config.md")
	actual := filepath.Join(root, "elsewhere", "config.md")
	writeConfig(t, actual, configDocument("project: {records: ../records}"))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(actual, path); err != nil {
		t.Fatal(err)
	}
	config, err := discovery.ReadConfig(path, discovery.SharedConfig)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(root, "checkout", "records")
	if config.Path != path || config.Project.Records.Canonical != want {
		t.Fatalf("config=%q, records=%#v, want encountered location %q", config.Path, config.Project.Records, want)
	}
}

func TestSavedPathsStayLiteralAndPreserveSymlinkParentTraversal(t *testing.T) {
	root := physicalTempDir(t)
	path := filepath.Join(root, ".context", "config.md")
	physical := filepath.Join(root, "physical", "child")
	if err := os.MkdirAll(physical, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(physical, filepath.Join(root, "alias")); err != nil {
		t.Fatal(err)
	}
	writeConfig(t, path, configDocument("project:\n  records: ../alias/../records\n  allowSources: ['~/docs', '$HOME/docs', '$(touch surprise)']"))
	config, err := discovery.ReadConfig(path, discovery.SharedConfig)
	if err != nil {
		t.Fatal(err)
	}
	if got := config.Project.Records.Canonical; got != filepath.Join(root, "physical", "records") {
		t.Fatalf("symlink parent = %q", got)
	}
	for i, authored := range []string{"~/docs", "$HOME/docs", "$(touch surprise)"} {
		want := filepath.Join(filepath.Dir(path), authored)
		value := config.Project.AllowSources[i]
		if value.Authored != authored || value.Path != want || value.Canonical != want || value.Err != nil {
			t.Fatalf("literal path = %#v, want %q", value, want)
		}
	}
	absolute := filepath.Join(root, "separate-records")
	config, err = discovery.ParseConfig(path, []byte(configDocument("project: {records: '"+absolute+"'}")), discovery.SharedConfig)
	if err != nil || config.Project.Records.Path != absolute {
		t.Fatalf("absolute records = %#v, %v", config, err)
	}
}

func TestConfigDefersUnresolvableTargetIdentity(t *testing.T) {
	root := physicalTempDir(t)
	path := filepath.Join(root, ".context", "config.md")
	loop := filepath.Join(root, "loop")
	if err := os.Symlink(loop, loop); err != nil {
		t.Fatal(err)
	}
	writeConfig(t, path, configDocument(`projects:
  - key: broken
    directory: ../loop/checkout
    records: ../loop/records
  - key: usable
    directory: ../checkout
    records: ../records`))
	config, err := discovery.ReadConfig(path, discovery.PersonalConfig)
	if err != nil {
		t.Fatal(err)
	}
	if config.Projects[0].Directory.Err == nil || config.Projects[0].Records.Err == nil || config.Projects[1].Records.Err != nil {
		t.Fatalf("canonicalization failures = %#v", config.Projects)
	}
}

func TestReadConfigRetainsFileErrors(t *testing.T) {
	root := t.TempDir()
	missing := filepath.Join(root, "missing.md")
	if _, err := discovery.ReadConfig(missing, discovery.PersonalConfig); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing file error = %v", err)
	}
	if _, err := discovery.ReadConfig(root, discovery.PersonalConfig); err == nil {
		t.Fatal("directory was read as configuration")
	}
	link := filepath.Join(root, "dangling.md")
	if err := os.Symlink(missing, link); err != nil {
		t.Fatal(err)
	}
	if _, err := discovery.ReadConfig(link, discovery.PersonalConfig); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("dangling configuration error = %v", err)
	}
	if _, err := os.Lstat(link); err != nil {
		t.Fatalf("resolver cannot distinguish present symlink: %v", err)
	}
}

func TestParseConfigKeepsItsOwnSourceSnapshot(t *testing.T) {
	source := []byte(configDocument("project: {records: ../records}") + "Original body.\n")
	want := append([]byte(nil), source...)
	config, err := discovery.ParseConfig(filepath.Join(t.TempDir(), "config.md"), source, discovery.SharedConfig)
	if err != nil {
		t.Fatal(err)
	}
	source[len(source)-2] = '!'
	if !bytes.Equal(config.Raw, want) || string(config.Body) != "Original body.\n" {
		t.Fatal("caller mutation changed retained source bytes")
	}
}
