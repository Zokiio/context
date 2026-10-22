//go:build !aix && !darwin && !dragonfly && !freebsd && !linux && !netbsd && !openbsd && !solaris

package factoryworker

import (
	"errors"
	"os/exec"
)

func platformSupport() error {
	return errors.New("factory process workers require a supported Unix host; service clients and reader commands remain available")
}
func configureProcess(*exec.Cmd) error       { return platformSupport() }
func stopOwnedProcess(*exec.Cmd, bool) error { return platformSupport() }
func ownedGroupAlive(*exec.Cmd) bool         { return true }
