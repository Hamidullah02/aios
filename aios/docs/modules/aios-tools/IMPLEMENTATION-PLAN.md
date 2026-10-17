# aios-tools — Implementation Plan

## 1. Purpose

This document states the implementation plan for the
aios-tools module.

## 2. Scope

The scope of this document is the aios-tools module.
The module depends on aios-core, aios-policy, and
aios-sandbox.

## 3. Reference Documents

- aios-tools PRD. See `docs/modules/aios-tools/PRD.md`.
- aios-tools SRS. See `docs/modules/aios-tools/SRS.md`.
- aios-tools TRD. See `docs/modules/aios-tools/TRD.md`.
- aios-tools Application Flow. See
  `docs/modules/aios-tools/APP-FLOW.md`.

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

### Phase 2: Tool interface

1. Create the file `internal/tools/interface.py`.
2. Create the file `tests/test_interface.py`.
3. Implement the `Tool` interface.
4. Implement the `ToolCall` schema.
5. Implement the `ToolResult` schema.
6. Write the test cases.

The deliverable is the tool interface with tests.

### Phase 3: Read File

1. Create the file `internal/tools/read_file.py`.
2. Create the file `tests/test_read_file.py`.
3. Implement the `ReadFileTool` class.
4. Implement the path validation.
5. Write the test cases.

The deliverable is the read file tool with tests.

### Phase 4: Write File

1. Create the file `internal/tools/write_file.py`.
2. Create the file `tests/test_write_file.py`.
3. Implement the `WriteFileTool` class.
4. Implement the atomic write.
5. Write the test cases.

The deliverable is the write file tool with tests.

### Phase 5: Search Files

1. Create the file `internal/tools/search_files.py`.
2. Create the file `tests/test_search_files.py`.
3. Implement the `SearchFilesTool` class.
4. Write the test cases.

The deliverable is the search files tool with tests.

### Phase 6: Run Shell

1. Create the file `internal/tools/run_shell.py`.
2. Create the file `tests/test_run_shell.py`.
3. Implement the `RunShellTool` class.
4. Write the test cases.

The deliverable is the run shell tool with tests.

### Phase 7: Tool Bus

1. Create the file `internal/tools/bus.py`.
2. Create the file `tests/test_bus.py`.
3. Implement the `ToolBus` class.
4. Implement the policy integration.
5. Implement the audit row write.
6. Write the test cases.

The deliverable is the tool bus with tests.

### Phase 8: Quality Gates

1. Run the test suite with `go test`.
2. Run the coverage report with `go test -cover`.
3. Run the type check with `go vet ./internal/tools`.
4. Run the linter with `gofmt + go vet check`.
5. Fix every failure.

The deliverable is a green test suite.

## 6. Deliverables

- The source files in `internal/tools/`.
- The test files in `tests/`.
- The `go.mod` file.
- The `README.md` file.

## 7. Test Plan

### 7.1 Unit Tests

- `tests/test_interface.py` covers
  `internal/tools/interface.py`.
- `tests/test_read_file.py` covers
  `internal/tools/read_file.py`.
- `tests/test_write_file.py` covers
  `internal/tools/write_file.py`.
- `tests/test_search_files.py` covers
  `internal/tools/search_files.py`.
- `tests/test_run_shell.py` covers
  `internal/tools/run_shell.py`.
- `tests/test_bus.py` covers
  `internal/tools/bus.py`.

### 7.2 Test Cases

The test cases include the following items.

- The read file tool returns the file content.
- The read file tool rejects paths outside the
  workspace.
- The write file tool writes the file content.
- The write file tool rejects paths outside the
  workspace.
- The write file tool uses atomic write.
- The search files tool returns the matching paths.
- The run shell tool runs the command.
- The run shell tool returns the exit code.
- The run shell tool times out.
- The tool bus dispatches every tool.
- The tool bus denies the call on policy deny.
- The tool bus writes the audit row.

## 8. Merge Plan

### 8.1 Pre-Merge Checklist

1. The test suite has a 100 percent pass rate.
2. The coverage report shows 100 percent coverage for
   the public functions.
3. The `go vet` report shows no errors.
4. The `gofmt + go vet` report shows no errors.

### 8.2 Merge Steps

1. Create a branch named `merge/aios-tools`.
2. Move the files from `modules/aios-tools/` to the
   target locations in the main repository.
3. Run the test suite on the main repository.
4. Open a pull request.
5. Merge after the review is approved.

## 9. Risk Register

| Risk | Impact | Probability | Mitigation |
| --- | --- | --- | --- |
| A tool reads a file outside the workspace. | High | Low | The path validation blocks the path. The tool test covers the path. |
| A tool writes a file outside the workspace. | High | Low | The path validation blocks the path. The tool test covers the path. |
| A shell command runs forever. | Medium | Medium | The tool has a timeout. The sandbox has a timeout. |
| The atomic write fails on a full disk. | Medium | Low | The temp file is removed on error. The tool returns an error. |

## 10. Schedule

| Phase | Duration | Dependency |
| --- | --- | --- |
| Phase 1: Project Skeleton | 1 day | nil |
| Phase 2: Tool interface | 1 day | Phase 1 |
| Phase 3: Read File | 1 day | Phase 2 |
| Phase 4: Write File | 1 day | Phase 2 |
| Phase 5: Search Files | 1 day | Phase 2 |
| Phase 6: Run Shell | 1 day | Phase 2 |
| Phase 7: Tool Bus | 2 days | All previous phases |
| Phase 8: Quality Gates | 1 day | All phases |

The total duration is 9 days.
