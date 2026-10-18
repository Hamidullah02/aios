//go:build windows

package sandbox

import "os"

// exitCodeFromWaitStatus is the Windows fallback. The
// syscall.WaitStatus type is not available on Windows;
// the runner falls back to state.ExitCode().
func exitCodeFromWaitStatus(state *os.ProcessState) (int, bool) {
	if state == nil {
		return 0, false
	}
	if state.Success() {
		return state.ExitCode(), true
	}
	return -1, false
}

// detectResourceLimit is the Windows fallback. The
// function always returns false because the kernel does
// not send SIGXCPU/SIGKILL on Windows.
func detectResourceLimit(state *os.ProcessState) (string, bool) {
	return "", false
}

// dropPrivileges is the Windows fallback. The function is
// a no-op because syscall.Setuid and syscall.Setgid are
// not available on Windows.
func dropPrivileges(l *ResourceLimits) error {
	return nil
}
