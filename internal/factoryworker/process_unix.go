//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package factoryworker

import (
	"errors"
	"os/exec"
	"syscall"
)

func platformSupport() error { return nil }
func configureProcess(cmd *exec.Cmd) error {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return nil
}

// Only the currently owned command is accepted. A retained PID cannot be passed.
func stopOwnedProcess(cmd *exec.Cmd, force bool) error {
	signal := syscall.SIGTERM
	if force {
		signal = syscall.SIGKILL
	}
	return syscall.Kill(-cmd.Process.Pid, signal)
}
func ownedGroupAlive(cmd *exec.Cmd) bool {
	return !errors.Is(syscall.Kill(-cmd.Process.Pid, 0), syscall.ESRCH)
}
