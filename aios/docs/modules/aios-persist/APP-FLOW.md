# aios-persist — Application Flow

## 1. Purpose

This document describes the application flow for the
aios-persist module.

## 2. Scope

The scope of this document is the aios-persist module.
The module is a library.

## 3. Reference Documents

- aios-persist PRD. See
  `docs/modules/aios-persist/PRD.md`.
- aios-persist SRS. See
  `docs/modules/aios-persist/SRS.md`.
- aios-persist TRD. See
  `docs/modules/aios-persist/TRD.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Caller | The code that calls the storage. |

## 5. User Flow

The aios-persist module has no user-facing flow. The
user interacts through the CLI.

## 6. System Flow

### 6.1 Add Rule Flow

The add rule flow runs as follows.

1. The caller calls `storage.add` with the rule.
2. The storage validates the rule.
3. The storage appends the rule to the file.
4. The storage returns success.

### 6.2 Match Rule Flow

The match rule flow runs as follows.

1. The caller calls `storage.match` with the tool
   call.
2. The storage iterates over the rules.
3. The storage returns the first matching rule.
4. When no rule matches, the storage returns
   nil.

### 6.3 Handoff Put Flow

The handoff put flow runs as follows.

1. The caller calls `handoff.put` with the task
   id, the key, and the value.
2. The handoff creates the directory.
3. The handoff writes the file.
4. The handoff returns success.

### 6.4 Handoff Get Flow

The handoff get flow runs as follows.

1. The caller calls `handoff.get` with the task
   id and the key.
2. The handoff reads the file.
3. The handoff returns the value.

## 7. Error Flows

### 7.1 Rule Not Found

The rule not found flow runs as follows.

1. The caller passes a missing rule id.
2. The storage raises `RuleNotFoundError`.
3. The caller catches the error.
4. The caller reports the error.

### 7.2 Handoff Not Found

The handoff not found flow runs as follows.

1. The caller passes a missing key.
2. The handoff raises `HandoffNotFoundError`.
3. The caller catches the error.
4. The caller reports the error.

## 8. Sequence Diagrams

### 8.1 Match Sequence

The match sequence has the steps below.

1. The caller calls `storage.match`.
2. The storage iterates over the rules.
3. The storage returns the first matching rule.
