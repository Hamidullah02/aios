# aios-planner — Application Flow

## 1. Purpose

This document describes the application flow for the
aios-planner module.

## 2. Scope

The scope of this document is the aios-planner module.
The module is a library.

## 3. Reference Documents

- aios-planner PRD. See
  `docs/modules/aios-planner/PRD.md`.
- aios-planner SRS. See
  `docs/modules/aios-planner/SRS.md`.
- aios-planner TRD. See
  `docs/modules/aios-planner/TRD.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Caller | The code that calls the planner. |

## 5. User Flow

The aios-planner module has no user-facing flow.

## 6. System Flow

### 6.1 Plan Flow

The plan flow runs as follows.

1. The caller calls `planner.plan` with the goal
   and the context.
2. The planner builds the prompt.
3. The planner calls the model router.
4. The model returns the plan.
5. The planner parses the plan.
6. The planner runs the verifier.
7. On pass, the planner returns the plan.
8. On fail, the planner calls the model with the
   errors.
9. The planner repeats the verifier check.
10. On pass, the planner returns the plan.
11. On exceed, the planner raises
    `PlanGenerationError`.

### 6.2 Verify Flow

The verify flow runs as follows.

1. The caller calls `verifier.verify` with the
   plan.
2. The verifier validates the plan against the
   schema.
3. The verifier validates the tool calls.
4. The verifier runs the diff scope check.
5. The verifier runs the custom rules.
6. The verifier returns the errors.

## 7. Error Flows

### 7.1 Plan Generation Error

The plan generation error flow runs as follows.

1. The planner exceeds the maximum repair
   iterations.
2. The planner raises `PlanGenerationError`.
3. The caller catches the error.
4. The caller reports the error.

### 7.2 Plan Validation Error

The plan validation error flow runs as follows.

1. The verifier returns errors.
2. The caller catches the errors.
3. The caller reports the errors.

## 8. Sequence Diagrams

### 8.1 Plan Sequence

The plan sequence has the steps below.

1. The CLI calls `planner.plan`.
2. The planner calls the model router.
3. The model returns the plan.
4. The planner runs the verifier.
5. The verifier returns pass.
6. The planner returns the plan.
