# aios-edit — Implementation Plan

## 1. Purpose

This document states the implementation plan for the
aios-edit module.

## 2. Scope

The scope of this document is the aios-edit module.
The module depends on aios-core and aios-tools.

## 3. Reference Documents

- aios-edit PRD. See `docs/modules/aios-edit/PRD.md`.
- aios-edit SRS. See `docs/modules/aios-edit/SRS.md`.
- aios-edit TRD. See `docs/modules/aios-edit/TRD.md`.
- aios-edit Application Flow. See
  `docs/modules/aios-edit/APP-FLOW.md`.

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

### Phase 2: Edit Tool

1. Create the file `internal/edit/edit.py`.
2. Create the file `tests/test_edit.py`.
3. Implement the `EditTool` class.
4. Implement the match count check.
5. Implement the atomic write.
6. Implement the audit row.
7. Write the test cases.

The deliverable is the Edit tool with tests.

### Phase 3: MultiEdit Tool

1. Create the file `internal/edit/multi_edit.py`.
2. Create the file `tests/test_multi_edit.py`.
3. Implement the `MultiEditTool` class.
4. Implement the in-memory edit.
5. Implement the rollback.
6. Write the test cases.

The deliverable is the MultiEdit tool with tests.

### Phase 4: apply_patch Tool

1. Create the file `internal/edit/apply_patch.py`.
2. Create the file `tests/test_apply_patch.py`.
3. Implement the `ApplyPatchTool` class.
4. Implement the patch parser.
5. Implement the hunk context check.
6. Implement the multi-file support.
7. Write the test cases.

The deliverable is the apply_patch tool with tests.

### Phase 5: NotebookEdit Tool

1. Create the file `internal/edit/notebook_edit.py`.
2. Create the file `tests/test_notebook_edit.py`.
3. Implement the `NotebookEditTool` class.
4. Write the test cases.

The deliverable is the NotebookEdit tool with tests.

### Phase 6: Write Tool

1. Create the file `internal/edit/write.py`.
2. Create the file `tests/test_write.py`.
3. Implement the `WriteTool` class.
4. Write the test cases.

The deliverable is the Write tool with tests.

### Phase 7: Quality Gates

1. Run the test suite with `go test`.
2. Run the coverage report with `go test -cover`.
3. Run the type check with `go vet ./internal/edit`.
4. Run the linter with `gofmt + go vet check`.
5. Fix every failure.

The deliverable is a green test suite.

## 6. Deliverables

- The source files in `internal/edit/`.
- The test files in `tests/`.
- The `go.mod` file.
- The `README.md` file.

## 7. Test Plan

### 7.1 Unit Tests

- `tests/test_edit.py` covers
  `internal/edit/edit.py`.
- `tests/test_multi_edit.py` covers
  `internal/edit/multi_edit.py`.
- `tests/test_apply_patch.py` covers
  `internal/edit/apply_patch.py`.
- `tests/test_notebook_edit.py` covers
  `internal/edit/notebook_edit.py`.
- `tests/test_write.py` covers
  `internal/edit/write.py`.

### 7.2 Test Cases

The test cases include the following items.

- The Edit tool replaces a unique match.
- The Edit tool fails on a non-unique match.
- The Edit tool replaces every match on
  replace_all.
- The Edit tool preserves the file encoding.
- The Edit tool writes the audit row.
- The MultiEdit tool applies the edits in order.
- The MultiEdit tool rolls back on the first
  failure.
- The apply_patch tool applies a unified diff.
- The apply_patch tool rejects a malformed diff.
- The apply_patch tool fails on hunk context
  mismatch.
- The NotebookEdit tool updates a cell.
- The NotebookEdit tool inserts a cell.
- The NotebookEdit tool deletes a cell.
- The Write tool uses the atomic write path.

## 8. Merge Plan

### 8.1 Pre-Merge Checklist

1. The test suite has a 100 percent pass rate.
2. The coverage report shows 100 percent coverage
   for the public functions.
3. The `go vet` report shows no errors.
4. The `gofmt + go vet` report shows no errors.

### 8.2 Merge Steps

1. Create a branch named `merge/aios-edit`.
2. Move the files from `modules/aios-edit/` to the
   target locations in the main repository.
3. Run the test suite on the main repository.
4. Open a pull request.
5. Merge after the review is approved.

## 9. Risk Register

| Risk | Impact | Probability | Mitigation |
| --- | --- | --- | --- |
| The Edit tool corrupts the file. | High | Low | The tool uses the atomic write. The tool validates the match count. |
| The MultiEdit tool leaves the file half-edited. | High | Low | The tool rolls back on the first failure. The test suite covers the rollback. |
| The apply_patch tool accepts a bad diff. | High | Low | The tool validates the hunk context. The tool rejects malformed diffs. |

## 10. Schedule

| Phase | Duration | Dependency |
| --- | --- | --- |
| Phase 1: Project Skeleton | 1 day | nil |
| Phase 2: Edit Tool | 2 days | Phase 1 |
| Phase 3: MultiEdit Tool | 1 day | Phase 2 |
| Phase 4: apply_patch Tool | 2 days | Phase 1 |
| Phase 5: NotebookEdit Tool | 1 day | Phase 1 |
| Phase 6: Write Tool | 1 day | Phase 1 |
| Phase 7: Quality Gates | 1 day | All phases |

The total duration is 9 days.
