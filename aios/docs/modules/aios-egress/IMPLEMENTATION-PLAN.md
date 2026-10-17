# aios-egress — Implementation Plan

## 1. Purpose

This document states the implementation plan for the
aios-egress module.

## 2. Scope

The scope of this document is the aios-egress module.
The module depends on aios-core.

## 3. Reference Documents

- aios-egress PRD. See `docs/modules/aios-egress/PRD.md`.
- aios-egress SRS. See `docs/modules/aios-egress/SRS.md`.
- aios-egress TRD. See `docs/modules/aios-egress/TRD.md`.
- aios-egress Application Flow. See
  `docs/modules/aios-egress/APP-FLOW.md`.

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

### Phase 2: Allow List

1. Create the file `internal/egress/allow.py`.
2. Create the file `tests/test_allow.py`.
3. Implement the `AllowList` class.
4. Write the test cases.

The deliverable is the allow list with tests.

### Phase 3: Header Rules

1. Create the file `internal/egress/headers.py`.
2. Create the file `tests/test_headers.py`.
3. Implement the `HeaderRules` class.
4. Write the test cases.

The deliverable is the header rules with tests.

### Phase 4: Proxy

1. Create the file `internal/egress/proxy.py`.
2. Create the file `tests/test_proxy.py`.
3. Implement the `EgressProxy` class.
4. Write the test cases.

The deliverable is the proxy with tests.

### Phase 5: Quality Gates

1. Run the test suite with `go test`.
2. Run the coverage report with `go test -cover`.
3. Run the type check with `go vet ./internal/egress`.
4. Run the linter with `gofmt + go vet check`.
5. Fix every failure.

The deliverable is a green test suite.

## 6. Deliverables

- The source files in `internal/egress/`.
- The test files in `tests/`.
- The `go.mod` file.
- The `README.md` file.
- A sample egress file in `examples/`.

## 7. Test Plan

### 7.1 Unit Tests

- `tests/test_allow.py` covers
  `internal/egress/allow.py`.
- `tests/test_headers.py` covers
  `internal/egress/headers.py`.
- `tests/test_proxy.py` covers
  `internal/egress/proxy.py`.

### 7.2 Test Cases

The test cases include the following items.

- The allow list matches an exact host.
- The allow list matches a glob host.
- The allow list denies an unknown host.
- The header rules remove a header.
- The header rules add a header.
- The proxy forwards a request.
- The proxy denies a request.
- The proxy rewrites the headers.
- The proxy writes the audit row.

## 8. Merge Plan

### 8.1 Pre-Merge Checklist

1. The test suite has a 100 percent pass rate.
2. The coverage report shows 100 percent coverage for
   the public functions.
3. The `go vet` report shows no errors.
4. The `gofmt + go vet` report shows no errors.

### 8.2 Merge Steps

1. Create a branch named `merge/aios-egress`.
2. Move the files from `modules/aios-egress/` to the
   target locations in the main repository.
3. Run the test suite on the main repository.
4. Open a pull request.
5. Merge after the review is approved.

## 9. Risk Register

| Risk | Impact | Probability | Mitigation |
| --- | --- | --- | --- |
| The proxy leaks the API key. | High | Low | The proxy removes the `Authorization` header. The proxy logs the request. |
| The proxy is slow. | Medium | Low | The proxy uses `aiohttp`. The proxy uses a connection pool. |
| The proxy has a bypass. | High | Low | The proxy checks the allow list on every request. The proxy supports the deny list. |

## 10. Schedule

| Phase | Duration | Dependency |
| --- | --- | --- |
| Phase 1: Project Skeleton | 1 day | nil |
| Phase 2: Allow List | 1 day | Phase 1 |
| Phase 3: Header Rules | 1 day | Phase 1 |
| Phase 4: Proxy | 2 days | Phase 2, Phase 3 |
| Phase 5: Quality Gates | 1 day | All phases |

The total duration is 6 days.
