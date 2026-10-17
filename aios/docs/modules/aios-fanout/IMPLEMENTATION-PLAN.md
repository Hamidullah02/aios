# aios-fanout — Implementation Plan

## 1. Purpose

This document states the implementation plan for the
aios-fanout module.

## 2. Scope

The scope of this document is the aios-fanout module.
The module depends on aios-core, aios-runtime,
aios-git, aios-watch, and aios-models.

## 3. Reference Documents

- aios-fanout PRD. See
  `docs/modules/aios-fanout/PRD.md`.
- aios-fanout SRS. See
  `docs/modules/aios-fanout/SRS.md`.
- aios-fanout TRD. See
  `docs/modules/aios-fanout/TRD.md`.
- aios-fanout Application Flow. See
  `docs/modules/aios-fanout/APP-FLOW.md`.

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

### Phase 2: Worktree Isolation

1. Create the file `internal/fanout/worktree.py`.
2. Create the file `tests/test_worktree.py`.
3. Implement the `WorktreeIsolation` class.
4. Write the test cases.

The deliverable is the worktree isolation with tests.

### Phase 3: Result Merger

1. Create the file `internal/fanout/merge.py`.
2. Create the file `tests/test_merge.py`.
3. Implement the `ResultMerger` class.
4. Implement the strategies.
5. Write the test cases.

The deliverable is the result merger with tests.

### Phase 4: Fan-out Tool

1. Create the file `internal/fanout/fanout.py`.
2. Create the file `tests/test_fanout.py`.
3. Implement the `FanOutTool` class.
4. Implement the parallel calls.
5. Implement the worktree integration.
6. Write the test cases.

The deliverable is the fan-out tool with tests.

### Phase 5: Scheduler

1. Create the file `internal/fanout/scheduler.py`.
2. Create the file `tests/test_scheduler.py`.
3. Implement the `Scheduler` class.
4. Implement the batch runner.
5. Write the test cases.

The deliverable is the scheduler with tests.

### Phase 6: Quality Gates

1. Run the test suite with `go test`.
2. Run the coverage report with `go test -cover`.
3. Run the type check with `go vet ./internal/fanout`.
4. Run the linter with `gofmt + go vet check`.
5. Fix every failure.

The deliverable is a green test suite.

## 6. Deliverables

- The source files in `internal/fanout/`.
- The test files in `tests/`.
- The `go.mod` file.
- The `README.md` file.

## 7. Test Plan

### 7.1 Unit Tests

- `tests/test_worktree.py` covers
  `internal/fanout/worktree.py`.
- `tests/test_merge.py` covers
  `internal/fanout/merge.py`.
- `tests/test_fanout.py` covers
  `internal/fanout/fanout.py`.
- `tests/test_scheduler.py` covers
  `internal/fanout/scheduler.py`.

### 7.2 Test Cases

The test cases include the following items.

- The worktree isolation creates a worktree.
- The worktree isolation removes a worktree.
- The result merger concatenates the results.
- The result merger summarizes the results.
- The result merger votes the results.
- The fan-out tool spawns a sub-agent for every
  task.
- The fan-out tool runs the sub-agents in
  parallel.
- The scheduler respects the budget.

## 8. Merge Plan

### 8.1 Pre-Merge Checklist

1. The test suite has a 100 percent pass rate.
2. The coverage report shows 100 percent coverage
   for the public functions.
3. The `go vet` report shows no errors.
4. The `gofmt + go vet` report shows no errors.

### 8.2 Merge Steps

1. Create a branch named `merge/aios-fanout`.
2. Move the files from `modules/aios-fanout/` to
   the target locations in the main repository.
3. Run the test suite on the main repository.
4. Open a pull request.
5. Merge after the review is approved.

## 9. Risk Register

| Risk | Impact | Probability | Mitigation |
| --- | --- | --- | --- |
| The sub-agent runs forever. | Medium | Low | The sub-agent has a timeout. The scheduler has a timeout. |
| The worktree path collides. | Medium | Low | The worktree path includes a unique id. |
| The merge strategy loses information. | Medium | Low | The merger supports the `concat` strategy. |

## 10. Schedule

| Phase | Duration | Dependency |
| --- | --- | --- |
| Phase 1: Project Skeleton | 1 day | nil |
| Phase 2: Worktree Isolation | 1 day | Phase 1 |
| Phase 3: Result Merger | 1 day | Phase 1 |
| Phase 4: Fan-out Tool | 3 days | Phase 2, Phase 3 |
| Phase 5: Scheduler | 1 day | Phase 4 |
| Phase 6: Quality Gates | 1 day | All phases |

The total duration is 8 days.
