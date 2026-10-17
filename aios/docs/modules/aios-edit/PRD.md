# aios-edit — Product Requirements Document

## 1. Purpose

This document states the product requirements for the
aios-edit module. The module is the code-edit tool
set. The tool set lets the agent change source files
through patch operations. The tool set covers three
formats: search-replace, old-string new-string, and
apply_patch.

## 2. Scope

The aios-edit module is in scope for this document.

- The Edit tool (old-string + new-string, optional
  replace_all).
- The MultiEdit tool (a list of edits in one call).
- The apply_patch tool (unified-diff style).
- The NotebookEdit tool (Jupyter notebook cell
  edits).
- The Write tool (whole-file overwrite, already
  partly in aios-tools; the module re-exports the
  tool under a unified name).

The aios-edit module is not in scope for this
document.

- The Read tool. The Read tool is in aios-tools.
- The Bash tool. The Bash tool is in aios-tools.
- The Grep and the Glob tools. The tools are in
  aios-search.

## 3. Reference Documents

- Project design document, section 7.
- aios-core PRD. See `docs/modules/aios-core/PRD.md`.
- aios-tools PRD. See `docs/modules/aios-tools/PRD.md`.
- Claude Code Edit tool reference.
- Aider edit-formats reference.
- OpenAI apply_patch reference.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Edit | One change to one file at one location. |
| Patch | A text block that describes an edit through a diff. |
| old_string | The exact text to find in the file. |
| new_string | The text to replace the old_string with. |
| replace_all | The flag to replace every match of old_string. |
| SEARCH/REPLACE block | The Aider-style edit format. |
| apply_patch | The OpenAI-style diff edit format. |
| Notebook cell | A single cell in a Jupyter notebook file. |


## 5. Determinism

The whole point of aios is to make complex coding and large systems deterministic. The determinism stance of this module is **Bounded non-determinism**.

File system is the only source of non-determinism. The edit tool is deterministic for a fixed file content.

The module provides the following determinism guarantees.

- Edits are content addressed.

- The edit format is a pure function of the input and the file content.
## 6. User Personas

The user persona is the agent that needs to change
source files without rewriting the whole file. The
agent needs precise, auditable, and safe edits.

## 7. User Stories

An agent wants to rename a function. The agent calls
the Edit tool with the old_string and the new_string.
The tool replaces the function. The tool returns the
diff.

An agent wants to change many lines at once. The
agent calls the MultiEdit tool with a list of edits.
The tool applies every edit. The tool returns the
list of diffs.

An agent wants to apply a unified diff. The agent
calls the apply_patch tool with the diff text. The
tool parses the diff. The tool applies the diff. The
tool returns the result.

An agent wants to edit a Jupyter notebook. The agent
calls the NotebookEdit tool with the cell id, the
new source, and the cell type. The tool updates the
cell. The tool returns the result.

## 8. Functional Requirements Summary

The aios-edit module provides the following
functions.

- The Edit tool for old-string + new-string edits.
- The MultiEdit tool for a list of edits.
- The apply_patch tool for unified-diff edits.
- The NotebookEdit tool for Jupyter cell edits.
- The Write tool for whole-file overwrite.
- The Write permission check (whole-file writes
  require approval tier-3 by default).

## 9. Non-Functional Requirements Summary

- The module runs on Go 1.23.
- The module depends on aios-core and aios-tools.
- The module has 100 percent test coverage for the
  public functions.
- Every edit operation is atomic. The operation
  either succeeds for all targets or fails for all
  targets.

## 10. Success Criteria

- The Edit tool replaces a unique match.
- The Edit tool fails on a non-unique match when
  replace_all is false.
- The Edit tool replaces every match when
  replace_all is true.
- The MultiEdit tool applies the edits in order.
- The MultiEdit tool rolls back on the first
  failure.
- The apply_patch tool applies a unified diff.
- The apply_patch tool rejects a malformed diff.
- The NotebookEdit tool updates a cell.
- The Write tool writes the file through the
  atomic-write path.
- The test suite has a 100 percent pass rate.

## 11. Out of Scope

- The user interface for the edits.
- The remote access to the edits.
- The undo stack.
