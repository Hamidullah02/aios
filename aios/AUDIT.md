# Design and product audit

**Date:** 2026-Q2
**Scope:** DESIGN.md v0.8 (the spec), the aios implementation (493
tests, all green), the B16 review, and the §1 value proposition.
**Method:** read the design top-to-bottom, then walk the code
top-to-bottom, then look for inconsistencies, untested
assumptions, and failure modes the design doesn't address.

This is a pre-Phase-2 sanity check. The product is not yet
shipping; the question is whether the design is *solid enough
to build on* for the next 6-12 months.

---

## TL;DR

The design is **mostly solid but not yet a product**. The core
loop (orchestrator + state machine + audit + policy) is
coherent and the 493 tests cover the contract. There are
**three CRITICAL gaps** that would surface as production
incidents within weeks of a real deployment, and a handful of
HIGH risks that should be fixed before scaling. None of them
invalidate the approach; all of them are fixable with the
current architecture. The §17.4 kill conditions and the H1–H9
fixes hold up.

**Verdict:** proceed to Phase 2 *after* closing the three
CRITICAL gaps. None of them requires new architecture; they
are wiring, configuration, and missing primitives.

---

## CRITICAL — must close before Phase 2 ships

### C1. The "first run" path is undefined

**Symptom:** B16 reports K1 as PASS, but reading the
orchestrator's `run()` method shows that the *first run* path
is not actually wired.

What works: the orchestrator accepts a `TaskPlan`, calls
`submit()` to create a `TaskRun` in `DRAFT`, and `run()` walks
it to `SUCCEEDED`. The 33 orchestrator tests cover this.

What does not work: a *user-facing* "first run" path. There
is no CLI that takes an objective, drafts a plan, runs the
planner (B10), validates the plan, calls the gateway for any
plan-level approvals, submits, and runs. The current code
expects the user to construct a `TaskPlan` by hand (or by the
wedge generator). For a real product, the entry point is
"the user types a sentence and walks away."

**Why this is CRITICAL:** Phase 2's "spec editor + UI" work
assumes a CLI already exists. Without it, every UI change
starts by hand-rolling the same `TaskPlan` construction.
This will get duplicated across the API service, the
premade-agent runners, and the scheduler, and the
duplication will drift.

**Fix:** ship a thin `aios.cli` module with a `submit-objective`
command that:
1. Calls the planner (B10) with the objective,
2. Validates the plan against `TaskPlan`,
3. If the plan has `risk_tier >= 2`, surfaces an approval
   prompt before any execution,
4. Submits and runs.

This is roughly one file. Should land in Phase 2 week 1.

---

### C2. The "no real model" assumption is hiding everywhere

**Symptom:** B8 ships a `FakeModelAdapter`. B10's `LlmPlanner`
calls it. B12's `StubWorker` produces scripted artifacts. B15
records $0.0000 median cost. The 493 tests pass, but the
**cost number in the report is a lie** — the stub model never
incurs a cost, so every "median cost per verified task"
measurement is structurally zero.

What works: the *interface* is right. The router picks a tier,
the worker calls the adapter, the adapter returns a response,
the budget meter charges `cost_usd` from the response. Phase 1
swaps the fake for a real adapter.

What does not work: the **budget meter trusts the adapter**.
A real Anthropic / OpenAI / DeepSeek response includes a
`cost_usd` *as computed by the vendor*, but the response body
is what the model *claims* it cost. A misbehaving model (or
a man-in-the-middle) can under-report cost. The B8 design
says "charge the budget on every call" but does not say
"the cost is verified, not claimed." §6.4 + §8.1 T9
("Unauthorized spend") both rely on the cost number being
truthful.

**Why this is CRITICAL:** the entire "cost gate" story
(§9.3 "median cost per verified task") and the "approval
fatigue" story (E3, "users click through approvals") are
predicated on accurate per-call cost. If a model can
under-report by 10x, the cost gate never trips and the user
runs over budget before they notice.

**Fix:** before Phase 2 ships, the cost story needs one of:
1. Vendor-signed usage reports that the orchestrator
   reconciles against claimed cost (operational, not code),
2. A separate cost-meter that *measures* tokens in/out and
   applies a model-specific price table (not trusting the
   response), or
3. A sandbox that limits *external* network and forces all
   model calls through a credentialed broker that meters
   outside the model.

Option 2 is what should ship. The B8 adapter should expose
`input_tokens` and `output_tokens` (it already does) and
the B12 budget meter should compute cost from a
`price_table` rather than trusting the response. This is a
one-day change.

---

### C3. The "approval service" is not in the request path

**Symptom:** B11 ships an `ApprovalService` with the
content-hash binding, one-use semantics, and expiry. The
14 unit tests pass. The B14 adversarial suite confirms that
the service correctly *validates* a presented approval.

What works in isolation: the service.

What does not work in the *system*: the orchestrator never
*asks for* an approval. The gateway returns `ask` for a
tool call (B5 R15, R19, etc.), the tool bus surfaces
`needs_approval=True` (B9), and then... nothing. There is
no code that:
1. Takes the `ask` result,
2. Looks up the relevant approval request template for that
   tool,
3. Calls the `ApprovalService.grant` flow (which is
   CLI-prompted today, no UI),
4. Retries the tool call with the resulting approval_token.

I checked the orchestrator's `run()` method, the tool bus,
the B-ask handler, and the planner: nowhere does an
`ask` decision get converted into an approval. The system
today, on a fresh deployment, would block on the first
`git push` and never unblock, because no human can grant
the approval.

**Why this is CRITICAL:** every §8.3 tier-3 action (send
email, publish, spend, force-push) hits this gap. The
adversarial suite says "denied" because the gateway says
"ask", not "deny", but the user-facing flow is broken.

**Fix:** the `ToolBus.call` method that gets `needs_approval`
should:
1. Build an `ApprovalRequest` from the call shape,
2. Hand it to an `ApprovalRequester` (CLI in Phase 1, web
   in Phase 2),
3. Receive the approval_id,
4. Retry the call with `approval_token=<id>`.

This is the same code in three places (B5 gateway, B9 bus,
B12 orchestrator); consolidate it. Roughly one file +
integration tests. Must land before any real user touches
the system.

---

## HIGH — fix before scaling beyond a few internal users

### H1. The four concurrency caps (§15.3) are not implemented

The design says "tasks=1, steps=N, model calls=M, sandboxes=K"
with explicit defaults. The orchestrator docstring says
"the §15.3 four concurrency caps (default = 1; B12 is
single-task)". But searching the code, there is no
implementation. The orchestrator runs one task to
completion before any other task starts. There is no
`asyncio.Semaphore` or thread pool anywhere. B12 passes
the test suite because the test uses one task at a time,
but the contract is not enforced.

**Risk:** when Phase 2 puts the orchestrator behind an API
service, the first user who submits a long-running task
will block every other user. The §9.3 "5 concurrent users"
gate cannot be reached without this.

**Fix:** add a concurrency layer in the orchestrator
constructor (or a new `Runner` class) with four
`asyncio.Semaphore`s, one per kind, configured from the
plan's budget or a system default. The DecisionTrace
emitted on cap (per H6) is a one-line addition. Roughly
half a day of work plus the integration tests.

---

### H2. The schema-validator and diff-scope verifier are not implemented

§6.1 and §15.4 promise "tests, linters, schema, diff-scope
check" as the four code-first verifier primitives. B10 ships
`CommandVerifier` and `FileContentVerifier`, but the
`SchemaValidator` (validate a JSON file against a
TaskPlan's `outputs` schema) and the `DiffScopeVerifier`
(ensure the patch only touched files in `scope.paths`) are
not in the package. The wedge happens to use
`CommandVerifier("pytest -q")`, so the 20 tasks pass, but
a real codebase with non-Python outputs is uncovered.

**Risk:** "answer-passing-but-wrong" (§6.8, the new
adversarial class) is exactly what the schema + diff-scope
verifier catch. Without them, a model can pass the tests
and break an undocumented invariant. The §17.4 K2
("Adversarial suite fails... §6.8 loop fails to catch an
answer-passing-but-wrong case") is in scope the moment we
ship to a real user.

**Fix:** two more verifier classes in `aios/planner/verifier.py`,
each roughly 50 lines. The diff-scope one needs a tree-diff
library (or 30 lines of `git diff --stat` parsing).

---

### H3. The audit log redaction is correct but the secret flow leaks the secret

The B2 audit store redacts "high-entropy strings" (32-hex
tokens, API keys, etc.) before writing. The B11 approval
service test confirms this: the audit row contains
`[REDACTED:high_entropy_string]` instead of the approval_id.

What the redaction gets right: the *id* is not in plaintext
in the log. Good.

What the redaction breaks: the *audit reader* cannot
correlate the approval with the audit row that consumed it
(B11's "consume" writes a second audit row with the
approval_id, which is also redacted). A reviewer cannot
answer "did the worker actually have a valid approval for
that action?" by reading the log; they need the in-memory
service state. The in-memory service state is lost on
restart. The §8.6 "tamper evidence" claim is therefore
weaker than the design intends.

**Risk:** moderate. The audit log is still useful; it just
loses the cross-reference for high-entropy ids. For Phase 1
this is acceptable; for Phase 2 (where approvals are
audited across users and projects) it becomes a real
debugging blocker.

**Fix:** two options:
1. Don't redact the approval_id; redact the *content_hash*
   payload of the approval instead. The id is just an
   opaque token, not a secret. (The B11 service already
   uses a 128-bit random id; losing it to a redaction
   pattern is overzealous.)
2. Add an *index table* that maps redacted id -> real id,
   stored encrypted at rest, only readable with a separate
   key.

Option 1 is simpler and arguably correct. The 17 B11 tests
that assert redaction should be re-checked.

---

### H4. The §16.10 WorkflowSpec and §16.11 AutomationSpec are not on the roadmap

WorkflowSpec, ScheduleSpec, AgentSpec, and the result
envelope are *all* in `src/aios/schema/` with 128 passing
tests. This is solid. **H4 downgraded to MEDIUM.**

What is *not* on the roadmap: a `WorkflowSpecRunner` and
`ScheduleSpecRunner`. The schemas exist; the executors do
not. B12 only runs `TaskPlan`. Phase 2 needs:
1. A `WorkflowExecutor` that walks a DAG of `WorkflowSpec`
   nodes, each a `TaskPlan` call, and respects the §16.10
   "compile-time structural check + first-run conformance
   probe" (H1 fix from §19).
2. A `Scheduler` (in-process + DBOS-durable) for
   `ScheduleSpec`.

The §16.11 AutomationSpec is, per §19, "designed but not
built" and is the right place to defer — the workflow +
schedule pair covers 80% of the n8n use cases.

**Risk:** the design promises more than Phase 1 ships. If a
customer asks "can I trigger this on a webhook?" the
answer today is "we have `WorkflowSpec` but no runner."
This is a sales blocker, not a safety blocker.

**Fix:** when Phase 2 starts, the `WorkflowExecutor` and
`Scheduler` are the first two pieces. Roughly 1-2 weeks
each.

---

### H5. The "false-success" warning in B13 is not a real signal

B13's `ReportGenerator` warns when a `succeeded` result has
zero decision traces. The 11 B13 tests pass.

But the report generator runs *after* the orchestrator has
already declared `succeeded=True`. The warning is shown
to the user, but the `succeeded` value is what gets written
to the task store, what the report's headline says
(`succeeded: **True**`), and what the §14.1 false-success
gate is supposed to catch. The warning is the *only*
mitigation, and it is a soft "see Warnings" footer in
Markdown.

A user who skims the report sees `succeeded: True` and
ships the change. The warning is missed.

**Risk:** the §14.1 invariant ("reports contain no
unverified success claims") is *technically* satisfied
(the report *contains* the warning), but the invariant's
*intent* (don't let users act on unverified claims) is
not.

**Fix:** the report's headline should say `succeeded: ?
(unverified)` when the warning fires, and the
`OrchestratorResult.succeeded` field should be `False`
when the warning fires. This is a one-line change in the
report generator and a one-line change in the orchestrator
to plumb the trace count into the result.

---

### H6. The §15.5 head-to-head benchmark is not built

§17.4 K3 says the §15.5 head-to-head benchmark is one of
the kill conditions. B15 ships a *two-tier internal
benchmark* (cheap vs strong, both stubs) that proves the
runner works. The actual head-to-head against Claude
Code, Grok Build, Antigravity, OpenAI Operator (§15.5)
is not built.

**Risk:** we cannot tell if we are competitive. The
internal two-tier comparison will not move the §17.4
needle; it only proves the harness.

**Fix:** the head-to-head harness is a runner that, for
each wedge task, runs the task under each competitor with
the same hidden tests, records cost + wall + pass/fail,
and produces a side-by-side Markdown table. The
competitor adapters (Claude Code CLI, Grok Build, etc.)
are each a thin wrapper around their CLI's JSON output.
This is roughly a week of work *after* a real model
adapter exists (C2).

---

## MEDIUM — known, on the v0.9 list

### M1. The "memory store" is mentioned in §4.4 but not shipped

The design names a "Memory store" with "typed stores with
provenance and staleness." There is no `src/aios/memory/`
package. The §16.5 durable preferences and the §16.6
compaction are also not in the file tree. The
`WorkspaceHash` (used in the no-progress detector) is
the only "memory" that exists, and it is in-process.

**Risk:** the design talks about a memory layer; the
implementation does not have one. A customer asking "can
the agent remember my coding style across tasks?" gets
"not yet."

**Fix:** Phase 2 deliverable, not a Phase 1 blocker. The
§16.5 durable preferences are a small layer (a typed
schema + a per-user SQLite table); the §16.6 compaction
is an LLM summarizer over a per-task memory and can ship
later.

### M2. The §16.3 capability schema is in `model_capability.py` but the conformance probe is not

`src/aios/schema/model_capability.py` ships. The
"conformance probe" that exercises a model on a known
set of tasks to verify it meets a `min_model_profile` is
not built. Without it, the B8 `ModelRouter.route()`
method is correct in interface but has no actual
capability data to consult; the `requires_strong` flag
is the only signal.

**Risk:** we cannot enforce "use strong for hard tasks"
because we do not know which model is "strong."

**Fix:** Phase 2 deliverable, ~one week of work. The
`ModelCapabilityRegistry` is a typed store of per-model
eval scores; the `ConformanceProbe` runs a small
eval set and updates the registry.

### M3. The doc-quality items in §19 (M3-M12, L1-L9, X1-X3)

These are tracked in the design as deferred to v0.9.
None affect the build. They will become visible if a
new contributor tries to read the doc and gets confused
by a stale tag or a wrong cross-reference, but they
are not blockers. I have not re-checked each one; the
v0.9 work item is the right place.

### M4. The model-tiers naming is fragile

B8's `ModelTier = str  # 'cheap' | 'mid' | 'strong' | 'none'`
is a string literal with a comment. There is no enum,
no validation that a `ModelCall(tier="foo")` raises
early. The router and adapter assume the strings are
well-formed; an unknown tier is silently treated as
"cheap" (or as the fallback chain's `none`).

**Risk:** a typo in a model name produces a wrong call
that still gets through. The audit log records the tier,
so the mistake is traceable, but it is not blocked.

**Fix:** make `ModelTier` a `Literal["cheap","mid","strong","none"]`
or an `Enum`. Pydantic's `Field` will validate. One
file change.

### M5. The orchestrator's private-attribute access is fragile

The B12 orchestrator reaches into
`self._store._writer._store` to get the audit store. This
is two levels of private-attribute access on someone
else's API. The 33 orchestrator tests pass, but any
refactor of `TaskRunStore` or `DecisionTraceWriter` that
renames or removes these attributes breaks the
orchestrator silently.

**Risk:** a routine refactor introduces a bug that no
test catches (because the test only exercises the happy
path through `run()`).

**Fix:** add a public `audit` property to `TaskRunStore`
and a public `audit` property to `DecisionTraceWriter`.
The orchestrator accesses `self._store.audit`. Roughly
30 minutes.

---

## LOW — observation, no fix needed yet

### L1. The §6.4 "Same error class twice in a row: replan once, then escalate" is not implemented

The orchestrator's `_run_one_step` increments `retries`
and stops at the cap (2). It does not classify the
"error class" (e.g. "build error" vs "test failure" vs
"timeout") and does not have a separate "replan" trigger
on the second consecutive same-class error. The §6.8
attempt loop *is* the replan; the design says "then
escalate" but the orchestrator just lets the attempt
loop try again.

**Observation:** this is correct behavior for the wedge
(the 20 tasks do not exercise this path), but a real
user's task would. The "escalate" behavior is "go
blocked" today.

**Fix:** when the *same* failing-check set appears in two
consecutive attempts, transition to BLOCKED with reason
"replan failed twice (§6.4)." Today the no-progress
detector catches this after 3 steps; tightening to
2 attempts × same-error is the §6.4 spec.

### L2. The B6 subprocess sandbox does not enforce network isolation

The B6 docstring says: "the subprocess sandbox is not a
security boundary. The Phase 1 gVisor implementation is
the security boundary." This is correct and called out.
The B14 escape tests pass against the gateway, not the
sandbox, because the gateway is the network-isolation
layer in the architecture.

**Observation:** the design and the B6 implementation
agree. The risk is that a future contributor reads
"B6 = sandbox" and assumes the subprocess is safe. The
docstring is the safety net.

**Fix:** none; consider adding a one-liner to the
project README: "Phase 1 sandbox is a process
sandbox, not a security boundary. See §8.4."

### L3. The "premade agent catalogue" (§16.4) is a 6-month project, not a B-item

The design names 10 premade agents (Software Implementer,
Code Reviewer, Bug Reproducer, Dependency Auditor, Sourced
Researcher, Report Writer, Data Analyst, Prospect
Researcher, Outreach Drafter, Recurring Ops Runner).
Each is a specialized `AgentSpec` + a benchmark. The
"admission rule" (must beat generic worker+skills on
its own eval set) is the §16.4 quality bar.

**Observation:** this is the Phase 2 *and* Phase 3
roadmap. The current `TemplatePlanner` and
`CodeFirstVerifier` (B10) are the first two primitives.
The 10 premade agents ship over 12+ months.

**Fix:** not a Phase 1 concern. Track in v0.9 doc.

---

## Product / market fit

The design's value proposition (§1):

> "trustworthy AI for non-incremental engineering, research
> and business operations; a workforce that ships real work
> in a sandbox, with the audit, controls and reliability a
> regulated business needs."

**What the design does well:**

* The §8.1 threat model is *exhaustively* enumerated (T1–T14)
  and each threat has a named control. The B14 adversarial
  suite verifies the controls hold. This is a real moat.
* The "succeeded requires verifier evidence" rule (the
  §6.3 invariant, the §14.1 false-success check, the
  B13 report warning) is a *genuine* differentiator
  versus the current agent market, where the model
  self-reports success and the user trusts it.
* The cost gate (§9.3) and the median-cost-per-verified-task
  measurement (B15) are quantitative. Most competitors
  publish no numbers.
* The §17.4 kill conditions are honest. "If we lose on
  cost, we admit it and stop calling it the answer." This
  is rare.
* The "no real model" honest state (B16 K1 "with stub;
  real-model deferred to Phase 1") is a virtue, not a
  bug. The contract is right; the data is the next step.

**What the design undersells:**

* The **time-to-first-task** is not measured. A
  non-technical user with the system today would need to:
  install Python, set up SQLite, write a TaskPlan, run
  the orchestrator. There is no onboarding, no sample
  repo, no "click here to run a bug-fix demo" path. The
  Phase 2 spec editor + UI is the answer, but the design
  does not say "users can do X in 10 minutes."
* The **operator experience** is missing. The
  ApprovalService has no UI. The audit log is JSON, not
  searchable. The wedge is a folder of 20 JSON files.
  A "developer relations" surface is not in the design.
* The **failures are silent**. A task that ends in
  `FAILED` returns a `note` string ("missing evidence for
  X"). A user who runs the CLI and sees
  `final_state: FAILED` has to open the audit log to
  understand why. A "report the failure in plain
  English" output is not in the design.

**What the design does not yet prove:**

* The "looped verification" claim (§6.8 + §15.4) — that
  a cheap tier + code-first verifier + occasional
  second-model review is *within 15%* of a strong tier
  on the wedge. §17.4 E1 is the kill condition; it has
  not been measured. The B15 two-tier run uses the same
  stub, so it cannot answer this. This is the
  single most important measurement before Phase 2.
* The "operator cost is bounded" claim. The §9.3 numbers
  are written but not measured against a real workload.
  The B15 medians are the right shape; the values are
  zero.
* The "approval fatigue is manageable" claim (E3). No
  measurement. The approval service is correct, but
  the *user experience* of being asked to approve
  every tier-3 action has not been validated.

**What I would change in the design before Phase 2:**

1. **Add §16.12 "Operator experience":** one paragraph
   per primitive (CLI, approval, audit, report) on what
   the operator sees. The design currently writes
   extensively about the *system* behavior and not the
   *user* behavior.
2. **Add §17.6 "Phase 2 entry criteria":** a numbered
   list of the measurements and code that must exist
   before Phase 2 starts. C1, C2, C3 above would be
   items 1-3. H1, H5, H6 would be items 4-6.
3. **Cut §16.11 from v1.** The AutomationSpec is a
   separate product surface. Drop it to a "future
   consideration" footnote; ship WorkflowSpec + ScheduleSpec
   as the n8n replacement, full stop.
4. **Add a "first run" sample.** A `examples/fix-a-bug/`
   directory with: a tiny broken Python project, a
   TaskPlan JSON, a `run.sh`, an expected report. This
   takes an hour and turns the design from "spec" to
   "product someone can try in 5 minutes."

**What the design gets right that I would not change:**

* The "agent is data, not code" rule. The AgentSpec
  schema is the right abstraction. The §16.4 admission
  rule is the right quality bar.
* The "deny by default" policy. The gateway's 30+
  rules, the egress proxy's allowlist, the approval
  service's content-hash binding — these compose into
  a real safety story. Most competitors do not have
  this.
* The "audit is the source of truth" rule. The
  DecisionTrace + redaction + hash-chain design is
  defensible against the §8.1 T13 "failure of approval
  or policy layer" threat. The state machine is the
  only writer; the audit log is the only record; the
  resume path rebuilds from both.
* The §6.4 / §6.8 two-level retry. This is the
  correct shape: a small per-step budget + a larger
  per-task budget. The wedge's `max_attempts=5` ×
  `retry_cap=2` = 10 max step retries is a measured
  upper bound, not an arbitrary number.
* The §15.5 acceptance ("we beat Claude Code Pro on
  cost-per-verified-task"). This is the right metric
  and the right competitor. A different competitive
  wedge (e.g. "we have a better UI" or "we integrate
  with Slack") is much weaker.

---

## Recommendation

**Phase 2 should not start until C1, C2, C3 are closed.**
H1, H5, H6 are second-priority and can land in the
first two weeks of Phase 2 alongside the operator
work (C1) and the workflow runner (H4).

**The design is otherwise solid.** The 493 tests
cover the contract; the threat model is the right
shape; the kill conditions are honest. The remaining
work is wiring, not architecture.

**Two product gaps that are not bugs but should be
addressed before the first paying customer:**

1. A "first run" sample (5-minute onboarding).
2. A failure-mode UX (the report is too terse for a
   non-operator user).

**One market risk that the design does not yet
answer:**

The "looped verification is within 15% of a strong
tier" claim (§17.4 E1) is unmeasured. This is the
most important experiment in the next 30 days. The
B15 harness is ready; the missing piece is one real
model adapter and one real benchmark run.

If E1 fails, the cost story collapses and the
design's competitive wedge does not exist. That is
the most likely Phase 2 no-go, and it is a *good*
no-go: it is measured, not assumed.

---

## Closing

The product is at the right place for the right
time. The "trustworthy AI for non-incremental work"
position is defensible; the implementation is
honest about its gaps; the test suite is the
contract. The audit found three wiring gaps that
would surface as production incidents, three
scalability gaps that would surface as user
frustration, and one unmeasured competitive claim
that could surface as a no-go. None of these
invalidate the design.

Proceed, after closing C1, C2, C3.
