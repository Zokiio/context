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

func conformanceMkdir(t *testing.T, path string) string {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func conformanceProject(t *testing.T, path, id string) string {
	t.Helper()
	writeConfig(t, filepath.Join(path, "project.md"), fmt.Sprintf("---\ntype: Project\nid: %q\ntitle: Project %s\n---\n", id, id))
	return path
}

func conformanceAssertProject(t *testing.T, scope discovery.Scope, binding, records string) {
	t.Helper()
	if scope.Kind != discovery.Project || scope.Project == nil || scope.Workspace != nil {
		t.Fatalf("expected project scope, got %+v", scope)
	}
	if scope.BindingDirectory != binding || scope.Project.Records != records {
		t.Fatalf("scope selected binding %q and records %q, want %q and %q", scope.BindingDirectory, scope.Project.Records, binding, records)
	}
}

func TestResolveConformanceCoalescesEquivalentWorkspacesAndKeepsMemberOrder(t *testing.T) {
	root := physicalTempDir(t)
	checkout := conformanceMkdir(t, filepath.Join(root, "checkout"))
	home := conformanceMkdir(t, filepath.Join(root, "home"))
	first := conformanceProject(t, filepath.Join(root, "first-records"), "same-project-id")
	second := conformanceProject(t, filepath.Join(root, "second-records"), "same-project-id")
	firstCheckout := conformanceMkdir(t, filepath.Join(root, "first-checkout"))
	secondCheckout := conformanceMkdir(t, filepath.Join(root, "second-checkout"))
	docs := conformanceMkdir(t, filepath.Join(root, "docs"))
	otherDocs := conformanceMkdir(t, filepath.Join(root, "other-docs"))
	alias := filepath.Join(root, "first-records-alias")
	if err := os.Symlink(first, alias); err != nil {
		t.Fatal(err)
	}
	sharedFile := filepath.Join(checkout, ".context", "config.md")
	personalFile := filepath.Join(home, ".context", "config.md")
	writeConfig(t, sharedFile, configDocument(fmt.Sprintf(`workspace:
  id: shared-workspace
  title: Shared workspace
  members:
    - key: z-first
      title: First checkout
      records: ../../first-records
      directory: ../../first-checkout
      allowSources: [%q, %q]
    - key: a-second
      records: ../../second-records
      directory: ../../second-checkout`, docs, otherDocs)))
	writeConfig(t, personalFile, configDocument(fmt.Sprintf(`workspaces:
  - key: saved-workspace
    directory: %q
    id: shared-workspace
    title: Shared workspace
    members:
      - key: z-first
        title: First checkout
        records: %q
        directory: %q
        allowSources: [%q, %q, %q]
      - key: a-second
        records: %q
        directory: %q`, checkout, alias, firstCheckout, otherDocs, docs, docs, second, secondCheckout)))

	scope, err := discovery.Resolve(context.Background(), discovery.Request{Cwd: checkout, Home: home})
	if err != nil {
		t.Fatal(err)
	}
	if scope.Kind != discovery.Workspace || scope.Project != nil || scope.Workspace == nil || scope.Workspace.ID != "shared-workspace" {
		t.Fatalf("expected selected workspace, got %+v", scope)
	}
	members := scope.Workspace.Members
	if len(members) != 2 || members[0].Key != "z-first" || members[1].Key != "a-second" {
		t.Fatalf("authored member order or distinct checkouts were lost: %+v", members)
	}
	if members[0].Records.Canonical != first || members[1].Records.Canonical != second {
		t.Fatalf("workspace member record stores changed: %+v", members)
	}
	if scope.BindingDirectory != checkout {
		t.Fatalf("binding = %q, want %q", scope.BindingDirectory, checkout)
	}
	wantOrigins := map[string]bool{sharedFile: false, personalFile: false}
	for _, origin := range scope.Origins {
		if _, ok := wantOrigins[origin.Path]; !ok {
			t.Fatalf("unexpected coalesced origin: %+v", origin)
		}
		wantOrigins[origin.Path] = true
	}
	if len(scope.Origins) != 2 || !wantOrigins[sharedFile] || !wantOrigins[personalFile] {
		t.Fatalf("both configuration origins must survive coalescing: %+v", scope.Origins)
	}
}

func conformanceIndent(text string, width int) string {
	prefix := strings.Repeat(" ", width)
	return prefix + strings.ReplaceAll(strings.TrimSuffix(text, "\n"), "\n", "\n"+prefix) + "\n"
}

func TestResolveConformanceWorkspaceIDDoesNotMergeDifferentDeclarations(t *testing.T) {
	for _, difference := range []string{"records", "allowed roots", "checkout", "member title", "member order", "workspace title"} {
		t.Run(difference, func(t *testing.T) {
			root := physicalTempDir(t)
			checkout := conformanceMkdir(t, filepath.Join(root, "checkout"))
			home := conformanceMkdir(t, filepath.Join(root, "home"))
			first := conformanceProject(t, filepath.Join(root, "first"), "first")
			second := conformanceProject(t, filepath.Join(root, "second"), "second")
			firstCheckout := conformanceMkdir(t, filepath.Join(root, "first-checkout"))
			secondCheckout := conformanceMkdir(t, filepath.Join(root, "second-checkout"))
			docs := conformanceMkdir(t, filepath.Join(root, "docs"))
			otherDocs := conformanceMkdir(t, filepath.Join(root, "other-docs"))
			firstMember := fmt.Sprintf("- key: first\n  title: First member\n  records: %q\n  directory: %q\n  allowSources: [%q]\n", first, firstCheckout, docs)
			secondMember := fmt.Sprintf("- key: second\n  records: %q\n", second)
			declaration := "id: stable-workspace-id\ntitle: Workspace title\nmembers:\n" + conformanceIndent(firstMember+secondMember, 2)
			changed := declaration
			switch difference {
			case "records":
				changed = strings.Replace(changed, fmt.Sprintf("records: %q", first), fmt.Sprintf("records: %q", second), 1)
			case "allowed roots":
				changed = strings.Replace(changed, fmt.Sprintf("allowSources: [%q]", docs), fmt.Sprintf("allowSources: [%q]", otherDocs), 1)
			case "checkout":
				changed = strings.Replace(changed, fmt.Sprintf("directory: %q", firstCheckout), fmt.Sprintf("directory: %q", secondCheckout), 1)
			case "member title":
				changed = strings.Replace(changed, "title: First member", "title: Other member title", 1)
			case "member order":
				changed = "id: stable-workspace-id\ntitle: Workspace title\nmembers:\n" + conformanceIndent(secondMember+firstMember, 2)
			case "workspace title":
				changed = strings.Replace(changed, "title: Workspace title", "title: Another workspace title", 1)
			}
			sharedFile := filepath.Join(checkout, ".context", "config.md")
			personalFile := filepath.Join(home, ".context", "config.md")
			writeConfig(t, sharedFile, configDocument("workspace:\n"+conformanceIndent(declaration, 2)))
			writeConfig(t, personalFile, configDocument(fmt.Sprintf("workspaces:\n  - key: saved-workspace\n    directory: %q\n", checkout)+conformanceIndent(changed, 4)))

			_, err := discovery.Resolve(context.Background(), discovery.Request{Cwd: checkout, Home: home, Kind: discovery.Workspace, Selector: checkout})
			if err == nil {
				t.Fatalf("different %s under the same workspace ID must conflict", difference)
			}
			for _, origin := range []string{sharedFile, personalFile} {
				if !strings.Contains(err.Error(), origin) {
					t.Errorf("conflict must identify configuration %q: %v", origin, err)
				}
			}
		})
	}
}

func TestResolveConformanceSearchesOnlyPhysicalAncestors(t *testing.T) {
	root := physicalTempDir(t)
	home := conformanceMkdir(t, filepath.Join(root, "home"))
	checkout := conformanceMkdir(t, filepath.Join(root, "physical", "checkout"))
	conformanceMkdir(t, filepath.Join(checkout, "src"))
	records := conformanceProject(t, filepath.Join(root, "records"), "physical-project")
	lexical := conformanceMkdir(t, filepath.Join(root, "lexical"))
	link := filepath.Join(lexical, "checkout-link")
	if err := os.Symlink(checkout, link); err != nil {
		t.Fatal(err)
	}
	writeConfig(t, filepath.Join(lexical, ".context", "config.md"), "Malformed configuration on an ancestor of the symlink spelling.\n")
	writeConfig(t, filepath.Join(checkout, ".context", "config.md"), configDocument(fmt.Sprintf("project:\n  records: %q", records)))
	entered := filepath.Join(link, "src")
	for _, request := range []discovery.Request{
		{Cwd: entered, Home: home},
		{Cwd: root, Home: home, Kind: discovery.Project, Selector: entered},
	} {
		scope, err := discovery.Resolve(context.Background(), request)
		if err != nil {
			t.Fatalf("physical discovery from %+v: %v", request, err)
		}
		conformanceAssertProject(t, scope, checkout, records)
	}
}

func TestResolveConformanceFollowsSymlinkBeforeParentTraversal(t *testing.T) {
	root := physicalTempDir(t)
	home := conformanceMkdir(t, filepath.Join(root, "home"))
	physical := conformanceMkdir(t, filepath.Join(root, "physical"))
	linkTarget := conformanceMkdir(t, filepath.Join(physical, "target"))
	checkout := conformanceMkdir(t, filepath.Join(physical, "next"))
	conformanceMkdir(t, filepath.Join(checkout, "src"))
	records := conformanceProject(t, filepath.Join(root, "right-records"), "right-project")
	lexical := conformanceMkdir(t, filepath.Join(root, "lexical"))
	wrongCheckout := conformanceMkdir(t, filepath.Join(lexical, "next"))
	conformanceMkdir(t, filepath.Join(wrongCheckout, "src"))
	wrongRecords := conformanceProject(t, filepath.Join(root, "wrong-records"), "wrong-project")
	link := filepath.Join(lexical, "checkout-link")
	if err := os.Symlink(linkTarget, link); err != nil {
		t.Fatal(err)
	}
	writeConfig(t, filepath.Join(checkout, ".context", "config.md"), configDocument(fmt.Sprintf("project:\n  records: %q", records)))
	writeConfig(t, filepath.Join(wrongCheckout, ".context", "config.md"), configDocument(fmt.Sprintf("project:\n  records: %q", wrongRecords)))
	entered := link + string(filepath.Separator) + ".." + string(filepath.Separator) + "next" + string(filepath.Separator) + "src"
	for _, request := range []discovery.Request{
		{Cwd: entered, Home: home},
		{Cwd: root, Home: home, Kind: discovery.Project, Selector: entered},
	} {
		scope, err := discovery.Resolve(context.Background(), request)
		if err != nil {
			t.Fatalf("symlink-parent discovery from %+v: %v", request, err)
		}
		conformanceAssertProject(t, scope, checkout, records)
	}
}

func TestResolveConformancePersonalRegistryIsNotAnAncestorMarker(t *testing.T) {
	for _, aliasHome := range []bool{false, true} {
		t.Run(fmt.Sprintf("home-alias=%t", aliasHome), func(t *testing.T) {
			root := physicalTempDir(t)
			physicalHome := conformanceMkdir(t, filepath.Join(root, "physical-home"))
			home := physicalHome
			if aliasHome {
				home = filepath.Join(root, "home-alias")
				if err := os.Symlink(physicalHome, home); err != nil {
					t.Fatal(err)
				}
			}
			cwd := conformanceMkdir(t, filepath.Join(physicalHome, "checkout", "src"))
			writeConfig(t, filepath.Join(root, ".context", "config.md"), configDocument("workspace:\n  id: enclosing-workspace\n  title: Enclosing workspace\n  members: []"))
			writeConfig(t, filepath.Join(physicalHome, ".context", "config.md"), configDocument("workspaces:\n  - key: personal-entry\n    alias: personal-only\n    id: personal-workspace\n    title: Personal workspace\n    members: []"))

			scope, err := discovery.Resolve(context.Background(), discovery.Request{Cwd: cwd, Home: home})
			if err != nil {
				t.Fatal(err)
			}
			if scope.Kind != discovery.Workspace || scope.Workspace == nil || scope.Workspace.ID != "enclosing-workspace" || scope.BindingDirectory != root {
				t.Fatalf("personal registry created an implicit home scope: %+v", scope)
			}
		})
	}
}

func TestResolveConformanceMatchesPhysicalBindingsOnCaseInsensitiveVolumes(t *testing.T) {
	root := physicalTempDir(t)
	home := conformanceMkdir(t, filepath.Join(root, "home"))
	checkout := conformanceMkdir(t, filepath.Join(root, "Checkout"))
	cwd := conformanceMkdir(t, filepath.Join(checkout, "src"))
	enteredCheckout := filepath.Join(root, "checkout")
	actualInfo, err := os.Stat(checkout)
	if err != nil {
		t.Fatal(err)
	}
	aliasInfo, err := os.Stat(enteredCheckout)
	if os.IsNotExist(err) {
		t.Skip("fixture filesystem distinguishes path casing")
	}
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(actualInfo, aliasInfo) {
		t.Fatal("fixture paths do not identify the same checkout")
	}
	records := conformanceProject(t, filepath.Join(root, "Records"), "case-project")
	docs := conformanceMkdir(t, filepath.Join(root, "Docs"))
	sharedFile := filepath.Join(checkout, ".context", "config.md")
	personalFile := filepath.Join(home, ".context", "config.md")
	writeConfig(t, sharedFile, configDocument(fmt.Sprintf("project:\n  records: %q\n  allowSources: [%q]", records, docs)))
	writeConfig(t, personalFile, configDocument(fmt.Sprintf("projects:\n  - key: same-project\n    directory: %q\n    records: %q\n    allowSources: [%q]", enteredCheckout, filepath.Join(root, "records"), filepath.Join(root, "docs"))))

	scope, err := discovery.Resolve(context.Background(), discovery.Request{Cwd: cwd, Home: home})
	if err != nil {
		t.Fatal(err)
	}
	if scope.Project == nil || scope.Kind != discovery.Project || len(scope.Origins) != 2 {
		t.Fatalf("physically identical declarations were not coalesced: %+v", scope)
	}
	conformanceAssertSameFile(t, scope.BindingDirectory, checkout)
	conformanceAssertSameFile(t, scope.Project.Records, records)
	if len(scope.Project.AllowSources) != 1 {
		t.Fatalf("allowed root set changed: %+v", scope.Project.AllowSources)
	}
	conformanceAssertSameFile(t, scope.Project.AllowSources[0], docs)
}

func conformanceAssertSameFile(t *testing.T, got, want string) {
	t.Helper()
	gotInfo, err := os.Stat(got)
	if err != nil {
		t.Fatal(err)
	}
	wantInfo, err := os.Stat(want)
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(gotInfo, wantInfo) {
		t.Fatalf("selected path %q does not identify %q", got, want)
	}
}

func TestResolveConformanceCrossesHomeAndGitBoundaries(t *testing.T) {
	for _, gitKind := range []string{"directory", "file"} {
		t.Run(gitKind, func(t *testing.T) {
			root := physicalTempDir(t)
			home := conformanceMkdir(t, filepath.Join(root, "home"))
			repository := conformanceMkdir(t, filepath.Join(home, "repository"))
			cwd := conformanceMkdir(t, filepath.Join(repository, "src", "nested"))
			records := conformanceProject(t, filepath.Join(root, "records"), "enclosing-project")
			if gitKind == "directory" {
				conformanceMkdir(t, filepath.Join(repository, ".git"))
			} else {
				writeConfig(t, filepath.Join(repository, ".git"), "gitdir: ../../separate-git-directory\n")
			}
			writeConfig(t, filepath.Join(root, ".context", "config.md"), configDocument("project:\n  records: ../records"))
			writeConfig(t, filepath.Join(home, ".context", "config.md"), configDocument("projects: []\nworkspaces: []"))

			scope, err := discovery.Resolve(context.Background(), discovery.Request{Cwd: cwd, Home: home})
			if err != nil {
				t.Fatal(err)
			}
			conformanceAssertProject(t, scope, root, records)
		})
	}
}

func TestResolveConformanceStopsBeforeBroaderConfigurationThatCannotWin(t *testing.T) {
	root := physicalTempDir(t)
	home := conformanceMkdir(t, filepath.Join(root, "home"))
	checkout := conformanceMkdir(t, filepath.Join(root, "checkout"))
	cwd := conformanceMkdir(t, filepath.Join(checkout, "src"))
	broaderConfig := filepath.Join(root, ".context", "config.md")
	writeConfig(t, broaderConfig, "Invalid shared configuration at the broader scope.\n")
	writeConfig(t, filepath.Join(checkout, ".context", "config.md"), configDocument("workspace:\n  id: nearest-workspace\n  title: Nearest workspace\n  members: []"))
	for _, request := range []discovery.Request{
		{Cwd: cwd, Home: home},
		{Cwd: cwd, Home: home, Kind: discovery.Workspace, Selector: "."},
	} {
		scope, err := discovery.Resolve(context.Background(), request)
		if err != nil {
			t.Fatalf("broader configuration cannot win for %+v: %v", request, err)
		}
		if scope.Kind != discovery.Workspace || scope.Workspace == nil || scope.Workspace.ID != "nearest-workspace" || scope.BindingDirectory != checkout {
			t.Fatalf("selected scope = %+v", scope)
		}
	}
	_, err := discovery.Resolve(context.Background(), discovery.Request{Cwd: cwd, Home: home, Kind: discovery.Project, Selector: "."})
	if err == nil || !strings.Contains(err.Error(), broaderConfig) {
		t.Fatalf("project selection must inspect the still-relevant broader configuration: %v", err)
	}
}
