# aios-optimizer — Application Flow

## 1. Purpose

This document describes the application flow for the
aios-optimizer module.

## 2. Scope

The scope of this document is the aios-optimizer
module. The module is a Go package.

## 3. Reference Documents

- aios-optimizer PRD. See
  `docs/modules/aios-optimizer/PRD.md`.
- aios-optimizer SRS. See
  `docs/modules/aios-optimizer/SRS.md`.
- aios-optimizer TRD. See
  `docs/modules/aios-optimizer/TRD.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Caller | The code that calls the router. |
| Operator | The developer who runs the CLI. |

## 5. User Flow

The user flow for the optimizer is the `aios stats`
CLI. The operator types `aios stats` to see the
cost/quality table. The operator types
`aios stats --fingerprint=<hash>` to see a single
row. The operator types `aios stats --reset` to
clear the table.

## 6. System Flow

### 6.1 Route Flow

The route flow runs as follows.

1. The caller calls `router.Route` with the
   fingerprint, the override, and the budget.
2. The router checks the override.
3. When the override is `off`, the router returns
   the strong tier.
4. When the override is `strong`, `medium`, or
   `weak`, the router returns the override tier.
5. The router looks up the fingerprint in the
   table.
6. When the fingerprint is in the table for two
   or more tiers, the router returns the tier
   with the highest score.
7. When the fingerprint is not in the table, the
   router uses the default rule.
8. The router enforces the budget.
9. The router returns the decision.

### 6.2 Record Flow

The record flow runs as follows.

1. The caller calls `table.Record` with the
   fingerprint, the tier, the cost, and the
   verify result.
2. The table updates the in-memory row.
3. The table writes the file atomically.
4. The table returns nil.

### 6.3 Reset Flow

The reset flow runs as follows.

1. The operator types `aios stats --reset`.
2. The CLI calls `table.Reset`.
3. The table clears the in-memory rows.
4. The table writes the empty table to the file.
5. The CLI returns success.

### 6.4 Stats Flow

The stats flow runs as follows.

1. The operator types `aios stats`.
2. The CLI reads the table from the file.
3. The CLI prints the table as JSON.
4. The CLI prints the summary.

## 7. Error Flows

### 7.1 Budget Exhausted

The budget exhausted flow runs as follows.

1. The budget is exhausted.
2. The router returns `ErrBudgetExhausted`.
3. The caller stops the run.

### 7.2 Unknown Tier

The unknown tier flow runs as follows.

1. The override is not a known tier.
2. The router returns `ErrUnknownTier`.
3. The caller reports the error.

### 7.3 Table Corrupt

The table corrupt flow runs as follows.

1. The file fails to parse.
2. The table returns `ErrTableCorrupt`.
3. The caller reports the error.

## 8. Sequence Diagrams

### 8.1 Route Sequence

The route sequence has the steps below.

1. The caller calls `router.Route`.
2. The router checks the override.
3. The router looks up the fingerprint in the
   table.
4. The router enforces the budget.
5. The router returns the decision.

### 8.2 Record Sequence

The record sequence has the steps below.

1. The caller calls `table.Record`.
2. The table updates the in-memory row.
3. The table writes the file atomically.
4. The table returns nil.
