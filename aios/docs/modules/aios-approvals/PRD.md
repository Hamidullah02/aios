# aios-approvals — Product Requirements Document

## 1. Purpose

This document states the product requirements for the
aios-approvals module. The module is the approval
service. The service decides whether a tier-3 action
gets approved. The service persists the decision. The
service notifies the waiter.

## 2. Scope

The aios-approvals module is in scope for this document.

- The approval service.
- The approval decision.
- The approval persistence.
- The approval notification.

The aios-approvals module is not in scope for this
document.

- The policy gateway. The gateway is in aios-policy.
- The CLI. The CLI is in aios-cli.
- The orchestrator. The orchestrator is in aios-runtime.

## 3. Reference Documents

- Project design document, sections 7, 11.
- aios-core PRD. See `docs/modules/aios-core/PRD.md`.
- aios-policy PRD. See `docs/modules/aios-policy/PRD.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Approval | A permission to perform a tier-3 action. |
| Tier-3 action | A high-risk action that needs user approval. |
| Approval payload | The typed object that describes the action. |
| Approval id | The unique identifier of an approval. |
| Approval state | The state of an approval: pending, granted, denied, expired. |


## 5. Determinism

The whole point of aios is to make complex coding and large systems deterministic. The determinism stance of this module is **Bounded non-determinism**.

The operator is the only source of non-determinism. The rule table and the TTL are deterministic.

The module provides the following determinism guarantees.

- Approval decisions are pure functions of the rule table and the request.

- Rule table is content addressed.

- TTL is a pure function of the rule and the clock.
## 6. User Personas

The user persona is a Go developer with experience
in async programming and HTTP services.

## 7. User Stories

A developer wants to request an approval. The
developer passes the approval payload to the service.
The service creates the approval. The service
returns the approval id.

A developer wants to grant an approval. The developer
passes the approval id to the service. The service
sets the state to granted. The service notifies the
waiter.

A developer wants to deny an approval. The developer
passes the approval id to the service. The service
sets the state to denied. The service notifies the
waiter.

A developer wants to wait for an approval. The
developer passes the approval id to the service. The
service blocks the call until the approval state
changes.

## 8. Functional Requirements Summary

The aios-approvals module provides the following
functions.

- The approval service.
- The approval decision.
- The approval persistence.
- The approval notification.

## 9. Non-Functional Requirements Summary

- The module runs on Go 1.23.
- The module depends on aios-core and aios-policy.
- The module has 100 percent test coverage for the
  public functions.

## 10. Success Criteria

- The service creates the approval.
- The service grants the approval.
- The service denies the approval.
- The service notifies the waiter.
- The service persists the approval.
- The test suite has a 100 percent pass rate.

## 11. Out of Scope

- The user interface for the approval.
- The remote access to the approval service.
- The approval expiration policy.

## 12. Open Questions

- Does the service support approval expiration? The
  current design supports expiration through a TTL.
- Does the service support approval binding? The
  current design supports binding through
  `ApprovalPayload.bound_to`.
