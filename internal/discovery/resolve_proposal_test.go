package discovery

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func proposalFixture(t *testing.T) (Request, string) {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	cwd, home, records := filepath.Join(root, "checkout"), filepath.Join(root, "home"), filepath.Join(root, "checkout", "records")
	for _, directory := range []string{cwd, home, records} {
		if err := os.MkdirAll(directory, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	manifest := []byte("---\ntype: Project\nid: fixture\ntitle: Fixture\n---\n")
	if err := os.WriteFile(filepath.Join(records, "project.md"), manifest, 0o644); err != nil {
		t.Fatal(err)
	}
	return Request{Cwd: cwd, Home: home, Kind: Project}, records
}

func TestResolveProposedSharedConfigurationBeforeParentExists(t *testing.T) {
	request, records := proposalFixture(t)
	path := filepath.Join(request.Cwd, ".context", "config.md")
	proposed, err := ParseConfig(path, []byte("---\ntype: ContextConfig\nversion: 1\nproject:\n  records: ../records\n  allowSources: [..]\n---\n"), SharedConfig)
	if err != nil {
		t.Fatal(err)
	}
	scope, err := resolve(context.Background(), request, proposed)
	if err != nil || scope.Project.Records != records || len(scope.Project.AllowSources) != 1 || scope.Project.AllowSources[0] != request.Cwd {
		t.Fatalf("proposed scope = %#v, %v", scope, err)
	}
	if _, err := os.Lstat(filepath.Dir(path)); !os.IsNotExist(err) {
		t.Fatalf("proposal created a directory: %v", err)
	}
}

func TestResolveProposedConfigurationDoesNotCleanBrokenTarget(t *testing.T) {
	request, _ := proposalFixture(t)
	path := filepath.Join(request.Cwd, ".context", "config.md")
	proposed, err := ParseConfig(path, []byte("---\ntype: ContextConfig\nversion: 1\nproject: {records: ../missing/../records}\n---\n"), SharedConfig)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := resolve(context.Background(), request, proposed); err == nil || !strings.Contains(err.Error(), "unavailable") {
		t.Fatalf("proposal ignored broken target component: %v", err)
	}
}

func TestResolveProposedConfigurationRetainsPhysicalParentTraversal(t *testing.T) {
	request, _ := proposalFixture(t)
	physical := filepath.Join(request.Cwd, "physical")
	child := filepath.Join(physical, "child")
	records := filepath.Join(physical, "records")
	for _, directory := range []string{child, records} {
		if err := os.MkdirAll(directory, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(child, filepath.Join(request.Cwd, "alias")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(records, "project.md"), []byte("---\ntype: Project\nid: physical\ntitle: Physical\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(request.Cwd, ".context", "config.md")
	proposed, err := ParseConfig(path, []byte("---\ntype: ContextConfig\nversion: 1\nproject: {records: ../alias/../records}\n---\n"), SharedConfig)
	if err != nil {
		t.Fatal(err)
	}
	scope, err := resolve(context.Background(), request, proposed)
	if err != nil || scope.Project.Records != records {
		t.Fatalf("proposed physical scope = %#v, %v", scope, err)
	}
}

func TestResolveProposedPersonalConfigurationPreservesOtherFileConflicts(t *testing.T) {
	request, records := proposalFixture(t)
	local := filepath.Join(request.Cwd, ".context", "config.md")
	if err := os.Mkdir(filepath.Dir(local), 0o755); err != nil {
		t.Fatal(err)
	}
	content := fmt.Sprintf("---\ntype: ContextConfig\nversion: 1\nproject:\n  records: %q\n  allowSources: [%q]\n---\n", records, request.Cwd)
	if err := os.WriteFile(local, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	personal := filepath.Join(request.Home, ".context", "config.md")
	content = fmt.Sprintf("---\ntype: ContextConfig\nversion: 1\nprojects:\n  - key: new\n    directory: %q\n    records: %q\n---\n", request.Cwd, records)
	proposed, err := ParseConfig(personal, []byte(content), PersonalConfig)
	if err != nil {
		t.Fatal(err)
	}
	_, err = resolve(context.Background(), request, proposed)
	if err == nil || !strings.Contains(err.Error(), "conflicting") || !strings.Contains(err.Error(), personal) || !strings.Contains(err.Error(), local) {
		t.Fatalf("proposal hid other-file conflict: %v", err)
	}
	if _, err := os.Lstat(personal); !os.IsNotExist(err) {
		t.Fatalf("proposal wrote personal configuration: %v", err)
	}
}
