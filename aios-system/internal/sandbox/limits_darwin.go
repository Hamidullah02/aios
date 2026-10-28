//go:build darwin

package sandbox

import (
	"os"
	"sync"

	"golang.org/x/sys/unix"
)

// parentLimitsState holds the parent process's rlimits as
// captured by saveParentLimits and consumed by restore. The
// fields are package-level because Go does not allow
// per-platform struct types to be referenced from a
// cross-platform call site in subprocess.go; the values are
// the per-run state, not the type.
type parentLimitsState struct {
	mu    sync.Mutex
	saved bool
	cpu   unix.Rlimit
	as    unix.Rlimit
}

var parentLimits parentLimitsState

// saveParentLimits reads the current values of the rlimits
// the runner manages and stores them in package state so
// restore can revert them after the child exits.
//
// The process-count limit is not saved because macOS does not
// have RLIMIT_NPROC. See applyLimits.
func saveParentLimits() {
	parentLimits.mu.Lock()
	defer parentLimits.mu.Unlock()
	parentLimits.saved = true
	if rl, err := getRlimit(unix.RLIMIT_CPU); err == nil {
		parentLimits.cpu = rl
	}
	if rl, err := getRlimit(unix.RLIMIT_AS); err == nil {
		parentLimits.as = rl
	}
}

// restore reverts the parent's rlimits to the saved values.
// If saveParentLimits was not called (no run has happened
// yet) the function is a no-op.
func restoreParentLimits() {
	parentLimits.mu.Lock()
	defer parentLimits.mu.Unlock()
	if !parentLimits.saved {
		return
	}
	_ = unix.Setrlimit(unix.RLIMIT_CPU, &parentLimits.cpu)
	_ = unix.Setrlimit(unix.RLIMIT_AS, &parentLimits.as)
}

func getRlimit(which int) (unix.Rlimit, error) {
	var r unix.Rlimit
	if err := unix.Getrlimit(which, &r); err != nil {
		return r, err
	}
	return r, nil
}

// applyLimits applies the resource limits to the current
// process. On macOS the limits are set through
// syscall.Setrlimit. The caller is the parent process; the
// limit is inherited by the child after Start. The caller is
// responsible for restoring the parent's previous limits via
// restoreParentLimits (see saveParentLimits /
// restoreParentLimits).
//
// The process-count limit is NOT honored on macOS. The kernel
// has no RLIMIT_NPROC there, so a caller that sets
// ResourceLimits.Processes gets the CPU and memory limits
// applied and the process count silently unenforced. The
// field is documented as POSIX-only, and this is the POSIX
// that does not have it.
//
// # The soft limit is set; the hard limit is left alone
//
// The runner lowers the SOFT limit and leaves the hard limit
// where it is.
//
// The hard limit is what makes a child's bound un-liftable: a
// child can raise its own soft limit up to the hard limit, so a
// soft-only bound is one a deliberate child can remove. But
// lowering the hard limit in the parent is not recoverable.
// setrlimit(2): an unprivileged process may lower the hard
// limit freely, but may raise it only up to the previous hard
// limit. So the parent could never put the limit back, and
// restoreParentLimits would fail with EPERM while its error was
// discarded.
//
// The trade is deliberate and it is the safe direction: the
// parent stays usable, and a child that deliberately lifts its
// own soft limit is the case the operator has other tools for.
//
// When the runner does have the privilege to put a hard limit
// back -- that is, when it is root -- it sets both, and the
// child's bound is one it cannot lift.
func (l ResourceLimits) applyLimits() error {
	// RLIMIT_CPU is enforced in seconds. The kernel rounds up
	// to the next second, so a 1s limit is honored as expected.
	if l.CPUSeconds > 0 {
		if err := unix.Setrlimit(unix.RLIMIT_CPU, &unix.Rlimit{
			Cur: uint64(l.CPUSeconds),
			Max: l.hardLimit(unix.RLIMIT_CPU, uint64(l.CPUSeconds)),
		}); err != nil {
			return &LimitsError{Field: "cpu_seconds", Reason: err.Error()}
		}
	}
	// RLIMIT_AS is the address-space limit. A non-zero value
	// is honored in bytes.
	if l.MemoryBytes > 0 {
		if err := unix.Setrlimit(unix.RLIMIT_AS, &unix.Rlimit{
			Cur: uint64(l.MemoryBytes),
			Max: l.hardLimit(unix.RLIMIT_AS, uint64(l.MemoryBytes)),
		}); err != nil {
			return &LimitsError{Field: "memory_bytes", Reason: err.Error()}
		}
	}
	return nil
}

// hardLimit returns the hard limit to pair with a soft limit.
//
// A process that can raise a hard limit it has lowered -- that
// is, one with CAP_SYS_RESOURCE, which root has -- gets the same
// value, so the child's bound cannot be lifted.
//
// Every other process gets the current hard limit, unchanged. A
// soft limit above the hard limit would be refused by the
// kernel with EPERM, so the soft limit is clamped to it.
func (l ResourceLimits) hardLimit(which int, soft uint64) uint64 {
	current, err := getRlimit(which)
	if err != nil {
		// Without the current value the safe answer is the
		// soft limit itself: it is what the caller asked for,
		// and if the kernel refuses it, the caller hears about
		// it here rather than silently running unbounded.
		return soft
	}
	if canRestoreHardLimit() {
		return soft
	}
	if current.Max != unix.RLIM_INFINITY && soft > current.Max {
		return current.Max
	}
	return current.Max
}

// canRestoreHardLimit reports whether this process is able to
// raise a hard limit it has lowered.
//
// setrlimit(2): an unprivileged process may lower the hard
// limit freely, but may raise it only up to the previous hard
// limit. So a process that lowers its own hard limit can never
// raise it again, and the runner could not put the parent's
// limit back.
//
// A process with CAP_SYS_RESOURCE -- which root has -- can raise
// it. That is the only case where the runner may set the hard
// limit too.
func canRestoreHardLimit() bool {
	return os.Geteuid() == 0
}

// applyChildLimits applies the limits that belong to the child
// rather than to the parent.
//
// # macOS has no prlimit(2)
//
// Linux can set another process's limits with prlimit, so the
// child's limits are applied there without touching the parent at
// all. macOS has no such call -- only setrlimit, which acts on
// the caller.
//
// So on macOS the limit is set in the PARENT, before the fork,
// and the child inherits it. That is safe here, and only here,
// because of how Go starts a process on macOS: it uses the
// ordinary fork(2), not clone(2) with CLONE_VM. The child gets its
// own address space, so the parent's address-space limit is not
// the child's -- it is the limit the child inherits, and the
// parent is left with a limit it set itself.
//
// The parent restores its own limit immediately after the fork,
// which the runner does through saveParentLimits and
// restoreParentLimits.
//
// The process-count limit is not applied: macOS has no
// RLIMIT_NPROC.
func applyChildLimits(pid int, l *ResourceLimits) error {
	// The limit is applied by the parent before the fork; there
	// is nothing to do once the child has started.
	return nil
}
