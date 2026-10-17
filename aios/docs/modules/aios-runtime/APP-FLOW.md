# aios-runtime — Application Flow

## 1. Purpose

This document describes the application flow for the
aios-runtime module.

## 2. Scope

The scope of this document is the aios-runtime module.
The module is a library. The flows are programming
flows.

## 3. Reference Documents

- aios-runtime PRD. See `docs/modules/aios-runtime/PRD.md`.
- aios-runtime SRS. See `docs/modules/aios-runtime/SRS.md`.
- aios-runtime TRD. See `docs/modules/aios-runtime/TRD.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Caller | The code that calls the orchestrator. |
| Plan | The TaskPlan to execute. |
| Checkpoint | The saved state for resume. |

## 5. User Flow

The aios-runtime module has no user-facing flow.

## 6. System Flow

### 6.1 Run Flow

The run flow runs as follows.

1. The caller calls `orchestrator.run` with the plan
   and the checkpoint.
2. The orchestrator reads the checkpoint.
3. The orchestrator starts the task at the saved
   index.
4. The orchestrator calls the worker for the task.
5. The worker returns the result.
6. The orchestrator calls the verifier with the
   result.
7. The verifier returns pass or fail.
8. On pass, the orchestrator writes the checkpoint.
9. The orchestrator moves to the next task.
10. On fail, the orchestrator writes the report.
11. The orchestrator returns the report.

### 6.2 Worker Flow

The worker flow runs as follows.

1. The worker calls the model adapter.
2. The model returns a tool call.
3. The worker calls the policy gateway.
4. The gateway returns a decision.
5. On allow, the worker calls the tool bus.
6. On deny, the worker raises an error.
7. On ask, the worker raises `WaitingForApproval`.
8. The worker returns the result to the
   orchestrator.

### 6.3 Budget Check Flow

The budget check flow runs as follows.

1. The orchestrator calls `budget_meter.can_proceed`
   before each operation.
2. When the budget is not exhausted, the operation
   continues.
3. When the budget is exhausted, the orchestrator
   pauses the task.
4. The orchestrator saves the checkpoint.

### 6.4 No-Progress Check Flow

The no-progress check flow runs as follows.

1. The worker returns the result.
2. The orchestrator calls
   `no_progress.check(result)`.
3. When the detector trips, the orchestrator stops
   the task.
4. The orchestrator writes the failure to the
   report.

### 6.5 Cancel Flow

The cancel flow runs as follows.

1. The caller calls `orchestrator.cancel` with the
   task id.
2. The orchestrator sets a cancel flag.
3. The orchestrator waits for the current worker to
   end.
4. The orchestrator sets the task state to CANCELED.
5. The orchestrator writes the audit row.

### 6.6 Resume Flow

The resume flow runs as follows.

1. The caller calls `orchestrator.run` with the
   checkpoint.
2. The orchestrator reads the checkpoint.
3. The orchestrator restores the plan, the context,
   and the budget.
4. The orchestrator continues from the saved task
   index.

## 7. Error Flows

### 7.1 Waiting For Approval

The waiting for approval flow runs as follows.

1. The worker calls the policy gateway.
2. The gateway returns an `ask` decision.
3. The worker raises `WaitingForApproval`.
4. The orchestrator catches the exception.
5. The orchestrator pauses the task.
6. The orchestrator saves the checkpoint.
7. The orchestrator returns the report with the
   approval request.
8. The caller waits for the approval.
9. The caller resumes the task with the approval
   result.

### 7.2 Budget Exhausted

The budget exhausted flow runs as follows.

1. The budget meter returns false from
   `can_proceed`.
2. The orchestrator pauses the task.
3. The orchestrator saves the checkpoint.
4. The orchestrator returns the report.

### 7.3 No Progress Detected

The no-progress detected flow runs as follows.

1. The no-progress detector trips.
2. The orchestrator stops the task.
3. The orchestrator sets the task state to FAILED.
4. The orchestrator writes the failure to the report.

## 8. Sequence Diagrams

### 8.1 Run Sequence

The run sequence has the steps below.

1. The caller calls `orchestrator.run`.
2. The orchestrator reads the checkpoint.
3. The orchestrator calls the worker.
4. The worker calls the model adapter.
5. The model returns a tool call.
6. The worker calls the policy gateway.
7. The gateway returns the decision.
8. The worker calls the tool bus.
9. The tool bus returns the result.
10. The worker returns the result.
11. The orchestrator calls the verifier.
12. The verifier returns pass.
13. The orchestrator writes the checkpoint.
14. The orchestrator moves to the next task.
15. The orchestrator returns the report.
