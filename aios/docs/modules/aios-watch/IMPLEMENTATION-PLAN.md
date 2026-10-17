# aios-watch — Implementation Plan

## 1. Purpose

This document states the implementation plan for the
aios-watch module.

## 2. Scope

The scope of this document is the aios-watch module.
The module depends on aios-core, aios-models,
aios-runtime, and aios-memory.

## 3. Reference Documents

- aios-watch PRD. See `docs/modules/aios-watch/PRD.md`.
- aios-watch SRS. See `docs/modules/aios-watch/SRS.md`.
- aios-watch TRD. See `docs/modules/aios-watch/TRD.md`.
- aios-watch Application Flow. See
  `docs/modules/aios-watch/APP-FLOW.md`.
- aios-watch Wireframes. See
  `docs/modules/aios-watch/WIREFRAMES.md`.

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

### Phase 2: Stream Channel

1. Create the file `internal/watch/stream.py`.
2. Create the file `tests/test_stream.py`.
3. Implement the `StreamChannel` class.
4. Implement the subscriber list.
5. Write the test cases.

The deliverable is the stream channel with tests.

### Phase 3: File Watcher Tool

1. Create the file `internal/watch/file_watcher.py`.
2. Create the file `tests/test_file_watcher.py`.
3. Implement the `FileWatcherTool` class.
4. Implement the github.com/fsnotify/fsnotify integration.
5. Implement the ignore integration.
6. Write the test cases.

The deliverable is the file watcher with tests.

### Phase 4: Sub-agent Task Tool

1. Create the file `internal/watch/task.py`.
2. Create the file `tests/test_task.py`.
3. Implement the `TaskTool` class.
4. Implement the sub-agent factory.
5. Implement the parallel calls.
6. Write the test cases.

The deliverable is the Task tool with tests.

### Phase 5: Quality Gates

1. Run the test suite with `go test`.
2. Run the coverage report with `go test -cover`.
3. Run the type check with `go vet ./internal/watch`.
4. Run the linter with `gofmt + go vet check`.
5. Fix every failure.

The deliverable is a green test suite.

## 6. Deliverables

- The source files in `internal/watch/`.
- The test files in `tests/`.
- The `go.mod` file.
- The `README.md` file.

## 7. Test Plan

### 7.1 Unit Tests

- `tests/test_stream.py` covers
  `internal/watch/stream.py`.
- `tests/test_file_watcher.py` covers
  `internal/watch/file_watcher.py`.
- `tests/test_task.py` covers
  `internal/watch/task.py`.

### 7.2 Test Cases

The test cases include the following items.

- The stream channel yields chunks.
- The stream channel supports multiple
  subscribers.
- The file watcher detects file changes.
- The file watcher respects `.aiosignore`.
- The Task tool spawns a sub-agent.
- The Task tool runs parallel calls.

## 8. Merge Plan

### 8.1 Pre-Merge Checklist

1. The test suite has a 100 percent pass rate.
2. The coverage report shows 100 percent coverage
   for the public functions.
3. The `go vet` report shows no errors.
4. The `gofmt + go vet` report shows no errors.

### 8.2 Merge Steps

1. Create a branch named `merge/aios-watch`.
2. Move the files from `modules/aios-watch/` to
   the target locations in the main repository.
3. Run the test suite on the main repository.
4. Open a pull request.
5. Merge after the review is approved.

## 9. Risk Register

| Risk | Impact | Probability | Mitigation |
| --- | --- | --- | --- |
| The stream channel is slow. | Medium | Low | The channel is async. The channel buffers the chunks. |
| The file watcher misses an event. | Medium | Low | The watcher uses `github.com/fsnotify/fsnotify`. |
| The sub-agent runs forever. | Medium | Low | The sub-agent has a timeout. |

## 10. Schedule

| Phase | Duration | Dependency |
| --- | --- | --- |
| Phase 1: Project Skeleton | 1 day | nil |
| Phase 2: Stream Channel | 1 day | Phase 1 |
| Phase 3: File Watcher Tool | 1 day | Phase 1 |
| Phase 4: Sub-agent Task Tool | 3 days | Phase 1 |
| Phase 5: Quality Gates | 1 day | All phases |

The total duration is 7 days.
