# aios-approvals — Implementation Plan

## 1. Purpose

This document states the implementation plan for the
aios-approvals module.

## 2. Scope

The scope of this document is the aios-approvals
module. The module depends on aios-core and
aios-policy.

## 3. Reference Documents

- aios-approvals PRD. See
  `docs/modules/aios-approvals/PRD.md`.
- aios-approvals SRS. See
  `docs/modules/aios-approvals/SRS.md`.
- aios-approvals TRD. See
  `docs/modules/aios-approvals/TRD.md`.
- aios-approvals Application Flow. See
  `docs/modules/aios-approvals/APP-FLOW.md`.
- aios-approvals Wireframes. See
  `docs/modules/aios-approvals/WIREFRAMES.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Phase | A unit of work in the implementation plan. |

## 5. Phases

### Phase 1: Project Skeleton

1. Create the project folders.
2. Create the `go.mod` file.
3. Create the `README.md` file.

The deliverable is the project skeleton.

### Phase 2: Schema

1. Create the file `internal/approvals/schema.py`.
2. Create the file `tests/test_schema.py`.
3. Implement the `Approval` class.
4. Implement the `ApprovalState` enum.
5. Write the test cases.

The deliverable is the schema with tests.

### Phase 3: Binding

1. Create the file `internal/approvals/binding.py`.
2. Create the file `tests/test_binding.py`.
3. Implement the `BindingValidator` class.
4. Write the test cases.

The deliverable is the binding with tests.

### Phase 4: Service

1. Create the file `internal/approvals/service.py`.
2. Create the file `tests/test_service.py`.
3. Implement the `ApprovalService` class.
4. Implement the `request` method.
5. Implement the `grant` method.
6. Implement the `deny` method.
7. Implement the `wait` method.
8. Implement the `get` method.
9. Implement the TTL timer.
10. Write the test cases.

The deliverable is the service with tests.

### Phase 5: Quality Gates

1. Run the test suite with `go test`.
2. Run the coverage report with `go test -cover`.
3. Run the type check with `go vet ./internal/approvals`.
4. Run the linter with `gofmt + go vet check`.
5. Fix every failure.

The deliverable is a green test suite.

## 6. Deliverables

- The source files in `internal/approvals/`.
- The test files in `tests/`.
- The `go.mod` file.
- The `README.md` file.

## 7. Test Plan

### 7.1 Unit Tests

- `tests/test_schema.py` covers
  `internal/approvals/schema.py`.
- `tests/test_binding.py` covers
  `internal/approvals/binding.py`.
- `tests/test_service.py` covers
  `internal/approvals/service.py`.

### 7.2 Test Cases

The test cases include the following items.

- The approval is created.
- The approval is granted.
- The approval is denied.
- The approval is expired.
- The approval is waited on.
- The binding is validated.
- The binding mismatch raises an error.
- The state transition is validated.
- The TTL timer expires the approval.
- The audit row is written for every state change.

## 8. Merge Plan

### 8.1 Pre-Merge Checklist

1. The test suite has a 100 percent pass rate.
2. The coverage report shows 100 percent coverage for
   the public functions.
3. The `go vet` report shows no errors.
4. The `gofmt + go vet` report shows no errors.

### 8.2 Merge Steps

1. Create a branch named `merge/aios-approvals`.
2. Move the files from `modules/aios-approvals/` to
   the target locations in the main repository.
3. Run the test suite on the main repository.
4. Open a pull request.
5. Merge after the review is approved.

## 9. Risk Register

| Risk | Impact | Probability | Mitigation |
| --- | --- | --- | --- |
| The approval is granted for the wrong task. | High | Low | The binding check blocks the grant. The binding check has a test case for every field. |
| The approval never expires. | Medium | Low | The TTL timer is started on creation. The TTL timer is unit tested. |
| The waiter is blocked forever. | High | Low | The TTL timer sets the event. The wait method has a timeout. |

## 10. Schedule

| Phase | Duration | Dependency |
| --- | --- | --- |
| Phase 1: Project Skeleton | 1 day | nil |
| Phase 2: Schema | 1 day | Phase 1 |
| Phase 3: Binding | 1 day | Phase 1 |
| Phase 4: Service | 3 days | Phase 2, Phase 3 |
| Phase 5: Quality Gates | 1 day | All phases |

The total duration is 7 days.
