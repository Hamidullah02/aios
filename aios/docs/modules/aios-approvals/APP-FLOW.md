# aios-approvals — Application Flow

## 1. Purpose

This document describes the application flow for the
aios-approvals module.

## 2. Scope

The scope of this document is the aios-approvals
module.

## 3. Reference Documents

- aios-approvals PRD. See
  `docs/modules/aios-approvals/PRD.md`.
- aios-approvals SRS. See
  `docs/modules/aios-approvals/SRS.md`.
- aios-approvals TRD. See
  `docs/modules/aios-approvals/TRD.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Worker | The code that requests the approval. |
| Operator | The user that grants or denies the approval. |

## 5. User Flow

The operator flow runs as follows.

1. The CLI shows the approval request to the
   operator.
2. The operator reviews the request.
3. The operator types `y` to grant or `n` to deny.
4. The CLI calls the service.
5. The service updates the state.
6. The service notifies the worker.

## 6. System Flow

### 6.1 Request Flow

The request flow runs as follows.

1. The worker calls `service.request` with the
   payload and the actor.
2. The service creates the approval.
3. The service writes the audit row.
4. The service starts the TTL timer.
5. The service returns the approval id.

### 6.2 Grant Flow

The grant flow runs as follows.

1. The operator calls `service.grant` with the
   approval id and the actor.
2. The service validates the binding.
3. The service updates the state to granted.
4. The service writes the audit row.
5. The service sets the event.

### 6.3 Deny Flow

The deny flow runs as follows.

1. The operator calls `service.deny` with the
   approval id and the actor.
2. The service updates the state to denied.
3. The service writes the audit row.
4. The service sets the event.

### 6.4 Wait Flow

The wait flow runs as follows.

1. The worker calls `service.wait` with the
   approval id.
2. The service blocks on the event.
3. When the state changes, the service wakes up.
4. The service returns the approval.

### 6.5 Expiration Flow

The expiration flow runs as follows.

1. The TTL timer wakes up.
2. The service checks the state.
3. When the state is pending, the service updates
   the state to expired.
4. The service writes the audit row.
5. The service sets the event.

## 7. Error Flows

### 7.1 Approval Not Found

The approval not found flow runs as follows.

1. The caller passes an unknown approval id.
2. The service raises `ApprovalNotFoundError`.
3. The caller catches the error.
4. The caller reports the error.

### 7.2 Binding Mismatch

The binding mismatch flow runs as follows.

1. The grant call has a binding mismatch.
2. The service raises `BindingMismatchError`.
3. The caller catches the error.
4. The caller reports the error.

### 7.3 Invalid State Transition

The invalid state transition flow runs as follows.

1. The caller tries to grant a denied approval.
2. The service raises
   `InvalidStateTransitionError`.
3. The caller catches the error.
4. The caller reports the error.

## 8. Sequence Diagrams

### 8.1 Request and Grant Sequence

The request and grant sequence has the steps below.

1. The worker calls `service.request`.
2. The service writes the audit row.
3. The service returns the approval id.
4. The worker calls `service.wait`.
5. The operator reviews the request.
6. The operator calls `service.grant`.
7. The service updates the state.
8. The service writes the audit row.
9. The service sets the event.
10. The worker wakes up.
11. The service returns the approval.
