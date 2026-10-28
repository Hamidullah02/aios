//go:build linux

package sandbox

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

// TestZZDiag gathers the facts about what a child process needs
// on this machine. It is a diagnostic, not an assertion: it
// prints and passes, and the numbers are read out of the CI log.
//
// The questions it answers:
//
//   - How large is the address space a child Go binary reserves?
//   - How many tasks does the UID already have?
//   - How many threads does the child create?
//   - Does a child survive RLIMIT_AS at various sizes?
//   - Does a child survive RLIMIT_NPROC at various budgets?
func TestZZDiag(t *testing.T) {
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}

	uid := os.Getuid()
	existing, ok := countProcessesForUID(uid)
	t.Logf("UID %d already has %d tasks (count ok=%v)", uid, existing, ok)
	t.Logf("parent VmSize: %s", procField(os.Getpid(), "VmSize"))
	t.Logf("parent Threads: %s", procField(os.Getpid(), "Threads"))
	t.Logf("parent nproc limit: %+v", mustLimit(RLIMIT_NPROC))
	t.Logf("parent as limit: %+v", mustLimit(RLIMIT_AS))

	// Start the child the way the runner does.
	cmd := exec.Command(self, "-test.run=^TestHelperEcho$", "-test.coverprofile=")
	cmd.Env = append(os.Environ(), "AIOS_SANDBOX_HELPER=1")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	pid := cmd.Process.Pid

	// Let the child get going.
	time.Sleep(1500 * time.Millisecond)
	t.Logf("child VmSize: %s", procField(pid, "VmSize"))
	t.Logf("child Threads: %s", procField(pid, "Threads"))
	t.Logf("uid tasks now: %d", func() uint64 { n, _ := countProcessesForUID(uid); return n }())

	_ = cmd.Wait()
}

// TestZZDiagLimits tries the limits the runner applies and
// reports whether the child survives each one.
func TestZZDiagLimits(t *testing.T) {
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	uid := os.Getuid()

	for _, as := range []uint64{1 << 30, 2 << 30, 4 << 30, 8 << 30} {
		for _, extra := range []uint64{4, 64, 256} {
			name := fmt.Sprintf("as=%dMB extra=%d", as>>20, extra)
			t.Run(name, func(t *testing.T) {
				existing, _ := countProcessesForUID(uid)
				cmd := exec.Command(self, "-test.run=^TestHelperEcho$", "-test.coverprofile=")
				cmd.Env = append(os.Environ(), "AIOS_SANDBOX_HELPER=1")
				if err := cmd.Start(); err != nil {
					t.Fatal(err)
				}
				pid := cmd.Process.Pid
				time.Sleep(1200 * time.Millisecond)

				vm := procField(pid, "VmSize")
				th := procField(pid, "Threads")
				budget := existing + extra

				asErr := setChildLimit(pid, RLIMIT_AS, as)
				npErr := setChildLimit(pid, RLIMIT_NPROC, budget)

				out, waitErr := cmd.CombinedOutput()
				status := "OK"
				if waitErr != nil {
					status = "DIED: " + waitErr.Error()
				}
				first := ""
				for _, line := range strings.Split(string(out), "\n") {
					if strings.TrimSpace(line) != "" {
						first = strings.TrimSpace(line)
						break
					}
				}
				t.Logf("as=%dMB budget=%d (uid had %d) childVm=%s childThreads=%s asErr=%v npErr=%v -> %s | %s",
					as>>20, budget, existing, vm, th, asErr, npErr, status, first)
			})
		}
	}
}

// procField reads one field of /proc/<pid>/status.
func procField(pid int, field string) string {
	data, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/status")
	if err != nil {
		return "err:" + err.Error()
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, field+":") {
			return strings.TrimSpace(strings.TrimPrefix(line, field+":"))
		}
	}
	return "?"
}

// mustLimit reads a rlimit, ignoring the error.
func mustLimit(which int) string {
	r, err := getRlimit(which)
	if err != nil {
		return "err:" + err.Error()
	}
	return fmt.Sprintf("cur=%d max=%d", r.Cur, r.Max)
}

// setChildLimit sets a limit on another process.
func setChildLimit(pid, which int, value uint64) error {
	return unix.Prlimit(pid, which, &unix.Rlimit{Cur: value, Max: value}, nil)
}
