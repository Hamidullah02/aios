# aios-edit — Software Requirements Specification

## 1. Purpose

This document states the software requirements for the
aios-edit module.

## 2. Scope

The scope of this document is the aios-edit module.
The module is a library. The module exports the
edit-tool classes.

## 3. Reference Documents

- aios-edit PRD. See `docs/modules/aios-edit/PRD.md`.
- aios-core SRS. See `docs/modules/aios-core/SRS.md`.
- aios-tools SRS. See `docs/modules/aios-tools/SRS.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Edit | A typed object that describes one change. |
| Patch | A text block in unified-diff format. |
| Match | The location in the file where old_string is found. |

## 5. Functional Requirements

### 5.1 Edit Tool

**REQ-EDT-EDT-001** The module shall provide the
`EditTool` class. The type shall implement the
`Tool` protocol from aios-tools.

**REQ-EDT-EDT-002** The tool shall accept a `path`
argument, an `old_string` argument, a `new_string`
argument, and a `replace_all` flag.

**REQ-EDT-EDT-003** The default for `replace_all`
shall be `false`.

**REQ-EDT-EDT-004** When `replace_all` is false and
the old_string is not unique, the tool shall raise
`EditNotUniqueError`. The error shall include the
match count.

**REQ-EDT-EDT-005** When the old_string is not
found, the tool shall raise `EditNotFoundError`.

**REQ-EDT-EDT-006** The tool shall reject empty
old_string values. The tool shall raise
`EditEmptyError`.

**REQ-EDT-EDT-007** The tool shall preserve the
file encoding. The tool shall read the file as
text, apply the edit, and write the file back in
the same encoding.

**REQ-EDT-EDT-008** The tool shall write an audit
row for every edit. The row type shall be
`file_edit`. The row shall include the path, the
match count, the duration, and the diff.

**REQ-EDT-EDT-009** The tool shall return a
`ToolResult` with the diff in the output.

### 5.2 MultiEdit Tool

**REQ-EDT-MED-001** The module shall provide the
`MultiEditTool` class. The type shall implement
the `Tool` protocol.

**REQ-EDT-MED-002** The tool shall accept a `path`
argument and an `edits` argument. The `edits`
argument shall be a list of edit objects.

**REQ-EDT-MED-003** The tool shall apply the edits
in the order the edits are listed.

**REQ-EDT-MED-004** The tool shall roll back every
applied edit when any edit fails. The tool shall
raise `MultiEditError` on the first failure.

**REQ-EDT-MED-005** The tool shall write one audit
row for the whole operation. The row type shall be
`file_multi_edit`. The row shall include the path,
the edit count, and the combined diff.

### 5.3 apply_patch Tool

**REQ-EDT-APP-001** The module shall provide the
`ApplyPatchTool` class. The type shall implement
the `Tool` protocol.

**REQ-EDT-APP-002** The tool shall accept a `patch`
argument. The `patch` is a string in unified-diff
format.

**REQ-EDT-APP-003** The tool shall parse the
unified diff. The tool shall use the `unidiff`
library or an internal parser.

**REQ-EDT-APP-004** The tool shall apply the diff
to the target file. The tool shall fail when the
hunk context does not match the file content.

**REQ-EDT-APP-005** The tool shall support a
multi-file patch. The tool shall apply every
file block in the patch.

**REQ-EDT-APP-006** The tool shall reject a
malformed patch. The tool shall raise
`PatchParseError`.

**REQ-EDT-APP-007** The tool shall write one audit
row per file. The row type shall be `file_patch`.
The row shall include the path and the diff.

### 5.4 NotebookEdit Tool

**REQ-EDT-NTE-001** The module shall provide the
`NotebookEditTool` class. The type shall implement
the `Tool` protocol.

**REQ-EDT-NTE-002** The tool shall accept a `path`
argument, a `cell_id` argument, a `new_source`
argument, and a `cell_type` argument.

**REQ-EDT-NTE-003** The supported cell types are
`code` and `markdown`.

**REQ-EDT-NTE-004** The tool shall support a
`cell_mode` argument. The supported modes are
`replace`, `insert`, and `delete`.

**REQ-EDT-NTE-005** The tool shall use the `nbformat`
library to read and write the notebook.

**REQ-EDT-NTE-006** The tool shall write an audit
row for every edit. The row type shall be
`notebook_edit`.

### 5.5 Write Tool

**REQ-EDT-WRT-001** The module shall provide the
`WriteTool` class. The type shall re-export the
write-file tool from aios-tools under a unified
name. The type shall implement the `Tool`
protocol.

**REQ-EDT-WRT-002** The tool shall require a
tier-3 approval by default. The policy gateway
shall return the ask decision.

**REQ-EDT-WRT-003** The tool shall use the atomic
write path from aios-tools.

## 6. Non-Functional Requirements

**REQ-EDT-NFR-001** The module shall run on Go 1.23
3.12.

**REQ-EDT-NFR-002** The module shall have 100 percent
coverage for the public functions.

**REQ-EDT-NFR-003** Every edit operation shall be
atomic. The operation shall not leave the file in
a half-edited state.

## 7. Interface Requirements

**REQ-EDT-IFC-001** The module shall export the
following items from `aios.edit`.

- `EditTool`
- `MultiEditTool`
- `ApplyPatchTool`
- `NotebookEditTool`
- `WriteTool`
- `EditNotUniqueError`
- `EditNotFoundError`
- `EditEmptyError`
- `MultiEditError`
- `PatchParseError`

## 8. Data Requirements

**REQ-EDT-DAT-001** The Edit schema shall include
the fields below.

- `path` (string)
- `old_string` (string)
- `new_string` (string)
- `replace_all` (bool, default false)

**REQ-EDT-DAT-002** The audit row for an edit
shall include the diff. The diff shall be a unified
diff format.

## 9. Error Handling

**REQ-EDT-ERR-001** The module shall raise
`EditNotUniqueError` when the old_string matches
more than one location and replace_all is false.

**REQ-EDT-ERR-002** The module shall raise
`EditNotFoundError` when the old_string does not
match.

**REQ-EDT-ERR-003** The module shall raise
`PatchParseError` when the patch is malformed.

**REQ-EDT-ERR-004** The module shall raise
`PatchContextMismatch` when the hunk context does
not match.

## 10. Acceptance Criteria

1. Every REQ in this document has a test case.
2. The test suite has a 100 percent pass rate.
3. The test suite has 100 percent coverage for the
   public functions.
4. The Edit tool replaces a unique match.
5. The Edit tool fails on a non-unique match.
6. The Edit tool replaces every match on
   replace_all.
7. The MultiEdit tool applies the edits in order.
8. The MultiEdit tool rolls back on the first
   failure.
9. The apply_patch tool applies a unified diff.
10. The apply_patch tool rejects a malformed diff.
11. The NotebookEdit tool updates a cell.
12. The Write tool writes through the atomic
    write path.
