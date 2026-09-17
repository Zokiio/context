//go:build !darwin && !linux

package discovery

import (
	"fmt"
	"os"
	"runtime"
)

func sameSetupPermissions(_, _ os.FileInfo) bool {
	// sameSetupFile still compares portable mode bits. Do not compare opaque
	// system metadata: reading the document can change its access time.
	// Replacement fails below, so unknown permissions cannot be overwritten.
	return true
}

type setupExtendedPermissions struct{}

func captureSetupExtendedPermissions(_ string, _ os.FileInfo) (setupExtendedPermissions, error) {
	return setupExtendedPermissions{}, fmt.Errorf("preserving existing file permissions is unsupported on %s; configuration was not replaced", runtime.GOOS)
}

func sameSetupExtendedPermissions(_, _ setupExtendedPermissions) bool { return true }

// Keep creation available on other platforms, but do not replace an existing
// file until its owner and extended permissions can be preserved and verified.
func preserveSetupPermissions(tempPath string, before setupPermissionSnapshot) error {
	if before.info != nil {
		return fmt.Errorf("preserving existing file permissions is unsupported on %s; configuration was not replaced", runtime.GOOS)
	}
	return os.Chmod(tempPath, 0o644)
}
