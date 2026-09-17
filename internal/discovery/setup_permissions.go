package discovery

import "os"

// A permission snapshot belongs to the observed document. Inspection failures
// remain available for Apply to report, without preventing a no-op or dry-run.
type setupPermissionSnapshot struct {
	info          os.FileInfo
	extended      setupExtendedPermissions
	inspectionErr error
}

func captureSetupPermissions(path string, info os.FileInfo) setupPermissionSnapshot {
	if info == nil {
		return setupPermissionSnapshot{}
	}
	extended, err := captureSetupExtendedPermissions(path, info)
	return setupPermissionSnapshot{info: info, extended: extended, inspectionErr: err}
}

func sameSetupPermissionSnapshots(left, right setupPermissionSnapshot) bool {
	if left.info == nil || right.info == nil {
		return left.info == nil && right.info == nil
	}
	if left.info.Mode() != right.info.Mode() || !sameSetupPermissions(left.info, right.info) {
		return false
	}
	if left.inspectionErr != nil || right.inspectionErr != nil {
		// Unknown permissions cannot be restored. Treat two unavailable reads
		// as equal for no-op setup; preservation will refuse any replacement.
		return left.inspectionErr != nil && right.inspectionErr != nil
	}
	return sameSetupExtendedPermissions(left.extended, right.extended)
}
