# aios-cli — Product Requirements Document

## 1. Purpose

This document states the product requirements for the
aios-cli module. The module is the first-run CLI and
the approval daemon. The CLI provides the entry
point for the user. The daemon keeps the operator
in the loop for tier-3 actions.

## 2. Scope

The aios-cli module is in scope for this document.

- The first-run CLI.
- The approval daemon.
- The command parser.
- The error display.
- The operator UI, served by the `web` subcommand and by
  `run --web`. The UI is the surface an operator watches a
  run on and the surface a gated tool call is approved on,
  so it is part of the CLI rather than a separate product.

The aios-cli module is not in scope for this
document.

- The other modules. The other modules are imported
  by the CLI.

## 3. Reference Documents

- Project design document, section 16.
- aios-core PRD. See `docs/modules/aios-core/PRD.md`.
- aios-runtime PRD. See `docs/modules/aios-runtime/PRD.md`.
- aios-approvals PRD. See
  `docs/modules/aios-approvals/PRD.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| CLI | The command-line interface. |
| Daemon | A long-running process that handles requests. |
| Subcommand | A command under the main command. |
| Tier-3 action | A high-risk action that needs operator approval. |


## 5. Determinism

The whole point of aios is to make complex coding and large systems deterministic. The determinism stance of this module is **Bounded non-determinism**.

The operator and the wall clock are the only sources of non-determinism. The CLI is a pure function of the command and the config.

The module provides the following determinism guarantees.

- The CLI is replayable through the audit log.
## 6. User Personas

The user persona is an operator who needs to run the
agent. The operator may have a non-technical
background. The operator needs clear prompts and
clear errors.

## 7. User Stories

An operator wants to run a plan. The operator types
`aios run plan.yaml`. The CLI loads the plan. The
CLI runs the plan. The CLI prints the report.

An operator wants to approve a tier-3 action. The
CLI shows the approval prompt. The operator types
`y` to grant. The CLI sends the grant to the
service.

An operator wants to view a report. The operator
types `aios report task-001`. The CLI prints the
report.

An operator wants to start the daemon. The operator
types `aios daemon start`. The CLI starts the
daemon. The daemon listens for approval requests.

An operator wants to watch a run. The operator types
`aios run plan.yaml --web`. The CLI serves the operator
UI and prints a link that carries the session token. The
operator opens the link and sees the panels update as the
plan runs.

An operator wants to give the product something to do.
The operator opens the UI and types what they want done
in their own words. The CLI turns the sentence into a
plan, repairs the plan if it is wrong, and runs it. The
operator follows it on a task card. The operator does not
write a plan, a policy, or a configuration file.

An operator wants to approve a gated tool call from the
browser. The UI shows the pending approval with the action
and the target. The operator clicks "Allow once". The run
continues.

An operator wants to approve a gated tool call from
another terminal. The operator types `aios approve <id>`.
The CLI sends the grant to the daemon.

## 8. Functional Requirements Summary

The aios-cli module provides the following
functions.

- The first-run CLI with the `aios` command.
- The approval daemon.
- The command parser.
- The error display.
- The operator UI, behind a per-process session token.

## 9. Non-Functional Requirements Summary

- The module runs on Go 1.23.
- The module depends on every other module.
- The module has 100 percent test coverage for the
  public functions.
- The module is the only entry point for the user.

## 10. Success Criteria

- The CLI loads a plan.
- The CLI runs a plan.
- The CLI shows the approval prompt.
- The CLI sends the grant.
- The CLI prints the report.
- The CLI starts the daemon.
- The CLI starts the daemon.
- The operator UI refuses an unauthenticated request.
- A gated tool call can be approved from the browser.
- The test suite has a 100 percent pass rate.

## 11. Out of Scope

- The web interface for the CLI.
- The mobile interface for the CLI.
- The remote access to the CLI. The operator UI binds
  loopback by default. Reaching it from another machine
  is an explicit act of port forwarding, not a default.


## 12. Open Questions

- Does the CLI support a config file? The current
  design supports `./var/config.yaml`.
- Does the CLI support logging? The current design
  supports a log file at `./var/aios.log`.
