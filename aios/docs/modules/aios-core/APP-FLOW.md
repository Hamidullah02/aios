# aios-core — Application Flow

## 1. Purpose

This document describes the application flow for the
aios-core module. The document covers the data flow for
schemas, the audit log, the state machine, and the
redaction layer.

## 2. Scope

The scope of this document is the aios-core module.
The module has no user-facing surface. The flows in
this document are internal flows between components
of the module and between the module and the other
modules.

## 3. Reference Documents

- aios-core PRD. See `docs/modules/aios-core/PRD.md`.
- aios-core SRS. See `docs/modules/aios-core/SRS.md`.
- aios-core TRD. See `docs/modules/aios-core/TRD.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Caller | The code that calls a component in aios-core. |
| Component | A type or a function in aios-core. |
| Flow | A sequence of calls from a caller to a component. |

## 5. User Flow

The aios-core module has no user-facing flow. The
flows in this document are programming flows.

## 6. System Flow

### 6.1 Schema Validation Flow

The schema validation flow runs as follows.

1. The caller creates a dictionary with the candidate
   data.
2. The caller passes the dictionary to the schema
   constructor.
3. The constructor calls the struct + go-playground/validator (or hand-rolled tag validation) validator.
4. The validator returns success or raises a
   `SchemaValidationError`.
5. On success, the constructor returns the schema
   object.
6. The caller uses the schema object.

### 6.2 Audit Write Flow

The audit write flow runs as follows.

1. The caller calls `audit_log.append` with the task
   id, the type, the actor, the payload, and the
   idempotency key.
2. The component looks up the idempotency key in the
   database.
3. When the key exists, the component returns the
   existing row.
4. When the key does not exist, the component runs
   the redaction layer on the payload.
5. The component reads the hash of the previous row.
6. The component computes the hash of the current row.
7. The component writes the new row to the database.
8. The component returns the new row.

### 6.3 Audit Read Flow

The audit read flow runs as follows.

1. The caller calls `audit_log.get_by_task` with the
   task id.
2. The component executes a SQL query for the task id
   in time order.
3. The component returns the list of rows.
4. The caller processes the list of rows.

### 6.4 State Transition Flow

The state transition flow runs as follows.

1. The caller calls `state_machine.transition` with
   the task id and the target state.
2. The component reads the current state for the task.
3. The component checks the transition table for the
   target state.
4. When the target state is not in the table, the
   component raises an `InvalidTransitionError`.
5. When the target state is in the table, the
   component updates the state in memory.
6. The component writes an audit row for the
   transition.
7. The component returns the new state.

### 6.5 Resume Flow

The resume flow runs as follows.

1. The caller calls `state_machine.resume` with the
   task id.
2. The component reads the current state for the task.
3. When the current state is not PAUSED or
   AWAITING_APPROVAL, the component raises an
   `InvalidTransitionError`.
4. The component updates the state to RUNNING.
5. The component writes an audit row for the resume
   with the resume marker.
6. The component returns the new state.

### 6.6 Cancel Flow

The cancel flow runs as follows.

1. The caller calls `state_machine.cancel` with the
   task id.
2. The component reads the current state for the task.
3. When the current state is a terminal state, the
   component raises an `InvalidTransitionError`.
4. The component updates the state to CANCELED.
5. The component writes an audit row for the cancel.
6. The component returns the new state.

### 6.7 Redaction Flow

The redaction flow runs as follows.

1. The caller calls `redaction.redact` with the
   payload.
2. The component iterates over the values in the
   payload.
3. For each value, the component matches the
   redaction patterns.
4. When a pattern matches, the component replaces the
   value with the placeholder and records a
   redaction marker.
5. When no pattern matches, the component keeps the
   value unchanged.
6. The component returns the redacted payload and the
   list of redaction markers.

## 7. Error Flows

### 7.1 Schema Validation Error

The schema validation error flow runs as follows.

1. The caller passes invalid data to the schema
   constructor.
2. The constructor raises a `SchemaValidationError`.
3. The caller catches the error.
4. The caller reports the error to the user.

### 7.2 Audit Integrity Error

The audit integrity error flow runs as follows.

1. The component reads the previous hash from the
   database.
2. The component computes the expected hash from the
   previous row.
3. When the expected hash does not match the stored
   hash, the component raises an
   `AuditIntegrityError`.
4. The caller catches the error.
5. The caller reports the error and stops the
   pipeline.

### 7.3 Invalid Transition Error

The invalid transition error flow runs as follows.

1. The caller calls `state_machine.transition` with
   an illegal target.
2. The component raises an `InvalidTransitionError`.
3. The caller catches the error.
4. The caller reports the error and stops the task.

### 7.4 Duplicate Write Error

The duplicate write error flow runs as follows.

1. The caller calls `audit_log.append` with an
   idempotency key that already exists.
2. The component returns the existing row. The
   component does not raise an error.
3. The caller uses the existing row.

## 8. Sequence Diagrams

### 8.1 Audit Write Sequence

The audit write sequence has the steps below.

1. The caller calls `audit_log.append`.
2. The component calls the redaction layer.
3. The redaction layer returns the redacted payload
   and the markers.
4. The component reads the previous hash.
5. The component computes the new hash.
6. The component writes the row.
7. The component returns the row.

### 8.2 State Transition Sequence

The state transition sequence has the steps below.

1. The caller calls `state_machine.transition`.
2. The component reads the current state.
3. The component validates the target state.
4. The component updates the state in memory.
5. The component calls `audit_log.append` for the
   transition.
6. The component returns the new state.
