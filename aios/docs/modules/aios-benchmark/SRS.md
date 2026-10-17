# aios-benchmark — Software Requirements Specification

## 1. Purpose

This document states the software requirements for the
aios-benchmark module.

## 2. Scope

The scope of this document is the aios-benchmark
module. The module is a library and a CLI.

## 3. Reference Documents

- aios-benchmark PRD. See
  `docs/modules/aios-benchmark/PRD.md`.
- aios-core SRS. See `docs/modules/aios-core/SRS.md`.
- aios-runtime SRS. See `docs/modules/aios-runtime/SRS.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Dataset | A typed list of tasks for the benchmark. |
| Metrics | A typed object that holds the results. |

## 5. Functional Requirements

### 5.1 Dataset

**REQ-BCH-DAT-001** The module shall provide the
`WedgeDataset` class. The type shall accept the
list of tasks.

**REQ-BCH-DAT-002** The dataset shall include the
following wedges.

- File read wedge
- File write wedge
- File search wedge
- Shell run wedge
- Approval wedge
- Ask-user wedge
- Resume wedge
- Cancel wedge

**REQ-BCH-DAT-003** The dataset shall load the
tasks from a JSON file. The default path is
`./benchmark/wedge.json`.

### 5.2 Runner

**REQ-BCH-RUN-001** The module shall provide the
`BenchmarkRunner` class. The type shall accept the
orchestrator and the dataset.

**REQ-BCH-RUN-002** The runner shall provide the
`run` method. The method shall return the metrics.

**REQ-BCH-RUN-003** The runner shall run the
orchestrator on every task in the dataset.

**REQ-BCH-RUN-004** The runner shall record the
duration, the cost, the success, and the loop count
for every task.

### 5.3 Two-Tier Runner

**REQ-BCH-2TR-001** The module shall provide the
`TwoTierRunner` class. The type shall accept two
orchestrators with different model tiers.

**REQ-BCH-2TR-002** The runner shall run the
dataset with the strong tier. The runner shall run
the dataset with the weak tier.

**REQ-BCH-2TR-003** The runner shall compute the
delta in the metrics. The runner shall return the
delta.

### 5.4 Loop Check

**REQ-BCH-LPC-001** The module shall provide the
`LoopChecker` class. The type shall accept the
orchestrator and the dataset.

**REQ-BCH-LPC-002** The checker shall run the
orchestrator on every task. The checker shall
record the loop count.

**REQ-BCH-LPC-003** The checker shall return the
metrics. The metrics shall include the average
loop count, the maximum loop count, and the
percentage of tasks that looped.

## 6. Non-Functional Requirements

**REQ-BCH-NFR-001** The module shall run on Go 1.23
3.12.

**REQ-BCH-NFR-002** The module shall have 100 percent
coverage for the public functions.

## 7. Interface Requirements

**REQ-BCH-IFC-001** The module shall export the
following items from `aios.benchmark`.

- `WedgeDataset`
- `BenchmarkRunner`
- `TwoTierRunner`
- `LoopChecker`
- `Metrics`

## 8. Data Requirements

**REQ-BCH-DAT-004** The `Metrics` schema shall
include the fields below.

- `task_id` (string)
- `duration_ms` (int)
- `cost_usd` (float)
- `success` (bool)
- `loop_count` (int)
- `tool_calls` (int)
- `model_calls` (int)
- `approvals` (int)

## 9. Error Handling

**REQ-BCH-ERR-001** The module shall raise
`BenchmarkTimeout` when a task exceeds the
timeout.

**REQ-BCH-ERR-002** The module shall raise
`BenchmarkError` when a task fails.

## 10. Acceptance Criteria

1. Every REQ in this document has a test case.
2. The test suite has a 100 percent pass rate.
3. The test suite has 100 percent coverage for the
   public functions.
4. The runner runs the wedge dataset.
5. The two-tier runner produces the comparison.
6. The loop checker produces the metrics.
