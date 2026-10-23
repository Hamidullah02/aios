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
	"strings"
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
	// Network is the network policy this runner applies to
	// every child it starts.
	//
	// The policy lives on the runner rather than on each
	// RunOptions because the runner is the sandbox. A policy
	// that had to be passed at every call site could be
	// forgotten at exactly one call site -- the one that runs
	// the interesting command -- and the failure would be
	// silent. Here, a runner with a policy cannot start a
	// child without it.
	//
	// A nil policy means no network policy, which is the
	// historical behaviour.
	Network *NetworkPolicy
}

// NewSubprocessRunner constructs a SubprocessRunner that
// uses time.Now for duration measurement. Tests should
// use NewSubprocessRunnerWithClock instead.
func NewSubprocessRunner() *SubprocessRunner {
	return &SubprocessRunner{now: time.Now}
}

// NewSubprocessRunnerWithNetwork constructs a runner that applies
// a network policy to every child.
//
// It is the constructor a production wiring uses: the proxy is
// already running, and the runner is told to point children at it.
func NewSubprocessRunnerWithNetwork(policy *NetworkPolicy) *SubprocessRunner {
	return &SubprocessRunner{now: time.Now, Network: policy}
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

// mergeEnv merges the caller's environment map over a list of
// "KEY=VALUE" strings.
//
// The map wins: an explicit opts.Env entry is a deliberate
// choice by the caller and overrides the inherited value. The
// result is a map because the network policy has to rewrite
// individual keys, and doing that on a slice would mean parsing
// and re-serialising every entry.
func mergeEnv(overrides map[string]string, base []string) map[string]string {
	out := make(map[string]string, len(base)+len(overrides))
	for _, entry := range base {
		key, value, ok := strings.Cut(entry, "=")
		if !ok || key == "" {
			continue
		}
		out[key] = value
	}
	for k, v := range overrides {
		out[k] = v
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
	// Network is the child's network policy.
	//
	// A nil Network means no policy: the child gets the
	// environment the caller asked for, unchanged. That is the
	// historical behaviour, and a caller that has not opted in
	// is not silently changed.
	//
	// When set, the policy REWRITES the child's environment:
	// the proxy variables are replaced with the policy's, and
	// any proxy variable the caller had is removed. See
	// NetworkPolicy.Apply for why removal matters.
	Network *NetworkPolicy
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
	// The network policy is applied LAST, so its proxy
	// variables win over anything the caller or the parent
	// environment supplied. A child that could read a
	// different proxy out of its environment could route
	// around the one the policy chose.
	//
	// The runner's own policy applies when the call site did
	// not set one. The runner is the sandbox: a child started
	// through it inherits the runner's policy unless the caller
	// explicitly overrides it.
	policy := opts.Network
	if policy == nil {
		policy = r.Network
	}
	if policy != nil {
		if err := policy.Validate(); err != nil {
			return SandboxResult{}, &SandboxError{Op: "network policy", Err: err}
		}
		// The policy is applied to the FULL environment the
		// child would have had, not just to opts.Env. The
		// inherited variables are part of what the child
		// sees, and a proxy variable hiding in there is
		// exactly the escape hatch the policy exists to
		// close. Applying the policy to opts.Env alone
		// would leave the inherited HTTP_PROXY in place for
		// an InheritEnv caller, which is the run_tests tool.
		env = formatEnv(policy.Apply(mergeEnv(opts.Env, env)))
	}
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
	// Process-tree containment. On POSIX the child is put in
	// its own process group; on Windows the child is put in a
	// Job Object. Both give the runner one handle on the whole
	// tree, so a command that backgrounds work does not leave
	// grandchildren running past the timeout.
	configureProcessGroup(cmd)
	job, jobErr := startJobObject(&limits)
	if jobErr != nil {
		restoreParentLimits()
		return SandboxResult{}, &SandboxError{Op: "start job object", Err: jobErr}
	}
	if job != nil {
		defer job.close()
	}
	// The Cancel hook fires when the context is done, which is
	// the timeout path. It kills the tree rather than just the
	// direct child, so a `sh -c "sleep 999 &"` does not leave
	// the sleep behind.
	//
	// The default Cancel is replaced because the default only
	// kills the direct child.
	cmd.Cancel = func() error {
		killProcessTree(cmd, job)
		return nil
	}
	// The Job Object assignment has to happen before the child
	// can spawn anything, or a fast child could escape the job
	// before the assignment lands. The child is created
	// suspended and resumed after the assignment.
	if job != nil {
		prepareJobAssignment(cmd)
	}
	runErr := cmd.Start()
	if runErr != nil {
		restoreParentLimits()
		if job != nil {
			job.close()
		}
		return SandboxResult{}, &SandboxError{Op: "start", Err: runErr}
	}
	// The Job Object assignment has to happen before the child
	// can spawn anything, or a fast child could escape the job
	// before the assignment lands.
	if job != nil {
		if err := job.assignProcess(cmd); err != nil {
			killProcessTree(cmd, job)
			restoreParentLimits()
			return SandboxResult{}, &SandboxError{Op: "assign job object", Err: err}
		}
		if err := resumeAfterAssignment(cmd); err != nil {
			killProcessTree(cmd, job)
			restoreParentLimits()
			return SandboxResult{}, &SandboxError{Op: "resume child", Err: err}
		}
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
