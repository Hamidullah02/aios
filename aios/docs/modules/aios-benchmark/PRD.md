# aios-benchmark — Product Requirements Document

## 1. Purpose

This document states the product requirements for the
aios-benchmark module. The module is the benchmark
harness. The harness runs the wedge dataset against
the orchestrator. The harness produces the metrics
for the report.

## 2. Scope

The aios-benchmark module is in scope for this
document.

- The wedge dataset.
- The benchmark runner.
- The two-class runner.
- The metrics calculator.

The aios-benchmark module is not in scope for this
document.

- The orchestrator. The orchestrator is in aios-runtime.
- The report generator. The generator is in aios-report.

## 3. Reference Documents

- Project design document, section 15.
- aios-runtime PRD. See `docs/modules/aios-runtime/PRD.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Wedge | A set of tasks that exercise a specific capability. |
| Strong class | A larger model with higher capability. |
| Cheap class | A smaller model with lower capability. |
| Loop check | A check that the agent did not loop. |
| Head-to-head | A comparison between two systems on the same task. |


## 5. Determinism

The whole point of aios is to make complex coding and large systems **reproducible and auditable**, not to make them 'deterministic' in the sense that a model-mediated system cannot be. The determinism stance of this module is **Highly deterministic** for the benchmark harness itself; the model is one input, never the foundation.

The benchmark harness is a pure function of the dataset and the orchestrator.

The module provides the following determinism guarantees.

- Datasets are content addressed.

- Benchmark results are content addressed.
## 6. User Personas

The user persona is a Go developer who needs to
measure the performance of the orchestrator.

## 7. User Stories

A developer wants to run the wedge dataset. The
developer passes the dataset to the runner. The
runner runs the orchestrator on the dataset. The
runner returns the metrics.

A developer wants to run the two-class comparison.
The developer passes the dataset to the runner. The
runner runs the orchestrator with the strong class
and the cheap class. The runner returns the comparison.

A developer wants to measure the loop check. The
developer passes the dataset to the runner. The
runner runs the orchestrator. The runner measures
the loop count. The runner returns the metrics.

## 8. Functional Requirements Summary

The aios-benchmark module provides the following
functions.

- The wedge dataset.
- The benchmark runner.
- The two-class runner.
- The metrics calculator.

## 9. Non-Functional Requirements Summary

- The module runs on Go 1.23.
- The module depends on aios-core, aios-runtime, and
  aios-models.
- The module has 100 percent behavioural coverage of the public functions. Line coverage is tracked as a guideline, not a gate.

## 10. Success Criteria

- The runner runs the wedge dataset.
- The runner produces the metrics.
- The two-class runner produces the comparison.
- The loop check produces the metrics.
- The behavioural test suite has a 100 percent pass rate.

## 11. Out of Scope

- The user interface for the runner.
- The remote access to the runner.
- The historical metrics.
