# aios-search — Product Requirements Document

## 1. Purpose

This document states the product requirements for the
aios-search module. The module is the code-search
tool set. The tool set lets the agent search through
source files. The tool set covers filename glob
search, content search, and code structure search.

## 2. Scope

The aios-search module is in scope for this document.

- The Glob tool (filename match).
- The Grep tool (content search with ripgrep).
- The CodeSearch tool (tree-sitter based symbol
  search).
- The Diagnostics tool (lint and type-check
  results).

The aios-search module is not in scope for this
document.

- The Read tool. The Read tool is in aios-tools.
- The Edit tool. The Edit tool is in aios-edit.

## 3. Reference Documents

- Project design document, section 7.
- aios-core PRD. See `docs/modules/aios-core/PRD.md`.
- aios-tools PRD. See `docs/modules/aios-tools/PRD.md`.
- Claude Code Grep and Glob tool reference.
- ripgrep documentation.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Glob | A pattern for filename matching. |
| Grep | A pattern for content matching. |
| Regex | A regular expression for pattern matching. |
| Symbol | A named program element: function, class, variable. |
| Diagnostics | Lint and type-check results for source code. |


## 5. Determinism

The whole point of aios is to make complex coding and large systems deterministic. The determinism stance of this module is **Bounded non-determinism**.

File system is the only source of non-determinism. The search results are deterministic for a fixed file system state.

The module provides the following determinism guarantees.

- Search results are content addressed.

- The result format is a pure function of the file content.
## 6. User Personas

The user persona is the agent that needs to find
files, lines, or symbols in the workspace.

## 7. User Stories

An agent wants to find a file by name. The agent
calls the Glob tool. The tool returns the matching
paths.

An agent wants to find a line by content. The agent
calls the Grep tool. The tool returns the matching
lines.

An agent wants to find a function by name. The agent
calls the CodeSearch tool. The tool returns the
function location.

An agent wants to check for type errors. The agent
calls the Diagnostics tool. The tool runs the
type-checker. The tool returns the diagnostics.

## 8. Functional Requirements Summary

The aios-search module provides the following
functions.

- The Glob tool for filename matching.
- The Grep tool for content matching.
- The CodeSearch tool for symbol search.
- The Diagnostics tool for code checks.

## 9. Non-Functional Requirements Summary

- The module runs on Go 1.23.
- The module depends on aios-core, aios-tools, and
  a direct invocation of the `rg` binary.
- The module has 100 percent test coverage for the
  public functions.
- The Grep tool searches 1 MB of source code in
  under 1 second.

## 10. Success Criteria

- The Glob tool returns the matching paths.
- The Grep tool returns the matching lines.
- The CodeSearch tool returns the symbol location.
- The Diagnostics tool returns the diagnostics.
- The test suite has a 100 percent pass rate.

## 11. Out of Scope

- The user interface for the searches.
- The remote access to the searches.
- The semantic search through embeddings.
