# aios-models — Implementation Plan

## 1. Purpose

This document states the implementation plan for the
aios-models module.

## 2. Scope

The scope of this document is the aios-models module.
The module depends on aios-core.

## 3. Reference Documents

- aios-models PRD. See `docs/modules/aios-models/PRD.md`.
- aios-models SRS. See `docs/modules/aios-models/SRS.md`.
- aios-models TRD. See `docs/modules/aios-models/TRD.md`.
- aios-models Application Flow. See
  `docs/modules/aios-models/APP-FLOW.md`.

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

### Phase 2: Adapter interface

1. Create the file `internal/models/adapter.py`.
2. Create the file `tests/test_adapter.py`.
3. Implement the `ModelAdapter` interface.
4. Implement the `ModelUsage` schema.
5. Implement the `ModelChunk` schema.
6. Write the test cases.

The deliverable is the adapter interface with tests.

### Phase 3: OpenAI Adapter

1. Create the file `internal/models/openai.py`.
2. Create the file `tests/test_openai.py`.
3. Implement the `OpenAIAdapter` class.
4. Use a recorded response for the test.
5. Write the test cases.

The deliverable is the OpenAI adapter with tests.

### Phase 4: Anthropic Adapter

1. Create the file `internal/models/anthropic.py`.
2. Create the file `tests/test_anthropic.py`.
3. Implement the `AnthropicAdapter` class.
4. Use a recorded response for the test.
5. Write the test cases.

The deliverable is the Anthropic adapter with tests.

### Phase 5: Price Table

1. Create the file `internal/models/price.py`.
2. Create the file `tests/test_price.py`.
3. Implement the `Price` schema.
4. Implement the `PriceTable` class.
5. Implement the YAML loader.
6. Implement the override.
7. Write the test cases.

The deliverable is the price table with tests.

### Phase 6: Capability Registry

1. Create the file `internal/models/capability.py`.
2. Create the file `tests/test_capability.py`.
3. Implement the `ModelCapability` schema.
4. Implement the `CapabilityRegistry` class.
5. Write the test cases.

The deliverable is the registry with tests.

### Phase 7: Router

1. Create the file `internal/models/router.py`.
2. Create the file `tests/test_router.py`.
3. Implement the `ModelRouter` class.
4. Implement the `complete` method.
5. Implement the `stream` method.
6. Write the test cases.

The deliverable is the router with tests.

### Phase 8: Quality Gates

1. Run the test suite with `go test`.
2. Run the coverage report with `go test -cover`.
3. Run the type check with `go vet ./internal/models`.
4. Run the linter with `gofmt + go vet check`.
5. Fix every failure.

The deliverable is a green test suite.

## 6. Deliverables

- The source files in `internal/models/`.
- The test files in `tests/`.
- The `go.mod` file.
- The `README.md` file.
- Sample price and capability files in `examples/`.

## 7. Test Plan

### 7.1 Unit Tests

- `tests/test_adapter.py` covers
  `internal/models/adapter.py`.
- `tests/test_openai.py` covers
  `internal/models/openai.py`.
- `tests/test_anthropic.py` covers
  `internal/models/anthropic.py`.
- `tests/test_price.py` covers
  `internal/models/price.py`.
- `tests/test_capability.py` covers
  `internal/models/capability.py`.
- `tests/test_router.py` covers
  `internal/models/router.py`.

### 7.2 Test Cases

The test cases include the following items.

- The adapter interface compiles.
- The OpenAI adapter maps the prompt.
- The OpenAI adapter maps the response.
- The Anthropic adapter maps the prompt.
- The Anthropic adapter maps the response.
- The price table loads from YAML.
- The price table loads the override.
- The price table computes the cost.
- The price table raises `PriceNotFoundError`.
- The capability registry adds a capability.
- The capability registry returns a capability.
- The capability registry lists the models.
- The router returns the correct adapter.
- The router completes a prompt.
- The router streams a prompt.

## 8. Merge Plan

### 8.1 Pre-Merge Checklist

1. The test suite has a 100 percent pass rate.
2. The coverage report shows 100 percent coverage for
   the public functions.
3. The `go vet` report shows no errors.
4. The `gofmt + go vet` report shows no errors.

### 8.2 Merge Steps

1. Create a branch named `merge/aios-models`.
2. Move the files from `modules/aios-models/` to the
   target locations in the main repository.
3. Run the test suite on the main repository.
4. Open a pull request.
5. Merge after the review is approved.

## 9. Risk Register

| Risk | Impact | Probability | Mitigation |
| --- | --- | --- | --- |
| The provider changes the API. | High | Medium | The adapter is the only component that calls the provider. The adapter is versioned. |
| The price file has a missing model. | High | Medium | The price table raises `PriceNotFoundError`. The CI checks the price file against the model list. |
| The router is slow. | Medium | Low | The router caches the adapter. The router uses a single connection pool. |

## 10. Schedule

| Phase | Duration | Dependency |
| --- | --- | --- |
| Phase 1: Project Skeleton | 1 day | nil |
| Phase 2: Adapter interface | 1 day | Phase 1 |
| Phase 3: OpenAI Adapter | 2 days | Phase 2 |
| Phase 4: Anthropic Adapter | 2 days | Phase 2 |
| Phase 5: Price Table | 1 day | Phase 1 |
| Phase 6: Capability Registry | 1 day | Phase 1 |
| Phase 7: Router | 2 days | Phase 3, Phase 4, Phase 5, Phase 6 |
| Phase 8: Quality Gates | 1 day | All phases |

The total duration is 11 days.
