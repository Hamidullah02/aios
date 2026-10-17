# aios-planner — Implementation Plan

## 1. Purpose

This document states the implementation plan for the
aios-planner module.

## 2. Scope

The scope of this document is the aios-planner module.
The module depends on aios-core, aios-runtime, and
aios-models.

## 3. Reference Documents

- aios-planner PRD. See
  `docs/modules/aios-planner/PRD.md`.
- aios-planner SRS. See
  `docs/modules/aios-planner/SRS.md`.
- aios-planner TRD. See
  `docs/modules/aios-planner/TRD.md`.
- aios-planner Application Flow. See
  `docs/modules/aios-planner/APP-FLOW.md`.

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

### Phase 2: Prompts

1. Create the file `internal/planner/prompts.py`.
2. Create the file `tests/test_prompts.py`.
3. Implement the prompt templates.
4. Write the test cases.

The deliverable is the prompts with tests.

### Phase 3: Verifier

1. Create the file `internal/planner/verifier.py`.
2. Create the file `tests/test_verifier.py`.
3. Implement the `CodeFirstVerifier` class.
4. Implement the schema check.
5. Implement the tool call check.
6. Implement the diff scope check.
7. Write the test cases.

The deliverable is the verifier with tests.

### Phase 4: Planner

1. Create the file `internal/planner/planner.py`.
2. Create the file `tests/test_planner.py`.
3. Implement the `Planner` class.
4. Implement the plan method.
5. Implement the repair loop.
6. Write the test cases.

The deliverable is the planner with tests.

### Phase 5: Quality Gates

1. Run the test suite with `go test`.
2. Run the coverage report with `go test -cover`.
3. Run the type check with `go vet ./internal/planner`.
4. Run the linter with `gofmt + go vet check`.
5. Fix every failure.

The deliverable is a green test suite.

## 6. Deliverables

- The source files in `internal/planner/`.
- The test files in `tests/`.
- The `go.mod` file.
- The `README.md` file.
- The prompt templates in `internal/planner/templates/`.

## 7. Test Plan

### 7.1 Unit Tests

- `tests/test_prompts.py` covers
  `internal/planner/prompts.py`.
- `tests/test_verifier.py` covers
  `internal/planner/verifier.py`.
- `tests/test_planner.py` covers
  `internal/planner/planner.py`.

### 7.2 Test Cases

The test cases include the following items.

- The prompt template renders.
- The verifier validates a plan.
- The verifier rejects an invalid plan.
- The verifier rejects a too-large diff scope.
- The planner returns a valid plan.
- The planner repairs an invalid plan.
- The planner raises `PlanGenerationError` on
  exceed.

## 8. Merge Plan

### 8.1 Pre-Merge Checklist

1. The test suite has a 100 percent pass rate.
2. The coverage report shows 100 percent coverage for
   the public functions.
3. The `go vet` report shows no errors.
4. The `gofmt + go vet` report shows no errors.

### 8.2 Merge Steps

1. Create a branch named `merge/aios-planner`.
2. Move the files from `modules/aios-planner/` to
   the target locations in the main repository.
3. Run the test suite on the main repository.
4. Open a pull request.
5. Merge after the review is approved.

## 9. Risk Register

| Risk | Impact | Probability | Mitigation |
| --- | --- | --- | --- |
| The model returns invalid JSON. | High | Medium | The planner parses the response. The planner raises an error on parse failure. |
| The repair loop runs forever. | Medium | Low | The planner has a maximum iteration count. |
| The diff scope is too small. | Medium | Low | The limit is configurable. The limit is documented. |

## 10. Schedule

| Phase | Duration | Dependency |
| --- | --- | --- |
| Phase 1: Project Skeleton | 1 day | nil |
| Phase 2: Prompts | 1 day | Phase 1 |
| Phase 3: Verifier | 2 days | Phase 1 |
| Phase 4: Planner | 3 days | Phase 2, Phase 3 |
| Phase 5: Quality Gates | 1 day | All phases |

The total duration is 8 days.
