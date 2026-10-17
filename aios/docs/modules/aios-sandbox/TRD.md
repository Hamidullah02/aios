# aios-sandbox — Technical Requirements Document

## 1. Purpose

This document states the technical approach for the
aios-sandbox module.

## 2. Scope

The scope of this document is the aios-sandbox module.
The module is a Go package.

## 3. Reference Documents

- aios-sandbox PRD. See `docs/modules/aios-sandbox/PRD.md`.
- aios-sandbox SRS. See `docs/modules/aios-sandbox/SRS.md`.
- aios-core TRD. See `docs/modules/aios-core/TRD.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| goroutines + channels + errgroup | The Go concurrency primitives for async I/O. |
| syscall.Setrlimit | The POSIX resource-limit API, called through syscall. |
| os/exec | The Go standard-library external command runner. |

## 5. Architecture

The aios-sandbox module has three components.

- `internal/sandbox/runner` — the runner interface.
- `aios.sandbox.subprocess` — the os/exec wrapper.
- `aios.sandbox.limits` — the resource limits.

The three components depend on aios-core. The
components depend on each other in this order. The
runner interface has no dependencies. The subprocess
runner depends on the runner interface and the
resource limits.

## 6. Components

### 6.1 Runner interface

The runner interface has the `run` method. The method
accepts the command, the env, the cwd, the timeout,
and the limits. The method returns the
`SandboxResult`.

### 6.2 os/exec wrapper

The os/exec wrapper uses
`os/exec`. The runner sets
the working directory. The runner sets the
environment variables. The runner applies the
resource limits before the start. The runner waits
for the process with the timeout.

### 6.3 Resource Limits

The resource limits component uses the `resource`
module. The component sets the CPU limit, the memory
limit, and the process limit. The component applies
the limits through `resource.setrlimit`.

## 7. Data Structures

### 7.1 ResourceLimits

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `cpu_seconds` | int | no | The CPU limit in seconds. |
| `memory_bytes` | int | no | The memory limit in bytes. |
| `time_seconds` | int | yes | The time limit in seconds. |
| `processes` | int | no | The process limit. |

### 7.2 SandboxResult

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `exit_code` | int | yes | The exit code of the process. |
| `stdout` | string | yes | The stdout of the process. |
| `stderr` | string | yes | The stderr of the process. |
| `duration_ms` | int | yes | The duration in milliseconds. |

## 8. Interfaces

### 8.1 SubprocessRunner

```go
type SubprocessRunner:
    """The os/exec wrapper."""

    func run(
        self,
        command: list[str],
        env: dict[str, str] | nil = nil,
        cwd: str | nil = nil,
        timeout: int = 60,
        limits: ResourceLimits | nil = nil,
    ) -> SandboxResult:
        """Run a command in the sandbox.

        Args:
            command: The command to run.
            env: The environment variables.
            cwd: The working directory.
            timeout: The timeout in seconds.
            limits: The resource limits.

        Returns:
            The result of the run.
        """
```

## 9. Algorithms

### 9.1 Run Algorithm

The run algorithm works as follows.

1. Set the working directory.
2. Set the environment variables.
3. Apply the resource limits.
4. Start the process.
5. Wait for the process with the timeout.
6. When the process exceeds the timeout, kill the
   process and raise `SandboxTimeout`.
7. Return the `SandboxResult`.

### 9.2 Resource Limit Algorithm

The resource limit algorithm works as follows.

1. On POSIX, call `resource.setrlimit` for each
   limit.
2. On Windows, skip the limit. The runner has a
   timeout fallback.

## 10. Dependencies

### 10.1 Runtime Dependencies

- Go 1.23
- aios-core

### 10.2 Module Dependencies

The aios-sandbox module depends on the following
internal modules.

- aios-core

The aios-sandbox module is depended on by the following
internal modules.

- aios-tools

## 11. Deployment

The module compiles into the monorepo under `internal/<module>/`. The
package name is `github.com/<org>/aios/internal/sandbox`.

## 12. Risks and Mitigations

| Risk | Impact | Probability | Mitigation |
| --- | --- | --- | --- |
| The process exceeds the resource limit. | Medium | Medium | The runner applies the limit. The runner kills the process on the limit. |
| The process writes a large file. | Medium | Medium | The runner enforces a disk quota through the file system. |
| The process spawns child processes. | Medium | Low | The runner sets the process limit. The runner kills the process group. |
