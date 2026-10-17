# aios-search — Software Requirements Specification

## 1. Purpose

This document states the software requirements for the
aios-search module.

## 2. Scope

The scope of this document is the aios-search module.
The module is a library.

## 3. Reference Documents

- aios-search PRD. See `docs/modules/aios-search/PRD.md`.
- aios-core SRS. See `docs/modules/aios-core/SRS.md`.
- aios-tools SRS. See `docs/modules/aios-tools/SRS.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Glob | A typed object that holds a glob pattern. |
| Match | A typed object that holds a content match. |

## 5. Functional Requirements

### 5.1 Glob Tool

**REQ-SRH-GLB-001** The module shall provide the
`GlobTool` class. The type shall implement the
`Tool` protocol.

**REQ-SRH-GLB-002** The tool shall accept a
`pattern` argument. The pattern is a glob. The tool
shall return the matching paths.

**REQ-SRH-GLB-003** The tool shall support an
`include` glob. The tool shall support an `exclude`
glob.

**REQ-SRH-GLB-004** The tool shall respect the
`.aiosignore` file from aios-memory.

**REQ-SRH-GLB-005** The tool shall respect the
workspace root. The tool shall not return paths
outside the workspace.

**REQ-SRH-GLB-006** The tool shall write an audit
row. The row type shall be `glob_search`.

### 5.2 Grep Tool

**REQ-SRH-GRP-001** The module shall provide the
`GrepTool` class. The type shall implement the
`Tool` protocol.

**REQ-SRH-GRP-002** The tool shall accept a
`pattern` argument. The pattern is a regex. The
tool shall return the matching lines.

**REQ-SRH-GRP-003** The tool shall accept a
`path` argument. The tool shall search the path
and the subdirectories.

**REQ-SRH-GRP-004** The tool shall accept a
`glob` argument. The tool shall restrict the
search to files that match the glob.

**REQ-SRH-GRP-005** The tool shall accept an
`output_mode` argument. The supported modes are
`content`, `files_with_matches`, and `count`.

**REQ-SRH-GRP-006** The tool shall accept a
`-n` flag. The flag adds line numbers to the
content output.

**REQ-SRH-GRP-007** The tool shall use ripgrep as
the backend.

**REQ-SRH-GRP-008** The tool shall respect the
`.aiosignore` file.

**REQ-SRH-GRP-009** The tool shall write an audit
row. The row type shall be `grep_search`.

### 5.3 CodeSearch Tool

**REQ-SRH-CDS-001** The module shall provide the
`CodeSearchTool` class. The type shall implement
the `Tool` protocol.

**REQ-SRH-CDS-002** The tool shall accept a
`symbol` argument. The tool shall accept a
`kind` argument. The supported kinds are
`function`, `class`, `method`, and `variable`.

**REQ-SRH-CDS-003** The tool shall use tree-sitter
to parse the source files. The tool shall use the
tree-sitter language pack for the file extension.

**REQ-SRH-CDS-004** The tool shall return the
location of the symbol. The location includes the
file path, the line, and the column.

**REQ-SRH-CDS-005** The tool shall write an audit
row. The row type shall be `code_search`.

### 5.4 Diagnostics Tool

**REQ-SRH-DGN-001** The module shall provide the
`DiagnosticsTool` class. The type shall implement
the `Tool` protocol.

**REQ-SRH-DGN-002** The tool shall accept a `path`
argument. The tool shall accept a `checker`
argument. The supported checkers are `go vet`,
`gofmt + go vet`, `pylint`, and `eslint`.

**REQ-SRH-DGN-003** The tool shall run the
checker as a subprocess. The tool shall parse the
checker output.

**REQ-SRH-DGN-004** The tool shall return the
diagnostics. Each diagnostic includes the file
path, the line, the column, the severity, and the
message.

**REQ-SRH-DGN-005** The tool shall write an audit
row. The row type shall be `diagnostics`.

## 6. Non-Functional Requirements

**REQ-SRH-NFR-001** The module shall run on Go 1.23
3.12.

**REQ-SRH-NFR-002** The module shall have 100 percent
coverage for the public functions.

**REQ-SRH-NFR-003** The Grep tool shall search
1 MB of source code in under 1 second on a modern
laptop.

## 7. Interface Requirements

**REQ-SRH-IFC-001** The module shall export the
following items from `aios.search`.

- `GlobTool`
- `GrepTool`
- `CodeSearchTool`
- `DiagnosticsTool`
- `Match`
- `Diagnostic`

## 8. Data Requirements

**REQ-SRH-DAT-001** The `Match` schema shall
include the fields below.

- `path` (string)
- `line` (int)
- `column` (int, optional)
- `text` (string)
- `context_before` (list of string, optional)
- `context_after` (list of string, optional)

**REQ-SRH-DAT-002** The `Diagnostic` schema shall
include the fields below.

- `path` (string)
- `line` (int)
- `column` (int, optional)
- `severity` (string: error, warning, info)
- `code` (string, optional)
- `message` (string)

## 9. Error Handling

**REQ-SRH-ERR-001** The module shall raise
`CheckerNotFoundError` when the checker binary is
not on the PATH.

**REQ-SRH-ERR-002** The module shall raise
`GrepTimeoutError` when ripgrep exceeds the
timeout.

**REQ-SRH-ERR-003** The module shall raise
`UnsupportedLanguageError` when tree-sitter has
no grammar for the file.

## 10. Acceptance Criteria

1. Every REQ in this document has a test case.
2. The test suite has a 100 percent pass rate.
3. The test suite has 100 percent coverage for the
   public functions.
4. The Glob tool returns the matching paths.
5. The Grep tool returns the matching lines.
6. The CodeSearch tool returns the symbol
   location.
7. The Diagnostics tool returns the diagnostics.
8. The tools respect the `.aiosignore` file.
