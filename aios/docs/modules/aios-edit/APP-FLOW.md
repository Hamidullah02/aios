# aios-edit — Application Flow

## 1. Purpose

This document describes the application flow for the
aios-edit module.

## 2. Scope

The scope of this document is the aios-edit module.
The module is a library.

## 3. Reference Documents

- aios-edit PRD. See `docs/modules/aios-edit/PRD.md`.
- aios-edit SRS. See `docs/modules/aios-edit/SRS.md`.
- aios-edit TRD. See `docs/modules/aios-edit/TRD.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Agent | The code that calls the edit tool. |

## 5. User Flow

The aios-edit module has no user-facing flow.

## 6. System Flow

### 6.1 Edit Flow

The edit flow runs as follows.

1. The agent calls the Edit tool.
2. The tool reads the file.
3. The tool counts the matches of old_string.
4. The tool validates the match count.
5. The tool replaces the matches.
6. The tool writes the file through the atomic
   write.
7. The tool writes the audit row.
8. The tool returns the diff.

### 6.2 MultiEdit Flow

The MultiEdit flow runs as follows.

1. The agent calls the MultiEdit tool.
2. The tool reads the file.
3. The tool iterates over the edits.
4. For each edit, the tool applies the edit to
   the in-memory content.
5. When an edit fails, the tool restores the
   original content and writes it back.
6. On success, the tool writes the final content.
7. The tool writes the audit row.
8. The tool returns the combined diff.

### 6.3 apply_patch Flow

The apply_patch flow runs as follows.

1. The agent calls the apply_patch tool.
2. The tool parses the patch.
3. The tool iterates over the file blocks.
4. For each block, the tool reads the file.
5. The tool validates the hunk context.
6. The tool applies the hunks.
7. The tool writes the file.
8. The tool writes the audit row.

### 6.4 NotebookEdit Flow

The NotebookEdit flow runs as follows.

1. The agent calls the NotebookEdit tool.
2. The tool reads the notebook.
3. The tool finds the cell.
4. The tool updates the cell.
5. The tool writes the notebook.
6. The tool writes the audit row.

## 7. Error Flows

### 7.1 Edit Not Unique

The edit not unique flow runs as follows.

1. The old_string matches more than one location.
2. The tool raises `EditNotUniqueError`.
3. The agent catches the error.
4. The agent adds more context to the old_string.

### 7.2 Edit Not Found

The edit not found flow runs as follows.

1. The old_string does not match.
2. The tool raises `EditNotFoundError`.
3. The agent catches the error.
4. The agent reads the file again to find the
   current text.

### 7.3 MultiEdit Failure

The MultiEdit failure flow runs as follows.

1. An edit fails in the list.
2. The tool rolls back the applied edits.
3. The tool raises `MultiEditError`.
4. The agent catches the error.
5. The agent fixes the failing edit.

## 8. Sequence Diagrams

### 8.1 Edit Sequence

The edit sequence has the steps below.

1. The agent calls the Edit tool.
2. The tool reads the file.
3. The tool replaces the match.
4. The tool writes the file.
5. The tool writes the audit row.
6. The tool returns the diff.
