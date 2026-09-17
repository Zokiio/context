package discovery

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
)

// The filesystem's volume and file IDs provide one order across path aliases.
// Retaining sidecars keeps that identity stable across cooperating writers.
type setupLockID [3]uint64

func orderSetupLocks(paths []string) ([]string, error) {
	type lock struct {
		path string
		id   setupLockID
	}
	locks := make([]lock, 0, len(paths))
	for _, path := range paths {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return nil, fmt.Errorf("create setup lock directory %s: %w", filepath.Dir(path), err)
		}
		id, err := identifySetupLock(path)
		if err != nil {
			return nil, fmt.Errorf("lock setup configuration %s: %w", path, err)
		}
		locks = append(locks, lock{path: path, id: id})
	}
	// Path spelling cannot define this order on case-insensitive filesystems.
	// Alias deduplication also avoids acquiring the same exclusive lock twice.
	slices.SortFunc(locks, func(left, right lock) int { return slices.Compare(left.id[:], right.id[:]) })
	locks = slices.CompactFunc(locks, func(left, right lock) bool { return left.id == right.id })
	ordered := make([]string, 0, len(locks))
	for _, lock := range locks {
		ordered = append(ordered, lock.path)
	}
	return ordered, nil
}

func identifySetupLock(path string) (setupLockID, error) {
	// Inspect before opening so special files cannot block lock preparation.
	if err := checkRegularFile(path); err != nil && !os.IsNotExist(err) {
		return setupLockID{}, err
	}
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o666)
	if err != nil {
		return setupLockID{}, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return setupLockID{}, err
	}
	if !info.Mode().IsRegular() {
		return setupLockID{}, fmt.Errorf("%s is not a regular file", path)
	}
	return setupLockIdentity(file, info)
}
