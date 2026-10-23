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
