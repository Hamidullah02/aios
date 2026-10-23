//go:build windows

package sandbox

import (
	"fmt"
	"os/exec"
	"unsafe"

	"golang.org/x/sys/windows"
)

// configureProcessGroup is a no-op on Windows.
//
// The process-tree containment on Windows is the Job Object, not
// a process group: see startJobObject. Windows has no direct
// equivalent of a POSIX process group that a parent can signal,
// and the Job Object is strictly better -- the kernel tracks
// membership, so a descendant cannot escape by changing its
// parent.
func configureProcessGroup(cmd *exec.Cmd) {}

// createSuspended is the CREATE_SUSPENDED creation flag.
//
// The child is created suspended so the Job Object assignment
// lands before the child runs a single instruction. Without it
// there is a window between CreateProcess returning and the
// assignment call, and a child that spawns a grandchild in that
// window produces a grandchild that is not in the job and
// therefore survives the timeout.
const createSuspended = 0x00000004

// prepareJobAssignment configures the command to be created
// suspended.
//
// The assignment itself happens after cmd.Start, because only
// then does the process exist. The child stays suspended until
// resumeAfterAssignment is called.
func prepareJobAssignment(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &windows.SysProcAttr{}
	}
	cmd.SysProcAttr.CreationFlags |= createSuspended
}

// resumeAfterAssignment resumes the suspended child.
//
// NtResumeProcess is used rather than ResumeThread because it
// takes a process handle, and Go's exec package does not expose
// the primary thread handle. The thread-enumeration route needs
// the thread creation time to pick the primary thread, and the
// toolhelp ThreadEntry32 structure does not carry one.
//
// If the process cannot be resumed the child stays suspended
// forever. That is why the caller kills the process tree when
// this fails: a hung child is worse than a failed one, because
// it would burn the whole timeout.
func resumeAfterAssignment(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return &LimitsError{Field: "job_object", Reason: "process not started"}
	}
	handle, err := windows.OpenProcess(
		windows.PROCESS_SUSPEND_RESUME|windows.PROCESS_QUERY_LIMITED_INFORMATION,
		false,
		uint32(cmd.Process.Pid),
	)
	if err != nil {
		return &LimitsError{Field: "job_object", Reason: err.Error()}
	}
	defer windows.CloseHandle(handle)

	status, _, _ := ntResumeProcess.Call(uintptr(handle))
	// NtResumeProcess returns the NTSTATUS directly in the
	// first return value. The third value is the thread's last
	// error, which is meaningless here: it is only meaningful
	// for functions that signal failure through GetLastError,
	// and it holds a stale value on success. Reading it would
	// report a failure for a resume that actually worked.
	if status != 0 {
		return &LimitsError{Field: "job_object", Reason: fmt.Sprintf("resume failed: NTSTATUS 0x%08x", status)}
	}
	return nil
}

// ntResumeProcess is the ntdll entry point that resumes a
// process suspended with CREATE_SUSPENDED.
//
// It is not exported by the Go standard library or by x/sys, so
// it is resolved here. The library is loaded lazily: the cost is
// paid on the first run, and a machine without ntdll (which
// cannot happen on Windows, but the loader does not know that)
// does not fail the build.
var ntResumeProcess = windows.NewLazySystemDLL("ntdll.dll").NewProc("NtResumeProcess")

// jobObject holds the Windows Job Object that contains a run's
// process tree.
//
// The Job Object is the Windows equivalent of the POSIX process
// group, and it does more: the kernel enforces the memory and
// process-count limits, and it kills every process in the job
// when the job handle is closed. A descendant that spawns a
// grandchild is still in the job, so the whole tree goes.
type jobObject struct {
	handle windows.Handle
}

// startJobObject creates a Job Object and applies the limits.
//
// The limits are the same ones the POSIX path applies through
// rlimits, expressed in the Job Object's own vocabulary:
//
//   - Processes: JOB_OBJECT_LIMIT_ACTIVE_PROCESS caps the number
//     of processes in the job.
//   - MemoryBytes: JOB_OBJECT_LIMIT_JOB_MEMORY caps the total
//     committed memory across the job.
//   - KILL_ON_JOB_CLOSE: when the handle is closed, every
//     process in the job is terminated. This is the property
//     that makes the timeout reliable -- the runner closes the
//     handle and the whole tree dies, whether or not the runner
//     got around to signalling anything.
//
// The function returns nil, nil when the limits ask for nothing
// that a Job Object provides, so the caller can skip the
// assignment entirely.
func startJobObject(limits *ResourceLimits) (*jobObject, error) {
	handle, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return nil, &LimitsError{Field: "job_object", Reason: err.Error()}
	}
	job := &jobObject{handle: handle}

	info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
	info.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	if limits != nil {
		if limits.Processes > 0 {
			info.BasicLimitInformation.ActiveProcessLimit = uint32(limits.Processes)
			info.BasicLimitInformation.LimitFlags |= windows.JOB_OBJECT_LIMIT_ACTIVE_PROCESS
		}
		if limits.MemoryBytes > 0 {
			info.JobMemoryLimit = uintptr(limits.MemoryBytes)
			info.BasicLimitInformation.LimitFlags |= windows.JOB_OBJECT_LIMIT_JOB_MEMORY
		}
	}

	_, err = windows.SetInformationJobObject(
		handle,
		windows.JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&info)),
		uint32(unsafe.Sizeof(info)),
	)
	if err != nil {
		windows.CloseHandle(handle)
		return nil, &LimitsError{Field: "job_object", Reason: err.Error()}
	}
	return job, nil
}

// assignProcess puts the child into the Job Object.
//
// The call must happen before the child starts running, or the
// child can spawn a grandchild that escapes the job before the
// assignment lands. Go's exec package does not offer a hook
// between fork and exec, so the child is started suspended and
// resumed after the assignment.
//
// The process handle is opened from the pid rather than taken
// from the os.Process struct: Go does not expose the underlying
// handle, and AssignProcessToJobObject needs a real handle, not
// a pid.
func (j *jobObject) assignProcess(cmd *exec.Cmd) error {
	if j == nil || j.handle == 0 {
		return nil
	}
	if cmd.Process == nil {
		return &LimitsError{Field: "job_object", Reason: "process not started"}
	}
	// PROCESS_SET_QUOTA is what AssignProcessToJobObject needs;
	// PROCESS_TERMINATE lets the job kill the process.
	handle, err := windows.OpenProcess(
		windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE,
		false,
		uint32(cmd.Process.Pid),
	)
	if err != nil {
		return &LimitsError{Field: "job_object", Reason: err.Error()}
	}
	defer windows.CloseHandle(handle)
	if err := windows.AssignProcessToJobObject(j.handle, handle); err != nil {
		return &LimitsError{Field: "job_object", Reason: err.Error()}
	}
	return nil
}

// killProcessTree kills every process in the job.
//
// TerminateJobObject is the call: it terminates all the
// processes associated with the job. The subsequent close of the
// handle is what makes the kill stick, because
// KILL_ON_JOB_CLOSE would otherwise be the only backstop.
func killProcessTree(cmd *exec.Cmd, job *jobObject) {
	if job != nil && job.handle != 0 {
		_ = windows.TerminateJobObject(job.handle, 1)
	}
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
}

// close releases the job handle.
//
// Closing the handle with KILL_ON_JOB_CLOSE set terminates any
// process still in the job, so this is also the safety net for
// a run that finished normally but left a background child
// behind.
func (j *jobObject) close() {
	if j == nil || j.handle == 0 {
		return
	}
	_ = windows.CloseHandle(j.handle)
	j.handle = 0
}
