# aios-approvals — Wireframes

## 1. Purpose

This document describes the wireframes for the
aios-approvals module. The module has one user-facing
surface: the approval prompt in the CLI.

## 2. Scope

The scope of this document is the approval prompt in
the CLI. The prompt is a text-based prompt on the
terminal.

## 3. Reference Documents

- aios-approvals PRD. See
  `docs/modules/aios-approvals/PRD.md`.
- aios-approvals SRS. See
  `docs/modules/aios-approvals/SRS.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Wireframe | A visual layout for a user-facing surface. |
| Prompt | A text request for a user decision. |

## 5. Screen List

The module has one screen: the approval prompt.

## 6. Wireframes

### 6.1 Approval Prompt

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

### 6.2 Approval Granted

```
================================================================
  APPROVAL GRANTED
================================================================
  Approval id:  apr-9f2a
  Granted at:   2025-01-15T10:30:00Z
================================================================
```

### 6.3 Approval Denied

```
================================================================
  APPROVAL DENIED
================================================================
  Approval id:  apr-9f2a
  Denied at:    2025-01-15T10:30:00Z
  Reason:       (none)
================================================================
```

## 7. Interaction Notes

- The operator types `y` and presses Enter to
  grant.
- The operator types `n` and presses Enter to deny.
- The operator types `a` and presses Enter to allow
  the rule for the session.
- The operator types `v` and presses Enter to view
  the diff before the decision.
- The default choice is `n` when the input is empty.
