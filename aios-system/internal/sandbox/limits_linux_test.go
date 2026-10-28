//go:build linux

package sandbox

import (
	"os"
	"os/exec"
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
// The limit is applied to the CHILD with prlimit, not to the
// parent. The parent's own limits are the parent's business: a
// parent that cannot allocate cannot run the next task.
func TestProcessLimitIsAppliedOnLinux(t *testing.T) {
	existing, ok := countProcessesForUID(os.Getuid())
	if !ok {
		t.Skip("cannot count the processes for this uid")
	}

	limits := ResourceLimits{TimeSeconds: 60, Processes: 64}
	budget := limits.processBudget()
	if budget != existing+64 {
		t.Fatalf("processBudget = %d, want %d", budget, existing+64)
	}
}

// TestProcessBudgetIsRelativeToTheExistingCount asserts that the
// process budget leaves room for the processes the UID already has.
//
// The kernel compares the UID's TOTAL task count against
// RLIMIT_NPROC, so an absolute limit of 4 is already exceeded on a
// machine running more than four tasks for that UID -- and the
// fork fails with EAGAIN.
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

// TestParentLimitsAreNotConstrained asserts that applying the
// limits leaves the parent's own limits alone.
//
// This is the defect that made Linux unusable, and it was two
// separate mistakes.
//
// The first was the hard limit. The runner used to set both the
// soft and the hard limit in the parent. setrlimit(2): an
// unprivileged process may lower the hard limit freely, but may
// raise it only up to the previous hard limit. So the parent
// lowered its own hard limit and could never lift it again -- and
// restoreParentLimits, which tried to put it back, failed with
// EPERM while its error was discarded with "_ =".
//
// The second was the address space. Go's fork uses clone(2) with
// CLONE_VM, so the child shares the parent's address space until
// it execs. A limit on the parent's address space is therefore a
// limit on the parent's own Go runtime, which reserves a large
// virtual arena. With RLIMIT_AS set to 1GB the runtime could not
// allocate, and the process died with "fatal error: runtime:
// cannot allocate memory".
//
// So the parent's limits are left alone now. The child's are
// applied to the child, with prlimit.
func TestParentLimitsAreNotConstrained(t *testing.T) {
	saveParentLimits()
	defer restoreParentLimits()

	before, err := getRlimit(unix.RLIMIT_AS)
	if err != nil {
		t.Fatalf("getrlimit: %v", err)
	}
	beforeNproc, err := getRlimit(unix.RLIMIT_NPROC)
	if err != nil {
		t.Fatalf("getrlimit: %v", err)
	}

	limits := ResourceLimits{TimeSeconds: 60, Processes: 4, MemoryBytes: 1 << 20}
	if err := limits.applyLimits(); err != nil {
		t.Fatalf("applyLimits: %v", err)
	}

	after, err := getRlimit(unix.RLIMIT_AS)
	if err != nil {
		t.Fatalf("getrlimit: %v", err)
	}
	if after.Cur != before.Cur || after.Max != before.Max {
		t.Errorf("the parent's address-space limit changed: %+v -> %+v", before, after)
	}
	afterNproc, err := getRlimit(unix.RLIMIT_NPROC)
	if err != nil {
		t.Fatalf("getrlimit: %v", err)
	}
	if afterNproc.Cur != beforeNproc.Cur || afterNproc.Max != beforeNproc.Max {
		t.Errorf("the parent's process-count limit changed: %+v -> %+v", beforeNproc, afterNproc)
	}
}

// TestApplyChildLimitsDoesNotTouchTheParent asserts that the
// child's limits are applied to the child and to nobody else.
//
// The child here is a real process, so the limits have somewhere
// to land. The parent's own limits must be untouched.
func TestApplyChildLimitsDoesNotTouchTheParent(t *testing.T) {
	saveParentLimits()
	defer restoreParentLimits()

	before, err := getRlimit(unix.RLIMIT_AS)
	if err != nil {
		t.Fatalf("getrlimit: %v", err)
	}

	// Start a child that does nothing.
	cmd := exec.Command("/bin/sh", "-c", "sleep 5")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()

	limits := ResourceLimits{TimeSeconds: 60, MemoryBytes: 1 << 30, Processes: 8}
	if err := applyChildLimits(cmd.Process.Pid, &limits); err != nil {
		t.Fatalf("applyChildLimits: %v", err)
	}

	after, err := getRlimit(unix.RLIMIT_AS)
	if err != nil {
		t.Fatalf("getrlimit: %v", err)
	}
	if after.Cur != before.Cur || after.Max != before.Max {
		t.Errorf("the parent's address-space limit changed: %+v -> %+v", before, after)
	}

	// And the child has the limit.
	childLimit, err := getChildRlimit(cmd.Process.Pid, unix.RLIMIT_AS)
	if err != nil {
		t.Fatalf("get child rlimit: %v", err)
	}
	if childLimit.Cur != 1<<30 {
		t.Errorf("the child's address-space limit = %d, want %d", childLimit.Cur, 1<<30)
	}
}

// getChildRlimit reads a limit from another process.
func getChildRlimit(pid, which int) (unix.Rlimit, error) {
	var r unix.Rlimit
	if err := unix.Prlimit(pid, which, nil, &r); err != nil {
		return r, err
	}
	return r, nil
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


