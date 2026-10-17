# aios-retry — Implementation Plan

## 1. Purpose

This document states the implementation plan for the
aios-retry module.

## 2. Scope

The scope of this document is the aios-retry module.
The module depends on aios-core and aios-tools.

## 3. Reference Documents

- aios-retry PRD. See `docs/modules/aios-retry/PRD.md`.
- aios-retry SRS. See `docs/modules/aios-retry/SRS.md`.
- aios-retry TRD. See `docs/modules/aios-retry/TRD.md`.
- aios-retry Application Flow. See
  `docs/modules/aios-retry/APP-FLOW.md`.

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

### Phase 2: Error Classifier

1. Create the file `internal/retry/classifier.py`.
2. Create the file `tests/test_classifier.py`.
3. Implement the `ErrorClassifier` class.
4. Implement the default patterns.
5. Write the test cases.

The deliverable is the classifier with tests.

### Phase 3: Retry Strategy

1. Create the file `internal/retry/strategy.py`.
2. Create the file `tests/test_strategy.py`.
3. Implement the `RetryStrategy` class.
4. Write the test cases.

The deliverable is the strategy with tests.

### Phase 4: Retry Budget

1. Create the file `internal/retry/budget.py`.
2. Create the file `tests/test_budget.py`.
3. Implement the `RetryBudget` class.
4. Write the test cases.

The deliverable is the budget with tests.

### Phase 5: Retry Policy

1. Create the file `internal/retry/policy.py`.
2. Create the file `tests/test_policy.py`.
3. Implement the `RetryPolicy` class.
4. Implement the fix prompts.
5. Write the test cases.

The deliverable is the policy with tests.

### Phase 6: Quality Gates

1. Run the test suite with `go test`.
2. Run the coverage report with `go test -cover`.
3. Run the type check with `go vet ./internal/retry`.
4. Run the linter with `gofmt + go vet check`.
5. Fix every failure.

The deliverable is a green test suite.

## 6. Deliverables

- The source files in `internal/retry/`.
- The test files in `tests/`.
- The `go.mod` file.
- The `README.md` file.

## 7. Test Plan

### 7.1 Unit Tests

- `tests/test_classifier.py` covers
  `internal/retry/classifier.py`.
- `tests/test_strategy.py` covers
  `internal/retry/strategy.py`.
- `tests/test_budget.py` covers
  `internal/retry/budget.py`.
- `tests/test_policy.py` covers
  `internal/retry/policy.py`.

### 7.2 Test Cases

The test cases include the following items.

- The classifier classifies the error.
- The strategy computes the backoff.
- The budget caps the retries.
- The policy decides on retry or stop.
- The fix prompt is included on fixable errors.

## 8. Merge Plan

### 8.1 Pre-Merge Checklist

1. The test suite has a 100 percent pass rate.
2. The coverage report shows 100 percent coverage
   for the public functions.
3. The `go vet` report shows no errors.
4. The `gofmt + go vet` report shows no errors.

### 8.2 Merge Steps

1. Create a branch named `merge/aios-retry`.
2. Move the files from `modules/aios-retry/` to
   the target locations in the main repository.
3. Run the test suite on the main repository.
4. Open a pull request.
5. Merge after the review is approved.

## 9. Risk Register

| Risk | Impact | Probability | Mitigation |
| --- | --- | --- | --- |
| The classifier misses an error class. | High | Medium | The classifier supports custom patterns. |
| The retry loop runs forever. | High | Low | The budget caps the retries. |
| The fix prompt is wrong. | Medium | Low | The fix prompts are reviewed. |

## 10. Schedule

| Phase | Duration | Dependency |
| --- | --- | --- |
| Phase 1: Project Skeleton | 1 day | nil |
| Phase 2: Error Classifier | 1 day | Phase 1 |
| Phase 3: Retry Strategy | 1 day | Phase 1 |
| Phase 4: Retry Budget | 1 day | Phase 1 |
| Phase 5: Retry Policy | 1 day | Phase 2, Phase 3, Phase 4 |
| Phase 6: Quality Gates | 1 day | All phases |

The total duration is 6 days.
