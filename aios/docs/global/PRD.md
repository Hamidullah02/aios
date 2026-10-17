# aios — Global Product Requirements Document

## 1. Purpose

This document states the product requirements for the
aios project. The document covers the project as a
whole. The document is the umbrella for the per-module
PRDs. The per-module PRDs are the source of truth for
the module detail. This document is the source of
truth for the cross-module concerns.

## 2. Scope

The aios project is in scope for this document.

- The product vision.
- The user personas.
- The user stories.
- The functional and non-functional requirements
  that span more than one module.
- The success criteria.
- The out-of-scope list.

The aios project is not in scope for this document.

- The per-module detail. The detail is in the
  per-module PRDs under `docs/modules/<module>/PRD.md`.
- The competitive research. The research is in
  `DESIGN.md` §2 at the project root.
- The implementation plan. The plan is in
  `docs/global/IMPLEMENTATION-PLAN.md`.

## 3. Reference Documents

- `DESIGN.md` at the project root. The document is
  the source of truth for the architecture and
  the competitive research.
- `aios/AUDIT.md`. The document is the source of
  truth for the design audit and the gap list.
- `aios/docs/README.md`. The document is the entry
  point for the documentation.
- `aios/docs/INDEX.md`. The document is the file
  index.
- The per-module PRDs under
  `aios/docs/modules/<module>/PRD.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| aios | The agentic operating system. The system is the project. |
| Module | A standalone Go package under `internal/<module>`. |
| Layer | A horizontal band of modules. The bands are foundation, capability, agentic, and operator. |
| Wedge | A set of 20 well-defined software delivery tasks. The set is the benchmark for the orchestrator. |
| Tier | A risk level for an action. The levels are 1 (read), 2 (local write), 3 (network or irreversible). |

## 5. Product Vision

The aios project is a small, durable task runner for
software delivery. The user states an objective. The
system turns the objective into a typed task plan. The
system executes the plan in a sandbox. The system
verifies the result with tests and schema checks. The
system checkpoints every step. The system audits every
decision. The system pauses for human approval on
risky actions.

The system is not a chatbot. The system is not a
platform. The system is a library that one person can
operate. The system runs on a Linux host with Docker.
The system works with any model provider through a
typed adapter layer. The system costs less than
$50 per month for a single user at the prototype
stage.

## 6. User Personas

### 6.1 Builder

The builder is a software engineer. The builder uses
aios to ship code. The builder wants the agent to
follow the project conventions. The builder wants the
agent to ask before risky actions. The builder wants
to be able to read the audit log and understand what
the agent did.

### 6.2 Operator

The operator is the person who runs the system. The
operator sets the model, the budget, the policy, and
the approval rules. The operator wants the system to
stay within the budget. The operator wants the system
to learn from the results.

### 6.3 Reviewer

The reviewer is a person who audits the run. The
reviewer reads the report. The reviewer checks the
audit log. The reviewer signs off on the run.

## 7. User Stories

### 7.1 Builder Stories

The builder wants to give the system a goal. The
builder types the goal at the CLI. The system turns
the goal into a plan. The builder reviews the plan.
The builder approves the plan. The system runs the
plan.

The builder wants the system to follow the project
conventions. The system reads `AGENTS.md` at the
project root. The system follows the conventions
in the file.

The builder wants the system to ask before risky
actions. The system pauses the run. The system
shows the approval prompt. The builder approves
or denies the action.

### 7.2 Operator Stories

The operator wants to set the model. The operator
edits `var/models.yaml`. The system uses the model
for the next run.

The operator wants to set the budget. The operator
passes the budget through the CLI. The system
tracks the spend. The system stops the run when
the budget is exhausted.

The operator wants to set the policy. The operator
edits `var/policy.yaml`. The system enforces the
policy. The system logs the policy decisions.

The operator wants to set the approval rules. The
operator types `y, always` at the approval prompt.
The system saves the rule. The system applies the
rule to future runs.

### 7.3 Reviewer Stories

The reviewer wants to read the report. The reviewer
runs `aios report`. The system shows the report in
text, JSON, or HTML.

The reviewer wants to read the audit log. The
reviewer runs `aios audit`. The system shows the
audit log with the hash chain verified.

The reviewer wants to replay a run. The reviewer
runs `aios replay <session_id>`. The system
re-executes the plan. The system produces a new
report.

## 8. Functional Requirements Summary

The aios project provides the following functions.
The functions are realized by the modules listed
under each function.

### 8.1 Foundation Functions

- The typed schemas, the audit log, the state
  machine, and the redaction layer. The functions
  are in `github.com/<org>/aios/internal/core`.
- The policy gateway. The function is in
  `github.com/<org>/aios/internal/policy`.
- The sandbox runner. The function is in
  `github.com/<org>/aios/internal/sandbox`.
- The egress proxy. The function is in
  `github.com/<org>/aios/internal/egress`.
- The model adapter layer. The function is in
  `github.com/<org>/aios/internal/models`.
- The model router with cost optimization. The
  function is in `github.com/<org>/aios/internal/optimizer`.
- The approval service. The function is in
  `github.com/<org>/aios/internal/approvals`.
- The always-allow rule storage, the session
  checkpoint, the structured handoff, and the
  session history. The functions are in
  `github.com/<org>/aios/internal/persist`.

### 8.2 Capability Functions

- The tool bus and the built-in tools. The function
  is in `github.com/<org>/aios/internal/tools`.
- The glob search, the content search, and the
  file read at scale. The functions are in
  `github.com/<org>/aios/internal/search`.
- The string edit, the line edit, and the file
  write at scale. The functions are in
  `github.com/<org>/aios/internal/edit`.
- The git integration. The function is in
  `github.com/<org>/aios/internal/git`.
- The key-value store with namespacing and TTL.
  The function is in `github.com/<org>/aios/internal/memory`.
- The file system watcher. The function is in
  `github.com/<org>/aios/internal/watch`.
- The parallel sub-agent runner. The function is
  in `github.com/<org>/aios/internal/fanout`.
- The retry policy with error feedback. The
  function is in `github.com/<org>/aios/internal/retry`.
- The self-check and the run_tests tool. The
  function is in `github.com/<org>/aios/internal/verify`.
- The agent role registry. The function is in
  `github.com/<org>/aios/internal/roles`.
- The multimodal input and the structured output.
  The function is in `github.com/<org>/aios/internal/image`.
- The ask-user tool. The function is in
  `github.com/<org>/aios/internal/ask`.
- The report generator. The function is in
  `github.com/<org>/aios/internal/report`.

### 8.3 Agentic Functions

- The planner and the code-first verifier. The
  function is in `github.com/<org>/aios/internal/planner`.
- The orchestrator. The function is in
  `github.com/<org>/aios/internal/runtime`.
- The benchmark harness. The function is in
  `github.com/<org>/aios/internal/benchmark`.

### 8.4 Operator Functions

- The first-run CLI and the approval daemon. The
  function is in `github.com/<org>/aios/internal/cli`.

## 9. Non-Functional Requirements Summary

The aios project provides the following non-functional
qualities. The qualities span every module.

### 9.1 Performance

- The orchestrator dispatches a task within 1 second
  when the plan is loaded.
- The self-check tool returns a verification result
  within 10 seconds.
- The run_tests tool returns a test result within
  the duration of the test suite plus 5 seconds.
- The router decision has a maximum latency of
  5 milliseconds.

### 9.2 Security

- Every tool call goes through the policy gateway.
- The sandbox isolates every task by default.
- The egress proxy blocks the network by default.
- The audit log is append-only with a hash chain.
- The redaction layer strips secrets before any
  log write.
- The approval queue pauses for human input on
  tier-3 actions.

### 9.3 Reliability

- The state machine saves a checkpoint after every
  step.
- The orchestrator resumes from a checkpoint after
  a crash.
- The retry policy caps the retries on a tool
  failure.
- The approval service persists the decision.
- The watcher notifies the orchestrator of file
  system events.

### 9.4 Maintainability

- Every module is a standalone Go package.
- Every module has the same six documents.
- Every module has 100 percent test coverage for
  the public functions.
- Every module has a typed interface.
- Every module is tested in isolation.
- The merge order is bottom-up. The lower-level
  modules merge first.

### 9.5 Cost

- The prototype budget is under $50 per month for
  a single user.
- The model router picks the cheapest tier that
  meets the quality bar.
- The cost tracker records the cost for every
  call.
- The budget meter stops the run when the budget
  is exhausted.

## 10. Cross-Module Requirements

The aios project has the following requirements that
span more than one module.

### 10.1 Module Interface Contract

Every module shall export a typed interface through
`github.com/<org>/aios/internal/<module>`. The interface shall be stable across
minor versions. The interface shall be tested in
isolation. The interface shall not depend on the
implementation of any other module.

### 10.2 Audit Log Contract

Every module that produces a side effect shall write
an audit row. The row shall include the timestamp,
the actor, the action, the result, and the hash of
the previous row. The row type shall be in the
`github.com/<org>/aios/internal/core` module.

### 10.3 Policy Contract

Every tool call shall go through the policy gateway.
The policy gateway shall return `allow`, `deny`, or
`ask`. The `ask` action shall route to the approval
service.

### 10.4 Cost Contract

Every model call shall record the cost. The cost
shall be computed through the price table in
`github.com/<org>/aios/internal/models`. The cost shall be recorded in the
audit log.

### 10.5 Approval Contract

Every tier-3 action shall route to the approval
service. The approval service shall persist the
decision. The approval service shall notify the
waiter.

### 10.6 Retry Contract

Every tool call that fails shall go through the
retry policy. The retry policy shall return `retry`,
`retry_with_fix`, or `stop`. The retry policy
shall cap the retries.

### 10.7 Verify Contract

Every task result shall go through the self-check
before the task is marked as done. The self-check
shall return `pass`, `fail`, or `inconclusive`.

### 10.8 Role Contract

Every task shall resolve to a role. The role shall
include the system prompt, the tool set, the model
tier, and the output format. The role shall be
loaded from `var/roles.yaml`.

### 10.9 Persist Contract

Every standing approval shall be persisted. The
persisted rule shall be applied to future runs. The
rule shall be stored at `~/.aios/rules.json` or
`var/rules.json`.

### 10.10 Handoff Contract

Every task that produces an artifact shall publish
the artifact to the handoff store. The next task
shall read the artifact from the handoff store. The
artifact shall be a file path or a JSON-serializable
value.

## 11. Success Criteria

The aios project is successful when the following
criteria are met.

- The orchestrator completes the 20-task wedge
  dataset with at least 90 percent of the tasks
  passing.
- The average cost per task is under $0.50.
- The average time per task is under 5 minutes.
- The approval queue pauses for human input on
  every tier-3 action.
- The audit log is hash-chained and verifiable.
- The system resumes from a crash without losing
  the in-flight state.
- The test suite has 100 percent pass rate.
- The test suite has 100 percent coverage for the
  public functions of every module.
- The documentation set is complete for every
  module.

## 12. Out of Scope

The aios project does not cover the following items
at the prototype stage.

- The parallel agents. The pattern is in
  `github.com/<org>/aios/internal/fanout` but is not the default.
- The business connectors beyond read-only.
- The self-built tools in production.
- The persistent shared VM.
- The browser computer-use.
- The multi-user tenancy.
- The agent hierarchies.
- The LLM as the security boundary.
- The broad default permissions.
- The agents editing their own policy.
- The always-on unattended outbound messaging.

## 13. Risks

The aios project has the following risks. The risks
are tracked in the implementation plan.

- The model quality is below the threshold for
  some tasks. The mitigation is the model router
  and the cascade mode.
- The cost is above the budget. The mitigation is
  the cost tracker and the budget meter.
- The approval fatigue makes the operator ignore
  the queue. The mitigation is the auto-allow
  rules and the policy templates.
- The audit log is too large to inspect. The
  mitigation is the report generator and the
  search tool.
- The hash chain breaks. The mitigation is the
  re-anchor tool and the audit log repair.

## 14. Version

This document is version 1.0. The previous version
was 0.9. The change is the addition of the
optimizer, retry, verify, roles, persist, and
image modules, and the closure of the remaining
v0.9 audit gaps.
