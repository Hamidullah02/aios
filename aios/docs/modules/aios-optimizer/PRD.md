# aios-optimizer — Product Requirements Document

## 1. Purpose

This document states the product requirements for the
aios-optimizer module. The module is the model
router. The module picks the cheapest model that the
verify layer will accept. The module learns the
routing from the empirical pass rate per call
fingerprint. The module is honest: every decision is
auditable, every outcome is recorded, and the
operator can override per call.

The user is a developer, not a general operator.
The developer reads the audit log. The developer
reads the cost/quality table. The developer
overrides the router with `--tier=strong` or
`--tier=weak` when the developer knows better. The
module serves the developer, not the other way
around.

## 2. Scope

The aios-optimizer module is in scope for this
document.

- The tiered model registry.
- The call fingerprint.
- The cost/quality table.
- The empirical pass rate.
- The routing decision.
- The override surface.
- The `aios stats` CLI.

The aios-optimizer module is not in scope for this
document.

- The model adapters. The adapters are in
  aios-models.
- The price table. The list price is in
  aios-models. The empirical cost is recorded
  here.
- The verify layer. The verify layer is in
  aios-verify. The optimizer reads the verify
  result, but does not own the verify step.
- The self-evaluation of the model. The optimizer
  never asks the model to grade itself.

## 3. Reference Documents

- aios-core PRD. See `docs/modules/aios-core/PRD.md`.
- aios-models PRD. See `docs/modules/aios-models/PRD.md`.
- aios-verify PRD. See `docs/modules/aios-verify/PRD.md`.
- aios-runtime PRD. See `docs/modules/aios-runtime/PRD.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Tier | A level of model capability: strong, medium, weak. |
| Fingerprint | A SHA-256 hash of the (prompt shape, tool set, role, context length bucket) tuple. |
| Pass rate | The empirical fraction of calls at a fingerprint that the verify layer accepted. |
| Empirical cost | The actual USD cost of a call, recorded in the audit log. |
| Verify result | A boolean: the verify layer accepted or rejected the call output. |
| Override | A per-call `--tier=strong` or `--tier=weak` flag that bypasses the router. |
| Default | The routing mode when the cost/quality table is empty for a fingerprint. |

## 5. Determinism

The whole point of aios is to make complex coding and large systems deterministic. The determinism stance of this module is **Highly deterministic**.

The router is a pure function of the cost/quality table, the override, the budget, and the fingerprint. The cost/quality table is content addressed. The fingerprint is content addressed. The override is a CLI flag. The budget is a CLI flag. The router has no state, no clock, and no random source.

The module provides the following determinism guarantees.

- The router is a pure function of its inputs. Two calls with the same fingerprint, the same cost/quality table, the same override, and the same budget return the same decision.
- The cost/quality table is content addressed. The table id is the SHA-256 of the table payload.
- The pass rate is computed from the verify results, not from the model grading itself. The model never grades itself.
- The empirical cost is the actual recorded cost, not the list price. The empirical cost is exact.

## 6. User Personas

The user persona is the developer who runs the
agent. The developer is also the operator. The
developer sets the budget. The developer reads the
audit log. The developer reads `aios stats`. The
developer overrides the router when the developer
knows better.

The developer does not need a clever classifier.
The developer needs an honest one.

## 7. User Stories

A developer wants the agent to use the cheapest
model that will pass the verify layer. The
developer runs the agent. The optimizer picks the
weakest model that has a pass rate above the
threshold at the fingerprint. The optimizer
records the outcome. The next similar call uses
the same routing.

A developer wants to override the router. The
developer types `--tier=strong` on the CLI. The
router returns the strong tier for this call. The
cost/quality table is updated, but the override is
recorded as the source of the decision.

A developer wants to see why the router picked a
tier. The developer runs `aios stats
--fingerprint=<hash>`. The CLI shows the pass
rate, the cost, and the call count for the
fingerprint.

A developer wants to clear the cost/quality
table. The developer runs `aios stats --reset`.
The table is empty. The next call uses the
default routing.

A developer wants to disable the optimizer. The
developer types `--optimizer=off` on the CLI. The
router returns the strong tier for every call. The
cost/quality table is not updated.

A developer wants to set a hard budget. The
developer types `--budget=0.10` on the CLI. The
router returns the strongest tier that fits the
budget. The router stops the run when the budget
is exhausted.

## 8. Functional Requirements Summary

The aios-optimizer module provides the following
functions.

- The tiered model registry.
- The call fingerprint.
- The cost/quality table.
- The empirical pass rate.
- The routing decision.
- The override surface.
- The `aios stats` CLI.

## 9. Non-Functional Requirements Summary

- The module runs on Go 1.23.
- The module depends on aios-core, aios-models,
  and aios-verify.
- The module has 100 percent test coverage for
  the public functions.
- The router decision has a maximum latency of 5
  milliseconds.
- The cost/quality table is a single JSON file.
  The file is at `var/optimizer.json`. The file is
  readable by the developer with `cat` and `jq`.

## 10. Success Criteria

- The router picks the cheapest tier that the
  verify layer will accept, within the budget.
- The cost/quality table records the empirical
  pass rate per (fingerprint, tier) pair.
- The override surface works. `--tier=strong`
  always returns strong. `--tier=weak` always
  returns weak. `--optimizer=off` always returns
  strong.
- The `aios stats` CLI shows the table. The CLI
  shows the routing decisions. The CLI shows the
  override count.
- The router never asks the model to grade itself.
- The test suite has a 100 percent pass rate.

## 11. Out of Scope

- The model self-evaluation. The optimizer never
  asks the model to grade itself.
- The ML classifier. The optimizer is a lookup
  table, not a learned model.
- The keyword heuristic. The keyword heuristic
  was removed. The fingerprint replaces it.
- The cascade mode. The cascade was removed. The
  cost/quality table replaces it.
- The quality threshold. The threshold was
  removed. The pass rate is reported, not used as
  a gate.
- The model training.
- The model fine-tuning.
- The model evaluation suite.
