# aios-policy — Product Requirements Document

## 1. Purpose

This document states the product requirements for the
aios-policy module. The module is the policy gateway.
The gateway decides whether a tool call is allowed,
denied, or requires an approval.

## 2. Scope

The aios-policy module is in scope for this document.

- The policy rule table.
- The policy gateway.
- The policy decision.
- The policy audit row.

The aios-policy module is not in scope for this
document.

- The approval service. The service is in aios-approvals.
- The ask-user tool. The tool is in aios-ask.
- The orchestrator. The orchestrator is in aios-runtime.

## 3. Reference Documents

- Project design document, sections 7, 11.
- aios-core PRD. See `docs/modules/aios-core/PRD.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Policy | A rule that decides the action for a tool call. |
| Rule | A tuple of match conditions and an action. |
| Action | The decision the gateway returns: allow, deny, or ask. |
| Match | A condition that selects a subset of tool calls. |
| Resource | The path or the URL the tool call targets. |


## 5. Determinism

The whole point of aios is to make complex coding and large systems deterministic. The determinism stance of this module is **Highly deterministic**.

Policy gateway is a pure function of the rule table and the tool call. No state, no clock.

The module provides the following determinism guarantees.

- Policy decisions are pure functions of inputs.

- Rule table is content addressed.

- Policy is immutable after the run starts.
## 6. User Personas

The user persona is a Go developer with experience
in rule engines. The user writes a policy file and
loads it into the gateway.

## 7. User Stories

A developer wants to deny all writes to a sensitive
directory. The developer adds a rule to the policy
file. The rule matches the path. The rule has the
action `deny`. The gateway denies writes to the path.

A developer wants to require an approval for a
specific tool. The developer adds a rule to the policy
file. The rule matches the tool name. The rule has the
action `ask`. The gateway returns the ask decision.

A developer wants to allow a read-only tool without an
approval. The developer adds a rule to the policy
file. The rule matches the tool name. The rule has the
action `allow`. The gateway returns the allow
decision.

## 8. Functional Requirements Summary

The aios-policy module provides the following
functions.

- A policy rule table that matches tool calls.
- A policy gateway that returns the decision for a
  tool call.
- A policy decision type that represents the result.
- A policy audit row that records the decision.

## 9. Non-Functional Requirements Summary

The aios-policy module has the following non-functional
requirements.

- The module runs on Go 1.23.
- The module depends only on aios-core and the Go
  standard library.
- The module has 100 percent test coverage for the
  public functions.
- The gateway decision has a maximum latency of 1
  millisecond.

## 10. Success Criteria

The aios-policy module is successful when the following
conditions are met.

- The rule table matches every test rule.
- The gateway returns the correct decision for every
  test case.
- The gateway writes the correct audit row.
- The test suite has a 100 percent pass rate.
- The test suite has 100 percent coverage for the
  public functions.

## 11. Out of Scope

The aios-policy module does not include the following
functions. The functions are in other modules.

- The user interface for the policy file.
- The remote access to the policy file.
- The policy file editor.

## 12. Open Questions

The following questions are open. The project will
resolve the questions before the implementation phase.

- Does the gateway support rule priorities? The
  current design uses first-match-wins.
- Does the gateway support wildcard rules? The current
  design uses glob patterns.
- Does the gateway support the `only` modifier? The
  current design does not support `only`.
