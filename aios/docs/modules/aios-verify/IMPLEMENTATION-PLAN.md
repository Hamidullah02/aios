# aios-verify — Implementation Plan

## 1. Purpose

This document states the implementation plan for the
aios-verify module.

## 2. Scope

The scope of this document is the aios-verify module.
The module depends on aios-core, aios-tools,
aios-models, and aios-sandbox.

## 3. Reference Documents

- aios-verify PRD. See
  `docs/modules/aios-verify/PRD.md`.
- aios-verify SRS. See
  `docs/modules/aios-verify/SRS.md`.
- aios-verify TRD. See
  `docs/modules/aios-verify/TRD.md`.
- aios-verify Application Flow. See
  `docs/modules/aios-verify/APP-FLOW.md`.

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

### Phase 2: Verification Result

1. Create the file `internal/verify/result.py`.
2. Create the file `tests/test_result.py`.
3. Implement the `VerificationResult` schema.
4. Implement the `TestResult` schema.
5. Implement the `TestFailure` schema.
6. Write the test cases.

The deliverable is the result schemas with tests.

### Phase 3: Verification Prompt

1. Create the file `internal/verify/prompt.py`.
2. Create the file `tests/test_prompt.py`.
3. Implement the `VerificationPrompt` class.
4. Write the test cases.

The deliverable is the prompt with tests.

### Phase 4: run_tests Tool

1. Create the file `internal/verify/run_tests.py`.
2. Create the file `tests/test_run_tests.py`.
3. Implement the `RunTestsTool` class.
4. Implement the per-runner parsers.
5. Write the test cases.

The deliverable is the run_tests tool with tests.

### Phase 5: Self-Check Tool

1. Create the file `internal/verify/self_check.py`.
2. Create the file `tests/test_self_check.py`.
3. Implement the `SelfCheckTool` class.
4. Write the test cases.

The deliverable is the self-check tool with tests.

### Phase 6: Quality Gates

1. Run the test suite with `go test`.
2. Run the coverage report with `go test -cover`.
3. Run the type check with `go vet ./internal/verify`.
4. Run the linter with `gofmt + go vet check`.
5. Fix every failure.

The deliverable is a green test suite.

## 6. Deliverables

- The source files in `internal/verify/`.
- The test files in `tests/`.
- The `go.mod` file.
- The `README.md` file.
- The default verification prompt in
  `internal/verify/templates/verify_prompt.txt`.

## 7. Test Plan

### 7.1 Unit Tests

- `tests/test_result.py` covers
  `internal/verify/result.py`.
- `tests/test_prompt.py` covers
  `internal/verify/prompt.py`.
- `tests/test_run_tests.py` covers
  `internal/verify/run_tests.py`.
- `tests/test_self_check.py` covers
  `internal/verify/self_check.py`.

### 7.2 Test Cases

The test cases include the following items.

- The result schema validates.
- The prompt builds the verification prompt.
- The run_tests tool runs go test.
- The run_tests tool runs jest.
- The self-check tool returns the verification
  result.

## 8. Merge Plan

### 8.1 Pre-Merge Checklist

1. The test suite has a 100 percent pass rate.
2. The coverage report shows 100 percent coverage
   for the public functions.
3. The `go vet` report shows no errors.
4. The `gofmt + go vet` report shows no errors.

### 8.2 Merge Steps

1. Create a branch named `merge/aios-verify`.
2. Move the files from `modules/aios-verify/` to
   the target locations in the main repository.
3. Run the test suite on the main repository.
4. Open a pull request.
5. Merge after the review is approved.

## 9. Risk Register

| Risk | Impact | Probability | Mitigation |
| --- | --- | --- | --- |
| The self-check model is wrong. | High | Medium | The tool supports a custom prompt. |
| The test runner output changes. | Medium | Medium | The tool supports a custom parser. |
| The verification result is not typed. | Medium | Low | The component uses struct + go-playground/validator (or hand-rolled tag validation). |

## 10. Schedule

| Phase | Duration | Dependency |
| --- | --- | --- |
| Phase 1: Project Skeleton | 1 day | nil |
| Phase 2: Verification Result | 1 day | Phase 1 |
| Phase 3: Verification Prompt | 1 day | Phase 1 |
| Phase 4: run_tests Tool | 2 days | Phase 1 |
| Phase 5: Self-Check Tool | 1 day | Phase 2, Phase 3 |
| Phase 6: Quality Gates | 1 day | All phases |

The total duration is 7 days.
