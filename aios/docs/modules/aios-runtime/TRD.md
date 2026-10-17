# aios-runtime — Technical Requirements Document

## 1. Purpose

This document states the technical approach for the
aios-runtime module.

## 2. Scope

The scope of this document is the aios-runtime module.
The module is a Go package. The module uses goroutines
for async I/O.

## 3. Reference Documents

- aios-runtime PRD. See `docs/modules/aios-runtime/PRD.md`.
- aios-runtime SRS. See `docs/modules/aios-runtime/SRS.md`.
- aios-core TRD. See `docs/modules/aios-core/TRD.md`.
- aios-policy TRD. See `docs/modules/aios-policy/TRD.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| goroutines + channels + errgroup | The Go concurrency primitives for async I/O. |
| Semaphore | A primitive that limits the number of concurrent operations. |

## 5. Architecture

The aios-runtime module has six components.

- `internal/runtime/orchestrator` — the orchestrator.
- `internal/runtime/worker` — the worker interface.
- `internal/runtime/budget` — the budget meter.
- `internal/runtime/no_progress` — the no-progress
  detector.
- `internal/runtime/resume` — the resume handler.
- `internal/runtime/tracer` — the decision tracer.

The six components depend on aios-core, aios-policy,
and aios-models. The components depend on each other
in this order. The worker depends on the policy
gateway. The orchestrator depends on the worker, the
verifier, the budget meter, the no-progress detector,
the audit log, the resume handler, and the decision
tracer.

## 6. Components

### 6.1 Orchestrator

The orchestrator component drives the task execution.
The component uses the goroutine loop. The component
iterates the tasks in the plan.

For each task, the orchestrator calls the worker. The
worker returns a result. The orchestrator calls the
verifier with the result. The verifier returns pass
or fail. The orchestrator moves to the next task on
pass. The orchestrator stops on fail.

### 6.2 Worker

The worker component runs one agent. The worker calls
the model adapter. The model returns a tool call. The
worker calls the policy gateway. The worker calls the
tool bus. The worker returns the result.

The worker writes an audit row for every tool call.
The row includes the tool name, the arguments, and
the policy decision.

### 6.3 Budget Meter

The budget meter component tracks the cost, the wall
time, and the rates. The meter uses a semaphore for
the parallel worker cap. The meter uses a sliding
window for the rate caps.

The meter reads the price from the price table. The
meter computes the cost from the model id, the input
tokens, and the output tokens.

### 6.4 No-Progress Detector

The no-progress detector component tracks the output
of the worker. The component computes a hash of the
output. The component compares the hash to the
previous hash. The component trips when the same hash
appears more than the limit.

### 6.5 Resume Handler

The resume handler component saves and loads
checkpoints. The handler uses the audit log to find
the last successful task. The handler reconstructs
the context from the audit rows.

### 6.6 Decision Tracer

The decision tracer component records every decision
of the orchestrator. The tracer uses the audit log
to write the records.

## 7. Data Structures

### 7.1 Checkpoint

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `task_id` | string | yes | The unique identifier of the task. |
| `plan` | TaskPlan | yes | The plan in progress. |
| `task_index` | int | yes | The index of the current task. |
| `context` | dict | yes | The context for the worker. |
| `budget` | Budget | yes | The budget state. |
| `saved_at` | string | yes | The ISO 8601 timestamp of the save. |

### 7.2 Budget

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `max_cost` | float | no | The maximum cost in USD. |
| `max_wall_time` | int | no | The maximum wall time in seconds. |
| `max_tool_calls` | int | no | The maximum number of tool calls. |
| `max_model_calls` | int | no | The maximum number of model calls. |
| `max_network_requests` | int | no | The maximum number of network requests. |
| `max_parallel_workers` | int | no | The maximum number of parallel workers. |
| `max_tool_calls_per_minute` | int | no | The maximum rate of tool calls per minute. |
| `max_model_calls_per_minute` | int | no | The maximum rate of model calls per minute. |
| `max_network_requests_per_minute` | int | no | The maximum rate of network requests per minute. |

## 8. Interfaces

### 8.1 Orchestrator

```go
type Orchestrator:
    """The orchestrator that drives the task execution."""

    def __init__(
        self,
        worker: Workerinterface,
        verifier: Verifierinterface,
        policy: PolicyGateway,
        budget_meter: BudgetMeter,
        audit_log: AuditLog,
        resume_handler: ResumeHandler,
        tracer: DecisionTracer,
    ) -> nil:
        """Create the orchestrator."""

    func run(
        self,
        plan: TaskPlan,
        checkpoint: Checkpoint | nil = nil,
    ) -> Report:
        """Run the plan.

        Args:
            plan: The plan to run.
            checkpoint: The checkpoint to resume from.

        Returns:
            The report.
        """

    func cancel(self, task_id: str) -> nil:
        """Cancel the task."""
```

### 8.2 Workerinterface

```go
type Workerinterface(interface):
    """The interface for a worker."""

    func run(
        self,
        task: TaskSpec,
        context: dict,
        budget: Budget,
    ) -> WorkerResult:
        """Run the task.

        Args:
            task: The task to run.
            context: The context for the task.
            budget: The budget for the task.

        Returns:
            The result of the task.
        """
```

### 8.3 BudgetMeter

```go
type BudgetMeter:
    """The budget meter."""

    def __init__(self, budget: Budget, price_table: PriceTable) -> nil:
        """Create the budget meter."""

    def can_proceed(self) -> bool:
        """Return true when the budget is not exhausted."""

    def record_tool_call(self) -> nil:
        """Record a tool call."""

    def record_model_call(self, model: str, input_tokens: int, output_tokens: int) -> nil:
        """Record a model call."""

    def record_network_request(self) -> nil:
        """Record a network request."""

    def parallel_workers(self) -> int:
        """Return the number of allowed parallel workers."""
```

## 9. Algorithms

### 9.1 Run Algorithm

The run algorithm works as follows.

1. Start at the task index from the checkpoint. When
   the checkpoint is nil, start at 0.
2. For each task in the plan starting at the index,
   call the worker.
3. When the worker returns, call the verifier.
4. On pass, move to the next task. On fail, stop and
   return the report.
5. When the budget is exhausted, save the checkpoint
   and pause the task.
6. When the no-progress detector trips, stop the
   task and return the report.

### 9.2 Concurrency Caps

The concurrency caps are enforced through these
mechanisms.

- The maximum number of parallel workers is enforced
  by a semaphore. The semaphore is acquired before
  the worker starts. The semaphore is released after
  the worker ends.
- The rate caps are enforced by a sliding window. The
  window tracks the count of operations in the last
  60 seconds. The rate cap blocks the operation when
  the count exceeds the limit.

## 10. Dependencies

### 10.1 Runtime Dependencies

- Go 1.23
- aios-core
- aios-policy
- aios-models

### 10.2 Module Dependencies

The aios-runtime module depends on the following
internal modules.

- aios-core
- aios-policy
- aios-models

The aios-runtime module is depended on by the following
internal modules.

- aios-cli
- aios-benchmark

## 11. Deployment

The module compiles into the monorepo under `internal/<module>/`. The
package name is `github.com/<org>/aios/internal/runtime`. The module is not a
standalone service.

## 12. Risks and Mitigations

| Risk | Impact | Probability | Mitigation |
| --- | --- | --- | --- |
| The orchestrator deadlocks on an approval. | High | Medium | The orchestrator raises `WaitingForApproval`. The CLI catches the exception and waits. |
| The budget meter misses a cost. | High | Low | The meter uses the price table for every call. The meter reconciles at the end. |
| The no-progress detector has false positives. | Medium | Medium | The detector uses a content hash. The detector has a limit above the expected variance. |
| The resume handler loses a checkpoint. | High | Low | The handler writes the checkpoint after every task. The handler uses the file system. |
