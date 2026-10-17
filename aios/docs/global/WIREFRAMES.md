# aios — Global Wireframes

## 1. Purpose

This document describes the system-level wireframes
for the aios project. The wireframes cover the
user-facing surfaces that span more than one module.
The per-module WIREFRAMES documents are the source
of truth for the module detail.

## 2. Scope

The aios project is in scope for this document.

- The CLI prompts.
- The report.
- The watcher event stream.
- The benchmark dashboard.
- The audit log viewer.

## 3. Reference Documents

- Global PRD. See `docs/global/PRD.md`.
- The per-module WIREFRAMES under
  `docs/modules/<module>/WIREFRAMES.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Wireframe | A visual layout for a user-facing surface. |

## 5. Screen List

The aios project has the following system-level
surfaces.

- The CLI prompts.
- The report.
- The watcher event stream.
- The benchmark dashboard.
- The audit log viewer.

## 6. Wireframes

### 6.1 CLI Run Prompt

```
================================================================
  AIOS
================================================================
  Goal: refactor the auth module to use a session token
  Model: claude-sonnet-4-5 (strong)
  Budget: $1.00
================================================================
  Plan: 5 tasks
    1. explore    - read the auth module
    2. plan       - design the refactor
    3. code       - write the refactor
    4. test       - run the test suite
    5. verify     - check the result
================================================================
  [r] Run    [e] Edit plan    [c] Cancel
================================================================
  Choice: _
```

### 6.2 CLI Approval Prompt

```
================================================================
  APPROVAL REQUIRED
================================================================
  Task:    task-003 (code)
  Tool:    write_file
  Resource: internal/auth/session.py
================================================================
  Diff:
    + def create_session(user_id: str) -> str:
    +     return secrets.token_urlsafe(32)
================================================================
  [y] Grant            [n] Deny
  [a] Always allow     [v] View full diff
================================================================
  Choice: _
```

### 6.3 Report

```
================================================================
  RUN REPORT
================================================================
  Session:    sess-abc-123
  Goal:       refactor the auth module
  Started:    2026-10-17T10:30:00Z
  Duration:   4m 23s
  Cost:       $0.34 / $1.00
  Status:     succeeded
================================================================
  TASK     | ROLE     | MODEL     | DURATION | COST  | STATUS
  ---------|----------|-----------|----------|-------|--------
  explore  | explorer | haiku-4-5 |    8s    | $0.01 | passed
  plan     | planner  | sonnet-4-5|   45s    | $0.08 | passed
  code     | coder    | sonnet-4-5|   92s    | $0.15 | passed
  test     | tester   | haiku-4-5 |   31s    | $0.04 | passed
  verify   | verifier | sonnet-4-5|   24s    | $0.06 | passed
================================================================
  Approval events: 0
  Retry events:    1
  Verification:    pass
================================================================
```

### 6.4 Watcher Event Stream

```
================================================================
  WATCHER
================================================================
  10:30:01.234  [task_started]   task-001 explore
  10:30:01.456  [tool_call]      read_file internal/auth/session.py
  10:30:01.789  [tool_result]    ok (543 bytes)
  10:30:09.123  [task_completed] task-001 explore
  10:30:09.456  [task_started]   task-002 plan
  ...
================================================================
```

### 6.5 Benchmark Dashboard

```
================================================================
  BENCHMARK RESULTS
================================================================
  Dataset:  wedge-20
  Date:     2026-10-17
  Model:    claude-sonnet-4-5
================================================================
  TASK                          | PASS  | TIME   | COST
  ------------------------------|-------|--------|-------
  add-a-function                |  yes  |  12s   | $0.04
  fix-a-typo                    |  yes  |   4s   | $0.01
  refactor-a-module             |  yes  |  92s   | $0.18
  write-a-test                  |  yes  |  28s   | $0.06
  debug-a-failing-test          |  yes  |  41s   | $0.09
  ...
================================================================
  Pass rate: 18/20 (90.0%)
  Avg time:  32.4s
  Avg cost:  $0.07
  Total cost: $1.34
================================================================
```

### 6.6 Audit Log Viewer

```
================================================================
  AUDIT LOG
================================================================
  Session: sess-abc-123
  Rows:    127
  Chain:   verified
================================================================
  TIME                | ACTOR     | ACTION              | RESULT
  --------------------|-----------|---------------------|--------
  10:30:01.234        | orchestr. | run_started         | ok
  10:30:01.345        | planner   | plan_generated      | ok
  10:30:01.456        | orchestr. | task_started        | ok
  10:30:01.567        | tool_bus  | tool_call           | allow
  10:30:01.789        | tool_bus  | tool_result         | ok
  ...
  10:34:23.456        | report    | report_generated    | ok
  10:34:23.567        | orchestr. | run_completed       | ok
================================================================
  [n] Next page    [p] Previous page    [s] Search
  [v] Verify chain  [e] Export
================================================================
```

## 7. Interaction Notes

### 7.1 Run Prompt

- The operator types `r` to run the plan.
- The operator types `e` to edit the plan. The CLI
  opens the plan in the editor.
- The operator types `c` to cancel.

### 7.2 Approval Prompt

- The operator types `y` to grant the approval.
- The operator types `n` to deny the approval.
- The operator types `a` to grant and create a
  standing rule.
- The operator types `v` to view the full diff.

### 7.3 Report

- The report is shown in text by default.
- The report is shown in JSON with `--format json`.
- The report is shown in HTML with `--format html`.

### 7.4 Watcher Event Stream

- The watcher streams the events live.
- The operator can pause with `Ctrl+S`.
- The operator can filter with `Ctrl+F`.

### 7.5 Benchmark Dashboard

- The benchmark runs the wedge-20 dataset.
- The benchmark shows the pass rate, the average
  time, and the average cost.
- The benchmark exports the results to
  `var/reports/`.

### 7.6 Audit Log Viewer

- The viewer paginates the log.
- The viewer verifies the hash chain.
- The viewer exports the log to a file.
