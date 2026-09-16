package workspace_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Zokiio/context/internal/discovery"
	"github.com/Zokiio/context/internal/workspace"
)

func navigationRoot(t *testing.T) string {
	t.Helper()
	path, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return path
}

func navigationDirectory(t *testing.T, path string) string {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func navigationDeclaration(t *testing.T, root, members string) discovery.WorkspaceDeclaration {
	t.Helper()
	path := filepath.Join(root, ".context", "config.md")
	navigationDirectory(t, filepath.Dir(path))
	source := "---\ntype: ContextConfig\nversion: 1\nworkspace:\n  id: navigation-fixture\n  title: Navigation fixture\n  members:" + members + "\n---\n"
	if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	config, err := discovery.ReadConfig(path, discovery.SharedConfig)
	if err != nil {
		t.Fatal(err)
	}
	return *config.Workspace
}

func TestNavigateEmptyWorkspace(t *testing.T) {
	declaration := navigationDeclaration(t, navigationRoot(t), " []")
	result, err := workspace.Navigate(context.Background(), declaration)
	if err != nil || !result.Complete || result.WorkStatus != "unevaluated" || result.Workspace.ID != declaration.ID || result.Workspace.Title != declaration.Title {
		t.Fatalf("result = %#v, %v", result, err)
	}
	if result.Members == nil || len(result.Members) != 0 || result.Diagnostics == nil || len(result.Diagnostics) != 0 {
		t.Fatalf("empty collections = %#v", result)
	}
}

func TestNavigateKeepsMembersAndRootsWithoutReadingRecords(t *testing.T) {
	root := navigationRoot(t)
	records := navigationDirectory(t, filepath.Join(root, "records"))
	other := navigationDirectory(t, filepath.Join(root, "other-records"))
	// Neither valid nor invalid manifest content affects this access check.
	if err := os.WriteFile(filepath.Join(records, "project.md"), []byte("---\ntype: Project\nid: same-id\ntitle: Same project\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(other, "project.md"), []byte("not a project manifest"), 0o644); err != nil {
		t.Fatal(err)
	}
	missingCheckout := filepath.Join(root, "moved-checkout")
	missingRoot := filepath.Join(root, "missing-docs")
	otherRoot := filepath.Join(root, "other-docs")
	declaration := navigationDeclaration(t, root, fmt.Sprintf("\n    - key: z-first\n      title: First checkout\n      records: %q\n      directory: %q\n      allowSources: [%q]\n    - key: a-second\n      records: %q\n      directory: %q\n      allowSources: [%q]\n    - key: another-store\n      records: %q", records, missingCheckout, missingRoot, records, filepath.Join(root, "second-checkout"), otherRoot, other))
	result, err := workspace.Navigate(context.Background(), declaration)
	if err != nil || !result.Complete || len(result.Members) != 3 || len(result.Diagnostics) != 0 {
		t.Fatalf("result = %#v, %v", result, err)
	}
	for i, key := range []string{"z-first", "a-second", "another-store"} {
		if result.Members[i].Key != key || result.Members[i].Availability != workspace.Available {
			t.Fatalf("member %d = %#v", i, result.Members[i])
		}
	}
	if result.Members[0].Directory != missingCheckout || result.Members[0].AllowSources[0] != missingRoot || result.Members[1].AllowSources[0] != otherRoot {
		t.Fatalf("authored member settings changed: %#v", result.Members)
	}
}

func TestNavigateKeepsDistinctRecordStoresWithSameProjectID(t *testing.T) {
	root := navigationRoot(t)
	firstRecords := navigationDirectory(t, filepath.Join(root, "first-records"))
	secondRecords := navigationDirectory(t, filepath.Join(root, "second-records"))
	firstCheckout := navigationDirectory(t, filepath.Join(root, "first-checkout"))
	secondCheckout := navigationDirectory(t, filepath.Join(root, "second-checkout"))
	for _, records := range []string{firstRecords, secondRecords} {
		if err := os.WriteFile(filepath.Join(records, "project.md"), []byte("---\ntype: Project\nid: same-id\ntitle: Same project\n---\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	declaration := navigationDeclaration(t, root, fmt.Sprintf("\n    - key: z-first\n      records: %q\n      directory: %q\n    - key: a-second\n      records: %q\n      directory: %q", firstRecords, firstCheckout, secondRecords, secondCheckout))
	result, err := workspace.Navigate(context.Background(), declaration)
	if err != nil || !result.Complete || len(result.Members) != 2 || len(result.Diagnostics) != 0 {
		t.Fatalf("result = %#v, %v", result, err)
	}
	for i, want := range []struct{ key, records, directory string }{
		{"z-first", firstRecords, firstCheckout},
		{"a-second", secondRecords, secondCheckout},
	} {
		member := result.Members[i]
		if member.Key != want.key || member.Records != want.records || member.Directory != want.directory || member.Availability != workspace.Available {
			t.Fatalf("member %d = %#v; want %#v", i, member, want)
		}
	}
}

func TestNavigateUnavailableRecordsStillComplete(t *testing.T) {
	root := navigationRoot(t)
	available := navigationDirectory(t, filepath.Join(root, "available"))
	file := filepath.Join(root, "file")
	if err := os.WriteFile(file, []byte("file"), 0o644); err != nil {
		t.Fatal(err)
	}
	dangling, loop := filepath.Join(root, "dangling"), filepath.Join(root, "loop")
	if err := os.Symlink(filepath.Join(root, "absent"), dangling); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(loop, loop); err != nil {
		t.Fatal(err)
	}
	paths := []string{filepath.Join(root, "absent"), file, root + "/missing/../available", file + "/../available", dangling, loop}
	if mkfifo, err := exec.LookPath("mkfifo"); err == nil {
		fifo := filepath.Join(root, "fifo")
		if output, err := exec.Command(mkfifo, fifo).CombinedOutput(); err != nil {
			t.Fatalf("create FIFO: %v: %s", err, output)
		}
		paths = append(paths, fifo)
	}
	for _, path := range paths {
		t.Run(filepath.Base(path), func(t *testing.T) {
			declaration := navigationDeclaration(t, root, fmt.Sprintf("\n    - key: member\n      records: %q", path))
			// A canonical spelling must not repair the authored traversal.
			declaration.Members[0].Records.Canonical = available
			result, err := workspace.Navigate(context.Background(), declaration)
			if err != nil || !result.Complete || len(result.Members) != 1 || result.Members[0].Availability != workspace.Unavailable || result.Members[0].Records != path {
				t.Fatalf("result = %#v, %v", result, err)
			}
			if len(result.Diagnostics) != 1 || result.Diagnostics[0].MemberKey != "member" || result.Diagnostics[0].Path != path || result.Diagnostics[0].Field != "workspace.members[0].records" {
				t.Fatalf("diagnostic = %#v", result.Diagnostics)
			}
		})
	}
}

func TestNavigateDoesNotOpenMemberFIFOManifest(t *testing.T) {
	mkfifo, err := exec.LookPath("mkfifo")
	if err != nil {
		t.Skip("mkfifo is unavailable")
	}
	root := navigationRoot(t)
	records := navigationDirectory(t, filepath.Join(root, "records"))
	if output, err := exec.Command(mkfifo, filepath.Join(records, "project.md")).CombinedOutput(); err != nil {
		t.Fatalf("create FIFO: %v: %s", err, output)
	}
	declaration := navigationDeclaration(t, root, fmt.Sprintf("\n    - key: shallow\n      records: %q", records))
	result, err := workspace.Navigate(context.Background(), declaration)
	if err != nil || result.Members[0].Availability != workspace.Available {
		t.Fatalf("shallow access opened a named member file: %#v, %v", result, err)
	}
}

func TestNavigatePermissionFailureIsUnavailable(t *testing.T) {
	root := navigationRoot(t)
	records := navigationDirectory(t, filepath.Join(root, "records"))
	declaration := navigationDeclaration(t, root, fmt.Sprintf("\n    - key: private\n      records: %q", records))
	if err := os.Chmod(records, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(records, 0o755) })
	if _, err := os.ReadDir(records); err == nil {
		t.Skip("filesystem does not enforce permissions for this user")
	}
	result, err := workspace.Navigate(context.Background(), declaration)
	if err != nil || !result.Complete || result.Members[0].Availability != workspace.Unavailable {
		t.Fatalf("result = %#v, %v", result, err)
	}
}

func TestNavigateUnexpectedAccessFailureIsUnknown(t *testing.T) {
	root := navigationRoot(t)
	path := filepath.Join(root, strings.Repeat("x", 300))
	declaration := navigationDeclaration(t, root, fmt.Sprintf("\n    - key: unresolved\n      records: %q", path))
	result, err := workspace.Navigate(context.Background(), declaration)
	if err != nil || !result.Complete || result.Members[0].Availability != workspace.Unknown || len(result.Diagnostics) != 1 {
		t.Fatalf("result = %#v, %v", result, err)
	}
}

func TestNavigatePreservesPhysicalTraversalAndDoesNotWrite(t *testing.T) {
	root := navigationRoot(t)
	physical := navigationDirectory(t, filepath.Join(root, "physical", "child"))
	navigationDirectory(t, filepath.Join(root, "physical", "records"))
	if err := os.Symlink(physical, filepath.Join(root, "link")); err != nil {
		t.Fatal(err)
	}
	path := root + "/link/../records"
	declaration := navigationDeclaration(t, root, fmt.Sprintf("\n    - key: physical\n      records: %q\n      allowSources: [%q]", path, root+"/link/../docs"))
	configPath := filepath.Join(root, ".context", "config.md")
	before, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	result, err := workspace.Navigate(context.Background(), declaration)
	if err != nil || result.Members[0].Availability != workspace.Available || result.Members[0].Records != path || result.Members[0].AllowSources[0] != root+"/link/../docs" {
		t.Fatalf("result = %#v, %v", result, err)
	}
	after, err := os.ReadFile(configPath)
	if err != nil || string(after) != string(before) {
		t.Fatalf("configuration changed: %v", err)
	}
}

func TestNavigateRejectsInvalidDeclarationAndCancellation(t *testing.T) {
	root := navigationRoot(t)
	base := navigationDeclaration(t, root, " []")
	for _, declaration := range []discovery.WorkspaceDeclaration{
		{},
		{ID: "id"},
		{ID: "id", Title: "title", Members: []discovery.Member{{Key: "missing-records"}}},
		{ID: "id", Title: "title", Members: []discovery.Member{{Key: "same", Records: discovery.PathValue{Path: root}}, {Key: "same", Records: discovery.PathValue{Path: root}}}},
	} {
		if _, err := workspace.Navigate(context.Background(), declaration); err == nil {
			t.Fatalf("invalid declaration accepted: %#v", declaration)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := workspace.Navigate(ctx, base); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation = %v", err)
	}
}
