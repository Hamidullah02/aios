//go:build linux || darwin

package sandbox

import (
	"os/exec"
	"syscall"
)

// jobObject is the POSIX stub for the Windows Job Object.
//
// The POSIX containment is the process group, not a job, so
// there is nothing for the runner to create or close here. The
// type exists so the shared call site in subprocess.go compiles
// on every platform.
type jobObject struct{}

// startJobObject returns nil on POSIX: the process group is the
// containment mechanism, and the resource limits are applied
// through rlimits.
func startJobObject(limits *ResourceLimits) (*jobObject, error) {
	return nil, nil
}

// prepareJobAssignment is a no-op on POSIX: there is no Job
// Object to assign the child to, so the child does not need to
// be created suspended.
func prepareJobAssignment(cmd *exec.Cmd) {}

// resumeAfterAssignment is a no-op on POSIX: the child was never
// suspended.
func resumeAfterAssignment(cmd *exec.Cmd) error {
	return nil
}

// configureProcessGroup puts the child in its own process
// group.
//
// It is the POSIX half of the process-tree containment. Without
// it, a command that backgrounds work -- `npm run dev &`, a shell
// that spawns a compiler, a test that forks -- leaves processes
// behind when the runner's timeout fires. The runner kills the
// direct child, the grandchildren survive, and they keep running
// with the parent's privileges.
//
// A process group gives the runner one handle on the whole tree:
// killing the group kills every descendant that did not
// deliberately leave it. `Setpgid: true` with `Pgid: 0` makes the
// child its own group leader, so the group id is the child's
// pid.
//
// The function is called before cmd.Start, so the group is
// established at exec time rather than raced afterwards.
func configureProcessGroup(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
	cmd.SysProcAttr.Pgid = 0
}

// killProcessTree kills the child and every process in its
// group.
//
// The direct child is killed with SIGKILL, and so is the group.
// The group kill is what catches the grandchildren: a descendant
// that inherited the group id dies with it.
//
// The errors are deliberately ignored. The caller is already on
// the timeout path, the child is going away regardless, and a
// failed group kill (the group is already empty, or the child
// changed its group) is not something the caller can act on.
//
// The job parameter is unused on POSIX. It exists so the call
// site is the same on every platform.
func killProcessTree(cmd *exec.Cmd, job *jobObject) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	pid := cmd.Process.Pid
	// Kill the group first, then the child. The group kill
	// reaches the grandchildren; the direct kill covers the
	// case where the child changed its own group id.
	_ = syscall.Kill(-pid, syscall.SIGKILL)
	_ = cmd.Process.Kill()
}

// assignProcess is a no-op on POSIX: the process group is the
// containment mechanism, and the child joins it at exec time.
func (j *jobObject) assignProcess(cmd *exec.Cmd) error {
	return nil
}

// close is a no-op on POSIX: there is no job handle to release.
func (j *jobObject) close() {}
