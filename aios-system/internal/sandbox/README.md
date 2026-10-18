# aios-sandbox

The aios-sandbox module runs untrusted commands in a safe
environment. The module is a small Go package at
`internal/sandbox/` and depends only on the Go standard
library.

## Public surface

- `Runner` — the runner interface.
- `SubprocessRunner` — the os/exec wrapper.
- `NewSubprocessRunner()` — constructor with the system clock.
- `NewSubprocessRunnerWithClock(now)` — constructor with an
  injected clock.
- `RunOptions` — the input to `Run`.
- `SandboxResult` — the output of `Run`.
- `ResourceLimits` — the typed limits object.
- `SandboxError` — the base error type.
- `SandboxTimeout` — the timeout error.
- `SandboxResourceLimit` — the resource-limit error.
- `LimitsError` — the limits-validation error.
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

- POSIX (Linux, macOS): the resource limits are applied
  through `syscall.Setrlimit`. The CPU, memory, and
  process limits are honored. The wall-clock timeout is
  honored as a fallback.
- Windows: the resource limits are a no-op. The wall-clock
  timeout is the only bound. The runner falls back to
  the timeout on Windows because `syscall.Setrlimit` is
  not available there.

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
