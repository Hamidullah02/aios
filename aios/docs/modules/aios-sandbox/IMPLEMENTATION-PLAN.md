# aios-sandbox — Implementation Plan

## 1. Purpose

This document states the implementation plan for the
aios-sandbox module.

## 2. Scope

The scope of this document is the aios-sandbox module.
The module depends on aios-core.

## 3. Reference Documents

- aios-sandbox PRD. See `docs/modules/aios-sandbox/PRD.md`.
- aios-sandbox SRS. See `docs/modules/aios-sandbox/SRS.md`.
- aios-sandbox TRD. See `docs/modules/aios-sandbox/TRD.md`.
- aios-sandbox Application Flow. See
  `docs/modules/aios-sandbox/APP-FLOW.md`.

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

### Phase 2: Resource Limits

1. Create the file `internal/sandbox/limits.py`.
2. Create the file `tests/test_limits.py`.
3. Implement the `ResourceLimits` class.
4. Write the test cases.

The deliverable is the resource limits with tests.

### Phase 3: os/exec wrapper

1. Create the file `internal/sandbox/subprocess.py`.
2. Create the file `tests/test_subprocess.py`.
3. Implement the `SubprocessRunner` class.
4. Write the test cases.

The deliverable is the runner with tests.

### Phase 4: Quality Gates

1. Run the test suite with `go test`.
2. Run the coverage report with `go test -cover`.
3. Run the type check with `go vet ./internal/sandbox`.
4. Run the linter with `gofmt + go vet check`.
5. Fix every failure.

The deliverable is a green test suite.

## 6. Deliverables

- The source files in `internal/sandbox/`.
- The test files in `tests/`.
- The `go.mod` file.
- The `README.md` file.

## 7. Test Plan

### 7.1 Unit Tests

- `tests/test_limits.py` covers
  `internal/sandbox/limits.py`.
- `tests/test_subprocess.py` covers
  `internal/sandbox/subprocess.py`.

### 7.2 Test Cases

The test cases include the following items.

- The resource limits are created.
- The os/exec wrapper runs a command.
- The os/exec wrapper enforces the timeout.
- The os/exec wrapper enforces the memory limit.
- The os/exec wrapper enforces the CPU limit.
- The os/exec wrapper sets the working directory.
- The os/exec wrapper sets the environment
  variables.

## 8. Merge Plan

### 8.1 Pre-Merge Checklist

1. The test suite has a 100 percent pass rate.
2. The coverage report shows 100 percent coverage for
   the public functions.
3. The `go vet` report shows no errors.
4. The `gofmt + go vet` report shows no errors.

### 8.2 Merge Steps

1. Create a branch named `merge/aios-sandbox`.
2. Move the files from `modules/aios-sandbox/` to
   the target locations in the main repository.
3. Run the test suite on the main repository.
4. Open a pull request.
5. Merge after the review is approved.

## 9. Risk Register

| Risk | Impact | Probability | Mitigation |
| --- | --- | --- | --- |
| The process exceeds the memory limit. | Medium | Medium | The runner sets `RLIMIT_AS`. The runner kills the process. |
| The process forks a child. | Medium | Low | The runner sets the process limit. The runner kills the process group. |
| The resource limit is not supported on Windows. | Medium | High | The runner falls back to a timeout. The runner documents the limit. |

## 10. Schedule

| Phase | Duration | Dependency |
| --- | --- | --- |
| Phase 1: Project Skeleton | 1 day | nil |
| Phase 2: Resource Limits | 1 day | Phase 1 |
| Phase 3: os/exec wrapper | 2 days | Phase 2 |
| Phase 4: Quality Gates | 1 day | All phases |

The total duration is 5 days.
