# aios-models — Technical Requirements Document

## 1. Purpose

This document states the technical approach for the
aios-models module.

## 2. Scope

The scope of this document is the aios-models module.
The module is a Go package.

## 3. Reference Documents

- aios-models PRD. See `docs/modules/aios-models/PRD.md`.
- aios-models SRS. See `docs/modules/aios-models/SRS.md`.
- aios-core TRD. See `docs/modules/aios-core/TRD.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| net/http (stdlib) | The Go standard-library HTTP client. |
| OpenAI | A model provider. |
| Anthropic | A model provider. |

## 5. Architecture

The aios-models module has four components.

- `aios.models.adapter` — the adapter protocol.
- `internal/models/router` — the router.
- `aios.models.price` — the price table.
- `aios.models.capability` — the capability registry.

The four components depend on aios-core. The
components depend on each other in this order. The
adapter has no dependencies. The router depends on
the adapter and the capability registry. The price
table is independent.

## 6. Components

### 6.1 Adapter

The adapter component defines the protocol. The
protocol has the `complete` method and the `stream`
method. The methods accept the prompt, the model id,
and the options.

The OpenAI adapter uses the `net/http (stdlib)` library to call
the OpenAI API. The adapter maps the prompt to the
OpenAI request format. The adapter maps the OpenAI
response to the `ModelUsage` and the completion.

The Anthropic adapter uses the `net/http (stdlib)` library to
call the Anthropic API. The adapter maps the prompt
to the Anthropic request format. The adapter maps
the Anthropic response to the `ModelUsage` and the
completion.

### 6.2 Router

The router component selects the adapter. The router
uses the capability registry to look up the provider
for the model id. The router uses the adapter map to
find the adapter class. The router instantiates the
adapter and returns it.

### 6.3 Price Table

The price table component loads the price data from
a YAML file. The component supports an override file.
The override file replaces the prices for the models
in the override.

The component computes the cost through the formula
below.

```
cost = (input_tokens / 1000) * input_per_1k
     + (output_tokens / 1000) * output_per_1k
```

### 6.4 Capability Registry

The capability registry component stores the
capabilities in a dictionary. The dictionary maps
the model id to the capability. The component
supports the `get`, `set`, and `list` methods.

## 7. Data Structures

### 7.1 ModelUsage

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `model` | string | yes | The model id. |
| `input_tokens` | int | yes | The number of input tokens. |
| `output_tokens` | int | yes | The number of output tokens. |
| `cost_usd` | float | yes | The cost in USD. |

### 7.2 ModelCapability

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `model` | string | yes | The model id. |
| `provider` | string | yes | The provider id. |
| `context_window` | int | yes | The maximum context window. |
| `supports_tools` | bool | yes | Whether the model supports tools. |
| `supports_vision` | bool | no | Whether the model supports vision. |

### 7.3 Price

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `model` | string | yes | The model id. |
| `input_per_1k` | float | yes | The cost per 1000 input tokens. |
| `output_per_1k` | float | yes | The cost per 1000 output tokens. |

## 8. Interfaces

### 8.1 ModelRouter

```go
type ModelRouter:
    """The model router."""

    def __init__(
        self,
        adapters: dict[str, ModelAdapter],
        capabilities: CapabilityRegistry,
        price_table: PriceTable,
    ) -> nil:
        """Create the model router."""

    func complete(
        self,
        prompt: str,
        model: str,
        options: ModelOptions,
    ) -> tuple[str, ModelUsage]:
        """Complete the prompt with the model.

        Args:
            prompt: The prompt to complete.
            model: The model id.
            options: The options for the call.

        Returns:
            The completion and the usage.
        """

    func stream(
        self,
        prompt: str,
        model: str,
        options: ModelOptions,
    ) -> AsyncIterator[ModelChunk]:
        """Stream the completion of the prompt.

        Args:
            prompt: The prompt to complete.
            model: The model id.
            options: The options for the call.

        Yields:
            The chunks of the completion.
        """
```

### 8.2 PriceTable

```go
type PriceTable:
    """The price table."""

    def __init__(self, prices: dict[str, Price], overrides: dict[str, Price] | nil = nil) -> nil:
        """Create the price table."""

    def cost(self, model: str, input_tokens: int, output_tokens: int) -> float:
        """Return the cost for a call.

        Args:
            model: The model id.
            input_tokens: The number of input tokens.
            output_tokens: The number of output tokens.

        Returns:
            The cost in USD.
        """
```

## 9. Algorithms

### 9.1 Cost Computation

The cost computation algorithm works as follows.

1. Look up the price for the model in the override
   table. When the price exists, use the override.
2. When the override is empty, look up the price in
   the base table. When the price is missing, raise
   `PriceNotFoundError`.
3. Compute the cost through the formula in section
   6.3.

## 10. Dependencies

### 10.1 Runtime Dependencies

- Go 1.23
- net/http (stdlib)
- gopkg.in/yaml.v3
- aios-core

### 10.2 Module Dependencies

The aios-models module depends on the following
internal modules.

- aios-core

The aios-models module is depended on by the following
internal modules.

- aios-runtime
- aios-planner
- aios-benchmark

## 11. Deployment

The module compiles into the monorepo under `internal/<module>/`. The
package name is `github.com/<org>/aios/internal/models`. The price file and
the capability file are loaded from `./var/` by
default.

## 12. Risks and Mitigations

| Risk | Impact | Probability | Mitigation |
| --- | --- | --- | --- |
| The provider changes the API. | High | Medium | The adapter is the only component that calls the provider. The adapter is versioned. |
| The price changes. | High | High | The price table loads from a file. The file is updated through a release. |
| The router returns the wrong adapter. | High | Low | The router validates the model id. The router has a test case for every model. |
