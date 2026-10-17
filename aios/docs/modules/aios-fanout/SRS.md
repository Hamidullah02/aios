# aios-fanout — Software Requirements Specification

## 1. Purpose

This document states the software requirements for the
aios-fanout module.

## 2. Scope

The scope of this document is the aios-fanout module.
The module is a library.

## 3. Reference Documents

- aios-fanout PRD. See
  `docs/modules/aios-fanout/PRD.md`.
- aios-core SRS. See `docs/modules/aios-core/SRS.md`.
- aios-runtime SRS. See `docs/modules/aios-runtime/SRS.md`.
- aios-git SRS. See `docs/modules/aios-git/SRS.md`.
- aios-watch SRS. See `docs/modules/aios-watch/SRS.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Fan-out task | A typed object that describes a parallel task. |
| Fan-in result | A typed object that describes the merged result. |

## 5. Functional Requirements

### 5.1 Fan-out Tool

**REQ-FAN-FOT-001** The module shall provide the
`FanOutTool` class. The type shall implement the
`Tool` protocol.

**REQ-FAN-FOT-002** The tool shall accept a
`tasks` argument. The `tasks` argument is a list of
task objects. Each task has a `prompt`, an
`agent_type`, and an `id`.

**REQ-FAN-FOT-003** The tool shall spawn a
sub-agent for every task. The tool shall run the
sub-agents in parallel.

**REQ-FAN-FOT-004** The tool shall respect the
`max_parallel` setting. The default is 4. The tool
queues the tasks when the limit is reached.

**REQ-FAN-FOT-005** The tool shall merge the
results. The merged result is a list of result
objects. Each result has the `id`, the `output`,
the `status`, and the `duration`.

**REQ-FAN-FOT-006** The tool shall support
worktree isolation. The tool shall accept an
`isolate` flag. The tool creates a worktree for
every task when the flag is true.

**REQ-FAN-FOT-007** The tool shall write an audit
row for every task. The row type shall be
`fanout_task`. The row shall include the task id,
the agent type, the duration, and the result.

### 5.2 Worktree Isolation

**REQ-FAN-WTI-001** The module shall provide the
`WorktreeIsolation` class. The type shall accept
the git tool.

**REQ-FAN-WTI-002** The type shall provide the
`create` method. The method shall accept a `branch`
argument and a `path` argument. The method shall
create the worktree.

**REQ-FAN-WTI-003** The type shall provide the
`remove` method. The method shall accept a `path`
argument. The method shall remove the worktree.

**REQ-FAN-WTI-004** The type shall provide the
`run_in` method. The method shall accept a `path`
argument and a callable. The method shall run the
callable in the worktree.

### 5.3 Result Merge

**REQ-FAN-RMG-001** The module shall provide the
`ResultMerger` class. The type shall accept the
merge strategy.

**REQ-FAN-RMG-002** The type shall support the
`concat` strategy. The type shall concatenate the
results in order.

**REQ-FAN-RMG-003** The type shall support the
`summary` strategy. The type shall summarize the
results through the model router.

**REQ-FAN-RMG-004** The type shall support the
`vote` strategy. The type shall pick the result
that matches the majority of the other results.

### 5.4 Scheduler

**REQ-FAN-SCH-001** The module shall provide the
`Scheduler` class. The type shall accept the
fan-out tool and the budget.

**REQ-FAN-SCH-002** The type shall run the tasks
in batches. The batch size is the `max_parallel`
setting.

**REQ-FAN-SCH-003** The type shall respect the
budget. The type shall stop when the budget is
exhausted.

## 6. Non-Functional Requirements

**REQ-FAN-NFR-001** The module shall run on Go 1.23
3.12.

**REQ-FAN-NFR-002** The module shall have 100 percent
coverage for the public functions.

**REQ-FAN-NFR-003** The module shall support up
to 8 parallel sub-agents.

## 7. Interface Requirements

**REQ-FAN-IFC-001** The module shall export the
following items from `aios.fanout`.

- `FanOutTool`
- `WorktreeIsolation`
- `ResultMerger`
- `Scheduler`
- `FanOutTask`
- `FanOutResult`
- `MergeStrategy`

## 8. Data Requirements

**REQ-FAN-DAT-001** The `FanOutTask` schema shall
include the fields below.

- `id` (string, unique)
- `prompt` (string)
- `agent_type` (string)
- `isolate` (bool, default false)
- `branch` (string, optional)

**REQ-FAN-DAT-002** The `FanOutResult` schema
shall include the fields below.

- `id` (string)
- `output` (string)
- `status` (string: succeeded, failed, canceled)
- `duration_ms` (int)
- `cost_usd` (float)

## 9. Error Handling

**REQ-FAN-ERR-001** The module shall raise
`FanOutError` when a sub-agent fails.

**REQ-FAN-ERR-002** The module shall raise
`WorktreeError` when the worktree creation
fails.

## 10. Acceptance Criteria

1. Every REQ in this document has a test case.
2. The test suite has a 100 percent pass rate.
3. The test suite has 100 percent coverage for the
   public functions.
4. The fan-out tool spawns a sub-agent for every
   task.
5. The fan-out tool runs the sub-agents in
   parallel.
6. The worktree isolation creates a worktree.
7. The result merge merges the results.
8. The scheduler respects the budget.
