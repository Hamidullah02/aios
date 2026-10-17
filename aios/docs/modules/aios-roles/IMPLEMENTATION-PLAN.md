# aios-roles — Implementation Plan

## 1. Purpose

This document states the implementation plan for the
aios-roles module.

## 2. Scope

The scope of this document is the aios-roles module.
The module depends on aios-core and aios-models.

## 3. Reference Documents

- aios-roles PRD. See `docs/modules/aios-roles/PRD.md`.
- aios-roles SRS. See `docs/modules/aios-roles/SRS.md`.
- aios-roles TRD. See `docs/modules/aios-roles/TRD.md`.
- aios-roles Application Flow. See
  `docs/modules/aios-roles/APP-FLOW.md`.
- aios-roles Wireframes. See
  `docs/modules/aios-roles/WIREFRAMES.md`.

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

1. Create the file `internal/roles/schema.py`.
2. Create the file `tests/test_schema.py`.
3. Implement the `Role` class.
4. Implement the `OutputFormat` enum.
5. Write the test cases.

The deliverable is the schema with tests.

### Phase 3: Registry

1. Create the file `internal/roles/registry.py`.
2. Create the file `tests/test_registry.py`.
3. Implement the `RoleRegistry` class.
4. Implement the YAML loader.
5. Implement the default roles.
6. Write the test cases.

The deliverable is the registry with tests.

### Phase 4: Resolver

1. Create the file `internal/roles/resolver.py`.
2. Create the file `tests/test_resolver.py`.
3. Implement the `RoleResolver` class.
4. Write the test cases.

The deliverable is the resolver with tests.

### Phase 5: Quality Gates

1. Run the test suite with `go test`.
2. Run the coverage report with `go test -cover`.
3. Run the type check with `go vet ./internal/roles`.
4. Run the linter with `gofmt + go vet check`.
5. Fix every failure.

The deliverable is a green test suite.

## 6. Deliverables

- The source files in `internal/roles/`.
- The test files in `tests/`.
- The `go.mod` file.
- The `README.md` file.
- The default roles file in
  `internal/roles/templates/roles.yaml`.

## 7. Test Plan

### 7.1 Unit Tests

- `tests/test_schema.py` covers
  `internal/roles/schema.py`.
- `tests/test_registry.py` covers
  `internal/roles/registry.py`.
- `tests/test_resolver.py` covers
  `internal/roles/resolver.py`.

### 7.2 Test Cases

The test cases include the following items.

- The schema validates.
- The registry loads the default roles.
- The registry loads from a YAML file.
- The resolver returns the role.
- The resolver raises `UnknownRoleError`.
- The resolver returns the fallback.

## 8. Merge Plan

### 8.1 Pre-Merge Checklist

1. The test suite has a 100 percent pass rate.
2. The coverage report shows 100 percent coverage
   for the public functions.
3. The `go vet` report shows no errors.
4. The `gofmt + go vet` report shows no errors.

### 8.2 Merge Steps

1. Create a branch named `merge/aios-roles`.
2. Move the files from `modules/aios-roles/` to
   the target locations in the main repository.
3. Run the test suite on the main repository.
4. Open a pull request.
5. Merge after the review is approved.

## 9. Risk Register

| Risk | Impact | Probability | Mitigation |
| --- | --- | --- | --- |
| The role definition is invalid. | High | Low | The schema validates the role. |
| The role name is misspelled. | Medium | Medium | The resolver supports a fallback. |
| The role file is missing. | High | Low | The CLI creates the file on `aios init`. |

## 10. Schedule

| Phase | Duration | Dependency |
| --- | --- | --- |
| Phase 1: Project Skeleton | 1 day | nil |
| Phase 2: Schema | 1 day | Phase 1 |
| Phase 3: Registry | 1 day | Phase 2 |
| Phase 4: Resolver | 1 day | Phase 3 |
| Phase 5: Quality Gates | 1 day | All phases |

The total duration is 5 days.
