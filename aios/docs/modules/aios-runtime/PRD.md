# aios-runtime — Product Requirements Document

## 1. Purpose

This document states the product requirements for the
aios-runtime module. The module is the orchestrator.
The orchestrator drives the execution of a TaskPlan
through the worker, the verifier, the budget meter,
and the decision tracer.

## 2. Scope

The aios-runtime module is in scope for this document.

- The orchestrator and the worker protocols.
- The budget meter and the no-progress detector.
- The resume handler.
- The decision tracer.

The aios-runtime module is not in scope for this
document.

- The model adapter. The adapter is in aios-models.
- The tool bus. The bus is in aios-tools.
- The approval service. The service is in aios-approvals.
- The CLI. The CLI is in aios-cli.

## 3. Reference Documents

- Project design document, sections 5, 8, 12, 15.
- aios-core PRD. See `docs/modules/aios-core/PRD.md`.
- aios-policy PRD. See `docs/modules/aios-policy/PRD.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Orchestrator | The component that drives the task execution. |
| Worker | The component that runs one agent. |
| Verifier | The component that checks the result of a worker. |
| Budget meter | The component that tracks the cost and the time. |
| No-progress detector | The component that detects when a worker loops. |
| Resume | The operation that restarts a task from a checkpoint. |
| Checkpoint | The saved state of a task for resume. |

## 5. User Personas

The user persona is a Go developer with experience
in async programming and state machines.

## 6. User Stories

A developer wants to run a TaskPlan. The developer
passes the plan to the orchestrator. The orchestrator
runs the plan. The orchestrator returns the report.

A developer wants to resume a paused task. The
developer passes the task id to the orchestrator. The
orchestrator reads the checkpoint. The orchestrator
resumes the task from the checkpoint.

A developer wants to stop a running task. The
developer passes the task id to the orchestrator. The
orchestrator cancels the task.

## 7. Functional Requirements Summary

The aios-runtime module provides the following
functions.

- The orchestrator that runs a TaskPlan.
- The worker protocol that runs one agent.
- The budget meter that tracks cost and time.
- The no-progress detector that detects loops.
- The resume handler that restarts a paused task.
- The decision tracer that records every decision.

## 8. Non-Functional Requirements Summary

The aios-runtime module has the following non-functional
requirements.

- The module runs on Go 1.23.
- The module uses goroutines for async I/O.
- The module supports four concurrency caps.
  - The maximum number of workers in parallel.
  - The maximum number of tool calls per minute.
  - The maximum number of model calls per minute.
  - The maximum number of network requests per minute.
- The module has 100 percent test coverage for the
  public functions.

## 9. Success Criteria

The aios-runtime module is successful when the
following conditions are met.

- The orchestrator runs a TaskPlan to completion.
- The budget meter stops the task on budget exhaustion.
- The no-progress detector stops the task on a loop.
- The resume handler restarts a paused task.
- The decision tracer records every decision.
- The four concurrency caps are enforced.
- The test suite has a 100 percent pass rate.

## 10. Out of Scope

The aios-runtime module does not include the following
functions.

- The user interface for the orchestrator.
- The remote access to the orchestrator.
- The scheduler for periodic tasks.

## 11. Open Questions

The following questions are open.

- Does the orchestrator support parallel tasks? The
  current design supports sequential tasks. The
  scheduler is in aios-cli.
- Does the budget meter support a per-model budget?
  The current design supports a per-task budget.
