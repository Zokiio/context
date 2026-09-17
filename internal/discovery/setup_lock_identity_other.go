//go:build !darwin && !linux && !windows

package discovery

import (
	"fmt"
	"os"
	"runtime"
)

func setupLockIdentity(_ *os.File, _ os.FileInfo) (setupLockID, error) {
	return setupLockID{}, fmt.Errorf("stable setup lock identities are unsupported on %s; configuration was not written", runtime.GOOS)
}
