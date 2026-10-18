# aios-optimizer — Technical Requirements Document

## 1. Purpose

This document states the technical approach for the
aios-optimizer module.

## 2. Scope

The scope of this document is the aios-optimizer
module. The module is a Go package.

## 3. Reference Documents

- aios-optimizer PRD. See
  `docs/modules/aios-optimizer/PRD.md`.
- aios-optimizer SRS. See
  `docs/modules/aios-optimizer/SRS.md`.
- aios-core TRD. See `docs/modules/aios-core/TRD.md`.
- aios-models TRD. See `docs/modules/aios-models/TRD.md`.
- aios-verify TRD. See `docs/modules/aios-verify/TRD.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Fingerprint | A SHA-256 hash. |
| Pure function | A function whose return value is determined only by its input values, with no side effects. |
| Atomic write | A file write that is either fully visible to readers or fully absent. |

## 5. Architecture

The aios-optimizer module has four components.

- `internal/optimizer/registry` — the
  class-based model registry.
- `internal/optimizer/fingerprint` — the call
  fingerprint.
- `internal/optimizer/table` — the cost/quality
  table.
- `internal/optimizer/router` — the routing
  decision.

The four components depend on aios-core,
aios-models, and aios-verify. The components
depend on each other in this order. The registry
has no dependencies. The fingerprint has no
dependencies. The table depends on the registry.
The router depends on the registry, the table, and
the verify result type.

## 6. Components

### 6.1 Class-Based Model Registry

The registry holds the list of classes. The
registry loads the classes from
`var/optimizer.yaml`. The registry validates
the class definitions. The registry provides
the `Get` method. The method returns the
class for a name. The method returns
`ErrUnknownClass` when the name is not in the
registry.

### 6.2 Call Fingerprint

The fingerprint is a SHA-256 hash. The fingerprint
is computed from the (prompt shape, tool set,
role, context length bucket) tuple. The function
is a pure function. The function has no side
effects.

The prompt shape is computed by replacing the
variable parts of the prompt with their type tags.
A string is replaced by `str`. A number is
replaced by `num`. A path is replaced by `path`. A
code block is replaced by `code`. The result is a
prompt with no variable data.

The tool set is the sorted list of tool names.

The role is the role name.

The context length bucket is the input token count
rounded down to the nearest power of 2. The
buckets are 256, 512, 1024, 2048, 4096, 8192,
16384, 32768, 65536, 131072.

### 6.3 Cost/Quality Table

The table is a JSON file. The file is at
`var/optimizer.json`. The file is content
addressed. The file id is the SHA-256 of the file
payload.

The table provides the `Get`, `Record`, and
`Reset` methods. The `Record` method writes the
file. The file write is atomic through a
`rename` syscall. The file is written to a
temporary file. The temporary file is renamed to
the final path.

The in-memory table is a map. The key is a
`fingerprint:class` string. The value is a
`CostQualityRow` struct.

The pass rate is computed lazily. The pass rate is
the fraction of `pass_count` to `call_count`. The
pass rate is `0.0` when the call count is 0.

### 6.4 Router

The router is a pure function. The router accepts
a fingerprint, an override, a budget, the
registry, and the table. The router returns a
`RoutingDecision`.

The router algorithm is listed below.

1. When the override is `off`, return the
   strong class with `source=optimizer_off`.
2. When the override is `strong`, `mid`, or
   `cheap`, return the override class with
   `source=override`.
3. Look up the fingerprint in the table. When
   the fingerprint is in the table for two or
   more classes, return the class with the
   highest pass rate per unit cost. The cost
   is the empirical cost, not the list price.
4. When the fingerprint is not in the table,
   use the default rule. The default rule is
   `cost_first`. The default rule returns the
   cheapest class.
5. Enforce the budget. When the budget is
   exhausted, return the cheapest class that
   fits the remaining budget. When no class
   fits, return `ErrBudgetExhausted`.

The router does not call the model. The router
does not call the verify layer. The router is a
pure function of its inputs.

## 7. Data Structures

### 7.1 Class

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `Name` | string | yes | The name of the class. |
| `ModelID` | string | yes | The model id. |
| `CostPer1kInput` | float64 | yes | The list cost per 1000 input tokens. |
| `CostPer1kOutput` | float64 | yes | The list cost per 1000 output tokens. |
| `ContextWindow` | int | yes | The maximum context window. |

### 7.2 Fingerprint

The `Fingerprint` is a string. The string is a
SHA-256 hash encoded as a lowercase hex string of
64 characters.

### 7.3 RoutingDecision

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `Class` | string | yes | The selected class. |
| `ModelID` | string | yes | The selected model id. |
| `Reason` | string | yes | The human-readable reason. |
| `Source` | string | yes | The source of the decision. |

### 7.4 CostQualityRow

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `CallCount` | int | yes | The number of calls. |
| `PassCount` | int | yes | The number of verify-passed calls. |
| `TotalCostUSD` | float64 | yes | The total empirical cost in USD. |
| `FirstSeen` | string | yes | The ISO 8601 timestamp of the first call. |
| `LastSeen` | string | yes | The ISO 8601 timestamp of the last call. |

## 8. Interfaces

### 8.1 Router

```go
// Router picks the model class for a call.
type Router struct {
    registry *ClassBasedModelRegistry
    table    *CostQualityTable
}

// Route returns the routing decision for a call.
//
// The router is a pure function of its inputs.
// Two calls with the same fingerprint, the same
// override, the same budget, and the same table
// return the same decision.
func (r *Router) Route(
    fingerprint Fingerprint,
    override Override,
    budgetUSD float64,
) (RoutingDecision, error)

// Override is the per-call override.
type Override string

const (
    OverrideAuto  Override = ""
    OverrideStrong Override = "strong"
    OverrideMedium Override = "medium"
    OverrideWeak   Override = "weak"
    OverrideOff    Override = "off"
)
```

### 8.2 Fingerprint

```go
// NewFingerprint computes the fingerprint for a call.
//
// The function is a pure function. The function
// has no side effects. Two calls with the same
// inputs return the same fingerprint.
func NewFingerprint(
    promptShape string,
    tools []string,
    role string,
    inputTokens int,
) Fingerprint
```

### 8.3 CostQualityTable

```go
// CostQualityTable is the empirical pass rate
// and cost per (fingerprint, class) pair.
type CostQualityTable struct {
    path  string
    rows  map[string]CostQualityRow
}

// Get returns the row for the (fingerprint,
// class) pair. The function returns (nil,
// false) when the pair is not in the table.
func (t *CostQualityTable) Get(
    fp Fingerprint,
    class string,
) (CostQualityRow, bool)

// Record updates the table for the
// (fingerprint, class) pair. The function is
// not safe for concurrent use. The function
// writes the table to the file atomically.
func (t *CostQualityTable) Record(
    fp Fingerprint,
    class string,
    costUSD float64,
    verifyPassed bool,
) error

// Reset clears the table. The function writes
// the empty table to the file atomically.
func (t *CostQualityTable) Reset() error
```

## 9. Algorithms

### 9.1 Fingerprint Algorithm

The fingerprint algorithm works as follows.

1. Compute the prompt shape by replacing the
   variable parts of the prompt with their type
   tags.
2. Sort the tool names.
3. Concatenate the inputs with a separator.
4. Hash the concatenation with SHA-256.
5. Encode the hash as a lowercase hex string.

### 9.2 Routing Algorithm

The routing algorithm works as follows.

1. When the override is `off`, return the
   strong class with `source=optimizer_off`.
2. When the override is `strong`, `mid`, or
   `cheap`, return the override class with
   `source=override`.
3. Look up the fingerprint in the table. When
   the fingerprint is in the table for two or
   more classes, compute the score for each
   class. The score is the pass rate divided
   by the average empirical cost. Return the
   class with the highest score.
4. When the fingerprint is not in the table,
   use the default rule. The default rule is
   `cost_first`. The default rule returns the
   cheapest class.
5. Enforce the budget. When the budget is
   exhausted, return the cheapest class that
   fits the remaining budget. When no class
   fits, return `ErrBudgetExhausted`.

### 9.3 Atomic Write Algorithm

The atomic write algorithm works as follows.

1. Write the table to a temporary file. The
   temporary file is at `<path>.tmp`.
2. Rename the temporary file to the final path.
   The rename is atomic on POSIX file systems.

## 10. Dependencies

### 10.1 Runtime Dependencies

- Go 1.23
- aios-core
- aios-models
- aios-verify

### 10.2 Module Dependencies

The aios-optimizer module depends on the
following internal modules.

- aios-core
- aios-models
- aios-verify

The aios-optimizer module is depended on by the
following internal modules.

- aios-runtime
- aios-cli
- aios-benchmark

## 11. Deployment

The module is installed as a Go package. The
package path is
`github.com/<org>/aios/internal/optimizer`. The
default registry is loaded from
`var/optimizer.yaml`.

## 12. Risks and Mitigations

| Risk | Impact | Probability | Mitigation |
| --- | --- | --- | --- |
| The cost/quality table is stale. | High | Medium | The table is updated on every call. The table has a `last_seen` timestamp. The developer can clear the table with `aios stats --reset`. |
| The fingerprint is too coarse. | Medium | Medium | The fingerprint includes the prompt shape, the tool set, the role, and the context length bucket. The developer can override per call with `--class=strong` or `--class=cheap`. |
| The fingerprint is too fine. | Low | High | The table grows without bound. The table is pruned by the developer with `aios stats --reset`. |
| The pass rate is misleading. | High | Low | The pass rate is the empirical verify result, not the model self-evaluation. The developer can inspect the table with `aios stats`. |
| The budget is exceeded. | High | Low | The router enforces the budget. The router returns `ErrBudgetExhausted` when the budget is exhausted. |
| The override is wrong. | Low | Medium | The override is recorded in the audit log. The developer can see the override count with `aios stats`. |
