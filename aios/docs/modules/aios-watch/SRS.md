# aios-watch — Software Requirements Specification

## 1. Purpose

This document states the software requirements for the
aios-watch module.

## 2. Scope

The scope of this document is the aios-watch module.
The module is a library.

## 3. Reference Documents

- aios-watch PRD. See `docs/modules/aios-watch/PRD.md`.
- aios-core SRS. See `docs/modules/aios-core/SRS.md`.
- aios-runtime SRS. See `docs/modules/aios-runtime/SRS.md`.
- aios-models SRS. See `docs/modules/aios-models/SRS.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Stream channel | A typed channel that streams model chunks. |
| Watcher | A typed component that monitors a file. |
| Sub-agent | A typed child agent definition. |

## 5. Functional Requirements

### 5.1 Stream Channel

**REQ-WTC-STR-001** The module shall provide the
`StreamChannel` class. The type shall be an
async-iterable wrapper around the model router.

**REQ-WTC-STR-002** The channel shall yield model
chunks. The channel shall yield the final usage
when the model is done.

**REQ-WTC-STR-003** The channel shall support a
subscriber. The channel shall call the subscriber
on every chunk.

**REQ-WTC-STR-004** The channel shall support
multiple subscribers.

**REQ-WTC-STR-005** The channel shall close when
the model is done. The channel shall close on
error.

### 5.2 File Watcher Tool

**REQ-WTC-FWT-001** The module shall provide the
`FileWatcherTool` class. The type shall implement
the `Tool` protocol.

**REQ-WTC-FWT-002** The tool shall accept a `path`
argument and a `pattern` argument. The tool
monitors the path for files that match the
pattern.

**REQ-WTC-FWT-003** The tool shall use the
`github.com/fsnotify/fsnotify` (cross-platform file-system events). The tool detects
create, modify, delete, and move events.

**REQ-WTC-FWT-004** The tool shall return the
list of events. Each event includes the path, the
event type, and the timestamp.

**REQ-WTC-FWT-005** The tool shall respect the
`.aiosignore` file from aios-core.

**REQ-WTC-FWT-006** The tool shall write an audit
row. The row type shall be `file_watch`.

### 5.3 Sub-agent Task Tool

**REQ-WTC-SBT-001** The module shall provide the
`TaskTool` class. The type shall implement the
`Tool` protocol.

**REQ-WTC-SBT-002** The tool shall accept a
`prompt` argument and an `agent_type` argument.
The supported agent types are `general`,
`explore`, `plan`, and `code`.

**REQ-WTC-SBT-003** The tool shall spawn a
sub-agent. The sub-agent runs in its own context
window. The sub-agent returns the result.

**REQ-WTC-SBT-004** The tool shall return the
sub-agent result. The result is a string.

**REQ-WTC-SBT-005** The tool shall support a
`description` argument. The argument is a short
description of the task. The tool uses the
description in the audit log.

**REQ-WTC-SBT-006** The tool shall write an audit
row. The row type shall be `subagent_task`. The
row shall include the prompt, the agent type, the
description, the duration, and the result.

**REQ-WTC-SBT-007** The tool shall support
parallel calls. The model can call the Task tool
multiple times in a single turn. The tool runs
the calls in parallel.

## 6. Non-Functional Requirements

**REQ-WTC-NFR-001** The module shall run on Go 1.23
3.12.

**REQ-WTC-NFR-002** The module shall have 100 percent
coverage for the public functions.

**REQ-WTC-NFR-003** The stream channel shall yield
chunks within 100 milliseconds of the model
emission.

## 7. Interface Requirements

**REQ-WTC-IFC-001** The module shall export the
following items from `aios.watch`.

- `StreamChannel`
- `FileWatcherTool`
- `TaskTool`
- `WatchEvent`
- `AgentType`

## 8. Data Requirements

**REQ-WTC-DAT-001** The `WatchEvent` schema shall
include the fields below.

- `path` (string)
- `event_type` (string: created, modified, deleted,
  moved)
- `timestamp` (string, ISO 8601)

**REQ-WTC-DAT-002** The `AgentType` enum shall
include the values `general`, `explore`, `plan`,
and `code`.

## 9. Error Handling

**REQ-WTC-ERR-001** The module shall raise
`WatchError` when the file watcher fails.

**REQ-WTC-ERR-002** The module shall raise
`SubAgentError` when the sub-agent fails.

## 10. Acceptance Criteria

1. Every REQ in this document has a test case.
2. The test suite has a 100 percent pass rate.
3. The test suite has 100 percent coverage for the
   public functions.
4. The stream channel streams model chunks.
5. The stream channel supports multiple
   subscribers.
6. The file watcher detects file changes.
7. The Task tool spawns a sub-agent.
8. The Task tool runs parallel calls.
