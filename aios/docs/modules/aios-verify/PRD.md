# aios-verify — Product Requirements Document

## 1. Purpose

This document states the product requirements for the
aios-verify module. The module is the
self-evaluation layer. The module lets the agent
check the result before the agent reports the task
as done. The module covers the self-check, the
test-runner tool, and the verification prompt.

## 2. Scope

The aios-verify module is in scope for this document.

- The self-check tool.
- The run_tests tool.
- The verification prompt.
- The verification result.

The aios-verify module is not in scope for this
document.

- The shell tool. The tool is in aios-tools.
- The diagnostics tool. The tool is in aios-search.
- The orchestrator. The orchestrator is in
  aios-runtime.

## 3. Reference Documents

- Project design document, section 14.
- aios-core PRD. See `docs/modules/aios-core/PRD.md`.
- aios-tools PRD. See `docs/modules/aios-tools/PRD.md`.
- aios-runtime PRD. See `docs/modules/aios-runtime/PRD.md`.
- Reflexion research paper.
- Self-Refine research paper.
- SWE-Agent run_tests reference.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Self-check | A check the agent runs on the result before reporting done. |
| Test runner | A component that runs the test suite. |
| Verification prompt | A prompt that asks the model to verify the result. |
| Verification result | A typed object that holds the verification outcome. |

## 5. User Personas

The user persona is the agent that needs to check
the result. The agent wants to know whether the
result is correct. The agent wants to know whether
the tests pass.

## 6. User Stories

An agent wants to run the test suite. The agent
calls the run_tests tool. The tool returns the
test result. The agent uses the result to decide
the next step.

An agent wants to verify the result. The agent
calls the self-check tool. The tool asks the model
to verify the result. The tool returns the
verification result.

An agent wants to fix the result. The agent calls
the run_tests tool. The tool returns the test
failures. The agent fixes the failures.

## 7. Functional Requirements Summary

The aios-verify module provides the following
functions.

- The self-check tool.
- The run_tests tool.
- The verification prompt.
- The verification result.

## 8. Non-Functional Requirements Summary

- The module runs on Go 1.23.
- The module depends on aios-core, aios-tools, and
  aios-models.
- The module has 100 percent test coverage for the
  public functions.

## 9. Success Criteria

- The self-check tool verifies the result.
- The run_tests tool runs the test suite.
- The verification prompt asks the model.
- The verification result is typed.
- The test suite has a 100 percent pass rate.

## 10. Out of Scope

- The user interface for the verification.
- The remote access to the verification.
- The cross-task verification.
