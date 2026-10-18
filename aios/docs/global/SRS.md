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

The approval contract has four tiers. Tier
definitions are in the global PRD §11.5.

- **Tier 0** — read-only, local. No approval.
- **Tier 1** — local write, reversible. No
  approval.
- **Tier 2** — network or new connector
  scope. Approval **once per task scope**,
  not per call. Allowlisted host is
  auto-allowed; non-allowlisted host is
  tier 2.
- **Tier 3** — irreversible or privileged.
  Always requires explicit approval, every
  time.

**REQ-GLB-APR-001** Every tier-2 (non-
allowlisted host) and tier-3 action shall
route to the approval service. The approval
service shall block until the operator
responds. Tier-2 approvals are once-per-task-
scope. Tier-3 approvals are once-per-call.

**REQ-GLB-APR-002** The approval service
shall persist the decision. The decision
shall be stored in the audit log. The
decision row shall include the action, the
action hash, the principal, the channel, and
the expiry.

**REQ-GLB-APR-003** Approvals shall be
one-use, hash-bound, and expiring. A
standing approval (a rule that applies to a
class of actions for the rest of the
session) is **not** allowed for tier-3
actions. The gateway shall reject any
`aios allow add` rule that matches a tier-3
action at insertion time. The rejection is
enforced by code, not by convention.

**REQ-GLB-APR-004** When the gateway cannot
reach the human (no approval channel
available), tier-3 actions shall be blocked.
The block shall be recorded. The run shall
be paused. Tier-2 actions on allowlisted
hosts may proceed; tier-2 actions on non-
allowlisted hosts are also blocked.

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

**REQ-GLB-VRF-001** Every task result shall
go through the verify layer before the task
is marked as done. The verify layer is run
by the orchestrator, not by the agent. The
agent cannot skip the verify layer. The
verify layer shall return one of the three
statuses: `pass`, `fail`, `inconclusive`.
**`inconclusive` is treated as `fail`**: the
task is marked **blocked**, not done. The
verify layer is code-first: it runs the
project's tests, it validates the output
schema, and it checks the acceptance
criteria from the spec. The agent's self-
evaluation is never a verify result.

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
the cheapest tier that the verify layer will
accept. The router shall be a pure function of
the cost/quality table, the override, and the
budget.

**REQ-GLB-OPT-002** The cost tracker shall record
the empirical cost for every call. The tracker
shall enforce the per-task budget.

**REQ-GLB-OPT-003** The cost/quality table shall
record the empirical pass rate per
(fingerprint, tier) pair. The pass rate shall
come from the verify result, not from the model
self-evaluation.

**REQ-GLB-OPT-004** The router shall support the
override surface. The CLI shall accept the
`--tier`, `--optimizer`, and `--budget` flags.

**REQ-GLB-OPT-005** The router shall never ask
the model to grade itself. The pass rate shall
always be derived from the verify result.

## 16. Determinism Requirements

The system around the model is deterministic,
auditable, and replayable. Model variance is
measured and bounded, not eliminated. The
requirements in this section state the
contract for the system around the model.

**REQ-GLB-DET-101** The plan object shall be
immutable after the orchestrator dispatches the
first task. The plan object shall be content
addressed. The plan id shall be the SHA-256 of
the plan payload. A replan creates a new plan
with a new id and a new audit row; the prior
plan remains in the audit log.

**REQ-GLB-DET-102** The policy object shall be
immutable for the run **except for
tightening**. Tightening is allowed: revoking
a tool, blocking a host, raising a tier, or
invoking the kill switch. Loosening is
forbidden mid-run. The policy is content
addressed; the policy id is the SHA-256 of
the policy payload; the policy version
history is preserved in the audit log.

**REQ-GLB-DET-103** The clock shall be injected
through a typed interface. The interface
shall expose `Now() time.Time`. The default
implementation shall use the wall clock. The
test implementation shall use a fixed clock.
The injected clock is used in **decision
paths**: plan dispatch, approval timestamps,
audit row timestamps, and the memory
timeline. The wall clock is used in
**deadline paths**: per-tool timeouts, the
daily cost cap, the per-run budget, and
approval expiry. The split is explicit and
documented in the clock interface contract.

**REQ-GLB-DET-104** Artifacts in the memory,
the audit log, the checkpoint store, and the
plan store shall be identified by content
hash. The hash shall be SHA-256. The hash
shall be computed at the moment the artifact
is written. The hash shall be recorded in the
audit row. **Path strings remain in the
decision path** for policy scope matching
(`./src/**`) and diff-scope matching. Hashes
identify artifacts; paths locate them.

**REQ-GLB-DET-105** The model call shall accept
a `temperature` argument. The default
`temperature` shall be the lowest value the
provider supports. When the provider does not
support `temperature`, the absence shall be
recorded in the model call metadata, not
silently treated as zero. The recorded model
output shall be persisted with the call.
Best-effort `temperature=0` reduces variance;
it is not a determinism guarantee.

**REQ-GLB-DET-106** The checkpoint shall be
content addressed. The checkpoint id shall be
the SHA-256 of the checkpoint payload. The
checkpoint payload shall be a JSON object.
The checkpoint shall include the recorded
model outputs for every call in the run.

**REQ-GLB-DET-107** The replay command shall
re-execute a recorded session using the
**recorded model outputs**. The replay shall
not re-call the model. The replay shall be a
pure function of the recorded outputs, the
recorded inputs, the recorded plan, the
recorded policy, the recorded pin set, and
the pinned sandbox image. Two replays of the
same session on the same pinned image shall
produce the same audit log. Live re-execution
is not part of the replay contract.

**REQ-GLB-DET-108** The diff command shall
compare two runs of the same plan. The diff
shall be over the audit log, the recorded
model outputs, and the tool results. The diff
shall be in unified diff format. The diff
shall work for two runs of the same plan on
the same host and for two runs on different
hosts, provided both hosts have the same
pinned sandbox image and the same recorded
outputs.

**REQ-GLB-DET-109** The system shall be
**reproducible** on a second host under the
following conditions: the same input, the
same plan, the same policy, the same
recorded model outputs, the same pin set,
the same approval history, a fixed clock for
decision paths, and a pinned sandbox image
with pinned dependency versions. Under these
conditions, the system shall produce the
same audit log and the same outcome.
**Live re-execution with fresh model calls is
not reproducible**; the variance is recorded
in the bench, not eliminated by the system.

**REQ-GLB-DET-110** The system shall not use
the global random number generator in any
decision path. The system shall inject a
seeded random source through a typed
interface. The default seed shall be derived
from the run id. The seeded random is
replayable: the same seed produces the same
sequence.

**REQ-GLB-DET-111** The system shall record
the actual model output for every model
call. The recording is the source of truth
for replay. The recording is the source of
truth for the bench's variance analysis. The
recording includes the model id, the model
class, the `temperature` value or its
absence, the prompt hash, the output hash,
the token count, the cost, and the latency.

**REQ-GLB-DET-112** The variance between two
live runs of the same plan on the same
input shall be measured. The measurement
shall record the number of differing
recorded model outputs, the number of
differing tool results, and the number of
differing audit rows. The measurement is
exposed through `aios bench regression`.

**REQ-GLB-DET-113** The system shall not
pretend the model is a function. The
documentation, the CLI output, the bench
output, and the report shall state the
variance, not the false promise of
determinism. The thesis is reproducibility
and auditability, not determinism.

**REQ-GLB-DET-114** The policy file, the
roles file, the rules file, and the
retention file shall live outside the
workspace. The sandbox shall not have write
access to any of these files. The agent
shall not be able to reach the policy
socket, the roles socket, or the rules
socket. The control channel is a Unix
domain socket bound to a path the sandbox
cannot reach, authenticated by a one-shot
token the agent never holds.

## 17. Memory Requirements

The memory is the moat. The memory is the
content-addressed knowledge base that gives
the agent a context that is bigger, better, and
more honest than the competition. This section
states the cross-module requirements for the
memory. The per-module SRS for aios-memory is
the source of truth for the module detail.

**REQ-GLB-MEM-001** The memory shall be
content-addressed. Every artifact shall be
identified by its content hash. The content
hash shall be the SHA-256 of the artifact
payload.

**REQ-GLB-MEM-002** The memory shall support
three scopes: `session`, `project`, `user`.
The session scope shall die when the session
ends. The project scope shall persist across
sessions in the same project. The user scope
shall persist across projects.

**REQ-GLB-MEM-003** The memory shall support
three layers: `raw`, `indexed`, `compacted`.
The raw layer shall be append-only. The
indexed layer shall be the typed view. The
compacted layer shall be the lossy view that
fits the context budget.

**REQ-GLB-MEM-004** The orchestrator shall
retrieve context through the memory, not
through a message buffer. The retrieval
algorithm shall be a pure function of the
indexed layer, the query, the pin set, the
current task id, and the budget.

**REQ-GLB-MEM-005** The orchestrator shall
honor the pin set. Pinned artifacts shall
always be included in the compacted view
regardless of the budget.

**REQ-GLB-MEM-006** The planner, the verifier,
the reporter, and the fan-out runner shall
use the memory to store decisions,
observations, tool results, and model
outputs. The memory shall be on the critical
path of every model call.

**REQ-GLB-MEM-007** The CLI shall provide the
`aios memory` subcommand. The CLI shall print
output as grep-friendly JSON.

**REQ-GLB-MEM-008** The memory shall be
observable. The developer shall be able to
inspect any artifact by content hash. The
developer shall be able to diff two
sessions. The developer shall be able to
prune noise. The developer shall be able to
pin important artifacts across all compacted
views.

**REQ-GLB-MEM-009** The memory shall persist
across sessions. A decision made in session
A shall be retrievable in session B through
a query. The project scope shall be
`git`-trackable.

**REQ-GLB-MEM-010** The memory shall not be
the only determinism source. The memory
shall be a content-addressed key-value
store. The memory shall not own the audit
log. The audit log lives in
`github.com/<org>/aios/internal/persist`.

## 18. Non-Functional Requirements

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
