//go:build linux || darwin

package sandbox

import (
	"os"
	"syscall"
)

// processRunning reports whether the process is still alive.
//
// It uses the signal-0 probe: sending signal 0 to a running
// process checks that the process exists and that the caller has
// permission to signal it, without delivering anything.
//
// A process that exists but belongs to another user returns
// EPERM rather than ESRCH. The check treats EPERM as alive,
// because the process is genuinely still running.
func processRunning(proc *os.Process) bool {
	if proc == nil {
		return false
	}
	err := syscall.Kill(proc.Pid, 0)
	if err == nil {
		return true
	}
	return err == syscall.EPERM
}

// isWindowsShell reports whether the platform's shell is
// cmd.exe.
func isWindowsShell() bool {
	return false
}
