# aios-optimizer — Implementation Plan

## 1. Purpose

This document states the implementation plan for the
aios-optimizer module.

## 2. Scope

The scope of this document is the aios-optimizer
module. The module depends on aios-core,
aios-models, and aios-verify.

## 3. Reference Documents

- aios-optimizer PRD. See
  `docs/modules/aios-optimizer/PRD.md`.
- aios-optimizer SRS. See
  `docs/modules/aios-optimizer/SRS.md`.
- aios-optimizer TRD. See
  `docs/modules/aios-optimizer/TRD.md`.
- aios-optimizer Application Flow. See
  `docs/modules/aios-optimizer/APP-FLOW.md`.
- aios-optimizer Wireframes. See
  `docs/modules/aios-optimizer/WIREFRAMES.md`.

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

### Phase 2: Tiered Model Registry

1. Create the file
  `internal/optimizer/registry.go`.
2. Create the file `tests/registry_test.go`.
3. Implement the `TieredModelRegistry` type.
4. Implement the YAML loader.
5. Write the test cases.

The deliverable is the registry with tests.

### Phase 3: Call Fingerprint

1. Create the file
  `internal/optimizer/fingerprint.go`.
2. Create the file `tests/fingerprint_test.go`.
3. Implement the `NewFingerprint` function.
4. Implement the prompt shape function.
5. Implement the context length bucket function.
6. Write the test cases.

The deliverable is the fingerprint with tests.

### Phase 4: Cost/Quality Table

1. Create the file `internal/optimizer/table.go`.
2. Create the file `tests/table_test.go`.
3. Implement the `CostQualityTable` type.
4. Implement the atomic write.
5. Implement the `Get`, `Record`, and `Reset`
   methods.
6. Write the test cases.

The deliverable is the table with tests.

### Phase 5: Router

1. Create the file `internal/optimizer/router.go`.
2. Create the file `tests/router_test.go`.
3. Implement the `Router` type.
4. Implement the `Route` method.
5. Write the test cases.

The deliverable is the router with tests.

### Phase 6: CLI Subcommand

1. Create the file `internal/optimizer/cli.go`.
2. Create the file `tests/cli_test.go`.
3. Implement the `aios stats` subcommand.
4. Implement the `--fingerprint` flag.
5. Implement the `--tier` flag.
6. Implement the `--reset` flag.
7. Write the test cases.

The deliverable is the CLI with tests.

### Phase 7: Runtime Integration

1. Create the file
  `internal/optimizer/integration.go`.
2. Create the file `tests/integration_test.go`.
3. Implement the integration with aios-runtime.
4. Implement the `--tier` flag on `aios run`.
5. Implement the `--optimizer` flag on `aios
   run`.
6. Implement the `--budget` flag on `aios run`.
7. Write the test cases.

The deliverable is the runtime integration with
tests.

### Phase 8: Quality Gates

1. Run the test suite with `go test ./...`.
2. Run the coverage report with
   `go test -cover ./...`.
3. Run the type check with `go vet ./...`.
4. Run the linter with `staticcheck ./...`.
5. Run the formatter with `gofmt -l .`.
6. Fix every failure.

The deliverable is a green test suite.

## 6. Deliverables

- The source files in `internal/optimizer/`.
- The test files in `tests/`.
- The `go.mod` file.
- The `README.md` file.
- A sample `var/optimizer.yaml` file.
- A sample `var/optimizer.json` file.

## 7. Test Plan

### 7.1 Unit Tests

- `tests/registry_test.go` covers
  `internal/optimizer/registry.go`.
- `tests/fingerprint_test.go` covers
  `internal/optimizer/fingerprint.go`.
- `tests/table_test.go` covers
  `internal/optimizer/table.go`.
- `tests/router_test.go` covers
  `internal/optimizer/router.go`.
- `tests/cli_test.go` covers
  `internal/optimizer/cli.go`.
- `tests/integration_test.go` covers
  `internal/optimizer/integration.go`.

### 7.2 Test Cases

The test cases include the following items.

- The registry loads the tiers.
- The registry returns `ErrUnknownTier` for an
  unknown tier.
- The fingerprint is a pure function. Two calls
  with the same inputs return the same
  fingerprint.
- The fingerprint differs when the prompt shape
  differs.
- The fingerprint differs when the tool set
  differs.
- The fingerprint differs when the role differs.
- The fingerprint differs when the context
  length bucket differs.
- The table loads from the file.
- The table writes to the file atomically.
- The table records the pass count and the
  total cost.
- The table resets to empty.
- The router returns the override tier when the
  override is set.
- The router returns the strong tier when the
  override is `off`.
- The router returns the highest-score tier
  when the fingerprint is in the table.
- The router returns the weakest tier when the
  fingerprint is not in the table.
- The router returns `ErrBudgetExhausted` when
  the budget is exhausted.
- The `aios stats` subcommand prints the table.
- The `aios stats --fingerprint=<hash>`
  subcommand filters the table.
- The `aios stats --reset` subcommand clears
  the table.

## 8. Merge Plan

### 8.1 Pre-Merge Checklist

1. The test suite has a 100 percent pass rate.
2. The coverage report shows 100 percent coverage
   for the public functions.
3. The `go vet` report is clean.
4. The `staticcheck` report is clean.
5. The `gofmt` report is clean.

### 8.2 Merge Steps

1. Create a branch named `merge/aios-optimizer`.
2. Move the files from `internal/optimizer/` to
   the target locations in the main repository.
3. Run the test suite on the main repository.
4. Open a pull request.
5. Merge after the review is approved.

## 9. Risk Register

| Risk | Impact | Probability | Mitigation |
| --- | --- | --- | --- |
| The fingerprint is too coarse. | Medium | Medium | The fingerprint includes the prompt shape, the tool set, the role, and the context length bucket. The developer can override per call. |
| The fingerprint is too fine. | Low | High | The table grows without bound. The developer can clear the table with `aios stats --reset`. |
| The pass rate is misleading. | High | Low | The pass rate is the empirical verify result. The developer can inspect the table. |
| The table write is not atomic. | High | Low | The table writes through a `rename` syscall. The write is atomic on POSIX file systems. |
| The budget is exceeded. | High | Low | The router enforces the budget. The router returns `ErrBudgetExhausted`. |
| The override is wrong. | Low | Medium | The override is recorded in the audit log. The developer can see the override count. |

## 10. Schedule

| Phase | Duration | Dependency |
| --- | --- | --- |
| Phase 1: Project Skeleton | 1 day | None |
| Phase 2: Tiered Model Registry | 1 day | Phase 1 |
| Phase 3: Call Fingerprint | 2 days | Phase 1 |
| Phase 4: Cost/Quality Table | 2 days | Phase 1 |
| Phase 5: Router | 2 days | Phase 2, Phase 3, Phase 4 |
| Phase 6: CLI Subcommand | 1 day | Phase 4 |
| Phase 7: Runtime Integration | 2 days | Phase 5, Phase 6 |
| Phase 8: Quality Gates | 1 day | All phases |

The total duration is 12 days.
