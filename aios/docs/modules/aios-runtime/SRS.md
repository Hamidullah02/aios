# aios-runtime — Software Requirements Specification

## 1. Purpose

This document states the software requirements for the
aios-runtime module.

## 2. Scope

The scope of this document is the aios-runtime module.
The module is a library. The module exports the
orchestrator and the worker interface.

## 3. Reference Documents

- aios-runtime PRD. See `docs/modules/aios-runtime/PRD.md`.
- aios-core SRS. See `docs/modules/aios-core/SRS.md`.
- aios-policy SRS. See `docs/modules/aios-policy/SRS.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Worker | The interface for a component that runs one agent. |
| Budget | The limit on cost and time for a task. |
| Checkpoint | The persisted state for resume. |

## 5. Functional Requirements

### 5.1 Orchestrator

**REQ-RT-ORC-001** The module shall provide the
`Orchestrator` class. The type shall accept the
worker, the verifier, the policy gateway, the budget
meter, the audit log, and the tracer.

**REQ-RT-ORC-002** The orchestrator shall run a
TaskPlan. The method shall accept the plan and the
checkpoint. The method shall return the report.

**REQ-RT-ORC-003** The orchestrator shall iterate
the tasks of the plan. The orchestrator shall run one
task at a time.

**REQ-RT-ORC-004** The orchestrator shall call the
worker for the current task. The orchestrator shall
pass the task, the context, and the budget.

**REQ-RT-ORC-005** The orchestrator shall call the
verifier for the result of the worker. The
orchestrator shall pass the result and the task.

**REQ-RT-ORC-006** When the verifier passes, the
orchestrator shall move to the next task. When the
verifier fails, the orchestrator shall stop the plan.

**REQ-RT-ORC-007** When the budget is exhausted, the
orchestrator shall pause the task. The orchestrator
shall write a checkpoint.

**REQ-RT-ORC-008** When the no-progress detector
trips, the orchestrator shall stop the task. The
orchestrator shall mark the task as failed.

**REQ-RT-ORC-009** The orchestrator shall support
the cancel operation. The orchestrator shall set the
state to CANCELED. The orchestrator shall stop the
current worker.

**REQ-RT-ORC-010** The orchestrator shall support
the resume operation. The orchestrator shall read the
checkpoint. The orchestrator shall continue from the
saved task.

### 5.2 Worker interface

**REQ-RT-WRK-001** The module shall define the
`Workerinterface`. The interface shall declare the
`run` method. The method shall accept the task, the
context, and the budget. The method shall return the
result.

**REQ-RT-WRK-002** The worker shall use the policy
gateway before every tool call. The worker shall
deny the call when the gateway denies it. The worker
shall wait for an approval when the gateway asks.

**REQ-RT-WRK-003** The worker shall write an audit
row for every tool call. The row type shall be
`tool_call`.

### 5.3 Budget Meter

**REQ-RT-BGT-001** The module shall provide the
`BudgetMeter` class. The type shall accept the
budget.

**REQ-RT-BGT-002** The budget meter shall track the
cost, the wall time, the number of tool calls, the
number of model calls, and the number of network
requests.

**REQ-RT-BGT-003** The budget meter shall provide
the `can_proceed` method. The method shall return
true when the budget is not exhausted. The method
shall return false when the budget is exhausted.

**REQ-RT-BGT-004** The budget meter shall support
the four concurrency caps. The caps are the maximum
rate of tool calls, model calls, and network requests
per minute. The fourth cap is the maximum number of
parallel workers.

**REQ-RT-BGT-005** The budget meter shall support
the per-model cost tracking. The meter shall use the
price table from aios-models.

### 5.4 No-Progress Detector

**REQ-RT-NPR-001** The module shall provide the
`NoProgressDetector` class. The type shall accept
the loop limit.

**REQ-RT-NPR-002** The detector shall track the
output of the worker. The detector shall compare the
output to the previous output.

**REQ-RT-NPR-003** The detector shall trip when the
same output repeats more than the loop limit. The
detector shall return the trip reason.

### 5.5 Resume Handler

**REQ-RT-RSM-001** The module shall provide the
`ResumeHandler` class. The type shall accept the
checkpoint path.

**REQ-RT-RSM-002** The handler shall save the
checkpoint after every task. The checkpoint shall
include the plan, the current task index, the
context, and the budget.

**REQ-RT-RSM-003** The handler shall load the
checkpoint. The handler shall return the plan, the
task index, the context, and the budget.

### 5.6 Decision Tracer

**REQ-RT-DEC-001** The module shall provide the
`DecisionTracer` class. The type shall accept the
audit log.

**REQ-RT-DEC-002** The tracer shall record every
decision of the orchestrator. The record shall
include the decision, the input, the output, the
reason, and the timestamp.

## 6. Non-Functional Requirements

**REQ-RT-NFR-001** The module shall run on Go 1.23
3.12.

**REQ-RT-NFR-002** The module shall use async I/O
through `goroutines + channels (stdlib) + golang.org/x/sync/errgroup`.

**REQ-RT-NFR-003** The module shall support the four
concurrency caps from section 5.3.

**REQ-RT-NFR-004** The module shall have 100 percent
coverage for the public functions.

## 7. Interface Requirements

**REQ-RT-IFC-001** The module shall export the
following items from `aios.runtime`.

- `Orchestrator`
- `Workerinterface`
- `BudgetMeter`
- `NoProgressDetector`
- `ResumeHandler`
- `DecisionTracer`
- `WaitingForApproval`

**REQ-RT-IFC-002** The `WaitingForApproval`
exception shall carry the approval payload. The
exception shall carry the approval id. The
exception shall carry the decision.

## 8. Data Requirements

**REQ-RT-DAT-001** The checkpoint shall be a JSON
file. The default path is `./var/checkpoints/`. The
path is configurable.

**REQ-RT-DAT-002** The checkpoint file name shall
be the task id with the extension `.json`.

## 9. Error Handling

**REQ-RT-ERR-001** The module shall raise
`BudgetExhausted` when the budget is exhausted. The
error shall include the budget type and the limit.

**REQ-RT-ERR-002** The module shall raise
`NoProgressDetected` when the detector trips. The
error shall include the loop count.

**REQ-RT-ERR-003** The module shall raise
`WaitingForApproval` when the worker needs an
approval. The error shall include the approval
payload and the approval id.

## 10. Acceptance Criteria

The aios-runtime module is accepted when the following
conditions are met.

1. Every REQ in this document has a test case.
2. The test suite has a 100 percent pass rate.
3. The test suite has 100 percent coverage for the
   public functions.
4. The orchestrator runs a TaskPlan to completion.
5. The budget meter stops the task on budget
   exhaustion.
6. The no-progress detector stops the task on a loop.
7. The resume handler restarts a paused task.
8. The four concurrency caps are enforced.
9. The decision tracer records every decision.
