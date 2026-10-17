# aios-ask — Implementation Plan

## 1. Purpose

This document states the implementation plan for the
aios-ask module.

## 2. Scope

The scope of this document is the aios-ask module.
The module depends on aios-core.

## 3. Reference Documents

- aios-ask PRD. See `docs/modules/aios-ask/PRD.md`.
- aios-ask SRS. See `docs/modules/aios-ask/SRS.md`.
- aios-ask TRD. See `docs/modules/aios-ask/TRD.md`.
- aios-ask Application Flow. See
  `docs/modules/aios-ask/APP-FLOW.md`.
- aios-ask Wireframes. See
  `docs/modules/aios-ask/WIREFRAMES.md`.

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

### Phase 2: Schema

1. Create the file `internal/ask/schema.py`.
2. Create the file `tests/test_schema.py`.
3. Implement the `Question` class.
4. Implement the `Answer` class.
5. Implement the `AskOption` class.
6. Write the test cases.

The deliverable is the schema with tests.

### Phase 3: Tool

1. Create the file `internal/ask/tool.py`.
2. Create the file `tests/test_tool.py`.
3. Implement the `AskUserTool` class.
4. Implement the timeout.
5. Write the test cases.

The deliverable is the tool with tests.

### Phase 4: Quality Gates

1. Run the test suite with `go test`.
2. Run the coverage report with `go test -cover`.
3. Run the type check with `go vet ./internal/ask`.
4. Run the linter with `gofmt + go vet check`.
5. Fix every failure.

The deliverable is a green test suite.

## 6. Deliverables

- The source files in `internal/ask/`.
- The test files in `tests/`.
- The `go.mod` file.
- The `README.md` file.

## 7. Test Plan

### 7.1 Unit Tests

- `tests/test_schema.py` covers
  `internal/ask/schema.py`.
- `tests/test_tool.py` covers
  `internal/ask/tool.py`.

### 7.2 Test Cases

The test cases include the following items.

- The question is created.
- The question is validated.
- The answer is created.
- The tool runs the question.
- The tool returns the answer.
- The tool returns the default on timeout.
- The tool raises `AskTimeout` on timeout without
  default.

## 8. Merge Plan

### 8.1 Pre-Merge Checklist

1. The test suite has a 100 percent pass rate.
2. The coverage report shows 100 percent coverage for
   the public functions.
3. The `go vet` report shows no errors.
4. The `gofmt + go vet` report shows no errors.

### 8.2 Merge Steps

1. Create a branch named `merge/aios-ask`.
2. Move the files from `modules/aios-ask/` to the
   target locations in the main repository.
3. Run the test suite on the main repository.
4. Open a pull request.
5. Merge after the review is approved.

## 9. Risk Register

| Risk | Impact | Probability | Mitigation |
| --- | --- | --- | --- |
| The operator does not answer. | High | Medium | The tool has a timeout. The tool returns a default answer. |
| The question has no options. | High | Low | The tool validates the options. The tool raises an error when the options are empty. |

## 10. Schedule

| Phase | Duration | Dependency |
| --- | --- | --- |
| Phase 1: Project Skeleton | 1 day | nil |
| Phase 2: Schema | 1 day | Phase 1 |
| Phase 3: Tool | 1 day | Phase 2 |
| Phase 4: Quality Gates | 1 day | All phases |

The total duration is 4 days.
