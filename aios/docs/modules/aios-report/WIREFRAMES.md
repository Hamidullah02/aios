# aios-report — Wireframes

## 1. Purpose

This document describes the wireframes for the
aios-report module. The module produces the report
in three formats: JSON, Markdown, and HTML. The
wireframes show the layout of the Markdown and the
HTML reports.

## 2. Scope

The scope of this document is the report layout.

## 3. Reference Documents

- aios-report PRD. See `docs/modules/aios-report/PRD.md`.
- aios-report SRS. See `docs/modules/aios-report/SRS.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Wireframe | A visual layout for a user-facing surface. |

## 5. Screen List

The module has two report screens: the Markdown
report and the HTML report.

## 6. Wireframes

### 6.1 Markdown Report

```
# Run Report — task-001

**Status:** SUCCEEDED
**Duration:** 312.4s
**Cost:** $1.234
**Tokens:** 12,345 in / 4,567 out

## Plan

- Plan id: plan-abc
- Goal: Add a new module to the project
- Tasks: 5

## Steps

| # | Tool | Status | Duration | Cost |
|---|------|--------|----------|------|
| 1 | read_file | OK | 1.2s | $0.001 |
| 2 | write_file | OK | 2.3s | $0.002 |
| 3 | shell_run | OK | 41.5s | $0.041 |
| 4 | write_file | OK | 1.8s | $0.002 |
| 5 | shell_run | OK | 12.0s | $0.012 |

## Approvals

- apr-9f2a (granted)
- apr-3a1b (granted)
```

### 6.2 HTML Report

```
+----------------------------------------+
|  Run Report — task-001                 |
+----------------------------------------+
|  Status:   SUCCEEDED                   |
|  Duration: 312.4s                      |
|  Cost:     $1.234                      |
|  Tokens:   12,345 in / 4,567 out       |
+----------------------------------------+
|  Plan: plan-abc                        |
|  Goal:  Add a new module               |
+----------------------------------------+
|  Steps                                 |
|  +----+----------+-------+-------+    |
|  | #  | Tool     | Stat. | Dur.  |    |
|  +----+----------+-------+-------+    |
|  | 1  | read_fil | OK    | 1.2s  |    |
|  | 2  | write_fi | OK    | 2.3s  |    |
|  +----+----------+-------+-------+    |
+----------------------------------------+
|  Approvals                             |
|  - apr-9f2a (granted)                  |
|  - apr-3a1b (granted)                  |
+----------------------------------------+
```

## 7. Interaction Notes

- The Markdown report is written to
  `./reports/<task_id>.md`.
- The HTML report is written to
  `./reports/<task_id>.html`.
- The JSON report is written to
  `./reports/<task_id>.json`.
- The HTML report has a link to the audit log.
