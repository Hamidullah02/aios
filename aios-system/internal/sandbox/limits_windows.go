//go:build windows

package sandbox

// saveParentLimits is a no-op on Windows: applyLimits is a
// no-op there, so the parent's rlimits never change and
// there is nothing to save.
func saveParentLimits() {}

// restoreParentLimits is a no-op on Windows.
func restoreParentLimits() {}

// applyLimits is the Windows fallback. syscall.Setrlimit is
// not supported on Windows, so the runner relies on the
// wall-clock timeout to bound the run. The function returns
// nil so the runner can proceed.
func (l ResourceLimits) applyLimits() error {
	return nil
}

// applyChildLimits is a no-op on Windows.
//
// The Job Object is the kernel-enforced boundary there, and it
// already carries the process-count and memory limits: it caps
// the number of processes in the job and the total committed
// memory across it. The Job Object bounds the child without
// touching the parent, which is what the POSIX path uses
// prlimit to achieve.
func applyChildLimits(pid int, l *ResourceLimits) error {
	return nil
}
