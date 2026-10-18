package sandbox

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// selfExec returns the path to the test binary. The runner
// tests use the test binary itself as the child process
// so the tests run portably on Windows and POSIX.
func selfExec(t *testing.T) string {
	t.Helper()
	path, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable: %v", err)
	}
	return path
}

// runSelf invokes the test binary as a child with a
// matched sub-test. The sub-test prints, sets the exit
// code, and (optionally) sleeps. The function returns the
// SandboxResult and any runner error.
//
// The child is invoked with -test.coverprofile= set to
// the empty string. When the parent test binary was built
// with coverage, the child would otherwise try to write
// to the parent's coverage profile and fail with a
// permission error.
func runSelf(t *testing.T, r *SubprocessRunner, opts RunOptions, match string) (SandboxResult, error) {
	t.Helper()
	cmd := []string{
		selfExec(t),
		"-test.run=^" + match + "$",
		"-test.coverprofile=",
		"-test.outputdir=" + os.TempDir(),
	}
	return r.Run(mergeOpts(opts, RunOptions{Command: cmd}))
}

// mergeOpts returns a copy of base with the Command
// field replaced by the second argument. Other fields of
// override win on conflict.
func mergeOpts(base, override RunOptions) RunOptions {
	out := base
	if len(override.Command) > 0 {
		out.Command = override.Command
	}
	return out
}

// TestHelperEcho is a no-op when run as a normal test.
// When the test binary is invoked with -test.run matching
// this test, it prints "hello" to stdout and exits 0. The
// runner tests use this to verify stdout capture.
func TestHelperEcho(t *testing.T) {
	if !helperMode() {
		return
	}
	fmt.Print("hello")
}

// TestHelperFail is the failing-process helper. The
// helper exits 7 to verify the runner surfaces non-zero
// exit codes.
func TestHelperFail(t *testing.T) {
	if !helperMode() {
		return
	}
	os.Exit(7)
}

// TestHelperErr is the stderr helper. The helper writes
// "err-aios-sandbox" to stderr to verify stderr capture.
func TestHelperErr(t *testing.T) {
	if !helperMode() {
		return
	}
	fmt.Fprint(os.Stderr, "err-aios-sandbox")
}

// TestHelperSleep is the sleep helper. The helper sleeps
// for 5 seconds. The runner tests use it with a 1-second
// timeout to verify the timeout branch.
func TestHelperSleep(t *testing.T) {
	if !helperMode() {
		return
	}
	time.Sleep(5 * time.Second)
}

// TestHelperPwd is the pwd helper. The helper prints the
// current working directory. The runner tests use it to
// verify the cwd branch.
func TestHelperPwd(t *testing.T) {
	if !helperMode() {
		return
	}
	dir, err := os.Getwd()
	if err != nil {
		fmt.Fprint(os.Stderr, err.Error())
		os.Exit(1)
	}
	fmt.Print(dir)
}

// TestHelperEnv is the env helper. The helper prints
// "$PARENT_LEAK|$FOO" so the test can assert the env is
// isolated.
func TestHelperEnv(t *testing.T) {
	if !helperMode() {
		return
	}
	fmt.Print(os.Getenv("PARENT_LEAK") + "|" + os.Getenv("FOO"))
}

// helperMode reports whether the test binary was invoked
// in helper mode (i.e. by a child runner test, not by
// `go test`). The function checks for a sentinel env var
// that the runner tests set.
func helperMode() bool {
	return os.Getenv("AIOS_SANDBOX_HELPER") == "1"
}

// withHelperEnv returns a copy of opts with the helper
// sentinel env var set, and a writable TMP/TEMP so the
// child test binary's coverage runtime can create a temp
// directory on Windows.
//
// When the parent test runs with -cover, the child test
// binary's Go coverage runtime tries to create a temp
// directory under GOCOVERDIR. On Windows, when TMP and
// TEMP are not set, os.TempDir falls back to
// C:\WINDOWS which is not user-writable. Setting TMP to
// the system temp dir avoids the failure.
func withHelperEnv(opts RunOptions) RunOptions {
	out := opts
	if out.Env == nil {
		out.Env = map[string]string{}
	} else {
		cp := make(map[string]string, len(out.Env)+4)
		for k, v := range out.Env {
			cp[k] = v
		}
		out.Env = cp
	}
	out.Env["AIOS_SANDBOX_HELPER"] = "1"
	// Propagate TMP/TEMP from the parent so the child's
	// os.TempDir returns a writable path.
	out.Env["TMP"] = os.TempDir()
	out.Env["TEMP"] = os.TempDir()
	return out
}

// TestRunEmptyCommand covers the empty-command branch.
// The runner rejects an empty command before forking.
func TestRunEmptyCommand(t *testing.T) {
	r := NewSubprocessRunner()
	_, err := r.Run(RunOptions{Command: nil})
	if err == nil {
		t.Fatalf("expected error for empty command")
	}
	var se *SandboxError
	if !errors.As(err, &se) {
		t.Fatalf("expected *SandboxError, got %T", err)
	}
}

// TestRunSuccess covers the happy path: a small helper
// that prints to stdout and exits 0.
func TestRunSuccess(t *testing.T) {
	r := NewSubprocessRunner()
	opts := withHelperEnv(RunOptions{Timeout: 10})
	res, err := runSelf(t, r, opts, "TestHelperEcho")
	if err != nil {
		t.Fatalf("unexpected error: %v (stderr=%q)", err, res.Stderr)
	}
	if res.ExitCode != 0 {
		t.Fatalf("expected exit 0, got %d (stderr=%q)", res.ExitCode, res.Stderr)
	}
	if !strings.HasPrefix(res.Stdout, "hello") {
		t.Fatalf("expected stdout starting with 'hello', got %q", res.Stdout)
	}
	if res.DurationMs < 0 {
		t.Fatalf("duration is negative: %d", res.DurationMs)
	}
}

// TestRunFailure covers a non-zero exit. The runner
// surfaces the exit code in the result and returns a
// SandboxError wrapping the os/exec wait error.
func TestRunFailure(t *testing.T) {
	r := NewSubprocessRunner()
	opts := withHelperEnv(RunOptions{Timeout: 10})
	res, err := runSelf(t, r, opts, "TestHelperFail")
	if err == nil {
		t.Fatalf("expected error for non-zero exit, got nil")
	}
	var se *SandboxError
	if !errors.As(err, &se) {
		t.Fatalf("expected *SandboxError, got %T", err)
	}
	if res.ExitCode != 7 {
		t.Fatalf("expected exit 7, got %d", res.ExitCode)
	}
}

// TestRunTimeout covers the wall-clock timeout branch.
// The runner raises *SandboxTimeout.
func TestRunTimeout(t *testing.T) {
	r := NewSubprocessRunner()
	opts := withHelperEnv(RunOptions{Timeout: 1})
	_, err := runSelf(t, r, opts, "TestHelperSleep")
	if err == nil {
		t.Fatalf("expected timeout error, got nil")
	}
	var st *SandboxTimeout
	if !errors.Is(err, &SandboxTimeout{}) || !errors.As(err, &st) {
		t.Fatalf("expected *SandboxTimeout, got %T: %v", err, err)
	}
	if st.TimeoutSec != 1 {
		t.Fatalf("unexpected timeout: %d", st.TimeoutSec)
	}
}

// TestRunEnv covers the environment clear-and-set branch.
// The runner must not leak the parent's env: the child
// receives only the entries in opts.Env.
func TestRunEnv(t *testing.T) {
	r := NewSubprocessRunner()
	t.Setenv("PARENT_LEAK", "should-not-leak")
	opts := withHelperEnv(RunOptions{
		Env:     map[string]string{"FOO": "bar"},
		Timeout: 10,
	})
	res, err := runSelf(t, r, opts, "TestHelperEnv")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(res.Stdout, "|bar") {
		t.Fatalf("env not isolated: got %q", res.Stdout)
	}
}

// TestRunCwd covers the working-directory branch. The
// runner must start the process in opts.Cwd.
func TestRunCwd(t *testing.T) {
	r := NewSubprocessRunner()
	dir := t.TempDir()
	// On Windows, t.TempDir may return a path with 8.3
	// short names. Resolve the canonical long path so the
	// child sees the same string.
	resolved, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatalf("EvalSymlinks: %v", err)
	}
	opts := withHelperEnv(RunOptions{Cwd: resolved, Timeout: 10})
	res, err := runSelf(t, r, opts, "TestHelperPwd")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.ExitCode != 0 {
		t.Fatalf("expected exit 0, got %d stderr=%q", res.ExitCode, res.Stderr)
	}
	if !strings.HasPrefix(res.Stdout, resolved) {
		t.Fatalf("cwd not honored: got %q want %q", res.Stdout, resolved)
	}
}

// TestRunStderr covers the stderr capture branch. The
// runner captures both streams into the result.
func TestRunStderr(t *testing.T) {
	r := NewSubprocessRunner()
	opts := withHelperEnv(RunOptions{Timeout: 10})
	res, err := runSelf(t, r, opts, "TestHelperErr")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(res.Stderr, "err-aios-sandbox") {
		t.Fatalf("stderr not captured: %q", res.Stderr)
	}
}

// TestRunInvalidLimits covers the validation branch. The
// runner rejects limits with a negative time limit
// before forking. The runner overrides TimeSeconds with
// the resolved timeout, so the test exercises a
// non-time field that is still invalid.
func TestRunInvalidLimits(t *testing.T) {
	r := NewSubprocessRunner()
	bad := ResourceLimits{TimeSeconds: 10, CPUSeconds: -1}
	_, err := r.Run(RunOptions{
		Command: []string{selfExec(t), "-test.run=^TestHelperEcho$"},
		Env:     map[string]string{"AIOS_SANDBOX_HELPER": "1"},
		Limits:  &bad,
		Timeout: 10,
	})
	if err == nil {
		t.Fatalf("expected validation error")
	}
	var se *SandboxError
	if !errors.As(err, &se) {
		t.Fatalf("expected *SandboxError, got %T", err)
	}
}

// TestRunDefaultTimeout covers the default-timeout branch.
// A zero or negative Timeout becomes DefaultTimeout (60).
func TestRunDefaultTimeout(t *testing.T) {
	r := NewSubprocessRunner()
	opts := withHelperEnv(RunOptions{Command: []string{selfExec(t), "-test.run=^TestHelperEcho$"}})
	res, err := r.Run(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.ExitCode != 0 {
		t.Fatalf("expected exit 0, got %d", res.ExitCode)
	}
}

// TestRunMissingBinary covers the launch-failure branch.
// The runner returns *SandboxError wrapping the os/exec
// error.
func TestRunMissingBinary(t *testing.T) {
	r := NewSubprocessRunner()
	_, err := r.Run(RunOptions{
		Command: []string{"this-binary-does-not-exist-aios-sandbox"},
		Timeout: 5,
	})
	if err == nil {
		t.Fatalf("expected error for missing binary")
	}
	var se *SandboxError
	if !errors.As(err, &se) {
		t.Fatalf("expected *SandboxError, got %T", err)
	}
	if se.Op != "start" && se.Op != "wait" {
		t.Fatalf("unexpected op: %q", se.Op)
	}
}

// TestRunMergedLimits covers the limits-merge branch in
// Run. The runner copies the caller's limits and overrides
// the time limit with the resolved timeout.
func TestRunMergedLimits(t *testing.T) {
	r := NewSubprocessRunner()
	limits := ResourceLimits{
		CPUSeconds:  5,
		MemoryBytes: 1 << 30,
		Processes:   4,
		TimeSeconds: 999, // gets overridden by the timeout
	}
	opts := withHelperEnv(RunOptions{
		Limits:  &limits,
		Timeout: 10,
	})
	res, err := runSelf(t, r, opts, "TestHelperEcho")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.ExitCode != 0 {
		t.Fatalf("expected exit 0, got %d", res.ExitCode)
	}
}

// TestSandboxErrorIsTyped covers the typed-error match
// using errors.As.
func TestSandboxErrorIsTyped(t *testing.T) {
	err := (&SandboxError{Op: "test", Err: errors.New("boom")}).Error()
	if !strings.HasPrefix(err, "sandbox: test:") {
		t.Fatalf("unexpected message: %q", err)
	}
	if !strings.HasSuffix(err, ".") {
		t.Fatalf("missing trailing period: %q", err)
	}
}

// TestSandboxErrorUnwrap confirms the wrapped error is
// reachable through errors.Unwrap. The chain enables
// callers to match on the inner error too.
func TestSandboxErrorUnwrap(t *testing.T) {
	inner := errors.New("inner")
	se := &SandboxError{Op: "test", Err: inner}
	if got := errors.Unwrap(se); got != inner {
		t.Fatalf("expected inner, got %v", got)
	}
}

// TestSandboxTimeoutIs wraps the context error and is
// matchable via errors.Is.
func TestSandboxTimeoutIs(t *testing.T) {
	st := &SandboxTimeout{TimeoutSec: 3, Wrapped: errDeadline}
	if !errors.Is(st, &SandboxTimeout{}) {
		t.Fatalf("expected errors.Is to match")
	}
	if got := errors.Unwrap(st); got != errDeadline {
		t.Fatalf("expected unwrap to return errDeadline, got %v", got)
	}
	msg := st.Error()
	if !strings.Contains(msg, "3 seconds") {
		t.Fatalf("missing timeout value: %q", msg)
	}
	if !strings.HasSuffix(msg, ".") {
		t.Fatalf("missing trailing period: %q", msg)
	}
}

// TestSandboxResourceLimitIs is the typed-match test for
// the resource-limit error.
func TestSandboxResourceLimitIs(t *testing.T) {
	inner := errors.New("boom")
	sr := &SandboxResourceLimit{Field: "cpu_seconds", Reason: inner}
	if !errors.Is(sr, &SandboxResourceLimit{}) {
		t.Fatalf("expected errors.Is to match")
	}
	if !strings.Contains(sr.Error(), "cpu_seconds") {
		t.Fatalf("missing field name: %q", sr.Error())
	}
	if got := errors.Unwrap(sr); got != inner {
		t.Fatalf("expected unwrap to return inner, got %v", got)
	}
}

// TestNewSubprocessRunnerWithClockNil covers the nil-clock
// fallback. The constructor substitutes time.Now when
// the caller passes nil.
func TestNewSubprocessRunnerWithClockNil(t *testing.T) {
	r := NewSubprocessRunnerWithClock(nil)
	if r == nil {
		t.Fatalf("expected runner, got nil")
	}
	if r.now == nil {
		t.Fatalf("expected now to be set, got nil")
	}
	// The fallback should be time.Now, which is non-nil.
	_ = r.now()
}

// TestInjectedClock confirms the injected clock is used
// for duration. The fake clock returns a fixed time; the
// test asserts the duration is zero (no elapsed wall clock
// between start and Wait).
func TestInjectedClock(t *testing.T) {
	fixed := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	now := func() time.Time { return fixed }
	r := NewSubprocessRunnerWithClock(now)
	opts := withHelperEnv(RunOptions{Timeout: 10})
	res, err := runSelf(t, r, opts, "TestHelperEcho")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.ExitCode != 0 {
		t.Fatalf("expected exit 0, got %d", res.ExitCode)
	}
	if res.DurationMs != 0 {
		t.Fatalf("expected zero duration with fixed clock, got %d", res.DurationMs)
	}
}

// TestRunnableHelper is a guard test that prevents
// `helperMode` from being called as a real test on a
// regular `go test` run. The helper tests no-op when the
// env var is missing.
func TestRunnableHelper(t *testing.T) {
	if helperMode() {
		t.Fatalf("helper env var leaked into the parent test run")
	}
}

// errDeadline is the test-local deadline-exceeded marker.
var errDeadline = errors.New("deadline exceeded")
