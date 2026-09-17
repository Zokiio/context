package discovery

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

func sameSetupPermissions(left, right os.FileInfo) bool {
	l, lok := left.Sys().(*syscall.Stat_t)
	r, rok := right.Sys().(*syscall.Stat_t)
	// ACL edits change ctime even when mode and mtime stay the same.
	return lok && rok && l.Uid == r.Uid && l.Gid == r.Gid && l.Ctimespec == r.Ctimespec && l.Flags == r.Flags
}

func preserveSetupExtendedPermissions(sourcePath, tempPath string) error {
	info, err := os.Stat(sourcePath)
	if err != nil {
		return err
	}
	const securityFlags = unix.UF_IMMUTABLE | unix.UF_APPEND | unix.UF_DATAVAULT | unix.SF_IMMUTABLE | unix.SF_APPEND | unix.SF_NOUNLINK | unix.SF_RESTRICTED
	if info.Sys().(*syscall.Stat_t).Flags&securityFlags != 0 {
		return fmt.Errorf("cannot preserve macOS file protection flags during replacement")
	}
	acl, err := readSetupACL(sourcePath)
	if err != nil {
		return fmt.Errorf("read original ACL: %w", err)
	}
	current, err := readSetupACL(tempPath)
	if err != nil {
		return fmt.Errorf("read temporary ACL: %w", err)
	}
	if !bytes.Equal(acl, current) {
		if err := writeSetupACL(tempPath, acl); err != nil {
			return fmt.Errorf("preserve ACL: %w", err)
		}
	}
	current, err = readSetupACL(tempPath)
	if err != nil {
		return fmt.Errorf("verify temporary ACL: %w", err)
	}
	if !bytes.Equal(acl, current) {
		return fmt.Errorf("temporary file did not retain the original ACL")
	}
	return nil
}

// Darwin exposes ACLs through getattrlist, not through its protected Security
// xattr. Go has no getattrlist wrapper, but its syscall entry works without cgo.
func readSetupACL(path string) ([]byte, error) {
	cpath, err := syscall.BytePtrFromString(path)
	if err != nil {
		return nil, err
	}
	attrs := unix.Attrlist{Bitmapcount: unix.ATTR_BIT_MAP_COUNT, Commonattr: unix.ATTR_CMN_EXTENDED_SECURITY}
	buffer := make([]byte, 64*1024)
	_, _, errno := syscall.Syscall6(unix.SYS_GETATTRLIST, uintptr(unsafe.Pointer(cpath)), uintptr(unsafe.Pointer(&attrs)),
		uintptr(unsafe.Pointer(&buffer[0])), uintptr(len(buffer)), unix.FSOPT_REPORT_FULLSIZE, 0)
	if errno != 0 {
		return nil, errno
	}
	return decodeSetupACL(buffer)
}

// The SDK's sys/attr.h and sys/kauth.h define a length, attrreference, and
// kauth_filesec: a 44-byte header followed by at most 128 24-byte ACEs.
func decodeSetupACL(buffer []byte) ([]byte, error) {
	invalid := fmt.Errorf("invalid or unsupported macOS ACL attributes")
	if len(buffer) < 12 {
		return nil, invalid
	}
	size := uint64(binary.NativeEndian.Uint32(buffer[:4]))
	start := int64(4) + int64(int32(binary.NativeEndian.Uint32(buffer[4:8])))
	length := uint64(binary.NativeEndian.Uint32(buffer[8:12]))
	if size < 12 || size > uint64(len(buffer)) || start < 12 || uint64(start)+length > size {
		return nil, invalid
	}
	if length == 0 {
		return nil, nil
	}
	acl := buffer[start : uint64(start)+length]
	if len(acl) < 44 || binary.NativeEndian.Uint32(acl[:4]) != 0x012cc16d {
		return nil, invalid
	}
	count := binary.NativeEndian.Uint32(acl[36:40])
	if count == 0xffffffff && len(acl) == 44 {
		return nil, nil
	}
	if count > 128 || uint64(len(acl)) != 44+24*uint64(count) {
		return nil, invalid
	}
	return bytes.Clone(acl), nil
}

func writeSetupACL(path string, acl []byte) error {
	if acl == nil {
		// KAUTH_FILESEC_NOACL removes an inherited ACL. An empty ACL has
		// different permission semantics and must not replace this marker.
		acl = make([]byte, 44)
		binary.NativeEndian.PutUint32(acl[:4], 0x012cc16d)
		binary.NativeEndian.PutUint32(acl[36:40], 0xffffffff)
	}
	buffer := make([]byte, 8+len(acl))
	binary.NativeEndian.PutUint32(buffer[:4], 8)
	binary.NativeEndian.PutUint32(buffer[4:8], uint32(len(acl)))
	copy(buffer[8:], acl)
	attrs := unix.Attrlist{Bitmapcount: unix.ATTR_BIT_MAP_COUNT, Commonattr: unix.ATTR_CMN_EXTENDED_SECURITY}
	return unix.Setattrlist(path, &attrs, buffer, 0)
}
