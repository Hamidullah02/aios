# aios-sandbox

The aios-sandbox module runs untrusted commands in a contained
environment. The module is a Go package at `internal/sandbox/`.

## Public surface

- `Runner` — the runner interface.
- `SubprocessRunner` — the os/exec wrapper.
- `NewSubprocessRunner()` — constructor with the system clock.
- `NewSubprocessRunnerWithClock(now)` — constructor with an
  injected clock.
- `NewSubprocessRunnerWithNetwork(policy)` — constructor with a
  network policy.
- `RunOptions` — the input to `Run`.
- `SandboxResult` — the output of `Run`.
- `ResourceLimits` — the typed limits object.
- `NetworkPolicy` — the network scope the runner applies.
- `NamespacePolicy` — the Linux namespace scope the runner
  applies.
- `DefaultNamespacePolicy()` — the namespace policy a Linux run
  uses.
- `NamespacesSupported()` — whether the platform and the kernel
  allow the namespaces.
- `DefaultRunner()` / `SetDefaultRunner(r)` — the process-wide
  default runner.
- `SandboxError` — the base error type.
- `SandboxTimeout` — the timeout error.
- `SandboxResourceLimit` — the resource-limit error.
- `LimitsError` — the limits-validation error.
- `NetworkError` — the network-policy-validation error.
- `NamespaceError` — the namespace-policy-validation error.
- `DefaultTimeout` — the default wall-clock timeout (60s).

## Algorithm

The run algorithm is:

1. Resolve the timeout (default `DefaultTimeout`).
2. Validate the limits.
3. Build the `os/exec.Cmd` with the cwd and the env.
4. Apply the resource limits through the platform-specific
   path.
5. Start the process.
6. Wait for the process with the timeout.
7. Return a `SandboxResult` or raise a typed error.

## Platform support

- Linux: the resource limits are applied through
  `syscall.Setrlimit` via `golang.org/x/sys/unix`. The CPU,
  memory, and process limits are honored. The runner also puts the
  child in its own user, network, mount, and pid namespaces when
  the kernel allows them.
- macOS: the resource limits are applied through
  `syscall.Setrlimit` via `golang.org/x/sys/unix`. The CPU and
  memory limits are honored. The process-count limit is NOT
  honored, because macOS has no `RLIMIT_NPROC`. The child is put
  in its own process group.
- Windows: the resource limits are enforced by a Job Object
  (`JOB_OBJECT_LIMIT_ACTIVE_PROCESS`,
  `JOB_OBJECT_LIMIT_JOB_MEMORY`,
  `JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE`). The wall-clock timeout
  is the other bound. The child is created suspended and resumed
  after the job assignment lands, so a grandchild cannot escape
  the job.

## Process-tree containment

The child is put in its own process group on POSIX and its own Job
Object on Windows. The timeout kills the whole tree, not just the
direct child, so a command that backgrounds work does not leave
that work running.

## Path isolation

The runner sets the cwd through `cmd.Dir` and clears the
environment. The child receives only the env vars the
caller passed in `RunOptions.Env`. A nil `Env` produces
an empty environment.

The runner drops privileges on POSIX when the
`ResourceLimits.UID` and `ResourceLimits.GID` fields are
non-negative. The drop happens after the rlimit step and
before `cmd.Start`.

## Error handling

The runner returns one of three typed errors:

- `*SandboxTimeout` — the process exceeded the timeout.
- `*SandboxResourceLimit` — the process exceeded a
  resource limit. The `Field` names the limit.
- `*SandboxError` — any other failure (validation,
  launch, wait).

Callers can match with `errors.Is` or `errors.As`.

## Usage

```go
runner := sandbox.NewSubprocessRunner()
limits := &sandbox.ResourceLimits{
    CPUSeconds:  5,
    MemoryBytes: 1 << 30,
    TimeSeconds: 60,
}
result, err := runner.Run(sandbox.RunOptions{
    Command: []string{"echo", "hello"},
    Env:     map[string]string{"FOO": "bar"},
    Cwd:     "/tmp",
    Timeout: 30,
    Limits:  limits,
})
if err != nil {
    var st *sandbox.SandboxTimeout
    if errors.As(err, &st) {
        // process exceeded the timeout
    }
    return err
}
fmt.Println(result.Stdout)
```

## Tests

The tests run on Windows and POSIX. The runner tests
spawn the test binary itself as a child process so the
tests do not depend on `/bin/sh` or any other shell. The
tests set `TMP` and `TEMP` on the child so the Go
coverage runtime can write its temp directory when the
parent runs with `-cover`.

Coverage on the package is 86.5% on Windows. The
uncovered branches are error paths that are only
reachable on POSIX (e.g. `RLIMIT_CPU` exceeded) or
unexported helpers.
