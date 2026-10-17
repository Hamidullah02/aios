# aios-optimizer — Product Requirements Document

## 1. Purpose

This document states the product requirements for the
aios-optimizer module. The module is the model
router with cost and quality optimization. The module
picks the right model for each task. The module
tracks the cost. The module adapts the routing
strategy from the result of every call.

## 2. Scope

The aios-optimizer module is in scope for this
document.

- The tiered model registry.
- The task classifier.
- The cost tracker.
- The routing strategy.
- The quality feedback loop.

The aios-optimizer module is not in scope for this
document.

- The model adapters. The adapters are in
  aios-models.
- The price table. The price table is in aios-models.
- The capability registry. The registry is in
  aios-models.

## 3. Reference Documents

- Project design document, section 15.
- aios-core PRD. See `docs/modules/aios-core/PRD.md`.
- aios-models PRD. See `docs/modules/aios-models/PRD.md`.
- OpenAI Predicted Outputs reference.
- Anthropic prompt caching reference.
- RouteLLM research paper.
- NotDiamond and Martian routing documentation.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Tier | A level of model capability: strong, medium, weak. |
| Strategy | A rule that maps a task to a tier. |
| Classifier | A component that picks the tier for a task. |
| Quality score | A number that measures the result quality. |
| Cost budget | The maximum cost for a task. |


## 5. Determinism

The whole point of aios is to make complex coding and large systems deterministic. The determinism stance of this module is **Bounded by `temperature=0`**.

The router is deterministic for a fixed classifier. The cascade escalates on low confidence. Quality feedback updates the strategy from the result quality.

The module provides the following determinism guarantees.

- Router decisions are pure functions of the classifier and the cost tracker.

- Classifier heuristics are deterministic.

- Cascade escalation is deterministic for a fixed cascade threshold.
## 6. User Personas

The user persona is the operator who sets the cost
budget. The operator wants the agent to stay within
the budget. The operator also wants the agent to
produce high-quality results.

## 7. User Stories

An operator wants to set a cost budget. The
operator passes the budget to the optimizer. The
optimizer picks the cheapest model that meets the
quality bar.

An operator wants the optimizer to learn. The
operator runs the agent. The optimizer records the
result quality. The optimizer adjusts the strategy.

An agent wants to allocate a per-task budget. The
agent passes the task and the budget to the
optimizer. The optimizer picks the model. The
optimizer enforces the budget.

## 8. Functional Requirements Summary

The aios-optimizer module provides the following
functions.

- The tiered model registry.
- The task classifier.
- The cost tracker.
- The routing strategy.
- The quality feedback loop.

## 9. Non-Functional Requirements Summary

- The module runs on Go 1.23.
- The module depends on aios-core and aios-models.
- The module has 100 percent test coverage for the
  public functions.
- The router decision has a maximum latency of 5
  milliseconds.

## 10. Success Criteria

- The router picks the right tier for a task.
- The cost tracker records the cost.
- The quality feedback loop adjusts the strategy.
- The test suite has a 100 percent pass rate.

## 11. Out of Scope

- The model training.
- The model fine-tuning.
- The model evaluation suite.
