//go:build darwin || linux

package discovery

import (
	"fmt"
	"os"
	"syscall"
)

func setupLockIdentity(_ *os.File, info os.FileInfo) (setupLockID, error) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return setupLockID{}, fmt.Errorf("filesystem did not provide a setup lock identity")
	}
	return setupLockID{uint64(stat.Dev), 0, uint64(stat.Ino)}, nil
}
