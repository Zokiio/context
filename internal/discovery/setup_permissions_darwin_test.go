package discovery

import (
	"bytes"
	"context"
	"encoding/binary"
	"os/exec"
	"strings"
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
			if err := preserveSetupPermissions(path, temporary, setupPermissionStat(t, path)); err != nil {
				t.Fatal(err)
			}
			if got := setupDarwinACLText(t, temporary); got != want {
				t.Fatalf("temporary ACL = %q, want %q", got, want)
			}
		})
	}
}

func TestSetupPermissionsDetectsDarwinACLEdit(t *testing.T) {
	_, path := setupWriteFixture(t, true)
	before := setupPermissionStat(t, path)
	setupAddDarwinACL(t, path)
	after := setupPermissionStat(t, path)
	if before.Mode() != after.Mode() || !before.ModTime().Equal(after.ModTime()) {
		t.Fatal("fixture changed mode or mtime")
	}
	if sameSetupPermissions(before, after) {
		t.Fatal("ACL-only edit did not invalidate the snapshot")
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
