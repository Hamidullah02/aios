# aios-memory — Software Requirements Specification

## 1. Purpose

This document states the software requirements for the
aios-memory module.

## 2. Scope

The scope of this document is the aios-memory module.
The module is a library.

## 3. Reference Documents

- aios-memory PRD. See `docs/modules/aios-memory/PRD.md`.
- aios-core SRS. See `docs/modules/aios-core/SRS.md`.
- aios-models SRS. See `docs/modules/aios-models/SRS.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Project memory | A typed object that holds the project instructions. |
| Ignore pattern | A typed object that holds a glob pattern. |

## 5. Functional Requirements

### 5.1 Project Memory Loader

**REQ-MEM-PMM-001** The module shall provide the
`ProjectMemoryLoader` class. The type shall
accept the workspace root.

**REQ-MEM-PMM-002** The loader shall load the
AGENTS.md file from the workspace root. The loader
shall load the AGENTS.md files from the parent
directories up to the file system root.

**REQ-MEM-PMM-003** The loader shall merge the
files. The closer file overrides the parent file.

**REQ-MEM-PMM-004** The loader shall return the
merged project memory. The memory is a string.

**REQ-MEM-PMM-005** The loader shall cache the
result. The loader shall reload the result when
the file mtime changes.

### 5.2 Conventions Loader

**REQ-MEM-CNV-001** The module shall provide the
`ConventionsLoader` class. The type shall accept
the workspace root.

**REQ-MEM-CNV-002** The loader shall load the
CONVENTIONS.md file. The loader shall return the
conventions string.

**REQ-MEM-CNV-003** The loader shall support a
user-level conventions file at
`~/.aios/CONVENTIONS.md`.

**REQ-MEM-CNV-004** The loader shall merge the
user-level and the project-level files. The
project-level file overrides the user-level file.

### 5.3 Ignore Loader

**REQ-MEM-IGN-001** The module shall provide the
`IgnoreLoader` class. The type shall accept the
workspace root.

**REQ-MEM-IGN-002** The loader shall load the
`.aiosignore` file. The loader shall return the
list of patterns.

**REQ-MEM-IGN-003** The loader shall support the
`.gitignore` patterns. The loader shall reuse the
`pathspec` library.

**REQ-MEM-IGN-004** The loader shall support the
`!` negation. The loader shall support the `#`
comment.

**REQ-MEM-IGN-005** The loader shall provide the
`match` method. The method shall accept a path.
The method shall return true when the path matches
any pattern.

### 5.4 Context Compactor

**REQ-MEM-CMP-001** The module shall provide the
`ContextCompactor` class. The type shall accept
the model router.

**REQ-MEM-CMP-002** The compactor shall accept a
context. The context is a list of messages.

**REQ-MEM-CMP-003** The compactor shall return
the compacted context. The compactor shall use the
model router to summarize the older messages.

**REQ-MEM-CMP-004** The compactor shall keep the
last N messages verbatim. The default is 5.

**REQ-MEM-CMP-005** The compactor shall respect
the token budget. The compactor shall return the
compacted context when the token count is within
the budget.

**REQ-MEM-CMP-006** The compactor shall write an
audit row. The row type shall be `context_compact`.

## 6. Non-Functional Requirements

**REQ-MEM-NFR-001** The module shall run on Go 1.23
3.12.

**REQ-MEM-NFR-002** The module shall have 100 percent
coverage for the public functions.

**REQ-MEM-NFR-003** The loader shall load the file
in under 10 milliseconds.

## 7. Interface Requirements

**REQ-MEM-IFC-001** The module shall export the
following items from `aios.memory`.

- `ProjectMemoryLoader`
- `ConventionsLoader`
- `IgnoreLoader`
- `ContextCompactor`
- `ProjectMemory`
- `IgnorePattern`

## 8. Data Requirements

**REQ-MEM-DAT-001** The AGENTS.md file shall be a
Markdown file. The file has the sections below.

- `## Project` — one paragraph about the project.
- `## Commands` — the build, test, and lint
  commands.
- `## Conventions` — the code style conventions.
- `## Architecture` — the high-level architecture.
- `## Notes` — the notes for the agent.

**REQ-MEM-DAT-002** The CONVENTIONS.md file shall
be a Markdown file. The file has the sections
below.

- `## Style` — the code style rules.
- `## Libraries` — the approved libraries.
- `## Patterns` — the approved patterns.

**REQ-MEM-DAT-003** The .aiosignore file shall
follow the .gitignore format. The file lists the
glob patterns to skip.

## 9. Error Handling

**REQ-MEM-ERR-001** The module shall raise
`MemoryParseError` when the file is invalid.

**REQ-MEM-ERR-002** The module shall raise
`CompactionError` when the model call fails.

## 10. Acceptance Criteria

1. Every REQ in this document has a test case.
2. The test suite has a 100 percent pass rate.
3. The test suite has 100 percent coverage for the
   public functions.
4. The project memory loader loads the AGENTS.md
   file.
5. The conventions loader loads the CONVENTIONS.md
   file.
6. The ignore loader loads the .aiosignore file.
7. The context compactor summarizes the context.
8. The loaders cache the result.
9. The loaders reload the result on mtime change.
