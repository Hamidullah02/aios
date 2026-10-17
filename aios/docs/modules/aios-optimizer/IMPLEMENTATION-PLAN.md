# aios-optimizer — Implementation Plan

## 1. Purpose

This document states the implementation plan for the
aios-optimizer module.

## 2. Scope

The scope of this document is the aios-optimizer
module. The module depends on aios-core and
aios-models.

## 3. Reference Documents

- aios-optimizer PRD. See
  `docs/modules/aios-optimizer/PRD.md`.
- aios-optimizer SRS. See
  `docs/modules/aios-optimizer/SRS.md`.
- aios-optimizer TRD. See
  `docs/modules/aios-optimizer/TRD.md`.
- aios-optimizer Application Flow. See
  `docs/modules/aios-optimizer/APP-FLOW.md`.

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

### Phase 2: Tiered Model Registry

1. Create the file `internal/optimizer/registry.py`.
2. Create the file `tests/test_registry.py`.
3. Implement the `TieredModelRegistry` class.
4. Implement the YAML loader.
5. Write the test cases.

The deliverable is the registry with tests.

### Phase 3: Task Classifier

1. Create the file `internal/optimizer/classifier.py`.
2. Create the file `tests/test_classifier.py`.
3. Implement the `TaskClassifier` class.
4. Implement the strategies.
5. Write the test cases.

The deliverable is the classifier with tests.

### Phase 4: Cost Tracker

1. Create the file `internal/optimizer/cost_tracker.py`.
2. Create the file `tests/test_cost_tracker.py`.
3. Implement the `CostTracker` class.
4. Implement the budget enforcement.
5. Write the test cases.

The deliverable is the cost tracker with tests.

### Phase 5: Routing Strategy

1. Create the file `internal/optimizer/strategy.py`.
2. Create the file `tests/test_strategy.py`.
3. Implement the `RoutingStrategy` class.
4. Implement the modes.
5. Write the test cases.

The deliverable is the strategy with tests.

### Phase 6: Quality Feedback

1. Create the file `internal/optimizer/feedback.py`.
2. Create the file `tests/test_feedback.py`.
3. Implement the `QualityFeedback` class.
4. Write the test cases.

The deliverable is the feedback with tests.

### Phase 7: Quality Gates

1. Run the test suite with `go test`.
2. Run the coverage report with `go test -cover`.
3. Run the type check with `go vet ./internal/optimizer`.
4. Run the linter with `gofmt + go vet check`.
5. Fix every failure.

The deliverable is a green test suite.

## 6. Deliverables

- The source files in `internal/optimizer/`.
- The test files in `tests/`.
- The `go.mod` file.
- The `README.md` file.
- A sample optimizer file in `examples/`.

## 7. Test Plan

### 7.1 Unit Tests

- `tests/test_registry.py` covers
  `internal/optimizer/registry.py`.
- `tests/test_classifier.py` covers
  `internal/optimizer/classifier.py`.
- `tests/test_cost_tracker.py` covers
  `internal/optimizer/cost_tracker.py`.
- `tests/test_strategy.py` covers
  `internal/optimizer/strategy.py`.
- `tests/test_feedback.py` covers
  `internal/optimizer/feedback.py`.

### 7.2 Test Cases

The test cases include the following items.

- The registry loads the tiers.
- The classifier returns the right tier.
- The cost tracker records the cost.
- The strategy picks the model.
- The cascade mode escalates.
- The feedback updates the strategy.

## 8. Merge Plan

### 8.1 Pre-Merge Checklist

1. The test suite has a 100 percent pass rate.
2. The coverage report shows 100 percent coverage
   for the public functions.
3. The `go vet` report shows no errors.
4. The `gofmt + go vet` report shows no errors.

### 8.2 Merge Steps

1. Create a branch named `merge/aios-optimizer`.
2. Move the files from `modules/aios-optimizer/`
   to the target locations in the main
   repository.
3. Run the test suite on the main repository.
4. Open a pull request.
5. Merge after the review is approved.

## 9. Risk Register

| Risk | Impact | Probability | Mitigation |
| --- | --- | --- | --- |
| The classifier picks the wrong tier. | High | Medium | The cascade mode escalates on low confidence. |
| The cost tracker misses a call. | High | Low | The tracker wraps the model router. |
| The learned classifier is stale. | Medium | Medium | The classifier is retrained on a regular schedule. |

## 10. Schedule

| Phase | Duration | Dependency |
| --- | --- | --- |
| Phase 1: Project Skeleton | 1 day | nil |
| Phase 2: Tiered Model Registry | 1 day | Phase 1 |
| Phase 3: Task Classifier | 2 days | Phase 1 |
| Phase 4: Cost Tracker | 1 day | Phase 1 |
| Phase 5: Routing Strategy | 2 days | Phase 2, Phase 3, Phase 4 |
| Phase 6: Quality Feedback | 1 day | Phase 5 |
| Phase 7: Quality Gates | 1 day | All phases |

The total duration is 9 days.
