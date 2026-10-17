# aios-tools — Application Flow

## 1. Purpose

This document describes the application flow for the
aios-tools module.

## 2. Scope

The scope of this document is the aios-tools module.
The module is a library.

## 3. Reference Documents

- aios-tools PRD. See `docs/modules/aios-tools/PRD.md`.
- aios-tools SRS. See `docs/modules/aios-tools/SRS.md`.
- aios-tools TRD. See `docs/modules/aios-tools/TRD.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Caller | The code that calls the tool bus. |

## 5. User Flow

The aios-tools module has no user-facing flow.

## 6. System Flow

### 6.1 Dispatch Flow

The dispatch flow runs as follows.

1. The caller calls `bus.dispatch` with the tool
   call.
2. The bus looks up the tool by name.
3. The bus calls the policy gateway.
4. The gateway returns the decision.
5. On allow, the bus calls the tool.
6. On deny, the bus returns an error.
7. The tool returns the result.
8. The bus writes the audit row.
9. The bus returns the result.

### 6.2 Read File Flow

The read file flow runs as follows.

1. The caller calls `read_file.run` with the
   arguments.
2. The tool validates the path.
3. The tool reads the file.
4. The tool returns the content.

### 6.3 Write File Flow

The write file flow runs as follows.

1. The caller calls `write_file.run` with the
   arguments.
2. The tool validates the path.
3. The tool creates the temp file.
4. The tool writes the content to the temp file.
5. The tool renames the temp file to the target.
6. The tool returns the result.

### 6.4 Search Files Flow

The search files flow runs as follows.

1. The caller calls `search_files.run` with the
   arguments.
2. The tool walks the workspace.
3. The tool matches the paths against the glob.
4. The tool returns the matching paths.

### 6.5 Run Shell Flow

The run shell flow runs as follows.

1. The caller calls `run_shell.run` with the
   arguments.
2. The tool validates the command.
3. The tool calls the sandbox.
4. The sandbox returns the result.
5. The tool returns the result.

## 7. Error Flows

### 7.1 Path Not Allowed

The path not allowed flow runs as follows.

1. The caller passes a path outside the workspace.
2. The tool raises `PathNotAllowedError`.
3. The caller catches the error.
4. The caller reports the error.

### 7.2 Tool Timeout

The tool timeout flow runs as follows.

1. The tool exceeds the timeout.
2. The tool raises `ToolTimeoutError`.
3. The caller catches the error.
4. The caller reports the error.

### 7.3 Tool Not Found

The tool not found flow runs as follows.

1. The caller passes an unknown tool name.
2. The bus raises `ToolNotFoundError`.
3. The caller catches the error.
4. The caller reports the error.

## 8. Sequence Diagrams

### 8.1 Dispatch Sequence

The dispatch sequence has the steps below.

1. The orchestrator calls `bus.dispatch`.
2. The bus calls the policy gateway.
3. The gateway returns the decision.
4. The bus calls the tool.
5. The tool returns the result.
6. The bus writes the audit row.
7. The bus returns the result.
