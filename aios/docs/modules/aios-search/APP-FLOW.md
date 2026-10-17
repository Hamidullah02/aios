# aios-search — Application Flow

## 1. Purpose

This document describes the application flow for the
aios-search module.

## 2. Scope

The scope of this document is the aios-search module.
The module is a library.

## 3. Reference Documents

- aios-search PRD. See `docs/modules/aios-search/PRD.md`.
- aios-search SRS. See `docs/modules/aios-search/SRS.md`.
- aios-search TRD. See `docs/modules/aios-search/TRD.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Agent | The code that calls the search tool. |

## 5. User Flow

The aios-search module has no user-facing flow.

## 6. System Flow

### 6.1 Glob Flow

The glob flow runs as follows.

1. The agent calls the Glob tool.
2. The tool loads the `.aiosignore` patterns.
3. The tool walks the workspace.
4. The tool matches the paths against the glob.
5. The tool returns the matching paths.

### 6.2 Grep Flow

The grep flow runs as follows.

1. The agent calls the Grep tool.
2. The tool builds the ripgrep command.
3. The tool runs ripgrep.
4. The tool parses the output.
5. The tool returns the matches.

### 6.3 CodeSearch Flow

The code search flow runs as follows.

1. The agent calls the CodeSearch tool.
2. The tool walks the workspace.
3. The tool parses each source file.
4. The tool walks the AST.
5. The tool finds the symbol.
6. The tool returns the locations.

### 6.4 Diagnostics Flow

The diagnostics flow runs as follows.

1. The agent calls the Diagnostics tool.
2. The tool runs the checker.
3. The tool parses the output.
4. The tool returns the diagnostics.

## 7. Error Flows

### 7.1 Checker Not Found

The checker not found flow runs as follows.

1. The checker binary is not on the PATH.
2. The tool raises `CheckerNotFoundError`.
3. The agent catches the error.
4. The agent reports the error.

### 7.2 Grep Timeout

The grep timeout flow runs as follows.

1. ripgrep exceeds the timeout.
2. The tool raises `GrepTimeoutError`.
3. The agent catches the error.
4. The agent narrows the search.

## 8. Sequence Diagrams

### 8.1 Grep Sequence

The grep sequence has the steps below.

1. The agent calls the Grep tool.
2. The tool runs ripgrep.
3. ripgrep returns the matches.
4. The tool returns the matches.
