# aios-persist — Implementation Plan

## 1. Purpose

This document states the implementation plan for the
aios-persist module.

## 2. Scope

The scope of this document is the aios-persist module.
The module depends on aios-core and aios-policy.

## 3. Reference Documents

- aios-persist PRD. See
  `docs/modules/aios-persist/PRD.md`.
- aios-persist SRS. See
  `docs/modules/aios-persist/SRS.md`.
- aios-persist TRD. See
  `docs/modules/aios-persist/TRD.md`.
- aios-persist Application Flow. See
  `docs/modules/aios-persist/APP-FLOW.md`.
- aios-persist Wireframes. See
  `docs/modules/aios-persist/WIREFRAMES.md`.

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

### Phase 2: Rule Storage

1. Create the file `internal/persist/rules.py`.
2. Create the file `tests/test_rules.py`.
3. Implement the `RuleStorage` class.
4. Implement the user and project levels.
5. Write the test cases.

The deliverable is the rule storage with tests.

### Phase 3: Session Checkpoint

1. Create the file
  `internal/persist/checkpoint.py`.
2. Create the file `tests/test_checkpoint.py`.
3. Implement the `SessionCheckpoint` class.
4. Write the test cases.

The deliverable is the checkpoint with tests.

### Phase 4: Handoff

1. Create the file `internal/persist/handoff.py`.
2. Create the file `tests/test_handoff.py`.
3. Implement the `Handoff` class.
4. Write the test cases.

The deliverable is the handoff with tests.

### Phase 5: Session History

1. Create the file `internal/persist/history.py`.
2. Create the file `tests/test_history.py`.
3. Implement the `SessionHistory` class.
4. Write the test cases.

The deliverable is the history with tests.

### Phase 6: CLI Commands

1. Create the file `internal/persist/cli.py`.
2. Create the file `tests/test_cli.py`.
3. Implement the `aios rules` and `aios sessions`
   subcommands.
4. Write the test cases.

The deliverable is the CLI with tests.

### Phase 7: Quality Gates

1. Run the test suite with `go test`.
2. Run the coverage report with `go test -cover`.
3. Run the type check with `go vet ./internal/persist`.
4. Run the linter with `gofmt + go vet check`.
5. Fix every failure.

The deliverable is a green test suite.

## 6. Deliverables

- The source files in `internal/persist/`.
- The test files in `tests/`.
- The `go.mod` file.
- The `README.md` file.

## 7. Test Plan

### 7.1 Unit Tests

- `tests/test_rules.py` covers
  `internal/persist/rules.py`.
- `tests/test_checkpoint.py` covers
  `internal/persist/checkpoint.py`.
- `tests/test_handoff.py` covers
  `internal/persist/handoff.py`.
- `tests/test_history.py` covers
  `internal/persist/history.py`.
- `tests/test_cli.py` covers
  `internal/persist/cli.py`.

### 7.2 Test Cases

The test cases include the following items.

- The rule storage adds a rule.
- The rule storage removes a rule.
- The rule storage matches a rule.
- The session checkpoint saves the session.
- The session checkpoint loads the session.
- The handoff puts a value.
- The handoff gets a value.
- The history lists the events.
- The CLI lists the rules.

## 8. Merge Plan

### 8.1 Pre-Merge Checklist

1. The test suite has a 100 percent pass rate.
2. The coverage report shows 100 percent coverage
   for the public functions.
3. The `go vet` report shows no errors.
4. The `gofmt + go vet` report shows no errors.

### 8.2 Merge Steps

1. Create a branch named `merge/aios-persist`.
2. Move the files from `modules/aios-persist/` to
   the target locations in the main repository.
3. Run the test suite on the main repository.
4. Open a pull request.
5. Merge after the review is approved.

## 9. Risk Register

| Risk | Impact | Probability | Mitigation |
| --- | --- | --- | --- |
| The rule file is corrupted. | High | Low | The file is JSON-validated. The file is backed up. |
| The handoff file is too large. | Medium | Low | The handoff supports compression. |
| The session checkpoint is stale. | Medium | Low | The checkpoint has a TTL. |

## 10. Schedule

| Phase | Duration | Dependency |
| --- | --- | --- |
| Phase 1: Project Skeleton | 1 day | nil |
| Phase 2: Rule Storage | 2 days | Phase 1 |
| Phase 3: Session Checkpoint | 1 day | Phase 1 |
| Phase 4: Handoff | 1 day | Phase 1 |
| Phase 5: Session History | 1 day | Phase 1 |
| Phase 6: CLI Commands | 1 day | Phase 2 |
| Phase 7: Quality Gates | 1 day | All phases |

The total duration is 8 days.
