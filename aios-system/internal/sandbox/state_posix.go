//go:build linux || darwin

package sandbox

import (
	"os"
	"syscall"
)

// exitCodeFromWaitStatus reports the exit code from a
// ProcessState. The function returns (code, true) when
// the process exited normally. A signal-killed process
// returns (-1, false); the caller falls back to the wait
// error.
func exitCodeFromWaitStatus(state *os.ProcessState) (int, bool) {
	if state == nil {
		return 0, false
	}
	ws, ok := state.Sys().(syscall.WaitStatus)
	if !ok {
		return 0, false
	}
	if ws.Exited() {
		return ws.ExitStatus(), true
	}
	return -1, false
}

// detectResourceLimit inspects the ProcessState and
// reports whether the process was killed by a signal that
// indicates a resource-limit violation. The function
// returns ("field", true) on a violation and ("", false)
// otherwise.
//
// SIGXCPU is the kernel's signal for a CPU-limit
// violation. SIGKILL is ambiguous — it is also sent by an
// operator or by the OOM killer — but in the sandbox
// context the runner assumes it came from a memory or
// process violation.
func detectResourceLimit(state *os.ProcessState) (string, bool) {
	if state == nil {
		return "", false
	}
	ws, ok := state.Sys().(syscall.WaitStatus)
	if !ok {
		return "", false
	}
	if !ws.Signaled() {
		return "", false
	}
	switch ws.Signal() {
	case syscall.SIGXCPU:
		return "cpu_seconds", true
	case syscall.SIGKILL:
		return "memory_bytes", true
	}
	return "", false
}

// dropPrivileges is the POSIX implementation. The function
// calls Setgid when the GID names a user to drop to, then
// Setuid when the UID names one. The setgid must come first
// because a non-root process cannot setgid after Setuid.
//
// # Only a positive id is a drop request
//
// The zero value is not "drop to root". A sandbox that runs a
// child as root is not a sandbox, and uid 0 is what a
// zero-valued struct carries by accident. So the test is
// "greater than zero", not "not negative".
//
// This is also what fixed the Linux failure. The old test was
// "not negative", so the zero value reached Setgid(0) and
// failed with EPERM for any process that was not already root,
// and the failure was fatal -- the runner refused to start the
// child, so every subprocess the agent tried to run was
// refused with "sandbox: limits: gid: operation not
// permitted". The defect was invisible on Windows, where this
// function is a no-op.
//
// The function returns nil when no privilege drop was
// requested.
func dropPrivileges(l *ResourceLimits) error {
	if l == nil {
		return nil
	}
	if l.GID > 0 {
		if err := syscall.Setgid(l.GID); err != nil {
			return &LimitsError{Field: "gid", Reason: err.Error()}
		}
	}
	if l.UID > 0 {
		if err := syscall.Setuid(l.UID); err != nil {
			return &LimitsError{Field: "uid", Reason: err.Error()}
		}
	}
	return nil
}
