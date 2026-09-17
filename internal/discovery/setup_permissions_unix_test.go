//go:build darwin || linux

package discovery

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func setupDifferentGroup(t *testing.T, path string) uint32 {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	current := info.Sys().(*syscall.Stat_t).Gid
	groups, err := os.Getgroups()
	if err != nil {
		t.Fatal(err)
	}
	if os.Geteuid() == 0 {
		groups = append(groups, int(current)+1)
	}
	for _, group := range groups {
		if uint32(group) != current {
			if err := os.Chown(path, -1, group); err != nil {
				t.Fatal(err)
			}
			return uint32(group)
		}
	}
	t.Skip("requires a second permitted group")
	return 0
}

func TestSetupApplyPreservesFileGroup(t *testing.T) {
	request, path := setupWriteFixture(t, true)
	want := setupDifferentGroup(t, path)
	plan := setupWritePlan(t, request)
	if err := plan.Apply(context.Background()); err != nil {
		t.Fatal(err)
	}
	after, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := after.Sys().(*syscall.Stat_t).Gid; got != want {
		t.Fatalf("setup changed file group from %d to %d", want, got)
	}
}

func setupPermissionTemp(t *testing.T, path string) string {
	t.Helper()
	file, err := os.CreateTemp(filepath.Dir(path), ".permissions-test-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Remove(file.Name()) })
	if _, err := file.WriteString("replacement document\n"); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	return file.Name()
}

func setupPermissionStat(t *testing.T, path string) os.FileInfo {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	return info
}

func TestPreserveSetupPermissionsRetainsOwnershipAndSpecialMode(t *testing.T) {
	_, path := setupWriteFixture(t, true)
	setupDifferentGroup(t, path)
	temporary := setupPermissionTemp(t, path)
	mode := os.FileMode(0o640) | os.ModeSetuid | os.ModeSetgid
	if err := os.Chmod(path, mode); err != nil {
		t.Fatal(err)
	}
	before := setupPermissionStat(t, path)
	data := setupWriteRead(t, path)
	if err := preserveSetupPermissions(temporary, captureSetupPermissions(path, before)); err != nil {
		t.Fatal(err)
	}
	after := setupPermissionStat(t, temporary)
	want, got := before.Sys().(*syscall.Stat_t), after.Sys().(*syscall.Stat_t)
	if got.Uid != want.Uid || got.Gid != want.Gid || after.Mode() != before.Mode() {
		t.Fatalf("owner/group/mode = %d/%d/%v; want %d/%d/%v", got.Uid, got.Gid, after.Mode(), want.Uid, want.Gid, before.Mode())
	}
	if !bytes.Equal(data, setupWriteRead(t, path)) {
		t.Fatal("permission preservation modified the source document")
	}
}

func TestPreserveSetupPermissionsRetainsDifferentOwner(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip("changing a fixture's owner requires root")
	}
	_, path := setupWriteFixture(t, true)
	temporary := setupPermissionTemp(t, path)
	if err := os.Chown(path, 1, -1); err != nil {
		t.Fatal(err)
	}
	if err := preserveSetupPermissions(temporary, captureSetupPermissions(path, setupPermissionStat(t, path))); err != nil {
		t.Fatal(err)
	}
	if got := setupPermissionStat(t, temporary).Sys().(*syscall.Stat_t).Uid; got != 1 {
		t.Fatalf("owner = %d, want 1", got)
	}
}

func TestSetupPermissionSnapshotRetainsOriginalGroup(t *testing.T) {
	_, path := setupWriteFixture(t, true)
	temporary := setupPermissionTemp(t, path)
	before := setupPermissionStat(t, path)
	snapshot := captureSetupPermissions(path, before)
	group := setupDifferentGroup(t, path)
	if sameSetupPermissionSnapshots(snapshot, captureSetupPermissions(path, setupPermissionStat(t, path))) {
		t.Fatal("snapshot comparison ignored a concurrent group change")
	}
	if err := preserveSetupPermissions(temporary, snapshot); err != nil {
		t.Fatal(err)
	}
	if got, want := setupPermissionStat(t, temporary).Sys().(*syscall.Stat_t).Gid, before.Sys().(*syscall.Stat_t).Gid; got != want {
		t.Fatalf("temporary group = %d, want original %d", got, want)
	}
	if got := setupPermissionStat(t, path).Sys().(*syscall.Stat_t).Gid; got != group {
		t.Fatalf("concurrent group change was overwritten: %d, want %d", got, group)
	}
}

func TestSetupPermissionsDetectsChangeTime(t *testing.T) {
	_, path := setupWriteFixture(t, true)
	before := setupPermissionStat(t, path)
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, before.Mode()); err != nil {
		t.Fatal(err)
	}
	after := setupPermissionStat(t, path)
	if before.Mode() != after.Mode() || !before.ModTime().Equal(after.ModTime()) {
		t.Fatal("fixture changed mode or mtime")
	}
	if sameSetupPermissions(before, after) {
		t.Skip("filesystem change time did not advance; exact snapshot tests cover ACL changes within one clock tick")
	}
}

func TestPreserveSetupPermissionsUsesDefaultForNewFile(t *testing.T) {
	_, path := setupWriteFixture(t, true)
	temporary := setupPermissionTemp(t, path)
	if err := preserveSetupPermissions(temporary, setupPermissionSnapshot{}); err != nil {
		t.Fatal(err)
	}
	if got := setupPermissionStat(t, temporary).Mode().Perm(); got != 0o644 {
		t.Fatalf("new file mode = %v, want 0644", got)
	}
}

func TestSetupPermissionSnapshotRefusesUnknownPermissions(t *testing.T) {
	_, path := setupWriteFixture(t, true)
	temporary := setupPermissionTemp(t, path)
	before := captureSetupPermissions(path, setupPermissionStat(t, path))
	unavailable := before
	unavailable.inspectionErr = errors.New("ACL inspection denied")
	if !sameSetupPermissionSnapshots(unavailable, unavailable) {
		t.Fatal("unavailable permissions would prevent read-only no-op setup")
	}
	if sameSetupPermissionSnapshots(before, unavailable) {
		t.Fatal("a failed observation matched known permissions")
	}
	if err := preserveSetupPermissions(temporary, unavailable); err == nil || !strings.Contains(err.Error(), "ACL inspection denied") {
		t.Fatalf("unknown permissions were not rejected: %v", err)
	}
}
