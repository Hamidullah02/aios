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
| Tier | A risk level for an action. The levels are 0 (read-only, local), 1 (local write, reversible), 2 (network or connector scope, once-per-scope approval), 3 (irreversible or privileged, always requires explicit approval). |
| Determinism (bounded) | The system around the model is deterministic, auditable, and replayable. Model variance is measured and bounded, not eliminated. The same plan, on the same input, through the same orchestrator, with the same recorded model outputs, on a pinned sandbox image, produces the same audit log and the same outcome. The same plan re-executed with live model calls does not guarantee the same outcome; the variance is recorded. |
| Reproducibility | A run can be re-executed from its audit log and its recorded model outputs to produce an equivalent forward path. Live re-execution is not part of reproducibility. |
| Replay | Re-runs a recorded session using the recorded model outputs. The replay does not re-call the model. The replay is a pure function of the recorded outputs, the recorded inputs, and the pinned sandbox image. |
| Bounded variance | Model output drift is measured, recorded, and exposed through the optimizer. The system does not pretend the model is deterministic. |

## 5. The Determinism Thesis

The whole point of aios is to make complex coding
and large systems **reproducible and auditable** —
not to make them "deterministic" in the sense
that a model-mediated system cannot be.

A model is a stochastic sampler. It drifts between
runs for reasons outside the caller's control:
batching, hardware, silent provider updates, and
provider-specific deprecations of parameters
(including `temperature` on at least one Haiku
launch). Setting `temperature=0` is best-effort
variance reduction, not a determinism guarantee.
Bring-your-own-model means the parameter may
not even be settable. The system must be honest
about this.

aios fixes the right part of the problem: every
layer *around* the model is deterministic,
auditable, and replayable. The model is one
input, never the foundation. The result is a
system where the non-determinism is named,
measured, and bounded — not a system that
pretends the model is a function.

The non-determinism budget is listed below. Each
source is named. Each source has a measured
bound. Each source is recorded in the audit log.

- **Model output.** The model is a stochastic
  sampler. The budget for variance is the call
  fingerprint, the cost/quality table, the
  `temperature` setting when the provider
  supports it, and the recorded model output
  for every call. The `temperature` parameter
  is best-effort. When the provider does not
  support `temperature` (the parameter is
  rejected, ignored, or returned as an error
  on certain Haiku model versions and on
  some other vendor endpoints), the absence
  is recorded as a model call metadata field,
  not silently treated as zero. The recorded
  output is the source of truth for replay.
- **Wall clock.** The clock is non-deterministic
  for timestamps. The budget for timestamps
  is the injected clock (`Clock` interface,
  default `wallClock`, test `fixedClock`).
  The wall clock still drives timeouts, the
  daily cost cap, the per-run budget, and
  approval expiry — these are real deadlines,
  not decision inputs. Decision paths use the
  injected clock; deadline paths use the wall
  clock. The split is explicit.
- **Network.** The network is non-deterministic
  in availability. The budget for failure is
  the retry policy and the egress proxy. The
  budget for variance is the policy gateway,
  which restricts what network actions are
  even attempted.
- **Operator.** The operator is non-deterministic
  in approval timing. The budget for the
  operator is the approval rules, the
  one-use / hash-bound / expiring approval
  contract, and the kill switch.

Every other part of the system is reproducible
and auditable by construction: a pure function
of its inputs, with its outputs covered by
the audit log and the replay store.

- The plan is a typed value. The plan is
  immutable once dispatched. A replan creates
  a new plan version with a new id and a new
  audit row; the prior plan remains in the
  audit log.
- The policy is a typed value. The policy is
  immutable for the run except for **tightening**
  — a tool revocation (removing a tool from
  the allowed set) or a kill switch takes
  effect on the **next decision** for the
  affected action; the in-flight call
  completes unless kill is also in effect.
  Credential revocation is next-decision per
  §12.5.1. Loosening is not allowed mid-run.
- The audit row is the result of a pure
  function of its inputs (the action, the
  fingerprint, the policy, the verify result).
  Two rows with the same inputs have the same
  hash.
- The checkpoint is a content-addressed
  snapshot. The hash chain is a cryptographic
  proof. The chain is append-only; deletion
  uses signed truncation anchors that preserve
  the chain.
- The replay is a re-execution from recorded
  model outputs. The replay does not re-call
  the model. Two replays with the same
  recorded outputs, the same inputs, the same
  plan, the same policy, the same pin set,
  and the same pinned sandbox image produce
  the same audit log.

The agent's context is a query against a
content-addressed memory, not a buffer of the
most recent messages. Every artifact the agent
touches has a SHA-256. The agent retrieves
artifacts by hash, ranked by relevance, bounded
by the context budget. The retrieval and the
compaction are pure functions of the indexed
layer, the pin set, the budget, and the
ranking algorithm. The ranking algorithm is
pinned (full-text first, embeddings only when
the embedding model is also pinned). The
memory is observable: the developer can inspect
any artifact by content hash, diff two
sessions, prune noise, and pin important
artifacts across all compacted views.

The memory is a *candidate* moat. The claim
that it is the moat is unproven; "most
direct answer" is a design claim, not a
measured result. The system will earn or
lose the moat claim only after the bench,
the false-success rate, and the adversarial
suite show the system holding up over weeks
of use, not days. Until then, every
statement in this PRD about the memory is a
*design intent*, not a *measured
property*. The memory design — content-
addressed store, three scopes, three
layers, typed query language, observable
compaction — is the system's bet; the bench
is the test.

## 6. Product Vision

The aios project is a reproducible and auditable
task runner for software delivery. The user states
an objective. The system turns the objective into a
typed task plan. The system executes the plan in a
sandbox. The system verifies the result with tests
and schema checks. The system checkpoints every
step. The system audits every decision. The system
pauses for human approval on risky actions.

The system is not a chatbot. The system is not a
platform. The system is a library that one person
can operate. The system runs on a Linux host.
**Production isolation is gVisor or a microVM
(Firecracker-class) with a deny-by-default
seccomp/AppArmor profile, a read-only root
filesystem, no `privileged`, and a
network namespace separate from the host.**
Plain Docker with the default seccomp profile
is acceptable for local development only; the
production posture is gVisor or a microVM.
The system works with any model provider
through a typed adapter layer. The system costs
less than $50 per month for a single user at the
prototype stage.

The user can replay any run. The user can diff two
runs of the same plan. The user can answer the
question "why did the agent do X" by reading the
audit log. The user can answer the question "what
would the agent do" by replaying the plan from a
checkpoint.

The system is judged on its reproducibility and
auditability, not on the cleverness of its plans.
A simpler plan whose decisions can always be
explained and replayed is more valuable than a
clever plan whose decisions drift, cannot be
explained, or cannot be replayed. Variance is
measured, not eliminated (see §5).

## 7. User Personas

### 7.1 Builder

The builder is a software engineer. The builder uses
aios to ship code. The builder wants the agent to
follow the project conventions. The builder wants the
agent to ask before risky actions. The builder wants
to be able to read the audit log and understand what
the agent did.

### 7.2 Operator

The operator is the person who runs the system. The
operator sets the model, the budget, the policy, and
the approval rules. The operator wants the system to
stay within the budget. The operator wants the system
to learn from the results.

### 7.3 Reviewer

The reviewer is a person who audits the run. The
reviewer reads the report. The reviewer checks the
audit log. The reviewer signs off on the run.

## 8. User Stories

### 8.1 Builder Stories

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

### 8.2 Operator Stories

The operator wants to set the model. The operator
edits `var/models.yaml`. The system uses the model
for the next run.

The operator wants to set the budget. The operator
passes the budget through the CLI. The system
tracks the spend. The system stops the run when
the budget is exhausted.

The operator wants to set the policy. The operator
issues `aios policy set ...` through the control
channel. The system enforces the policy. The system
logs the policy decisions. The control channel is
the only path that can write policy; direct edits
to any policy file are rejected (see §11.3 and
§14).

The operator wants to set the approval rules. The
operator types `y, always` at the approval prompt
for tier 0 or tier 1 actions. The system writes
a standing rule through the control channel. The
system applies the rule to future runs whose
`action`, target, and scope match. `y, always` is
rejected for tier 2 and tier 3 actions; tier 2
uses per-scope approval and tier 3 uses
per-action approval (see §11.5 and §11.9).

### 8.3 Reviewer Stories

The reviewer wants to read the report. The reviewer
runs `aios report`. The system shows the report in
text, JSON, or HTML.

The reviewer wants to read the audit log. The
reviewer runs `aios audit`. The system shows the
audit log with the hash chain verified.

The reviewer wants to replay a run. The reviewer
runs `aios replay <session_id>`. The system
replays the decisions of the original run, using
the recorded model inputs and outputs and the
recorded tool results, against the same
content-addressed inputs. The system produces a
new report. Replay never re-calls models and
never re-executes tier 2 or tier 3 effects. If
the replay artifacts have aged out of the
encrypted replay store, replay fails with a
clear message (see §11.11 and §12.5.8).

## 9. Functional Requirements Summary

The aios project provides the following functions.
The functions are realized by the modules listed
under each function.

### 9.1 Foundation Functions

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
  router is a pure function of the cost/quality
  table, the override, and the budget. The
  function is in `github.com/<org>/aios/internal/optimizer`.
- The approval service. The function is in
  `github.com/<org>/aios/internal/approvals`.
- The always-allow rule storage. The function
  is in `github.com/<org>/aios/internal/policy`
  (the rule store lives with the policy
  module, not with the approval service or
  with `persist`; the approval service
  consults the rule store but does not own
  it).
- The credential broker. The function is in
  `github.com/<org>/aios/internal/cred`.
- The session checkpoint, the structured
  handoff, and the session history. The
  functions are in
  `github.com/<org>/aios/internal/persist`.
- The content-addressed memory. The function is
  in `github.com/<org>/aios/internal/memory`.

### 9.2 Capability Functions

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

### 9.3 Agentic Functions

- The planner and the code-first verifier. The
  function is in `github.com/<org>/aios/internal/planner`.
- The orchestrator. The function is in
  `github.com/<org>/aios/internal/runtime`.
- The benchmark harness. The function is in
  `github.com/<org>/aios/internal/benchmark`.

### 9.4 Operator Functions

- The first-run CLI and the approval daemon. The
  function is in `github.com/<org>/aios/internal/cli`.

## 10. Non-Functional Requirements Summary

The aios project provides the following non-functional
qualities. The qualities span every module.

### 10.1 Performance

- The orchestrator dispatches a task within 1 second
  when the plan is loaded.
- The self-check tool returns a verification result
  within 10 seconds.
- The run_tests tool returns a test result within
  the duration of the test suite plus 5 seconds.
- The router decision has a maximum latency of
  5 milliseconds.

### 10.2 Security

- Every tool call goes through the policy gateway.
- The sandbox isolates every task by default.
- The egress proxy blocks the network by default.
- The audit log is append-only with a hash chain.
- The redaction layer strips secrets before any
  log write.
- The approval queue pauses for human input on
  tier 2 and tier 3 actions. Tier 3 has no
  standing approvals; every tier 3 request is
  approved individually.

### 10.3 Reliability

- The state machine saves a checkpoint after every
  step.
- The orchestrator resumes from a checkpoint after
  a crash.
- The retry policy caps the retries on a tool
  failure.
- The approval service persists the decision.
- **No file system watcher in MVP.** The
  watcher is a §9.2 function and a §13.1
  post-MVP item. MVP uses polling on
  agent-driven steps; the orchestrator does
  not require file system events.

### 10.4 Maintainability

- Every module is a standalone Go package.
- Every module has the same six documents.
- Every module has a test suite. Line and branch
  coverage targets are tracked but are guidelines,
  not gates (see §12.4). The product gates that
  matter are behavioral: no false-success on the
  adversarial set, no silent policy drift, no
  replay divergence within retention.
- Every module has a typed interface.
- Every module is tested in isolation.
- The merge order is bottom-up. The lower-level
  modules merge first.

### 10.5 Cost

- The prototype budget is under $50 per month for
  a single user.
- The model router picks the cheapest class that
  meets the quality bar.
- The cost tracker records the cost for every
  call.
- The budget meter stops the run when the budget
  is exhausted.

## 11. Cross-Module Requirements

The aios project has the following requirements that
span more than one module.

### 11.1 Module Interface Contract

Every module shall export a typed interface through
`github.com/<org>/aios/internal/<module>`. The interface shall be stable across
minor versions. The interface shall be tested in
isolation. The interface shall not depend on the
implementation of any other module.

### 11.2 Audit Log Contract

Every module that produces a side effect shall
write an audit row. The row is written in three
logical stages, in order, by `aios-persist`:

1. **Compute.** The producer computes
   `content_hash` over the *unredacted* row
   (actor, action, action arguments, result,
   scope, tier, injected model inputs and
   outputs for this decision). This hash is the
   one that "same decision ⇒ same hash" refers
   to. The producer holds the unredacted
   payload only in memory; it never persists it
   to the redacted store.
2. **Redact.** The redactor removes secrets,
   PII, and oversize content from the row,
   producing the **redacted row** that the
   audit store actually keeps. The redaction
   is a pure function of the row and the rule
   set; the same inputs under the same rules
   produce the same redacted row. A separate
   `redacted_hash` is the SHA-256 of the
   canonical encoding of the redacted row, so
   the persisted row is content-addressed too.
3. **Chain.** The persist module writes
   `(redacted_hash, content_hash, ...)` to
   the redacted store and computes
   `chain_hash = SHA-256(content_hash ‖
   prev_chain_hash ‖ sequence_number ‖
   row_id)`, where `prev_chain_hash` is the
   chain hash of the row with `sequence_number
   - 1` in the **same store** (the redacted
   store and the encrypted replay store have
   independent chains with independent
   `sequence_number` counters; see §12.5.8).
   Identical inputs at different positions in
   a chain have different `chain_hash` values
   by design.

The row timestamp is the value of the
**decision clock** at the time of the write.
The audit module never reads the wall clock
directly; it asks the `DecisionClock` interface
(§11.11). The deadline clock is a separate
value and is recorded when the action was a
deadline action.

Replay recomputes `content_hash` from the
encrypted replay store (which keeps the
unredacted payload, see §12.5.8), not from
the redacted audit row. The redacted row is
sufficient to prove a row *existed* and to
recompute `redacted_hash`; it is not
sufficient to recompute `content_hash`. The
two stores are the source-of-truth split
that makes the chain defensible against
redaction.

The row type lives in `aios-persist` (it is
the only module that knows the redacted /
unredacted split). The chain is tamper-
evident (see §12.5.8): if the control plane
holds the signing key, an attacker with
control of the host can rewrite history. The
design is not tamper-proof; it is tamper-
evident. Higher assurance requires the
external export in §12.5.8.

### 11.3 Policy Contract

Every tool call shall go through the policy
gateway. The policy gateway shall return
`allow`, `deny`, or `ask`. The `ask` action
shall route to the approval service.

**Rule precedence.** When more than one rule
matches a tool call, the rules are evaluated in
this order, and the first match wins:

1. **Hard deny** rules (security, scope
   boundary, SSRF block, untrusted content
   type) — terminal `deny`, never overridden
   by an allow rule.
2. **Ask** rules (tier 2 or tier 3 by
   default, tier 1 by operator override) —
   routes to the approval service.
3. **Allow** rules (allowlisted host, tier 0/1
   default) — terminal `allow` for that scope.
4. **Default deny** — any call that matches
   no rule is `deny`.

The gateway is the first line of defense, but
**not** the security boundary. The sandbox is
the security boundary: the policy gateway can
be bypassed by an attacker who controls the
host, the kernel, the egress proxy, or the
sandbox runtime, and the design treats the
sandbox as the boundary that the gateway
complements. The sandbox's network namespace
is the only thing that makes "no network"
actually mean "no network." The gateway makes
"no network" mean "no successful network call,"
and the sandbox makes it mean "no attempt."

### 11.4 Cost Contract

Every model call shall record the cost. The cost
shall be computed through the price table in
`github.com/<org>/aios/internal/models`. The cost shall be recorded in the
audit log.

### 11.5 Approval Contract

The approval contract has four tiers.

- **Tier 0** — read-only, local. No approval
  required. Examples: file read, glob, grep,
  diagnostics.
- **Tier 1** — local write, reversible. No
  approval required. Examples: edit a file,
  write to a scratch path, run a test.
- **Tier 2** — network or new connector scope.
  Approval required **once per task scope**,
  not per call. An allowlisted host (e.g. a
  private package registry, an internal API)
  is auto-allowed; a non-allowlisted host is
  tier 2.
- **Tier 3** — irreversible or privileged.
  Always requires explicit approval, every
  time. Examples: `git push`, `rm -rf`, force
  push, schema drop, credentials revoke.

Approvals are one-use, hash-bound, and
expiring. A standing approval (a rule that
applies to a class of actions for the rest of
the session) is not allowed for tier 2 with a
non-allowlisted host, and is not allowed for
tier 3 at all. The gateway rejects any
always-allow rule that matches a tier 3
action; this is enforced by the gateway, not
by convention.

Tier 3 actions are blocked when the gateway
cannot reach the human (no approval channel
available). The block is recorded. The run
is paused.

### 11.6 Retry Contract

Every tool call that fails shall go through the
retry policy. The retry policy shall return `retry`,
`retry_with_fix`, or `stop`. The retry policy
shall cap the retries.

### 11.7 Verify Contract

Every task result shall go through the verify
layer before the task is marked as done. The
verify layer is run by the **orchestrator**,
not by the agent. The agent cannot skip the
verify layer. The verify layer shall return
`pass`, `fail`, or `inconclusive`. **`inconclusive`
is treated as `fail`**: the task is marked
**blocked**, not done. The verify layer is
code-first: it runs the project's tests, it
validates the output schema, and it checks
the acceptance criteria from the spec. The
agent's self-evaluation is never a verify
result.

**Verification cannot be gamed by editing
tests.** The verify layer has access to a
**hidden test subset** maintained by the
operator (in a control-channel-only path, not
mounted in the sandbox). The hidden subset
runs alongside the project's tests on every
verify pass. The verify layer also performs a
**diff-scope check**: a test file in the
diff that was written by the worker in this
run does not count toward `pass`; the test
result is recorded as `worker-written-test`
in the audit log and does not flip the
verify verdict. The agent-callable `run_tests`
tool is a **tool** used by the worker to
iterate, not the verifier. The verifier is a
separate orchestrator-side component that
runs the project's tests plus the hidden
subset, validates the schema, and checks the
acceptance criteria. The agent cannot call
the verifier directly; the orchestrator is
the only caller.

### 11.8 Role Contract

Every task shall resolve to a role. The role shall
include the system prompt, the tool set, the model
class, and the output format. Role definitions
live in a directory that is **not mounted into any
sandbox's namespace** (e.g. `~/.aios/roles/` or a
control-channel-protected project path). The
sandbox can read roles by reference but cannot
write them; writes go through the control channel
only. A role that names a `min_model_profile` that
no available model meets is rejected at task
start.

### 11.9 Persist Contract

There are three classes of approval:

- **Per-action** — a single human `y` or `n`
  on a single tool call. Used for tier 3 and
  for tier 2 with a non-allowlisted host.
  Never persisted.
- **Per-scope** — a single human `y` on a
  scope (a set of tool calls with the same
  `action`, same target pattern, and same
  task). Used for tier 2 with an allowlisted
  host. Lives in memory for the duration of
  the run; never persisted. A new run, a new
  scope, or a policy change invalidates the
  per-scope approval.
- **Persisted (standing) rule** — a rule
  written through the control channel that
  matches a recurring action and approves it
  for all future runs. Allowed only for tier
  0 and tier 1 actions. A persisted rule
  that names a tier 2 or tier 3 action is
  rejected at write time. Tier 2 and tier 3
  are never approved by a standing rule.

Rule files live in a directory that is **not
mounted into any sandbox's namespace** (e.g.
`~/.aios/rules/`). The sandbox can read rules
by reference but cannot write or delete them;
writes go through the control channel only.

### 11.10 Handoff Contract

Every task that produces an artifact shall publish
the artifact to the handoff store. The next task
shall read the artifact from the handoff store. The
artifact shall be a file path or a JSON-serializable
value.

### 11.11 Determinism Contract

The system shall be reproducible and auditable
for every part that does not involve the model,
the wall clock, the network, or the operator. The
model, the clock, the network, and the operator
are the only sources of non-determinism. Every
other part of the system is a pure function of
its inputs. Variance is measured, not eliminated;
this contract specifies what is pinned, what is
recorded, and what replay is allowed to do.

The clock is split into two interfaces:

- **decision clock** — used in any code path that
  affects a decision (ranking, retry choice,
  routing, model selection, scope matching). The
  decision clock is injected and fixed in tests
  and replay.
- **deadline clock** — used for timeouts,
  budgets, lease expirations, and any path that
  ends a run. The deadline clock is allowed to
  read the wall clock.

The system distinguishes between decision paths
and effect paths. A **decision path** is any code
that picks a tool, a model, a target, an argument,
or a plan step. An **effect path** is any code
that produces a side effect (file write, network
call, git push, command exec, message send).
Decision paths are pinned; effect paths run on
the deadline clock and may be replaced by a
recorded artifact in replay.

Replay is defined as **replays decisions, not
effects**. Replay must not re-call any model and
must not re-execute any tier 2 or tier 3 effect.
Replay substitutes recorded outputs and recorded
tool results for the original paths and verifies
that the resulting decision sequence matches the
recorded one bit-for-bit at the decision layer.

**REQ-GLB-DET-001** The system shall be replayable
from the audit log. Replay shall substitute the
recorded model inputs, the recorded model outputs,
and the **recorded tool results for every tool
call regardless of tier**. Tier 0/1 tool results
are stubbed from the recording in the encrypted
replay store (§12.5.8); they are not re-executed.
Replay shall never re-call a model and shall
never re-execute a tier 2 or tier 3 effect.
Replay shall satisfy REPL-EQ-001 (§12.3). If a
required recording has aged out of the encrypted
replay store, replay fails with a clear
message.

**REQ-GLB-DET-002** The system shall request
`temperature=0` for every model call when the
provider supports it. If the provider rejects
`temperature=0` or does not honor the parameter,
the system shall record the actual temperature in
the audit row and the replay artifact, and shall
flag the run as "variance possible" in the
report. Temperature is a best-effort hint, not a
guarantee.

**REQ-GLB-DET-003** The system shall not use the
wall clock in any **decision path**. The system
shall inject the decision clock through a typed
interface. Tests and replay pass a fixed decision
clock. The wall clock is allowed in **deadline
paths** (timeouts, budgets, lease expirations)
and is recorded in the audit row as the deadline
timestamp.

**REQ-GLB-DET-004** The system shall identify
files by content hash in any decision path. File
**path strings** are allowed in decision paths
only when the path is part of the action's
**scope** (e.g. "modify file X", "delete directory
Y"), in which case the path is an input to the
decision and is recorded. In all other cases
(retracing a file, scanning a tree, mapping
inputs) the system uses the content hash, not the
path.

**REQ-GLB-DET-005** The system shall use
content-addressed checkpoints. The checkpoint id
shall be the SHA-256 of the checkpoint payload.

**REQ-GLB-DET-006** The system shall reject
mid-flight changes to the plan. The plan is
content-addressed (§11.11 DET-005); any byte
change alters the content hash. The plan shall
be immutable from the moment the orchestrator
dispatches the first task on it. Any revision
of any byte in the plan requires a new plan id
and a new audit row. The carve-out for
"cosmetic changes" is rejected: a comment
reformat is a byte change, the new plan id
applies, and the audit row records the
revision. There is no "decision-irrelevant
byte" in a content-addressed plan.

**REQ-GLB-DET-007** Policy is read-only for the
agent and the sandbox for the duration of a run.
The control channel may **tighten** policy mid-run
(e.g. raise a tool from tier 1 to tier 2, narrow
a scope, require re-approval). The control
channel shall not loosen policy mid-run. A
tightening takes effect on the next decision; the
prior decision stands. A loosening requires a
new run id. Every policy change is recorded in
the audit log.

**REQ-GLB-DET-008** The system shall be diffable.
The user shall be able to diff two runs of the
same plan. The diff shall be over the audit log,
the tool results, and the model inputs and
outputs (read from the encrypted replay store
under the user's access key, see §12.5.8).

### 11.12 Fail-Closed Contract

The system shall fail closed when any of the
following is true at the start of a step or
during a step: the audit store cannot accept
the next row, the policy gateway is unreachable,
the approval channel is unreachable and the
next action is tier 2 or tier 3, the encrypted
replay store is unavailable and the next step
would record a row that the replay store must
hold, or the signing key cannot be loaded. The
step is blocked, the run is marked `blocked` in
the audit log with a `blocked` reason code, and
the orchestrator surfaces the reason to the
operator. The system does not silently skip an
audit write, does not fall back to a different
store, and does not proceed with a stale
policy. A run that is blocked cannot be
"unblocked" except by resolving the underlying
condition; replay of a blocked run is supported
so the operator can confirm the block.

### 11.13 Protected Paths Contract

The following files and directories are
**never** mounted into or visible from any
sandbox's mount namespace, and never
resolvable through symlinks inside the
sandbox:

- The policy file (`~/.aios/policy.yaml` or
  the operator-configured path).
- The redaction rules (`~/.aios/redaction.yaml`).
- The retention rules (`~/.aios/retention.yaml`).
- The role definitions (`~/.aios/roles/...`).
- The rule store (`~/.aios/rules/...`).
- The credential broker's data
  (`~/.aios/creds/...`).
- The audit stores
  (`~/.aios/audit/...`).
- The replay store
  (`~/.aios/replay/...`).
- The control-plane socket
  (path declared by `aios config
  control-socket-path`, with no symlink
  resolution from inside the sandbox).
- `AGENTS.md` (or its operator-configured
  equivalent) — the project-level agent
  instructions, treated as control-plane data.
- `var/models.yaml` and any other
  `var/*.yaml` whose name matches the
  control-plane config schema. The `var/`
  tree is **always** outside the sandbox's
  mount, not "outside some of it."
- The aios binary's own configuration
  (`~/.aios/config.yaml`).

The sandbox's mount is rooted at the project
directory, and symlink resolution is scoped
to that root. The sandbox runtime asserts on
startup that none of the protected paths is
reachable through the mount; the assertion
is part of the conformance probe in
§12.5.4. The control plane tests in §14
include adversarial cases where the agent
attempts to traverse a symlink that points
out of the project root.

### 11.14 Trust Boundary and Idempotency Contract

**Untrusted content is data, never
instructions.** Any string the system reads
from a network response, a tool result, a
file outside the project, a third-party
repository, a documentation page, an issue
tracker comment, or an environment variable
is treated as data, not as instructions for
the model. The orchestrator does not allow
external content to redefine the policy, the
role, the spec, the acceptance criteria, the
allowed tools, or the audit rules. The
adversarial probe set in §12.5.7 includes
prompt-injection probes that attempt
exactly this; the block rate is part of the
safety gate.

**Idempotency keys.** Every effect path
shall take an idempotency key derived from
the decision row's `content_hash`. The
effect executor shall verify, on a retry or
replay, that the key has the expected
result by reading back the resource
where the executor can. For external
systems that do not support idempotency
keys natively, the executor records a
"best-effort" mark in the audit row and
flags the row as `idempotent_partial` in
the report. The orchestrator does not retry
a tier 2 or tier 3 effect whose idempotency
key is unknown to the executor; the operator
is asked.

## 12. Success Criteria

The aios project is successful when the following
criteria are met. The criteria are behavioural
gates, not coverage targets. Coverage is a
guideline; behaviour is the gate.

### 12.1 Quality Gates (from §9.2)

The quality gates are behavioral targets with
concrete numbers. A gate that the operator can
satisfy by raising their own budget is a
configuration knob, not a gate, and is not
listed here.

- The **false-success rate** is below 5
  percent across the bench suite. A
  *false success* is a task that the bench
  labels `pass` and the orchestrator labels
  `pass`, but a human review (or the bench
  grader's hidden test subset) labels
  `fail`, or a task the bench holds as
  belonging to the **negative set** (a task
  the agent must refuse or escalate) and the
  agent returns a `pass` instead.
- The bench suite includes a held-out
  **negative set** (tasks the agent must
  refuse or escalate). The false-success
  rate on the negative set is zero, subject
  to the sample-size rule below. The
  negative set is the bench refusal set, not
  the safety probe set; the two are
  distinct.
- The cost per verified task is below the
  cost target in `aios.toml` (`[gates]
  cost_per_verified_task_usd`, **initial
  default 0.50, to be replaced by the
  measured v0.9 baseline before v1.0**).
  The target is a fixed number, not "the
  developer's declared budget." The `aios
  stats` view shows the current per-task cost
  and the target. **The defaults listed in
  this PRD are initial values, not
  release-pinned.** A number in this PRD
  becomes a release gate when the
  implementation plan says it is locked and
  the bench shows a defensible margin around
  it; until then the number is a placeholder.
- The time per task is below the time target
  in `aios.toml` (`[gates]
  time_per_task_seconds`, **initial default
  600, to be replaced by the measured v0.9
  baseline before v1.0**). The target is a
  fixed number, not "the developer's declared
  limit." Wall-clock time is measured at the
  deadline layer. The same placeholder rule
  applies.
- The system resumes from a crash without losing
  in-flight state, and the resume is verified
  by fault-injection tests.

The cost and time **per-run** ceilings remain
operator-configurable in `aios.toml` (e.g.
"abort a single run after $X or Y minutes").
Those are budget knobs, not product gates, and
live in §10.5.

### 12.2 Safety Gates (from §9.3)

- The approval queue pauses for human input on
  the first tier-2 action in a new scope
  (network, new connector scope, new
  destination). Subsequent tier-2 actions
  with the same `action`, same target
  pattern, and same task scope are approved
  per-scope (§11.9) and do not pause. Tier 3
  actions pause on every action (§11.5).
- Tier-3 actions are blocked when the gateway
  cannot reach the human (no approval channel
  available).
- Tier-2 actions require once-per-task-scope
  approval (per-scope class in §11.9), not
  per-call.
- The adversarial suite (sandbox-escape, egress
  bypass, policy bypass, prompt-injection,
  approval-replay, cost-exhaustion, fault-
  injection) has a 100 percent block rate on
  the **protected actions**. The protected
  actions are the actions the system must
  refuse; they are *not* the bench **negative
  set** (tasks the model must refuse or
  escalate). The two sets are distinct and
  must not be conflated; see §12.5.7 and
  §12.1.
  **Sample-size rule.** A "100% block rate"
  is meaningful only if the suite has enough
  probes for the result to be statistically
  defensible. The protected-action suite
  shall have **at least 60 probes per
  category** (sandbox-escape, egress bypass,
  policy bypass, prompt-injection,
  approval-replay, cost-exhaustion, fault-
  injection). With 60 probes and zero
  observed failures, the 95% one-sided
  upper bound on the true failure rate is
  approximately 5% (rule of three); the gate
  therefore claims "≤5% with 95%
  confidence," not "zero." A category with
  fewer than 60 probes is a configuration
  failure, not a gate; the bench fails to
  start until the operator adds probes.
  The probe set is the negative set, the
  protected set, the safety set — three
  names for the same thing — and is distinct
  from the bench refusal set in §12.1.
- The kill switch takes effect within one
  second of invocation on the in-flight tool
  call (SIGKILL to the sandbox PID; in-flight
  tool result marked `cancelled`, not
  `pass`; see §14).

### 12.3 Reproducibility Gates

Replay equivalence is pinned to a single
definition, REPL-EQ-001, to prevent the same
word meaning three different things in §5,
§11.11, and §12.3.

**REPL-EQ-001** Two runs of the same plan are
*replay-equivalent* if, given the recorded
model inputs, the recorded model outputs, the
recorded tool results, the recorded approvals,
the same plan id, the same policy snapshot, the
same pin set, and a pinned sandbox image, the
**sequence of decision rows** in the audit log
matches **bit-for-bit** (every `content_hash`
and every `redacted_hash` in the same order
with the same `action`, `scope`, and `tier`).
Wall-clock timestamps, `chain_hash` values,
external effect side effects, and live network
outcomes are **not** part of the equivalence
check. This is the only "replay" the system
guarantees.

- The audit log is hash-chained and verifiable.
  The hash chain is verified by `aios audit
  verify` on every run. The redacted chain and
  the encrypted replay chain are verified
  independently.
- The replay of any run, given the recorded
  artifacts in §12.5.8 (still within the
  replayability window) and the same plan id,
  same policy, same pin set, and a pinned
  sandbox image, satisfies REPL-EQ-001.
- The system runs equivalently on two hosts
  given the same recorded artifacts, the same
  plan id, the same policy, the same pin set,
  and a pinned sandbox image with pinned
  dependency versions. Identical `chain_hash`
  values are **not** required across hosts
  (timestamps are recorded from the decision
  clock and may differ).
- Live re-execution (re-running the run with
  fresh model calls) is not a reproducibility
  gate. The variance is measured, not
  eliminated. The recorded outputs are the
  source of truth.

### 12.4 Operational Gates

- The documentation set is complete for every
  shipped module. Each module has a PRD, SRS,
  TRD, APP-FLOW, WIREFRAMES, and
  IMPLEMENTATION-PLAN.
- The test suite has a 100 percent pass rate
  on the **safety** suite (the protected
  actions, §12.2): zero failures, subject to
  the sample-size rule in §12.2 (≥60 probes
  per category; the gate is "≤5% with 95%
  confidence," not "zero"). The test suite
  has a 90 percent pass rate on the
  **bench** suite as the initial release
  milestone; the long-term gate on the bench
  is the false-success rate in §12.1 (≤5%
  overall, 0% on the negative set with the
  sample-size rule). The bench and the
  safety suite are distinct (§12.2).
- The test suite has behavioural coverage of
  the public surface, not line coverage. Line
  coverage is tracked as a guideline, not a
  gate.
- The 20-task wedge is the *initial* bench,
  not the long-term quality gate. The wedge
  tasks are chosen by the developer, the
  hidden tests are written by the developer
  before the agent sees the tasks, and the
  bench is extended as the system proves out.
  The 90-percent threshold is the initial
  milestone; the long-term gate is the
  false-success rate, not the pass rate.

## 12.5 Required Subsystems (from the blueprint)

The subsystems in this section are core to the
blueprint and are in scope for the PRD. Each
subsystem has its own module in the dependency
graph. A subsystem that is not in this section
is out of scope for the prototype and is
listed in §13.

### 12.5.1 Credential Broker and Secrets

The credential broker is the only place where
long-lived secrets live. The model and the
agent code never see a long-lived secret and
never see a long-lived API key. The broker
injects short-lived tokens at the **egress
proxy** (the connection boundary), so the
token appears only in the outbound HTTP
headers and only at the moment a request is
made. The model and the agent see the
credential **id** (`cred_xyz`) and the
provider; they never see the token value.

The broker issues tokens on demand (per
outbound call) or pre-issues them with a
short TTL (default 5 minutes, configurable
per credential). Tokens are scoped to a
single egress destination and a single
provider. The broker rotates secrets on a
schedule (default 30 days) and rotates
tokens on each use unless the call is
idempotent and the token is still valid.

**Revocation.** Revocation is an admin
action, not a tier 3 model-side action. The
command is `aios creds revoke <id>` and is
control-channel-only. Revocation takes
effect on the **next decision** that needs
the credential: the in-flight HTTP call
completes with the token it already holds,
the next call finds the token revoked and
fails closed. If a kill is also in effect,
the in-flight call is terminated per §14.
The broker writes a `revoke` row to the
audit log with the credential id, the
revoker, and the timestamp from the
decision clock. The model-side request
that "asks" the broker for a token is a
tier 0/1 action — the broker decides
whether to issue or refuse; the model does
not see the secret and cannot influence
broker policy.

**Audit.** Secrets are never written to the
workspace. Secrets are never written to the
audit log. The audit log records the
credential id, the provider, the action,
the request hash, and the timestamp. The
secret is never a field in the audit row.

**Visibility.** The broker is the source of
truth for what credentials exist, when they
were issued, when they were last used, and
when they were revoked. The developer can
`aios creds list` to see the broker's view.
The developer can `aios creds show <id>` to
see metadata (provider, scope, expiry),
never the secret.

### 12.5.2 Stop, Pause, and Kill Switch

Three controls. Different semantics.

- **Pause** (`aios pause`) — the current run
  pauses. The orchestrator holds state. The
  developer can `aios resume` to continue.
  Pause is cooperative; an in-flight tool
  call finishes before the pause takes
  effect.
- **Stop** (`aios stop`) — the current run
  stops. The orchestrator writes a stop row
  to the audit log. The run is closed. The
  developer can `aios resume <session_id>`
  to start a new run from the last
  checkpoint.
- **Kill** (`aios kill`) — global kill
  switch. All runs in all sessions stop. The
  orchestrator writes a kill row to the
  audit log. The kill takes effect within
  one second. The kill is control-channel-
  only.

The kill switch is a precondition for
autonomy. The user can grant an agent more
autonomy (larger `allow-tier`, more tools,
more capabilities) only when the kill switch
is reachable. Reachable means: a control
channel is bound, the token is valid, the
operator is present. The orchestrator
verifies reachability on every run start and
on every tier-3 action.

### 12.5.3 Task Spec Confirmation

Every run starts with a spec. The spec is a
typed value with these required fields, all
of which must be present or the spec is
rejected:

- **goal** — the natural-language statement
  of the task.
- **acceptance criteria** — a list of
  conditions, each of which is observable and
  checkable. The verify layer (§11.7) checks
  these; without them the run is rejected.
- **tool scope** — the set of tools, with
  their tier and target patterns, that the
  task may use. Tools outside the scope
  cause the gateway to return `deny`.
- **risk tier** — the maximum tier the task
  may exercise. A task that needs a higher
  tier than the spec declares is rejected.
- **budgets** — the per-task cost ceiling
  and the per-task wall-clock ceiling.
  Defaults are read from `aios.toml`; the
  spec may tighten but not loosen the
  defaults.
- **constraints** — natural-language
  constraints the agent must respect
  (e.g. "do not modify migrations").
- **out-of-scope** — items the agent must
  not touch.
- **test plan** — the project's tests, the
  hidden test subset (§11.7), and any
  acceptance test the developer wants the
  verify layer to run.

If any of these fields is missing or empty,
the orchestrator refuses to start the run
and reports the missing fields to the
operator. Confirmation of a complete spec is
control-channel-only.

The verify layer has nothing to verify
without acceptance criteria. The spec is the
contract.

A spec is a typed value. A spec is content
addressed. Two specs with the same content
have the same id. A spec revision creates a
new spec with a new id. The orchestrator
rejects a spec revision from inside the
sandbox.

The spec is the unit of "what does done
mean?" The spec is the unit of "what does
blocked mean?" A task is blocked when the
agent cannot meet an acceptance criterion,
not when the agent decides to give up.

**No-progress detector.** The orchestrator
runs a no-progress check: if a task has
consumed more than its per-task budget
without writing any audit row whose action
made progress (a model call, a tool call
that returned a non-error result, a
verifier pass), the task is marked
`stalled` and the operator is notified. A
stalled task can be paused, killed, or
restarted from the last checkpoint; the
audit log records the stall and the
operator's choice.

### 12.5.4 Bring-Your-Own-Model and Model Governance

The system supports multiple model providers
and multiple model classes. The model
registry is content addressed. The model
list is in `var/models.yaml`. The developer
can `aios model add` to register a model.
The developer can `aios model probe` to run
the conformance probe suite.

The conformance probe suite is run on
synthetic data only. The probe set includes:
latency, schema conformance, error
behaviour, injection refusal, long-context
recall, and tool-call format. The probe
results are recorded per (model, prompt
version). The probe is not the bench; the
bench is the developer's own task set.

The provider allowlist is per-project. The
developer can `aios provider set-allowlist`
to restrict which providers the agent may
call. The developer can `aios provider add`
to register a new provider. The first time
a provider is added, the system shows a
first-use data-leaves-machine notice and
requires explicit confirmation.

The model adapter layer abstracts the
provider. The layer supports
`temperature` when the provider supports it
and records the absence when it does not.
The layer never silently substitutes a
parameter.

### 12.5.5 Custom Agents (AgentSpec)

The system supports custom agents defined
by an AgentSpec. An AgentSpec declares the
agent's goal, its tool set, its risk tier
ceiling, its prompt prefix, its evaluation
harness, and its permission diff against
the default agent. The custom agent builder
is `aios agent build`, an AI-assisted
tool that drafts an AgentSpec from a
natural-language description.

A custom agent is enabled through
`aios agent enable <name>`. The enable
command shows the permission diff against
the default agent. The enable is
control-channel-only. An agent cannot enable
itself. An agent cannot update its own
AgentSpec.

Premade agents ship with the system:
`code-review`, `refactor`, `test-writer`,
`docs-writer`, `bench-runner`. Each premade
agent has a published AgentSpec, a
permission diff, and (in Phase 4) an eval
harness. **The admission rule** for a
premade agent is "the eval harness is green
on the bench" — that rule cannot be applied
in Phase 1-3 because the harness is Phase
4. **Therefore: in Phase 1-3, the premade
agents ship with their AgentSpec and
permission diff but without an admission
gate.** They are documented as
"preliminary" and the report flags any
premade agent without a green harness as
"unverified at admission." The admission
gate becomes binding at Phase 4.

For the prototype, custom agents are
**explicitly in scope but deferred to
Phase 4**. The Phase 1-3 system ships with
the default agent, the premade agents
(preliminary, no admission gate), and the
AgentSpec validator. The custom agent
builder, the eval harness, and the
admission gate ship in Phase 4.

### 12.5.6 Daily Cost Cap and Provider Data Governance

A daily cost cap is exposed in
`aios optimizer set-daily-cap <usd>`. The
cap is enforced at the model adapter layer
across all sessions of the current user on
the current host. The cap is recorded in
the audit log. The cap is reset at 00:00
local time.

Provider data governance is the
developer's responsibility. The developer
can `aios provider show <name>` to see
what data the provider receives (prompts,
tool calls, tool results, file contents,
URLs). The first-use data-leaves-machine
notice is shown when a provider is added.
The notice lists what data leaves the
machine and what the provider's data
retention policy is. **The vendor's stated
retention and training policy is labeled
"as stated by the vendor, unverified" in
this notice; the system cannot verify the
vendor's claim and the label appears
alongside every field that comes from a
vendor.** The notice is recorded as an
audit row.

### 12.5.7 Adversarial Test Plan and Head-to-Head Bench

The adversarial test plan is a separate
test suite that probes the system the way
an attacker would. The plan covers, and
this list is exhaustive for v1.0:

- **sandbox-escape** (T3 in §14)
- **egress bypass** (SSRF and unapproved
  destinations)
- **policy bypass** (rule precedence,
  hard-deny override attempts, scope
  widening)
- **prompt-injection** (direct and indirect;
  untrusted-content rule in §11.14)
- **memory poisoning** (artifacts that try
  to redefine the spec, the role, the
  policy)
- **model-call manipulation** (tool-result
  injection, replayed model outputs,
  cross-session carryover)
- **credential exfiltration** (broker
  visibility, token leak, prompt-injection
  path to creds)
- **supply chain** (T5 in §14)
- **cross-project leakage** (T7 in §14)
- **approval replay** (an attempt to replay
  a recorded approval for a new action)
- **cost exhaustion** (daily cap bypass,
  per-task cap bypass, provider-data
  inflation)
- **fault injection** (kill mid-flight,
  store-unavailable, clock skew, partial
  writes)

The plan runs on synthetic, red-team
inputs. The plan's protected action set is
the safety probe set — actions that must be
denied or escalated. The block rate on the
safety probe set is the safety gate. The
safety probe set is distinct from the bench
**negative set** (refusal tasks in §12.1)
and from the bench **positive set** (tasks
the agent must solve). The three sets are
named distinctly: safety probe set, bench
negative set, bench positive set. The
sample-size rule in §12.2 applies to the
safety probe set.

The head-to-head bench compares aios against an
external subject in headless mode. The peer
set is the set of headless-mode coding agents
the user can run on a Linux host with a typed
adapter: Aider, Cursor (CLI), Continue
(headless mode), Cline, Codex CLI, Claude
Code CLI, and any vendor CLI that exposes a
headless mode and a typed interface for
inputs and outputs. **Grok Bot is not in the
peer set.** Grok Bot is a cloud-agent surface
for delegated cross-tool work; it is not a
headless coding agent, it does not expose a
model picker, and it has no documented
headless mode. A subject is added to the
peer set only after the adapter layer can
exercise it on the bench task set in CI.

**Default arm — each product's default.** The
external subject runs on the same task set, the
same inputs, and the same wall-clock budget, with
the subject's own default configuration. The
comparison records pass rate, false-success rate,
cost per verified task, and time per task. This
arm is the one that answers the user-facing
question "which product is better for me."

**Same-model arm — only where the product allows.**
Where the external subject exposes a model
selector (Aider, Cursor, Continue, Cline,
Codex CLI), a second arm pins both products to
the same model class and re-runs. Where the
subject does not expose a selector (Claude Code
CLI), the same-model arm is skipped and the
report records "same-model not applicable." A
run with a subject that is hard-tied to a
vendor (e.g. Claude Code CLI to Anthropic) is
reported with the vendor's name in the row,
never as "Claude Code vs. aios on a neutral
model."

**Significance rule.** A claim that one product
beats another on the head-to-head requires two
consecutive full runs in which the result holds
and a **one-tailed paired sign test** on the
per-task outcomes (treating ties as
non-informative) reaches p < 0.05. The test is
one-sided: the claim is "product X has a
strictly higher pass rate than product Y on
this task set," not "their pass rates differ."
With the current task set (20 tasks), p < 0.05
is reached only when **at least 15 of 20
discordant tasks favor the leader** (one-tailed
binomial, exact); 14 of 20 is not significant
either way. Ties reduce the effective sample
size. A result that does not clear the bar is
reported as "trend, not significant" and is not
used for product claims. The bench exposes the
raw per-task outcomes so the user can re-test
with a different statistic. The "trend" label
is reserved for cases where the leader wins
more than half the discordant tasks but the
count is below 15; otherwise the result is
"inconclusive."

The head-to-head runs through `aios bench
compare --external <subject> --arm default` or
`--arm same-model` (where applicable).

### 12.5.8 Audit Retention, Redaction, and Replay Store

The audit log is append-only. The hash chain is
preserved.

**Two stores.** The system keeps two
cryptographically separate stores, each with
its own chain, its own `sequence_number`
counter, its own key, and its own retention:

- **Redacted store** — the audit log as
  described in §10.2 and §11.2. The
  redaction layer strips secrets, PII, and
  file contents over a size threshold before
  any write. The redaction rules are in a
  location not mounted into the sandbox
  (e.g. `~/.aios/redaction.yaml`); writes go
  through the control channel only.
  Redaction is a pure function of the row and
  the rule set: the same row under the same
  rules produces the same redacted row.
- **Encrypted replay store** — the unredacted
  model inputs and outputs and the tool
  results needed for replay (§11.11). This
  store is encrypted at rest with a key held
  by the developer, not by aios. The store
  has its **own** retention, configured
  separately. The default is the same 90
  days as the redacted store. The two stores
  must not share a key and must not share a
  retention schedule without an explicit
  config flag.

A row's redacted `chain_hash` and the
matching unredacted `chain_hash` in the
replay store are linked by `(session_id,
sequence_number, content_hash)`. They are
not chained to each other; each store
chains only within itself.

**Truncation anchor direction.** Retention
deletes the **oldest** rows and keeps the
newest. The signed truncation anchor commits
to the **removed prefix** (a Merkle root over
the deleted rows, the count removed, and the
**chain hash of the last deleted row in that
store**, which by construction is the row
whose `sequence_number` is one less than the
first retained row). The retained suffix is
still verifiable from the anchor forward in
the same store. Truncation is applied
independently to each store; an anchor in the
redacted store is not an anchor in the replay
store, and the two anchors are linked by
`session_id` and by the fact that the
replay-store anchor's `sequence_number` is
greater than or equal to the redacted-store
anchor's `sequence_number` (the replay store
may lag if a row is being written). Truncation
is control-channel-only.

**Replay store retention and replayability
window.** When the encrypted replay store
ages out, replay fails with a clear message
("replay artifacts for this session have aged
out"). A redacted audit row survives past
that point for review, but the session is no
longer replayable. The bench and the report
distinguish "audit-only" sessions from
"fully replayable" sessions. The redacted
store's `chain_hash` is still verifiable
after replay artifacts have aged out; the
replay store's `chain_hash` is verifiable up
to its last truncation anchor.

**External export (assurance escalation).** A
background job, `aios audit export`, mirrors
both chains to a destination the host
control plane cannot rewrite (S3 with object
lock, an append-only NFS share, a notarized
log service). The job runs on a schedule and
on-demand. The exported chain is a
forward-only mirror: it does not import
back. The on-host chain is the live chain;
the export is the witness. Without an
external witness the chain is tamper-evident
only (see below); with the export it
becomes tamper-evident at a stronger
assurance level. The export is best-effort
by default; the operator can mark it
required and have the run fail if the export
fails.

**Signing key holder and trust model.** The
signing key for the audit chain and the
truncation anchor is held in the control plane
(per-user key, on the host, in a
control-channel-only file). This makes the
chain **tamper-evident**, not tamper-proof:
an attacker with control of the host can
rewrite history and re-sign. The trust model
is honest about this. For higher assurance,
the external export above is the practical
mitigation; a future revision may move the
key to a TPM, an HSM, or a third-party
signer.

**Retention rules.** Retention is configured
in a location not mounted into the sandbox.
The default is 90 days for both stores. The
operator can change the default. Retention
runs on a schedule; truncation anchors are
written when retention fires. Setting
replay-store retention shorter than
redacted-store retention is allowed; the
opposite is also allowed; the report
distinguishes the two windows.

**Provider data governance notice.** When the
operator registers a provider, the notice
displays the **vendor-stated** retention and
training policy. The system cannot verify
the vendor's claim; the field is labeled
"as stated by the vendor, unverified." The
same label is shown in §12.5.6 (the daily
cap and provider governance screen).

## 13. Out of Scope

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

### 13.1 Module Sprawl (deferred)

The blueprint's MVP is a thin loop with a few
components. The 25-module monorepo is a Phase
4 target, not a Phase 1 target. The TUI is
deferred until after the core loop is
validated. Several modules in the current
design — `fanout`, `watch`, `image`, `retry`,
`ask`, `report` — are tools inside `aios-tools`
or subpackages inside the larger modules until
they are actually needed.

The MVP ships **twelve modules**:

1. **aios-core** — orchestrator, state machine,
   planner, code-first verifier, content-hash
   audit row type, redacted audit store.
2. **aios-persist** — checkpoints, handoffs,
   session history, encrypted replay store,
   retention + truncation anchors.
3. **aios-policy** — policy gateway, always-allow
   rule storage (tier 0/1 only — tier 2 has
   per-scope approval but no standing rule;
   tier 3 is per-action), redaction rules,
   retention rules.
4. **aios-models** — typed adapters, the
   conformance probe, the model class table.
5. **aios-optimizer** — model router and cost
   optimization (a pure function of the cost /
   quality table, the override, and the
   budget).
6. **aios-approvals** — approval service and
   kill switch. The approval service
   **consults** the always-allow rule store
   in aios-policy but does not own it.
7. **aios-cred** — credential broker and
   secrets.
8. **aios-sandbox** — sandbox runner, egress
   proxy, sandbox image management.
9. **aios-memory** — content-addressed memory
   (raw / indexed / compacted layers).
10. **aios-tools** — tool bus, built-in tools
    (`grep`, `glob`, `code-search`,
    `diagnostics`, `edit`, `git`, `read`,
    `write`, `image`, `ask`, `retry`,
    `run_tests`, `report`).
11. **aios-cli** — CLI and control channel.
    The TUI is **not** part of MVP.
12. **aios-bench** — benchmark harness,
    adversarial suite, head-to-head.

The Phase 4 monorepo is the **25-package**
list in §9 (`core`, `policy`, `sandbox`,
`egress`, `models`, `optimizer`, `approvals`,
`cred`, `persist`, `memory`, `tools`, `search`,
`edit`, `git`, `watch`, `image`, `report`,
`harness`, `bench`, `fanout`, `cli`, `tui`,
`audit`, `ask`, `retry`). Twelve of those ship
as MVP modules; thirteen ship later (the
remaining ones are `egress` — folded into
`aios-sandbox` for MVP, `harness` — folded into
`aios-bench`, `audit` — folded into `aios-core`
and `aios-persist` per the split above,
`search` and `edit` — folded into `aios-tools`,
`watch` — deferred, `image` — a tool, `report`
— a tool, `fanout` — deferred, `tui` —
deferred, `ask` — a tool, `retry` — a tool).

**§9 → MVP module mapping.** Every §9 function
is owned by exactly one module. The audit log
is split: aios-core owns the row type and the
redacted store; aios-persist owns the
encrypted replay store, checkpoints, and
retention. The split prevents the "who owns
audit" ambiguity. The always-allow rule
storage is owned by aios-policy; the approval
service in aios-approvals consults but does
not own the rules.

| §9 function | MVP module |
| --- | --- |
| Typed schemas, state machine (§9.1) | aios-core |
| Audit row type, redacted audit store (§11.2) | aios-core |
| Plan + verify execution (§9.3 planner) | aios-core |
| Code-first verifier, hidden test subset, diff-scope check (§11.7) | aios-core |
| Checkpoints, handoffs, session history (§9.1) | aios-persist |
| Encrypted replay store, retention, truncation anchors (§12.5.8) | aios-persist |
| Policy gateway, redaction rules, retention rules (§9.1) | aios-policy |
| Always-allow rule storage (§9.1, §11.9) | aios-policy |
| Sandbox runner, egress proxy, sandbox image management (§9.1) | aios-sandbox |
| Model adapters, conformance probe (§9.1) | aios-models |
| Model router, cost optimization (§9.1) | aios-optimizer |
| Approval service, kill switch (§9.1) | aios-approvals |
| Credential broker (§12.5.1) | aios-cred |
| Content-addressed memory (§9.1) | aios-memory |
| Tool bus + all built-in tools: `grep`, `glob`, `code-search`, `diagnostics`, `image`, `ask`, `git`, `retry`, `run_tests`, `report`, `read`, `write`, `edit` (§9.2) | aios-tools |
| Orchestrator (§9.3) | aios-core |
| Benchmark, adversarial suite, head-to-head (§9.3, §12.5.7) | aios-bench |
| CLI, control channel (§9.4) | aios-cli |
| TUI (§9.4) | deferred (post-MVP) |
| File watcher (§9.2) | deferred (post-MVP) |
| Parallel agents / fanout (§9.3) | deferred (post-MVP) |

Each module is small enough that one person can
hold it in their head. The other 13 packages
ship in later phases, after the Phase 1 wedge
is validated by the bench.

## 14. Risks

The aios project has the following risks. The risks
are tracked in the implementation plan.

- The model quality is below the threshold for
  some tasks. The mitigation is the model router
  and the cost/quality table.
- The cost is above the budget. The mitigation is
  the cost tracker and the budget meter.
- The approval fatigue makes the operator ignore
  the queue. The mitigation is **never** to
  weaken the tier model or to broaden the
  auto-allow surface. The mitigations are
  batching (group identical-scope requests in
  one prompt), measurement (queue depth and
  decline rate are reported per session), and
  tightening (if a scope recurs, the operator
  is encouraged to make the policy match the
  reality, not to make the queue match the
  policy). The auto-allow rules remain a
  tier 0/1/2 surface only.
- The audit log is too large to inspect. The
  mitigation is the report generator and the
  search tool.
- The hash chain breaks. The mitigation is the
  re-anchor tool and the audit log repair.
- The model drifts between runs. The mitigation is
  best-effort `temperature` when the provider
  supports it, the call fingerprint, the
  cost/quality table, the recorded model output
  on every call, and the recorded variance on
  the bench. The drift is measured, not
  eliminated.
- The replay diverges from the original run. The
  mitigation is recorded-output replay (never
  re-call the model), the content-addressed
  checkpoints, the immutable plan, and the
  pinned sandbox image. The replay is a pure
  function of the recorded outputs and the
  pinned image.
- The agent edits its own policy. The mitigation
  is that the policy, roles, and rules live
  outside the sandbox and are writable only
  through the control channel. The sandbox
  cannot reach the policy file. The agent
  cannot call `aios policy set`,
  `aios allow add`, or any control-channel-only
  command, including the cost controls
  (`optimizer set-daily-cap`,
  `optimizer set-per-task-cap`) and the
  bench overrides (`aios bench conformance
  override`). The mechanism is a Unix domain
  socket under a path that is **not** mounted
  into or visible from any sandbox's mount
  namespace. The socket accepts a one-shot
  token bound to (approval hash, expiry,
  principal); the token is delivered to the
  operator out of band (e.g. a separate
  terminal multiplexer pane) and the agent
  does not hold it. File mode on the socket is
  0600, but that is not the security boundary
  — the boundary is that the sandbox cannot see
  the path. The control plane tests include
  adversarial cases where the agent tries to
  (a) enumerate the parent directory, (b) read
  the socket inode via `/proc`, (c) read the
  token from any environment variable or
  argv, and (d) call a control-channel-only
  command by name. Each of these is asserted
  to fail.
- The agent approves its own action. The
  mitigation is that every approval command
  (`aios approve`, `aios spec confirm`,
  `aios spec edit`) is control-channel-only.
  The sandbox cannot reach the approval
  channel. The approval row records the
  channel and the principal.
- The always-allow rule covers a tier 3
  action. The mitigation is the gateway: any
  `aios allow add` rule that matches a tier 3
  action is rejected at insertion time, not
  at action time. The rejection is enforced
  by code, not by convention.
- The memory is poisoned. The mitigation is
  provenance on every artifact, control-
  channel-only pin/unpin/prune, the
  observable compacted view, and the
  adversarial probe set for memory poisoning.
- The false-success rate is hidden. The
  mitigation is the bench's held-out negative
  set, the false-success report per model and
  per agent, and the head-to-head bench.
- The custom agents are out of scope but
  silently used. The mitigation is the
  explicit Phase 4 deferral in §12.5.5 and
  the AgentSpec validator in Phase 1-3
  (the system can validate an AgentSpec even
  though it cannot yet build one).
- **Sandbox escape (PRDR-3).** A kernel or
  container primitive that lets a tool call
  reach the host. The mitigation is the
  sandbox runner's seccomp/AppArmor profile,
  the no-`privileged` flag, the read-only root
  filesystem, the egress proxy that mediates
  every outbound connection, and the
  adversarial suite's sandbox-escape probe
  set. **Risk ID namespace note:** the
  blueprint's `T#` IDs and this PRD's
  `PRDR-#` IDs are different namespaces. The
  same letter and number (e.g. T6) refers
  to different risks in the two documents.
  Use the prefix to disambiguate.
- **Dependency supply chain (PRDR-5).** A
  poisoned or backdoored dependency enters
  the project through `go get`, an image
  pull, or a tool binary. The mitigation is
  the locked toolchain (`go.sum` checked,
  pinned Go version), the digest-pinned
  sandbox image (no `latest` tag), the
  vendor directory for the agent's own
  modules, the image pull-by-digest for any
  model artifact, and the adversarial
  suite's supply-chain probe set.
- **SSRF (PRDR-6).** A tool call reaches a
  metadata endpoint, an internal cloud
  service, or a loopback address it was never
  meant to reach. The mitigation is the egress
  proxy's allowlist of destinations, the
  block on link-local and loopback ranges
  (169.254.0.0/16, 127.0.0.0/8, ::1, the
  cloud metadata IPs), DNS resolution done by
  the proxy not the sandbox, and the
  adversarial suite's SSRF probe set.
  **Loopback, link-local, and metadata-IP
  destinations are not approvable.** A
  per-scope human approval for a tier 2
  network action cannot approve any of these
  destinations; the gateway rejects the
  approval at insert time. The allowlist
  itself is the only way to add a destination.
- **Cross-project leakage (PRDR-7).** A run
  on project A reads or writes project B
  because the sandbox's working directory is
  the
  user's home, the path is a symlink that
  crosses projects, or the tool resolution
  follows a shared directory. The mitigation
  is that the sandbox is rooted at the
  project directory, symlink resolution is
  scoped to the project, the egress proxy
  cannot read files outside the project, and
  the adversarial suite's cross-project probe
  set asserts no tool call can resolve a path
  outside the project's mount.
- **Kill in-flight tool call.** Kill must take
  effect within one second (§12.2). A
  long-running tool call (a test run, a build,
  a network fetch) may already be in flight.
  The kill handler cancels the orchestrator
  context, sends `SIGKILL` to the sandbox
  PID, terminates the egress proxy
  connection, and writes a `kill` audit row
  with the in-flight tool's id. The tool
  result that was running is marked
  `cancelled`; its partial output is not
  treated as a verified result. Pause is
  cooperative (it waits for the current tool
  to finish) and is a different gate.
- **Provider notice honesty.** The provider
  notice at registration time lists the
  vendor's stated retention and training
  policy. The system cannot verify the
  vendor's claim. The notice labels the
  fields "as stated by the vendor, unverified"
  and links to the vendor's public policy
  page. A user who needs verified treatment
  is referred to a self-hosted model
  (ollama-style) and a paid enterprise tier
  with a signed DPA.
- **Conformance gating.** The conformance
  probe (§12.5.4) records a per-model
  profile (pass/fail on each of the probe
  categories in §12.5.4). The role
  registry declares a role's required
  `min_model_profile` (the set of probe
  categories the role's model must pass).
  At task start, the role is rejected if
  **no** available model meets the
  `min_model_profile` for the role. The
  probe records what each model can do; the
  role declares what it needs; the gap is
  detected at task start. The role registry
  is the source of truth for which model
  class may fill which role; the optimizer
  and the router both consult it. The role
  is **not** rejected if at least one model
  meets the bar (the system picks one);
  it is rejected only if zero do.
- **Verify runtime bound.** The 10-second
  self-check limit in §10.1 applies to the
  lightweight "self-check" tool (a syntactic
  and schema check). The `run_tests` tool
  invokes the project's test suite and is
  bounded by the project's test timeout plus
  5 seconds, not by the 10-second self-check
  budget. The verify layer records which
  tool produced the result and the bound
  that was applied.

## 15. Version

This document is version 1.2. The previous
version was 1.1.

Changes from 1.1 to 1.2:

- §11.2 split the audit row into a
  pre-redaction `content_hash`, a
  post-redaction `redacted_hash`, and a
  per-store `chain_hash`. The row type is
  in `aios-persist`; the timestamp is read
  from the decision clock (not "the injected
  clock" generically).
- §11.11 DET-001 clarified that tier 0/1
  tool results are also stubbed from the
  recording; DET-006 dropped the "cosmetic
  changes" carve-out (a content-addressed
  plan has no decision-irrelevant byte).
- §11.12 added the Fail-Closed Contract
  (audit, gateway, approval channel, replay
  store, signing key unavailability).
- §11.13 added the Protected Paths Contract
  (a single source-of-truth list of paths
  the sandbox can never reach, including
  `AGENTS.md` and the `var/` tree).
- §11.14 added the Trust Boundary and
  Idempotency Contract (untrusted content is
  data; idempotency keys on every effect).
- §12.3 pinned replay equivalence to a
  single definition (REPL-EQ-001).
- §12.5.7 dropped Grok Bot from the peer
  set; expanded the adversarial probe set
  to an exhaustive list; added the
  one-tailed sign-test rule with the
  "15 of 20" threshold.
- §12.5.8 added the external export (a
  witness the host cannot rewrite) and
  parallel truncation anchors; tightened
  the truncation-anchor reference rule.
- §12.5.1 rewrote the credential broker
  so the model never sees a token; tokens
  are injected at the egress proxy.
- §12.1 defined "false success" explicitly
  and clarified the bench negative set is
  distinct from the safety probe set.
- §12.2 added the sample-size rule (≥60
  probes per category; "≤5% with 95%
  confidence") and clarified the tier 2
  pause (first action in a new scope, then
  per-scope).
- §12.4 split the test suite gate: 100%
  pass on the safety suite (with sample-
  size rule) and 90% pass on the bench
  (initial milestone).
- §12.5.3 added the spec field list
  (acceptance criteria, tool scope, risk
  tier, budgets) and refused to start
  without all of them; added the no-
  progress detector.
- §12.5.5 deferred the premade-agent
  admission gate to Phase 4 (the eval
  harness is Phase 4).
- §12.5.6 added the "as stated by the
  vendor, unverified" label.
- §13.1 corrected the count to twelve
  modules, named the Phase 4 packages
  explicitly, and placed `aios-cred` in
  the §9 → module mapping.
- §14 renamed the risk IDs from `T#` to
  `PRDR-#` and added a namespace note;
  added SSRF non-approvability, kill
  in-flight, conformance gating,
  provider-notice honesty, verify runtime
  bound.
- §11.3 added rule precedence
  (deny > ask > allow, first match wins)
  and the "sandbox is the security
  boundary" framing.
- §11.9 split approvals into per-action /
  per-scope / persisted (tier 0/1 only).
- §8.2 and §10.3 aligned with the new
  approval classes and the watcher
  deferral.

Changes from 1.0 to 1.1:

- §5, §6, §11.5, §11.7, §12, §13 rewritten to
  replace the "deterministic" claim with the
  bounded "reproducible and auditable" claim
  (variance is measured, not eliminated).
- §11.11 REQ-GLB-DET-001..008 rewritten to
  match: temperature is best-effort, the clock
  is split into decision and deadline
  interfaces, paths are allowed only in scope
  matching, policy may be tightened mid-run,
  replay uses recorded outputs and never
  re-calls models or re-executes tier 2/3
  effects.
- §11.2 split the audit row hash into
  content_hash (pure function of inputs) and
  chain_hash (content + previous), and pinned
  the timestamp to the injected clock.
- §12.5.8 split the audit store into a
  redacted store and an encrypted replay
  store with separate retention; clarified
  the truncation anchor direction (commits to
  the removed prefix); declared the design
  tamper-evident, not tamper-proof.
- §12.5.7 redefined the head-to-head bench
  as "each product's default config" with an
  optional same-model arm only where the
  product permits; added a significance rule
  with the small-sample caveat.
- §12.1 removed the circular "below the
  developer's declared budget / limit" gates
  and replaced them with fixed targets in
  `aios.toml`.
- §13.1 added a §9 → MVP module mapping table,
  resolved the audit log double-assignment,
  and deferred the TUI.
- §14 added T3 (sandbox escape), T5
  (dependency supply chain), T6 (SSRF), and
  T7 (cross-project leakage) risks; thickened
  the control-channel spec; replaced the
  approval-fatigue mitigation of "auto-allow
  rules" with batching and measurement; added
  kill in-flight tool call, provider notice
  honesty, conformance gating, and verify
  runtime bound.
- §8.2, §8.3, §10.2, §10.4, §11.8, §11.9
  aligned with the rewrite.

The change from 0.9 to 1.0 was the addition of
the optimizer, retry, verify, roles, persist,
and image modules, and the closure of the
remaining v0.9 audit gaps.
