# aios-fanout — Product Requirements Document

## 1. Purpose

This document states the product requirements for the
aios-fanout module. The module is the parallel
execution layer. The module covers three areas: the
sub-agent delegation, the parallel task execution,
and the worktree isolation.

## 2. Scope

The aios-fanout module is in scope for this document.

- The sub-agent delegation.
- The parallel task execution.
- The worktree isolation.
- The result merge.

The aios-fanout module is not in scope for this
document.

- The Task tool. The Task tool is in aios-watch.
- The git worktree. The worktree is in aios-git.

## 3. Reference Documents

- Project design document, section 7.
- aios-core PRD. See `docs/modules/aios-core/PRD.md`.
- aios-runtime PRD. See `docs/modules/aios-runtime/PRD.md`.
- aios-git PRD. See `docs/modules/aios-git/PRD.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Sub-agent | A child agent that runs in its own context. |
| Fan-out | The parallel execution of multiple tasks. |
| Fan-in | The merge of multiple task results. |
| Worktree | A separate working copy linked to the same repo. |
| Context | The information the model sees on a call. |

## 5. User Personas

The user persona is the agent that needs to do
several things at once. The agent wants to delegate
the work to sub-agents. The agent wants to merge
the results.

## 6. User Stories

An agent wants to run three research tasks at once.
The agent calls the fan-out tool. The tool spawns
three sub-agents. The tool returns the merged
result.

An agent wants to run a coding task in a worktree.
The agent calls the worktree tool. The tool creates
the worktree. The tool runs the task in the
worktree.

An agent wants to merge the work from two
worktrees. The agent calls the merge tool. The
tool merges the changes from both worktrees.

## 7. Functional Requirements Summary

The aios-fanout module provides the following
functions.

- The sub-agent delegation.
- The parallel task execution.
- The worktree isolation.
- The result merge.

## 8. Non-Functional Requirements Summary

- The module runs on Go 1.23.
- The module depends on aios-core, aios-runtime,
  aios-git, and aios-watch.
- The module has 100 percent test coverage for the
  public functions.
- The module supports up to 8 parallel sub-agents.

## 9. Success Criteria

- The sub-agent delegation spawns a sub-agent.
- The parallel task execution runs multiple tasks.
- The worktree isolation runs a task in a worktree.
- The result merge merges the results.
- The test suite has a 100 percent pass rate.

## 10. Out of Scope

- The user interface for the fan-out.
- The remote access to the fan-out.
- The cross-host execution.
