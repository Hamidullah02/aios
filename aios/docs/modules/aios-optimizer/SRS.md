# aios-optimizer — Software Requirements Specification

## 1. Purpose

This document states the software requirements for
the aios-optimizer module.

## 2. Scope

The scope of this document is the aios-optimizer
module. The module is a Go package.

## 3. Reference Documents

- aios-optimizer PRD. See
  `docs/modules/aios-optimizer/PRD.md`.
- aios-core SRS. See `docs/modules/aios-core/SRS.md`.
- aios-models SRS. See `docs/modules/aios-models/SRS.md`.
- aios-verify SRS. See `docs/modules/aios-verify/SRS.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Tier | A typed string: strong, medium, weak. |
| Fingerprint | A typed SHA-256 hash. |
| Pass rate | A typed float between 0 and 1. |
| Override | A typed string: strong, medium, weak, off. |

## 5. Functional Requirements

### 5.1 Tiered Model Registry

**REQ-OPT-REG-001** The module shall provide the
`TieredModelRegistry` type. The type shall hold
the list of tiers.

**REQ-OPT-REG-002** A tier shall have a `name`, a
`model_id`, a `cost_per_1k_input`, a
`cost_per_1k_output`, and a `context_window`.

**REQ-OPT-REG-003** The default registry shall
include one model per tier. The operator can
override the registry through the config.

### 5.2 Call Fingerprint

**REQ-OPT-FIN-001** The module shall provide the
`Fingerprint` type. The type shall be a SHA-256
hash.

**REQ-OPT-FIN-002** The fingerprint shall be
computed from the following inputs.

- The prompt shape. The shape is the prompt with
  the variable parts replaced by their type tags.
- The tool set. The set is the sorted list of
  tool names.
- The role. The role is the role name.
- The context length bucket. The bucket is the
  input token count rounded down to the nearest
  power of 2.

**REQ-OPT-FIN-003** Two calls with the same
inputs shall produce the same fingerprint. The
function shall be a pure function of the inputs.

### 5.3 Cost/Quality Table

**REQ-OPT-CQT-001** The module shall provide the
`CostQualityTable` type. The type shall hold the
empirical pass rate and the empirical cost per
(fingerprint, tier) pair.

**REQ-OPT-CQT-002** The table shall be loaded
from `var/optimizer.json`. The file is a JSON
object. The file is content addressed. The file
id is the SHA-256 of the file payload.

**REQ-OPT-CQT-003** The table shall provide the
`Get` method. The method shall accept a
fingerprint and a tier. The method shall return
the pass rate, the call count, and the average
empirical cost. The method shall return nil when
the pair is not in the table.

**REQ-OPT-CQT-004** The table shall provide the
`Record` method. The method shall accept a
fingerprint, a tier, an empirical cost, and a
verify result. The method shall update the table.

**REQ-OPT-CQT-005** The table shall provide the
`Reset` method. The method shall clear the table.
The method shall write the empty table to the
file.

**REQ-OPT-CQT-006** The table shall be append-only
in memory. The table shall be written to the file
in one shot on every `Record` call. The file
write shall be atomic through a `rename` syscall.

### 5.4 Empirical Pass Rate

**REQ-OPT-EPR-001** The pass rate shall be
computed from the verify results. The verify
result is a boolean. The pass rate is the
fraction of `true` results in the last N calls at
the (fingerprint, tier) pair. The default N is
20.

**REQ-OPT-EPR-002** The pass rate shall be a
float between 0 and 1. The pass rate shall be
`0.0` when the call count is 0. The pass rate
shall be `1.0` when all calls passed.

**REQ-OPT-EPR-003** The pass rate shall never be
self-reported by the model. The pass rate shall
always be derived from the verify result. The
verify result is owned by `aios-verify`.

### 5.5 Routing Decision

**REQ-OPT-RTE-001** The module shall provide the
`Router` type. The type shall accept the registry,
the table, and the budget.

**REQ-OPT-RTE-002** The router shall provide the
`Route` method. The method shall accept a
fingerprint, an override, and a budget. The
method shall return a `RoutingDecision`.

**REQ-OPT-RTE-003** The decision shall include
the `tier`, the `model_id`, the `reason`, and
the `source`.

**REQ-OPT-RTE-004** The `source` shall be one of
the following values.

- `override`. The decision came from the CLI
  flag.
- `table`. The decision came from the cost/quality
  table.
- `default`. The decision came from the default
  rule because the table is empty.
- `budget`. The decision came from the budget
  cap.
- `optimizer_off`. The decision came from the
  `--optimizer=off` flag.

**REQ-OPT-RTE-005** The router shall return the
override tier when the override is set. The
router shall record the decision as `override`.

**REQ-OPT-RTE-006** The router shall return the
strong tier for every call when the override is
`off`. The router shall record the decision as
`optimizer_off`.

**REQ-OPT-RTE-007** The router shall look up the
fingerprint in the table when the override is
empty. When the fingerprint is in the table, the
router shall return the tier with the highest
pass rate per unit cost. When the fingerprint is
not in the table, the router shall use the
default rule.

**REQ-OPT-RTE-008** The default rule shall be
`cost_first`. The router shall return the weakest
tier. The operator can change the default rule
through the config.

**REQ-OPT-RTE-009** The router shall enforce the
budget. When the budget is exhausted, the router
shall return the weakest tier that fits the
remaining budget. When no tier fits, the router
shall return an error.

### 5.6 Override Surface

**REQ-OPT-OVR-001** The CLI shall accept the
`--tier` flag. The flag shall accept the values
`strong`, `medium`, `weak`, and `auto`. The
default is `auto`.

**REQ-OPT-OVR-002** The CLI shall accept the
`--optimizer` flag. The flag shall accept the
values `on` and `off`. The default is `on`.

**REQ-OPT-OVR-003** The CLI shall accept the
`--budget` flag. The flag shall accept a USD
amount. The default is `0.00` which means no
budget cap.

### 5.7 `aios stats` CLI

**REQ-OPT-STA-001** The CLI shall provide the
`aios stats` subcommand. The subcommand shall
print the cost/quality table.

**REQ-OPT-STA-002** The subcommand shall accept
the `--fingerprint` flag. The flag shall filter
the table to a single fingerprint.

**REQ-OPT-STA-003** The subcommand shall accept
the `--tier` flag. The flag shall filter the
table to a single tier.

**REQ-OPT-STA-004** The subcommand shall accept
the `--reset` flag. The flag shall clear the
table.

**REQ-OPT-STA-005** The subcommand shall print
the table as a JSON object. The JSON object
shall be readable by the developer with `cat` and
`jq`.

**REQ-OPT-STA-006** The subcommand shall print a
summary by default. The summary shall include the
total call count, the total cost, the override
count, the average pass rate, and the average
cost per tier.

## 6. Non-Functional Requirements

**REQ-OPT-NFR-001** The module shall run on Go
1.23.

**REQ-OPT-NFR-002** The module shall have 100
percent coverage for the public functions.

**REQ-OPT-NFR-003** The router decision shall
have a maximum latency of 5 milliseconds.

**REQ-OPT-NFR-004** The cost/quality table shall
be a single JSON file. The file shall be at
`var/optimizer.json`. The file shall be readable
with `cat` and `jq`.

## 7. Interface Requirements

**REQ-OPT-IFC-001** The module shall export the
following items from
`github.com/<org>/aios/internal/optimizer`.

- `Router`
- `TieredModelRegistry`
- `CostQualityTable`
- `Fingerprint`
- `RoutingDecision`
- `Override`
- `Tier`
- `NewFingerprint`

## 8. Data Requirements

**REQ-OPT-DAT-001** The `Tier` schema shall
include the fields below.

- `name` (string)
- `model_id` (string)
- `cost_per_1k_input` (float64)
- `cost_per_1k_output` (float64)
- `context_window` (int)

**REQ-OPT-DAT-002** The `Fingerprint` schema
shall be a SHA-256 hash encoded as a lowercase
hex string of 64 characters.

**REQ-OPT-DAT-003** The `RoutingDecision` schema
shall include the fields below.

- `tier` (string)
- `model_id` (string)
- `reason` (string)
- `source` (string: override, table, default,
  budget, optimizer_off)

**REQ-OPT-DAT-004** The cost/quality table
schema shall be a JSON object. The keys are
`fingerprint:tier` pairs. The values are the
`CostQualityRow` objects.

**REQ-OPT-DAT-005** The `CostQualityRow` schema
shall include the fields below.

- `call_count` (int)
- `pass_count` (int)
- `total_cost_usd` (float64)
- `first_seen` (string, ISO 8601)
- `last_seen` (string, ISO 8601)

**REQ-OPT-DAT-006** The default `var/optimizer.yaml`
shall list the tiers. The file shall be loaded on
`aios init`.

## 9. Error Handling

**REQ-OPT-ERR-001** The router shall return
`ErrBudgetExhausted` when no tier fits the
remaining budget.

**REQ-OPT-ERR-002** The router shall return
`ErrUnknownTier` when the override is not a
known tier.

**REQ-OPT-ERR-003** The cost/quality table shall
return `ErrTableCorrupt` when the file fails to
parse.

## 10. Acceptance Criteria

1. Every REQ in this document has a test case.
2. The test suite has a 100 percent pass rate.
3. The test suite has 100 percent coverage for
   the public functions.
4. The router is a pure function of its inputs.
5. The fingerprint is a pure function of the
   (prompt shape, tool set, role, context length
   bucket) tuple.
6. The cost/quality table is a JSON file. The
   file is readable with `cat` and `jq`.
7. The override flag works. `--tier=strong`
   always returns strong.
8. The `--optimizer=off` flag always returns
   strong.
9. The `--budget` flag caps the cost.
10. The router never asks the model to grade
    itself.
11. The `aios stats` CLI prints the table and the
    summary.
