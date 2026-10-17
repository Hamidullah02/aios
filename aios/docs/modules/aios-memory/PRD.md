# aios-memory — Product Requirements Document

## 1. Purpose

This document states the product requirements for the
aios-memory module. The module manages the
agent memory. The module covers four areas: the
project memory file, the conventions file, the
ignore file, and the context compaction.

## 2. Scope

The aios-memory module is in scope for this document.

- The project memory file (AGENTS.md).
- The conventions file (CONVENTIONS.md).
- The ignore file (.aiosignore).
- The context compaction.

The aios-memory module is not in scope for this
document.

- The conversation history. The history is the
  model's responsibility.
- The model prompt. The prompt is in aios-planner.

## 3. Reference Documents

- Project design document, section 7.
- aios-core PRD. See `docs/modules/aios-core/PRD.md`.
- Claude Code CLAUDE.md reference.
- Aider CONVENTIONS.md reference.
- Cursor rules reference.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Project memory | A file that holds project-level instructions. |
| Conventions | A file that holds coding conventions. |
| Ignore file | A file that holds paths to skip. |
| Context | The information the model sees on a call. |
| Compaction | The summarization of a long context. |
| AGENTS.md | The aios project memory file. |
| CONVENTIONS.md | The aios conventions file. |
| .aiosignore | The aios ignore file. |


## 5. Determinism

The whole point of aios is to make complex coding and large systems deterministic. The determinism stance of this module is **Highly deterministic**.

The KV store is content addressed. The clock is only used for TTL ordering.

The module provides the following determinism guarantees.

- Keys and values are content addressed.

- The clock is injected through a typed interface.
## 6. User Personas

The user persona is the operator who edits the
project memory. The operator writes the conventions
for the project. The operator lists the paths to
skip.

## 7. User Stories

An operator wants to add project instructions. The
operator creates the AGENTS.md file. The CLI loads
the file on every run.

An operator wants to add coding conventions. The
operator creates the CONVENTIONS.md file. The CLI
loads the file on every run.

An operator wants to skip a directory. The operator
adds the directory to the .aiosignore file. The
tools skip the directory.

An agent wants to compact the context. The agent
calls the compact context tool. The tool returns the
summarized context.

## 8. Functional Requirements Summary

The aios-memory module provides the following
functions.

- The project memory loader.
- The conventions loader.
- The ignore loader.
- The context compactor.

## 9. Non-Functional Requirements Summary

- The module runs on Go 1.23.
- The module depends on aios-core, aios-models, and
  the Go standard library.
- The module has 100 percent test coverage for the
  public functions.

## 10. Success Criteria

- The project memory loader loads the AGENTS.md
  file.
- The conventions loader loads the CONVENTIONS.md
  file.
- The ignore loader loads the .aiosignore file.
- The context compactor summarizes the context.
- The test suite has a 100 percent pass rate.

## 11. Out of Scope

- The user interface for the memory.
- The remote access to the memory.
- The semantic search through the memory.
