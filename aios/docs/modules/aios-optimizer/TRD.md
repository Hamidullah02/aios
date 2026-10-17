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

## 4. Definitions

| Term | Definition |
| --- | --- |
| Cascade | A pattern that escalates from weak to strong on low confidence. |
| Heuristic | A rule that does not need a trained model. |
| Quality baseline | The expected quality of a tier. |

## 5. Architecture

The aios-optimizer module has five components.

- `internal/optimizer/registry` — the tiered model
  registry.
- `internal/optimizer/classifier` — the task
  classifier.
- `internal/optimizer/cost_tracker` — the cost
  tracker.
- `internal/optimizer/strategy` — the routing
  strategy.
- `internal/optimizer/feedback` — the quality
  feedback loop.

The five components depend on aios-core and
aios-models. The components depend on each other in
this order. The registry has no dependencies. The
classifier depends on the registry. The cost
tracker depends on the price table. The strategy
depends on the classifier and the cost tracker. The
feedback depends on the strategy and the audit log.

## 6. Components

### 6.1 Tiered Model Registry

The registry holds the list of tiers. The registry
loads the tiers from a YAML file. The registry
validates the tier definitions.

The registry provides the `get_tier` method. The
method returns the tier for a name. The method
raises `UnknownTier` when the name is not in the
registry.

### 6.2 Task Classifier

The classifier accepts a task. The classifier
extracts features from the task. The features
include the prompt length, the tool count, the
keyword matches, and the historical quality.

The classifier supports four strategies.

- `always_strong` — returns the `strong` tier.
- `always_weak` — returns the `weak` tier.
- `heuristic` — uses a rule-based heuristic. The
  heuristic returns `strong` for tasks that
  mention keywords like "refactor", "design",
  "plan", or "explain". The heuristic returns
  `weak` for tasks that mention keywords like
  "format", "lint", or "rename".
- `learned` — uses a trained model. The model is
  a logistic regression on the features.

The `cost_first` mode picks the cheapest tier that
meets the quality bar. The `quality_first` mode
picks the strongest tier that fits the budget.

### 6.3 Cost Tracker

The cost tracker uses the price table from
aios-models. The tracker records the cost for every
call. The tracker computes the cost through the
formula in aios-models.

The tracker enforces the per-task budget. The
tracker returns false when the budget is exhausted.

The tracker provides the `get_summary` method. The
method returns the total cost, the cost per model,
and the cost per task.

### 6.4 Routing Strategy

The strategy uses the classifier and the cost
tracker. The strategy picks the model id for a
task.

The strategy supports three modes.

- `static` — the strategy uses a fixed tier per
  task type. The mapping is in the config.
- `dynamic` — the strategy uses the classifier
  and the cost tracker. The strategy picks the
  tier that the classifier returns. The strategy
  downgrades the tier when the budget is low.
- `cascade` — the strategy tries the weak tier
  first. The strategy escalates to the strong
  tier when the confidence is low or the quality
  is low.

The `budget_aware` flag enables the budget check.
The flag downgrades the tier when the remaining
budget is below the threshold.

### 6.5 Quality Feedback Loop

The feedback loop records the result quality. The
quality is a number between 0 and 1. The feedback
updates the strategy.

The feedback supports the `auto_escalate` flag.
The flag escalates the tier when the quality score
is below the threshold.

The feedback provides the `get_stats` method. The
method returns the average quality score per tier
and the escalation rate.

## 7. Data Structures

### 7.1 Tier

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `name` | string | yes | The name of the tier. |
| `model_id` | string | yes | The model id. |
| `cost_per_1k_input` | float | yes | The cost per 1000 input tokens. |
| `cost_per_1k_output` | float | yes | The cost per 1000 output tokens. |
| `quality_baseline` | float | yes | The expected quality. |
| `context_window` | int | yes | The maximum context window. |

### 7.2 RoutingDecision

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `tier` | string | yes | The selected tier. |
| `model_id` | string | yes | The selected model id. |
| `confidence` | float | yes | The confidence score. |
| `reason` | string | yes | The reason for the decision. |

## 8. Interfaces

### 8.1 RoutingStrategy

```go
type RoutingStrategy:
    """The routing strategy."""

    def __init__(
        self,
        registry: TieredModelRegistry,
        classifier: TaskClassifier,
        cost_tracker: CostTracker,
        mode: RoutingMode = RoutingMode.CASCADE,
        budget_aware: bool = true,
    ) -> nil:
        """Create the routing strategy."""

    def route(self, task: TaskSpec) -> RoutingDecision:
        """Route the task to a model.

        Args:
            task: The task to route.

        Returns:
            The routing decision.
        """
```

## 9. Algorithms

### 9.1 Heuristic Classifier Algorithm

The heuristic classifier algorithm works as follows.

1. Lowercase the prompt.
2. Check for the strong keywords: "refactor",
   "design", "plan", "explain", "review",
   "architect".
3. Check for the weak keywords: "format", "lint",
   "rename", "spell", "typo".
4. Return `strong` when the strong keywords match.
5. Return `weak` when the weak keywords match.
6. Return `medium` when no keyword matches.

### 9.2 Cascade Algorithm

The cascade algorithm works as follows.

1. Try the weak tier.
2. Compute the confidence.
3. When the confidence is below the threshold,
   escalate to the medium tier.
4. When the medium tier confidence is still below
   the threshold, escalate to the strong tier.
5. Return the final decision.

### 9.3 Budget-Aware Algorithm

The budget-aware algorithm works as follows.

1. Compute the remaining budget.
2. When the remaining budget is below 10 percent,
   return the weak tier.
3. When the remaining budget is below 30 percent,
   return the medium tier.
4. Otherwise, return the classifier result.

## 10. Dependencies

### 10.1 Runtime Dependencies

- Go 1.23
- a hand-rolled heuristic (no ML dependency)
- aios-core
- aios-models

### 10.2 Module Dependencies

The aios-optimizer module depends on the following
internal modules.

- aios-core
- aios-models

The aios-optimizer module is depended on by the
following internal modules.

- aios-runtime
- aios-cli
- aios-benchmark

## 11. Deployment

The module compiles into the monorepo under `internal/<module>/`. The
package name is `github.com/<org>/aios/internal/optimizer`. The default
registry is loaded from `./var/optimizer.yaml`.

## 12. Risks and Mitigations

| Risk | Impact | Probability | Mitigation |
| --- | --- | --- | --- |
| The classifier picks the wrong tier. | High | Medium | The cascade mode escalates on low confidence. The feedback loop updates the strategy. |
| The cost tracker misses a call. | High | Low | The tracker wraps the model router. The tracker reconciles at the end. |
| The learned classifier is stale. | Medium | Medium | The classifier is retrained on a regular schedule. The classifier has a test case. |
