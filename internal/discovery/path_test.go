package discovery_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Zokiio/context/internal/discovery"
)

func TestCanonicalPathResolvesExistingAndMissingSymlinkTargets(t *testing.T) {
	root := physicalTempDir(t)
	real := filepath.Join(root, "physical")
	if err := os.Mkdir(real, 0o755); err != nil {
		t.Fatal(err)
	}
	for alias, target := range map[string]string{
		"alias":    real,
		"relative": "physical",
		"dangling": "alias/not-present",
	} {
		if err := os.Symlink(target, filepath.Join(root, alias)); err != nil {
			t.Fatal(err)
		}
	}
	for _, test := range []struct{ input, want string }{
		{"alias", real},
		{"relative", real},
		{"alias/not-present/child", filepath.Join(real, "not-present", "child")},
		{"dangling", filepath.Join(real, "not-present")},
		{"dangling/child", filepath.Join(real, "not-present", "child")},
		{"ordinary-missing/child", filepath.Join(root, "ordinary-missing", "child")},
	} {
		t.Run(test.input, func(t *testing.T) {
			got, err := discovery.CanonicalPath(filepath.Join(root, test.input))
			if err != nil || got != test.want {
				t.Fatalf("CanonicalPath = %q, %v; want %q", got, err, test.want)
			}
		})
	}
}

func TestCanonicalPathFollowsSymlinkBeforeParentTraversal(t *testing.T) {
	root := physicalTempDir(t)
	real := filepath.Join(root, "physical", "child")
	if err := os.MkdirAll(real, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "alias")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	got, err := discovery.CanonicalPath(link + "/../records")
	want := filepath.Join(root, "physical", "records")
	if err != nil || got != want {
		t.Fatalf("CanonicalPath = %q, %v; want %q", got, err, want)
	}
}

func TestCanonicalPathReportsNonDirectoryAncestor(t *testing.T) {
	root := physicalTempDir(t)
	file := filepath.Join(root, "file")
	if err := os.WriteFile(file, []byte("not a directory"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, suffix := range []string{"/records", "/../records", "/"} {
		path := file + suffix
		got, err := discovery.CanonicalPath(path)
		if err == nil || got == "" {
			t.Fatalf("CanonicalPath(%q) = %q, %v; want diagnostic target and error", path, got, err)
		}
	}
}

func TestCanonicalPathDetectsLoop(t *testing.T) {
	root := physicalTempDir(t)
	first, second := filepath.Join(root, "first"), filepath.Join(root, "second")
	if err := os.Symlink(second, first); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(first, second); err != nil {
		t.Fatal(err)
	}
	if path, err := discovery.CanonicalPath(filepath.Join(first, "records")); err == nil || path == "" {
		t.Fatalf("CanonicalPath loop = %q, %v", path, err)
	}
}

func TestSamePathComparesPhysicalIdentity(t *testing.T) {
	root := physicalTempDir(t)
	records := filepath.Join(root, "Records")
	if err := os.Mkdir(records, 0o755); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(root, "alias")
	if err := os.Symlink(records, alias); err != nil {
		t.Fatal(err)
	}
	if !discovery.SamePath(records, alias) || discovery.SamePath(root, records) || discovery.SamePath(records, filepath.Join(root, "missing")) {
		t.Fatal("same-file comparison did not preserve identity")
	}
	caseAlias := filepath.Join(root, "records")
	if _, err := os.Stat(caseAlias); err == nil && !discovery.SamePath(records, caseAlias) {
		t.Fatal("case-insensitive filesystem identity was lost")
	}
	missing := filepath.Join(root, "missing")
	if !discovery.SamePath(missing, missing) {
		t.Fatal("identical missing canonical paths did not match")
	}
}
