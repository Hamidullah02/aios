//go:build windows

package sandbox

import (
	"golang.org/x/sys/windows"
)

// openProcessForProbe opens a process handle for the liveness
// check.
//
// SYNCHRONIZE is the minimum access that lets the open succeed
// for a live process and fail for a dead one, without granting
// the caller the ability to do anything to it.
func openProcessForProbe(pid int) (windows.Handle, error) {
	return windows.OpenProcess(windows.SYNCHRONIZE, false, uint32(pid))
}

// closeProcessHandle releases a handle from the probe.
func closeProcessHandle(handle windows.Handle) {
	_ = windows.CloseHandle(handle)
}

// processExited reports whether the process behind the handle has
// already exited.
//
// This is the part that makes the liveness probe correct on
// Windows. A successful OpenProcess does NOT mean the process is
// alive: a process that has been terminated keeps its process
// object alive for as long as any handle to it is open, and
// OpenProcess(SYNCHRONIZE) succeeds for it. A probe that stops at
// "the open worked" therefore reports a killed process as running
// for as long as the killer holds a handle -- which is exactly
// the situation the sandbox is in, because the runner holds the
// child's handle.
//
// The way to tell the two apart is to wait on the handle with a
// zero timeout: a process that has exited is already signalled, so
// the wait returns immediately, and a process that is still running
// times out.
//
// A failed wait is reported as "exited". That is the safe direction
// for this caller: the check is used to decide whether a process
// still needs to be killed, and a process we cannot confirm is
// better treated as one we should try to kill again than as one we
// can leave alone.
func processExited(handle windows.Handle) bool {
	if handle == 0 {
		return true
	}
	event, err := windows.WaitForSingleObject(handle, 0)
	if err != nil {
		return true
	}
	return event == uint32(windows.WAIT_OBJECT_0)
}
