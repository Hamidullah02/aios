# aios-planner — Software Requirements Specification

## 1. Purpose

This document states the software requirements for the
aios-planner module.

## 2. Scope

The scope of this document is the aios-planner module.
The module is a library.

## 3. Reference Documents

- aios-planner PRD. See
  `docs/modules/aios-planner/PRD.md`.
- aios-core SRS. See `docs/modules/aios-core/SRS.md`.
- aios-runtime SRS. See `docs/modules/aios-runtime/SRS.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Plan | A typed object that describes the tasks for a goal. |
| Verifier | A typed component that checks the plan. |

## 5. Functional Requirements

### 5.1 Planner

**REQ-PLN-PLN-001** The module shall provide the
`Planner` class. The type shall accept the model
router, the tool map, and the verifier.

**REQ-PLN-PLN-002** The planner shall provide the
`plan` method. The method shall accept the goal and
the context. The method shall return the plan.

**REQ-PLN-PLN-003** The planner shall use the model
router to generate the plan. The planner shall pass
the goal, the tool map, and the context to the
model.

**REQ-PLN-PLN-004** The planner shall run the
verifier on the generated plan. The planner shall
return the plan on pass.

**REQ-PLN-PLN-005** The planner shall support the
repair loop. The planner shall call the model with
the errors on fail. The planner shall run the
verifier on the repaired plan. The planner shall
return the plan on pass.

**REQ-PLN-PLN-006** The planner shall have a
maximum repair iteration count. The default is 3.

### 5.2 Verifier

**REQ-PLN-VRF-001** The module shall provide the
`CodeFirstVerifier` class. The type shall implement
the `Verifier` protocol.

**REQ-PLN-VRF-002** The verifier shall validate the
plan against the schema. The verifier shall return
the errors.

**REQ-PLN-VRF-003** The verifier shall check the
schema of every tool call. The verifier shall
return the errors.

**REQ-PLN-VRF-004** The verifier shall check the
diff scope. The verifier shall return the errors
when the diff scope is too large.

**REQ-PLN-VRF-005** The verifier shall support
custom rules. The verifier shall run the custom
rules on the plan.

## 6. Non-Functional Requirements

**REQ-PLN-NFR-001** The module shall run on Go 1.23
3.12.

**REQ-PLN-NFR-002** The module shall have 100 percent
coverage for the public functions.

## 7. Interface Requirements

**REQ-PLN-IFC-001** The module shall export the
following items from `aios.planner`.

- `Planner`
- `CodeFirstVerifier`
- `Verifier`
- `PlanError`

## 8. Data Requirements

**REQ-PLN-DAT-001** The plan shall be a `TaskPlan`.
The plan shall include the goal, the tasks, the
tools, the model, and the budget.

**REQ-PLN-DAT-002** The plan error shall include
the error code, the field, and the message.

## 9. Error Handling

**REQ-PLN-ERR-001** The module shall raise
`PlanGenerationError` when the planner cannot
generate a valid plan after the maximum repair
iterations.

**REQ-PLN-ERR-002** The module shall raise
`PlanValidationError` when the verifier returns
errors.

## 10. Acceptance Criteria

1. Every REQ in this document has a test case.
2. The test suite has a 100 percent pass rate.
3. The test suite has 100 percent coverage for the
   public functions.
4. The planner returns a valid plan.
5. The verifier returns pass for a valid plan.
6. The verifier returns fail for an invalid plan.
7. The planner repairs an invalid plan.
