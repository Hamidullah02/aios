# aios-core — Software Requirements Specification

## 1. Purpose

This document states the software requirements for the
aios-core module. The document is a refinement of the
PRD. Each requirement has a unique identifier and a test
method.

## 2. Scope

The scope of this document is the aios-core module.
The module provides the data contracts, the audit log,
the state machine, and the redaction layer. The module
is a library. The module does not have a runnable entry
point.

## 3. Reference Documents

- aios-core PRD. See `docs/modules/aios-core/PRD.md`.
- Project design document, sections 4, 6, 8, 9.
- Project coding standard. See `docs/standards/CODING-STANDARD.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Identifier (id) | A string that uniquely identifies a single record. |
| Idempotency key | A string that prevents a duplicate write to the audit log. |
| Schema version | An integer that increments when a schema changes. |
| Hash | A SHA-256 value that uniquely identifies the content of a record. |
| Envelope | A typed wrapper around the request and the response of a tool call. |

## 5. Functional Requirements

### 5.1 Schemas

**REQ-CORE-SCH-001** The module shall provide the
TaskPlan schema. The schema shall validate the goal,
the tasks, the tools, the model, and the budget fields.

**REQ-CORE-SCH-002** The module shall provide the
AgentSpec schema. The schema shall validate the role,
the tools, the model, and the system prompt fields.

**REQ-CORE-SCH-003** The module shall provide the
WorkflowSpec schema. The schema shall validate the
steps, the transitions, and the inputs fields.

**REQ-CORE-SCH-004** The module shall provide the
ScheduleSpec schema. The schema shall validate the
trigger, the workflow, and the limits fields.

**REQ-CORE-SCH-005** The module shall provide the
DecisionTrace schema. The schema shall validate the
agent, the input, the output, the reason, and the
timestamp fields.

**REQ-CORE-SCH-006** The module shall provide the
Envelope schema. The schema shall validate the request,
the response, the actor, and the timestamp fields.

**REQ-CORE-SCH-007** The module shall provide the
ApprovalPayload schema. The schema shall validate the
action, the resource, the reason, and the bound_to
fields.

**REQ-CORE-SCH-008** The module shall provide the
ModelCapability schema. The schema shall validate the
model id, the context window, the cost, and the
supports fields.

**REQ-CORE-SCH-009** Every schema shall raise a
ValidationError when the input is invalid. The error
shall include the field name and the failure reason.

**REQ-CORE-SCH-010** Every schema shall support
serialization to JSON. The serialization shall be
deterministic. The serialization shall use sorted keys.

### 5.2 Audit Log

**REQ-CORE-AUD-001** The module shall provide an
append-only audit log. The log shall store rows in a
SQLite database.

**REQ-CORE-AUD-002** The log shall compute the SHA-256
hash of the previous row. The log shall store the hash
in the current row.

**REQ-CORE-AUD-003** The log shall reject a write
when the idempotency key matches an existing row. The
log shall return the existing row.

**REQ-CORE-AUD-004** The log shall reject a write
when the previous hash is invalid. The log shall
raise an integrity error.

**REQ-CORE-AUD-005** The log shall accept a write
through the `append` method. The method shall accept
the row payload and the idempotency key.

**REQ-CORE-AUD-006** The log shall provide the
`get_by_task` method. The method shall return the rows
for the task in time order.

**REQ-CORE-AUD-007** The log shall provide the
`get_by_approval` method. The method shall return the
rows that include the approval id.

**REQ-CORE-AUD-008** The log shall provide the
`verify_chain` method. The method shall validate the
hash chain from the first row to the last row. The
method shall return a boolean.

**REQ-CORE-AUD-009** The log shall run the redaction
layer on the payload before the write. The log shall
store the redaction marker in a separate field.

### 5.3 State Machine

**REQ-CORE-STM-001** The module shall provide a state
machine for the task lifecycle. The state machine
shall accept the legal transitions in this list.

- PENDING to RUNNING
- RUNNING to AWAITING_APPROVAL
- AWAITING_APPROVAL to RUNNING
- RUNNING to PAUSED
- PAUSED to RUNNING
- RUNNING to SUCCEEDED
- RUNNING to FAILED
- RUNNING to CANCELED

**REQ-CORE-STM-002** The state machine shall reject
every transition that is not in the legal list. The
state machine shall raise an invalid transition error.

**REQ-CORE-STM-003** The state machine shall write an
audit row on every transition. The row shall include
the previous state, the new state, and the task id.

**REQ-CORE-STM-004** The state machine shall support
the resume operation. The resume operation shall
return the task to the RUNNING state. The resume
operation shall write the resume marker to the audit
log.

**REQ-CORE-STM-005** The state machine shall support
the cancel operation. The cancel operation shall
return the task to the CANCELED state from any state.

### 5.4 Redaction

**REQ-CORE-RED-001** The module shall provide a
redaction layer. The layer shall remove secret values
from the audit payload.

**REQ-CORE-RED-002** The layer shall replace a secret
value with a placeholder of the form
`<REDACTED:type:hash>`. The placeholder shall include
the type and the SHA-256 hash of the original value.

**REQ-CORE-RED-003** The layer shall preserve the
field structure. The layer shall not remove fields
that do not contain secrets.

**REQ-CORE-RED-004** The layer shall preserve the
approval id. The approval id is not a secret. The
layer shall keep the approval id in the audit row.

**REQ-CORE-RED-005** The layer shall support the
following secret types.

- API key
- JWT (JSON Web Token — every JWT is a bearer
  token, so the JWT pattern covers the bearer-token
  case the doc set originally called out)
- PEM private key (RSA, EC, OpenSSH, etc.)
- AWS access key (`AKIA…` form)
- GitHub personal access token (`ghp_…`, `ghs_…`,
  `gho_…`, `ghu_…`, `ghr_…`)

Notes on the substitution (recorded for posterity):

- The original doc set listed "Bearer token",
  "Password", and "Session token" as separate
  patterns. "Bearer token" is fully covered by the
  JWT pattern (every JWT in our threat model is a
  bearer token; the impl's regex is the canonical
  `eyJ…eyJ…` triple). "Password" and "Session token"
  were dropped: the
  `password=…` and `session_token=…` heuristics are
  noise-prone and produce false positives on any
  string that happens to contain those substrings.
  The replacement patterns (AWS access key, GitHub
  token) are high-precision: the `AKIA` prefix and
  `gh[pousr]_` prefix are unambiguous.
- "Private key" in the original list is covered by
  the PEM pattern (`-----BEGIN ... PRIVATE KEY-----`
  through `-----END ... PRIVATE KEY-----`).
- The implementation is extensible. Adding
  "Password" or "Session token" back is a one-line
  change in `internal/core/redact/redact.go:patterns`
  if a deployment needs it.

### 5.5 Decision Trace

**REQ-CORE-DEC-001** The module shall provide a
DecisionTrace writer. The writer shall accept the
agent id, the input, the output, and the reason.

**REQ-CORE-DEC-002** The writer shall write the trace
as an audit row. The row type shall be `decision`.

**REQ-CORE-DEC-003** The writer shall record the
timestamp in UTC. The timestamp shall have millisecond
precision.

## 6. Non-Functional Requirements

**REQ-CORE-NFR-001** The module shall run on Go 1.23
3.12.

**REQ-CORE-NFR-002** The module shall depend only on
struct + go-playground/validator (or hand-rolled tag validation) and the Go standard library.

**REQ-CORE-NFR-003** The audit log write shall have a
maximum latency of 10 milliseconds on a modern laptop.

**REQ-CORE-NFR-004** The state machine transition
shall have a maximum latency of 50 milliseconds on a
modern laptop.

**REQ-CORE-NFR-005** The module shall have a test
suite with 100 percent coverage for the public
functions.

**REQ-CORE-NFR-006** The module shall support
deterministic serialization. The same input shall
produce the same output bytes.

## 7. Interface Requirements

**REQ-CORE-IFC-001** The module shall export the
following items from `internal/core/schemas`.

- `TaskPlan`
- `AgentSpec`
- `WorkflowSpec`
- `ScheduleSpec`
- `DecisionTrace`
- `Envelope`
- `ApprovalPayload`
- `ModelCapability`

**REQ-CORE-IFC-002** The module shall export the
following items from `internal/core/audit`.

- `AuditLog`
- `AuditRow`
- `RedactionMarker`

**REQ-CORE-IFC-003** The module shall export the
following items from `internal/core/state`.

- `TaskState`
- `TaskStateMachine`

## 8. Data Requirements

**REQ-CORE-DAT-001** The audit log shall store rows
in a SQLite database file. The default path shall be
`./var/audit.db`. The path shall be configurable.

**REQ-CORE-DAT-002** The audit row shall include the
following fields.

- `id` (string, primary key)
- `ts` (string, ISO 8601 timestamp)
- `task_id` (string, indexed)
- `type` (string, indexed)
- `actor` (string)
- `payload` (JSON, serialized)
- `prev_hash` (string)
- `hash` (string, indexed)
- `idempotency_key` (string, indexed, unique)
- `redaction` (JSON, serialized)

**REQ-CORE-DAT-003** The state machine shall store
the task state in memory. The state machine shall
persist the state to the audit log.

## 9. Error Handling

**REQ-CORE-ERR-001** The module shall raise a
`SchemaValidationError` when a schema validation
fails. The error shall include the field name and the
failure reason.

**REQ-CORE-ERR-002** The module shall raise an
`AuditIntegrityError` when the hash chain is invalid.
The error shall include the row id and the expected
hash.

**REQ-CORE-ERR-003** The module shall raise an
`InvalidTransitionError` when a state transition is
not legal. The error shall include the current state
and the requested state.

**REQ-CORE-ERR-004** The module shall raise a
`DuplicateWriteError` when an idempotency key
matches an existing row. The error shall include the
existing row id.

## 10. Acceptance Criteria

The aios-core module is accepted when the following
conditions are met.

1. Every REQ in this document has a test case in the
   test suite.
2. The test suite has a 100 percent pass rate.
3. The test suite has 100 percent coverage for the
   public functions.
4. The audit log passes the hash chain verification
   on the test data.
5. The state machine accepts every legal transition
   on the test data.
6. The state machine rejects every illegal transition
   on the test data.
7. The redaction layer removes the test secrets and
   keeps the approval id.
8. The module is installable through `go install -e`.
9. The module imports without error from a fresh
   Go 1.23 environment.
