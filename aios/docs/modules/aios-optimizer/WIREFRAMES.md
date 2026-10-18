# aios-optimizer — Wireframes

## 1. Purpose

This document describes the wireframes for the
aios-optimizer module. The module has one
user-facing surface: the `aios stats` CLI output
and the optimizer section of the run report.

## 2. Scope

The scope of this document is the `aios stats` CLI
output and the optimizer section of the run
report.

## 3. Reference Documents

- aios-optimizer PRD. See
  `docs/modules/aios-optimizer/PRD.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Wireframe | A visual layout for a user-facing surface. |

## 5. Screen List

The module has two screens.

- The `aios stats` output.
- The optimizer section of the run report.

## 6. Wireframes

### 6.1 `aios stats` Output

```
================================================================
  OPTIMIZER STATS
================================================================
  Source:    var/optimizer.json
  Table id:  sha256:9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08
  Updated:   2026-10-17T10:34:23Z
================================================================
  CLASS | CALLS | PASS  | RATE  | COST/1K  | AVG COST
  -------|-------|-------|-------|----------|----------
  strong |    12 |    12 | 1.000 | $0.0240  | $0.0240
  mid    |    23 |    22 | 0.957 | $0.0030  | $0.0030
  cheap  |   187 |   176 | 0.941 | $0.0003  | $0.0003
  -------|-------|-------|-------|----------|----------
  TOTAL  |   222 |   210 | 0.946 |          | $0.342
================================================================
  Overrides:    8
  Off:          0
  Budget hits:  0
================================================================
```

### 6.2 `aios stats --fingerprint=<hash>` Output

```
================================================================
  FINGERPRINT
================================================================
  Hash:     sha256:9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08
  Prompt:   "refactor the <path> module to use <type>"
  Tools:    [edit, read_file, search_files]
  Role:     coder
  Context:  1024
================================================================
  CLASS | CALLS | PASS  | RATE  | AVG COST | LAST SEEN
  -------|-------|-------|-------|----------|-----------
  strong |     2 |     2 | 1.000 | $0.0180  | 2026-10-17T10:30:00Z
  mid    |     5 |     5 | 1.000 | $0.0028  | 2026-10-17T10:31:00Z
  cheap  |    19 |    19 | 1.000 | $0.0003  | 2026-10-17T10:34:00Z
================================================================
  Default rule: cost_first
  Last decision: cheap (source: table, score: 3333)
================================================================
```

### 6.3 Optimizer Section of the Run Report

```
================================================================
  OPTIMIZER REPORT
================================================================
  Mode:        on  (override: auto, budget: $0.10)
  Total cost:  $0.087 / $0.10  (87.0%)
================================================================
  DECISIONS
  -------
  Class  | Source     | Count | Cost   | Pass rate
  -------|------------|-------|--------|----------
  strong | override   |     2 | $0.048 | 1.000
  mid    | table      |     3 | $0.009 | 1.000
  cheap  | table      |    41 | $0.012 | 0.976
  cheap  | default    |    12 | $0.018 | 0.917
  -------|------------|-------|--------|----------
  TOTAL  |            |    58 | $0.087 | 0.966
================================================================
  Budget hits: 0
  Table updates: 56
================================================================
```

## 7. Interaction Notes

- The operator types `aios stats` to see the
  summary.
- The operator types `aios stats
  --fingerprint=<hash>` to see a single
  fingerprint.
- The operator types `aios stats --class=cheap` to
  filter by class.
- The operator types `aios stats --reset` to
  clear the table.
- The operator types `cat var/optimizer.json |
  jq` to inspect the raw table.
- The operator types `aios run "<goal>"
  --class=strong` to override the router.
- The operator types `aios run "<goal>"
  --optimizer=off` to disable the router.
- The operator types `aios run "<goal>"
  --budget=0.10` to set a hard budget cap.
