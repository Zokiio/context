package discovery_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/Zokiio/context/internal/discovery"
)

func setupRequest(t *testing.T) discovery.SetupRequest {
	t.Helper()
	root := physicalTempDir(t)
	home, directory := filepath.Join(root, "home"), filepath.Join(root, "checkout")
	for _, path := range []string{home, directory, filepath.Join(directory, "nested"), filepath.Join(root, "docs")} {
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	records := resolverBundle(t, filepath.Join(root, "records"), "setup-project")
	return discovery.SetupRequest{Cwd: filepath.Join(directory, "nested"), Home: home, Directory: "..", Records: records, AllowSources: []string{filepath.Join(root, "docs")}}
}

func setupPrepare(t *testing.T, request discovery.SetupRequest) *discovery.SetupPlan {
	t.Helper()
	plan, err := discovery.PrepareSetup(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	return plan
}

func setupRead(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestSetupSharedRebasesNestedInputsAndPreparationNeverWrites(t *testing.T) {
	request := setupRequest(t)
	request.Records = "../../records"
	request.AllowSources = []string{"../../docs", "../../docs/."}
	plan := setupPrepare(t, request)
	summary := plan.Summary()
	if summary.Change != discovery.SetupCreate || summary.Directory != filepath.Dir(request.Cwd) || len(summary.AllowSources) != 1 {
		t.Fatalf("summary = %#v", summary)
	}
	if _, err := os.Stat(filepath.Dir(summary.Destination)); !os.IsNotExist(err) {
		t.Fatalf("prepare created a directory: %v", err)
	}
	config, err := discovery.ParseConfig(summary.Destination, []byte(plan.Document()), discovery.SharedConfig)
	if err != nil || config.Project.Records.Authored != "../../records" || config.Project.AllowSources[0].Authored != "../../docs" {
		t.Fatalf("relative proposal = %#v, %v", config, err)
	}
	if err := plan.Apply(context.Background()); err != nil {
		t.Fatal(err)
	}
	if string(setupRead(t, summary.Destination)) != plan.Document() {
		t.Fatal("applied bytes differ from proposed document")
	}
	scope, err := discovery.Resolve(context.Background(), discovery.Request{Cwd: request.Cwd, Home: request.Home})
	if err != nil || scope.Project.Records != summary.Records || !reflect.DeepEqual(scope.Project.AllowSources, summary.AllowSources) {
		t.Fatalf("discovery after setup = %#v, %v", scope, err)
	}
}

func TestSetupSharedReplacementPreservesMetadataBodyAndPermissions(t *testing.T) {
	request := setupRequest(t)
	path := filepath.Join(filepath.Dir(request.Cwd), ".context", "config.md")
	old := resolverBundle(t, filepath.Join(filepath.Dir(request.Home), "old-records"), "old")
	body := "\r\n# My notes\r\n\r\nKeep **every** body byte.\r\n"
	writeConfig(t, path, configDocument("owner: {team: tooling, flags: [one, two]}\nproject:\n  records: "+old+"\n  custom: {retain: true}\nworkspace: {id: team, title: Team, members: []}")+body)
	if err := os.Chmod(path, 0o640); err != nil {
		t.Fatal(err)
	}
	before := setupRead(t, path)
	if _, err := discovery.PrepareSetup(context.Background(), request); err == nil || !strings.Contains(err.Error(), "--replace") {
		t.Fatalf("conflict = %v", err)
	}
	if !bytes.Equal(before, setupRead(t, path)) {
		t.Fatal("conflicting preparation changed configuration")
	}
	request.Replace = true
	plan := setupPrepare(t, request)
	if plan.Summary().Change != discovery.SetupUpdate || plan.Summary().PreviousRecords != old {
		t.Fatalf("replacement summary = %#v", plan.Summary())
	}
	if err := plan.Apply(context.Background()); err != nil {
		t.Fatal(err)
	}
	config, err := discovery.ReadConfig(path, discovery.SharedConfig)
	if err != nil {
		t.Fatal(err)
	}
	if string(config.Body) != body || config.Workspace.ID != "team" || config.Metadata["owner"].(map[string]any)["team"] != "tooling" || config.Metadata["project"].(map[string]any)["custom"].(map[string]any)["retain"] != true {
		t.Fatalf("unrelated content lost: %#v", config)
	}
	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0o640 {
		t.Fatalf("permissions = %v", info.Mode())
	}
}

func TestSetupIdenticalBindingKeepsOriginalBytesAndCreatesNoLock(t *testing.T) {
	request := setupRequest(t)
	path := resolverShared(t, filepath.Dir(request.Cwd), request.Records, request.AllowSources...)
	before, _ := os.Stat(path)
	document := setupRead(t, path)
	plan := setupPrepare(t, request)
	if plan.Summary().Change != discovery.SetupUnchanged || plan.Document() != string(document) {
		t.Fatalf("no-op proposal = %#v", plan.Summary())
	}
	if err := plan.Apply(context.Background()); err != nil {
		t.Fatal(err)
	}
	after, _ := os.Stat(path)
	if !os.SameFile(before, after) || !before.ModTime().Equal(after.ModTime()) || !bytes.Equal(document, setupRead(t, path)) {
		t.Fatal("no-op rewrote configuration")
	}
	if _, err := os.Stat(path + ".lock"); !os.IsNotExist(err) {
		t.Fatalf("no-op created lock: %v", err)
	}
	if _, err := os.Stat(filepath.Join(request.Home, ".context")); !os.IsNotExist(err) {
		t.Fatalf("no-op changed personal configuration directory: %v", err)
	}
}

func TestSetupPersonalIdentityAliasesAndRootRemoval(t *testing.T) {
	request := setupRequest(t)
	request.Personal, request.Alias = true, "mine"
	plan := setupPrepare(t, request)
	if err := plan.Apply(context.Background()); err != nil {
		t.Fatal(err)
	}
	path := plan.Summary().Destination
	config, err := discovery.ReadConfig(path, discovery.PersonalConfig)
	if err != nil {
		t.Fatal(err)
	}
	entry := config.Projects[0]
	if !regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`).MatchString(entry.Key) || entry.Directory.Authored != filepath.Dir(request.Cwd) || entry.Records.Authored != request.Records || entry.Alias != "mine" {
		t.Fatalf("personal entry = %#v", entry)
	}
	request.Alias, request.AllowSources, request.Replace = "", nil, true
	plan = setupPrepare(t, request)
	if len(plan.Summary().RemovedSources) != 1 || plan.Summary().Alias != "mine" {
		t.Fatalf("removal summary = %#v", plan.Summary())
	}
	if err := plan.Apply(context.Background()); err != nil {
		t.Fatal(err)
	}
	config, _ = discovery.ReadConfig(path, discovery.PersonalConfig)
	if config.Projects[0].Key != entry.Key || config.Projects[0].Alias != "mine" || len(config.Projects[0].AllowSources) != 0 {
		t.Fatalf("replacement did not retain identity: %#v", config.Projects[0])
	}
	request.Alias = "renamed"
	plan = setupPrepare(t, request)
	if plan.Summary().PreviousAlias != "mine" {
		t.Fatalf("alias removal missing: %#v", plan.Summary())
	}
	if err := plan.Apply(context.Background()); err != nil {
		t.Fatal(err)
	}
	scope, err := discovery.Resolve(context.Background(), discovery.Request{Cwd: request.Cwd, Home: request.Home, Kind: discovery.Project, Selector: "@renamed"})
	if err != nil || scope.Project.Records != request.Records {
		t.Fatalf("alias selection = %#v, %v", scope, err)
	}
}

func TestSetupPersonalKeepsUnrelatedEntriesAndMatchesPhysicalDirectory(t *testing.T) {
	request := setupRequest(t)
	directory := filepath.Dir(request.Cwd)
	symlink := filepath.Join(filepath.Dir(request.Home), "checkout-link")
	if err := os.Symlink(directory, symlink); err != nil {
		t.Fatal(err)
	}
	path := resolverPersonal(t, request.Home, "owner: {keep: yes}\nprojects:\n"+resolverProjectEntry("retained-key", "mine", symlink, request.Records)+"    custom: {keep: true}\n"+resolverProjectEntry("other", "other-alias", "/missing-checkout", "/missing-records")+"workspaces: [{key: work, id: work, title: Work, members: []}]")
	request.Personal, request.Replace = true, true
	plan := setupPrepare(t, request)
	if err := plan.Apply(context.Background()); err != nil {
		t.Fatal(err)
	}
	config, err := discovery.ReadConfig(path, discovery.PersonalConfig)
	if err != nil {
		t.Fatal(err)
	}
	if len(config.Projects) != 2 || config.Projects[0].Key != "retained-key" || config.Projects[0].Alias != "mine" || config.Projects[1].Records.Authored != "/missing-records" || config.Workspaces[0].ID != "work" || config.Metadata["projects"].([]any)[0].(map[string]any)["custom"].(map[string]any)["keep"] != true {
		t.Fatalf("personal preservation = %#v", config)
	}
}

func TestSetupRejectsAmbiguousDirectoryAndAliasCollision(t *testing.T) {
	for _, duplicateDirectory := range []bool{false, true} {
		t.Run(map[bool]string{false: "alias", true: "directory"}[duplicateDirectory], func(t *testing.T) {
			request := setupRequest(t)
			directory, other := filepath.Dir(request.Cwd), request.Home
			if duplicateDirectory {
				other = directory
			}
			path := resolverPersonal(t, request.Home, "projects:\n"+resolverProjectEntry("one", "one", directory, request.Records)+resolverProjectEntry("two", "taken", other, request.Records))
			before := setupRead(t, path)
			request.Personal, request.Replace, request.Alias = true, true, "taken"
			if _, err := discovery.PrepareSetup(context.Background(), request); err == nil {
				t.Fatal("ambiguous registration accepted")
			}
			if !bytes.Equal(before, setupRead(t, path)) {
				t.Fatal("rejected request changed registry")
			}
		})
	}
}

func TestSetupReplaceCannotHideOtherFileConflict(t *testing.T) {
	for _, personal := range []bool{false, true} {
		t.Run(map[bool]string{false: "shared", true: "personal"}[personal], func(t *testing.T) {
			request := setupRequest(t)
			directory := filepath.Dir(request.Cwd)
			shared := resolverShared(t, directory, request.Records)
			registry := resolverPersonal(t, request.Home, "projects:\n"+resolverProjectEntry("one", "one", directory, request.Records))
			request.Personal, request.Replace = personal, true
			_, err := discovery.PrepareSetup(context.Background(), request)
			if err == nil || !strings.Contains(err.Error(), shared) || !strings.Contains(err.Error(), registry) {
				t.Fatalf("other-file conflict = %v", err)
			}
		})
	}
}

func TestSetupRejectsInvalidInputsAndMalformedConfiguration(t *testing.T) {
	for _, scenario := range []string{"manifestless", "missing-directory", "missing-root", "unclean-records", "unclean-root", "alias-without-personal", "empty-alias", "reserved-home", "malformed-config"} {
		t.Run(scenario, func(t *testing.T) {
			request := setupRequest(t)
			switch scenario {
			case "manifestless":
				request.Records = request.Home
			case "missing-directory":
				request.Directory = "absent"
			case "missing-root":
				request.AllowSources = []string{"absent"}
			case "unclean-records":
				request.Records += "/missing/.."
			case "unclean-root":
				request.AllowSources[0] += "/missing/.."
			case "alias-without-personal":
				request.Alias = "mine"
			case "empty-alias":
				request.Personal, request.AliasSet = true, true
			case "reserved-home":
				request.Directory = request.Home
			case "malformed-config":
				request.Replace = true
				writeConfig(t, filepath.Join(filepath.Dir(request.Cwd), ".context", "config.md"), "---\ntype: ContextConfig\nversion: [invalid]\n---\n")
			}
			if _, err := discovery.PrepareSetup(context.Background(), request); err == nil {
				t.Fatal("invalid setup accepted")
			}
		})
	}
}

func TestSetupSymlinkedConfigKeepsEncounteredBaseAndLink(t *testing.T) {
	request := setupRequest(t)
	path := filepath.Join(filepath.Dir(request.Cwd), ".context", "config.md")
	target := filepath.Join(filepath.Dir(request.Home), "elsewhere", "settings.md")
	writeConfig(t, target, configDocument("project: {records: ../../records}"))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, path); err != nil {
		t.Fatal(err)
	}
	request.Replace = true
	plan := setupPrepare(t, request)
	if err := plan.Apply(context.Background()); err != nil {
		t.Fatal(err)
	}
	info, _ := os.Lstat(path)
	if info.Mode()&os.ModeSymlink == 0 || !bytes.Equal(setupRead(t, path), setupRead(t, target)) {
		t.Fatal("setup replaced the symlink instead of its file target")
	}
	if _, err := os.Stat(target + ".lock"); err != nil {
		t.Fatalf("physical lock missing: %v", err)
	}
	config, err := discovery.ReadConfig(path, discovery.SharedConfig)
	if err != nil || config.Project.Records.Canonical != request.Records {
		t.Fatalf("relative path was rebased to symlink target: %#v, %v", config, err)
	}
}
