# aios-persist — Wireframes

## 1. Purpose

This document describes the wireframes for the
aios-persist module. The module has one user-facing
surface: the `aios rules` CLI commands.

## 2. Scope

The scope of this document is the `aios rules` CLI
output.

## 3. Reference Documents

- aios-persist PRD. See
  `docs/modules/aios-persist/PRD.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Wireframe | A visual layout for a user-facing surface. |

## 5. Screen List

The module has the following screens.

- The `aios rules list` output.
- The approval prompt with the "always" option.

## 6. Wireframes

### 6.1 Rules List

```
================================================================
  RULES
================================================================
  ID         | SCOPE   | MATCH              | ACTION
  -----------|---------|--------------------|--------
  rule-001   | user    | tool=read_file     | allow
  rule-002   | project | tool_glob=write_*  | ask
  rule-003   | session | path_prefix=tmp/   | allow
================================================================
  Total: 3 rules (1 user, 1 project, 1 session)
================================================================
```

### 6.2 Approval Prompt with Always

```
================================================================
  APPROVAL REQUIRED
================================================================
  Task:    task-001
  Tool:    write_file
  Resource: internal/core/schemas.py
================================================================
  [y] Grant            [n] Deny
  [a] Always allow     [v] View diff
================================================================
  Choice: _
```

### 6.3 Session List

```
================================================================
  SESSIONS
================================================================
  ID          | STARTED              | STATUS    | STEPS
  ------------|----------------------|-----------|------
  sess-abc    | 2025-01-15T10:30:00Z | succeeded |   12
  sess-def    | 2025-01-15T09:15:00Z | failed    |    3
  sess-ghi    | 2025-01-14T16:42:00Z | canceled  |    7
================================================================
  Total: 3 sessions
================================================================
```

## 7. Interaction Notes

- The operator types `a` to grant a standing
  approval.
- The operator types `aios rules list` to view the
  rules.
- The operator types `aios rules remove <id>` to
  remove a rule.
- The operator types `aios sessions list` to view
  the past sessions.
- The operator types `aios session show <id>` to
  view a session.
