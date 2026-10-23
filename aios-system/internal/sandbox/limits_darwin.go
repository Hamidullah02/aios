//go:build darwin

package sandbox

import (
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
func (l ResourceLimits) applyLimits() error {
	// RLIMIT_CPU is enforced in seconds. The kernel rounds up
	// to the next second, so a 1s limit is honored as expected.
	if l.CPUSeconds > 0 {
		if err := unix.Setrlimit(unix.RLIMIT_CPU, &unix.Rlimit{
			Cur: uint64(l.CPUSeconds),
			Max: uint64(l.CPUSeconds),
		}); err != nil {
			return &LimitsError{Field: "cpu_seconds", Reason: err.Error()}
		}
	}
	// RLIMIT_AS is the address-space limit. A non-zero value
	// is honored in bytes.
	if l.MemoryBytes > 0 {
		if err := unix.Setrlimit(unix.RLIMIT_AS, &unix.Rlimit{
			Cur: uint64(l.MemoryBytes),
			Max: uint64(l.MemoryBytes),
		}); err != nil {
			return &LimitsError{Field: "memory_bytes", Reason: err.Error()}
		}
	}
	return nil
}
