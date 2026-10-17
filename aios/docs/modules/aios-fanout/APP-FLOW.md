# aios-fanout — Application Flow

## 1. Purpose

This document describes the application flow for the
aios-fanout module.

## 2. Scope

The scope of this document is the aios-fanout module.
The module is a library.

## 3. Reference Documents

- aios-fanout PRD. See
  `docs/modules/aios-fanout/PRD.md`.
- aios-fanout SRS. See
  `docs/modules/aios-fanout/SRS.md`.
- aios-fanout TRD. See
  `docs/modules/aios-fanout/TRD.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Caller | The code that calls the fan-out tool. |

## 5. User Flow

The aios-fanout module has no user-facing flow. The
sub-agent output is shown through the watch layer.

## 6. System Flow

### 6.1 Fan-out Flow

The fan-out flow runs as follows.

1. The caller calls the fan-out tool.
2. The tool builds the semaphore.
3. The tool creates a task for every input.
4. The tool runs the tasks in parallel.
5. The tool collects the results.
6. The tool merges the results.
7. The tool returns the merged result.

### 6.2 Worktree Run Flow

The worktree run flow runs as follows.

1. The caller calls the worktree tool.
2. The tool creates the worktree.
3. The tool changes the working directory.
4. The tool runs the callable.
5. The tool restores the working directory.
6. The tool removes the worktree.

### 6.3 Merge Flow

The merge flow runs as follows.

1. The caller calls the merger.
2. The merger applies the strategy.
3. The merger returns the merged result.

## 7. Error Flows

### 7.1 Fan-out Error

The fan-out error flow runs as follows.

1. A sub-agent fails.
2. The tool raises `FanOutError`.
3. The caller catches the error.
4. The caller reports the error.

### 7.2 Worktree Error

The worktree error flow runs as follows.

1. The worktree creation fails.
2. The tool raises `WorktreeError`.
3. The caller catches the error.
4. The caller reports the error.

## 8. Sequence Diagrams

### 8.1 Fan-out Sequence

The fan-out sequence has the steps below.

1. The caller calls the fan-out tool.
2. The tool spawns a sub-agent for every task.
3. The sub-agents run in parallel.
4. The tool collects the results.
5. The tool merges the results.
6. The tool returns the merged result.
