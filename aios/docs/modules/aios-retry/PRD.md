# aios-retry — Product Requirements Document

## 1. Purpose

This document states the product requirements for the
aios-retry module. The module is the retry policy
layer. The module decides when to retry a failed
tool call. The module feeds the error back to the
model. The module caps the retry count.

## 2. Scope

The aios-retry module is in scope for this document.

- The retry policy.
- The error classifier.
- The retry strategy.
- The retry budget.

The aios-retry module is not in scope for this
document.

- The tool bus. The bus is in aios-tools.
- The model adapter. The adapter is in aios-models.
- The orchestrator. The orchestrator is in
  aios-runtime.

## 3. Reference Documents

- Project design document, section 7.
- aios-core PRD. See `docs/modules/aios-core/PRD.md`.
- aios-tools PRD. See `docs/modules/aios-tools/PRD.md`.
- Claude Code Edit error handling reference.
- Aider lint-and-fix loop reference.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Retry | A second attempt to run a tool call. |
| Error type | A category of error: transient, permanent, fixable. |
| Strategy | A rule that decides when to retry. |
| Backoff | A delay between retries. |
| Retry budget | The maximum number of retries per task. |


## 5. Determinism

The whole point of aios is to make complex coding and large systems deterministic. The determinism stance of this module is **Highly deterministic**.

The error classifier, the retry strategy, and the budget are deterministic. The wall clock is only used for backoff.

The module provides the following determinism guarantees.

- Classifier decisions are pure functions of the error.

- Backoff is a pure function of the retry count.

- Budget is a pure function of the policy.
## 6. User Personas

The user persona is the agent that calls a tool.
The agent wants the tool to succeed. The agent
wants the retry to use the error message.

## 7. User Stories

An agent wants to retry a transient error. The
agent calls the retry policy. The policy returns
retry. The agent retries the call.

An agent wants to fix a fixable error. The agent
calls the retry policy. The policy returns retry
with the fix prompt. The agent retries the call
with the fix.

An agent wants to stop on a permanent error. The
agent calls the retry policy. The policy returns
stop. The agent stops the task.

## 8. Functional Requirements Summary

The aios-retry module provides the following
functions.

- The retry policy.
- The error classifier.
- The retry strategy.
- The retry budget.

## 9. Non-Functional Requirements Summary

- The module runs on Go 1.23.
- The module depends on aios-core and aios-tools.
- The module has 100 percent test coverage for the
  public functions.

## 10. Success Criteria

- The retry policy decides on retry or stop.
- The error classifier classifies the error.
- The retry strategy applies the backoff.
- The retry budget caps the retries.
- The test suite has a 100 percent pass rate.

## 11. Out of Scope

- The user interface for the retry.
- The remote access to the retry.
- The cross-task retry.
