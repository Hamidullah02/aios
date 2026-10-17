# aios-edit — Technical Requirements Document

## 1. Purpose

This document states the technical approach for the
aios-edit module.

## 2. Scope

The scope of this document is the aios-edit module.
The module is a Go package.

## 3. Reference Documents

- aios-edit PRD. See `docs/modules/aios-edit/PRD.md`.
- aios-edit SRS. See `docs/modules/aios-edit/SRS.md`.
- aios-core TRD. See `docs/modules/aios-core/TRD.md`.
- aios-tools TRD. See `docs/modules/aios-tools/TRD.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| internal/diff | A home-grown unified-diff parser. |
| internal/nbformat | A home-grown `.ipynb` JSON reader. |

## 5. Architecture

The aios-edit module has five components.

- `aios.edit.edit` — the Edit tool.
- `aios.edit.multi_edit` — the MultiEdit tool.
- `aios.edit.apply_patch` — the apply_patch tool.
- `aios.edit.notebook_edit` — the NotebookEdit
  tool.
- `internal/edit/write` — the Write tool.

The five components depend on aios-core and
aios-tools. The components depend on each other in
this order. The Edit tool has no dependencies. The
MultiEdit tool depends on the Edit tool. The
apply_patch tool depends on the unidiff library.
The NotebookEdit tool depends on the nbformat
library. The Write tool depends on aios-tools.

## 6. Components

### 6.1 Edit Tool

The Edit tool uses the file read and write tools
from aios-tools. The tool reads the file. The tool
counts the matches of old_string. The tool validates
the match count. The tool replaces the matches. The
tool writes the file. The tool returns the diff.

The tool uses the audit log to write the edit row.
The tool writes the diff in unified-diff format.

### 6.2 MultiEdit Tool

The MultiEdit tool applies a list of edits. The
tool reads the original file. The tool applies the
edits in memory. The tool writes the file once at
the end. The tool rolls back when an edit fails.

The tool uses a staging area. The staging area
holds the file content during the edit. The tool
writes the staging area to the file on success. The
tool discards the staging area on failure.

### 6.3 apply_patch Tool

The apply_patch tool uses the `unidiff` library.
The tool parses the patch. The tool iterates over
the file blocks. The tool applies the hunks in
order.

The tool validates the hunk context. The tool
reads the lines around the hunk. The tool compares
the lines with the hunk context. The tool raises
`PatchContextMismatch` when the lines do not match.

### 6.4 NotebookEdit Tool

The NotebookEdit tool uses the `nbformat` library.
The tool reads the notebook. The tool finds the
cell. The tool updates the source. The tool writes
the notebook.

The tool supports the `replace`, `insert`, and
`delete` modes. The `insert` mode adds a new cell
after the target cell. The `delete` mode removes
the cell.

### 6.5 Write Tool

The Write tool re-exports the write-file tool from
aios-tools. The tool requires a tier-3 approval by
default. The tool uses the atomic write path.

## 7. Data Structures

### 7.1 EditSpec

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `path` | string | yes | The path of the file. |
| `old_string` | string | yes | The text to find. |
| `new_string` | string | yes | The replacement text. |
| `replace_all` | bool | no | Replace every match. |

### 7.2 NotebookEditSpec

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `path` | string | yes | The path of the notebook. |
| `cell_id` | string | yes | The id of the cell. |
| `new_source` | string | no | The new source. |
| `cell_type` | string | yes | The cell type. |
| `cell_mode` | string | no | The edit mode. |

## 8. Interfaces

### 8.1 EditTool

```go
type EditTool:
    """The edit tool."""

    func run(self, args: dict, context: ToolContext) -> ToolResult:
        """Run the edit tool.

        Args:
            args: The arguments of the call.
            context: The context for the call.

        Returns:
            The result of the call.
        """
```

## 9. Algorithms

### 9.1 Edit Algorithm

The edit algorithm works as follows.

1. Read the file.
2. Count the matches of old_string.
3. When replace_all is false and the count is not
   one, raise the appropriate error.
4. When the count is zero, raise
   `EditNotFoundError`.
5. Replace the matches with new_string.
6. Compute the diff.
7. Write the file through the atomic write.
8. Write the audit row.
9. Return the result with the diff.

### 9.2 MultiEdit Rollback Algorithm

The MultiEdit rollback algorithm works as follows.

1. Keep a list of the original file content.
2. Apply the edits in order.
3. When an edit fails, restore the original file
   content.
4. Write the original content back to the file.
5. Raise `MultiEditError`.

### 9.3 Patch Validation Algorithm

The patch validation algorithm works as follows.

1. For each hunk, read the lines around the hunk
   in the file.
2. Compare the lines with the hunk context.
3. When the lines do not match, raise
   `PatchContextMismatch`.

## 10. Dependencies

### 10.1 Runtime Dependencies

- Go 1.23
- unidiff
- nbformat
- aios-core
- aios-tools

### 10.2 Module Dependencies

The aios-edit module depends on the following
internal modules.

- aios-core
- aios-tools

The aios-edit module is depended on by the following
internal modules.

- aios-runtime
- aios-cli

## 11. Deployment

The module compiles into the monorepo under `internal/<module>/`. The
package name is `github.com/<org>/aios/internal/edit`. The module is
registered in the tool bus.

## 12. Risks and Mitigations

| Risk | Impact | Probability | Mitigation |
| --- | --- | --- | --- |
| The Edit tool corrupts the file. | High | Low | The tool uses the atomic write. The tool validates the match count. |
| The MultiEdit tool leaves the file half-edited. | High | Low | The tool rolls back on the first failure. The test suite covers the rollback. |
| The apply_patch tool accepts a bad diff. | High | Low | The tool validates the hunk context. The tool rejects malformed diffs. |
| The NotebookEdit tool corrupts the notebook. | High | Low | The tool uses the nbformat library. The tool validates the cell type. |
