# aios-persist — Product Requirements Document

## 1. Purpose

This document states the product requirements for the
aios-persist module. The module is the session
persistence layer. The module covers three areas:
the always-allow rule storage, the session state
checkpoint, and the structured handoff between
tasks.

## 2. Scope

The aios-persist module is in scope for this
document.

- The always-allow rule storage.
- The session state checkpoint.
- The structured handoff.
- The session history.

The aios-persist module is not in scope for this
document.

- The audit log. The audit log is in aios-core.
- The resume handler. The handler is in
  aios-runtime.
- The approval service. The service is in
  aios-approvals.

## 3. Reference Documents

- Project design document, section 7.
- aios-core PRD. See `docs/modules/aios-core/PRD.md`.
- aios-approvals PRD. See
  `docs/modules/aios-approvals/PRD.md`.
- aios-runtime PRD. See `docs/modules/aios-runtime/PRD.md`.
- Claude Code settings.local.json reference.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Always-allow rule | A rule that grants standing approval for a tool call. |
| Session | A single run of the agent. |
| Checkpoint | The saved state of a session. |
| Handoff | The data passed between two tasks. |
| Artifact | A file or value produced by a task. |


## 5. Determinism

The whole point of aios is to make complex coding and large systems deterministic. The determinism stance of this module is **Highly deterministic**.

Rule storage, checkpoints, handoffs, and history are all content addressed. The clock is only used for ordering.

The module provides the following determinism guarantees.

- Rules are content addressed.

- Checkpoints are content addressed.

- Handoffs are content addressed.

- History is an append-only log with a hash chain.
## 6. User Personas

The user persona is the operator who grants a
standing approval. The operator wants the rule to
persist across sessions. The operator wants to
review the rules.

## 7. User Stories

An operator wants to grant a standing approval.
The operator types `y, always` at the approval
prompt. The CLI saves the rule. The rule is
applied to future sessions.

An operator wants to view the rules. The operator
runs `aios rules list`. The CLI prints the rules.

A planner wants to pass an artifact between tasks.
The planner calls the handoff API. The API saves
the artifact. The next task reads the artifact.

## 8. Functional Requirements Summary

The aios-persist module provides the following
functions.

- The always-allow rule storage.
- The session state checkpoint.
- The structured handoff.
- The session history.

## 9. Non-Functional Requirements Summary

- The module runs on Go 1.23.
- The module depends on aios-core and aios-policy.
- The module has 100 percent test coverage for the
  public functions.

## 10. Success Criteria

- The always-allow rule storage saves the rules.
- The session state checkpoint saves the session.
- The structured handoff passes the artifact.
- The session history lists the past sessions.
- The test suite has a 100 percent pass rate.

## 11. Out of Scope

- The user interface for the rules.
- The remote access to the rules.
- The cross-host session.
