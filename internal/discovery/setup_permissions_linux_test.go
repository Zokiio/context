package discovery

import (
	"bytes"
	"encoding/binary"
	"errors"
	"os"
	"strings"
	"testing"

	"golang.org/x/sys/unix"
)

func setupLinuxACL(t *testing.T, path string) []byte {
	t.Helper()
	// Linux's posix_acl_xattr_header and entries use little-endian values.
	entries := []struct {
		tag, permissions uint16
		id               uint32
	}{
		{0x01, 6, 0xffffffff}, // ACL_USER_OBJ
		{0x02, 4, 65534},      // ACL_USER
		{0x04, 4, 0xffffffff}, // ACL_GROUP_OBJ
		{0x10, 4, 0xffffffff}, // ACL_MASK
		{0x20, 0, 0xffffffff}, // ACL_OTHER
	}
	acl := make([]byte, 4+8*len(entries))
	binary.LittleEndian.PutUint32(acl[:4], 2)
	for i, entry := range entries {
		offset := 4 + i*8
		binary.LittleEndian.PutUint16(acl[offset:], entry.tag)
		binary.LittleEndian.PutUint16(acl[offset+2:], entry.permissions)
		binary.LittleEndian.PutUint32(acl[offset+4:], entry.id)
	}
	if err := unix.Setxattr(path, setupAccessACL, acl, 0); err != nil {
		if errors.Is(err, unix.ENOTSUP) {
			t.Skip("test filesystem does not support POSIX ACLs")
		}
		t.Fatal(err)
	}
	return acl
}

func TestPreserveSetupPermissionsCopiesAndRemovesLinuxACL(t *testing.T) {
	for _, sourceACL := range []bool{false, true} {
		t.Run(map[bool]string{false: "remove inherited ACL", true: "copy source ACL"}[sourceACL], func(t *testing.T) {
			_, path := setupWriteFixture(t, true)
			temporary := setupPermissionTemp(t, path)
			var want []byte
			if sourceACL {
				want = setupLinuxACL(t, path)
			} else {
				setupLinuxACL(t, temporary)
			}
			if err := preserveSetupPermissions(path, temporary, setupPermissionStat(t, path)); err != nil {
				t.Fatal(err)
			}
			buffer := make([]byte, 1024)
			n, err := unix.Getxattr(temporary, setupAccessACL, buffer)
			if sourceACL {
				if err != nil || !bytes.Equal(buffer[:max(n, 0)], want) {
					t.Fatalf("temporary ACL = %x, %v; want %x", buffer[:max(n, 0)], err, want)
				}
			} else if !errors.Is(err, unix.ENODATA) {
				t.Fatalf("inherited ACL remains: %v", err)
			}
		})
	}
}

func TestSetupPermissionsDetectsLinuxACLEdit(t *testing.T) {
	_, path := setupWriteFixture(t, true)
	before := setupPermissionStat(t, path)
	setupLinuxACL(t, path)
	after := setupPermissionStat(t, path)
	if before.Mode() != after.Mode() || !before.ModTime().Equal(after.ModTime()) {
		t.Fatal("fixture changed mode or mtime")
	}
	if sameSetupPermissions(before, after) {
		t.Fatal("ACL-only edit did not invalidate the snapshot")
	}
}

func TestPreserveSetupPermissionsRefusesCapabilityLoss(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip("setting a capability fixture requires root")
	}
	_, path := setupWriteFixture(t, true)
	temporary := setupPermissionTemp(t, path)
	capability := make([]byte, 20)
	binary.LittleEndian.PutUint32(capability[:4], 0x02000000)
	binary.LittleEndian.PutUint32(capability[4:8], 1) // CAP_CHOWN permitted
	if err := unix.Setxattr(path, "security.capability", capability, 0); err != nil {
		if errors.Is(err, unix.ENOTSUP) || errors.Is(err, unix.EPERM) {
			t.Skipf("capability fixture unavailable: %v", err)
		}
		t.Fatal(err)
	}
	if err := preserveSetupPermissions(path, temporary, setupPermissionStat(t, path)); err == nil || !strings.Contains(err.Error(), "cannot preserve permission attribute") {
		t.Fatalf("capability loss = %v", err)
	}
}
