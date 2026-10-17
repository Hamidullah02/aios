# aios-sandbox — Software Requirements Specification

## 1. Purpose

This document states the software requirements for the
aios-sandbox module.

## 2. Scope

The scope of this document is the aios-sandbox module.
The module is a library.

## 3. Reference Documents

- aios-sandbox PRD. See `docs/modules/aios-sandbox/PRD.md`.
- aios-core SRS. See `docs/modules/aios-core/SRS.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Runner | A typed component that runs a process. |
| Resource limits | A typed object that holds the limits. |

## 5. Functional Requirements

### 5.1 Subprocess Runner

**REQ-SND-RUN-001** The module shall provide the
`SubprocessRunner` class. The type shall implement
the `Runner` protocol.

**REQ-SND-RUN-002** The runner shall accept a
`command` argument. The runner shall accept an
`env` argument. The runner shall accept a `cwd`
argument. The runner shall accept a `timeout`
argument.

**REQ-SND-RUN-003** The runner shall start the
process through `os/exec`.

**REQ-SND-RUN-004** The runner shall return the
exit code, the stdout, and the stderr.

### 5.2 Resource Limits

**REQ-SND-LIM-001** The module shall provide the
`ResourceLimits` class. The type shall accept the
CPU limit, the memory limit, the time limit, and the
process limit.

**REQ-SND-LIM-002** The runner shall enforce the
CPU limit through `resource.RLIMIT_CPU`.

**REQ-SND-LIM-003** The runner shall enforce the
memory limit through `resource.RLIMIT_AS`.

**REQ-SND-LIM-004** The runner shall enforce the
time limit through the timeout argument.

### 5.3 Path Isolation

**REQ-SND-ISO-001** The runner shall change the
working directory to the workspace.

**REQ-SND-ISO-002** The runner shall clear the
environment variables. The runner shall set only the
allowed variables.

**REQ-SND-ISO-003** The runner shall drop the
process privileges on POSIX systems. The runner
shall use `os.setuid` and `os.setgid` when the
target user is set.

## 6. Non-Functional Requirements

**REQ-SND-NFR-001** The module shall run on Go 1.23
3.12.

**REQ-SND-NFR-002** The module shall have 100 percent
coverage for the public functions.

## 7. Interface Requirements

**REQ-SND-IFC-001** The module shall export the
following items from `aios.sandbox`.

- `Runner`
- `SubprocessRunner`
- `ResourceLimits`
- `SandboxResult`

## 8. Data Requirements

**REQ-SND-DAT-001** The `SandboxResult` shall
include the fields below.

- `exit_code` (int)
- `stdout` (string)
- `stderr` (string)
- `duration_ms` (int)

## 9. Error Handling

**REQ-SND-ERR-001** The module shall raise
`SandboxTimeout` when the process exceeds the
timeout.

**REQ-SND-ERR-002** The module shall raise
`SandboxResourceLimit` when the process exceeds a
resource limit.

## 10. Acceptance Criteria

1. Every REQ in this document has a test case.
2. The test suite has a 100 percent pass rate.
3. The test suite has 100 percent coverage for the
   public functions.
4. The runner runs a command.
5. The resource limits are enforced.
6. The path isolation is enforced.
