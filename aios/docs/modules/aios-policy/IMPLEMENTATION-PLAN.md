# aios-policy — Implementation Plan

## 1. Purpose

This document states the implementation plan for the
aios-policy module.

## 2. Scope

The scope of this document is the aios-policy module.
The module depends on aios-core.

## 3. Reference Documents

- aios-policy PRD. See `docs/modules/aios-policy/PRD.md`.
- aios-policy SRS. See `docs/modules/aios-policy/SRS.md`.
- aios-policy TRD. See `docs/modules/aios-policy/TRD.md`.
- aios-policy Application Flow. See
  `docs/modules/aios-policy/APP-FLOW.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Phase | A unit of work in the implementation plan. |
| Deliverable | A file or a test that the phase produces. |

## 5. Phases

### Phase 1: Project Skeleton

1. Create the folder `modules/aios-policy/`.
2. Create the folder `modules/aios-policy/internal/policy/`.
3. Create the folder `modules/aios-policy/tests/`.
4. Create the file `go.mod` with the package
   metadata.
5. Create the file `README.md` with the module purpose.

The deliverable is the project skeleton.

### Phase 2: Schema

1. Create the file `internal/policy/schema.py`.
2. Create the file `tests/test_schema.py`.
3. Implement the `Match` class.
4. Implement the `Rule` class.
5. Implement the `Decision` class.
6. Write the test cases for each class.

The deliverable is the schema component with tests.

### Phase 3: Loader

1. Create the file `internal/policy/loader.py`.
2. Create the file `tests/test_loader.py`.
3. Implement the `load_policy` function.
4. Implement the include directive.
5. Write the test cases for valid files.
6. Write the test cases for invalid files.

The deliverable is the loader component with tests.

### Phase 4: Gateway

1. Create the file `internal/policy/gateway.py`.
2. Create the file `tests/test_gateway.py`.
3. Implement the `PolicyGateway` class.
4. Implement the first-match-wins algorithm.
5. Implement the match check algorithm.
6. Implement the template substitution algorithm.
7. Implement the audit row write.
8. Write the test cases for each match condition.
9. Write the test cases for each action.
10. Write the test case for the default deny.

The deliverable is the gateway component with tests.

### Phase 5: Integration Tests

1. Create the file `tests/test_integration.py`.
2. Write the test case for a full policy file.
3. Write the test case for an allow-ask-deny chain.
4. Write the test case for the audit row.

The deliverable is the integration test suite.

### Phase 6: Quality Gates

1. Run the test suite with `go test`.
2. Run the coverage report with `go test -cover`.
3. Run the type check with `go vet ./internal/policy`.
4. Run the linter with `gofmt + go vet check`.
5. Fix every failure.

The deliverable is a green test suite.

## 6. Deliverables

The module produces the following deliverables.

- The source files in `internal/policy/`.
- The test files in `tests/`.
- The `go.mod` file.
- The `README.md` file.
- A sample policy file in `examples/`.

## 7. Test Plan

### 7.1 Unit Tests

The unit tests cover every public function in the
module.

- `tests/test_schema.py` covers
  `internal/policy/schema.py`.
- `tests/test_loader.py` covers
  `internal/policy/loader.py`.
- `tests/test_gateway.py` covers
  `internal/policy/gateway.py`.

### 7.2 Test Cases

The test cases include the following items.

- A rule matches by tool name.
- A rule matches by tool glob.
- A rule matches by path prefix.
- A rule matches by path glob.
- A rule matches by host glob.
- A rule matches by tier.
- A rule matches by actor.
- A rule matches by task.
- The default deny is returned when no rule matches.
- An `ask` action includes the approval payload.
- The audit row is written for every decision.
- The template substitution works.
- The policy file with includes is loaded.

## 8. Merge Plan

### 8.1 Pre-Merge Checklist

1. The test suite has a 100 percent pass rate.
2. The coverage report shows 100 percent coverage for
   the public functions.
3. The `go vet` report shows no errors.
4. The `gofmt + go vet` report shows no errors.
5. The module is documented in the `README.md`.

### 8.2 Merge Steps

1. Create a branch named `merge/aios-policy`.
2. Move the files from `modules/aios-policy/` to the
   target locations in the main repository.
3. Run the test suite on the main repository.
4. Open a pull request.
5. Merge after the review is approved.

## 9. Risk Register

| Risk | Impact | Probability | Mitigation |
| --- | --- | --- | --- |
| The policy file has a syntax error. | High | Medium | The loader validates the file. The loader returns a clear error. |
| The glob match is slow. | Medium | Low | The gateway caches the compiled patterns. |
| The template substitution leaks secrets. | High | Low | The template only references declared values. The audit row is redacted. |
| The gateway has a race condition. | High | Low | The gateway is stateless. The gateway is safe for concurrent use. |

## 10. Schedule

| Phase | Duration | Dependency |
| --- | --- | --- |
| Phase 1: Project Skeleton | 1 day | nil |
| Phase 2: Schema | 1 day | Phase 1 |
| Phase 3: Loader | 1 day | Phase 2 |
| Phase 4: Gateway | 3 days | Phase 2, Phase 3 |
| Phase 5: Integration Tests | 1 day | Phase 4 |
| Phase 6: Quality Gates | 1 day | All phases |

The total duration is 8 days.
