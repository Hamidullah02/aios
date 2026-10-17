# aios-approvals — Software Requirements Specification

## 1. Purpose

This document states the software requirements for the
aios-approvals module.

## 2. Scope

The scope of this document is the aios-approvals
module. The module is a library and a service.

## 3. Reference Documents

- aios-approvals PRD. See
  `docs/modules/aios-approvals/PRD.md`.
- aios-core SRS. See `docs/modules/aios-core/SRS.md`.
- aios-policy SRS. See `docs/modules/aios-policy/SRS.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Approval | A typed permission to perform an action. |
| State | The current state of an approval. |

## 5. Functional Requirements

### 5.1 Approval Service

**REQ-APR-SVC-001** The module shall provide the
`ApprovalService` class. The type shall accept the
audit log.

**REQ-APR-SVC-002** The service shall provide the
`request` method. The method shall accept the
approval payload and the actor. The method shall
return the approval id.

**REQ-APR-SVC-003** The service shall provide the
`grant` method. The method shall accept the
approval id and the actor. The method shall set the
state to granted.

**REQ-APR-SVC-004** The service shall provide the
`deny` method. The method shall accept the approval
id and the actor. The method shall set the state to
denied.

**REQ-APR-SVC-005** The service shall provide the
`wait` method. The method shall accept the approval
id. The method shall block until the state changes.
The method shall return the decision.

**REQ-APR-SVC-006** The service shall provide the
`get` method. The method shall accept the approval
id. The method shall return the approval.

### 5.2 Persistence

**REQ-APR-PER-001** The service shall persist the
approval in the audit log. The row type shall be
`approval`.

**REQ-APR-PER-002** The service shall write the
approval id, the state, the payload, the actor, and
the timestamp.

**REQ-APR-PER-003** The service shall write the
state transition as a separate audit row. The row
type shall be `approval_state`.

### 5.3 Notification

**REQ-APR-NTF-001** The service shall notify the
waiter when the state changes. The service shall
use an `a buffered channel (chan struct{})` for the notification.

**REQ-APR-NTF-002** The service shall support
multiple waiters for the same approval.

### 5.4 Binding

**REQ-APR-BND-001** The service shall validate the
binding. The binding is the `bound_to` field of the
payload. The binding shall match the current task,
tool, actor, and resource.

**REQ-APR-BND-002** When the binding does not match,
the service shall deny the action. The service
shall raise `BindingMismatchError`.

### 5.5 Expiration

**REQ-APR-EXP-001** The service shall support the
TTL. The TTL is a field of the payload. When the
TTL expires, the service shall set the state to
expired.

## 6. Non-Functional Requirements

**REQ-APR-NFR-001** The module shall run on Go 1.23
3.12.

**REQ-APR-NFR-002** The module shall have 100 percent
coverage for the public functions.

## 7. Interface Requirements

**REQ-APR-IFC-001** The module shall export the
following items from `aios.approvals`.

- `ApprovalService`
- `Approval`
- `ApprovalState`
- `BindingMismatchError`

## 8. Data Requirements

**REQ-APR-DAT-001** The approval schema shall
include the fields below.

- `id` (string, unique)
- `state` (string: pending, granted, denied,
  expired)
- `payload` (ApprovalPayload)
- `actor` (string)
- `created_at` (string, ISO 8601)
- `updated_at` (string, ISO 8601)
- `ttl` (int, optional, seconds)

## 9. Error Handling

**REQ-APR-ERR-001** The module shall raise
`ApprovalNotFoundError` when the approval id is
not found.

**REQ-APR-ERR-002** The module shall raise
`InvalidStateTransitionError` when the state
transition is not legal.

**REQ-APR-ERR-003** The module shall raise
`BindingMismatchError` when the binding does not
match.

## 10. Acceptance Criteria

1. Every REQ in this document has a test case.
2. The test suite has a 100 percent pass rate.
3. The test suite has 100 percent coverage for the
   public functions.
4. The service creates the approval.
5. The service grants the approval.
6. The service denies the approval.
7. The service notifies the waiter.
8. The service persists the approval.
9. The service validates the binding.
