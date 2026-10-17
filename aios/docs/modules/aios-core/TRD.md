# aios-core — Technical Requirements Document

## 1. Purpose

This document states the technical approach for the
aios-core module. The document describes the data
structures, the components, the algorithms, and the
dependencies.

## 2. Scope

The scope of this document is the aios-core module.
The module is a Go package. The module has no
external services.

## 3. Reference Documents

- aios-core PRD. See `docs/modules/aios-core/PRD.md`.
- aios-core SRS. See `docs/modules/aios-core/SRS.md`.
- Project coding standard. See `docs/standards/CODING-STANDARD.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| struct + go-playground/validator (or hand-rolled tag validation) | The Go struct validator used for typed cross-module records. |
| SQLite | A file-based relational database engine. |
| SHA-256 | A cryptographic hash function that produces a 256-bit output. |
| ISO 8601 | A standard format for date and time strings. |
| JSON | The JavaScript Object Notation for data exchange. |

## 5. Architecture

The aios-core module has four components.

- `internal/core/schemas` — the typed data contracts.
- `internal/core/audit` — the audit log with hash chaining.
- `internal/core/state` — the state machine for task
  transitions.
- `internal/core/redact` — the redaction layer for
  secrets.

The four components depend on each other in this
order. The redaction layer depends on the schemas.
The audit log depends on the redaction layer and the
schemas. The state machine depends on the audit log
and the schemas.

## 6. Components

### 6.1 Schemas

The schemas component uses struct + go-playground/validator (or hand-rolled tag validation). The base type is
`AiosBaseModel`. The type enables strict validation.
The type rejects extra fields. The type orders the
keys for deterministic serialization.

The schemas component provides the following classes.

- `TaskPlan` — the top-level plan for a task.
- `AgentSpec` — the specification for one agent.
- `WorkflowSpec` — the specification for a workflow.
- `ScheduleSpec` — the specification for a schedule.
- `DecisionTrace` — the record of one decision.
- `Envelope` — the wrapper for a tool call.
- `ApprovalPayload` — the payload for an approval.
- `ModelCapability` — the capabilities of a model.
- `ToolCall` — the request for one tool call.
- `ToolResult` — the result of one tool call.

### 6.2 Audit Log

The audit log component uses SQLite. The component
opens the database file in WAL mode. WAL mode allows
concurrent reads with one write.

The component computes the hash chain. The hash of a
row is the SHA-256 of the previous hash concatenated
with the current payload. The concatenation uses a
single newline as a separator.

The component supports the idempotency key. The
component rejects a write when the key matches an
existing row. The component returns the existing row.

The component supports the redaction layer. The
component runs the redaction layer on the payload
before the write.

### 6.3 State Machine

The state machine component controls the legal
transitions. The component uses a transition table.
The table is a dictionary that maps the current state
to a set of allowed next states.

The component writes an audit row on every transition.
The row includes the previous state, the new state,
and the task id.

The component supports the resume operation. The
resume operation sets the task state to RUNNING. The
operation writes the resume marker.

The component supports the cancel operation. The
cancel operation sets the task state to CANCELED from
any state.

### 6.4 Redaction

The redaction component uses a pattern list. The
pattern list contains regular expressions. The
component matches the patterns against the payload
values.

The component supports the following patterns.

- API key pattern: `sk-[A-Za-z0-9]{20,}`
- Bearer token pattern: `Bearer [A-Za-z0-9._-]{20,}`
- Password pattern: `password=[^\s&]+`
- Private key pattern: `-----BEGIN [A-Z ]+PRIVATE KEY-----`
- Session token pattern: `session_token=[^\s&]+`

The component replaces the matched value with the
placeholder `<REDACTED:type:hash>`. The hash is the
SHA-256 of the matched value, truncated to the first
8 characters.

The component preserves the field structure. The
component does not remove fields. The component only
replaces values.

The component preserves the approval id. The
approval id is a UUID. The component does not match
the UUID pattern. The component keeps the approval
id unchanged.

## 7. Data Structures

### 7.1 TaskPlan

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `id` | string | yes | The unique identifier of the plan. |
| `goal` | string | yes | The goal of the plan. |
| `tasks` | list of TaskSpec | yes | The tasks in the plan. |
| `tools` | list of string | no | The tools the plan needs. |
| `model` | string | no | The model the plan uses. |
| `budget` | Budget | no | The budget the plan consumes. |

### 7.2 AgentSpec

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `id` | string | yes | The unique identifier of the agent. |
| `role` | string | yes | The role of the agent. |
| `tools` | list of string | yes | The tools the agent uses. |
| `model` | string | yes | The model the agent uses. |
| `system_prompt` | string | no | The system prompt for the agent. |

### 7.3 DecisionTrace

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `id` | string | yes | The unique identifier of the trace. |
| `agent` | string | yes | The agent that made the decision. |
| `input` | dict | yes | The input to the decision. |
| `output` | dict | yes | The output of the decision. |
| `reason` | string | yes | The reason for the decision. |
| `timestamp` | string | yes | The ISO 8601 timestamp of the decision. |

### 7.4 AuditRow

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `id` | string | yes | The unique identifier of the row. |
| `ts` | string | yes | The ISO 8601 timestamp of the write. |
| `task_id` | string | yes | The task id of the row. |
| `type` | string | yes | The type of the row. |
| `actor` | string | yes | The actor that wrote the row. |
| `payload` | string | yes | The JSON-serialized payload. |
| `prev_hash` | string | yes | The SHA-256 hash of the previous row. |
| `hash` | string | yes | The SHA-256 hash of the current row. |
| `idempotency_key` | string | yes | The idempotency key of the write. |
| `redaction` | string | no | The JSON-serialized redaction markers. |

### 7.5 TaskState

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `task_id` | string | yes | The unique identifier of the task. |
| `state` | string | yes | The current state. |
| `updated_at` | string | yes | The ISO 8601 timestamp of the update. |
| `attempt` | int | yes | The current attempt number. |

## 8. Interfaces

### 8.1 AuditLog

```go
type AuditLog:
    """The append-only audit log with hash chaining."""

    def __init__(self, db_path: str) -> nil:
        """Open the audit log at the given path.

        Args:
            db_path: The path to the SQLite database file.
        """

    def append(
        self,
        task_id: str,
        type: str,
        actor: str,
        payload: dict,
        idempotency_key: str,
    ) -> AuditRow:
        """Append a row to the audit log.

        Args:
            task_id: The task id for the row.
            type: The type of the row.
            actor: The actor that wrote the row.
            payload: The payload for the row.
            idempotency_key: The key that prevents duplicate writes.

        Returns:
            The appended row.
        """

    def get_by_task(self, task_id: str) -> list[AuditRow]:
        """Get the rows for a task in time order."""

    def get_by_approval(self, approval_id: str) -> list[AuditRow]:
        """Get the rows that include the approval id."""

    def verify_chain(self) -> bool:
        """Verify the hash chain from the first row to the last row."""
```

### 8.2 TaskStateMachine

```go
type TaskStateMachine:
    """The state machine for the task lifecycle."""

    def __init__(self, audit_log: AuditLog) -> nil:
        """Create the state machine.

        Args:
            audit_log: The audit log for the state machine.
        """

    def transition(self, task_id: str, target: str) -> TaskState:
        """Move the task to the target state.

        Args:
            task_id: The task id.
            target: The target state.

        Returns:
            The new state.
        """

    def resume(self, task_id: str) -> TaskState:
        """Move the task to the RUNNING state."""

    def cancel(self, task_id: str) -> TaskState:
        """Move the task to the CANCELED state."""
```

### 8.3 Redaction

```go
type Redaction:
    """The redaction layer for secrets."""

    def __init__(self, patterns: list[Pattern] | nil = nil) -> nil:
        """Create the redaction layer.

        Args:
            patterns: The patterns to match. Uses the default
                patterns when nil.
        """

    def redact(self, payload: dict) -> tuple[dict, list[RedactionMarker]]:
        """Redact the secrets in the payload.

        Args:
            payload: The payload to redact.

        Returns:
            The redacted payload and the list of redaction markers.
        """
```

## 9. Algorithms

### 9.1 Hash Chain

The hash chain algorithm works as follows.

1. The first row has `prev_hash` set to the SHA-256
   of the empty string. The value is
   `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`.
2. For each subsequent row, compute `prev_hash` as the
   `hash` field of the previous row.
3. Compute `hash` as the SHA-256 of the concatenation
   of `prev_hash`, `type`, `actor`, and the
   JSON-serialized payload. The concatenation uses a
   single newline as a separator.

### 9.2 Idempotent Write

The idempotent write algorithm works as follows.

1. Look up the idempotency key in the database.
2. When the key exists, return the existing row.
3. When the key does not exist, write the new row.
4. When the write fails because another process wrote
   the same key, look up the key and return the row
   written by the other process.

### 9.3 Legal Transitions

The legal transitions algorithm uses a transition
table. The table is a dictionary. The keys are the
current states. The values are sets of allowed next
states.

| Current State | Allowed Next States |
| --- | --- |
| PENDING | RUNNING, CANCELED |
| RUNNING | AWAITING_APPROVAL, PAUSED, SUCCEEDED, FAILED, CANCELED |
| AWAITING_APPROVAL | RUNNING, FAILED, CANCELED |
| PAUSED | RUNNING, CANCELED |
| SUCCEEDED | (terminal) |
| FAILED | (terminal) |
| CANCELED | (terminal) |

## 10. Dependencies

### 10.1 Runtime Dependencies

- Go 1.23
- struct + go-playground/validator (or hand-rolled tag validation) 2.x

### 10.2 Development Dependencies

- go test
- go test -cover
- go vet
- gofmt + go vet

### 10.3 Module Dependencies

The aios-core module depends on the following internal
modules.

- nil. The aios-core module is the foundation module.
  No other module is at a lower level.

The aios-core module is depended on by the following
internal modules.

- aios-policy
- aios-runtime
- aios-models
- aios-tools
- aios-sandbox
- aios-egress
- aios-approvals
- aios-ask
- aios-planner
- aios-benchmark
- aios-report
- aios-cli

## 11. Deployment

The module compiles into the monorepo under `internal/<module>/`. The
package name is `github.com/<org>/aios/internal/core`. The module is not a
standalone service. The module is imported by the
other modules.

The audit log database file is created on the first
write. The default path is `./var/audit.db`. The
path is configurable through the `AIOS_AUDIT_DB`
environment variable.

## 12. Risks and Mitigations

| Risk | Impact | Probability | Mitigation |
| --- | --- | --- | --- |
| The audit log database file is corrupted. | High | Low | The hash chain detects corruption. The database is backed up on a regular schedule. |
| The struct + go-playground/validator (or hand-rolled tag validation) version changes. | Medium | Medium | The module pins the struct + go-playground/validator (or hand-rolled tag validation) version in the requirements file. |
| The redaction patterns miss a new secret type. | High | Medium | The redaction layer supports custom patterns. The module documents how to add new patterns. |
| The state machine has a race condition. | High | Low | The state machine uses a lock to serialize transitions. |
