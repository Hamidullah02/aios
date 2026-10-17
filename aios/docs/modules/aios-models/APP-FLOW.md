# aios-models — Application Flow

## 1. Purpose

This document describes the application flow for the
aios-models module.

## 2. Scope

The scope of this document is the aios-models module.
The module is a library.

## 3. Reference Documents

- aios-models PRD. See `docs/modules/aios-models/PRD.md`.
- aios-models SRS. See `docs/modules/aios-models/SRS.md`.
- aios-models TRD. See `docs/modules/aios-models/TRD.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Caller | The code that calls the router. |
| Provider | The third-party service that runs the model. |

## 5. User Flow

The aios-models module has no user-facing flow.

## 6. System Flow

### 6.1 Complete Flow

The complete flow runs as follows.

1. The caller calls `router.complete` with the
   prompt, the model id, and the options.
2. The router looks up the capability for the model
   id.
3. The router selects the adapter for the provider.
4. The router calls the adapter.
5. The adapter calls the provider.
6. The provider returns the completion.
7. The adapter returns the completion and the usage.
8. The router computes the cost from the price table.
9. The router returns the completion and the usage.

### 6.2 Stream Flow

The stream flow runs as follows.

1. The caller calls `router.stream` with the prompt,
   the model id, and the options.
2. The router looks up the capability.
3. The router selects the adapter.
4. The router calls the adapter.
5. The adapter opens the stream from the provider.
6. The adapter yields chunks.
7. The router yields chunks to the caller.

### 6.3 Cost Flow

The cost flow runs as follows.

1. The caller calls `price_table.cost` with the model
   id, the input tokens, and the output tokens.
2. The price table looks up the price.
3. The price table computes the cost.
4. The price table returns the cost.

## 7. Error Flows

### 7.1 Unknown Model

The unknown model flow runs as follows.

1. The caller passes an unknown model id.
2. The router raises `UnknownModelError`.
3. The caller catches the error.
4. The caller reports the error.

### 7.2 Price Not Found

The price not found flow runs as follows.

1. The caller passes a model id without a price.
2. The price table raises `PriceNotFoundError`.
3. The caller catches the error.
4. The caller reports the error.

### 7.3 Provider Error

The provider error flow runs as follows.

1. The provider returns an error.
2. The adapter raises `ProviderError`.
3. The caller catches the error.
4. The caller retries with the backoff policy.

## 8. Sequence Diagrams

### 8.1 Complete Sequence

The complete sequence has the steps below.

1. The orchestrator calls `router.complete`.
2. The router looks up the capability.
3. The router selects the adapter.
4. The router calls the adapter.
5. The adapter calls the provider.
6. The provider returns the completion.
7. The adapter returns the completion and the
   usage.
8. The router computes the cost.
9. The router returns the completion and the usage.
