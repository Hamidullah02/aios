# aios-cli — Wireframes

## 1. Purpose

This document describes the wireframes for the
aios-cli module. The module has the following
user-facing surfaces: the help text, the init
output, the run output, the approval prompt, the
resume prompt, the report output, the audit output,
and the error output.

## 2. Scope

The scope of this document is the CLI surfaces. The
surfaces are text-based outputs on the terminal.

## 3. Reference Documents

- aios-cli PRD. See `docs/modules/aios-cli/PRD.md`.
- aios-cli SRS. See `docs/modules/aios-cli/SRS.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Wireframe | A visual layout for a user-facing surface. |
| ANSI | The American National Standards Institute codes for terminal text. |

## 5. Screen List

The module has the following screens.

- Help text
- Init output
- Run banner
- Approval prompt
- Resume prompt
- Report output
- Audit output
- Error output

## 6. Wireframes

### 6.1 Help Text

```
Usage: aios [OPTIONS] COMMAND [ARGS]...

  The aios agentic OS CLI.

Options:
  --debug            Enable debug output
  --config PATH      Path to the config file
  --help             Show this message and exit

Commands:
  run       Run a plan from a file
  resume    Resume a paused task
  cancel    Cancel a running task
  report    Print the report for a task
  daemon    Manage the approval daemon
  audit     Print the audit log for a task
  init      Initialize the workspace
```

### 6.2 Init Output

```
================================================================
  aios init
================================================================
  Creating workspace in ./var/
  - var/audit.db          created
  - var/policy.yaml       created
  - var/egress.yaml       created
  - var/prices.yaml       created
  - var/config.yaml       created
================================================================
  Next steps:
  1. Edit var/policy.yaml to set your rules.
  2. Edit var/prices.yaml to set the model prices.
  3. Run: aios run path/to/plan.yaml
================================================================
```

### 6.3 Run Banner

```
================================================================
  RUN
================================================================
  Plan:     plan-abc
  Goal:     Add a new module to the project
  Tasks:    5
  Model:    claude-sonnet-4-5
  Budget:   $5.00 / 600s
================================================================
```

### 6.4 Approval Prompt

```
================================================================
  APPROVAL REQUIRED
================================================================
  Task:    task-001
  Tool:    write_file
  Action:  Write a file
  Resource: internal/core/schemas.py
  Reason:  Write to a project file
  Tier:    3
================================================================
  [y] Grant    [n] Deny    [a] Always allow    [v] View diff
================================================================
  Choice: _
```

### 6.5 Resume Prompt

```
================================================================
  PAUSED
================================================================
  Task:    task-001
  Reason:  Waiting for approval
  Checkpoint: ./var/checkpoints/task-001.json
================================================================
  Run: aios resume task-001
================================================================
```

### 6.6 Report Output

```
================================================================
  REPORT — task-001
================================================================
  Status:    SUCCEEDED
  Duration:  312.4s
  Cost:      $1.234
  Tokens:    12,345 in / 4,567 out
  Approvals: 2 (granted)
================================================================
  Plan:      plan-abc
  Steps:     5
================================================================
  Step 1: read_file  internal/core/schemas.py        OK   1.2s
  Step 2: write_file internal/core/schemas.py        OK   2.3s
  Step 3: shell_run  go test tests/                   OK  41.5s
  Step 4: write_file internal/core/audit.py          OK   1.8s
  Step 5: shell_run  go test tests/                   OK  12.0s
================================================================
```

### 6.7 Audit Output

```
================================================================
  AUDIT — task-001
================================================================
  2025-01-15T10:30:00Z  STATE       pending -> running
  2025-01-15T10:30:01Z  TOOL_CALL   read_file  -> OK   1.2s
  2025-01-15T10:30:02Z  POLICY      rule-1    allow
  2025-01-15T10:30:03Z  TOOL_CALL   write_file -> OK   2.3s
  2025-01-15T10:30:05Z  POLICY      rule-2    ask
  2025-01-15T10:30:05Z  APPROVAL    apr-9f2a  granted
  2025-01-15T10:30:06Z  TOOL_CALL   write_file -> OK   1.8s
  2025-01-15T10:30:18Z  STATE       running -> succeeded
================================================================
  Total: 8 rows. Use --tail 20 for the last 20 rows.
================================================================
```

### 6.8 Error Output

```
================================================================
  ERROR
================================================================
  Type:    PolicyValidationError
  Reason:  Rule rule-3 has no match conditions
  File:    var/policy.yaml
  Line:    12
================================================================
  Run with --debug for the full traceback.
================================================================
```

## 7. Interaction Notes

- The CLI uses ANSI color codes for the headers.
- The CLI supports a `--no-color` flag.
- The CLI supports a `--json` flag for the report
  subcommand.
- The CLI supports a `--tail N` flag for the audit
  subcommand.
- The CLI supports a `--format` flag for the report
  subcommand. The supported formats are text, json,
  markdown, and html.
- The approval prompt has a default choice of `n`
  when the input is empty.
- The approval prompt has a 5-minute timeout. The
  default is deny on timeout.
