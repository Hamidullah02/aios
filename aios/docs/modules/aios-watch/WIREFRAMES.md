# aios-watch — Wireframes

## 1. Purpose

This document describes the wireframes for the
aios-watch module. The module has one user-facing
surface: the CLI stream output.

## 2. Scope

The scope of this document is the CLI stream
output.

## 3. Reference Documents

- aios-watch PRD. See `docs/modules/aios-watch/PRD.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Wireframe | A visual layout for a user-facing surface. |

## 5. Screen List

The module has one screen: the CLI stream output.

## 6. Wireframes

### 6.1 CLI Stream Output

```
================================================================
  RUNNING  task-001  [step 3/5]  $0.234 / $5.00
================================================================
  Model:    claude-sonnet-4-5
  Tokens:   12,345 in / 4,567 out
  Plan:     plan-abc
  Goal:     Add a new module to the project
================================================================
> Reading the schemas file...
  [read_file] internal/core/schemas.py
  [OK] 1.2s
> Looking for the audit log class...
  [grep] "type AuditLog" in internal/core/
  [OK] 0.4s  (1 match)
> Editing the schemas file to add a new schema...
  [edit] internal/core/schemas.py
  [OK] 1.8s
> Running the tests...
  [bash] go test tests/test_schemas.py
  [OK] 41.5s  (24 passed)
================================================================
  File changes:
    M internal/core/schemas.py
================================================================
  [Type your message or press Enter to interrupt]
>
```

### 6.2 Sub-agent Output

```
================================================================
  SUB-AGENT  explore-1
================================================================
  Type:     explore
  Prompt:   Find all the places that call the audit log
  Status:   running
  Duration: 12.4s
  Cost:     $0.045
================================================================
  > Reading the runtime module...
  > Found 3 call sites
  > Returning the result
================================================================
  Result:   The audit log is called from orchestrator.py,
            worker.py, and resume.py.
================================================================
```

## 7. Interaction Notes

- The CLI updates the screen on every chunk.
- The CLI uses ANSI escape codes for the updates.
- The CLI shows the file changes in real time.
- The CLI shows the cost and the token count.
- The user can interrupt the run with Ctrl+C.
- The sub-agent output is indented and marked with
  a `SUB-AGENT` header.
