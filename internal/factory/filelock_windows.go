//go:build windows

package factory

import (
	"errors"
	"os"
)

var errFactoryFileLocksUnsupported = errors.New("factory runtime file locking is not supported on Windows")

func lockFile(_ *os.File, _ bool) error { return errFactoryFileLocksUnsupported }
func unlockFile(_ *os.File) error       { return nil }
func isLockContended(error) bool        { return false }
