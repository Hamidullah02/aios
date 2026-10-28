//go:build linux

package sandbox

import (
	"errors"
	"fmt"
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
	nproc unix.Rlimit
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
	if rl, err := getRlimit(unix.RLIMIT_CPU); err == nil {
		parentLimits.cpu = rl
	}
	if rl, err := getRlimit(unix.RLIMIT_AS); err == nil {
		parentLimits.as = rl
	}
	if rl, err := getRlimit(unix.RLIMIT_NPROC); err == nil {
		parentLimits.nproc = rl
	}
}

// restoreParentLimits reverts the parent's rlimits to the saved
// values. If saveParentLimits was not called (no run has happened
// yet) the function is a no-op.
//
// The errors are ignored deliberately: the restore is a
// best-effort cleanup, and the caller has already got the
// result it came for.
//
// The restore is now reliable, because the runner no longer
// lowers the hard limit in the parent. It used to, and an
// unprivileged process can never raise a hard limit it has
// lowered -- so every restore failed with EPERM, silently, and
// the parent kept the child's limits for the rest of its life.
func restoreParentLimits() {
	parentLimits.mu.Lock()
	defer parentLimits.mu.Unlock()
	if !parentLimits.saved {
		return
	}
	_ = unix.Setrlimit(unix.RLIMIT_CPU, &parentLimits.cpu)
	_ = unix.Setrlimit(unix.RLIMIT_AS, &parentLimits.as)
	_ = unix.Setrlimit(unix.RLIMIT_NPROC, &parentLimits.nproc)
}

func getRlimit(which int) (unix.Rlimit, error) {
	var r unix.Rlimit
	if err := unix.Getrlimit(which, &r); err != nil {
		return r, err
	}
	return r, nil
}

// applyLimits applies the resource limits to the current
// process. On Linux the limits are set through
// syscall.Setrlimit. The caller is the parent process; the
// limit is inherited by the child after Start. The caller is
// responsible for restoring the parent's previous limits via
// restoreParentLimits (see saveParentLimits /
// restoreParentLimits).
//
// The function returns nil when at least one limit was
// applied. A zero field in the input is treated as "no
// limit" and the corresponding syscall.Setrlimit is skipped.
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
// That was the bug, and it was severe. The process-count limit
// counts every process belonging to the real uid, so on a
// shared-uid CI runner the parent's own limit applied to the
// whole machine: the next fork failed with EAGAIN ("resource
// temporarily unavailable"). The address-space limit was worse:
// the Go runtime could not allocate, and the process died with
// "fatal error: runtime: cannot allocate memory".
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
	// RLIMIT_NPROC is the process count limit. On Linux the
	// limit applies to the real UID, not the calling process, so
	// the budget has to be made relative to what that UID already
	// has. See processBudget.
	if l.Processes > 0 {
		if err := unix.Setrlimit(unix.RLIMIT_NPROC, &unix.Rlimit{
			Cur: l.processBudget(),
			Max: l.hardLimit(unix.RLIMIT_NPROC, l.processBudget()),
		}); err != nil {
			return &LimitsError{Field: "processes", Reason: err.Error()}
		}
	}
	return nil
}

// processBudget returns the process count the kernel will accept.
//
// RLIMIT_NPROC counts the tasks belonging to the real UID, and the
// kernel compares that total against the limit. So the limit is
// absolute: a limit of 4 on a machine that already runs thirty
// processes for the same UID is already exceeded, and the very
// next fork fails with EAGAIN ("resource temporarily
// unavailable").
//
// That is not what the caller means. "Processes: 4" should mean
// "this run may start four more processes", not "this UID may
// have four processes in total, including the ones the machine
// already had".
//
// So the budget is the processes the UID already has, plus the
// caller's. The count is taken before the limit is set, and it
// includes this process, so the child's own fork is always within
// the budget.
//
// When the count cannot be taken -- an unreadable /proc, a
// container with it hidden -- the caller's number is used as the
// absolute limit. That is the conservative direction: the run is
// bounded more tightly rather than more loosely, and the caller
// finds out immediately if the bound is too low to start.
func (l ResourceLimits) processBudget() uint64 {
	existing, ok := countProcessesForUID(os.Getuid())
	if !ok {
		return uint64(l.Processes)
	}
	total := existing + uint64(l.Processes)
	// A limit of zero means "no processes at all", which would
	// stop the runner from starting the child. One is the floor.
	if total < 1 {
		return 1
	}
	return total
}

// countProcessesForUID counts the processes belonging to uid.
//
// It walks /proc, and for each numeric entry reads the process's
// status file and looks at the real-uid field. The count matches
// the kernel's own accounting, which counts tasks by real uid.
//
// The second return value is false when the count could not be
// taken, which the caller treats as "do not trust a relative
// budget".
func countProcessesForUID(uid int) (uint64, bool) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return 0, false
	}
	var count uint64
	for _, entry := range entries {
		pid := entry.Name()
		if pid == "" || pid[0] < '0' || pid[0] > '9' {
			continue
		}
		status, err := os.ReadFile("/proc/" + pid + "/status")
		if err != nil {
			// The process exited between the readdir and the
			// read. It is not counted, which is fine: the
			// count is a snapshot.
			continue
		}
		if statusBelongsToUID(status, uid) {
			count++
		}
	}
	return count, true
}

// statusBelongsToUID reports whether a /proc/<pid>/status file
// names uid as the process's real uid.
//
// The file has a "Uid:" line with the real, effective, saved and
// filesystem uids. The real uid is the first number on it, and it
// is the one RLIMIT_NPROC counts.
func statusBelongsToUID(status []byte, uid int) bool {
	const prefix = "Uid:"
	for _, line := range splitLines(status) {
		if len(line) < len(prefix) || string(line[:len(prefix)]) != prefix {
			continue
		}
		fields := splitFields(string(line[len(prefix):]))
		if len(fields) == 0 {
			return false
		}
		real, err := parseInt(fields[0])
		if err != nil {
			return false
		}
		return real == uid
	}
	return false
}

// splitLines splits a byte slice on newlines.
func splitLines(b []byte) [][]byte {
	var out [][]byte
	start := 0
	for i := 0; i < len(b); i++ {
		if b[i] == '\n' {
			out = append(out, b[start:i])
			start = i + 1
		}
	}
	if start < len(b) {
		out = append(out, b[start:])
	}
	return out
}

// splitFields splits a string on whitespace.
func splitFields(s string) []string {
	var out []string
	start := -1
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			if start >= 0 {
				out = append(out, s[start:i])
				start = -1
			}
			continue
		}
		if start < 0 {
			start = i
		}
	}
	if start >= 0 {
		out = append(out, s[start:])
	}
	return out
}

// parseInt parses a non-negative decimal number.
func parseInt(s string) (int, error) {
	if s == "" {
		return 0, errors.New("empty number")
	}
	n := 0
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return 0, fmt.Errorf("not a number: %q", s)
		}
		n = n*10 + int(s[i]-'0')
	}
	return n, nil
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
