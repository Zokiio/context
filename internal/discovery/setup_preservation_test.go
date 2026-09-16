package discovery_test

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Zokiio/context/internal/discovery"
)

func TestSetupPreservesUnrelatedSharedAliasValues(t *testing.T) {
	for _, style := range []string{"mapping anchor", "mapping alias", "field anchor", "merged fields", "merged project"} {
		t.Run(style, func(t *testing.T) {
			request := setupRequest(t)
			request.Replace = true
			old := resolverBundle(t, filepath.Join(filepath.Dir(request.Home), "old-records"), "old")
			path := filepath.Join(filepath.Dir(request.Cwd), ".context", "config.md")
			var fields string
			switch style {
			case "mapping anchor":
				fields = fmt.Sprintf("project: &binding {key: member, records: %q}\ncustom: *binding\nworkspace: {id: team, title: Team, members: [*binding]}", old)
			case "mapping alias":
				fields = fmt.Sprintf("custom: &binding {records: %q}\nproject: *binding", old)
			case "field anchor":
				fields = fmt.Sprintf("project: {records: &records %q}\ncustom: *records", old)
			case "merged fields":
				fields = fmt.Sprintf("custom: &binding {records: %q, owner: !!timestamp '2026-09-17T12:00:00Z'}\nproject: {<<: *binding}", old)
			case "merged project":
				fields = fmt.Sprintf("custom: &defaults {project: {records: %q, owner: !!timestamp '2026-09-17T12:00:00Z'}}\n<<: *defaults", old)
			}
			writeConfig(t, path, configDocument(fields))
			before, err := discovery.ReadConfig(path, discovery.SharedConfig)
			if err != nil {
				t.Fatal(err)
			}
			plan := setupPrepare(t, request)
			if err := plan.Apply(context.Background()); err != nil {
				t.Fatal(err)
			}
			after, err := discovery.ReadConfig(path, discovery.SharedConfig)
			if err != nil {
				t.Fatal(err)
			}
			if after.Project.Records.Canonical != request.Records {
				t.Fatalf("selected binding = %q", after.Project.Records.Canonical)
			}
			if !reflect.DeepEqual(before.Metadata["custom"], after.Metadata["custom"]) {
				t.Fatalf("unrelated alias changed: before=%#v, after=%#v", before.Metadata["custom"], after.Metadata["custom"])
			}
			if before.Workspace != nil && after.Workspace.Members[0].Records.Canonical != old {
				t.Fatalf("unrelated workspace member was retargeted: %#v", after.Workspace.Members[0])
			}
		})
	}
}

func TestSetupPreservesAliasedPersonalEntriesAndArrays(t *testing.T) {
	for _, arrayAlias := range []bool{false, true} {
		t.Run(fmt.Sprintf("array=%t", arrayAlias), func(t *testing.T) {
			request := setupRequest(t)
			request.Personal, request.Replace = true, true
			old := resolverBundle(t, filepath.Join(filepath.Dir(request.Home), "old-records"), "old")
			entry := fmt.Sprintf("{key: retained, alias: mine, directory: %q, records: %q, custom: !!binary SGVsbG8=}", filepath.Dir(request.Cwd), old)
			fields := "projects: [ &saved " + entry + " ]\ncustom: *saved"
			if arrayAlias {
				fields = "projects: &saved [ " + entry + " ]\ncustom: *saved"
			}
			path := resolverPersonal(t, request.Home, fields)
			before, err := discovery.ReadConfig(path, discovery.PersonalConfig)
			if err != nil {
				t.Fatal(err)
			}
			plan := setupPrepare(t, request)
			if err := plan.Apply(context.Background()); err != nil {
				t.Fatal(err)
			}
			after, err := discovery.ReadConfig(path, discovery.PersonalConfig)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(before.Metadata["custom"], after.Metadata["custom"]) {
				t.Fatalf("personal alias changed: before=%#v, after=%#v", before.Metadata["custom"], after.Metadata["custom"])
			}
			if after.Projects[0].Key != "retained" || after.Projects[0].Alias != "mine" || after.Projects[0].Records.Canonical != request.Records {
				t.Fatalf("selected personal entry = %#v", after.Projects[0])
			}
		})
	}
}

func TestSetupPreservesUnknownTagsAndBody(t *testing.T) {
	request := setupRequest(t)
	request.Replace = true
	old := resolverBundle(t, filepath.Join(filepath.Dir(request.Home), "old-records"), "old")
	path := filepath.Join(filepath.Dir(request.Cwd), ".context", "config.md")
	body := "\r\nKeep **every** body byte.\r\n"
	writeConfig(t, path, configDocument(fmt.Sprintf(`# Keep this configuration comment.
binary: !!binary SGVsbG8=
timestamp: !!timestamp 2026-09-17T12:00:00Z
custom: !colour green
special: .nan
project: {records: %q, owner: !!timestamp '2026-09-17T12:00:00Z'}`, old))+body)
	before, err := discovery.ReadConfig(path, discovery.SharedConfig)
	if err != nil {
		t.Fatal(err)
	}
	plan := setupPrepare(t, request)
	if err := plan.Apply(context.Background()); err != nil {
		t.Fatal(err)
	}
	after, err := discovery.ReadConfig(path, discovery.SharedConfig)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"binary", "timestamp", "custom"} {
		if !reflect.DeepEqual(before.Metadata[key], after.Metadata[key]) {
			t.Fatalf("%s type/value changed: before=%T %#v, after=%T %#v", key, before.Metadata[key], before.Metadata[key], after.Metadata[key], after.Metadata[key])
		}
	}
	if !reflect.DeepEqual(before.Metadata["project"].(map[string]any)["owner"], after.Metadata["project"].(map[string]any)["owner"]) || string(after.Body) != body {
		t.Fatal("nested tagged metadata or Markdown body changed")
	}
	if value, ok := after.Metadata["special"].(float64); !ok || !math.IsNaN(value) {
		t.Fatalf("YAML NaN changed: %#v", after.Metadata["special"])
	}
	for _, retained := range []string{"!!binary", "!!timestamp", "!colour", "# Keep this configuration comment."} {
		if !strings.Contains(string(after.Raw), retained) {
			t.Fatalf("original YAML annotation %q was lost", retained)
		}
	}
}

func TestSetupPreservesTaggedPersonalProjects(t *testing.T) {
	for _, replace := range []bool{false, true} {
		t.Run(fmt.Sprintf("replace=%t", replace), func(t *testing.T) {
			request := setupRequest(t)
			request.Personal, request.Replace = true, replace
			projects := "[]"
			if replace {
				old := resolverBundle(t, filepath.Join(filepath.Dir(request.Home), "old-records"), "old")
				projects = fmt.Sprintf("[!!map {key: retained, directory: %q, records: %q}]", filepath.Dir(request.Cwd), old)
			}
			path := resolverPersonal(t, request.Home, "projects: !!seq "+projects)
			plan := setupPrepare(t, request)
			if err := plan.Apply(context.Background()); err != nil {
				t.Fatal(err)
			}
			after, err := discovery.ReadConfig(path, discovery.PersonalConfig)
			if err != nil {
				t.Fatal(err)
			}
			if len(after.Projects) != 1 || after.Projects[0].Records.Canonical != request.Records || !strings.Contains(plan.Document(), "!!seq") {
				t.Fatalf("tagged projects = %#v\n%s", after.Projects, plan.Document())
			}
			if replace && (after.Projects[0].Key != "retained" || !strings.Contains(plan.Document(), "!!map")) {
				t.Fatalf("tagged entry changed identity or lost its tag:\n%s", plan.Document())
			}
		})
	}
}

func TestSetupPreservesAliasDeclarationOrderAndScalarValues(t *testing.T) {
	for _, fields := range []string{
		"first: &value one\ncustom: {before: *value}\nsecond: &value two\nother: *value",
		"empty: &empty\ncustom: *empty",
		"empty: &empty null\ncustom: *empty",
		"key: &field colour\ncustom:\n  *field : green",
	} {
		t.Run(fields, func(t *testing.T) {
			request := setupRequest(t)
			request.Replace = true
			old := resolverBundle(t, filepath.Join(filepath.Dir(request.Home), "old-records"), "old")
			path := filepath.Join(filepath.Dir(request.Cwd), ".context", "config.md")
			writeConfig(t, path, configDocument(fields+fmt.Sprintf("\nproject: {records: %q}", old)))
			before, err := discovery.ReadConfig(path, discovery.SharedConfig)
			if err != nil {
				t.Fatal(err)
			}
			plan := setupPrepare(t, request)
			if err := plan.Apply(context.Background()); err != nil {
				t.Fatal(err)
			}
			after, err := discovery.ReadConfig(path, discovery.SharedConfig)
			if err != nil {
				t.Fatal(err)
			}
			delete(before.Metadata, "project")
			delete(after.Metadata, "project")
			if !reflect.DeepEqual(before.Metadata, after.Metadata) {
				t.Fatalf("alias values changed: before=%#v, after=%#v", before.Metadata, after.Metadata)
			}
		})
	}
}

func TestSetupSelectedFieldsOverrideExistingMerges(t *testing.T) {
	for _, mergeLast := range []bool{false, true} {
		t.Run(fmt.Sprintf("mergeLast=%t", mergeLast), func(t *testing.T) {
			request := setupRequest(t)
			request.Replace = true
			old := resolverBundle(t, filepath.Join(filepath.Dir(request.Home), "old-records"), "old")
			project := fmt.Sprintf("{<<: *base, records: %q}", old)
			if mergeLast {
				project = fmt.Sprintf("{records: %q, <<: *base}", old)
			}
			fields := fmt.Sprintf("base: &base {records: %q, owner: keep}\nproject: %s", old, project)
			path := filepath.Join(filepath.Dir(request.Cwd), ".context", "config.md")
			writeConfig(t, path, configDocument(fields))
			plan := setupPrepare(t, request)
			if err := plan.Apply(context.Background()); err != nil {
				t.Fatal(err)
			}
			after, err := discovery.ReadConfig(path, discovery.SharedConfig)
			if err != nil {
				t.Fatal(err)
			}
			if after.Project.Records.Canonical != request.Records || after.Metadata["project"].(map[string]any)["owner"] != "keep" || after.Metadata["base"].(map[string]any)["records"] != old {
				t.Fatalf("merged project or unrelated source changed: %#v\n%s", after.Metadata, plan.Document())
			}
		})
	}
}

func TestSetupLeavesRecursiveMetadataUnchangedOrRefusesRewrite(t *testing.T) {
	request := setupRequest(t)
	request.AllowSources = nil
	path := filepath.Join(filepath.Dir(request.Cwd), ".context", "config.md")
	writeConfig(t, path, configDocument(fmt.Sprintf("custom: &loop {self: *loop}\nproject: {records: %q}", request.Records)))
	before := setupRead(t, path)
	plan := setupPrepare(t, request)
	if plan.Summary().Change != discovery.SetupUnchanged || plan.Document() != string(before) {
		t.Fatal("no-op did not retain recursive metadata verbatim")
	}
	if err := plan.Apply(context.Background()); err != nil {
		t.Fatal(err)
	}
	request.Replace, request.AllowSources = true, []string{request.Cwd}
	if _, err := discovery.PrepareSetup(context.Background(), request); err == nil || !strings.Contains(err.Error(), "recursive") {
		t.Fatalf("rewrite should refuse recursive aliases before changing their meaning: %v", err)
	}
	if !bytes.Equal(before, setupRead(t, path)) {
		t.Fatal("refused rewrite changed original file")
	}
}
