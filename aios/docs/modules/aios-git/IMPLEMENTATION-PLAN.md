# aios-git — Implementation Plan

## 1. Purpose

This document states the implementation plan for the
aios-git module.

## 2. Scope

The scope of this document is the aios-git module.
The module depends on aios-core, aios-tools,
aios-policy, and aios-sandbox.

## 3. Reference Documents

- aios-git PRD. See `docs/modules/aios-git/PRD.md`.
- aios-git SRS. See `docs/modules/aios-git/SRS.md`.
- aios-git TRD. See `docs/modules/aios-git/TRD.md`.
- aios-git Application Flow. See
  `docs/modules/aios-git/APP-FLOW.md`.

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

### Phase 2: Status and Diff

1. Create the file `internal/git/status.py`.
2. Create the file `internal/git/diff.py`.
3. Create the file `tests/test_status.py`.
4. Create the file `tests/test_diff.py`.
5. Implement the `GitStatusTool` class.
6. Implement the `GitDiffTool` class.
7. Write the test cases.

The deliverable is the status and diff tools.

### Phase 3: Log Tool

1. Create the file `internal/git/log.py`.
2. Create the file `tests/test_log.py`.
3. Implement the `GitLogTool` class.
4. Write the test cases.

The deliverable is the log tool with tests.

### Phase 4: Commit Tool

1. Create the file `internal/git/commit.py`.
2. Create the file `tests/test_commit.py`.
3. Implement the `GitCommitTool` class.
4. Implement the policy integration.
5. Write the test cases.

The deliverable is the commit tool with tests.

### Phase 5: Branch and Worktree

1. Create the file `internal/git/branch.py`.
2. Create the file `internal/git/worktree.py`.
3. Create the file `tests/test_branch.py`.
4. Create the file `tests/test_worktree.py`.
5. Implement the `GitBranchTool` class.
6. Implement the `GitWorktreeTool` class.
7. Write the test cases.

The deliverable is the branch and worktree tools.

### Phase 6: Quality Gates

1. Run the test suite with `go test`.
2. Run the coverage report with `go test -cover`.
3. Run the type check with `go vet ./internal/git`.
4. Run the linter with `gofmt + go vet check`.
5. Fix every failure.

The deliverable is a green test suite.

## 6. Deliverables

- The source files in `internal/git/`.
- The test files in `tests/`.
- The `go.mod` file.
- The `README.md` file.

## 7. Test Plan

### 7.1 Unit Tests

- `tests/test_status.py` covers
  `internal/git/status.py`.
- `tests/test_diff.py` covers
  `internal/git/diff.py`.
- `tests/test_log.py` covers
  `internal/git/log.py`.
- `tests/test_commit.py` covers
  `internal/git/commit.py`.
- `tests/test_branch.py` covers
  `internal/git/branch.py`.
- `tests/test_worktree.py` covers
  `internal/git/worktree.py`.

### 7.2 Test Cases

The test cases include the following items.

- The status tool returns the status.
- The diff tool returns the diff.
- The log tool returns the log.
- The commit tool creates the commit.
- The commit tool requires approval.
- The branch tool creates and checks out the
  branch.
- The worktree tool creates the worktree.

## 8. Merge Plan

### 8.1 Pre-Merge Checklist

1. The test suite has a 100 percent pass rate.
2. The coverage report shows 100 percent coverage
   for the public functions.
3. The `go vet` report shows no errors.
4. The `gofmt + go vet` report shows no errors.

### 8.2 Merge Steps

1. Create a branch named `merge/aios-git`.
2. Move the files from `modules/aios-git/` to the
   target locations in the main repository.
3. Run the test suite on the main repository.
4. Open a pull request.
5. Merge after the review is approved.

## 9. Risk Register

| Risk | Impact | Probability | Mitigation |
| --- | --- | --- | --- |
| The git command fails. | Medium | Low | The tool raises `GitCommandError`. The tool returns the stderr. |
| The commit is created without approval. | High | Low | The tool checks the policy gateway. |
| The worktree path collides. | Medium | Low | The tool checks the path. |

## 10. Schedule

| Phase | Duration | Dependency |
| --- | --- | --- |
| Phase 1: Project Skeleton | 1 day | nil |
| Phase 2: Status and Diff | 1 day | Phase 1 |
| Phase 3: Log Tool | 1 day | Phase 1 |
| Phase 4: Commit Tool | 1 day | Phase 1 |
| Phase 5: Branch and Worktree | 2 days | Phase 1 |
| Phase 6: Quality Gates | 1 day | All phases |

The total duration is 7 days.
