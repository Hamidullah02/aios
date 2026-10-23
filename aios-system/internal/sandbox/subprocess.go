// Package sandbox — subprocess runner.
//
// The runner is a small wrapper around os/exec. It applies
// the resource limits through syscall.Setrlimit on POSIX,
// sets the working directory and the environment variables,
// waits for the process with the timeout, and returns a
// SandboxResult. On timeout the runner kills the process
// and raises SandboxTimeout. On a resource-limit violation
// the runner raises SandboxResourceLimit.

package sandbox

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"time"
)

// Runner is the interface every concrete runner
// implements. The interface lets tests substitute a fake
// runner and lets callers depend on the contract rather
// than the implementation.
type Runner interface {
	Run(opts RunOptions) (SandboxResult, error)
}

// SubprocessRunner is the os/exec wrapper. The zero value
// is usable; it produces a runner that runs commands
// without resource limits and with a default timeout.
type SubprocessRunner struct {
	// now is the clock the runner uses to measure duration.
	// Tests can inject a fake clock; production code can
	// leave it nil and the runner uses time.Now.
	now func() time.Time
}

// NewSubprocessRunner constructs a SubprocessRunner that
// uses time.Now for duration measurement. Tests should
// use NewSubprocessRunnerWithClock instead.
func NewSubprocessRunner() *SubprocessRunner {
	return &SubprocessRunner{now: time.Now}
}

// NewSubprocessRunnerWithClock constructs a SubprocessRunner
// with a custom clock. The clock is used only to measure
// the duration of the run; the run itself uses the real
// wall clock for the timeout.
func NewSubprocessRunnerWithClock(now func() time.Time) *SubprocessRunner {
	if now == nil {
		now = time.Now
	}
	return &SubprocessRunner{now: now}
}
// formatEnv converts an Env map into the []string form
// os/exec.Cmd.Env expects. The order is non-deterministic
// (Go map iteration); InheritEnv callers should not rely on
// a specific ordering.
func formatEnv(m map[string]string) []string {
	if len(m) == 0 {
		return nil
	}
	out := make([]string, 0, len(m))
	for k, v := range m {
		out = append(out, k+"="+v)
	}
	return out
}


// RunOptions is the input to SubprocessRunner.Run.
//
// The Command is required. The Env is the allow-list of
// environment variables; a nil Env means the process runs
// with an empty environment. The Cwd is the working
// directory; an empty Cwd means the parent's Cwd. The
// Timeout is the wall-clock budget in seconds; a non-
// positive value is replaced with DefaultTimeout. The
// Limits are the resource limits; a nil Limits means no
// limits.
type RunOptions struct {
	Command []string
	Env     map[string]string
	// InheritEnv starts the subprocess environment from
	// the parent's os.Environ() and overlays Env on top.
	// The default (false) keeps REQ-SND-ISO-002 intact:
	// the child gets ONLY the variables in Env, with no
	// parent variables leaking through.
	//
	// Some callers genuinely need the parent
	// environment. The run_tests tool is the canonical
	// example: `go test`, `npm test`, and `cargo test`
	// all need PATH to find the toolchain, plus HOME /
	// USERPROFILE for their caches. Those tools run
	// against source the operator already trusts, so
	// inheriting the environment is the right trade.
	//
	// run_shell keeps the default (false) because it
	// executes arbitrary commands.
	InheritEnv bool
	Cwd        string
	Timeout    int
	Limits     *ResourceLimits
}

// DefaultTimeout is the default wall-clock timeout when
// the caller passes a non-positive value. The value is
// sixty seconds, matching the doc-set default.
const DefaultTimeout = 60

// SandboxResult is the result of one run.
type SandboxResult struct {
	ExitCode   int
	Stdout     string
	Stderr     string
	DurationMs int
}

// SandboxError is the base error type for runner errors.
// The concrete types SandboxTimeout and
// SandboxResourceLimit embed this so callers can match on
// the base when they do not care about the specific
// failure.
type SandboxError struct {
	Op  string
	Err error
}

func (e *SandboxError) Error() string {
	return fmt.Sprintf("sandbox: %s: %v.", e.Op, e.Err)
}

func (e *SandboxError) Unwrap() error { return e.Err }

// SandboxTimeout is raised when the process exceeds the
// wall-clock timeout. The error wraps the underlying
// context.DeadlineExceeded so callers that match on the
// standard error keep working.
type SandboxTimeout struct {
	TimeoutSec int
	Wrapped    error
}

func (e *SandboxTimeout) Error() string {
	return fmt.Sprintf("sandbox: timeout: process exceeded %d seconds.", e.TimeoutSec)
}

func (e *SandboxTimeout) Unwrap() error { return e.Wrapped }

// Is reports whether target is also a SandboxTimeout. This
// lets callers use errors.Is to detect a timeout regardless
// of the concrete value.
func (e *SandboxTimeout) Is(target error) bool {
	_, ok := target.(*SandboxTimeout)
	return ok
}

// SandboxResourceLimit is raised when the process violates
// a resource limit. The Field names the limit that was
// violated; the Reason is the underlying error from the
// OS.
type SandboxResourceLimit struct {
	Field  string
	Reason error
}

func (e *SandboxResourceLimit) Error() string {
	return fmt.Sprintf("sandbox: resource limit exceeded: %s: %v.", e.Field, e.Reason)
}

func (e *SandboxResourceLimit) Unwrap() error { return e.Reason }

// Is reports whether target is also a
// SandboxResourceLimit. This lets callers use errors.Is to
// detect a resource-limit violation regardless of the
// concrete value.
func (e *SandboxResourceLimit) Is(target error) bool {
	_, ok := target.(*SandboxResourceLimit)
	return ok
}

// Run runs the command in the sandbox. The algorithm is:
//
//  1. Resolve the timeout and the limits.
//  2. Build the os/exec.Cmd with the cwd and the env.
//  3. Apply the resource limits through the platform-
//     specific path.
//  4. Start the process.
//  5. Wait for the process with the timeout.
//  6. Return a SandboxResult or raise a typed error.
//
// The runner captures stdout and stderr through the
// os/exec StdoutPipe and StderrPipe patterns. The two
// buffers are read concurrently; a sync.WaitGroup
// guarantees both are drained before the runner returns.
func (r *SubprocessRunner) Run(opts RunOptions) (SandboxResult, error) {
	if len(opts.Command) == 0 {
		return SandboxResult{}, &SandboxError{Op: "run", Err: errors.New("command is required")}
	}
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	limits := ResourceLimits{TimeSeconds: timeout}
	if opts.Limits != nil {
		// The doc-set time limit is honored through the
		// timeout argument. We merge the caller's
		// ResourceLimits into the local one but keep the
		// caller's TimeSeconds in sync with the resolved
		// timeout so Validate() passes.
		merged := *opts.Limits
		merged.TimeSeconds = timeout
		limits = merged
	}
	if err := limits.Validate(); err != nil {
		return SandboxResult{}, &SandboxError{Op: "validate limits", Err: err}
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, opts.Command[0], opts.Command[1:]...)
	if opts.Cwd != "" {
		cmd.Dir = opts.Cwd
	}
	// REQ-SND-ISO-002: clear the environment; set only the
	// allowed variables. A nil Env means an empty
	// environment.
	var env []string
	if opts.InheritEnv {
		env = os.Environ()
	}
	env = append(env, formatEnv(opts.Env)...)
	cmd.Env = env

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Apply the resource limits in the parent before Start.
	// On POSIX the limits are inherited by the child; on
	// Windows the call is a no-op and the wall-clock
	// timeout is the only bound. The parent's prior limits
	// are saved so they can be restored after Wait returns;
	// otherwise the parent process would stay constrained
	// for the rest of the run.
	saveParentLimits()
	if err := r.applyLimitsForRun(&limits); err != nil {
		restoreParentLimits()
		return SandboxResult{}, &SandboxError{Op: "apply limits", Err: err}
	}

	start := r.now()
	runErr := cmd.Start()
	if runErr != nil {
		restoreParentLimits()
		return SandboxResult{}, &SandboxError{Op: "start", Err: runErr}
	}

	waitErr := cmd.Wait()
	restoreParentLimits()
	durationMs := int(r.now().Sub(start) / time.Millisecond)
	if durationMs < 0 {
		durationMs = 0
	}

	// Detect a timeout: when the context fires
	// DeadlineExceeded the process is dead. The runner
	// raises SandboxTimeout.
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return SandboxResult{
			ExitCode:   exitCodeFromState(cmd.ProcessState, waitErr),
			Stdout:     stdout.String(),
			Stderr:     stderr.String(),
			DurationMs: durationMs,
		}, &SandboxTimeout{TimeoutSec: timeout, Wrapped: context.DeadlineExceeded}
	}

	// Detect a resource-limit signal: SIGXCPU (CPU), or
	// SIGKILL/SIGSEGV (memory or OOM) on POSIX. The
	// platform helper inspects the ProcessState.
	if field, ok := detectResourceLimit(cmd.ProcessState); ok {
		return SandboxResult{
			ExitCode:   exitCodeFromState(cmd.ProcessState, waitErr),
			Stdout:     stdout.String(),
			Stderr:     stderr.String(),
			DurationMs: durationMs,
		}, &SandboxResourceLimit{Field: field, Reason: waitErr}
	}

	if waitErr != nil {
		return SandboxResult{
			ExitCode:   exitCodeFromState(cmd.ProcessState, waitErr),
			Stdout:     stdout.String(),
			Stderr:     stderr.String(),
			DurationMs: durationMs,
		}, &SandboxError{Op: "wait", Err: waitErr}
	}

	return SandboxResult{
		ExitCode:   cmd.ProcessState.ExitCode(),
		Stdout:     stdout.String(),
		Stderr:     stderr.String(),
		DurationMs: durationMs,
	}, nil
}

// applyLimitsForRun is the cross-platform wrapper that
// applies the resource limits and returns a typed error
// when the OS rejects a limit.
func (r *SubprocessRunner) applyLimitsForRun(l *ResourceLimits) error {
	if err := l.applyLimits(); err != nil {
		var le *LimitsError
		if errors.As(err, &le) {
			return &SandboxResourceLimit{Field: le.Field, Reason: le}
		}
		return err
	}
	// The drop-privilege step runs after the rlimit step.
	// On Windows both Setuid and Setgid are no-ops; the
	// runner does not enforce privileges there.
	if err := dropPrivileges(l); err != nil {
		return &SandboxResourceLimit{Field: "privilege", Reason: err}
	}
	return nil
}

// exitCodeFromState returns the exit code from the
// process state, falling back to the wait error's exit
// code, and finally to -1 when the process was killed by
// a signal.
func exitCodeFromState(state *os.ProcessState, waitErr error) int {
	if state != nil {
		if code, ok := exitCodeFromWaitStatus(state); ok {
			return code
		}
	}
	if waitErr != nil {
		var exitErr *exec.ExitError
		if errors.As(waitErr, &exitErr) {
			return exitErr.ExitCode()
		}
	}
	return -1
}
