package discovery_test

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Zokiio/context/internal/discovery"
)

func TestResolveSelectedRootsAndAliasRecordsKeepOriginalSpelling(t *testing.T) {
	for _, target := range []string{"allowed-root", "alias-records"} {
		t.Run(target, func(t *testing.T) {
			f := newResolverFixture(t)
			resolverBundle(t, f.cwd, "broader-project")
			records := resolverBundle(t, filepath.Join(f.root, "records"), "selected")
			docs := conformanceMkdir(t, filepath.Join(f.root, "docs"))
			nested := conformanceMkdir(t, filepath.Join(f.cwd, "nested"))
			request := f.request(discovery.Any, nested)
			var authored, field string
			if target == "allowed-root" {
				authored = f.root + "/missing/../docs"
				field = "project.allowSources[0]"
				resolverShared(t, nested, records, authored)
			} else {
				authored = f.root + "/missing/../records"
				field = "projects[0].records"
				resolverPersonal(t, f.home, "projects:\n"+resolverProjectEntry("selected", "saved", nested, authored, docs))
				request.Kind, request.Selector = discovery.Project, "@saved"
			}
			_, err := discovery.Resolve(context.Background(), request)
			for _, detail := range []string{field, authored, "unavailable", "repair"} {
				if err == nil || !strings.Contains(err.Error(), detail) {
					t.Fatalf("selected path was cleaned or fell back: %v; missing %q", err, detail)
				}
			}
		})
	}
}

func TestResolveWorkspaceIgnoresBrokenProjectTargets(t *testing.T) {
	f := newResolverFixture(t)
	config := filepath.Join(f.cwd, ".context", "config.md")
	writeConfig(t, config, configDocument("project: {records: ../missing-records, allowSources: [../missing-docs]}\nworkspace: {id: team, title: Team, members: []}"))
	scope, err := discovery.Resolve(context.Background(), f.request(discovery.Workspace, ""))
	if err != nil || scope.Kind != discovery.Workspace || scope.Workspace.ID != "team" {
		t.Fatalf("explicit workspace depended on filtered project targets: %+v, %v", scope, err)
	}
	if _, err := discovery.Resolve(context.Background(), f.request(discovery.Any, "")); err == nil || !strings.Contains(err.Error(), "project.records") {
		t.Fatalf("implicit selection ignored broken project candidate: %v", err)
	}
}

func TestResolveWorkspaceMembersDoNotCreateBindingsOrRequireAvailableRecords(t *testing.T) {
	f := newResolverFixture(t)
	member := conformanceMkdir(t, filepath.Join(f.root, "member-checkout"))
	missingRecords := filepath.Join(f.root, "missing-member-records")
	config := filepath.Join(f.cwd, ".context", "config.md")
	writeConfig(t, config, configDocument(fmt.Sprintf("workspace:\n  id: team\n  title: Team\n  members:\n    - key: member\n      directory: %q\n      records: %q\n      allowSources: [%q]", member, missingRecords, filepath.Join(f.root, "missing-member-docs"))))
	scope, err := discovery.Resolve(context.Background(), f.request(discovery.Any, ""))
	if err != nil || scope.Kind != discovery.Workspace || len(scope.Workspace.Members) != 1 || scope.Workspace.Members[0].Records.Canonical != missingRecords {
		t.Fatalf("unavailable member prevented workspace selection: %+v, %v", scope, err)
	}
	request := f.request(discovery.Any, member)
	if _, err := discovery.Resolve(context.Background(), request); err == nil || !strings.Contains(err.Error(), "no scope found") {
		t.Fatalf("workspace member became a cwd binding: %v", err)
	}
}

func TestResolveDirectoryBindingsRespectPathComponentBoundaries(t *testing.T) {
	f := newResolverFixture(t)
	first := conformanceMkdir(t, filepath.Join(f.cwd, "app"))
	start := conformanceMkdir(t, filepath.Join(f.cwd, "apple"))
	records := resolverBundle(t, filepath.Join(f.root, "broader"), "broader")
	other := resolverBundle(t, filepath.Join(f.root, "other"), "other")
	resolverShared(t, f.cwd, records)
	resolverPersonal(t, f.home, "projects:\n"+resolverProjectEntry("sibling", "", first, other))
	scope, err := discovery.Resolve(context.Background(), f.request(discovery.Any, start))
	if err != nil {
		t.Fatal(err)
	}
	conformanceAssertProject(t, scope, f.cwd, records)
}
