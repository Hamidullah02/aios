# aios-models — Product Requirements Document

## 1. Purpose

This document states the product requirements for the
aios-models module. The module is the model adapter
layer. The module connects to model providers, routes
requests across models, and tracks the cost through
the price table.

## 2. Scope

The aios-models module is in scope for this document.

- The model adapter protocol.
- The model router.
- The price table.
- The model capability registry.

The aios-models module is not in scope for this
document.

- The model providers. The providers are third-party
  libraries.
- The prompt templates. The templates are in
  aios-planner.

## 3. Reference Documents

- Project design document, sections 4, 6, 9, 15.
- aios-core PRD. See `docs/modules/aios-core/PRD.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Adapter | A component that talks to one model provider. |
| Router | A component that selects a model for a request. |
| Price table | A file that maps a model to a cost per token. |
| Capability | A property of a model: context window, cost, features. |
| Provider | A third-party service that runs the model. |


## 5. Determinism

The whole point of aios is to make complex coding and large systems deterministic. The determinism stance of this module is **Bounded by `temperature=0`**.

Model output is the only source of non-determinism. The router, the price table, and the adapter interface are deterministic.

The module provides the following determinism guarantees.

- Router is a pure function of the request and the model registry.

- Price table is content addressed.

- Model calls default to `temperature=0`.
## 6. User Personas

The user persona is a Go developer with experience
in HTTP clients and async programming.

## 7. User Stories

A developer wants to call a model. The developer
passes the model id and the prompt to the router.
The router selects the adapter. The adapter calls the
provider. The adapter returns the result.

A developer wants to know the cost of a call. The
developer passes the model id and the token count to
the price table. The price table returns the cost.

A developer wants to register a new model. The
developer adds the model to the capability
registry. The developer adds the price to the price
table.

## 8. Functional Requirements Summary

The aios-models module provides the following
functions.

- The model adapter protocol.
- The model router that selects the adapter.
- The price table that computes the cost.
- The capability registry that tracks the model
  features.

## 9. Non-Functional Requirements Summary

- The module runs on Go 1.23.
- The module uses goroutines for async I/O.
- The module depends only on aios-core, the net/http (stdlib)
  library, and the Go standard library.
- The module has 100 percent test coverage for the
  public functions.

## 10. Success Criteria

- The router returns the correct adapter.
- The price table returns the correct cost.
- The capability registry tracks the model features.
- The test suite has a 100 percent pass rate.

## 11. Out of Scope

- The user interface for the price table.
- The remote access to the price table.
- The model fine-tuning.

## 12. Open Questions

- Does the router support model fallbacks? The current
  design supports the primary model only.
- Does the price table support tiered pricing? The
  current design supports a flat price per token.
