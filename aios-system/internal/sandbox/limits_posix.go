//go:build unix || darwin || linux

package sandbox

import (
	"sync"
	"syscall"
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
	cpu   syscall.Rlimit
	as    syscall.Rlimit
	nproc syscall.Rlimit
}

var parentLimits parentLimitsState

// saveParentLimits reads the current values of the rlimits
// the runner manages and stores them in package state so
// restore can revert them after the child exits. The
// package-level state is mutex-guarded so concurrent runs
// (which should not happen today, but a future goroutine
// may dispatch them) do not race.
func saveParentLimits() {
	parentLimits.mu.Lock()
	defer parentLimits.mu.Unlock()
	parentLimits.saved = true
	if rl, err := getRlimit(syscall.RLIMIT_CPU); err == nil {
		parentLimits.cpu = rl
	}
	if rl, err := getRlimit(syscall.RLIMIT_AS); err == nil {
		parentLimits.as = rl
	}
	if rl, err := getRlimit(syscall.RLIMIT_NPROC); err == nil {
		parentLimits.nproc = rl
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
	_ = syscall.Setrlimit(syscall.RLIMIT_CPU, &parentLimits.cpu)
	_ = syscall.Setrlimit(syscall.RLIMIT_AS, &parentLimits.as)
	_ = syscall.Setrlimit(syscall.RLIMIT_NPROC, &parentLimits.nproc)
}

func getRlimit(which int) (syscall.Rlimit, error) {
	var r syscall.Rlimit
	if err := syscall.Getrlimit(which, &r); err != nil {
		return r, err
	}
	return r, nil
}

// applyLimits applies the resource limits to the current
// process. On POSIX the limits are set through
// syscall.Setrlimit. The caller is the parent process; the
// limit is inherited by the child after Start. The caller is
// responsible for restoring the parent's previous limits via
// restoreParentLimits (see saveParentLimits /
// restoreParentLimits).
//
// The function returns nil when at least one limit was
// applied. A zero field in the input is treated as "no
// limit" and the corresponding syscall.Setrlimit is skipped.
func (l ResourceLimits) applyLimits() error {
	// RLIMIT_CPU is enforced in seconds. The kernel rounds up
	// to the next second, so a 1s limit is honored as expected.
	if l.CPUSeconds > 0 {
		if err := syscall.Setrlimit(syscall.RLIMIT_CPU, &syscall.Rlimit{
			Cur: uint64(l.CPUSeconds),
			Max: uint64(l.CPUSeconds),
		}); err != nil {
			return &LimitsError{Field: "cpu_seconds", Reason: err.Error()}
		}
	}
	// RLIMIT_AS is the address-space limit. A non-zero value
	// is honored in bytes.
	if l.MemoryBytes > 0 {
		if err := syscall.Setrlimit(syscall.RLIMIT_AS, &syscall.Rlimit{
			Cur: uint64(l.MemoryBytes),
			Max: uint64(l.MemoryBytes),
		}); err != nil {
			return &LimitsError{Field: "memory_bytes", Reason: err.Error()}
		}
	}
	// RLIMIT_NPROC is the process count limit. On Linux the
	// limit applies to the real UID, not the calling process.
	if l.Processes > 0 {
		if err := syscall.Setrlimit(syscall.RLIMIT_NPROC, &syscall.Rlimit{
			Cur: uint64(l.Processes),
			Max: uint64(l.Processes),
		}); err != nil {
			return &LimitsError{Field: "processes", Reason: err.Error()}
		}
	}
	return nil
}
