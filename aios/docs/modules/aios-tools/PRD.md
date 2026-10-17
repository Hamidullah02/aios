# aios-tools — Product Requirements Document

## 1. Purpose

This document states the product requirements for the
aios-tools module. The module is the tool bus and the
built-in tools. The tool bus dispatches tool calls
to the right tool. The built-in tools cover file
read, file write, file search, and shell run.

## 2. Scope

The aios-tools module is in scope for this document.

- The tool bus.
- The file read tool.
- The file write tool.
- The file search tool.
- The shell run tool.

The aios-tools module is not in scope for this
document.

- The sandbox runner. The runner is in aios-sandbox.
- The egress proxy. The proxy is in aios-egress.
- The ask-user tool. The tool is in aios-ask.

## 3. Reference Documents

- Project design document, sections 7, 11.
- aios-core PRD. See `docs/modules/aios-core/PRD.md`.
- aios-policy PRD. See `docs/modules/aios-policy/PRD.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Tool | A function the agent can call. |
| Tool bus | A component that dispatches tool calls. |
| Tool call | A request to invoke a tool with arguments. |
| Tool result | The return value of a tool call. |


## 5. Determinism

The whole point of aios is to make complex coding and large systems deterministic. The determinism stance of this module is **Bounded non-determinism**.

File system is the only source of non-determinism. The tool bus and the policy gateway are deterministic.

The module provides the following determinism guarantees.

- Tool calls are content addressed.

- Tool results are content addressed.

- The bus is a pure function of the call and the policy.
## 6. User Personas

The user persona is a Go developer who needs
safe file and shell operations.

## 7. User Stories

A developer wants to read a file. The developer calls
the file read tool. The tool returns the file
content.

A developer wants to write a file. The developer
calls the file write tool. The tool writes the
content to the file.

A developer wants to search files. The developer
calls the file search tool. The tool returns the
matching files.

A developer wants to run a shell command. The
developer calls the shell run tool. The tool runs
the command through the sandbox. The tool returns
the result.

## 8. Functional Requirements Summary

The aios-tools module provides the following
functions.

- The tool bus that dispatches calls.
- The file read tool.
- The file write tool.
- The file search tool.
- The shell run tool.

## 9. Non-Functional Requirements Summary

- The module runs on Go 1.23.
- The module depends on aios-core, aios-policy,
  aios-sandbox, and the Go standard library.
- The module has 100 percent test coverage for the
  public functions.
- The tools return within the budget limits.

## 10. Success Criteria

- The tool bus dispatches every tool.
- The file read tool returns the file content.
- The file write tool writes the file content.
- The file search tool returns the matching files.
- The shell run tool runs the command.
- The test suite has a 100 percent pass rate.

## 11. Out of Scope

- The user interface for the tools.
- The remote access to the tools.
- The tool marketplace.

## 12. Open Questions

- Does the file write tool support atomic writes?
  The current design uses a temp file and rename.
- Does the file search tool support content search?
  The current design supports filename search.
