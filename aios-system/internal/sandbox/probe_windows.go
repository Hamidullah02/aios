//go:build windows

package sandbox

import "os"

// processRunning reports whether the process is still alive.
//
// It opens the process and then waits on the handle with a zero
// timeout. The open alone is not enough to answer the question: a
// terminated process keeps its process object alive for as long as
// any handle to it is open, and the open succeeds for it. See
// processExited for why the wait is the part that decides.
func processRunning(proc *os.Process) bool {
	if proc == nil {
		return false
	}
	handle, err := openProcessForProbe(proc.Pid)
	if err != nil {
		return false
	}
	defer closeProcessHandle(handle)
	return !processExited(handle)
}

// isWindowsShell reports whether the platform's shell is
// cmd.exe.
func isWindowsShell() bool {
	return true
}
