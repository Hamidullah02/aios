# aios-ask — Product Requirements Document

## 1. Purpose

This document states the product requirements for the
aios-ask module. The module is the ask-user tool. The
tool lets the agent ask the operator a multiple-choice
question. The operator answers the question. The
agent receives the answer.

## 2. Scope

The aios-ask module is in scope for this document.

- The ask-user tool.
- The question schema.
- The answer schema.

The aios-ask module is not in scope for this document.

- The CLI. The CLI is in aios-cli.
- The approval service. The service is in
  aios-approvals.

## 3. Reference Documents

- Project design document, section 7.
- aios-core PRD. See `docs/modules/aios-core/PRD.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Question | A typed object that describes the ask request. |
| Answer | A typed object that describes the operator answer. |
| Option | A choice in a multiple-choice question. |


## 5. Determinism

The whole point of aios is to make complex coding and large systems deterministic. The determinism stance of this module is **Bounded non-determinism**.

The operator is the only source of non-determinism. The question and the options are deterministic.

The module provides the following determinism guarantees.

- Questions and options are content addressed.

- The timeout is a pure function of the policy and the clock.
## 6. User Personas

The user persona is an operator who needs to answer
questions from the agent.

## 7. User Stories

An agent wants to ask a question. The agent calls
the ask-user tool. The tool shows the question to the
operator. The operator answers the question. The
tool returns the answer.

## 8. Functional Requirements Summary

The aios-ask module provides the following functions.

- The ask-user tool.
- The question schema.
- The answer schema.

## 9. Non-Functional Requirements Summary

- The module runs on Go 1.23.
- The module depends on aios-core.
- The module has 100 percent test coverage for the
  public functions.

## 10. Success Criteria

- The tool shows the question.
- The tool returns the answer.
- The test suite has a 100 percent pass rate.

## 11. Out of Scope

- The user interface for the question.
- The remote access to the question.
- The free-form question.
