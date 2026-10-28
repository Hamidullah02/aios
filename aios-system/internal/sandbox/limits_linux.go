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
// applyLimits applies the resource limits that are safe to apply
// to the parent.
//
// # On Linux the parent gets no limits at all
//
// The runner starts the child with clone(2) and CLONE_VM, so the
// child shares the parent's address space until it execs. Every
// limit set on the parent is therefore also a limit on the child
// -- and, far worse, on the parent's own Go runtime, which
// reserves a large virtual arena.
//
// Setting RLIMIT_AS in the parent therefore made the runtime fail
// to allocate, and the process died with "fatal error: runtime:
// cannot allocate memory". Setting RLIMIT_NPROC in the parent
// counted the parent's own tasks against the limit, and the
// runner's own fork failed with EAGAIN.
//
// So the parent's limits are left alone. The child's limits are
// applied to the child, after it has started, with prlimit(2).
// See applyChildLimits.
//
// The child is a separate process once it has exec'd, so a limit
// set on it with prlimit is its own, and the parent is
// unconstrained. That is the same arrangement the Windows Job
// Object gives for free.
func (l ResourceLimits) applyLimits() error {
	// The CPU limit is applied to the child with prlimit, like
	// the others. RLIMIT_CPU is a per-process CPU-time counter
	// and the child inherits the parent's, so it needs no
	// special handling.
	return nil
}

// applyChildLimits applies the limits that belong to the child
// rather than to the parent.
//
// It is called after the child has started, with the child's pid.
// prlimit(2) can set another process's limits without touching
// the caller, so the parent stays unconstrained.
//
// The memory limit is applied as RLIMIT_AS. The child is a fresh
// process after exec, so it has its own address space and the
// limit is its own.
//
// The process limit is applied as RLIMIT_NPROC, made relative to
// the tasks the UID already has. See processBudget.
func applyChildLimits(pid int, l *ResourceLimits) error {
	if l == nil {
		return nil
	}
	if l.CPUSeconds > 0 {
		if err := unix.Prlimit(pid, unix.RLIMIT_CPU, &unix.Rlimit{
			Cur: uint64(l.CPUSeconds),
			Max: uint64(l.CPUSeconds),
		}, nil); err != nil {
			return &LimitsError{Field: "cpu_seconds", Reason: err.Error()}
		}
	}
	if l.MemoryBytes > 0 {
		if err := unix.Prlimit(pid, unix.RLIMIT_AS, &unix.Rlimit{
			Cur: uint64(l.MemoryBytes),
			Max: uint64(l.MemoryBytes),
		}, nil); err != nil {
			return &LimitsError{Field: "memory_bytes", Reason: err.Error()}
		}
	}
	if l.Processes > 0 {
		budget := l.processBudget()
		if err := unix.Prlimit(pid, unix.RLIMIT_NPROC, &unix.Rlimit{
			Cur: budget,
			Max: budget,
		}, nil); err != nil {
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

// countProcessesForUID counts the tasks belonging to uid.
//
// It walks /proc, and for each numeric entry reads the process's
// status file and looks at the real-uid field.
//
// # The count is of TASKS, not of processes
//
// RLIMIT_NPROC counts threads as well as processes. The kernel
// keeps a per-user count that is incremented for every new task,
// and a thread is a task. So a Go program with forty goroutine
// threads is forty tasks, and the budget has to leave room for
// them.
//
// The status file carries a "Threads:" line, so the count is the
// sum of the thread counts of the processes belonging to the uid.
// Counting only the /proc entries -- one per process -- undercounts
// badly, and the limit ends up already exceeded: the fork fails
// with EAGAIN, "resource temporarily unavailable".
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
		if !statusBelongsToUID(status, uid) {
			continue
		}
		count += statusThreadCount(status)
	}
	return count, true
}

// statusThreadCount returns the number of tasks the process has.
//
// The "Threads:" line of a /proc status file carries it. A status
// file without the line counts as one task, which is the
// conservative answer for a process that has not reported yet.
func statusThreadCount(status []byte) uint64 {
	const prefix = "Threads:"
	for _, line := range splitLines(status) {
		if len(line) < len(prefix) || string(line[:len(prefix)]) != prefix {
			continue
		}
		fields := splitFields(string(line[len(prefix):]))
		if len(fields) == 0 {
			return 1
		}
		n, err := parseInt(fields[0])
		if err != nil || n < 1 {
			return 1
		}
		return uint64(n)
	}
	return 1
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
