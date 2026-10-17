# aios-retry — Software Requirements Specification

## 1. Purpose

This document states the software requirements for the
aios-retry module.

## 2. Scope

The scope of this document is the aios-retry module.
The module is a library.

## 3. Reference Documents

- aios-retry PRD. See `docs/modules/aios-retry/PRD.md`.
- aios-core SRS. See `docs/modules/aios-core/SRS.md`.
- aios-tools SRS. See `docs/modules/aios-tools/SRS.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Retry decision | A typed object that holds the retry action. |
| Error type | A typed string: transient, permanent, fixable. |

## 5. Functional Requirements

### 5.1 Error Classifier

**REQ-RTR-CLS-001** The module shall provide the
`ErrorClassifier` class. The type shall accept the
error.

**REQ-RTR-CLS-002** The classifier shall return the
error class. The supported classes are `transient`,
`permanent`, and `fixable`.

**REQ-RTR-CLS-003** The classifier shall classify
the following errors as `transient`: timeout,
network error, rate limit, server error.

**REQ-RTR-CLS-004** The classifier shall classify
the following errors as `permanent`: permission
denied, not found, invalid input, syntax error.

**REQ-RTR-CLS-005** The classifier shall classify
the following errors as `fixable`: edit not
unique, edit not found, file conflict, merge
conflict.

### 5.2 Retry Policy

**REQ-RTR-POL-001** The module shall provide the
`RetryPolicy` class. The type shall accept the
classifier, the strategy, and the budget.

**REQ-RTR-POL-002** The policy shall accept the
tool call, the error, and the retry count. The
policy shall return the retry decision.

**REQ-RTR-POL-003** The decision shall include the
action: `retry`, `retry_with_fix`, or `stop`.

**REQ-RTR-POL-004** The decision shall include the
backoff in milliseconds when the action is
`retry`.

**REQ-RTR-POL-005** The decision shall include the
fix prompt when the action is `retry_with_fix`.

**REQ-RTR-POL-006** The policy shall return `stop`
when the retry count is at the budget.

**REQ-RTR-POL-007** The policy shall return `stop`
when the error is `permanent`.

### 5.3 Retry Strategy

**REQ-RTR-STR-001** The module shall provide the
`RetryStrategy` class. The type shall accept the
strategy name.

**REQ-RTR-STR-002** The strategy shall support the
following modes.

- `none` — no retry.
- `immediate` — retry with no delay.
- `linear` — retry with a linear backoff.
- `exponential` — retry with an exponential
  backoff.

**REQ-RTR-STR-003** The exponential backoff shall
double the delay every retry. The maximum delay
shall be 30 seconds.

### 5.4 Retry Budget

**REQ-RTR-BGT-001** The module shall provide the
`RetryBudget` class. The type shall accept the
max retries.

**REQ-RTR-BGT-002** The budget shall track the
retry count for every task. The budget shall
return the remaining count.

**REQ-RTR-BGT-003** The budget shall support a
per-tool limit. The budget shall support a
per-error-type limit.

## 6. Non-Functional Requirements

**REQ-RTR-NFR-001** The module shall run on Go 1.23
3.12.

**REQ-RTR-NFR-002** The module shall have 100 percent
coverage for the public functions.

**REQ-RTR-NFR-003** The policy decision shall have
a maximum latency of 1 millisecond.

## 7. Interface Requirements

**REQ-RTR-IFC-001** The module shall export the
following items from `aios.retry`.

- `RetryPolicy`
- `ErrorClassifier`
- `RetryStrategy`
- `RetryBudget`
- `RetryDecision`
- `ErrorClass`

## 8. Data Requirements

**REQ-RTR-DAT-001** The `RetryDecision` schema
shall include the fields below.

- `action` (string: retry, retry_with_fix, stop)
- `backoff_ms` (int)
- `fix_prompt` (string, optional)
- `reason` (string)

**REQ-RTR-DAT-002** The default retry budget shall
be 3 retries per task.

## 9. Error Handling

**REQ-RTR-ERR-001** The module shall raise
`RetryBudgetExceeded` when the budget is
exhausted.

## 10. Acceptance Criteria

1. Every REQ in this document has a test case.
2. The test suite has a 100 percent pass rate.
3. The test suite has 100 percent coverage for the
   public functions.
4. The classifier classifies the error.
5. The policy decides on retry or stop.
6. The strategy applies the backoff.
7. The budget caps the retries.
