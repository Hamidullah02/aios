# aios-models — Software Requirements Specification

## 1. Purpose

This document states the software requirements for the
aios-models module.

## 2. Scope

The scope of this document is the aios-models module.
The module is a library.

## 3. Reference Documents

- aios-models PRD. See `docs/modules/aios-models/PRD.md`.
- aios-core SRS. See `docs/modules/aios-core/SRS.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Adapter | A typed component that calls one model provider. |
| Router | A typed component that selects the adapter. |
| Capability | A typed record of the model features. |

## 5. Functional Requirements

### 5.1 Model Adapter interface

**REQ-MOD-ADP-001** The module shall define the
`ModelAdapter` interface. The interface shall declare
the `complete` method. The method shall accept the
prompt, the model id, and the options.

**REQ-MOD-ADP-002** The interface shall declare the
`stream` method. The method shall return an async
iterator of token chunks.

**REQ-MOD-ADP-003** The module shall provide the
`OpenAIAdapter` class. The type shall implement the
`ModelAdapter` interface for the OpenAI provider.

**REQ-MOD-ADP-004** The module shall provide the
`AnthropicAdapter` class. The type shall implement
the `ModelAdapter` interface for the Anthropic
provider.

### 5.2 Router

**REQ-MOD-RTR-001** The module shall provide the
`ModelRouter` class. The type shall accept the
adapters and the capability registry.

**REQ-MOD-RTR-002** The router shall select the
adapter for a model id. The router shall raise an
`UnknownModelError` when the model id is not in the
registry.

**REQ-MOD-RTR-003** The router shall provide the
`complete` method. The method shall accept the
prompt, the model id, and the options. The method
shall return the completion and the usage.

**REQ-MOD-RTR-004** The router shall provide the
`stream` method. The method shall return an async
iterator of token chunks and the usage.

### 5.3 Price Table

**REQ-MOD-PRC-001** The module shall provide the
`PriceTable` class. The type shall accept the price
data.

**REQ-MOD-PRC-002** The price table shall compute
the cost for a call. The method shall accept the
model id, the input tokens, and the output tokens.
The method shall return the cost in USD.

**REQ-MOD-PRC-003** The price table shall load the
price data from a YAML file. The default path is
`./var/prices.yaml`. The path is configurable.

**REQ-MOD-PRC-004** The price table shall support
an override file. The override file replaces the
prices for the models in the override.

### 5.4 Capability Registry

**REQ-MOD-CAP-001** The module shall provide the
`CapabilityRegistry` class. The type shall accept
the list of capabilities.

**REQ-MOD-CAP-002** The registry shall provide the
`get` method. The method shall accept the model id.
The method shall return the capability.

**REQ-MOD-CAP-003** The registry shall provide the
`set` method. The method shall accept the capability.
The method shall update the registry.

**REQ-MOD-CAP-004** The registry shall provide the
`list` method. The method shall return the list of
model ids.

## 6. Non-Functional Requirements

**REQ-MOD-NFR-001** The module shall run on Go 1.23
3.12.

**REQ-MOD-NFR-002** The module shall use async I/O
through `goroutines + channels (stdlib) + golang.org/x/sync/errgroup` and `net/http (stdlib)`.

**REQ-MOD-NFR-003** The module shall have 100 percent
coverage for the public functions.

## 7. Interface Requirements

**REQ-MOD-IFC-001** The module shall export the
following items from `aios.models`.

- `ModelAdapter`
- `ModelRouter`
- `PriceTable`
- `CapabilityRegistry`
- `OpenAIAdapter`
- `AnthropicAdapter`
- `ModelUsage`
- `ModelChunk`

## 8. Data Requirements

**REQ-MOD-DAT-001** The price file shall be a YAML
file. The structure is below.

```yaml
prices:
  gpt-4o:
    input_per_1k: 0.005
    output_per_1k: 0.015
  claude-sonnet-4-5:
    input_per_1k: 0.003
    output_per_1k: 0.015
```

**REQ-MOD-DAT-002** The capability file shall be a
YAML file. The structure is below.

```yaml
capabilities:
  gpt-4o:
    context_window: 128000
    supports_tools: true
    supports_vision: true
  claude-sonnet-4-5:
    context_window: 200000
    supports_tools: true
    supports_vision: true
```

## 9. Error Handling

**REQ-MOD-ERR-001** The module shall raise
`UnknownModelError` when the model id is not in the
registry.

**REQ-MOD-ERR-002** The module shall raise
`PriceNotFoundError` when the model id is not in the
price table.

**REQ-MOD-ERR-003** The module shall raise
`ProviderError` when the provider returns an error.
The error shall include the provider code and the
message.

## 10. Acceptance Criteria

1. Every REQ in this document has a test case.
2. The test suite has a 100 percent pass rate.
3. The test suite has 100 percent coverage for the
   public functions.
4. The router returns the correct adapter.
5. The price table returns the correct cost.
6. The capability registry tracks the model features.
