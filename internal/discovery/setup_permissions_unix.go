//go:build darwin || linux

package discovery

import (
	"fmt"
	"os"
	"syscall"
)

// preserveSetupPermissions prepares the replacement before the caller syncs and
// renames it. Ownership changes can clear mode bits, so ownership comes first.
// Unsupported permissions or failed verification leave the source untouched.
// The caller must recheck its snapshot after this function and before rename.
func preserveSetupPermissions(sourcePath, tempPath string, before os.FileInfo) error {
	if before == nil {
		return os.Chmod(tempPath, 0o644)
	}
	current, err := os.Stat(sourcePath)
	if err != nil {
		return err
	}
	if !sameSetupFile(before, current) || !sameSetupPermissions(before, current) {
		return setupChanged(sourcePath)
	}
	want, ok := before.Sys().(*syscall.Stat_t)
	if !ok {
		return fmt.Errorf("cannot inspect ownership of %s", sourcePath)
	}
	temporary, err := os.Stat(tempPath)
	if err != nil {
		return err
	}
	got, ok := temporary.Sys().(*syscall.Stat_t)
	if !ok || !temporary.Mode().IsRegular() {
		return fmt.Errorf("cannot inspect temporary file ownership")
	}
	if got.Uid != want.Uid || got.Gid != want.Gid {
		if err := os.Chown(tempPath, int(want.Uid), int(want.Gid)); err != nil {
			return fmt.Errorf("preserve owner and group: %w", err)
		}
	}
	if err := os.Chmod(tempPath, before.Mode()); err != nil {
		return fmt.Errorf("preserve mode: %w", err)
	}
	if err := preserveSetupExtendedPermissions(sourcePath, tempPath); err != nil {
		return err
	}
	temporary, err = os.Stat(tempPath)
	if err != nil {
		return err
	}
	got, ok = temporary.Sys().(*syscall.Stat_t)
	if !ok || got.Uid != want.Uid || got.Gid != want.Gid || temporary.Mode() != before.Mode() {
		return fmt.Errorf("temporary file did not retain the original owner, group, and mode")
	}
	current, err = os.Stat(sourcePath)
	if err != nil {
		return err
	}
	if !sameSetupFile(before, current) || !sameSetupPermissions(before, current) {
		return setupChanged(sourcePath)
	}
	return nil
}
