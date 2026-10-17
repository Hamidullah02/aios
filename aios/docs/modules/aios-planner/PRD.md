# aios-planner — Product Requirements Document

## 1. Purpose

This document states the product requirements for the
aios-planner module. The module is the planner. The
planner turns a goal into a TaskPlan. The planner
uses a code-first verifier to check the plan.

## 2. Scope

The aios-planner module is in scope for this document.

- The planner.
- The code-first verifier.
- The plan repair loop.

The aios-planner module is not in scope for this
document.

- The orchestrator. The orchestrator is in aios-runtime.
- The model adapter. The adapter is in aios-models.

## 3. Reference Documents

- Project design document, sections 5, 14, 15.
- aios-core PRD. See `docs/modules/aios-core/PRD.md`.
- aios-runtime PRD. See `docs/modules/aios-runtime/PRD.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Plan | A typed object that describes the tasks for a goal. |
| Planner | The component that turns a goal into a plan. |
| Verifier | The component that checks the plan is valid. |
| Repair | The loop that fixes a plan when the verifier fails. |


## 5. Determinism

The whole point of aios is to make complex coding and large systems deterministic. The determinism stance of this module is **Bounded by `temperature=0`**.

The planner uses the model. The plan is deterministic for a fixed model and `temperature=0`. The plan is content addressed.

The module provides the following determinism guarantees.

- Plans are content addressed.

- The plan is immutable after the first dispatch.
## 6. User Personas

The user persona is a Go developer who needs to
turn a goal into a plan.

## 7. User Stories

A developer wants to plan a goal. The developer
passes the goal to the planner. The planner returns
the plan.

A developer wants to verify a plan. The developer
passes the plan to the verifier. The verifier returns
pass or fail. The verifier returns the errors when
the plan fails.

A developer wants to repair a plan. The developer
passes the plan and the errors to the planner. The
planner returns the repaired plan.

## 8. Functional Requirements Summary

The aios-planner module provides the following
functions.

- The planner.
- The code-first verifier.
- The plan repair loop.

## 9. Non-Functional Requirements Summary

- The module runs on Go 1.23.
- The module depends on aios-core, aios-runtime, and
  aios-models.
- The module has 100 percent test coverage for the
  public functions.

## 10. Success Criteria

- The planner returns a valid plan.
- The verifier returns pass for a valid plan.
- The verifier returns fail for an invalid plan.
- The planner repairs an invalid plan.
- The test suite has a 100 percent pass rate.

## 11. Out of Scope

- The user interface for the planner.
- The remote access to the planner.
- The plan persistence.
