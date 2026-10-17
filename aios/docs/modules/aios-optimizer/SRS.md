# aios-optimizer — Software Requirements Specification

## 1. Purpose

This document states the software requirements for the
aios-optimizer module.

## 2. Scope

The scope of this document is the aios-optimizer
module. The module is a library.

## 3. Reference Documents

- aios-optimizer PRD. See
  `docs/modules/aios-optimizer/PRD.md`.
- aios-core SRS. See `docs/modules/aios-core/SRS.md`.
- aios-models SRS. See `docs/modules/aios-models/SRS.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Tier | A typed object that holds a model tier. |
| Strategy | A typed object that holds the routing strategy. |
| Quality score | A typed number between 0 and 1. |

## 5. Functional Requirements

### 5.1 Tiered Model Registry

**REQ-OPT-REG-001** The module shall provide the
`TieredModelRegistry` class. The type shall accept
the list of tiers.

**REQ-OPT-REG-002** A tier shall have a `name`, a
`model_id`, a `cost_per_1k_input`, a
`cost_per_1k_output`, and a `quality_baseline`.

**REQ-OPT-REG-003** The supported tiers are
`strong`, `medium`, and `weak`. The default
registry includes one model per tier.

**REQ-OPT-REG-004** The registry shall support
custom tiers. The registry shall validate the
tier definitions.

### 5.2 Task Classifier

**REQ-OPT-CLS-001** The module shall provide the
`TaskClassifier` class. The type shall accept the
feature extractor and the classifier model.

**REQ-OPT-CLS-002** The classifier shall accept a
task. The task includes the `prompt`, the `tools`,
and the `context`.

**REQ-OPT-CLS-003** The classifier shall return the
predicted tier. The classifier shall return the
confidence score.

**REQ-OPT-CLS-004** The classifier shall support
the following strategies.

- `always_strong` — the classifier always returns
  `strong`.
- `always_weak` — the classifier always returns
  `weak`.
- `heuristic` — the classifier uses a rule-based
  heuristic.
- `learned` — the classifier uses a trained
  model.

**REQ-OPT-CLS-005** The classifier shall support
the `cost_first` strategy. The classifier shall
return the cheapest tier that fits the budget.

**REQ-OPT-CLS-006** The classifier shall support
the `quality_first` strategy. The classifier shall
return the strongest tier that fits the budget.

### 5.3 Cost Tracker

**REQ-OPT-CST-001** The module shall provide the
`CostTracker` class. The type shall accept the
price table and the budget.

**REQ-OPT-CST-002** The tracker shall record the
cost for every call. The tracker shall record the
model id, the input tokens, the output tokens, and
the cost.

**REQ-OPT-CST-003** The tracker shall enforce the
per-task budget. The tracker shall return false
when the budget is exhausted.

**REQ-OPT-CST-004** The tracker shall enforce the
per-call budget. The tracker shall return false
when the per-call budget is exhausted.

**REQ-OPT-CST-005** The tracker shall provide the
`get_summary` method. The method shall return the
total cost, the cost per model, and the cost per
task.

### 5.4 Routing Strategy

**REQ-OPT-RTE-001** The module shall provide the
`RoutingStrategy` class. The type shall accept the
classifier and the cost tracker.

**REQ-OPT-RTE-002** The strategy shall accept a
task. The strategy shall return the model id.

**REQ-OPT-RTE-003** The strategy shall support the
following modes.

- `static` — the strategy uses a fixed tier per
  task type.
- `dynamic` — the strategy uses the classifier
  and the cost tracker.
- `cascade` — the strategy tries the weak tier
  first, then escalates to the strong tier on
  low confidence.

**REQ-OPT-RTE-004** The strategy shall support
the `budget_aware` flag. The strategy shall
downgrade the tier when the budget is low.

### 5.5 Quality Feedback Loop

**REQ-OPT-FBK-001** The module shall provide the
`QualityFeedback` class. The type shall accept the
strategy and the audit log.

**REQ-OPT-FBK-002** The feedback shall accept a
result. The result includes the `task`, the
`model_id`, the `output`, and the `quality_score`.

**REQ-OPT-FBK-003** The feedback shall record the
result. The feedback shall update the strategy.

**REQ-OPT-FBK-004** The feedback shall support the
`auto_escalate` flag. The feedback shall
automatically escalate the tier when the quality
score is below the threshold.

**REQ-OPT-FBK-005** The feedback shall provide the
`get_stats` method. The method shall return the
average quality score per tier and the escalation
rate.

## 6. Non-Functional Requirements

**REQ-OPT-NFR-001** The module shall run on Go 1.23
3.12.

**REQ-OPT-NFR-002** The module shall have 100 percent
coverage for the public functions.

**REQ-OPT-NFR-003** The router decision shall have
a maximum latency of 5 milliseconds.

## 7. Interface Requirements

**REQ-OPT-IFC-001** The module shall export the
following items from `aios.optimizer`.

- `TieredModelRegistry`
- `TaskClassifier`
- `CostTracker`
- `RoutingStrategy`
- `QualityFeedback`
- `Tier`
- `RoutingMode`

## 8. Data Requirements

**REQ-OPT-DAT-001** The `Tier` schema shall include
the fields below.

- `name` (string)
- `model_id` (string)
- `cost_per_1k_input` (float)
- `cost_per_1k_output` (float)
- `quality_baseline` (float, 0 to 1)
- `context_window` (int)

**REQ-OPT-DAT-002** The default registry shall be
loaded from `./var/optimizer.yaml`. The file
lists the tiers and the strategy.

## 9. Error Handling

**REQ-OPT-ERR-001** The module shall raise
`BudgetExhausted` when the cost tracker exhausts
the budget.

**REQ-OPT-ERR-002** The module shall raise
`UnknownTier` when the tier is not in the
registry.

## 10. Acceptance Criteria

1. Every REQ in this document has a test case.
2. The test suite has a 100 percent pass rate.
3. The test suite has 100 percent coverage for the
   public functions.
4. The router picks the right tier for a task.
5. The cost tracker records the cost.
6. The quality feedback updates the strategy.
7. The cascade mode escalates on low confidence.
