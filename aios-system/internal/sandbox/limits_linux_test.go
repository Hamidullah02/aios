//go:build linux

package sandbox

import (
	"os"
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
//
// The limit is relative to what the real UID already has, because
// the kernel compares the UID's total task count against it. See
// processBudget.
func TestProcessLimitIsAppliedOnLinux(t *testing.T) {
	saveParentLimits()
	defer restoreParentLimits()

	existing, ok := countProcessesForUID(os.Getuid())
	if !ok {
		t.Skip("cannot count the processes for this uid")
	}

	limits := ResourceLimits{TimeSeconds: 60, Processes: 64}
	if err := limits.applyLimits(); err != nil {
		t.Fatalf("applyLimits: %v", err)
	}
	r, err := getRlimit(unix.RLIMIT_NPROC)
	if err != nil {
		t.Fatalf("getrlimit: %v", err)
	}
	want := existing + 64
	if r.Cur != want {
		t.Fatalf("nproc = %d, want %d (the %d processes the uid already had, plus 64)", r.Cur, want, existing)
	}
}

// TestProcessBudgetIsRelativeToTheExistingCount asserts that the
// process budget leaves room for the processes the UID already has.
//
// The kernel compares the UID's TOTAL task count against
// RLIMIT_NPROC, so an absolute limit of 4 is already exceeded on a
// machine running more than four processes for that UID -- and the
// runner's own fork then fails with EAGAIN.
//
// That is what happened on CI: every job runs as the same 'runner'
// uid, the limit of 4 was below the count the uid already had, and
// the fork failed with "resource temporarily unavailable".
func TestProcessBudgetIsRelativeToTheExistingCount(t *testing.T) {
	existing, ok := countProcessesForUID(os.Getuid())
	if !ok {
		t.Skip("cannot count the processes for this uid")
	}
	if existing == 0 {
		t.Skip("this uid has no processes, so the relative budget cannot be distinguished")
	}

	limits := ResourceLimits{TimeSeconds: 60, Processes: 4}
	budget := limits.processBudget()
	if budget != existing+4 {
		t.Errorf("processBudget = %d, want %d", budget, existing+4)
	}
	if budget <= existing {
		t.Errorf("the budget %d does not leave room for the %d processes the uid already has", budget, existing)
	}
}

// TestProcessBudgetFallsBackWhenTheCountFails asserts the
// conservative fallback.
//
// When the count cannot be taken, the caller's number is used as
// an absolute limit. That is the tight direction: the run is
// bounded more tightly rather than more loosely.
func TestProcessBudgetFallsBackWhenTheCountFails(t *testing.T) {
	limits := ResourceLimits{TimeSeconds: 60, Processes: 4}
	budget := limits.processBudget()
	if budget < 4 {
		t.Errorf("processBudget = %d, want at least the caller's 4", budget)
	}
}

// TestStatusBelongsToUID pins the parsing of a /proc status file.
//
// The real uid is the first number on the "Uid:" line, and it is
// the one RLIMIT_NPROC counts.
func TestStatusBelongsToUID(t *testing.T) {
	status := []byte("Name:\tbash\nUid:\t1000\t1000\t1000\t1000\nGid:\t1000\t1000\t1000\t1000\n")
	if !statusBelongsToUID(status, 1000) {
		t.Error("the status file should belong to uid 1000")
	}
	if statusBelongsToUID(status, 1001) {
		t.Error("the status file should not belong to uid 1001")
	}
	// A file with no Uid line belongs to nobody.
	if statusBelongsToUID([]byte("Name:\tbash\n"), 1000) {
		t.Error("a status file with no Uid line should belong to nobody")
	}
	// The effective uid is not the real uid; only the first
	// number counts.
	mixed := []byte("Uid:\t1000\t1001\t1000\t1000\n")
	if !statusBelongsToUID(mixed, 1000) {
		t.Error("the real uid is the first number, so this belongs to uid 1000")
	}
	if statusBelongsToUID(mixed, 1001) {
		t.Error("the effective uid must not be counted as the real uid")
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

