# aios-sandbox — Application Flow

## 1. Purpose

This document describes the application flow for the
aios-sandbox module.

## 2. Scope

The scope of this document is the aios-sandbox module.
The module is a library.

## 3. Reference Documents

- aios-sandbox PRD. See `docs/modules/aios-sandbox/PRD.md`.
- aios-sandbox SRS. See `docs/modules/aios-sandbox/SRS.md`.
- aios-sandbox TRD. See `docs/modules/aios-sandbox/TRD.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Caller | The code that calls the runner. |

## 5. User Flow

The aios-sandbox module has no user-facing flow.

## 6. System Flow

### 6.1 Run Flow

The run flow runs as follows.

1. The caller calls `runner.run` with the command
   and the options.
2. The runner sets the working directory.
3. The runner sets the environment variables.
4. The runner applies the resource limits.
5. The runner starts the process.
6. The runner waits for the process.
7. The runner returns the `SandboxResult`.

### 6.2 Timeout Flow

The timeout flow runs as follows.

1. The process exceeds the timeout.
2. The runner kills the process.
3. The runner raises `SandboxTimeout`.
4. The caller catches the error.
5. The caller reports the error.

## 7. Error Flows

### 7.1 Resource Limit Exceeded

The resource limit exceeded flow runs as follows.

1. The process exceeds a resource limit.
2. The runner kills the process.
3. The runner raises `SandboxResourceLimit`.
4. The caller catches the error.
5. The caller reports the error.

## 8. Sequence Diagrams

### 8.1 Run Sequence

The run sequence has the steps below.

1. The tool calls `runner.run`.
2. The runner applies the limits.
3. The runner starts the process.
4. The process runs.
5. The process ends.
6. The runner returns the result.
