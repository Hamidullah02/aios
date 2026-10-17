# aios — Global Software Requirements Specification

## 1. Purpose

This document states the software requirements for
the aios project at the cross-module level. The
per-module SRS documents are the source of truth for
the module detail. This document is the source of
truth for the requirements that span more than one
module.

## 2. Scope

The aios project is in scope for this document.

- The module interface contract.
- The audit log contract.
- The policy contract.
- The cost contract.
- The approval contract.
- The retry contract.
- The verify contract.
- The role contract.
- The persist contract.
- The handoff contract.

## 3. Reference Documents

- Global PRD. See `docs/global/PRD.md`.
- `DESIGN.md` at the project root.
- The per-module SRS under
  `docs/modules/<module>/SRS.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| REQ | A typed requirement identifier. |
| NFR | A non-functional requirement identifier. |

## 5. Module Interface Requirements

**REQ-GLB-INT-001** Every module shall export a
typed interface through the package path
`github.com/<org>/aios/internal/<module>`. The
interface shall be a Go package.

**REQ-GLB-INT-002** Every module shall provide a
public API. The API shall be the union of the items
listed in the module's SRS.

**REQ-GLB-INT-003** Every module shall not depend
on the implementation of another module. The
dependency shall be on the interface.

**REQ-GLB-INT-004** Every module shall compile into
the same monorepo. The module shall be wired into
`go.work` or the monorepo `go.mod`.

## 6. Audit Log Requirements

**REQ-GLB-AUD-001** Every module that produces a
side effect shall write an audit row. The row type
shall be defined in `github.com/<org>/aios/internal/core`.

**REQ-GLB-AUD-002** Every audit row shall include
the following fields.

- `timestamp` — the ISO 8601 timestamp.
- `actor` — the actor that produced the row.
- `action` — the action that was performed.
- `result` — the result of the action.
- `prev_hash` — the hash of the previous row.
- `row_hash` — the hash of the current row.

**REQ-GLB-AUD-003** The audit log shall be
append-only. The audit log shall reject any update
or delete operation.

**REQ-GLB-AUD-004** The audit log shall be
hash-chained. The hash chain shall be verifiable
through the `verify_chain` method.

## 7. Policy Requirements

**REQ-GLB-POL-001** Every tool call shall go through
the policy gateway. The policy gateway shall return
one of the three actions: `allow`, `deny`, `ask`.

**REQ-GLB-POL-002** The `ask` action shall route to
the approval service. The approval service shall
pause the run.

**REQ-GLB-POL-003** The policy decision shall be
recorded in the audit log. The decision shall
include the tool, the resource, the action, and
the reason.

## 8. Cost Requirements

**REQ-GLB-CST-001** Every model call shall record
the cost. The cost shall be computed through the
price table in `github.com/<org>/aios/internal/models`.

**REQ-GLB-CST-002** The cost shall be recorded in
the audit log. The cost shall include the model id,
the input tokens, the output tokens, and the cost
in USD.

**REQ-GLB-CST-003** The cost tracker shall enforce
the per-task budget. The tracker shall raise
`BudgetExhausted` when the budget is exhausted.

**REQ-GLB-CST-004** The cost tracker shall enforce
the per-call budget. The tracker shall raise
`BudgetExhausted` when the per-call budget is
exhausted.

## 9. Approval Requirements

**REQ-GLB-APR-001** Every tier-3 action shall route
to the approval service. The approval service shall
block until the operator responds.

**REQ-GLB-APR-002** The approval service shall
persist the decision. The decision shall be stored
in the audit log.

**REQ-GLB-APR-003** The approval service shall
support the "always" option. The "always" option
shall create a standing rule in `github.com/<org>/aios/internal/persist`.

## 10. Retry Requirements

**REQ-GLB-RTR-001** Every tool call that fails
shall go through the retry policy. The retry
policy shall return one of the three actions:
`retry`, `retry_with_fix`, `stop`.

**REQ-GLB-RTR-002** The retry policy shall cap the
retries. The default cap is 3 retries per task.

**REQ-GLB-RTR-003** The retry policy shall support
a per-tool limit. The retry policy shall support
a per-error-type limit.

## 11. Verify Requirements

**REQ-GLB-VRF-001** Every task result shall go
through the self-check before the task is marked
as done. The self-check shall return one of the
three statuses: `pass`, `fail`, `inconclusive`.

**REQ-GLB-VRF-002** The verify module shall
support a `goal` argument. The goal shall be
used in the verification prompt.

**REQ-GLB-VRF-003** The verify module shall support
the `run_tests` tool. The tool shall run the
project's test command and return the result.

## 12. Role Requirements

**REQ-GLB-ROL-001** Every task shall resolve to a
role. The role shall include the system prompt, the
tool set, the model tier, and the output format.

**REQ-GLB-ROL-002** The role shall be loaded from
`var/roles.yaml`. The file shall be created on
`aios init`.

**REQ-GLB-ROL-003** The role registry shall provide
the default roles: `planner`, `coder`, `editor`,
`explorer`, `reviewer`, `verifier`.

## 13. Persist Requirements

**REQ-GLB-PRS-001** Every standing approval shall
be persisted. The rule shall be stored at
`~/.aios/rules.json` (user) or `var/rules.json`
(project). The project-level rule shall override
the user-level rule.

**REQ-GLB-PRS-002** The session state shall be
checkpointed. The checkpoint shall be stored at
`var/sessions/<session_id>.json`.

**REQ-GLB-PRS-003** The handoff store shall be
used to pass artifacts between tasks. The store
shall be at `var/handoffs/<task_id>/<key>.json`.

**REQ-GLB-PRS-004** The session history shall be
queryable. The query shall return the events in
time order.

## 14. Image Requirements

**REQ-GLB-IMG-001** The image read tool shall
accept a path. The tool shall return the image
data with the MIME type and the dimensions.

**REQ-GLB-IMG-002** The image read tool shall
support the following formats: PNG, JPEG, GIF,
WebP.

**REQ-GLB-IMG-003** The image read tool shall
enforce a maximum size of 20 MB. The tool shall
raise `ImageTooLargeError` when the file is too
large.

**REQ-GLB-IMG-004** The structured output
validator shall validate the response against
the schema. The schema shall be a JSON Schema (github.com/santhosh-tekuri/jsonschema/v5)
or a struct.

## 15. Optimizer Requirements

**REQ-GLB-OPT-001** The model router shall pick
the right tier for each task. The router shall
support the following modes: `static`, `dynamic`,
`cascade`.

**REQ-GLB-OPT-002** The cost tracker shall record
the cost for every call. The tracker shall enforce
the per-task budget.

**REQ-GLB-OPT-003** The quality feedback loop
shall update the strategy from the result quality.
The loop shall support the `auto_escalate` flag.

## 16. Determinism Requirements

**REQ-GLB-DET-101** The plan object shall be
immutable after the orchestrator dispatches the
first task. The plan object shall be content
addressed. The plan id shall be the SHA-256 of
the plan payload.

**REQ-GLB-DET-102** The policy object shall be
immutable after the run starts. The policy
object shall be content addressed. The policy
id shall be the SHA-256 of the policy payload.

**REQ-GLB-DET-103** The clock shall be injected
through a typed interface. The interface shall
expose `Now() time.Time`. The default
implementation shall use the wall clock. The
test implementation shall use a fixed clock.

**REQ-GLB-DET-104** The file system shall be
identified by content hash. The hash shall be
SHA-256. The hash shall be computed at the
moment the file is written. The hash shall be
recorded in the audit row.

**REQ-GLB-DET-105** The model call shall accept
a temperature argument. The default temperature
shall be 0. The router shall reject any call
with a temperature above 0 unless the call is
wrapped in a section tagged as non-deterministic.

**REQ-GLB-DET-106** The checkpoint shall be
content addressed. The checkpoint id shall be
the SHA-256 of the checkpoint payload. The
checkpoint payload shall be a JSON object.

**REQ-GLB-DET-107** The replay command shall
re-execute the plan from a checkpoint. The
replay shall use the same model adapter, the
same model id, the same temperature, and the
same role resolver. The replay shall produce an
equivalent audit log for every step that does
not involve the model.

**REQ-GLB-DET-108** The diff command shall
compare two runs of the same plan. The diff
shall be over the audit log and the tool
results. The diff shall be in unified diff
format.

**REQ-GLB-DET-109** The system shall be
reproducible on a second host. Two hosts given
the same input, the same plan, the same
approvals, the same model, the same
temperature, and a fixed clock shall produce
the same audit log and the same outcome.

**REQ-GLB-DET-110** The system shall not use
the global random number generator in any
decision path. The system shall inject a
seeded random source. The default seed shall
be derived from the run id.

## 17. Non-Functional Requirements

**REQ-GLB-NFR-001** The system shall run on Go
1.23.

**REQ-GLB-NFR-002** The system shall run on Linux
with Docker.

**REQ-GLB-NFR-003** The system shall have 100
percent test coverage for the public functions of
every module.

**REQ-GLB-NFR-004** The system shall have a test
suite that runs in under 5 minutes.

**REQ-GLB-NFR-005** The system shall have a `go vet`
clean report. The system shall have a `staticcheck`
clean report.

**REQ-GLB-NFR-006** The system shall have a
`gofmt -l` empty report. The system shall have a
`go vet ./...` clean report.

**REQ-GLB-NFR-007** The system shall be
documented in STE-100 prose.

**REQ-GLB-NFR-008** The system shall be mergeable
bottom-up. The merge order is in
`docs/README.md`.

## 18. Acceptance Criteria

1. Every REQ in this document has a test case.
2. The test suite has a 100 percent pass rate.
3. The test suite has 100 percent coverage for the
   public functions of every module.
4. The audit log is hash-chained and verifiable.
5. The policy gateway blocks every tier-3 action.
6. The cost tracker stops the run on budget
   exhaustion.
7. The approval service persists the decision.
8. The retry policy caps the retries.
9. The verify module catches every failure.
10. The role registry provides the default roles.
11. The persist module stores the rules.
12. The handoff module passes the artifacts.
13. The image module reads and attaches the
    images.
14. The optimizer module routes to the right tier.
15. The replay of any run produces an equivalent
    audit log for every step that does not involve
    the model.
16. The diff of two runs of the same plan
    produces a unified diff over the audit log.
17. The system rejects mid-flight changes to the
    plan and to the policy.
