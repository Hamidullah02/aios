# aios-runtime — Implementation Plan

## 1. Purpose

This document states the implementation plan for the
aios-runtime module.

## 2. Scope

The scope of this document is the aios-runtime module.
The module depends on aios-core, aios-policy, and
aios-models.

## 3. Reference Documents

- aios-runtime PRD. See `docs/modules/aios-runtime/PRD.md`.
- aios-runtime SRS. See `docs/modules/aios-runtime/SRS.md`.
- aios-runtime TRD. See `docs/modules/aios-runtime/TRD.md`.
- aios-runtime Application Flow. See
  `docs/modules/aios-runtime/APP-FLOW.md`.

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

### Phase 2: Budget Meter

1. Create the file `internal/runtime/budget.py`.
2. Create the file `tests/test_budget.py`.
3. Implement the `Budget` schema.
4. Implement the `BudgetMeter` class.
5. Implement the parallel workers semaphore.
6. Implement the rate caps sliding window.
7. Implement the cost computation.
8. Write the test cases.

The deliverable is the budget meter with tests.

### Phase 3: No-Progress Detector

1. Create the file `internal/runtime/no_progress.py`.
2. Create the file `tests/test_no_progress.py`.
3. Implement the `NoProgressDetector` class.
4. Write the test cases.

The deliverable is the detector with tests.

### Phase 4: Resume Handler

1. Create the file `internal/runtime/resume.py`.
2. Create the file `tests/test_resume.py`.
3. Implement the `Checkpoint` schema.
4. Implement the `ResumeHandler` class.
5. Write the test cases.

The deliverable is the handler with tests.

### Phase 5: Decision Tracer

1. Create the file `internal/runtime/tracer.py`.
2. Create the file `tests/test_tracer.py`.
3. Implement the `DecisionTracer` class.
4. Write the test cases.

The deliverable is the tracer with tests.

### Phase 6: Worker interface

1. Create the file `internal/runtime/worker.py`.
2. Create the file `tests/test_worker.py`.
3. Implement the `Workerinterface` interface.
4. Implement a stub worker for tests.
5. Implement the policy gateway integration.
6. Implement the `WaitingForApproval` exception.
7. Write the test cases.

The deliverable is the worker with tests.

### Phase 7: Orchestrator

1. Create the file `internal/runtime/orchestrator.py`.
2. Create the file `tests/test_orchestrator.py`.
3. Implement the `Orchestrator` class.
4. Implement the run algorithm.
5. Implement the cancel algorithm.
6. Implement the resume algorithm.
7. Write the test cases.

The deliverable is the orchestrator with tests.

### Phase 8: Integration Tests

1. Create the file `tests/test_integration.py`.
2. Write the test case for a full plan run.
3. Write the test case for an approval wait.
4. Write the test case for a budget exhaustion.
5. Write the test case for a no-progress detection.
6. Write the test case for a resume.
7. Write the test case for the four concurrency caps.

The deliverable is the integration test suite.

### Phase 9: Quality Gates

1. Run the test suite with `go test`.
2. Run the coverage report with `go test -cover`.
3. Run the type check with `go vet ./internal/runtime`.
4. Run the linter with `gofmt + go vet check`.
5. Fix every failure.

The deliverable is a green test suite.

## 6. Deliverables

- The source files in `internal/runtime/`.
- The test files in `tests/`.
- The `go.mod` file.
- The `README.md` file.

## 7. Test Plan

### 7.1 Unit Tests

- `tests/test_budget.py` covers
  `internal/runtime/budget.py`.
- `tests/test_no_progress.py` covers
  `internal/runtime/no_progress.py`.
- `tests/test_resume.py` covers
  `internal/runtime/resume.py`.
- `tests/test_tracer.py` covers
  `internal/runtime/tracer.py`.
- `tests/test_worker.py` covers
  `internal/runtime/worker.py`.
- `tests/test_orchestrator.py` covers
  `internal/runtime/orchestrator.py`.

### 7.2 Test Cases

The test cases include the following items.

- The budget meter tracks cost and time.
- The budget meter stops on budget exhaustion.
- The four concurrency caps are enforced.
- The no-progress detector trips on a loop.
- The resume handler saves the checkpoint.
- The resume handler loads the checkpoint.
- The decision tracer records every decision.
- The worker calls the policy gateway.
- The worker raises `WaitingForApproval`.
- The orchestrator runs a plan.
- The orchestrator cancels a task.
- The orchestrator resumes a task.

## 8. Merge Plan

### 8.1 Pre-Merge Checklist

1. The test suite has a 100 percent pass rate.
2. The coverage report shows 100 percent coverage for
   the public functions.
3. The `go vet` report shows no errors.
4. The `gofmt + go vet` report shows no errors.

### 8.2 Merge Steps

1. Create a branch named `merge/aios-runtime`.
2. Move the files from `modules/aios-runtime/` to the
   target locations in the main repository.
3. Run the test suite on the main repository.
4. Open a pull request.
5. Merge after the review is approved.

## 9. Risk Register

| Risk | Impact | Probability | Mitigation |
| --- | --- | --- | --- |
| The orchestrator deadlocks. | High | Low | The orchestrator uses a cancel flag. The orchestrator has a timeout. |
| The budget meter has a race. | High | Low | The meter uses locks. The meter is unit tested for concurrency. |
| The no-progress detector has false positives. | Medium | Medium | The detector uses a content hash. The detector has a tunable limit. |
| The resume handler is slow. | Medium | Low | The handler uses goroutines for async I/O. The handler caches the checkpoint. |

## 10. Schedule

| Phase | Duration | Dependency |
| --- | --- | --- |
| Phase 1: Project Skeleton | 1 day | nil |
| Phase 2: Budget Meter | 2 days | Phase 1 |
| Phase 3: No-Progress Detector | 1 day | Phase 1 |
| Phase 4: Resume Handler | 1 day | Phase 1 |
| Phase 5: Decision Tracer | 1 day | Phase 1 |
| Phase 6: Worker interface | 2 days | Phase 2, Phase 3 |
| Phase 7: Orchestrator | 3 days | All previous phases |
| Phase 8: Integration Tests | 2 days | Phase 7 |
| Phase 9: Quality Gates | 1 day | All phases |

The total duration is 14 days.
