# aios-approvals — Technical Requirements Document

## 1. Purpose

This document states the technical approach for the
aios-approvals module.

## 2. Scope

The scope of this document is the aios-approvals
module. The module is a Go package and a service.

## 3. Reference Documents

- aios-approvals PRD. See
  `docs/modules/aios-approvals/PRD.md`.
- aios-approvals SRS. See
  `docs/modules/aios-approvals/SRS.md`.
- aios-core TRD. See `docs/modules/aios-core/TRD.md`.
- aios-policy TRD. See `docs/modules/aios-policy/TRD.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| goroutines + channels + errgroup | The Go concurrency primitives for async I/O. |
| Event | An goroutines + channels (stdlib) + golang.org/x/sync/errgroup primitive for notification. |

## 5. Architecture

The aios-approvals module has four components.

- `internal/approvals/schema` — the approval schemas.
- `internal/approvals/service` — the approval service.
- `internal/approvals/binding` — the binding validator.
- `internal/approvals/ttl` — the expiration timer.

The four components depend on aios-core and
aios-policy. The components depend on each other in
this order. The schema has no dependencies. The
binding has no dependencies. The service depends on
the schema, the binding, the audit log, and the TTL.
The TTL is part of the service.

## 6. Components

### 6.1 Schema

The schema component defines the `Approval` and the
`ApprovalState` classes. The component uses struct + go-playground/validator (or hand-rolled tag validation).

### 6.2 Binding

The binding component validates the binding. The
component compares the `bound_to` fields of the
payload with the current context. The component
returns true when the fields match. The component
returns false when the fields do not match.

### 6.3 Service

The service component manages the approvals. The
component uses a dictionary for the in-memory state.
The component uses the audit log for persistence.
The component uses `a buffered channel (chan struct{})` for notification.

The component supports the `request`, `grant`,
`deny`, `wait`, and `get` methods.

The `request` method creates a new approval. The
method writes the audit row. The method returns the
approval id.

The `grant` method sets the state to granted. The
method validates the binding. The method writes the
audit row. The method sets the event.

The `deny` method sets the state to denied. The
method writes the audit row. The method sets the
event.

The `wait` method blocks until the event is set.
The method returns the decision.

The `get` method returns the approval.

### 6.4 TTL

The TTL component uses an `a goroutine started with `go func()`` to schedule
the expiration. The task waits for the TTL. The task
sets the state to expired. The task writes the audit
row.

## 7. Data Structures

### 7.1 Approval

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `id` | string | yes | The unique identifier. |
| `state` | ApprovalState | yes | The current state. |
| `payload` | ApprovalPayload | yes | The approval payload. |
| `actor` | string | yes | The actor that requested the approval. |
| `created_at` | string | yes | The ISO 8601 timestamp of the request. |
| `updated_at` | string | yes | The ISO 8601 timestamp of the update. |
| `ttl` | int | no | The TTL in seconds. |

### 7.2 ApprovalState

| Value | Description |
| --- | --- |
| PENDING | The approval is waiting for a decision. |
| GRANTED | The approval is granted. |
| DENIED | The approval is denied. |
| EXPIRED | The approval expired. |

## 8. Interfaces

### 8.1 ApprovalService

```go
type ApprovalService:
    """The approval service."""

    def __init__(self, audit_log: AuditLog) -> nil:
        """Create the approval service."""

    func request(
        self,
        payload: ApprovalPayload,
        actor: str,
    ) -> str:
        """Request an approval.

        Args:
            payload: The approval payload.
            actor: The actor that requested the approval.

        Returns:
            The approval id.
        """

    func grant(self, approval_id: str, actor: str) -> nil:
        """Grant an approval.

        Args:
            approval_id: The approval id.
            actor: The actor that granted the approval.
        """

    func deny(self, approval_id: str, actor: str) -> nil:
        """Deny an approval.

        Args:
            approval_id: The approval id.
            actor: The actor that denied the approval.
        """

    func wait(self, approval_id: str) -> Approval:
        """Wait for an approval decision.

        Args:
            approval_id: The approval id.

        Returns:
            The approval.
        """
```

## 9. Algorithms

### 9.1 Binding Check

The binding check algorithm works as follows.

1. Compare the `task` field of the `bound_to` with
   the current task id. When the fields do not
   match, return false.
2. Compare the `tool` field of the `bound_to` with
   the current tool name. When the fields do not
   match, return false.
3. Compare the `actor` field of the `bound_to` with
   the current actor. When the fields do not match,
   return false.
4. Compare the `resource` field of the `bound_to`
   with the current resource. When the fields do
   not match, return false.
5. Return true.

### 9.2 Expiration Timer

The expiration timer algorithm works as follows.

1. When the approval is created with a TTL, start an
   `a goroutine started with `go func()`` that waits for the TTL.
2. When the task wakes up, check the state.
3. When the state is PENDING, set the state to
   EXPIRED.
4. Write the audit row.
5. Set the event.

## 10. Dependencies

### 10.1 Runtime Dependencies

- Go 1.23
- aios-core
- aios-policy

### 10.2 Module Dependencies

The aios-approvals module depends on the following
internal modules.

- aios-core
- aios-policy

The aios-approvals module is depended on by the
following internal modules.

- aios-runtime
- aios-cli

## 11. Deployment

The module compiles into the monorepo under `internal/<module>/`. The
package name is `github.com/<org>/aios/internal/approvals`. The module can run
as an in-process service. The module can run as a
remote service through an HTTP interface.

## 12. Risks and Mitigations

| Risk | Impact | Probability | Mitigation |
| --- | --- | --- | --- |
| The approval is granted for the wrong task. | High | Low | The binding check blocks the grant. The binding check has a test case for every field. |
| The approval never expires. | Medium | Low | The TTL timer is started on creation. The TTL timer is unit tested. |
| The waiter is blocked forever. | High | Low | The TTL timer sets the event. The wait method has a timeout. |
