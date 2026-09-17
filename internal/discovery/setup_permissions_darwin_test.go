package discovery

import (
	"bytes"
	"context"
	"encoding/binary"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
)

func setupAddDarwinACL(t *testing.T, path string) {
	t.Helper()
	if output, err := exec.Command("chmod", "+a", "everyone deny write", path).CombinedOutput(); err != nil {
		t.Fatalf("create ACL fixture: %v: %s", err, output)
	}
}

func setupDarwinACLText(t *testing.T, path string) string {
	t.Helper()
	output, err := exec.Command("ls", "-le", path).CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}
	_, acl, _ := strings.Cut(string(output), "\n")
	return acl
}

func TestSetupApplyPreservesDarwinACL(t *testing.T) {
	request, path := setupWriteFixture(t, true)
	setupAddDarwinACL(t, path)
	want := setupDarwinACLText(t, path)
	if want == "" {
		t.Fatal("fixture has no ACL")
	}
	plan := setupWritePlan(t, request)
	if err := plan.Apply(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := setupDarwinACLText(t, path); got != want {
		t.Fatalf("setup changed ACL: got %q, want %q", got, want)
	}
}

func TestPreserveSetupPermissionsCopiesAndRemovesDarwinACL(t *testing.T) {
	for _, sourceACL := range []bool{false, true} {
		t.Run(map[bool]string{false: "remove inherited ACL", true: "copy source ACL"}[sourceACL], func(t *testing.T) {
			_, path := setupWriteFixture(t, true)
			temporary := setupPermissionTemp(t, path)
			if sourceACL {
				setupAddDarwinACL(t, path)
			} else {
				setupAddDarwinACL(t, temporary)
			}
			want := setupDarwinACLText(t, path)
			if err := preserveSetupPermissions(temporary, captureSetupPermissions(path, setupPermissionStat(t, path))); err != nil {
				t.Fatal(err)
			}
			if got := setupDarwinACLText(t, temporary); got != want {
				t.Fatalf("temporary ACL = %q, want %q", got, want)
			}
		})
	}
}

func TestSetupPermissionSnapshotRetainsOriginalDarwinACL(t *testing.T) {
	_, path := setupWriteFixture(t, true)
	temporary := setupPermissionTemp(t, path)
	setupAddDarwinACL(t, path)
	want := setupDarwinACLText(t, path)
	before := captureSetupPermissions(path, setupPermissionStat(t, path))
	if output, err := exec.Command("chmod", "-a#", "0", path).CombinedOutput(); err != nil {
		t.Fatalf("change source ACL: %v: %s", err, output)
	}
	if err := preserveSetupPermissions(temporary, before); err != nil {
		t.Fatal(err)
	}
	if got := setupDarwinACLText(t, temporary); got != want {
		t.Fatalf("temporary ACL = %q, want original %q", got, want)
	}
	if got := setupDarwinACLText(t, path); got != "" {
		t.Fatalf("source ACL was modified: %q", got)
	}
}

func TestSetupPermissionsDetectsDarwinACLEdit(t *testing.T) {
	_, path := setupWriteFixture(t, true)
	before := setupPermissionStat(t, path)
	snapshot := captureSetupPermissions(path, before)
	setupAddDarwinACL(t, path)
	after := setupPermissionStat(t, path)
	if before.Mode() != after.Mode() || !before.ModTime().Equal(after.ModTime()) {
		t.Fatal("fixture changed mode or mtime")
	}
	if sameSetupPermissionSnapshots(snapshot, captureSetupPermissions(path, after)) {
		t.Fatal("ACL-only edit did not invalidate the snapshot")
	}
}

func TestSetupRejectsDarwinACLRevocationWithinOneClockTick(t *testing.T) {
	for _, duringWrite := range []bool{false, true} {
		t.Run(map[bool]string{false: "before apply", true: "during write"}[duringWrite], func(t *testing.T) {
			request, path := setupWriteFixture(t, true)
			if output, err := exec.Command("chmod", "+a", "everyone allow read", path).CombinedOutput(); err != nil {
				t.Fatalf("grant fixture ACL: %v: %s", err, output)
			}
			plan := setupWritePlan(t, request)
			original := setupWriteRead(t, path)
			revoke := func() {
				if output, err := exec.Command("chmod", "-a#", "0", path).CombinedOutput(); err != nil {
					t.Fatalf("revoke fixture ACL: %v: %s", err, output)
				}
				current := setupPermissionStat(t, path)
				// Model a filesystem clock tick that contains both operations.
				// Permission comparison must work when stat times are identical.
				plan.before.info.Sys().(*syscall.Stat_t).Ctimespec = current.Sys().(*syscall.Stat_t).Ctimespec
			}
			ops := defaultSetupWriteOps()
			if duringWrite {
				ops.createTemp = func(directory, pattern string) (setupTempFile, error) {
					file, err := os.CreateTemp(directory, pattern)
					return &setupFaultFile{file: file, onSync: revoke}, err
				}
			} else {
				revoke()
			}
			if err := plan.apply(context.Background(), ops); err == nil || !strings.Contains(err.Error(), "changed since setup read") {
				t.Errorf("ACL revocation was not rejected: %v", err)
			}
			if acl := setupDarwinACLText(t, path); acl != "" {
				t.Errorf("revoked ACL grant was restored: %q", acl)
			}
			if !bytes.Equal(original, setupWriteRead(t, path)) {
				t.Error("stale plan replaced the original document")
			}
		})
	}
}

func TestDecodeSetupACLBounds(t *testing.T) {
	empty := make([]byte, 56)
	binary.NativeEndian.PutUint32(empty[:4], uint32(len(empty)))
	binary.NativeEndian.PutUint32(empty[4:8], 8)
	binary.NativeEndian.PutUint32(empty[8:12], 44)
	binary.NativeEndian.PutUint32(empty[12:16], 0x012cc16d)
	if acl, err := decodeSetupACL(empty); err != nil || !bytes.Equal(acl, empty[12:]) {
		t.Fatalf("empty ACL must remain distinct from no ACL: %x, %v", acl, err)
	}
	for _, field := range []struct {
		name   string
		offset int
		value  uint32
	}{
		{"short result", 0, 11},
		{"oversized result", 0, 65537},
		{"negative reference", 4, 0xffffffff},
		{"reference beyond buffer", 4, 56},
		{"payload beyond buffer", 8, 45},
		{"short security header", 8, 43},
		{"invalid magic", 12, 0},
		{"entry beyond buffer", 48, 1},
		{"unsupported entry count", 48, 129},
	} {
		t.Run(field.name, func(t *testing.T) {
			buffer := bytes.Clone(empty)
			binary.NativeEndian.PutUint32(buffer[field.offset:field.offset+4], field.value)
			if _, err := decodeSetupACL(buffer); err == nil {
				t.Fatal("invalid kernel attribute buffer was accepted")
			}
		})
	}
	if _, err := decodeSetupACL(empty[:7]); err == nil {
		t.Fatal("truncated attribute reference was accepted")
	}
	noACL := bytes.Clone(empty)
	binary.NativeEndian.PutUint32(noACL[48:52], 0xffffffff)
	if acl, err := decodeSetupACL(noACL); err != nil || acl != nil {
		t.Fatalf("KAUTH_FILESEC_NOACL = %x, %v", acl, err)
	}
}
