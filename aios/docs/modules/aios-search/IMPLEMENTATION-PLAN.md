# aios-search — Implementation Plan

## 1. Purpose

This document states the implementation plan for the
aios-search module.

## 2. Scope

The scope of this document is the aios-search module.
The module depends on aios-core, aios-tools, and
aios-sandbox.

## 3. Reference Documents

- aios-search PRD. See `docs/modules/aios-search/PRD.md`.
- aios-search SRS. See `docs/modules/aios-search/SRS.md`.
- aios-search TRD. See `docs/modules/aios-search/TRD.md`.
- aios-search Application Flow. See
  `docs/modules/aios-search/APP-FLOW.md`.

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

### Phase 2: Glob Tool

1. Create the file `internal/search/glob.py`.
2. Create the file `tests/test_glob.py`.
3. Implement the `GlobTool` class.
4. Implement the `.aiosignore` integration.
5. Write the test cases.

The deliverable is the Glob tool with tests.

### Phase 3: Grep Tool

1. Create the file `internal/search/grep.py`.
2. Create the file `tests/test_grep.py`.
3. Implement the `GrepTool` class.
4. Implement the ripgrep integration.
5. Implement the output modes.
6. Write the test cases.

The deliverable is the Grep tool with tests.

### Phase 4: CodeSearch Tool

1. Create the file `internal/search/code_search.py`.
2. Create the file `tests/test_code_search.py`.
3. Implement the `CodeSearchTool` class.
4. Implement the tree-sitter integration.
5. Write the test cases.

The deliverable is the CodeSearch tool with tests.

### Phase 5: Diagnostics Tool

1. Create the file `internal/search/diagnostics.py`.
2. Create the file `tests/test_diagnostics.py`.
3. Implement the `DiagnosticsTool` class.
4. Implement the per-checker parsers.
5. Write the test cases.

The deliverable is the Diagnostics tool with tests.

### Phase 6: Quality Gates

1. Run the test suite with `go test`.
2. Run the coverage report with `go test -cover`.
3. Run the type check with `go vet ./internal/search`.
4. Run the linter with `gofmt + go vet check`.
5. Fix every failure.

The deliverable is a green test suite.

## 6. Deliverables

- The source files in `internal/search/`.
- The test files in `tests/`.
- The `go.mod` file.
- The `README.md` file.

## 7. Test Plan

### 7.1 Unit Tests

- `tests/test_glob.py` covers
  `internal/search/glob.py`.
- `tests/test_grep.py` covers
  `internal/search/grep.py`.
- `tests/test_code_search.py` covers
  `internal/search/code_search.py`.
- `tests/test_diagnostics.py` covers
  `internal/search/diagnostics.py`.

### 7.2 Test Cases

The test cases include the following items.

- The Glob tool returns the matching paths.
- The Glob tool respects `.aiosignore`.
- The Grep tool returns the matching lines.
- The Grep tool supports the output modes.
- The Grep tool respects `.aiosignore`.
- The CodeSearch tool returns the symbol
  location.
- The Diagnostics tool returns the diagnostics.

## 8. Merge Plan

### 8.1 Pre-Merge Checklist

1. The test suite has a 100 percent pass rate.
2. The coverage report shows 100 percent coverage
   for the public functions.
3. The `go vet` report shows no errors.
4. The `gofmt + go vet` report shows no errors.

### 8.2 Merge Steps

1. Create a branch named `merge/aios-search`.
2. Move the files from `modules/aios-search/` to
   the target locations in the main repository.
3. Run the test suite on the main repository.
4. Open a pull request.
5. Merge after the review is approved.

## 9. Risk Register

| Risk | Impact | Probability | Mitigation |
| --- | --- | --- | --- |
| The Grep tool is slow on large repos. | Medium | Medium | The tool uses ripgrep. The tool respects `.aiosignore`. |
| The CodeSearch tool has no grammar. | Medium | Medium | The tool raises `UnsupportedLanguageError`. |
| The Diagnostics tool misses a checker. | Low | Medium | The tool supports a custom checker. |

## 10. Schedule

| Phase | Duration | Dependency |
| --- | --- | --- |
| Phase 1: Project Skeleton | 1 day | nil |
| Phase 2: Glob Tool | 1 day | Phase 1 |
| Phase 3: Grep Tool | 2 days | Phase 1 |
| Phase 4: CodeSearch Tool | 3 days | Phase 1 |
| Phase 5: Diagnostics Tool | 2 days | Phase 1 |
| Phase 6: Quality Gates | 1 day | All phases |

The total duration is 10 days.
