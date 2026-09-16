package discovery_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Zokiio/context/internal/discovery"
)

type resolverFixture struct {
	root string
	cwd  string
	home string
}

func newResolverFixture(t *testing.T) resolverFixture {
	t.Helper()
	root := physicalTempDir(t)
	fixture := resolverFixture{root: root, cwd: filepath.Join(root, "checkout"), home: filepath.Join(root, "home")}
	for _, directory := range []string{fixture.cwd, fixture.home} {
		if err := os.MkdirAll(directory, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return fixture
}

func (f resolverFixture) request(kind discovery.Kind, selector string) discovery.Request {
	return discovery.Request{Cwd: f.cwd, Home: f.home, Kind: kind, Selector: selector}
}

func resolverBundle(t *testing.T, directory, id string) string {
	t.Helper()
	writeConfig(t, filepath.Join(directory, "project.md"), fmt.Sprintf("---\ntype: Project\nid: %q\ntitle: %q\n---\n", id, "Project "+id))
	return directory
}

func resolverRoots(roots []string) string {
	values := make([]string, len(roots))
	for i, root := range roots {
		values[i] = fmt.Sprintf("%q", root)
	}
	return "[" + strings.Join(values, ", ") + "]"
}

func resolverShared(t *testing.T, directory, records string, roots ...string) string {
	t.Helper()
	path := filepath.Join(directory, ".context", "config.md")
	writeConfig(t, path, configDocument(fmt.Sprintf("project:\n  records: %q\n  allowSources: %s", records, resolverRoots(roots))))
	return path
}

func resolverProjectEntry(key, alias, directory, records string, roots ...string) string {
	entry := fmt.Sprintf("  - key: %q\n    directory: %q\n    records: %q\n    allowSources: %s\n", key, directory, records, resolverRoots(roots))
	if alias != "" {
		entry += fmt.Sprintf("    alias: %q\n", alias)
	}
	return entry
}

func resolverPersonal(t *testing.T, home, fields string) string {
	t.Helper()
	path := filepath.Join(home, ".context", "config.md")
	writeConfig(t, path, configDocument(fields))
	return path
}

func TestResolveNearestDirectoryAcrossStorageLocations(t *testing.T) {
	for _, personalNearer := range []bool{true, false} {
		t.Run(fmt.Sprintf("personal-nearer=%t", personalNearer), func(t *testing.T) {
			f := newResolverFixture(t)
			nested := filepath.Join(f.cwd, "nested")
			start := filepath.Join(nested, "source", "deep")
			if err := os.MkdirAll(start, 0o755); err != nil {
				t.Fatal(err)
			}
			broadRecords := resolverBundle(t, filepath.Join(f.root, "broad-records"), "broad")
			narrowRecords := resolverBundle(t, filepath.Join(f.root, "narrow-records"), "narrow")
			if personalNearer {
				resolverShared(t, f.cwd, broadRecords)
				resolverPersonal(t, f.home, "projects:\n"+resolverProjectEntry("near", "", nested, narrowRecords))
			} else {
				resolverShared(t, nested, narrowRecords)
				resolverPersonal(t, f.home, "projects:\n"+resolverProjectEntry("broad", "", f.cwd, broadRecords))
			}
			request := f.request(discovery.Any, "")
			request.Cwd = start
			scope, err := discovery.Resolve(context.Background(), request)
			if err != nil || scope.Kind != discovery.Project || scope.Project.Records != narrowRecords || scope.BindingDirectory != nested || scope.StartDirectory != start {
				t.Fatalf("scope = %#v, %v", scope, err)
			}
			// An explicit project path uses exactly the same ancestor discovery.
			request.Cwd, request.Selector, request.Kind = f.home, start, discovery.Project
			explicit, err := discovery.Resolve(context.Background(), request)
			if err != nil || explicit.Project.Records != scope.Project.Records || explicit.BindingDirectory != scope.BindingDirectory {
				t.Fatalf("explicit scope = %#v, %v", explicit, err)
			}
		})
	}
}

func TestResolveWorkspaceStopsImplicitProjectFallback(t *testing.T) {
	f := newResolverFixture(t)
	records := resolverBundle(t, filepath.Join(f.root, "records"), "broad")
	resolverShared(t, f.cwd, records)
	nested := filepath.Join(f.cwd, "team")
	writeConfig(t, filepath.Join(nested, ".context", "config.md"), configDocument("workspace: {id: team, title: Team, members: []}"))
	start := filepath.Join(nested, "source")
	if err := os.Mkdir(start, 0o755); err != nil {
		t.Fatal(err)
	}
	request := f.request(discovery.Any, start)
	scope, err := discovery.Resolve(context.Background(), request)
	if err != nil || scope.Kind != discovery.Workspace || scope.Workspace.ID != "team" || scope.Project != nil {
		t.Fatalf("implicit scope = %#v, %v", scope, err)
	}
	request.Kind = discovery.Project
	scope, err = discovery.Resolve(context.Background(), request)
	if err != nil || scope.Kind != discovery.Project || scope.Project.Records != records {
		t.Fatalf("explicit project scope = %#v, %v", scope, err)
	}
}

func TestResolveProjectWinsAtEqualDepthBeforeWorkspaceConflicts(t *testing.T) {
	f := newResolverFixture(t)
	records := resolverBundle(t, filepath.Join(f.root, "records"), "selected")
	shared := filepath.Join(f.cwd, ".context", "config.md")
	writeConfig(t, shared, configDocument(fmt.Sprintf("project: {records: %q}\nworkspace: {id: local, title: Local, members: []}", records)))
	resolverPersonal(t, f.home, fmt.Sprintf("workspaces:\n  - key: other\n    directory: %q\n    id: personal\n    title: Personal\n    members: []", f.cwd))
	scope, err := discovery.Resolve(context.Background(), f.request(discovery.Any, ""))
	if err != nil || scope.Kind != discovery.Project || scope.Project.Records != records {
		t.Fatalf("same-depth project = %#v, %v", scope, err)
	}
	if _, err := discovery.Resolve(context.Background(), f.request(discovery.Workspace, "")); err == nil || !strings.Contains(err.Error(), "conflicting workspace") {
		t.Fatalf("explicit workspace conflict = %v", err)
	}
}

func TestResolveProjectConflictIncludesBothDeclarations(t *testing.T) {
	for _, differentRecords := range []bool{true, false} {
		t.Run(fmt.Sprintf("different-records=%t", differentRecords), func(t *testing.T) {
			f := newResolverFixture(t)
			first := resolverBundle(t, filepath.Join(f.root, "records-one"), "same-id")
			second := first
			if differentRecords {
				second = resolverBundle(t, filepath.Join(f.root, "records-two"), "same-id")
			}
			local := resolverShared(t, f.cwd, first, f.cwd)
			personal := resolverPersonal(t, f.home, "projects:\n"+resolverProjectEntry("conflict", "", f.cwd, second, f.home))
			_, err := discovery.Resolve(context.Background(), f.request(discovery.Any, ""))
			for _, detail := range []string{"conflicting project", f.cwd, local, personal, "records", "allowSources", "repair"} {
				if err == nil || !strings.Contains(err.Error(), detail) {
					t.Fatalf("error = %v, missing %q", err, detail)
				}
			}
		})
	}
}

func TestResolveCoalescesEffectiveAllowedSourceSets(t *testing.T) {
	f := newResolverFixture(t)
	records := resolverBundle(t, filepath.Join(f.root, "records"), "one")
	rootAlias := filepath.Join(f.root, "home-alias")
	if err := os.Symlink(f.home, rootAlias); err != nil {
		t.Fatal(err)
	}
	local := resolverShared(t, f.cwd, records, f.cwd, f.home)
	personal := resolverPersonal(t, f.home, "projects:\n"+resolverProjectEntry("same", "", f.cwd, records, rootAlias, f.cwd, f.cwd))
	scope, err := discovery.Resolve(context.Background(), f.request(discovery.Any, ""))
	if err != nil {
		t.Fatal(err)
	}
	if scope.Project.Records != records || len(scope.Project.AllowSources) != 2 || len(scope.Origins) != 2 {
		t.Fatalf("scope = %#v, project = %#v", scope, scope.Project)
	}
	found := map[string]bool{}
	for _, origin := range scope.Origins {
		found[origin.Path] = true
	}
	if !found[local] || !found[personal] {
		t.Fatalf("origins = %#v", scope.Origins)
	}
}

func TestResolveMarkerAndAgreeingMappingSupplyAllowedSources(t *testing.T) {
	for _, personal := range []bool{true, false} {
		t.Run(fmt.Sprintf("personal=%t", personal), func(t *testing.T) {
			f := newResolverFixture(t)
			resolverBundle(t, f.cwd, "embedded")
			if personal {
				resolverPersonal(t, f.home, "projects:\n"+resolverProjectEntry("same", "", f.cwd, f.cwd, f.home))
			} else {
				resolverShared(t, f.cwd, f.cwd, f.home)
			}
			scope, err := discovery.Resolve(context.Background(), f.request(discovery.Any, ""))
			if err != nil || len(scope.Origins) != 2 || len(scope.Project.AllowSources) != 1 || scope.Project.AllowSources[0] != f.home {
				t.Fatalf("scope = %#v, %v", scope, err)
			}
		})
	}
}

func TestResolveMarkerDoesNotHideAuthoredRootsConflict(t *testing.T) {
	f := newResolverFixture(t)
	resolverBundle(t, f.cwd, "embedded")
	resolverShared(t, f.cwd, f.cwd, f.cwd)
	resolverPersonal(t, f.home, "projects:\n"+resolverProjectEntry("different-roots", "", f.cwd, f.cwd, f.home))
	_, err := discovery.Resolve(context.Background(), f.request(discovery.Any, ""))
	if err == nil || !strings.Contains(err.Error(), "conflicting project") {
		t.Fatalf("error = %v", err)
	}
}

func TestResolveBrokenSelectedMappingNeverFallsBack(t *testing.T) {
	tests := []string{"missing-records", "missing-root", "manifestless-records", "invalid-manifest", "unusable-coalesced-path"}
	for _, name := range tests {
		t.Run(name, func(t *testing.T) {
			f := newResolverFixture(t)
			broad := resolverBundle(t, filepath.Join(f.root, "broad"), "broad")
			resolverShared(t, f.cwd, broad)
			nested := filepath.Join(f.cwd, "nested")
			records := resolverBundle(t, filepath.Join(f.root, "selected"), "selected")
			roots := []string{}
			switch name {
			case "missing-records":
				records = filepath.Join(f.root, "missing")
			case "missing-root":
				roots = []string{filepath.Join(f.root, "missing")}
			case "manifestless-records":
				if err := os.Remove(filepath.Join(records, "project.md")); err != nil {
					t.Fatal(err)
				}
			case "invalid-manifest":
				writeConfig(t, filepath.Join(records, "project.md"), "---\ntype: WorkItem\nid: wrong\ntitle: Wrong\n---\n")
			case "unusable-coalesced-path":
				resolverPersonal(t, f.home, "projects:\n"+resolverProjectEntry("bad-path", "", nested, f.root+"/missing/../selected"))
			}
			local := resolverShared(t, nested, records, roots...)
			_, err := discovery.Resolve(context.Background(), f.request(discovery.Any, nested))
			if err == nil || !strings.Contains(err.Error(), "repair") {
				t.Fatalf("broken selected declaration at %s fell back: %v", local, err)
			}
			if !strings.Contains(err.Error(), "field") || !strings.Contains(err.Error(), "resolves to") {
				t.Fatalf("selected-target error lacks provenance: %v", err)
			}
		})
	}
}

func TestResolveAliasesIgnoreCheckoutAndCwdAvailability(t *testing.T) {
	f := newResolverFixture(t)
	records := resolverBundle(t, filepath.Join(f.root, "records"), "saved")
	checkout := filepath.Join(f.root, "missing-checkout")
	personal := resolverPersonal(t, f.home, "projects:\n"+resolverProjectEntry("saved", "work", checkout, records, f.home)+
		"workspaces:\n  - key: saved-workspace\n    alias: work\n    id: saved-workspace\n    title: Saved workspace\n    members: []")
	request := f.request(discovery.Project, "@work")
	request.Cwd = filepath.Join(f.root, "missing-cwd")
	scope, err := discovery.Resolve(context.Background(), request)
	if err != nil || scope.Kind != discovery.Project || scope.Project.Records != records || len(scope.Origins) != 1 || scope.Origins[0].Path != personal {
		t.Fatalf("project alias = %#v, %v", scope, err)
	}
	request.Kind = discovery.Workspace
	scope, err = discovery.Resolve(context.Background(), request)
	if err != nil || scope.Kind != discovery.Workspace || scope.Workspace.ID != "saved-workspace" || scope.BindingDirectory != "" {
		t.Fatalf("workspace alias = %#v, %v", scope, err)
	}
	// A cwd mapping cannot replace the concrete saved alias.
	resolverShared(t, f.cwd, filepath.Join(f.root, "broken-records"))
	request.Kind, request.Cwd = discovery.Project, f.cwd
	if _, err := discovery.Resolve(context.Background(), request); err != nil {
		t.Fatalf("alias depended on cwd configuration: %v", err)
	}
}

func TestResolveUnknownAndInvalidAliasFail(t *testing.T) {
	for _, selector := range []string{"@unknown", "@"} {
		t.Run(selector, func(t *testing.T) {
			f := newResolverFixture(t)
			resolverBundle(t, f.cwd, "cwd")
			_, err := discovery.Resolve(context.Background(), f.request(discovery.Project, selector))
			if err == nil || !strings.Contains(err.Error(), "alias") {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestResolveLiteralAtDirectoryIsCwdRelative(t *testing.T) {
	f := newResolverFixture(t)
	directory := resolverBundle(t, filepath.Join(f.cwd, "@literal"), "literal")
	scope, err := discovery.Resolve(context.Background(), f.request(discovery.Project, "./@literal"))
	if err != nil || scope.Project.Records != directory || scope.StartDirectory != directory {
		t.Fatalf("literal selector = %#v, %v", scope, err)
	}
}

func TestResolveRegistryStructureAlwaysValidated(t *testing.T) {
	for _, broken := range []string{"malformed", "mixed-profile", "duplicate-alias", "dangling-file", "dangling-directory"} {
		t.Run(broken, func(t *testing.T) {
			f := newResolverFixture(t)
			resolverBundle(t, f.cwd, "usable")
			path := filepath.Join(f.home, ".context", "config.md")
			switch broken {
			case "malformed":
				writeConfig(t, path, "---\nprojects: [broken\n---\n")
			case "mixed-profile":
				writeConfig(t, path, configDocument("project: {records: ../records}"))
			case "duplicate-alias":
				resolverPersonal(t, f.home, "projects:\n"+resolverProjectEntry("one", "same", f.cwd, f.cwd)+resolverProjectEntry("two", "same", f.home, f.cwd))
			case "dangling-file":
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(filepath.Join(f.root, "missing"), path); err != nil {
					t.Fatal(err)
				}
			case "dangling-directory":
				if err := os.Symlink(filepath.Join(f.root, "missing"), filepath.Dir(path)); err != nil {
					t.Fatal(err)
				}
			}
			_, err := discovery.Resolve(context.Background(), f.request(discovery.Any, ""))
			if err == nil || !strings.Contains(err.Error(), path) || !strings.Contains(err.Error(), f.cwd) {
				t.Fatalf("registry failure = %v", err)
			}
		})
	}
}

func TestResolveUnreadableRegistryFailsEvenWithLocalProject(t *testing.T) {
	f := newResolverFixture(t)
	resolverBundle(t, f.cwd, "usable")
	path := resolverPersonal(t, f.home, "projects: []")
	if err := os.Chmod(path, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(path, 0o644) })
	if _, err := os.ReadFile(path); err == nil {
		t.Skip("filesystem does not enforce file read permissions for this user")
	}
	_, err := discovery.Resolve(context.Background(), f.request(discovery.Any, ""))
	if err == nil || !strings.Contains(err.Error(), path) || !strings.Contains(err.Error(), "repair") {
		t.Fatalf("unreadable registry was ignored: %v", err)
	}
}

func TestResolveMalformedLocalOrMarkerCannotFallThrough(t *testing.T) {
	for _, broken := range []string{"local", "marker", "dangling-marker", "dangling-local", "marker-escape"} {
		t.Run(broken, func(t *testing.T) {
			f := newResolverFixture(t)
			broad := resolverBundle(t, filepath.Join(f.root, "broad"), "broad")
			resolverShared(t, f.cwd, broad)
			nested := filepath.Join(f.cwd, "nested")
			if err := os.Mkdir(nested, 0o755); err != nil {
				t.Fatal(err)
			}
			marker := filepath.Join(nested, "project.md")
			local := filepath.Join(nested, ".context", "config.md")
			switch broken {
			case "local":
				writeConfig(t, local, "not a configuration")
			case "marker":
				writeConfig(t, marker, "---\ntype: Project\nid: broken\n---\n")
			case "dangling-marker":
				if err := os.Symlink(filepath.Join(f.root, "missing"), marker); err != nil {
					t.Fatal(err)
				}
			case "dangling-local":
				if err := os.Mkdir(filepath.Dir(local), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(filepath.Join(f.root, "missing"), local); err != nil {
					t.Fatal(err)
				}
			case "marker-escape":
				if err := os.Symlink(filepath.Join(broad, "project.md"), marker); err != nil {
					t.Fatal(err)
				}
			}
			_, err := discovery.Resolve(context.Background(), f.request(discovery.Any, nested))
			if err == nil || !strings.Contains(err.Error(), nested) {
				t.Fatalf("invalid candidate fell through: %v", err)
			}
		})
	}
}

func TestResolveExplicitWorkspaceIgnoresMalformedProjectMarker(t *testing.T) {
	f := newResolverFixture(t)
	writeConfig(t, filepath.Join(f.cwd, ".context", "config.md"), configDocument("workspace: {id: team, title: Team, members: []}"))
	writeConfig(t, filepath.Join(f.cwd, "project.md"), "malformed marker")
	scope, err := discovery.Resolve(context.Background(), f.request(discovery.Workspace, ""))
	if err != nil || scope.Kind != discovery.Workspace {
		t.Fatalf("explicit workspace = %#v, %v", scope, err)
	}
	if _, err := discovery.Resolve(context.Background(), f.request(discovery.Any, "")); err == nil {
		t.Fatal("implicit discovery ignored same-depth malformed project marker")
	}
}

func TestResolveUnrelatedRecordTargetsRemainLazy(t *testing.T) {
	f := newResolverFixture(t)
	resolverBundle(t, f.cwd, "selected")
	loop := filepath.Join(f.root, "loop")
	if err := os.Symlink(loop, loop); err != nil {
		t.Fatal(err)
	}
	resolverPersonal(t, f.home, "projects:\n"+resolverProjectEntry("other", "", filepath.Join(f.root, "moved-checkout"), loop, loop))
	scope, err := discovery.Resolve(context.Background(), f.request(discovery.Any, ""))
	if err != nil || scope.Project.Records != f.cwd {
		t.Fatalf("unrelated targets blocked selection: %#v, %v", scope, err)
	}
}

func TestResolveUnresolvedBindingCannotHideSymlinkIntoStart(t *testing.T) {
	f := newResolverFixture(t)
	resolverBundle(t, f.cwd, "fallback")
	start := filepath.Join(f.cwd, "nested")
	hidden := filepath.Join(f.root, "hidden")
	for _, directory := range []string{start, hidden} {
		if err := os.Mkdir(directory, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	link := filepath.Join(hidden, "link")
	if err := os.Symlink(f.cwd, link); err != nil {
		t.Fatal(err)
	}
	resolverPersonal(t, f.home, "projects:\n"+resolverProjectEntry("hidden", "", appendPathForTest(link, "nested"), f.cwd))
	if err := os.Chmod(hidden, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(hidden, 0o755) })
	if _, err := os.Stat(link); err == nil {
		t.Skip("filesystem does not enforce directory access permissions for this user")
	}
	_, err := discovery.Resolve(context.Background(), f.request(discovery.Any, start))
	if err == nil || !strings.Contains(err.Error(), "projects[0].directory") || !strings.Contains(err.Error(), "cannot establish") {
		t.Fatalf("unresolved binding disappeared: %v", err)
	}
}

func appendPathForTest(parent, child string) string {
	return parent + string(filepath.Separator) + child
}

func TestResolveSelectedBindingChecksOriginalSpelling(t *testing.T) {
	f := newResolverFixture(t)
	resolverBundle(t, f.cwd, "selected")
	resolverPersonal(t, f.home, "projects:\n"+resolverProjectEntry("broken-binding", "", f.root+"/missing/../checkout", f.cwd))
	_, err := discovery.Resolve(context.Background(), f.request(discovery.Any, ""))
	if err == nil || !strings.Contains(err.Error(), "projects[0].directory") {
		t.Fatalf("broken authored directory was cleaned into an applicable binding: %v", err)
	}
}

func TestResolveLeavesFilesAndProcessDirectoryUnchanged(t *testing.T) {
	f := newResolverFixture(t)
	records := resolverBundle(t, filepath.Join(f.root, "records"), "one")
	config := resolverShared(t, f.cwd, records, f.cwd)
	before, err := os.ReadFile(config)
	if err != nil {
		t.Fatal(err)
	}
	processCwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := discovery.Resolve(context.Background(), f.request(discovery.Any, "")); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(config)
	if err != nil {
		t.Fatal(err)
	}
	finalCwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) || finalCwd != processCwd {
		t.Fatal("discovery changed configuration or process cwd")
	}
}

func TestResolveRejectsInvalidInvocationAndHonorsCancellation(t *testing.T) {
	f := newResolverFixture(t)
	for _, selector := range []string{filepath.Join(f.root, "missing"), filepath.Join(f.root, "file")} {
		if strings.HasSuffix(selector, "file") {
			writeConfig(t, selector, "file")
		}
		if _, err := discovery.Resolve(context.Background(), f.request(discovery.Project, selector)); err == nil {
			t.Fatalf("invalid start %s accepted", selector)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := discovery.Resolve(ctx, f.request(discovery.Any, "")); err == nil || !strings.Contains(err.Error(), "canceled") {
		t.Fatalf("cancellation error = %v", err)
	}
}

func TestResolveNoScopeExplainsRootSearchAndDirectBundleMigration(t *testing.T) {
	f := newResolverFixture(t)
	// Discovery never descends into this conventional records location.
	resolverBundle(t, filepath.Join(f.cwd, ".scratch", "records"), "not-an-ancestor")
	_, err := discovery.Resolve(context.Background(), f.request(discovery.Project, ""))
	for _, detail := range []string{f.cwd, "physical ancestors", "filesystem root", "ctx setup", "--bundle"} {
		if err == nil || !strings.Contains(err.Error(), detail) {
			t.Fatalf("no-scope error = %v, missing %q", err, detail)
		}
	}
}
