# aios-memory — Implementation Plan

## 1. Purpose

This document states the implementation plan for the
aios-memory module.

## 2. Scope

The scope of this document is the aios-memory module.
The module depends on aios-core and aios-models.

## 3. Reference Documents

- aios-memory PRD. See `docs/modules/aios-memory/PRD.md`.
- aios-memory SRS. See `docs/modules/aios-memory/SRS.md`.
- aios-memory TRD. See `docs/modules/aios-memory/TRD.md`.
- aios-memory Application Flow. See
  `docs/modules/aios-memory/APP-FLOW.md`.
- aios-memory Wireframes. See
  `docs/modules/aios-memory/WIREFRAMES.md`.

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

### Phase 2: Ignore Loader

1. Create the file `internal/memory/ignore.py`.
2. Create the file `tests/test_ignore.py`.
3. Implement the `IgnoreLoader` class.
4. Implement the pathspec integration.
5. Write the test cases.

The deliverable is the ignore loader with tests.

### Phase 3: Project Memory Loader

1. Create the file
  `internal/memory/project_memory.py`.
2. Create the file `tests/test_project_memory.py`.
3. Implement the `ProjectMemoryLoader` class.
4. Implement the merge algorithm.
5. Implement the mtime cache.
6. Write the test cases.

The deliverable is the project memory loader.

### Phase 4: Conventions Loader

1. Create the file
  `internal/memory/conventions.py`.
2. Create the file `tests/test_conventions.py`.
3. Implement the `ConventionsLoader` class.
4. Write the test cases.

The deliverable is the conventions loader.

### Phase 5: Context Compactor

1. Create the file
  `internal/memory/compaction.py`.
2. Create the file `tests/test_compaction.py`.
3. Implement the `ContextCompactor` class.
4. Implement the sliding window.
5. Write the test cases.

The deliverable is the context compactor.

### Phase 6: Quality Gates

1. Run the test suite with `go test`.
2. Run the coverage report with `go test -cover`.
3. Run the type check with `go vet ./internal/memory`.
4. Run the linter with `gofmt + go vet check`.
5. Fix every failure.

The deliverable is a green test suite.

## 6. Deliverables

- The source files in `internal/memory/`.
- The test files in `tests/`.
- The `go.mod` file.
- The `README.md` file.
- The AGENTS.md template in
  `internal/memory/templates/AGENTS.md`.
- The .aiosignore template in
  `internal/memory/templates/aiosignore`.

## 7. Test Plan

### 7.1 Unit Tests

- `tests/test_ignore.py` covers
  `internal/memory/ignore.py`.
- `tests/test_project_memory.py` covers
  `internal/memory/project_memory.py`.
- `tests/test_conventions.py` covers
  `internal/memory/conventions.py`.
- `tests/test_compaction.py` covers
  `internal/memory/compaction.py`.

### 7.2 Test Cases

The test cases include the following items.

- The ignore loader loads the .aiosignore file.
- The ignore loader supports the negation.
- The project memory loader loads the AGENTS.md
  file.
- The project memory loader merges the files.
- The project memory loader caches the result.
- The conventions loader loads the CONVENTIONS.md
  file.
- The context compactor summarizes the context.

## 8. Merge Plan

### 8.1 Pre-Merge Checklist

1. The test suite has a 100 percent pass rate.
2. The coverage report shows 100 percent coverage
   for the public functions.
3. The `go vet` report shows no errors.
4. The `gofmt + go vet` report shows no errors.

### 8.2 Merge Steps

1. Create a branch named `merge/aios-memory`.
2. Move the files from `modules/aios-memory/` to
   the target locations in the main repository.
3. Run the test suite on the main repository.
4. Open a pull request.
5. Merge after the review is approved.

## 9. Risk Register

| Risk | Impact | Probability | Mitigation |
| --- | --- | --- | --- |
| The AGENTS.md file is too large. | Medium | Medium | The loader has a size limit. |
| The compaction changes the meaning. | High | Low | The compactor keeps the last N messages verbatim. |
| The ignore pattern blocks a needed file. | Medium | Low | The loader supports the `!` negation. |

## 10. Schedule

| Phase | Duration | Dependency |
| --- | --- | --- |
| Phase 1: Project Skeleton | 1 day | nil |
| Phase 2: Ignore Loader | 1 day | Phase 1 |
| Phase 3: Project Memory Loader | 2 days | Phase 1 |
| Phase 4: Conventions Loader | 1 day | Phase 1 |
| Phase 5: Context Compactor | 2 days | Phase 1 |
| Phase 6: Quality Gates | 1 day | All phases |

The total duration is 8 days.
