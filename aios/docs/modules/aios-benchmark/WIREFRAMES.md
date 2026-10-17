# aios-benchmark — Wireframes

## 1. Purpose

This document describes the wireframes for the
aios-benchmark module. The module has one user-facing
surface: the CLI output.

## 2. Scope

The scope of this document is the CLI output of the
benchmark.

## 3. Reference Documents

- aios-benchmark PRD. See
  `docs/modules/aios-benchmark/PRD.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Wireframe | A visual layout for a user-facing surface. |

## 5. Screen List

The module has one screen: the benchmark report.

## 6. Wireframes

### 6.1 Benchmark Report

```
================================================================
  BENCHMARK REPORT
================================================================
  Dataset:    wedge (12 tasks)
  Tier:       strong (claude-sonnet-4-5)
  Duration:   312.4s
  Cost:       $1.234
================================================================
  WEDGE             | TASKS | PASS | DURATION | COST   | LOOP
  ------------------|-------|------|----------|--------|------
  file_read         |     2 |    2 |    12.1s | $0.012 |  0
  file_write        |     2 |    2 |    18.3s | $0.020 |  0
  file_search       |     1 |    1 |     8.2s | $0.008 |  0
  shell_run         |     2 |    2 |    41.5s | $0.041 |  0
  approval          |     2 |    2 |    95.0s | $0.310 |  0
  ask_user          |     1 |    1 |    22.4s | $0.025 |  0
  resume            |     1 |    1 |    48.2s | $0.150 |  0
  cancel            |     1 |    1 |     6.7s | $0.005 |  0
================================================================
  TOTAL             |    12 |   12 |   312.4s | $1.234 |  0
================================================================
```

### 6.2 Two-Tier Report

```
================================================================
  TWO-TIER REPORT
================================================================
  Metric        | Strong    | Weak      | Delta
  --------------|-----------|-----------|--------
  Pass rate     |    100.0% |     91.7% |   +8.3%
  Avg duration  |    26.0s  |    32.4s  |   -6.4s
  Avg cost      |  $0.103   |  $0.087   |  +$0.016
  Loop rate     |     0.0%  |     8.3%  |   -8.3%
================================================================
```

## 7. Interaction Notes

- The CLI prints the report after the run.
- The CLI writes the report to a JSON file.
- The CLI returns exit code 0 on success.
- The CLI returns exit code 1 on any task failure.
