package sandbox

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// processAlive reports whether a pid is still running.
//
// It uses the signal-0 probe on POSIX and the OpenProcess probe
// on Windows. The signal-0 probe returns an error for a
// process that exists but belongs to another user, so the check
// is deliberately conservative: it says "alive" unless the
// kernel says the process is gone.
func processAlive(t *testing.T, pid int) bool {
	t.Helper()
	if pid <= 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return processRunning(proc)
}

// writePIDFile is the helper the tree test uses: the child
// writes the pid of a grandchild to a file so the parent test
// can check whether the grandchild survived.
//
// It is written as a Go program rather than a shell script so
// the test does not depend on a shell being present.
func writePIDFile(t *testing.T, dir string, content string) string {
	t.Helper()
	path := filepath.Join(dir, "pid.txt")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestProcessTreeKill_GrandchildDoesNotSurvive is the test for
// the claim the sandbox makes: a command that backgrounds work
// does not leave that work running after the timeout.
//
// The child spawns a grandchild that would outlive it, writes
// the grandchild's pid to a file, and then sleeps. The parent
// test times the run out and then checks whether the grandchild
// is still alive.
//
// Before the process-group containment, the grandchild
// survived: the runner killed the direct child, the sleep kept
// running, and it went on with the parent's privileges.
//
// The child and the grandchild are copies of the test binary
// rather than shell commands, so the test does not depend on
// shell syntax that differs between Windows and POSIX.
func TestProcessTreeKill_GrandchildDoesNotSurvive(t *testing.T) {
	dir := t.TempDir()
	pidFile := filepath.Join(dir, "grandchild.pid")

	r := NewSubprocessRunner()
	opts := RunOptions{
		Command: []string{
			selfExec(t),
			"-test.run=^TestHelperSpawnTree$",
			"-test.coverprofile=",
			"-test.outputdir=" + os.TempDir(),
		},
		Env: map[string]string{
			"AIOS_TREE_HELPER":  "spawn",
			"AIOS_TREE_PIDFILE": pidFile,
			"TMP":               os.TempDir(),
			"TEMP":              os.TempDir(),
		},
		Timeout: 2,
		Cwd:     dir,
	}
	res, err := r.Run(opts)
	if err == nil {
		t.Fatalf("expected a timeout, got exit %d (stderr=%q)", res.ExitCode, res.Stderr)
	}
	var st *SandboxTimeout
	if !errors.As(err, &st) {
		t.Fatalf("expected *SandboxTimeout, got %T: %v", err, err)
	}

	// The pid file may not have been written: the timeout can
	// land before the child gets to it. The test is only
	// meaningful when the grandchild was actually recorded.
	pid := readPIDFile(t, pidFile)
	if pid == 0 {
		t.Skip("the grandchild pid was not recorded before the timeout")
	}

	// The tree kill is asynchronous on some platforms, so give
	// the kernel a moment to reap the process.
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if !processAlive(t, pid) {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("the grandchild (pid %d) survived the timeout", pid)
}

// TestHelperSpawnTree is the child helper for the tree test.
//
// In "spawn" mode it starts a grandchild -- another copy of the
// test binary in "sleep" mode -- records the grandchild's pid,
// and then sleeps. The parent test then times the run out and
// checks whether the grandchild died with it.
func TestHelperSpawnTree(t *testing.T) {
	mode := os.Getenv("AIOS_TREE_HELPER")
	if mode == "" {
		return
	}
	if mode == "sleep" {
		time.Sleep(120 * time.Second)
		return
	}
	if mode != "spawn" {
		return
	}
	self, err := os.Executable()
	if err != nil {
		return
	}
	cmd := exec.Command(self,
		"-test.run=^TestHelperSpawnTree$",
		"-test.coverprofile=",
		"-test.outputdir="+os.TempDir(),
	)
	cmd.Env = append(os.Environ(), "AIOS_TREE_HELPER=sleep")
	if err := cmd.Start(); err != nil {
		return
	}
	if path := os.Getenv("AIOS_TREE_PIDFILE"); path != "" {
		_ = os.WriteFile(path, []byte(strconv.Itoa(cmd.Process.Pid)), 0o644)
	}
	// The child sleeps long enough for the parent to time it
	// out. The grandchild is left running on purpose.
	time.Sleep(120 * time.Second)
}

// TestProcessTreeKill_ChildIsKilledOnTimeout is the simpler
// half: the direct child does not survive the timeout.
func TestProcessTreeKill_ChildIsKilledOnTimeout(t *testing.T) {
	r := NewSubprocessRunner()
	res, err := r.Run(RunOptions{
		Command: sleepCommand(60),
		Timeout: 1,
	})
	if err == nil {
		t.Fatalf("expected a timeout, got exit %d", res.ExitCode)
	}
	var st *SandboxTimeout
	if !errors.As(err, &st) {
		t.Fatalf("expected *SandboxTimeout, got %T: %v", err, err)
	}
}

// TestJobObject_LimitsAreApplied covers the Windows Job Object
// path: the job is created, the limits are set, and the handle
// is released cleanly.
//
// The test is Windows-only because the Job Object is a Windows
// API. On POSIX the equivalent containment is the process group
// and the limits are rlimits, which the limits tests cover.
func TestJobObject_LimitsAreApplied(t *testing.T) {
	job, err := startJobObject(&ResourceLimits{TimeSeconds: 60, Processes: 4, MemoryBytes: 1 << 30})
	if err != nil {
		t.Fatalf("startJobObject: %v", err)
	}
	if job == nil {
		t.Skip("no job object on this platform")
	}
	job.close()
	// Closing twice must be safe: the runner defers the close
	// and the error path closes as well.
	job.close()
}

// TestNamespacePolicy_Validate covers the namespace policy
// validation.
func TestNamespacePolicy_Validate(t *testing.T) {
	p := DefaultNamespacePolicy()
	if err := p.Validate(); err != nil {
		t.Fatalf("default policy is invalid: %v", err)
	}
	if !p.User || !p.Network || !p.Mount || !p.PID {
		t.Fatalf("default policy = %+v", p)
	}
	var nilPolicy *NamespacePolicy
	if err := nilPolicy.Validate(); err != nil {
		t.Fatalf("nil policy is invalid: %v", err)
	}
}

// TestNamespacePolicy_DisabledIsValid covers the "no
// namespaces" case: it is valid, because a caller may want the
// old behaviour on purpose.
func TestNamespacePolicy_DisabledIsValid(t *testing.T) {
	p := NamespacePolicy{}
	if err := p.Validate(); err != nil {
		t.Fatalf("disabled policy is invalid: %v", err)
	}
}

// TestNamespaceErrorIsTyped confirms the error type so callers
// can match on it.
func TestNamespaceErrorIsTyped(t *testing.T) {
	err := &NamespaceError{Field: "user", Reason: "not permitted"}
	if !strings.HasPrefix(err.Error(), "sandbox: namespace:") {
		t.Fatalf("message = %q", err.Error())
	}
	if !strings.HasSuffix(err.Error(), ".") {
		t.Fatalf("message = %q", err.Error())
	}
}

// TestConfigureNamespacesIsSafeWhenDisabled checks that the
// namespace configuration is a no-op when the policy asks for
// nothing. A runner without a namespace policy must behave
// exactly as before.
func TestConfigureNamespacesIsSafeWhenDisabled(t *testing.T) {
	cmd := exec.Command("cmd.exe", "/c", "exit", "0")
	if !isWindowsShell() {
		cmd = exec.Command("sh", "-c", "exit 0")
	}
	if err := configureNamespaces(cmd, namespacePolicy{}); err != nil {
		t.Fatalf("configureNamespaces: %v", err)
	}
	// The disabled policy must not set any namespace flags.
	// The flag field is platform specific, so the check is on
	// the observable outcome: the command still runs.
	if err := cmd.Run(); err != nil {
		t.Fatalf("the command failed after a disabled policy: %v", err)
	}
}

// readPIDFile reads a pid from a file, returning 0 when the file
// is missing or unreadable.
func readPIDFile(t *testing.T, path string) int {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0
	}
	return pid
}

// sleepCommand returns a command that sleeps for the given
// number of seconds, in the form the platform supports.
func sleepCommand(seconds int) []string {
	if isWindowsShell() {
		return []string{"cmd.exe", "/c", "ping -n " + strconv.Itoa(seconds+1) + " 127.0.0.1 > nul"}
	}
	return []string{"sleep", strconv.Itoa(seconds)}
}
