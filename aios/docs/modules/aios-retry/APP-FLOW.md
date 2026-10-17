# aios-retry — Application Flow

## 1. Purpose

This document describes the application flow for the
aios-retry module.

## 2. Scope

The scope of this document is the aios-retry module.
The module is a library.

## 3. Reference Documents

- aios-retry PRD. See `docs/modules/aios-retry/PRD.md`.
- aios-retry SRS. See `docs/modules/aios-retry/SRS.md`.
- aios-retry TRD. See `docs/modules/aios-retry/TRD.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Caller | The code that calls the retry policy. |

## 5. User Flow

The aios-retry module has no user-facing flow.

## 6. System Flow

### 6.1 Decide Flow

The decide flow runs as follows.

1. The caller calls `policy.decide` with the tool
   call, the error, and the retry count.
2. The policy calls the classifier.
3. The classifier returns the error class.
4. The policy checks the budget.
5. The policy returns the decision.

### 6.2 Retry Flow

The retry flow runs as follows.

1. The caller calls the tool.
2. The tool fails.
3. The caller calls `policy.decide`.
4. The policy returns `retry` with the backoff.
5. The caller waits the backoff.
6. The caller retries the tool.

### 6.3 Fix Flow

The fix flow runs as follows.

1. The caller calls the tool.
2. The tool fails.
3. The caller calls `policy.decide`.
4. The policy returns `retry_with_fix` with the
   fix prompt.
5. The caller calls the model with the fix prompt.
6. The model returns the fixed tool call.
7. The caller retries the tool.

## 7. Error Flows

### 7.1 Budget Exhausted

The budget exhausted flow runs as follows.

1. The retry count is at the budget.
2. The policy returns `stop`.
3. The caller stops the task.

## 8. Sequence Diagrams

### 8.1 Retry Sequence

The retry sequence has the steps below.

1. The caller calls the tool.
2. The tool fails.
3. The caller calls `policy.decide`.
4. The policy returns `retry`.
5. The caller waits the backoff.
6. The caller retries the tool.
