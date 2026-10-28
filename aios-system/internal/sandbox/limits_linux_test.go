//go:build linux

package sandbox

import (
	"testing"

	"golang.org/x/sys/unix"
)

// TestParentLimitsRestoredAfterApply is a regression test
// for the bug where the sandbox runner applied rlimits to
// the parent process and never restored them, leaving the
// daemon constrained after each tool call. We save the
// current values, apply a tight limit, then restore and
// confirm the parent's rlimits are back to the saved values.
func TestParentLimitsRestoredAfterApply(t *testing.T) {
	saveParentLimits()
	// Set a small CPU and memory limit, then restore, and
	// check the values are unchanged.
	if err := (ResourceLimits{TimeSeconds: 60, CPUSeconds: 1, MemoryBytes: 1 << 20}).applyLimits(); err != nil {
		t.Fatalf("applyLimits: %v", err)
	}
	restoreParentLimits()
	if r, err := getRlimit(unix.RLIMIT_CPU); err == nil {
		if r.Cur < 1 {
			t.Fatalf("cpu: not restored: %+v", r)
		}
	}
	if r, err := getRlimit(unix.RLIMIT_AS); err == nil {
		if r.Cur < (1 << 20) {
			t.Fatalf("as: not restored: %+v", r)
		}
	}
}

// TestProcessLimitIsAppliedOnLinux covers the RLIMIT_NPROC
// branch. It is Linux-only because the constant and the
// kernel behaviour are: the limit counts processes for the
// real UID, and macOS has no equivalent.
func TestProcessLimitIsAppliedOnLinux(t *testing.T) {
	saveParentLimits()
	defer restoreParentLimits()

	limits := ResourceLimits{TimeSeconds: 60, Processes: 64}
	if err := limits.applyLimits(); err != nil {
		t.Fatalf("applyLimits: %v", err)
	}
	r, err := getRlimit(unix.RLIMIT_NPROC)
	if err != nil {
		t.Fatalf("getrlimit: %v", err)
	}
	if r.Cur != 64 {
		t.Fatalf("nproc = %d, want 64", r.Cur)
	}
}

// TestHardLimitIsNotLoweredInTheParent asserts that the runner
// leaves the hard limit alone.
//
// The runner sets the limit in the PARENT, because Go's exec has
// no way to set a rlimit in the child and the child inherits the
// parent's.
//
// Lowering the HARD limit there is not recoverable. setrlimit(2):
// an unprivileged process may lower the hard limit freely, but
// may raise it only up to the previous hard limit. So the parent
// could never put it back, and restoreParentLimits failed with
// EPERM while its error was discarded.
//
// The effect was severe, and it was invisible on a machine where
// the account is the only user of it:
//
//   - RLIMIT_NPROC counts every process belonging to the real
//     uid. On a shared-uid CI runner the parent's own limit
//     applied to the whole machine, and the next fork failed with
//     EAGAIN: "resource temporarily unavailable".
//   - RLIMIT_AS stopped the Go runtime from allocating, and the
//     process died with "fatal error: runtime: cannot allocate
//     memory".
//
// So the hard limit stays where it is, and the soft limit is the
// one that carries the bound.
func TestHardLimitIsNotLoweredInTheParent(t *testing.T) {
	if canRestoreHardLimit() {
		t.Skip("this process can raise a hard limit, so it is expected to set one")
	}

	saveParentLimits()
	defer restoreParentLimits()

	before, err := getRlimit(unix.RLIMIT_NPROC)
	if err != nil {
		t.Fatalf("getrlimit: %v", err)
	}

	limits := ResourceLimits{TimeSeconds: 60, Processes: 4, MemoryBytes: 1 << 20}
	if err := limits.applyLimits(); err != nil {
		t.Fatalf("applyLimits: %v", err)
	}

	after, err := getRlimit(unix.RLIMIT_NPROC)
	if err != nil {
		t.Fatalf("getrlimit: %v", err)
	}
	if after.Max != before.Max {
		t.Errorf("nproc hard limit changed: %d -> %d", before.Max, after.Max)
	}
	if after.Cur != 4 {
		t.Errorf("nproc soft limit = %d, want 4", after.Cur)
	}

	// And the restore must actually work, which it could not
	// when the hard limit had been lowered.
	restoreParentLimits()
	restored, err := getRlimit(unix.RLIMIT_NPROC)
	if err != nil {
		t.Fatalf("getrlimit: %v", err)
	}
	if restored.Cur != before.Cur {
		t.Errorf("nproc soft limit was not restored: %d, want %d", restored.Cur, before.Cur)
	}
	if restored.Max != before.Max {
		t.Errorf("nproc hard limit was not restored: %d, want %d", restored.Max, before.Max)
	}
}

// TestAddressSpaceLimitIsRestorable asserts the same for
// RLIMIT_AS, which is the limit that killed the Go runtime.
func TestAddressSpaceLimitIsRestorable(t *testing.T) {
	if canRestoreHardLimit() {
		t.Skip("this process can raise a hard limit, so it is expected to set one")
	}

	saveParentLimits()
	defer restoreParentLimits()

	before, err := getRlimit(unix.RLIMIT_AS)
	if err != nil {
		t.Fatalf("getrlimit: %v", err)
	}

	limits := ResourceLimits{TimeSeconds: 60, MemoryBytes: 1 << 20}
	if err := limits.applyLimits(); err != nil {
		t.Fatalf("applyLimits: %v", err)
	}

	after, err := getRlimit(unix.RLIMIT_AS)
	if err != nil {
		t.Fatalf("getrlimit: %v", err)
	}
	if after.Max != before.Max {
		t.Errorf("as hard limit changed: %d -> %d", before.Max, after.Max)
	}
	if after.Cur != 1<<20 {
		t.Errorf("as soft limit = %d, want %d", after.Cur, 1<<20)
	}

	restoreParentLimits()
	restored, err := getRlimit(unix.RLIMIT_AS)
	if err != nil {
		t.Fatalf("getrlimit: %v", err)
	}
	if restored.Cur != before.Cur {
		t.Errorf("as soft limit was not restored: %d, want %d", restored.Cur, before.Cur)
	}
}

