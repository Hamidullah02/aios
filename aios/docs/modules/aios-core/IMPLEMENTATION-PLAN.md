# aios-core — Implementation Plan

## 1. Purpose

This document states the implementation plan for the
aios-core module. The plan covers the phases, the
deliverables, the test plan, the merge plan, and the
risk register.

## 2. Scope

The scope of this document is the aios-core module.
The module is the foundation module. The plan assumes
the module is the first to be built.

## 3. Reference Documents

- aios-core PRD. See `docs/modules/aios-core/PRD.md`.
- aios-core SRS. See `docs/modules/aios-core/SRS.md`.
- aios-core TRD. See `docs/modules/aios-core/TRD.md`.
- aios-core Application Flow. See
  `docs/modules/aios-core/APP-FLOW.md`.
- Project coding standard. See
  `docs/standards/CODING-STANDARD.md`.
- ASD-STE100 writing standard. See
  `docs/standards/STE-STYLE-GUIDE.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Phase | A unit of work in the implementation plan. |
| Deliverable | A file or a test that the phase produces. |
| Test case | A single test in the test suite. |
| Coverage | The percentage of lines the test suite executes. |

## 5. Phases

### Phase 1: Project Skeleton

The phase produces the project skeleton.

1. Create the folder `modules/aios-core/`.
2. Create the folder `modules/aios-core/internal/core/`.
3. Create the folder `modules/aios-core/tests/`.
4. Create the file `go.mod` with the package
   metadata.
5. Create the file `README.md` with the module purpose.

The deliverable is the project skeleton.

### Phase 2: Schemas

The phase produces the schemas component.

1. Create the file `internal/core/schemas.py`.
2. Create the file `tests/test_schemas.py`.
3. Implement the `AiosBaseModel` base class.
4. Implement the `TaskPlan` schema.
5. Implement the `AgentSpec` schema.
6. Implement the `WorkflowSpec` schema.
7. Implement the `ScheduleSpec` schema.
8. Implement the `DecisionTrace` schema.
9. Implement the `Envelope` schema.
10. Implement the `ApprovalPayload` schema.
11. Implement the `ModelCapability` schema.
12. Implement the `ToolCall` schema.
13. Implement the `ToolResult` schema.
14. Write the test cases for each schema.

The deliverable is the schemas component with tests.

### Phase 3: Redaction

The phase produces the redaction component.

1. Create the file `internal/core/redact.py`.
2. Create the file `tests/test_redact.py`.
3. Implement the `Redaction` class.
4. Implement the default pattern list.
5. Implement the placeholder generator.
6. Write the test cases for each pattern.
7. Write the test case for the approval id
   preservation.

The deliverable is the redaction component with tests.

### Phase 4: Audit Log

The phase produces the audit log component.

1. Create the file `internal/core/audit.py`.
2. Create the file `tests/test_audit.py`.
3. Implement the `AuditRow` schema.
4. Implement the `AuditLog` class.
5. Implement the hash chain.
6. Implement the idempotent write.
7. Implement the `get_by_task` method.
8. Implement the `get_by_approval` method.
9. Implement the `verify_chain` method.
10. Write the test cases for each method.

The deliverable is the audit log component with tests.

### Phase 5: State Machine

The phase produces the state machine component.

1. Create the file `internal/core/state.py`.
2. Create the file `tests/test_state.py`.
3. Implement the `TaskState` schema.
4. Implement the `TaskStateMachine` class.
5. Implement the transition table.
6. Implement the `transition` method.
7. Implement the `resume` method.
8. Implement the `cancel` method.
9. Write the test cases for each method.

The deliverable is the state machine component with tests.

### Phase 6: Integration Tests

The phase produces the integration tests.

1. Create the file `tests/test_integration.py`.
2. Write the test case for a full task lifecycle.
3. Write the test case for a paused task resume.
4. Write the test case for a failed task audit.
5. Write the test case for the hash chain
   verification on a multi-step task.

The deliverable is the integration test suite.

### Phase 7: Quality Gates

The phase produces the quality gates.

1. Run the test suite with `go test`.
2. Run the coverage report with `go test -cover`.
3. Run the type check with `go vet ./internal/core`.
4. Run the linter with `gofmt + go vet check`.
5. Fix every failure.

The deliverable is a green test suite with 100
percent coverage for the public functions.

## 6. Deliverables

The module produces the following deliverables.

- The source files in `internal/core/`.
- The test files in `tests/`.
- The `go.mod` file.
- The `README.md` file.
- The test report with 100 percent coverage.

## 7. Test Plan

### 7.1 Unit Tests

The unit tests cover every public function in the
module. The unit tests are in the files that match the
source files.

- `tests/test_schemas.py` covers `internal/core/schemas.py`.
- `tests/test_redact.py` covers `internal/core/redact.py`.
- `tests/test_audit.py` covers `internal/core/audit.py`.
- `tests/test_state.py` covers `internal/core/state.py`.

### 7.2 Integration Tests

The integration tests cover the interactions between
the components. The integration tests are in
`tests/test_integration.py`.

### 7.3 Coverage Target

The test suite targets 100 percent coverage for the
public functions. The test suite accepts a minimum of
95 percent coverage for the private functions.

### 7.4 Test Cases

The test cases include the following items.

- A schema rejects an invalid value.
- A schema accepts a valid value.
- The audit log writes a row in hash-chained order.
- The audit log rejects a duplicate write.
- The audit log returns the rows for a task.
- The audit log returns the rows for an approval id.
- The audit log verifies the hash chain.
- The state machine accepts a legal transition.
- The state machine rejects an illegal transition.
- The state machine writes an audit row on transition.
- The state machine resumes a paused task.
- The state machine cancels a running task.
- The redaction layer removes a secret.
- The redaction layer preserves the approval id.
- The redaction layer returns the correct marker.

## 8. Merge Plan

### 8.1 Pre-Merge Checklist

The module is ready for merge when the following
conditions are met.

1. The test suite has a 100 percent pass rate.
2. The coverage report shows 100 percent coverage for
   the public functions.
3. The `go vet` report shows no errors.
4. The `gofmt + go vet` report shows no errors.
5. The module is documented in the `README.md`.
6. The module has a `CHANGELOG.md` entry.

### 8.2 Merge Steps

The merge steps are below.

1. Create a branch named `merge/aios-core`.
2. Move the files from `modules/aios-core/` to the
   target locations in the main repository.
3. Run the test suite on the main repository.
4. Open a pull request.
5. Request a review from the architecture owner.
6. Merge after the review is approved.

### 8.3 Post-Merge Steps

The post-merge steps are below.

1. Delete the `modules/aios-core/` folder.
2. Update the `docs/README.md` to remove the entry.
3. Notify the team through the project channel.

## 9. Risk Register

| Risk | Impact | Probability | Mitigation |
| --- | --- | --- | --- |
| The struct + go-playground/validator (or hand-rolled tag validation) version breaks the schemas. | High | Low | The module pins the struct + go-playground/validator (or hand-rolled tag validation) version. The test suite runs on every change. |
| The SQLite file locks during a write. | Medium | Low | The module uses WAL mode. The module retries on lock. |
| The hash chain fails on a large payload. | Medium | Low | The module uses streaming for large payloads. The test suite covers the limit. |
| The redaction patterns miss a new secret type. | High | Medium | The module supports custom patterns. The project adds new patterns on a regular schedule. |
| The state machine has a race condition. | High | Low | The module uses a lock. The test suite covers the race. |

## 10. Schedule

| Phase | Duration | Dependency |
| --- | --- | --- |
| Phase 1: Project Skeleton | 1 day | nil |
| Phase 2: Schemas | 3 days | Phase 1 |
| Phase 3: Redaction | 1 day | Phase 2 |
| Phase 4: Audit Log | 3 days | Phase 2, Phase 3 |
| Phase 5: State Machine | 2 days | Phase 2, Phase 4 |
| Phase 6: Integration Tests | 2 days | Phase 4, Phase 5 |
| Phase 7: Quality Gates | 1 day | All phases |

The total duration is 13 days. The schedule assumes
one developer working full time.
