# aios-fanout — Technical Requirements Document

## 1. Purpose

This document states the technical approach for the
aios-fanout module.

## 2. Scope

The scope of this document is the aios-fanout module.
The module is a Go package.

## 3. Reference Documents

- aios-fanout PRD. See
  `docs/modules/aios-fanout/PRD.md`.
- aios-fanout SRS. See
  `docs/modules/aios-fanout/SRS.md`.
- aios-core TRD. See `docs/modules/aios-core/TRD.md`.
- aios-runtime TRD. See `docs/modules/aios-runtime/TRD.md`.
- aios-git TRD. See `docs/modules/aios-git/TRD.md`.
- aios-watch TRD. See `docs/modules/aios-watch/TRD.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Semaphore | An goroutines + channels (stdlib) + golang.org/x/sync/errgroup primitive that limits the number of concurrent operations. |
| errgroup.Group (golang.org/x/sync/errgroup) | The goroutines + channels (stdlib) + golang.org/x/sync/errgroup function that runs coroutines in parallel. |

## 5. Architecture

The aios-fanout module has four components.

- `aios.fanout.fanout` — the FanOut tool.
- `aios.fanout.worktree` — the worktree isolation.
- `internal/fanout/merge` — the result merger.
- `aios.fanout.scheduler` — the scheduler.

The four components depend on aios-core, aios-runtime,
aios-git, and aios-watch. The components depend on
each other in this order. The fanout tool has no
dependencies. The worktree isolation depends on
aios-git. The merger depends on aios-models. The
scheduler depends on aios-runtime.

## 6. Components

### 6.1 FanOut Tool

The fanout tool accepts a list of tasks. The tool
spawns a sub-agent for every task. The tool uses
`errgroup.Group (golang.org/x/sync/errgroup)` to run the sub-agents in parallel.
The tool respects the `max_parallel` setting through
a semaphore.

The tool supports worktree isolation. The tool
creates a worktree for every task when the `isolate`
flag is true. The tool runs the sub-agent in the
worktree.

### 6.2 Worktree Isolation

The worktree isolation wraps the git worktree tool.
The type provides the `create`, `remove`, and
`run_in` methods. The `run_in` method changes the
working directory before the callable and restores
it after.

### 6.3 Result Merger

The result merger accepts a list of results and a
strategy. The merger supports the `concat`,
`summary`, and `vote` strategies.

The `concat` strategy concatenates the results in
order. The `summary` strategy summarizes the
results through the model router. The `vote`
strategy picks the result that matches the
majority.

### 6.4 Scheduler

The scheduler runs the tasks in batches. The batch
size is the `max_parallel` setting. The scheduler
respects the budget. The scheduler stops when the
budget is exhausted.

## 7. Data Structures

### 7.1 FanOutTask

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `id` | string | yes | The unique identifier. |
| `prompt` | string | yes | The prompt for the sub-agent. |
| `agent_type` | string | yes | The agent type. |
| `isolate` | bool | no | Whether to isolate in a worktree. |
| `branch` | string | no | The branch for the worktree. |

### 7.2 FanOutResult

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `id` | string | yes | The unique identifier. |
| `output` | string | yes | The output of the sub-agent. |
| `status` | string | yes | The status of the sub-agent. |
| `duration_ms` | int | yes | The duration in milliseconds. |
| `cost_usd` | float | yes | The cost in USD. |

## 8. Interfaces

### 8.1 FanOutTool

```go
type FanOutTool:
    """The fan-out tool."""

    def __init__(
        self,
        sub_agent_factory: SubAgentFactory,
        worktree: WorktreeIsolation,
        merger: ResultMerger,
        max_parallel: int = 4,
    ) -> nil:
        """Create the fan-out tool."""

    func run(self, args: dict, context: ToolContext) -> ToolResult:
        """Run the fan-out tool.

        Args:
            args: The arguments of the call.
            context: The context for the call.

        Returns:
            The result of the call.
        """
```

## 9. Algorithms

### 9.1 Fan-out Algorithm

The fan-out algorithm works as follows.

1. Build a semaphore with the `max_parallel`
   setting.
2. For each task, acquire the semaphore.
3. Create the sub-agent.
4. Run the sub-agent.
5. Release the semaphore.
6. Collect the results.
7. Merge the results.
8. Return the merged result.

### 9.2 Worktree Run Algorithm

The worktree run algorithm works as follows.

1. Create the worktree.
2. Change the working directory to the worktree.
3. Run the callable.
4. Restore the working directory.
5. Remove the worktree.

### 9.3 Vote Algorithm

The vote algorithm works as follows.

1. Compare each result to the other results.
2. Pick the result that has the highest similarity
   score to the other results.
3. Return the result.

## 10. Dependencies

### 10.1 Runtime Dependencies

- Go 1.23
- aios-core
- aios-runtime
- aios-git
- aios-watch
- aios-models

### 10.2 Module Dependencies

The aios-fanout module depends on the following
internal modules.

- aios-core
- aios-runtime
- aios-git
- aios-watch
- aios-models

The aios-fanout module is depended on by the
following internal modules.

- aios-runtime
- aios-cli

## 11. Deployment

The module compiles into the monorepo under `internal/<module>/`. The
package name is `github.com/<org>/aios/internal/fanout`.

## 12. Risks and Mitigations

| Risk | Impact | Probability | Mitigation |
| --- | --- | --- | --- |
| The sub-agent runs forever. | Medium | Low | The sub-agent has a timeout. The scheduler has a timeout. |
| The worktree path collides. | Medium | Low | The worktree path includes a unique id. The tool checks the path. |
| The merge strategy loses information. | Medium | Low | The merger supports the `concat` strategy. The merger has a test case. |
