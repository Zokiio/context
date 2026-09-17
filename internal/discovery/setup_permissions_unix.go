//go:build darwin || linux

package discovery

import (
	"fmt"
	"os"
	"syscall"
)

// preserveSetupPermissions restores the captured permissions on the replacement.
// It never reads the source file for desired values. Ownership changes can clear
// mode bits, so ownership comes first. The caller must compare a fresh permission
// snapshot with the captured snapshot before rename.
func preserveSetupPermissions(tempPath string, before setupPermissionSnapshot) error {
	if before.info == nil {
		return os.Chmod(tempPath, 0o644)
	}
	if before.inspectionErr != nil {
		return fmt.Errorf("cannot inspect original permissions: %w", before.inspectionErr)
	}
	want, ok := before.info.Sys().(*syscall.Stat_t)
	if !ok {
		return fmt.Errorf("cannot inspect original ownership")
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
	if err := os.Chmod(tempPath, before.info.Mode()); err != nil {
		return fmt.Errorf("preserve mode: %w", err)
	}
	if err := preserveSetupExtendedPermissions(tempPath, before.extended); err != nil {
		return err
	}
	temporary, err = os.Stat(tempPath)
	if err != nil {
		return err
	}
	got, ok = temporary.Sys().(*syscall.Stat_t)
	if !ok || got.Uid != want.Uid || got.Gid != want.Gid || temporary.Mode() != before.info.Mode() {
		return fmt.Errorf("temporary file did not retain the original owner, group, and mode")
	}
	return nil
}
