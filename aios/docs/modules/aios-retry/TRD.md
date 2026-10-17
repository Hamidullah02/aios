# aios-retry — Technical Requirements Document

## 1. Purpose

This document states the technical approach for the
aios-retry module.

## 2. Scope

The scope of this document is the aios-retry module.
The module is a Go package.

## 3. Reference Documents

- aios-retry PRD. See `docs/modules/aios-retry/PRD.md`.
- aios-retry SRS. See `docs/modules/aios-retry/SRS.md`.
- aios-core TRD. See `docs/modules/aios-core/TRD.md`.
- aios-tools TRD. See `docs/modules/aios-tools/TRD.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Backoff | A delay between retries. |
| Fix prompt | A prompt that tells the model how to fix the error. |

## 5. Architecture

The aios-retry module has four components.

- `internal/retry/classifier` — the error classifier.
- `internal/retry/policy` — the retry policy.
- `internal/retry/strategy` — the retry strategy.
- `internal/retry/budget` — the retry budget.

The four components depend on aios-core and
aios-tools. The components depend on each other in
this order. The classifier has no dependencies. The
strategy has no dependencies. The budget has no
dependencies. The policy depends on the classifier,
the strategy, and the budget.

## 6. Components

### 6.1 Error Classifier

The classifier accepts an error. The classifier
inspects the error type and the error message. The
classifier returns the error class.

The classifier uses a pattern list. The pattern
list contains regular expressions. The classifier
matches the error message against the patterns. The
classifier returns the type for the first match.

The classifier supports custom patterns. The
operator can add new patterns through the config.

### 6.2 Retry Policy

The policy accepts the tool call, the error, and
the retry count. The policy calls the classifier.
The policy calls the strategy. The policy checks
the budget. The policy returns the decision.

The policy returns `stop` when the error is
`permanent`. The policy returns `stop` when the
budget is exhausted. The policy returns `retry`
when the error is `transient`. The policy returns
`retry_with_fix` when the error is `fixable`.

The fix prompt is generated from the error type.
The fix prompt tells the model how to fix the
error. For example, when the error is
`EditNotUniqueError`, the fix prompt says "add
more context to the old_string".

### 6.3 Retry Strategy

The strategy computes the backoff. The strategy
supports four modes.

- `nil` — returns 0.
- `immediate` — returns 0.
- `linear` — returns `base_delay * retry_count`.
- `exponential` — returns `base_delay * 2^retry_count`.

The default `base_delay` is 1000 milliseconds. The
maximum delay is 30 seconds.

### 6.4 Retry Budget

The budget tracks the retry count for every task.
The budget is a dictionary that maps the task id
to the count. The budget returns the remaining
count.

The budget supports a per-tool limit. The budget
supports a per-error-type limit. The budget
returns the minimum of the three limits.

## 7. Data Structures

### 7.1 RetryDecision

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `action` | string | yes | The action: retry, retry_with_fix, stop. |
| `backoff_ms` | int | no | The backoff in milliseconds. |
| `fix_prompt` | string | no | The fix prompt. |
| `reason` | string | yes | The reason for the decision. |

### 7.2 ErrorClass

| Value | Description |
| --- | --- |
| transient | A temporary error. Retry with backoff. |
| permanent | A non-recoverable error. Stop. |
| fixable | An error that the model can fix. Retry with the fix prompt. |

## 8. Interfaces

### 8.1 RetryPolicy

```go
type RetryPolicy:
    """The retry policy."""

    def __init__(
        self,
        classifier: ErrorClassifier,
        strategy: RetryStrategy,
        budget: RetryBudget,
    ) -> nil:
        """Create the retry policy."""

    def decide(
        self,
        tool_call: ToolCall,
        error: Exception,
        retry_count: int,
    ) -> RetryDecision:
        """Decide on the retry.

        Args:
            tool_call: The tool call.
            error: The error.
            retry_count: The current retry count.

        Returns:
            The retry decision.
        """
```

## 9. Algorithms

### 9.1 Classification Algorithm

The classification algorithm works as follows.

1. Match the error message against the patterns.
2. Return the type for the first match.
3. When no pattern matches, return `permanent`.

### 9.2 Decision Algorithm

The decision algorithm works as follows.

1. Call the classifier.
2. When the type is `permanent`, return `stop`.
3. When the budget is exhausted, return `stop`.
4. When the type is `fixable`, return
   `retry_with_fix` with the fix prompt.
5. When the type is `transient`, return `retry`
   with the backoff.

## 10. Dependencies

### 10.1 Runtime Dependencies

- Go 1.23
- aios-core
- aios-tools

### 10.2 Module Dependencies

The aios-retry module depends on the following
internal modules.

- aios-core
- aios-tools

The aios-retry module is depended on by the
following internal modules.

- aios-runtime
- aios-tools
- aios-edit

## 11. Deployment

The module compiles into the monorepo under `internal/<module>/`. The
package name is `github.com/<org>/aios/internal/retry`. The default patterns
are loaded from the module. The operator can add
custom patterns through `./var/retry.yaml`.

## 12. Risks and Mitigations

| Risk | Impact | Probability | Mitigation |
| --- | --- | --- | --- |
| The classifier misses an error class. | High | Medium | The classifier supports custom patterns. The test suite covers every default pattern. |
| The retry loop runs forever. | High | Low | The budget caps the retries. The policy returns `stop` on budget exhaustion. |
| The fix prompt is wrong. | Medium | Low | The fix prompts are reviewed. The test suite covers every fix prompt. |
