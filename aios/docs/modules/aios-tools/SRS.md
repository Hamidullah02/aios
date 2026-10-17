# aios-tools — Software Requirements Specification

## 1. Purpose

This document states the software requirements for the
aios-tools module.

## 2. Scope

The scope of this document is the aios-tools module.
The module is a library.

## 3. Reference Documents

- aios-tools PRD. See `docs/modules/aios-tools/PRD.md`.
- aios-core SRS. See `docs/modules/aios-core/SRS.md`.
- aios-policy SRS. See `docs/modules/aios-policy/SRS.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Tool | A typed function the agent can call. |
| Tool call | A typed request to invoke a tool. |
| Tool result | A typed response from a tool. |

## 5. Functional Requirements

### 5.1 Tool Bus

**REQ-TOL-BUS-001** The module shall provide the
`ToolBus` class. The type shall accept the policy
gateway, the audit log, the sandbox, and the tool
map.

**REQ-TOL-BUS-002** The bus shall provide the
`dispatch` method. The method shall accept the tool
name, the arguments, the actor, and the task id.

**REQ-TOL-BUS-003** The bus shall call the policy
gateway before the tool. The bus shall deny the call
when the gateway denies it.

**REQ-TOL-BUS-004** The bus shall call the sandbox
before the shell tool. The bus shall deny the call
when the sandbox denies it.

**REQ-TOL-BUS-005** The bus shall write an audit
row for every tool call. The row type shall be
`tool_call`. The row shall include the tool name,
the arguments, the policy decision, the duration, and
the result summary.

### 5.2 File Read Tool

**REQ-TOL-RD-001** The module shall provide the
`ReadFileTool` class. The type shall implement the
`Tool` protocol.

**REQ-TOL-RD-002** The tool shall accept a `path`
argument. The tool shall return the file content.

**REQ-TOL-RD-003** The tool shall reject paths
outside the workspace. The tool shall raise
`PathNotAllowedError`.

**REQ-TOL-RD-004** The tool shall support line
offsets. The tool shall support line limits.

### 5.3 File Write Tool

**REQ-TOL-WR-001** The module shall provide the
`WriteFileTool` class. The type shall implement the
`Tool` protocol.

**REQ-TOL-WR-002** The tool shall accept `path` and
`content` arguments. The tool shall write the
content to the file.

**REQ-TOL-WR-003** The tool shall reject paths
outside the workspace.

**REQ-TOL-WR-004** The tool shall support atomic
writes through a temp file and rename.

### 5.4 File Search Tool

**REQ-TOL-SR-001** The module shall provide the
`SearchFilesTool` class. The type shall implement
the `Tool` protocol.

**REQ-TOL-SR-002** The tool shall accept a `pattern`
argument. The pattern is a glob. The tool shall
return the matching paths.

**REQ-TOL-SR-003** The tool shall support an
`include` glob. The tool shall support an `exclude`
glob.

### 5.5 Shell Run Tool

**REQ-TOL-SH-001** The module shall provide the
`RunShellTool` class. The type shall implement the
`Tool` protocol.

**REQ-TOL-SH-002** The tool shall accept a
`command` argument. The tool shall run the command
through the sandbox.

**REQ-TOL-SH-003** The tool shall support a
`timeout` argument. The tool shall support a
`cwd` argument. The tool shall support an
`env` argument.

**REQ-TOL-SH-004** The tool shall return the exit
code, the stdout, and the stderr.

## 6. Non-Functional Requirements

**REQ-TOL-NFR-001** The module shall run on Go 1.23
3.12.

**REQ-TOL-NFR-002** The module shall have 100 percent
coverage for the public functions.

**REQ-TOL-NFR-003** The tools shall be safe. The
tools shall not access paths outside the workspace.

## 7. Interface Requirements

**REQ-TOL-IFC-001** The module shall export the
following items from `aios.tools`.

- `Tool`
- `ToolBus`
- `ReadFileTool`
- `WriteFileTool`
- `SearchFilesTool`
- `RunShellTool`

## 8. Data Requirements

**REQ-TOL-DAT-001** The workspace root shall be
configurable. The default is the current working
directory.

## 9. Error Handling

**REQ-TOL-ERR-001** The module shall raise
`PathNotAllowedError` when a path is outside the
workspace.

**REQ-TOL-ERR-002** The module shall raise
`ToolTimeoutError` when a tool exceeds the timeout.

**REQ-TOL-ERR-003** The module shall raise
`ToolNotFoundError` when a tool name is not in the
tool map.

## 10. Acceptance Criteria

1. Every REQ in this document has a test case.
2. The test suite has a 100 percent pass rate.
3. The test suite has 100 percent coverage for the
   public functions.
4. The tool bus dispatches every tool.
5. The file read tool returns the file content.
6. The file write tool writes the file content.
7. The file search tool returns the matching paths.
8. The shell run tool runs the command.
9. The tools reject paths outside the workspace.
