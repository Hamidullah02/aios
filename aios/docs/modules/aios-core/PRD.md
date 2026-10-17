# aios-core — Product Requirements Document

## 1. Purpose

This document states the product requirements for the
aios-core module. The aios-core module is the foundation
module of the project. The module defines the typed data
contracts and the audit log. Every other module depends
on aios-core.

## 2. Scope

The aios-core module is in scope for this document.

- Typed schemas for the data contracts.
- The audit log with hash-chained records.
- The state machine for task execution.
- The redaction layer for secrets in audit rows.
- The DecisionTrace writer.

The aios-core module is not in scope for this document.

- The orchestrator. The orchestrator is in aios-runtime.
- The policy gateway. The gateway is in aios-policy.
- The model adapters. The adapters are in aios-models.
- The user interface. The interface is in aios-cli.

## 3. Reference Documents

- Project design document, sections 4, 6, 8, 9.
- ASD-STE100 writing standard. See `docs/standards/STE-STYLE-GUIDE.md`.
- Project coding standard. See `docs/standards/CODING-STANDARD.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Schema | A typed data structure that validates the shape and the content of a value. |
| Audit log | The append-only log of every state transition and tool call. |
| State machine | The component that controls the legal transitions between task states. |
| Decision trace | A typed record of one decision the orchestrator makes. |
| Redaction | The replacement of a secret value with a placeholder in an audit row. |
| Hash chain | A sequence of records where each record includes the hash of the previous record. |

## 5. User Personas

The aios-core module has no direct end users. The
consumers of the module are the developers of the other
modules. The user persona is a Go developer with
experience in struct + go-playground/validator (or hand-rolled tag validation) and SQLite.

## 6. User Stories

A developer wants to validate a task plan. The developer
imports the TaskPlan type from aios-core. The developer
passes a dictionary to the class. The type returns a
typed object or raises a validation error.

A developer wants to record a state transition. The
developer calls the state machine with a transition
request. The state machine validates the transition. The
state machine writes the audit row. The state machine
returns the new state.

A developer wants to find every audit row for one task.
The developer passes the task identifier to the audit
log. The audit log returns a list of rows in time order.

## 7. Functional Requirements Summary

The aios-core module provides the following functions.

- Typed schemas for the data contracts. The schemas
  validate the input. The schemas reject invalid input.
- An audit log that appends rows in hash-chained order.
  The audit log prevents duplicate writes by the
  idempotency key.
- A state machine that controls task transitions. The
  state machine is the only component that changes the
  task state.
- A redaction layer that removes secrets from audit
  rows. The redaction layer writes a redaction marker
  when it removes a secret.
- A DecisionTrace writer that writes a typed trace on
  every decision.

## 8. Non-Functional Requirements Summary

The aios-core module has the following non-functional
requirements.

- The module runs on Go 1.23.
- The module depends only on the struct + go-playground/validator (or hand-rolled tag validation) library and
  the Go standard library.
- The module persists audit rows in a SQLite database.
- The module has 100 percent test coverage for the
  public functions.
- The module has a maximum latency of 10 milliseconds
  for a single audit row write.
- The module has a maximum latency of 50 milliseconds
  for a single state transition.

## 9. Success Criteria

The aios-core module is successful when the following
conditions are met.

- All schemas validate the test cases in the test suite.
- The audit log writes rows in hash-chained order.
- The audit log rejects duplicate writes.
- The state machine accepts every legal transition.
- The state machine rejects every illegal transition.
- The redaction layer removes the test secrets from the
  test audit rows.
- The test suite has 100 percent pass rate.
- The test suite has 100 percent coverage for the
  public functions.

## 10. Out of Scope

The aios-core module does not include the following
functions. The functions are in other modules.

- The user interface for the audit log.
- The remote access to the audit log.
- The replication of the audit log across hosts.
- The retention policy for the audit log.

## 11. Open Questions

The following questions are open. The project will
resolve the questions before the implementation phase.

- Does the audit log support encryption at rest? The
  current design stores the audit log in clear text.
- Does the state machine support parallel transitions?
  The current design supports one transition at a time.
- Does the redaction layer support custom patterns? The
  current design supports a fixed set of patterns.
