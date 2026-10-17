# aios-benchmark — Application Flow

## 1. Purpose

This document describes the application flow for the
aios-benchmark module.

## 2. Scope

The scope of this document is the aios-benchmark
module.

## 3. Reference Documents

- aios-benchmark PRD. See
  `docs/modules/aios-benchmark/PRD.md`.
- aios-benchmark SRS. See
  `docs/modules/aios-benchmark/SRS.md`.
- aios-benchmark TRD. See
  `docs/modules/aios-benchmark/TRD.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| User | The developer that runs the benchmark. |

## 5. User Flow

The user flow runs as follows.

1. The user runs the CLI command.
2. The CLI loads the dataset.
3. The CLI runs the benchmark.
4. The CLI prints the metrics.

## 6. System Flow

### 6.1 Run Flow

The run flow runs as follows.

1. The CLI calls `runner.run`.
2. The runner iterates over the tasks.
3. For each task, the runner calls the orchestrator.
4. The orchestrator runs the task.
5. The orchestrator returns the result.
6. The runner records the metrics.
7. The runner returns the list of metrics.
8. The CLI prints the metrics.

### 6.2 Two-Tier Flow

The two-tier flow runs as follows.

1. The CLI calls `two_tier.run`.
2. The runner runs the strong orchestrator.
3. The runner runs the weak orchestrator.
4. The runner computes the delta.
5. The runner returns the delta.
6. The CLI prints the delta.

### 6.3 Loop Check Flow

The loop check flow runs as follows.

1. The CLI calls `loop.run`.
2. The checker runs the orchestrator on every task.
3. The checker records the loop count.
4. The checker computes the metrics.
5. The checker returns the metrics.
6. The CLI prints the metrics.

## 7. Error Flows

### 7.1 Benchmark Timeout

The benchmark timeout flow runs as follows.

1. The task exceeds the timeout.
2. The runner raises `BenchmarkTimeout`.
3. The runner records the failure.
4. The runner returns the metrics.

## 8. Sequence Diagrams

### 8.1 Run Sequence

The run sequence has the steps below.

1. The CLI calls `runner.run`.
2. The runner calls the orchestrator.
3. The orchestrator runs the task.
4. The orchestrator returns the result.
5. The runner records the metrics.
6. The runner returns the metrics.
