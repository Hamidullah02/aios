# aios-optimizer — Wireframes

## 1. Purpose

This document describes the wireframes for the
aios-optimizer module. The module has one
user-facing surface: the optimizer report printed
by `aios report`.

## 2. Scope

The scope of this document is the optimizer
section of the report.

## 3. Reference Documents

- aios-optimizer PRD. See
  `docs/modules/aios-optimizer/PRD.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Wireframe | A visual layout for a user-facing surface. |

## 5. Screen List

The module has one screen: the optimizer report
section.

## 6. Wireframes

### 6.1 Optimizer Report

```
================================================================
  OPTIMIZER REPORT
================================================================
  Mode:        cascade  budget_aware: on
  Total cost:  $1.234 / $5.00  (24.7%)
================================================================
  TIER  | CALLS | TOKENS (in/out) | COST   | QUALITY
  -------|-------|-----------------|--------|--------
  strong |     3 |   2,345 / 1,234 | $0.890 |  0.94
  medium |     8 |   8,901 / 4,567 | $0.310 |  0.88
  weak   |    12 |   1,234 /   567 | $0.034 |  0.81
  -------|-------|-----------------|--------|--------
  TOTAL  |    23 |  12,480 / 6,368 | $1.234 |  0.87
================================================================
  Escalations:   2 (8.7%)
  Cascade wins:  7 of 12 weak-tier tasks
================================================================
```

### 6.2 Routing Decision

```
================================================================
  ROUTING DECISION
================================================================
  Task:    refactor the auth module
  Tier:    strong
  Model:   claude-sonnet-4-5
  Reason:  strong-keyword match
  Confidence: 0.92
  Budget remaining: 75.3%
================================================================
```

## 7. Interaction Notes

- The optimizer report is appended to the run
  report.
- The routing decision is shown when the
  `--verbose` flag is set.
- The operator can override the tier with the
  `--tier` flag.
