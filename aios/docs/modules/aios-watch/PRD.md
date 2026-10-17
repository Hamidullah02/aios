# aios-watch — Product Requirements Document

## 1. Purpose

This document states the product requirements for the
aios-watch module. The module is the operator
visibility layer. The module covers three areas: the
streaming output, the file watcher, and the
sub-agent task tool.

## 2. Scope

The aios-watch module is in scope for this document.

- The streaming output channel.
- The file watcher tool.
- The sub-agent Task tool.

The aios-watch module is not in scope for this
document.

- The model streaming. The streaming is in
  aios-models.
- The web UI. The UI is a future enhancement.

## 3. Reference Documents

- Project design document, section 7.
- aios-core PRD. See `docs/modules/aios-core/PRD.md`.
- aios-runtime PRD. See `docs/modules/aios-runtime/PRD.md`.
- aios-models PRD. See `docs/modules/aios-models/PRD.md`.
- Claude Code Task tool reference.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Stream | A typed channel of model output. |
| Watcher | A component that monitors a file for changes. |
| Sub-agent | A child agent that runs in its own context. |
| Task tool | A tool that spawns a sub-agent. |


## 5. Determinism

The whole point of aios is to make complex coding and large systems deterministic. The determinism stance of this module is **Bounded non-determinism**.

The file system and the wall clock are the only sources of non-determinism. The event format is deterministic.

The module provides the following determinism guarantees.

- Events are content addressed.

- The watcher is a pure function of the file system and the clock.
## 6. User Personas

The user persona is the operator who watches the
agent work. The operator wants to see the model
output as it streams. The operator wants to see
which files the agent changes.

## 7. User Stories

An operator wants to see the model output. The CLI
shows the output as the model streams. The CLI
updates the screen on every chunk.

An operator wants to see file changes. The CLI
shows the changed files as the agent edits them.
The CLI uses the file watcher.

An agent wants to delegate a research task. The
agent calls the Task tool. The Task tool spawns a
sub-agent. The sub-agent returns the result.

## 8. Functional Requirements Summary

The aios-watch module provides the following
functions.

- The streaming output channel.
- The file watcher tool.
- The sub-agent Task tool.

## 9. Non-Functional Requirements Summary

- The module runs on Go 1.23.
- The module depends on aios-core, aios-models, and
  aios-runtime.
- The module has 100 percent test coverage for the
  public functions.

## 10. Success Criteria

- The streaming channel streams model chunks.
- The file watcher detects file changes.
- The Task tool spawns a sub-agent.
- The test suite has a 100 percent pass rate.

## 11. Out of Scope

- The web interface for the stream.
- The mobile interface for the stream.
- The remote access to the stream.
