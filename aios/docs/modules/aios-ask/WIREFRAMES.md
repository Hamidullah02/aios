# aios-ask — Wireframes

## 1. Purpose

This document describes the wireframes for the
aios-ask module. The module has one user-facing
surface: the question prompt in the CLI.

## 2. Scope

The scope of this document is the question prompt in
the CLI. The prompt is a text-based prompt on the
terminal.

## 3. Reference Documents

- aios-ask PRD. See `docs/modules/aios-ask/PRD.md`.
- aios-ask SRS. See `docs/modules/aios-ask/SRS.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Wireframe | A visual layout for a user-facing surface. |
| Prompt | A text request for a user decision. |

## 5. Screen List

The module has one screen: the question prompt.

## 6. Wireframes

### 6.1 Question Prompt

```
================================================================
  QUESTION
================================================================
  Task:    task-001
  Question id: q-1
================================================================
  Which database engine do you want to use?
  1. PostgreSQL (recommended for production)
  2. SQLite (recommended for development)
  3. MySQL
  4. Cancel
================================================================
  Timeout: 60s  Default: 1
================================================================
  Choice: _
```

### 6.2 Question Answered

```
================================================================
  ANSWERED
================================================================
  Question id: q-1
  Selected:    1 (PostgreSQL)
================================================================
```

### 6.3 Question Timed Out

```
================================================================
  TIMED OUT
================================================================
  Question id: q-1
  Default:     1 (PostgreSQL)
================================================================
```

## 7. Interaction Notes

- The operator types the number and presses Enter.
- The operator types `cancel` and presses Enter to
  cancel the question.
- The default option is used when the timeout
  expires.
