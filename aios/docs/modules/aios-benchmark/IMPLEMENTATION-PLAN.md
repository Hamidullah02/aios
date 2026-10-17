# aios-benchmark — Implementation Plan

## 1. Purpose

This document states the implementation plan for the
aios-benchmark module.

## 2. Scope

The scope of this document is the aios-benchmark
module. The module depends on aios-core, aios-runtime,
and aios-models.

## 3. Reference Documents

- aios-benchmark PRD. See
  `docs/modules/aios-benchmark/PRD.md`.
- aios-benchmark SRS. See
  `docs/modules/aios-benchmark/SRS.md`.
- aios-benchmark TRD. See
  `docs/modules/aios-benchmark/TRD.md`.
- aios-benchmark Application Flow. See
  `docs/modules/aios-benchmark/APP-FLOW.md`.
- aios-benchmark Wireframes. See
  `docs/modules/aios-benchmark/WIREFRAMES.md`.

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

### Phase 2: Dataset

1. Create the file `internal/benchmark/dataset.py`.
2. Create the file `tests/test_dataset.py`.
3. Implement the `WedgeDataset` class.
4. Implement the wedge tasks.
5. Write the test cases.

The deliverable is the dataset with tests.

### Phase 3: Runner

1. Create the file `internal/benchmark/runner.py`.
2. Create the file `tests/test_runner.py`.
3. Implement the `BenchmarkRunner` class.
4. Implement the `Metrics` schema.
5. Write the test cases.

The deliverable is the runner with tests.

### Phase 4: Two-Tier Runner

1. Create the file `internal/benchmark/two_tier.py`.
2. Create the file `tests/test_two_tier.py`.
3. Implement the `TwoTierRunner` class.
4. Write the test cases.

The deliverable is the two-tier runner with tests.

### Phase 5: Loop Checker

1. Create the file `internal/benchmark/loop.py`.
2. Create the file `tests/test_loop.py`.
3. Implement the `LoopChecker` class.
4. Write the test cases.

The deliverable is the loop checker with tests.

### Phase 6: CLI

1. Create the file `internal/benchmark/cli.py`.
2. Create the file `tests/test_cli.py`.
3. Implement the CLI entry point.
4. Implement the report printer.
5. Write the test cases.

The deliverable is the CLI with tests.

### Phase 7: Quality Gates

1. Run the test suite with `go test`.
2. Run the coverage report with `go test -cover`.
3. Run the type check with `go vet ./internal/benchmark`.
4. Run the linter with `gofmt + go vet check`.
5. Fix every failure.

The deliverable is a green test suite.

## 6. Deliverables

- The source files in `internal/benchmark/`.
- The test files in `tests/`.
- The `go.mod` file.
- The `README.md` file.
- The sample dataset in `benchmark/wedge.json`.

## 7. Test Plan

### 7.1 Unit Tests

- `tests/test_dataset.py` covers
  `internal/benchmark/dataset.py`.
- `tests/test_runner.py` covers
  `internal/benchmark/runner.py`.
- `tests/test_two_tier.py` covers
  `internal/benchmark/two_tier.py`.
- `tests/test_loop.py` covers
  `internal/benchmark/loop.py`.
- `tests/test_cli.py` covers
  `internal/benchmark/cli.py`.

### 7.2 Test Cases

The test cases include the following items.

- The dataset loads the tasks.
- The runner runs the dataset.
- The runner records the metrics.
- The two-tier runner computes the delta.
- The loop checker records the loop count.
- The CLI prints the report.
- The CLI returns the exit code.

## 8. Merge Plan

### 8.1 Pre-Merge Checklist

1. The test suite has a 100 percent pass rate.
2. The coverage report shows 100 percent coverage for
   the public functions.
3. The `go vet` report shows no errors.
4. The `gofmt + go vet` report shows no errors.

### 8.2 Merge Steps

1. Create a branch named `merge/aios-benchmark`.
2. Move the files from `modules/aios-benchmark/` to
   the target locations in the main repository.
3. Run the test suite on the main repository.
4. Open a pull request.
5. Merge after the review is approved.

## 9. Risk Register

| Risk | Impact | Probability | Mitigation |
| --- | --- | --- | --- |
| The task exceeds the timeout. | Medium | Medium | The runner has a timeout. The runner records the failure. |
| The model cost is wrong. | High | Low | The runner uses the price table. The runner reconciles at the end. |
| The wedge dataset is incomplete. | High | Low | The dataset has a test case for every wedge. The dataset is reviewed. |

## 10. Schedule

| Phase | Duration | Dependency |
| --- | --- | --- |
| Phase 1: Project Skeleton | 1 day | nil |
| Phase 2: Dataset | 2 days | Phase 1 |
| Phase 3: Runner | 2 days | Phase 2 |
| Phase 4: Two-Tier Runner | 1 day | Phase 3 |
| Phase 5: Loop Checker | 1 day | Phase 3 |
| Phase 6: CLI | 1 day | Phase 3, Phase 4, Phase 5 |
| Phase 7: Quality Gates | 1 day | All phases |

The total duration is 9 days.
