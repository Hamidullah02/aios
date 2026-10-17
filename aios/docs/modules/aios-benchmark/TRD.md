# aios-benchmark — Technical Requirements Document

## 1. Purpose

This document states the technical approach for the
aios-benchmark module.

## 2. Scope

The scope of this document is the aios-benchmark
module. The module is a Go package and a CLI.

## 3. Reference Documents

- aios-benchmark PRD. See
  `docs/modules/aios-benchmark/PRD.md`.
- aios-benchmark SRS. See
  `docs/modules/aios-benchmark/SRS.md`.
- aios-core TRD. See `docs/modules/aios-core/TRD.md`.
- aios-runtime TRD. See `docs/modules/aios-runtime/TRD.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Wedge | A set of tasks that exercise a specific capability. |
| Tier | A level of model capability. |

## 5. Architecture

The aios-benchmark module has four components.

- `internal/benchmark/dataset` — the wedge dataset.
- `internal/benchmark/runner` — the benchmark runner.
- `aios.benchmark.two_tier` — the two-tier runner.
- `aios.benchmark.loop` — the loop checker.

The four components depend on aios-core and
aios-runtime. The components depend on each other in
this order. The dataset has no dependencies. The
runner depends on the dataset. The two-tier runner
depends on the dataset and the runner. The loop
checker depends on the dataset and the runner.

## 6. Components

### 6.1 Dataset

The dataset component loads the wedge tasks from a
JSON file. The component validates the tasks. The
component returns the list of tasks.

### 6.2 Runner

The runner component runs the orchestrator on every
task. The component records the metrics. The
component returns the list of metrics.

### 6.3 Two-Tier Runner

The two-tier runner component runs the dataset with
two orchestrators. The component computes the
delta. The component returns the delta.

### 6.4 Loop Checker

The loop checker component runs the orchestrator on
every task. The component records the loop count.
The component computes the average, the maximum,
and the percentage. The component returns the
metrics.

## 7. Data Structures

### 7.1 Metrics

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `task_id` | string | yes | The unique identifier of the task. |
| `duration_ms` | int | yes | The duration in milliseconds. |
| `cost_usd` | float | yes | The cost in USD. |
| `success` | bool | yes | Whether the task succeeded. |
| `loop_count` | int | yes | The loop count. |
| `tool_calls` | int | yes | The number of tool calls. |
| `model_calls` | int | yes | The number of model calls. |
| `approvals` | int | yes | The number of approvals. |

### 7.2 WedgeTask

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `id` | string | yes | The unique identifier. |
| `wedge` | string | yes | The wedge id. |
| `goal` | string | yes | The goal of the task. |
| `expected_result` | dict | no | The expected result. |
| `timeout_seconds` | int | no | The timeout. |

## 8. Interfaces

### 8.1 BenchmarkRunner

```go
type BenchmarkRunner:
    """The benchmark runner."""

    def __init__(
        self,
        orchestrator: Orchestrator,
        dataset: WedgeDataset,
    ) -> nil:
        """Create the benchmark runner."""

    func run(self) -> list[Metrics]:
        """Run the benchmark.

        Returns:
            The list of metrics.
        """
```

### 8.2 TwoTierRunner

```go
type TwoTierRunner:
    """The two-tier runner."""

    def __init__(
        self,
        strong_orchestrator: Orchestrator,
        weak_orchestrator: Orchestrator,
        dataset: WedgeDataset,
    ) -> nil:
        """Create the two-tier runner."""

    func run(self) -> TwoTierMetrics:
        """Run the two-tier benchmark.

        Returns:
            The two-tier metrics.
        """
```

## 9. Algorithms

### 9.1 Run Algorithm

The run algorithm works as follows.

1. For each task in the dataset, run the
   orchestrator.
2. Record the duration, the cost, the success, and
   the loop count.
3. When the task fails, record the failure.
4. Return the list of metrics.

### 9.2 Two-Tier Algorithm

The two-tier algorithm works as follows.

1. Run the dataset with the strong orchestrator.
2. Run the dataset with the weak orchestrator.
3. Compute the delta in the metrics.
4. Return the delta.

### 9.3 Loop Check Algorithm

The loop check algorithm works as follows.

1. For each task in the dataset, run the
   orchestrator.
2. Record the loop count.
3. Compute the average, the maximum, and the
   percentage.
4. Return the metrics.

## 10. Dependencies

### 10.1 Runtime Dependencies

- Go 1.23
- aios-core
- aios-runtime
- aios-models

### 10.2 Module Dependencies

The aios-benchmark module depends on the following
internal modules.

- aios-core
- aios-runtime
- aios-models

The aios-benchmark module is depended on by the
following internal modules.

- aios-cli

## 11. Deployment

The module compiles into the monorepo under `internal/<module>/`. The
package name is `github.com/<org>/aios/internal/benchmark`. The module has a
CLI entry point.

## 12. Risks and Mitigations

| Risk | Impact | Probability | Mitigation |
| --- | --- | --- | --- |
| The task exceeds the timeout. | Medium | Medium | The runner has a timeout. The runner records the failure. |
| The model cost is wrong. | High | Low | The runner uses the price table. The runner reconciles at the end. |
| The loop count is wrong. | Medium | Low | The runner reads the loop count from the orchestrator. The runner has a test case. |
