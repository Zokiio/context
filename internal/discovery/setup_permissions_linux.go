package discovery

import (
	"bytes"
	"errors"
	"fmt"
	"maps"
	"os"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"
)

func sameSetupPermissions(left, right os.FileInfo) bool {
	l, lok := left.Sys().(*syscall.Stat_t)
	r, rok := right.Sys().(*syscall.Stat_t)
	return lok && rok && l.Uid == r.Uid && l.Gid == r.Gid && l.Ctim == r.Ctim
}

const setupAccessACL = "system.posix_acl_access"

// Copy POSIX access ACLs. Other security, system, or trusted attributes must
// already match on the replacement, otherwise setup fails before replacement.
// This avoids discarding capabilities, security labels, or other ACL formats.
func preserveSetupExtendedPermissions(sourcePath, tempPath string) error {
	want, err := readSetupPermissionXattrs(sourcePath)
	if err != nil {
		return fmt.Errorf("read original extended permissions: %w", err)
	}
	got, err := readSetupPermissionXattrs(tempPath)
	if err != nil {
		return fmt.Errorf("read temporary extended permissions: %w", err)
	}
	for _, attrs := range []map[string][]byte{want, got} {
		for name := range attrs {
			if name == setupAccessACL {
				continue
			}
			w, wpresent := want[name]
			g, gpresent := got[name]
			if wpresent != gpresent || !bytes.Equal(w, g) {
				return fmt.Errorf("cannot preserve permission attribute %q during replacement", name)
			}
		}
	}
	acl, present := want[setupAccessACL]
	if present {
		if err := unix.Setxattr(tempPath, setupAccessACL, acl, 0); err != nil {
			return fmt.Errorf("preserve POSIX ACL: %w", err)
		}
	} else if _, inherited := got[setupAccessACL]; inherited {
		if err := unix.Removexattr(tempPath, setupAccessACL); err != nil {
			return fmt.Errorf("remove inherited POSIX ACL: %w", err)
		}
	}
	got, err = readSetupPermissionXattrs(tempPath)
	if err != nil {
		return fmt.Errorf("verify temporary extended permissions: %w", err)
	}
	if !maps.EqualFunc(want, got, bytes.Equal) {
		return fmt.Errorf("temporary file did not retain the original extended permissions")
	}
	return nil
}

func readSetupPermissionXattrs(path string) (map[string][]byte, error) {
	attrs := make(map[string][]byte)
	size, err := unix.Listxattr(path, nil)
	if errors.Is(err, unix.ENOTSUP) {
		return attrs, nil
	}
	if err != nil {
		return nil, err
	}
	if size < 0 || size > 64*1024 {
		return nil, fmt.Errorf("permission attribute list exceeds supported size")
	}
	if size == 0 {
		return attrs, nil
	}
	names := make([]byte, size)
	n, err := unix.Listxattr(path, names)
	if err != nil {
		return nil, err
	}
	if n < 0 || n > len(names) || (n > 0 && names[n-1] != 0) {
		return nil, fmt.Errorf("invalid permission attribute list")
	}
	for _, name := range strings.Split(string(names[:n]), "\x00") {
		if !strings.HasPrefix(name, "system.") && !strings.HasPrefix(name, "security.") && !strings.HasPrefix(name, "trusted.") {
			continue
		}
		size, err := unix.Getxattr(path, name, nil)
		if err != nil {
			return nil, err
		}
		if size < 0 || size > 64*1024 {
			return nil, fmt.Errorf("permission attribute %q exceeds supported size", name)
		}
		value := make([]byte, size)
		n, err := unix.Getxattr(path, name, value)
		if err != nil {
			return nil, err
		}
		if n < 0 || n > len(value) {
			return nil, fmt.Errorf("invalid permission attribute %q", name)
		}
		attrs[name] = value[:n]
	}
	return attrs, nil
}
