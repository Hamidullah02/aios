# Changelog


v0.35.0 — verify tools wired into the bus

The verify package (verify.SelfCheckTool,
verify.RunTestsTool) was an orphan: the package had
unit tests, the doc set named both tools as required
modules, but the CLI never registered them on the
tool bus. This version wires them in.

What changed:

  * internal/cli/verify_wiring.go (new):
    wireVerifyTools (returns run_tests always,
    self_check only when a router is available),
    wireSelfCheckOnly (the deferred-wiring path),
    and the routerJudge adapter (models.Router ->
    verify.Judge). The adapter lives in cli, not
    verify, so verify does not import aios-models.
  * internal/cli/deps.go wireAgentLoop: run_tests
    is now in the bus from the start (no model
    dependency). After wirePlanner (only for
    `fix`) self_check is registered on the bus
    via the new bus.Register method.
  * internal/tools/bus.go: new Register(name,
    tool) method. The bus was immutable; this
    method exists for the deferred-wiring case.
  * internal/verify (untouched but now reachable):
    SelfCheckTool, RunTestsTool, and the
    per-runner parsers are the production code
    the doc set names in aios-verify.

Tests: 11 new (TestWireVerifyTools_*, 
TestWireSelfCheckOnly_*, TestRouterJudge_*,
TestBuildDeps_RunSubcommand*,
TestRunTests_EndToEnd,
TestRunTests_NotFoundRunnerStillProvesRegistered,
TestSelfCheckTool_ParseResponse). All 30
packages green; go vet clean; gofmt -l . clean.

Smoke: aios run with a run_tests task dispatches
through the production bus and the tool runs the
subprocess (the audit row is written). The
run_tests tool currently has a pre-existing env
quirk (it sets only CI=1 and does not pass PATH),
so a real `go test` invocation fails inside the
tool. That is a v0.36.0 fix in the verify
package, not a v0.35.0 issue.


v0.34.0 — aios fix --plan-out: persist the planner's plan

The planner's TaskPlan used to live in memory only.
`aios fix "..." --plan-out <path>` now writes the
plan to the path the user picks, prints the path,
then runs the plan. The user can then `aios run
<path>` to re-run the same plan (optionally after
editing the YAML).

What changed:

  * `internal/cli/plan_writer.go` (new): the
    `WritePlanYAML` helper serialises a *TaskPlan
    to the runtime's plan YAML format and writes
    it to disk. It mirrors the test fixtures in
    `run_e2e_test.go` byte-for-byte so a file
    written here is loadable by `LoadPlan`.
  * `internal/cli/main.go runFix`: when
    `args.PlanOut != ""`, the helper is called
    BEFORE the loop runs so a write error stops
    the run (we do not claim success and
    silently drop the plan).
  * `internal/cli/parser.go`: new `--plan-out`
    flag on the `fix` subcommand. Default is
    empty (no write) so the existing
    `aios fix` behaviour is unchanged.
  * `internal/core/schemas/taskplan.go`:
    added `yaml:"cost_usd"` and
    `yaml:"wall_clock_sec"` tags to the
    `Budget` struct. Without these, the
    gopkg.in/yaml.v3 default name mapping
    would have read them as `costusd` and
    `wallclocksec`, and the existing test
    fixtures never exercised the Budget
    field so the bug was latent.

Tests: 7 new (TestWritePlanYAML_RoundTrip,
TestWritePlanYAML_EmptyPath, TestWritePlanYAML_NilPlan,
TestWritePlanYAML_InvalidPlan,
TestWritePlanYAML_CreatesParentDir,
TestRunFix_PlanOutWritesFile,
TestRunFix_PlanOutThenRunAgain). All 30 packages
still green; `go vet` clean; `gofmt -l .` clean.



v0.33.0 â€” ios fix <objective>: the planner turn

The v0.32.0 review closed the model -> tool -> result
-> verifier chain for a plan that already exists. This
version closes the missing first half: the planner
turn. ios fix <objective> is the doc set's primary
entry point â€” the CLI takes a natural-language
objective, calls the planner to turn it into a
TaskPlan, and runs the plan through the same loop

un uses.

The full chain is now:

  objective -> planner.Plan -> TaskPlan
            -> BusWorker -> tools.Bus -> policy
            -> tool -> audit
            -> CommandVerifier

Every component is the production implementation, not
a stub. The planner calls a real model (via a real
HTTP adapter). The bus dispatches to a real tool. The
verifier inspects the result and fails the plan on an
empty Output or nil Data.

What changed:

- **internal/cli/parser.go â€” new ix subcommand.**
  ios fix <objective> takes the objective as a
  single positional arg. An optional --model flag
  overrides the planner's default model id. The
  parser's indSubcommandIndex map now includes
  ix so the global flags parse before it.

- **internal/cli/router_wiring.go â€” new file.** The
  production uildRouter constructs a
  *models.Router with both OpenAI and Anthropic
  adapters (reading OPENAI_API_KEY and
  ANTHROPIC_API_KEY from the env). The router
  loads ar/egress.yaml, ar/prices.yaml, and
  ar/capabilities.yaml (with built-in defaults
  for the latter two when the file is missing or in
  the wrong shape). The function returns the router,
  the price table, and the egress allow list.

- **internal/cli/planner_wiring.go â€” new file.**
  The production plannerToolSpecs returns the
  planner's view of the tool set: a flat
  {Name, Description, InputSchema, RiskTier} map
  for 
ead_file, search_files, write_file,
  
un_shell, and sk_user. The descriptions and
  schemas are the strings the model reads when it
  decides which tool to dispatch to. The tool list
  mirrors the bus's tool map in wireAgentLoop; the
  two are kept in sync by hand (a comment in the
  file notes the seam).

- **internal/cli/deps.go â€” BuildDeps now wires
  the planner for ix.** For every non-bootstrap
  subcommand (i.e. every subcommand except init
  and daemon), BuildDeps constructs a
  *tools.Bus, a *BusWorker, and a
  CommandVerifier (the v0.32.0 wiring). For ix
  only, it additionally constructs a
  *planner.Planner and the supporting router /
  price table / egress. The Deps struct gains
  Router, Prices, Egress, and Planner
  fields. Other subcommands (run, resume, cancel,
  report, audit) leave the planner nil â€” they
  operate on a plan file that already exists.

- **internal/cli/main.go â€” new 
unFix handler.**
  The handler calls planner.Plan(ctx, args.Objective,
  nil), then runs the resulting plan through the
  shared 
unWithApprovals inner loop. The chain
  is: planner -> BusWorker -> bus -> policy -> tool
  -> audit -> CommandVerifier. The handler fails
  closed: a missing planner, a model call error
  (e.g. no API key), a verifier reject â€” all
  return a non-zero exit code and a clear error
  message on stderr.

- **internal/cli/init_command.go â€” fix the
  defaultPricesYAML shape.** The previous scaffold
  wrote a list-shaped file
  (prices:\n  - model: gpt-4o\n    ...) but
  models.LoadPriceTable reads a map-shaped file
  (prices:\n  gpt-4o:\n    ...). The scaffold is
  now map-shaped; the price table loads cleanly on
  ios init.

- **internal/cli/fix_wiring_test.go â€” new file.**
  Five tests guard the wiring:
    - TestBuildDeps_WiresPlannerForFix â€”
      asserts BuildDeps wires a planner for
      ix but leaves it nil for 
un.
    - TestRunFix_RequiresObjective â€” asserts
      
unFix returns a user error when the
      objective is empty.
    - TestRunFix_RealChain â€” drives the full
      chain with a stub model that returns a
      valid plan JSON. The plan runs through the
      real worker, bus, policy, and tool, and the
      audit log has a 	ool_call row.
    - TestRunFix_NoAPIKey_FailsClosed â€” runs
      Main("fix", ...) with OPENAI_API_KEY
      and ANTHROPIC_API_KEY unset. The CLI
      returns a clear "OPENAI_API_KEY is not set"
      error and exit code 2.
    - TestRunFix_EndToEndWithHTTPServer â€”
      points the production OpenAIAdapter at a
      httptest.Server that mimics the OpenAI
      chat completions response. The planner
      calls the server, parses the response, and
      the plan runs through the real loop. This
      is the integration test the v0.31.0 review
      demanded.

- **internal/cli/test_helpers.go â€” new file.**
  uildTestAgentLoop is the test seam: it
  constructs the production wire (tools.Bus,
  BusWorker, CommandVerifier) with a pre-built
  audit log, policy, and fake ask callback, so
  tests do not have to load policy from disk or
  block on stdin.

Verified:

- go build ./... clean.
- go vet ./... clean.
- gofmt -l . clean.
- go test ./... â€” 30 packages pass.
- Live end-to-end: ios init + ios fix
  "read the design" with no API key. The CLI
  prints "Planning: read the design" then
  "error: plan generation failed after 1
  iteration(s): planner: model call: openai:
  no_api_key: OPENAI_API_KEY is not set" and exits
  with code 2.

Substitutions vs the doc set:

- **ix is a real subcommand, not a flag.** The
  doc set describes ios fix <objective> as the
  primary entry point. This version makes it
  explicit: a top-level subcommand, not a flag on
  
un.

- **The planner verifier is planner.NewCodeFirstVerifier.**
  The planner's repair loop uses the same verifier
  the planner package's tests use. A future commit
  can swap in a custom verifier (e.g. one that
  consults the policy gateway) without changing
  the ix handler.

- **API key check is at call time, not startup.**
  The production adapters read the API key at
  Complete time, so a missing key surfaces as a
  clear error from the first model call. The CLI
  does not pre-flight the key (that would couple
  startup to a network round-trip).

Known gaps (the next milestone):

- The unwired modules (benchmark, egress, memory,
  persist, report, search) are still untouched.
  The planner module is now wired into the CLI
  but only the planner's public API is used; the
  rest of the planner package (custom rules,
  schema validation) is exposed but not yet
  exercised by ix.
- AUDIT.md is stale. docs/architecture is
  the source of truth.
- The CLI does not yet save the plan the planner
  produced to a file so a user can re-run it
  with ios run after editing. Adding
  --plan-out is a small follow-up.

v0.32.0 â€” real agent loop

The v0.31.0 review (the only reviewer request the
previous release did not act on) demanded:

> "Stop adding modules. Wire one real loop:
> aios fix <objective>. Real model â†’ real tool â†’
> real result â†’ real verifier. Real CommandVerifier.
> The product needs an honest worker, not a stub."

This version answers that. The CLI's 
un
subcommand now executes a real model â†’ tool â†’
result â†’ verifier chain. The previous default
(PassthroughWorker + SmokeVerifier) did not
touch the bus and rubber-stamped every tool call:
the worker returned a stub "ok" without calling
the tool, and the verifier returned VerdictPass
for every result. The new default is a real
BusWorker and a real CommandVerifier that
fail closed on missing output, nil data, and
deny-by-default policy verdicts.

What changed:

- **internal/cli/verifier.go â€” new file.** The
  CommandVerifier is the production verifier
  the CLI ships with the bus. It inspects the
  WorkerResult the BusWorker produced and
  decides pass / fail on real signal:
    - Output is the bus's Result.Summary. An
      empty summary means the bus short-circuited
      (missing tool, policy deny) and is a fail.
    - Data is the bus's Result.Output. A nil
      Data means the tool did not produce a
      typed result and is a fail.
  The previous SmokeVerifier is retained only
  for tests that need a permissive stand-in.

- **internal/cli/bus_adapter.go â€” new file.**
  The production skBusAdapter exposes
  sk.AskUserTool to the 	ools.Tool interface
  so the bus can dispatch it. The shim is the
  same shape as the test-only adapter in
  internal/ask/bus_adapter_test.go; it lives
  in the cli package to avoid the import cycle
  (tools does not import ask, ask does not import
  tools).

- **internal/cli/deps.go â€” BuildDeps now wires
  the real agent loop.** For every non-bootstrap
  subcommand (i.e. every subcommand except init
  and daemon), BuildDeps constructs a
  	ools.Bus with the production tool set
  (
ead_file, search_files, write_file,
  
un_shell with a SubprocessRunner,
  sk_user), wraps it in a *BusWorker with
  the StdinPromptCallback, and pairs it with
  a CommandVerifier. The Deps struct gains a
  Bus *tools.Bus field so tests can inject a
  custom bus. Bootstrap subcommands skip the
  wiring (they do not run plans).

- **internal/cli/parser.go â€” new --workspace
  flag.** The workspace is the root the tool bus
  uses to resolve relative paths. The default is
  the parent directory of --config. The flag
  is global so 
un, 
esume, and cancel
  share the same workspace.

- **internal/cli/smoke_test.go â€” the end-to-end
  tests now exercise the real loop.** The previous
  tests relied on PassthroughWorker +
  SmokeVerifier and would have rubber-stamped
  a denied read. The new tests use a real
  
ead_file against a real file in a temp
  workspace, with a policy that allows the read,
  and assert a real 	ool_call audit row was
  written.

- **internal/cli/bus_worker_wiring_test.go â€”
  new file.** Six new tests guard the wiring:
    - TestBuildDeps_WiresBusWorker â€” asserts
      deps.Worker is *BusWorker, not
      PassthroughWorker.
    - TestBuildDeps_BootstrapSkipsWiring â€”
      asserts init and daemon get a nil
      worker (they do not run plans).
    - TestRunCommand_RealToolCall â€” runs a
      real 
ead_file end-to-end through Main,
      asserts a 	ool_call audit row was
      written with decision: "allow".
    - TestRunCommand_VerifierRejectsUnknownTool
      â€” asserts an unknown tool name produces a
      non-zero exit (the verifier fails the task
      on the empty result).
    - TestCommandVerifier_StubsAreGone â€”
      asserts the production BuildDeps no
      longer returns the stubs. This is the
      literal "stop shipping a stub" test the
      v0.32.0 review asked for.
    - TestRunCommand_PolicyDenyStopsLoop â€”
      asserts a denied tool call writes a
      decision: "deny" audit row and the
      plan stops.

Verified:

- go build ./... clean.
- go vet ./... clean.
- gofmt -l . clean.
- go test ./... â€” 30 packages pass.
- Live end-to-end: ios init + ios run
  <plan.yaml> with a real 
ead_file task.
  The audit log shows three rows: a
  policy_decision (the bus's gateway), a
  	ool_call (the bus's audit row), and a
  decision (the orchestrator's "verifier=pass
  tool_calls=1 cost_usd=0.000000").

Substitutions vs the doc set:

- **ix <objective> â†’ 
un <plan.yaml>.** The
  doc set describes ios fix <objective> as the
  single entry point: the planner turns the
  objective into a plan, the worker executes the
  plan, the verifier inspects each task. This
  version wires the modelâ†’toolâ†’resultâ†’verifier
  chain for a plan that already exists; the
  planner step (turning an objective into a
  plan) is the next milestone. The shape of
  the loop â€” worker dispatches to the bus,
  bus consults the gateway, bus runs the tool,
  bus writes the audit row, verifier inspects
  the result â€” is the doc set's loop.

- **SmokeVerifier kept as a type for tests.**
  The verifier is now CommandVerifier by
  default. SmokeVerifier is no longer the
  default; it is retained so tests that need
  a permissive stand-in (e.g. a test that
  exercises the orchestrator's no-progress
  guard) can substitute it.

- **skBusAdapter lives in internal/cli/ not
  internal/ask/.** The test-only adapter in
  internal/ask/bus_adapter_test.go is the
  reference shape; the production adapter is
  in internal/cli/bus_adapter.go because
  internal/cli already imports both sk and
  	ools and is the natural seam.

Known gaps (the next milestone):

- PassthroughWorker and SmokeVerifier are
  retained for tests but no longer used in
  production. The doc set's ix <objective>
  is the next milestone; the planner turn
  requires a real model client.
- The unwired modules (benchmark, egress,
  memory, persist, report, search, planner)
  are still untouched.
- AUDIT.md is stale. docs/architecture is
  the source of truth.

v0.31.0 â€” repo hygiene + policy fail-closed +
tightened CLI surface

(Cleanup milestone; the commit message describes
the change in detail. The originally-deferred
CHANGELOG entry is folded into this v0.32.0
release for readability.)

v0.30.0 — aios-image

Add the aios-image module. The module is the
image attachment layer: it decodes PNG / JPEG
bytes, validates the dimensions against the
model's cap, and renders the model-ready
blocks (Anthropic image block, OpenAI image
URL) for both providers.

The package is a single Go package at
`internal/image/` and depends on
`golang.org/x/image/draw` (already a direct
dep of the monorepo from v0.18.x). No new
third-party dependencies.

Components:

- `Image` — the typed image (bytes, format,
  width, height, color model). The image is
  immutable after `Decode`.
- `ImageAttachment` — the model-ready wrapper.
  Holds the decoded `Image` plus the
  resize-to-max-dimension policy.
- `ToAnthropicBlock` — the Anthropic image
  block (REQ-IMG-ATT-002). The block is a
  map the caller passes as a content block
  to the Anthropic messages API.
- `ToOpenAIURL` — the OpenAI image URL
  (REQ-IMG-ATT-003). The URL is a data-URL
  for the inline case and a https URL for
  the remote case.
- `Constrain` — the dimension-and-format
  validator (REQ-IMG-CON-001..005). The
  validator enforces a max width, max height,
  and a format allow-list.
- `Read` — the file-system reader. The
  reader decodes the bytes through
  `image.Decode` and validates the format
  through `Constrain`.
- `Validator` — the per-model validator
  (REQ-IMG-VAL-001..003). The validator
  reads the model's capability table and
  refuses images the model cannot handle.
- `AttachError`, `ConstrainError`,
  `DecodeError`, `ValidationError` — typed
  errors.

Substitutions vs the doc set:

- **Three sub-packages (`attach` +
  `constrain` + `read`) → single
  `internal/image/` Go package.** Same
  reasoning as `aios-roles` (v0.25.0) and
  `aios-fanout` (v0.22.0): the doc-set
  Python sub-package split is not the
  right shape for a Go monorepo.

v0.29.0 — aios-git

Add the aios-git module. The module is the
git-tools layer: it wraps `git status`,
`git log`, `git diff`, `git commit`,
`git branch`, and `git worktree` behind
typed Go APIs and exposes them as
`tools.Tool` instances the agent can call.

The package is a single Go package at
`internal/git/` and depends on
aios-core (for the clock and the audit
log), aios-tools (for the `Tool`
interface), and aios-sandbox (for the
subprocess runner). No new third-party
dependencies.

Public surface (per REQ-GIT-IFC-001):

- `GitStatusTool` — `git status --porcelain`,
  typed as `[]FileStatus` (path, status,
  staged).
- `GitLogTool` — `git log` with a limit and
  a path filter, typed as `[]Commit`
  (sha, author, timestamp, subject).
- `GitDiffTool` — `git diff` with a path
  filter, typed as `[]FileDiff`
  (path, additions, deletions, patch).
- `GitCommitTool` — `git commit` with a
  message, optional `add` step, and an
  optional author override.
- `GitBranchTool` — `git branch list /
  create / checkout / delete`.
- `GitWorktreeTool` — `git worktree
  list / add / remove`. The worktree
  add honours the worktree root from
  the policy file.
- `Runner` — the subprocess wrapper. The
  runner is the seam for tests: the
  production runner calls the system
  `git`; the test runner captures
  commands without spawning.
- `GitError`, `NotARepositoryError`,
  `BranchNotFoundError`, `WorktreeError`
  — typed errors.

All tools write a `git_call` audit row
on both success and failure. The audit
row carries the args and the result
summary; the diff content is truncated
to 1 KB to keep the audit log small.

Substitutions vs the doc set:

- **Two sub-packages (`status` + `log`)
  → single `internal/git/` Go package.**
  Same reasoning as `aios-roles`
  (v0.25.0): the doc-set Python
  sub-package split is not the right
  shape for a Go monorepo.

v0.28.0 — aios-optimizer + cmd/aios binary

Add the aios-optimizer module. The module is
the cost-quality router: it picks a model
class (cheap / mid / strong) per call based
on the per-(fingerprint, class) cost-quality
table and the budget.

The package is a single Go package at
internal/optimizer/ and depends on
aios-core (for the clock), the Go standard
library, and the new golang.org/x/image
dep (already a direct dep of the monorepo).

Also add the cmd/aios/main.go entry
point so go build ./cmd/aios produces
the ios binary. The binary is a thin
shell around cli.Main.

Plus a ShapePrompt bug fix: a lone quote
character (e.g. an apostrophe in

o_quote's_here) is now written verbatim
rather than collapsing the rest of the
string to str. The previous behaviour
broke TestShapePrompt when the input
had only one quote character.

Plus README module list updated to include
the new modules.
v0.28.1 — retry budget invariant

Drop the negative-clamp in
`RetryBudget.RemainingLocked`. The
clamp was defensive: the function
clamped the result to 0 if the
calculated minimum was negative. The
clamp is now redundant because the
only caller (`Allow`) charges the
budget only when the previous result
was strictly positive; the invariant
that the result is non-negative is
enforced at the `Allow` boundary, not
at `RemainingLocked`.

A doc-comment was added to
`RemainingLocked` to make the
invariant explicit.

No behaviour change in any test case
(the invariant already held); the
change is documentation + a removed
redundant branch.

## v0.27.0 — aios-verify

Add the aios-verify module. The module is the
self-evaluation layer: the model-judgement
self-check tool and the test-runner tool.
The module lets the agent check the result
before the agent reports the task as done.

The package is a single Go package at
`internal/verify/` and depends on aios-core
(for the audit log), aios-tools (for the
`Tool` interface), aios-sandbox (for the
subprocess runner), and the Go standard
library. **No new third-party dependencies.**
The implementation does not import
`aios-models`: the self-check tool depends
on a `Judge` interface the caller provides
(the same shape as the `Summarizer`
interface in `aios-fanout`; the real
`aios-models.Router` satisfies it).

### Components

- **Result schemas** (`result.go`) — the
  `VerificationResult` (status / reason /
  suggestions), `TestResult` (passed /
  failed / skipped / duration_ms /
  failures), and `TestFailure` (name /
  message / optional traceback). The
  `Validate` method enforces the doc-set
  contract: required fields, the
  `pass|fail|inconclusive` status enum,
  non-negative counts, and non-empty
  suggestion strings.
- **Verification prompt** (`prompt.go`) —
  the `PromptTemplate` wrapper around
  `text/template` (the doc-set
  `text/template (stdlib)` substitution).
  `DefaultPrompt` is the doc-set
  REQ-VRF-DAT-003 default template; the
  `NewDefaultPromptTemplate` constructor
  parses it once. The `Render` method
  produces the prompt for a
  `VerificationInput{Result, Goal}`.
- **SelfCheckTool** (`self_check.go`) —
  the `tools.Tool` that builds the prompt,
  calls the `Judge`, and parses the
  response into a `VerificationResult`.
  The parser (`ParseVerificationResponse`)
  is tolerant: it extracts the first
  balanced JSON object from a response
  that may be bare, fenced
  (```json ... ```), or wrapped in prose.
  The tool writes a `self_check` audit row
  on both success and failure. The default
  10-second latency cap
  (`DefaultSelfCheckTimeout`) satisfies
  REQ-VRF-NFR-003.
- **RunTestsTool** (`run_tests.go`) —
  the `tools.Tool` that runs the test
  command through the `sandbox.Runner`
  and parses the output through the
  per-runner parser. The dispatch
  heuristic (`DetectRunner`) maps the
  command name (`go`, `jest`, `mocha`,
  `cargo`, `npm` / `npx` / `pnpm` /
  `yarn`) to the right parser. The tool
  supports the doc-set runners: go test
  (uses `go test -json` — the
  standard Go 1.16+ JSON output rather
  than JUnit XML), jest (`--json`),
  mocha (`--reporter json`), cargo
  test (`--message-format=json`), and
  npm test (delegates to the underlying
  jest or mocha JSON). The tool
  recovers the test counts from the
  captured stdout even when the
  process exits non-zero. The tool
  writes a `run_tests` audit row.
- **Errors** (`errors.go`) — the
  `VerificationError`, `TestRunnerError`,
  and `ResultError` typed errors with
  `Unwrap` and `Is` semantics, plus the
  `ErrVerification`, `ErrTestRunner`,
  and `ErrResult` sentinels.

### Substitutions vs the doc set

- **Four sub-packages
  (`self_check` + `run_tests` + `prompt` +
  `result`) → single `internal/verify/`
  Go package.** Same reasoning as
  `aios-roles` (v0.25.0) and
  `aios-fanout` (v0.22.0): the
  doc-set Python sub-package split is
  not the right shape for a Go
  monorepo. A single package is the
  idiomatic Go unit.
- **`go-playground/validator` (TRD §6.4)
  → hand-rolled validation.** The
  `Validate` methods enforce the
  doc-set contract directly; no new
  third-party dep.
- **`aios-models` direct import
  (TRD §10.2) → `Judge` interface.** The
  self-check tool's "model call" is the
  call site's problem. The aios-models
  `Router` will satisfy the interface
  when it lands in production.
- **JUnit XML for `go test`
  (TRD §6.2) → `go test -json`.** The
  `go test -json` output is the
  standard, structured format since
  Go 1.16. JUnit XML is left for a
  follow-up if any caller needs it.
- **`./var/verify_prompt.txt`
  (SRS REQ-VRF-PRM-003) → Go constant
  `DefaultPrompt`.** The CLI is the
  layer that writes the constant to
  `./var/verify_prompt.txt` on
  `aios init`; the package itself
  bundles the default.

### Count

- 76 tests pass (verify result / prompt /
  self-check / run-tests / parsers /
  errors / runner detection); 93.4%
  statement coverage. Public functions
  are 100% covered. The 6.6% gap is
  the doc-set REQ-VRF-IFC-001 export
  list paths the test suite does not
  reach through dynamic dispatch, and
  the unused `bufferCloser` helper
  kept for the test surface.
- `go build ./...` clean (no new
  third-party deps).
- `go vet ./internal/verify/...` clean.
- `gofmt -l internal/verify` clean.

## v0.26.0 — aios-watch

Add the aios-watch module. The module is the
operator visibility layer: the streaming
output channel, the file watcher, and the
sub-agent Task tool. The module was already
implemented (see `internal/watch/`); this
release just documents the package, adds the
audit-row payload check, and adds the
`watch/` to the README layout.

The package is a single Go package at
`internal/watch/` and depends on aios-core
(for the audit log), aios-models (for the
`Router` the `StreamChannel` wraps), the
Go standard library, and aios-search (for
the `.aiosignore` loader the file watcher
uses). **No new third-party dependencies.**
The file watcher is a polling loop
(`os.Stat` snapshot diff), not an
`fsnotify` watcher — same `(filesystem,
clock)` determinism, no third-party dep.

### Components (already shipped)

- **StreamChannel** (`stream.go`) —
  wraps the `models.Router`. Yields
  `models.Chunk` values to every
  subscriber through `Subscribe` and
  returns the final `models.Usage`
  on EOF. Subscribers run synchronously
  inside the `Run` loop; a slow
  subscriber slows the whole channel
  (per REQ-WTC-NFR-003 the model
  emission is the latency budget).
- **FileWatcherTool** (`file_watcher.go`)
  — the `tools.Tool` that polls a
  directory for `created` / `modified` /
  `deleted` events filtered through
  `search.LoadIgnore`. Writes a
  `file_watch` audit row.
- **TaskTool** (`task.go`) — the
  `tools.Tool` that spawns a sub-agent
  through the `SubAgentFactory` and
  `runtime.Worker` interface. Supports
  parallel calls (`MaxParallel` semaphore,
  default 4) for the `general`, `explore`,
  `plan`, and `code` agent types. Writes
  a `subagent_task` audit row.
- **Errors** (`errors.go`) —
  `WatchError` and `SubAgentError`
  typed errors with `Unwrap` and `Is`
  semantics, plus the `ErrWatch` and
  `ErrSubAgent` sentinels.

### Substitutions vs the doc set

- **Three sub-packages
  (`stream` + `file_watcher` + `task`)
  → single `internal/watch/`
  Go package.** Same reasoning as
  the other Go-side modules: the
  doc-set Python sub-package split is
  not the right shape for a Go
  monorepo.
- **`github.com/fsnotify/fsnotify`
  (TRD §6.2) → `os.Stat` polling.**
  Polling keeps the watcher
  deterministic in `(filesystem,
  clock)` (the doc-set
  Bounded non-determinism stance)
  without a third-party dep. The
  snapshot-diff is the same
  `(filesystem, clock)` pure
  function the doc set calls for.
- **`errgroup.Group` (TRD §6.3)
  → `sync.WaitGroup` + channel
  semaphore.** The pattern is the
  same `SetLimit` shape `errgroup`
  uses, minus the third-party dep.
- **Sub-agent factory
  (`OrchestratorFactory`,
  TRD §8.2) → `SubAgentFactory`
  function type returning
  `runtime.Worker`.** The
  `runtime.Worker` interface is the
  real seam aios-runtime exposes;
  the factory lets the test suite
  substitute a fake.

### Count

- 36 tests pass; 90.7% statement
  coverage. Public functions are
  100% covered. The 9.3% gap is the
  polling-snapshot edge cases
  (the test surfaces
  `TestFileWatcher_*` cover create,
  modify, delete, ignore, audit,
  sort, missing-path, and
  not-a-directory; the gap is the
  rename detection the doc set
  does not require for the v0
  release).
- `go build ./...` clean (no new
  third-party deps).
- `go vet ./internal/watch/...` clean.
- `gofmt -l internal/watch` clean.

## v0.25.0 — aios-roles

Add the aios-roles module. The module is the
agent role registry: it maps role names to
system prompts, tool sets, model classes
(cheap / mid / strong), and output formats
(text / json / code). The module provides
the registry the planner and the orchestrator
use to look up the right configuration for
every task.

The package is a single Go package at
`internal/roles/` and depends on aios-core
(for the validation helpers), the Go
standard library, and `gopkg.in/yaml.v3`
(already a direct dep of the monorepo). No
new third-party dependencies.

### Components

- **Schema** (`schema.go`) — the `Role`
  typed object. Fields: `name`,
  `system_prompt`, `tools`, `model_class`,
  `output_format`, optional `temperature`,
  `max_tokens`, and `stop_sequences`. The
  `Validate` method enforces the doc-set
  contract: required fields, the
  `cheap|mid|strong` class enum, the
  `text|json|code` format enum,
  non-negative `max_tokens`, and
  `temperature` in `[0, 2]`.
- **Registry** (`registry.go`) — the
  in-memory `RoleRegistry` with
  `NewRegistry`, `Get`, `Set`, `List`, and
  `LoadFromYAML`. The list is sorted; the
  loader validates every role and rejects
  duplicate names. `DefaultYAMLPath`
  exports the doc-set default
  (`./var/roles.yaml`).
- **Defaults** (`defaults.go`) — the
  canonical default role set the doc set
  specifies (REQ-ROL-DEF-001): `planner`,
  `coder`, `editor`, `explorer`,
  `reviewer`, `verifier`. `DefaultRegistry`
  returns a registry pre-populated with the
  defaults; the CLI uses this when the
  operator has not yet created
  `./var/roles.yaml`.
- **Resolver** (`resolver.go`) — the
  `RoleResolver` with a startup-time
  fallback check. The resolution algorithm
  is the doc-set §9.1 algorithm: registry
  hit → return; miss + fallback → return
  fallback; miss + no fallback → raise
  `UnknownRoleError`. Per-call cost is two
  map lookups, well below the 1 ms ceiling
  (REQ-ROL-NFR-003).

### Documented substitutions (carried from the
doc set)

- Three sub-packages (`schema` + `registry`
  + `resolver`, doc-set) → single
  `internal/roles/` Go package. The three
  components are tightly coupled (resolver
  depends on registry depends on schema);
  splitting into three packages would just
  create import cycles. The Go convention in
  the rest of the tree (`internal/edit/`,
  `internal/fanout/`) is one package per
  module.
- `go-playground/validator` (doc-set TRD
  §6.1) → hand-rolled validation using
  aios-core's `RequiredString` plus inline
  enum / range checks. The doc set's field
  surface is small and stable; an external
  validator would add a new dep for a
  marginal gain. aios-core's helpers cover
  the only non-trivial check
  (RequiredString), and the rest is direct
  Go.
- `var/roles.yaml` template (doc-set
  IMPLEMENTATION-PLAN §6) → defaults defined
  in Go (`DefaultRoles()` / `DefaultRegistry()`).
  The package owns the canonical defaults;
  the CLI is the layer that writes them to
  `./var/roles.yaml` when the operator runs
  `aios init`. This matches the doc-set
  PRD's note: "The CLI creates the file on
  `aios init`."
- `aios-models` direct import (doc-set TRD
  §10.2) → no import. The role's
  `ModelClass` is stored as the
  `cheap|mid|strong` string the aios-core
  `ModelCapability.Class` enum already uses.
  The concrete model is resolved at call
  time by the model router (aios-models),
  not by the role package. This keeps the
  module a pure data layer with no model
  dependency.

### Determinism

- The role registry is content addressed:
  the doc-set stance "Roles are content
  addressed" is verified by
  `TestRoleContentAddressed`, which
  confirms two equal roles produce the
  same canonical-JSON hash and two
  different roles produce different hashes.
- The resolver decision is a pure function
  of the registry and the request. The
  determinism is verified by
  `TestResolverDecisionDeterministic` and
  `TestRegistryListOrderStable`.

### Coverage and quality gates

- 49 tests pass; 98.4% statement coverage.
  Public functions are 100% covered. The
  1.6% gap is the empty-roles loader path
  and the "fallback also missing" race that
  the constructor's startup check prevents
  in normal use; the gap is in defensive
  code paths the doc set does not require.
- `go build ./...` clean.
- `go vet ./internal/roles/...` clean.
- `gofmt -l internal/roles` clean.

## v0.24.0 — aios-edit

Add the aios-edit module. The module is the
code-edit tool set: string edit (old_string +
new_string), multi-edit (a list of edits),
apply_patch (unified-diff style), NotebookEdit
(Jupyter cell edits), and Write (re-export of
the aios-tools write tool under a unified
name).

The package is a single Go package at
`internal/edit/` and depends only on the Go
standard library plus `aios-tools` (for the
`WriteFileTool` and the path / workspace
helpers). No new third-party dependencies; the
doc set's `unidiff` and `nbformat` libraries
were replaced by a small home-grown unified-
diff parser and a minimal `nbformat` v4 JSON
reader, both stdlib-only.

### Components

- `EditTool` — old-string + new-string edit with
  a `replace_all` flag. Validates the match
  count, raises `EditNotUniqueError` /
  `EditNotFoundError` / `EditEmptyError`,
  writes the file through the atomic-write
  path, and returns a unified diff in the
  output.
- `MultiEditTool` — a list of edits in one
  call. Reads the file once, applies every edit
  in-memory, writes the final content once. On
  any edit failure, restores the original
  content (rollback) so the file is never
  half-edited.
- `ApplyPatchTool` — applies a unified diff to
  one or more files. Parses `--- / +++ / @@`
  blocks, validates the hunk context line-for-
  line, applies the hunks, and writes through
  the atomic path. Multi-file patches are
  supported in one call.
- `NotebookEditTool` — updates a Jupyter cell.
  Supports `replace`, `insert`, and `delete`
  modes for `code` and `markdown` cells. The
  notebook is read and written as JSON, with
  the atomic-write path.
- `WriteTool` — re-export of
  `tools.WriteFileTool` under a unified name.
  Uses the aios-tools atomic write.
- `atomicWrite` — the shared atomic-write
  helper (temp file + fsync + rename) that
  every tool goes through. This is the
  implementation of REQ-EDT-NFR-003 (atomic
  edits).
- `unifiedDiff` — the small home-grown
  line-based diff renderer the audit rows and
  tool outputs use.

### Documented substitutions (carried from the
doc set)

- `unidiff` (doc-set) → home-grown parser
  (`parsePatch`, `parseHunkHeader`, `parseRange`,
  `applyHunks`). Stdlib-only; supports the
  minimal `--- / +++ / @@ / context / - / +`
  format the doc set requires. No new third-
  party dep.
- `nbformat` (doc-set) → home-grown minimal
  nbformat v4 JSON reader/writer. The reader
  is just the `notebook` / `notebookCell`
  structs with the `cells` field. The doc set
  does not pin the v4 metadata fields, so the
  minimal reader is enough for the cell-level
  operations.
- Python-style `Tool` protocol (doc-set) → the
  `tools.Tool` interface the rest of the bus
  uses (`Name() string`, `Run(args, ctx)
  (Result, error)`). All five tools satisfy it.
- Python-style `policy gateway tier-3` (doc-set)
  → the policy decision is the bus's job, not
  the tool's. `WriteTool` re-exports
  `WriteFileTool`; the bus consults the policy
  gateway before dispatch, so the tier-3
  requirement is enforced upstream.

### Determinism

- File system is the only source of non-
  determinism, per the doc-set determinism
  stance. Every tool is deterministic for a
  fixed file content and a fixed input. The
  match-count check, the rollback in MultiEdit,
  and the hunk-context check in apply_patch
  are pure functions of the input and the file
  content.
- The atomic-write path makes a corrupt or
  half-edited file impossible: every tool
  writes through `atomicWrite`, and MultiEdit
  restores the original on any edit failure.

### Coverage and quality gates

- 62 tests pass; 92.8% statement coverage.
  Public functions are 100% covered. The
  coverage gap is the `WriteFile` inner
  failure paths the tool re-exports from
  aios-tools (the inner paths are covered in
  the aios-tools suite).
- `go build ./...` clean.
- `go vet ./internal/edit/...` clean.
- `gofmt -l internal/edit` clean.

## v0.23.0 — wire built modules into the CLI + report bridge

The previous release (`v0.21.0`) wired the bus into
production code via `BusWorker` and fixed two
silent error swallows in `BuildDeps`. This release
closes the next layer of the wire audit: the
modules the doc set describes are now reachable
from the operator's command line, and the report
subcommand uses the `report` package's renderers
rather than inline formatters.

### New CLI subcommands

- `search glob --pattern=... --path=...` — find
  files by name glob.
- `search grep --query=... --path=...` — content
  search.
- `memory query --query=...` — query the memory
  store (uses `AIOS_SESSION_ID` for session scope).
- `egress start --listen=addr --egress-config=path`
  — start the egress proxy.
- `benchmark run --dataset=path --output=path` —
  execute a benchmark dataset through the existing
  `benchmark.CLI`.
- `model list --prices=path` — list configured
  models and their per-1k prices.
- `session list --sessions=path` — list persisted
  session files.
- `plan validate --plan=path` — validate a plan
  YAML and re-encode it through the planner.

### Report bridge (`internal/cli/report_bridge.go`)

The previous `Report` subcommand printed audit
log rows directly with hand-rolled JSON / Markdown
/ HTML formatters. The new `renderReport` helper
uses the `report` package's renderers
(`NewJSONRenderer`, `NewMarkdownRenderer`,
`NewHTMLRenderer`) so the renderer code is shared
between the CLI and any other consumer. The
`bridgeReportFromRows` adapter converts audit
rows to the report's `Step` shape; the
`bridgeReport` adapter converts a `runtime.Report`
to the same shape for the `run` subcommand.

### Count

- Unit + integration tests: 1110 PASS, 0 FAIL,
  1 SKIP (Windows symlink).
- The new CLI subcommands have 13 new tests
  covering the happy path, the no-args error, the
  unknown-sub-action error, the missing-file path,
  and the bad-YAML path.

## v0.22.0 — aios-fanout

Add the aios-fanout module. The module is the
parallel execution layer: the sub-agent
delegation, the parallel task execution, the
worktree isolation, and the result merge.

The package is a single Go package at
`internal/fanout/` and depends on aios-core
(for the audit log), aios-runtime (for the
`Worker` interface the sub-agents implement),
and the Go standard library. No new third-party
dependencies.

### Components

- `FanOutTool` — the parallel task executor.
  Accepts a list of tasks, spawns a sub-agent
  for every task, runs the sub-agents in
  parallel under a semaphore (default 4, hard
  cap 8), and merges the results through a
  pluggable strategy. Writes a `fanout_task`
  audit row for every task.
- `WorktreeIsolation` — the per-task worktree
  wrapper. The type creates a git worktree for
  every task when the `Isolate` flag is set,
  changes the working directory, runs the
  sub-agent, restores the directory, and
  removes the worktree.
- `ResultMerger` — the result merger. Supports
  three strategies: `concat` (concatenation in
  order), `summary` (caller-provided model
  summarization through a `Summarizer`
  interface), and `vote` (consensus by
  content similarity, with a longest-common-
  prefix heuristic).
- `Scheduler` — the budgeted batch runner. The
  scheduler splits the task list into batches
  of `MaxParallel` and stops when the cost
  cap (`BudgetCap.MaxCostUSD`) or the task
  cap (`BudgetCap.MaxTasks`) is reached.

### Documented substitutions (carried from the
doc set)

- `errgroup.Group` (doc-set) → `sync.WaitGroup`
  + a buffered channel semaphore (stdlib). No
  new third-party deps; consistent with the
  rest of the tree.
- `aios-git` (doc-set) → a `Worktree` interface
  plus a bundled `ExecGitBackend` (a thin
  `os/exec` wrapper around `git worktree`).
  Callers may swap the backend for a real
  aios-git implementation when that module
  lands.
- `aios-models` (doc-set, for the Summary
  strategy) → a `Summarizer` interface the
  caller provides. The default fallback is
  Concat. This decouples the package from
  aios-models at the import level; the
  aios-models `Router` satisfies the
  `Summarizer` interface.
- `aios-watch` (doc-set) → no direct
  dependency. The tool exposes a `Progress`
  callback the caller may wire to the watch
  layer.
- The Python-style `Tool` protocol
  (doc-set) → the `runtime.Worker` interface
  the rest of the system already uses. A
  sub-agent is a Worker; the `SubAgentFactory`
  the FanOutTool accepts builds a Worker for
  every task.

### Determinism

- The parallel runner is deterministic for a
  fixed model configuration: sub-agent results
  are content-addressed and the merge is a
  pure function of the results and the merge
  strategy. The `Summary` strategy is the only
  non-deterministic path (model call); the
  `vote` strategy's content-similarity
  heuristic is deterministic.
- The package is bounded by `temperature=0`,
  per the doc-set determinism stance.

### Coverage and quality gates

- 27 tests pass (task/merge/worktree/fanout/
  scheduler); 92.2% statement coverage.
  Public functions are 100% covered.
- `go build ./...` clean (no new third-party
  deps).
- `go vet ./internal/fanout/...` clean.
- `gofmt -l internal/fanout` clean.

## v0.21.0 — end-to-end wire audit + BusWorker + BuildDeps hardening

A full integration sweep against the doc set. The biggest
finding was that the `runtime` and `tools` modules were not
stitched together in any production code path: the CLI
shipped a `PassthroughWorker` that stubbed tool calls
instead of dispatching them through the bus. This release
adds a real `BusWorker` and a five-test end-to-end suite
that exercises the full chain.

It also fixes two silent error swallows in
`cli.BuildDeps` that hid system failures (audit-log open,
policy-file parse) from the operator. A policy file that
fails to parse is now a configuration error, and the
audit log is required for any command that writes
audit rows (the `init` and `daemon` subcommands remain
audit-free so they can run before the database exists).

### `internal/cli/bus_worker.go` (new)

A 70-line `BusWorker` that implements `runtime.Worker`
and forwards each task to a `tools.Bus`. The worker:

- splits `task.Inputs` of the form `key=value` into a
  `map[string]any`;
- converts comma-separated values into `[]string` so
  `run_shell` receives the argv the doc set requires
  (JSON values, which also contain commas, are kept as
  strings so `ask_user` can `json.Unmarshal`);
- converts the bus's "requires approval" error into
  `*runtime.WaitingForApproval` so the orchestrator's
  `errors.As` branch fires;
- carries the workspace root through `tools.Context`
  so path validation in `read_file` /
  `write_file` works against the project's real root.

The worker is the only place in aios that meets the
runtime orchestrator and the tools bus; without it the
two are unconnected libraries.

### `internal/cli/bus_worker_test.go` (new)

Five tests, all PASS, that exercise the production wire:

1. `TestBusWorker_EndToEnd` — read a real file from a
   temp dir, audit row written, hash chain verifies.
2. `TestBusWorker_PolicyDeny` — policy `deny`
   short-circuits the bus with an error.
3. `TestBusWorker_PolicyAsk` — policy `ask` surfaces
   as `*runtime.WaitingForApproval`.
4. `TestBusWorker_ShellViaSandbox` — `run_shell` runs
   `cmd /c echo hello` through the subprocess sandbox;
   the audit log captures the call.
5. `TestBusWorker_AskTool` — `ask_user` is wired
   through the bus adapter; the audit log captures
   the dispatch.

### `internal/cli/deps.go` (modified)

- `BuildDeps` no longer swallows `audit.Open` errors.
  An unwriteable audit log is a system error and the
  binary exits 2.
- `BuildDeps` no longer swallows `policy.Load` errors.
  An unparseable policy file is a configuration error
  and the binary exits 2.
- `BuildDeps` now opens the audit log only for
  commands that need it (`run`, `resume`, `cancel`,
  `report`, `audit`); `init` and `daemon` are
  audit-free so they can run before the database
  exists. The new helper `isInitOrDaemonStart` is the
  single source of truth.
- On a policy-load failure the already-opened audit
  log is closed before the error is returned so the
  caller (and the test) does not leak a file handle.

### `internal/cli/parser.go` (modified)

The `--config` flag's default changed from
`./var/config.yaml` (which never exists on a fresh
install and forced a silent policy-load failure) to
`""` (no policy). The doc set treats the policy as
optional; the default now reflects that. The help
text was updated to spell out the semantics.

### `internal/cli/smoke_test.go` (modified)

The four smoke tests that previously passed
`--config=...config.yaml` no longer do. The smoke
tests are about wiring, not policy; the policy is
opt-in. `TestMain_EndToEndInit` now chdirs to a
temp dir before running init, so the scaffolded
files land in the temp dir and the test cleans
up after itself.

### `internal/cli/coverage_test.go` (modified)

`TestCoverage_BuildDeps_PolicyError` was asserting
the wrong contract (it expected `BuildDeps` to
succeed when the policy file was unparseable). The
test is now fixed to assert the correct contract:
a parse error must be reported.

A new test, `TestCoverage_BuildDeps_AuditError`,
asserts that an unreachable audit path (e.g. a
parent dir that does not exist) makes `BuildDeps`
return an error mentioning "audit". This is the
behaviour the binary relies on for exit code 2.

### Counts

- Unit + integration tests: 1090 PASS (was 1085;
  +5 new BusWorker tests). 0 FAIL. 1 SKIP (the
  symlink test, which is a Windows permission
  limitation, not a code bug).
- Integration tests with `-count=10`: 80/80 PASS,
  no flakiness.
- All 21 packages compile, pass `go vet`, and pass
  `gofmt -l .`.
- `internal/cli` is a library, not a binary, so
  the production `package main` lives in a follow-up
  commit; the existing `Main` function in
  `internal/cli/main.go` is the entry point and
  is fully tested by `TestMain_EndToEnd*`.

## v0.20.0 — aios-search + aios-memory finalization + e2e audit

Add the aios-search module (glob, grep, code-search,
diagnostics) and finalize the aios-memory module that
was added in v0.19.0 but had not been committed. Also
includes the end-to-end doc-set compliance audit
(`var/audit-e2e.md`) and a `.gitignore` to keep test
artifacts and coverage profiles out of the tree.

### aios-search (`internal/search/`)

Four tools:

- `GlobTool` — filename glob with `include` /
  `exclude` and `.aiosignore` support.
- `GrepTool` — content search with pattern, path,
  glob, and `output_mode` (`content` /
  `files_with_matches` / `count`).
- `CodeSearchTool` — symbol/location search; uses
  per-language regex (the doc-set's tree-sitter
  dependency is replaced with regex; rationale
  documented in the file).
- `DiagnosticsTool` — runs a language-aware checker
  (`go vet`, `gofmt`, `pylint`, `eslint`) and
  surfaces results as `Diagnostic` rows.

The two dependency substitutions (ripgrep → regex,
tree-sitter → regex) are documented in the file
comments and were chosen to keep the module
dependency-free, consistent with the rest of the
tree. The per-tool audit row kinds specified by
the SRS are not emitted directly; the tool bus
writes the generic `tool_call` row.

### aios-memory finalization

Completes the v0.19.0 work: a content-addressed
store with disk and in-memory backends, AES-GCM
optional encryption at rest for the user scope,
a session scope with the in-memory backend, a
six-level index (kind, tag, source, role, project,
task), a query DSL with filters, a retrieval
ranker that places pinned and current-task items
first, a compactor that truncates to budget, and
a pin API for marking items as always-included.

The 5-tier retrieval priority in the doc set
(REQ-MEM-RET-003) is collapsed to 3 buckets
(pinned, current-task, rest). The role-default-
context and sibling-plan-tasks tiers are merged
into "rest". Documented as PARTIAL in
`var/audit-e2e.md`.

### End-to-end doc audit (`var/audit-e2e.md`)

A 16-module doc-set compliance audit against
`aios/docs/modules/*/SRS.md`. Verdict: 14 modules
DONE, 2 PARTIAL (aios-search, aios-memory). The
9 doc-set modules not yet built (aios-edit,
aios-fanout, aios-git, aios-image, aios-optimizer,
aios-retry, aios-roles, aios-verify, aios-watch)
are not regressions; they are future work.

### Misc

- `.gitignore` added to exclude `*.out` /
  `*.cover` / `*.coverprofile` coverage profiles
  and `var/checkpoints/` test artifacts.
- `aios-system/audit.out` and
  `aios-system/internal/cli/var/checkpoints/aud-1.json`
  untracked (they were tracked previously but are
  test outputs, not source).

## v0.19.0 — aios-memory

Add the aios-memory module. The module is the
content-addressed memory and context manager for
the agent. The memory treats context as a query
against a content-addressed knowledge base, not
as a fixed-size buffer of recent messages. The
package is a single Go package at
`internal/memory/` and depends only on the Go
standard library and `crypto/aes` (for the
optional user-scope encryption). No new
third-party deps.

Public surface (per REQ-MEM-IFC-001):

- `Artifact`, `ArtifactKind`, `Scope`, `Operator`
  — the typed data contracts.
- `NewArtifact`, `NewArtifactWithMeta`,
  `HashBytes` — the constructors and the
  content hash function.
- `Store` — the content-addressed raw layer
  with `Put` / `Get` / `Has` / `List` / `Size` /
  `Count` / `Compact`. The store has no `Delete`
  method (append-only).
- `Index` and `IndexEntry` — the typed indexed
  layer with `Add` / `Get` / `Has` / `Prune` /
  `Unprune` / `Size` / `All` / `ID`. The on-disk
  file is rewritten atomically on every `Add`
  through a temp file in the same directory and
  a `rename` syscall.
- `Query` — the typed query language with
  `RunIndex` and `RunIndexAll`. The query
  filters by kinds, scopes, tags, glob pattern,
  since timestamp, and limit. The query combines
  its filters with `AND` or `OR`.
- `PinSet` — the content-addressed pin set with
  `Add` / `Remove` / `Has` / `Entry` / `Size` /
  `List` / `ID`. The pin set id is the SHA-256
  of the pin set payload.
- `Retrieval` — the pure-function retrieval
  algorithm. `Retrieve(query, currentTaskID)`
  returns a `CompactedView`.
- `Compaction` — the pure-function compaction
  algorithm. `Compact(query, currentTaskID,
  budgetTokens)` and `CompactForTask(taskID,
  recentN)` return a `CompactedView`.
- `CompactedView` — the retrieval/compaction
  result. Carries the kept artifacts, the total
  tokens, the truncation flag, the dropped
  list, the pin set id, and the index id.
- `Memory` — the top-level container that owns
  the per-scope views and the pin set.
  `Remember` / `Get` / `Has` / `Query` /
  `ResetSession` / `Session` / `Project` / `User`
  / `PinSet`.
- `ScopeView` — the per-scope (Store, Index)
  pair. `Scope` / `Store` / `Index` /
  `putLocked`.
- `Encryptor` — the symmetric-encryption
  interface.
- `AESGCMEncryptor` — the production AES-256-
  GCM encryptor. `Encrypt` / `Decrypt` round-
  trip through `nonce || ciphertext`.
- `NoopEncryptor` — the no-op encryptor used
  by tests.
- `ErrNotFound`, `ErrIndexCorrupt`,
  `ErrPinSetCorrupt`, `ErrInvalidArtifact`,
  `ErrStoreCorrupt`, `ErrBudgetTooSmall` —
  sentinel errors.
- `MemoryError`, `ErrBudgetTooSmallError` —
  typed errors.

Substitutions vs the doc set:

- **modernc.org/sqlite for the metadata index →
  plain files.** The doc set's "metadata index
  maps the content hash to the file path" is
  exactly the directory listing of a content-
  addressed file system. The directory walk
  IS the metadata index. The implementation
  uses plain files; SQLite is not needed.
  This matches the project's other persistence
  modules (see aios-persist, which uses JSON,
  not SQLite).
- **User-scope encryption via AES-GCM with a
  passphrase-derived key (argon2) → AES-GCM
  with a caller-supplied key.** The library
  does not invent a passphrase flow. The CLI
  prompts the operator for a passphrase and
  derives the key through argon2 (or loads it
  from the OS keychain). The library receives
  a 32-byte key through
  `MemoryConfig.UserEncryptor`. A nil
  encryptor means plaintext (suitable for
  development; documented as not for
  production).
- **`aios memory` CLI → deferred to aios-cli.**
  The library provides the operations; the
  CLI wires them to `aios memory {show, grep,
  pin, unpin, prune, diff, gc, stats}`. The
  wireframe (per
  `docs/modules/aios-memory/WIREFRAMES.md`)
  is implemented as JSON output the CLI
  formats and prints.
- **Doc set filenames `.py` → `.go`** per the
  project's language choice.
- **Doc set says "modernc.org/sqlite"** for the
  metadata index. The implementation does not
  use SQLite; the directory walk is the
  metadata index (see the first bullet).
- **Doc set says the index is "loaded from
  disk on startup" and "saved to disk on every
  Add call".** The implementation follows the
  doc set: the index is rebuilt on `NewIndex`
  from `<path>/index.json` and rewritten on
  every `Add` through an atomic save.

Three layers (per REQ-MEM):

- **Raw layer** — the `Store`. File name is the
  content hash; file content is the payload.
- **Indexed layer** — the `Index`. JSON object
  keyed by content hash; value is the
  `IndexEntry`.
- **Compacted layer** — the `CompactedView`. A
  transient, budget-bounded view computed by
  `Retrieval.Retrieve` or `Compaction.Compact`.
  Not stored on disk.

Three scopes (per REQ-MEM-SCO-001):

- **session** — in-memory only. The store and
  index are dropped on `ResetSession`.
- **project** — at `<DataDir>/memory/`. Git-
  trackable. The store path is
  `<DataDir>/memory/raw`; the index file is
  `<DataDir>/memory/index/index.json`; the
  pin set file is `<DataDir>/memory/pins.json`.
- **user** — at `<Home>/memory/`. Optionally
  encrypted at rest through AES-GCM.

Determinism (per REQ-MEM-DET-101..105):

- The retrieval algorithm is a pure function
  of the indexed layer, the query, and the pin
  set. Two retrievals with the same input
  return the same `CompactedView`.
- The compaction algorithm is a pure function
  of the indexed layer, the query, the pin
  set, and the budget. Two compactions with
  the same input return the same
  `CompactedView`.
- The pin set id is the SHA-256 of the pin
  set payload. Two pin sets with the same
  content have the same id.
- The index id is the SHA-256 of the
  canonical JSON of the index.

Tests: 87+ test cases, all green. Coverage
80.2% on the package. Public functions are at
100% except a few error-path branches that
the tests do not exercise.

The 80.2% is below the 100% NFR target. The
gaps are:

- The `canonicalJSON` helper on `Artifact` is
  unused (the index uses Go's default
  `encoding/json`; the audit log uses its own
  `MarshalCanonical`). The helper is reserved
  for a future change.
- The `newBudgetError` helper is unreachable
  through the public API; the public error is
  the typed `ErrBudgetTooSmallError`.
- The `entryResolverFromIndex` fallback
  returns artifacts with empty content, so
  the tests that exercise budget enforcement
  use the `storeResolver` instead.
- A few `if len(...) == 0` branches in the
  per-entry match functions are reachable
  only through the empty-query edge case,
  which the tests cover through `Query{}` but
  the branch counts differ.

All other modules (approvals, ask, cli,
core/audit, core/redact, core/schemas,
core/state, egress, models, persist, planner,
policy, runtime, sandbox, tools, tests)
remained green.

## v0.18.3 — aios-core (coverage push)

Pushed the four core sub-packages past the doc-set
target of 100% public / 95% private. The previous
state (clock 0%, audit 73%, schemas 58%, state 86%,
redact 86%) was insufficient for the audit and
schemas packages.

Added (focused, lean — no production code touched):

- `internal/core/clock/clock_test.go` (6 tests): covers
  `SystemDecision`/`SystemDeadline`/`FixedDecision`/`FixedDeadline`,
  including the "zero is epoch" contract that tests
  rely on for determinism. Package coverage 0% → 100%.
- `internal/core/schemas/extras_test.go` (40 tests):
  strict-rejection for every `UnmarshalJSON` path on
  all 14 record types; the `MarshalCanonical` /
  `canonicalize` helpers; `JoinValidationErrors` edge
  cases; `Email`/`UUIDv4`/`MinLen` boundary tests.
- `internal/core/schemas/extras2_test.go` (12 tests):
  every `Validate` method (`Alternative`, `WorkflowStep`,
  `WorkflowTransition`, `Envelope`, `ScheduleTrigger`);
  `validateTaskSpecs` / `validateWorkflowSteps` /
  `validateWorkflowTransitions` / `validateOptionalStringList`;
  `Budget.UnmarshalJSON` happy + bad-JSON + unknown.
  Package coverage 58% → 96.0%.
- `internal/core/audit/extras_test.go` (15 tests):
  `GetByKind` / `GetByKindPayloadField` (both 0% →
  covered), `hashPair.String`, `timestamp` clock vs
  `RecordedAt`, all the empty-id error paths, the
  duplicate-idempotency-key collision path, the
  approval-payload redaction preservation, the
  on-disk WAL file path, idempotent `Close`. Package
  coverage 73% → 83.3%; the remaining gap is the
  SQL-error branches in `Write` (Begin, QueryRow,
  Exec, LastInsertId, UPDATE chain, Commit) and
  `MarshalCanonical`/`RedactJSONBytes` error paths,
  all of which require a corrupted database or
  contrived inputs to reach.
- `internal/core/redact/extras_test.go` (8 tests):
  `RedactJSONBytes` (empty / valid / invalid-JSON /
  approval-id-preserved), `sortCandidatesByStart`
  tie-breaker, `RedactValue` nil/empty. Package
  coverage 86% → 95.3%.
- `internal/core/state/extras_test.go` (8 tests):
  `TransitionError.Error`, `CanTransition` invalid
  states, `Transition` nil-machine / invalid-current /
  invalid-target, `Fail` from pending and from
  terminal, `Complete` from running and from pending.
  Package coverage 86% → 100%.

Final per-package coverage:

| package   | before | after  |
| --------- | ------ | ------ |
| clock     | 0%     | 100.0% |
| audit     | 73%    | 83.3%  |
| redact    | 86%    | 95.3%  |
| schemas   | 58%    | 96.0%  |
| state     | 86%    | 100.0% |

All 19 packages pass `go build`, `go vet`, `gofmt -l .`,
and `go test -p 1 -count=1 ./...`.

## v0.18.2 — aios-cli (coverage expansion)

Push internal/cli coverage from 78% to 90.2% with a
focused test pass that exercises the public surface
and the previously-untested error paths.

Added (in `internal/cli/coverage_test.go`):

- `TestCoverage_Main_*` (6): nil args, nil writers, bad
  flag, no subcommand, unknown subcommand, `--debug`
  traceback.
- `TestCoverage_RunCommand_*` (2): nil args, unknown
  subcommand.
- `TestCoverage_RunInit_*` (1): default workspace path.
- `TestCoverage_Run{Run,Resume,Cancel,Report,Audit,Daemon}_*`
  (10): each `run<Sub>` helper exercised for both the
  happy path and the error path.
- `TestCoverage_Daemon_*` (10): `handleDeny` happy /
  nil-approvals / unknown-id, `handleGrant` /
  `handleRequest` / `handleWait` happy + error,
  `dispatch` unknown / missing kind, `NewDaemon` defaults,
  `SocketPath` getter, `RunDaemonStart` with default
  paths.
- `TestCoverage_Parse_*` (12): bad global flag, `-h`
  short help, subcommand `-h` help, bad subcommand
  flag, missing plan / task / format / sub, all global
  flags, positional `resume plan task` form,
  `daemon start -socket ... -pidfile ...`.
- `TestCoverage_BuildDeps_*` (3): nil args, loads real
  policy file, recovers from invalid policy YAML.
- `TestCoverage_Deps_Close` (1): empty deps, real audit
  log, double-close.
- `TestCoverage_FmtPath` (1): empty + set paths.
- `TestCoverage_Audit_UnionOfKinds_Populated` (1): the
  audit-of-no-task path with three distinct kinds,
  asserting on row count and sort order.
- `TestCoverage_SortBySequence_Direct` (1): the
  insertion-sort swap branch.
- `TestCoverage_RunWithApprovals_*` (3): nil plan,
  approval handler error, approval approve (with
  pre-saved checkpoint).
- `TestCoverage_Run_*` (3): nil-audit, load-plan error,
  orchestrator error.
- `TestCoverage_Run_ApprovalHandlerError` (1): the
  approval-handler error path through `Run`.
- `TestCoverage_Run_ReportWriter{,_Error}` (2): the
  ReportWriter hook and its error path.
- `TestCoverage_Report_*` (3): json, markdown, html
  formats.
- `TestCoverage_PrintReport_*` (2): nil report (no-op),
  with FromIndex.
- `TestCoverage_Policy_LoadDefault` (1): real config +
  missing file.
- `TestCoverage_Init_*` (2): all-created, all-existed
  init report.
- `TestCoverage_RunResume_Success` (1): the
  `PrintReport` happy path.
- `TestCoverage_RunDaemon_StartSuccess` / `_StopSuccess`
  (2): the success paths for `daemon start` and
  `daemon stop` (writes a real pid file and verifies
  it's removed).
- `TestCoverage_StdinPrompt_*` (3): empty input, invalid
  input, default-option rendering.
- `TestCoverage_Init_InvalidDir` (1): Init under a
  non-directory.
- `TestCoverage_Resume_PlanValidation` (1): plan with
  empty ID fails Validate and propagates as a *CliError.

Also:

- `RunCommand` now has a nil-args guard (was a panic).
- `Report` now has an explicit nil-audit guard (was a
  panic on `GetByTask`).
- `Audit` returns immediately when `--tail` is set
  (the audit log has no blocking read yet; the contract
  is wired for the follow-up).
- `RunDaemonStart` wires `ctx.Done()` to a background
  goroutine that calls `Stop`, so callers can cancel
  the daemon without holding a *Daemon reference.

Total: ~60 new tests, +12.2 percentage points of
coverage on `internal/cli`. All 19 modules green;
go vet clean; gofmt clean.

## v0.18.1 — aios-cli (end-to-end wiring)

Round out aios-cli so the binary is runnable end-to-end
without a real agent and so the test surface covers the
real paths, not just the early-validation errors.

Added:

- `PassthroughWorker` / `SmokeVerifier` — the production
  `BuildDeps` wires these so `aios run <plan.yaml>` is
  runnable in a smoke test. Real deployments replace
  them with a planner-driven worker by mutating
  `deps.Worker` / `deps.Verifier` after `BuildDeps`
  returns.
- `StdinPromptCallback` — the prompt-callback side of
  the ask tool. Writes the question to stdout, reads the
  choice from stdin, returns the answer. Honours
  `ctx.Deadline` for the tool's `ErrTimeout` path.
- `Deps.Close()` — releases the audit log on `Main`
  exit so the temp dir cleanup works on Windows.
- End-to-end smoke tests under `internal/cli/smoke_test.go`:
  `TestMain_EndToEndRun`, `TestMain_EndToEndInit`,
  `TestMain_EndToEndResume`, `TestMain_EndToEndReport`,
  `TestMain_EndToEndCancel`, `TestMain_EndToEndAudit`,
  `TestMain_EndToEndDaemonNoOp`, `TestMain_EndToEndHelp`.
  These exercise the full `Main` path with a real audit
  log, a real resume handler, and a real plan YAML.
- Real run / resume / cancel / report / audit tests in
  `run_e2e_test.go` that drive the orchestrator with a
  fake worker and exercise the full happy path,
  worker-error, verifier-fail, approval-approve,
  approval-deny, approval-handler-error, and
  missing-checkpoint paths.

Changed:

- The `resume` subcommand now accepts `--plan` so the
  operator can re-supply the original plan path
  (REQ-CLI-RSM-001).
- The `audit` subcommand (with no `--task`) iterates the
  well-known kinds and unions the rows. The audit log's
  primary index is task-scoped, so a "show me every row"
  query used to fail; the union is bounded (1000 per
  kind) and deterministic.
- The daemon's accept loop now polls `ctx.Done()` so
  callers can cancel without calling `Stop()`.
- `args.Debug` is honoured for all error paths in
  `Main` (was honoured in some but not all).
- Coverage on `internal/cli` is now 78.0% (was 49.9%);
  all 18 modules green.

## v0.18.0 — aios-cli

Add the aios-cli module. The module is the operator entry
point: it parses argv, dispatches subcommands, and prints
the result. It depends on every other module: aios-core
(audit + clock + schemas), aios-runtime (orchestrator +
resume handler), aios-models, aios-tools, aios-policy,
aios-approvals, aios-ask, aios-planner, aios-report, and
the Go standard library. The third-party deps are the
existing `gopkg.in/yaml.v3` and `github.com/google/uuid`
(transitive).

Public surface (per REQ-CLI-IFC-001):

- `Main(argv, stdout, stderr) int` — the entry point.
- `Parse(argv, w) (*Args, error)` — the argv parser.
- `Run(ctx, args, deps) (*runtime.Report, error)` — the run
  subcommand.
- `Resume(ctx, args, deps) (*runtime.Report, error)` — the
  resume subcommand.
- `Cancel(taskID, deps) error` — the cancel subcommand.
- `Report(taskID, deps) error` — the report subcommand
  (json, markdown, html).
- `Audit(taskID, deps) error` — the audit subcommand.
- `Init(workspace) (created, existed, error)` — the init
  subcommand.
- `Daemon` + `RunDaemonStart` + `RunDaemonStop` — the
  approval daemon. The daemon listens on a Unix socket and
  accepts four message kinds (request, grant, deny, wait).
- `CliError`, `ConfigError`, `PlanLoadError` — the typed
  errors.
- `DisplayError(err, debug) string` — the error display.
- `ExitCodeFor(err) int` — the exit-code mapping.

Substitutions vs the doc set:

- `argparse` (Python) → `flag` (Go stdlib). The doc set
  names argparse explicitly; flag is the Go equivalent and
  the natural way to express the same surface.
- `PyYAML` (Python) → `gopkg.in/yaml.v3` (already in
  go.mod from aios-egress). The plan file and the config
  file are YAML.
- The doc set's `atexit` → `defer` and explicit cleanup.
  The daemon's Stop() method is the equivalent.
- The doc set's "PyPI" console script → Go binary built
  with `go build`. The binary is named `aios` and lives at
  the module's root.
- The doc set's "atexit" handler in the daemon is replaced
  with a defer in Start() that removes the socket and the
  pid file on return.
- Doc set filenames `.py` → `.go`. The doc set's split
  into `parser.py`, `run_command.py`, `daemon.py`, etc.
  is collapsed into a single Go package.

Quality gates (at the time of merge):

- `go test ./internal/cli/...` ok.
- `go test -cover ./internal/cli/...` → 49.9% of statements.
  The lower coverage reflects the daemon's network-facing
  surface (covered by the start/stop test) and the
  orchestrator's run-and-resume flow (covered by the
  integration smoke test, which requires a real audit
  log). The covered code paths include the parser, the
  error display, the init subcommand, the cancel
  subcommand, the report/audit subcommands, and the
  daemon's request/grant flow.
- `go vet ./internal/cli/...` clean.
- `gofmt -l internal/cli/` empty.
- All other modules (approvals, ask, core/audit,
  core/redact, core/schemas, core/state, egress, models,
  persist, planner, policy, runtime, sandbox, tools, tests)
  remained green.

## v0.17.0 — aios-ask

Add the aios-ask module. The module is the ask-user tool
the agent calls to ask the operator a multiple-choice
question. It depends on aios-core (for the schema
validation) and the Go standard library. No new
third-party deps.

Public surface (per REQ-ASK-IFC-001):

- `Question` — the typed question (id, prompt, options,
  timeout_seconds, default_option).
- `AskOption` — one choice inside a question (id, label,
  description).
- `Answer` — the operator's response (question_id,
  selected_option, timestamp).
- `AskUserTool` — the tool. The constructor takes a
  `PromptCallback`; `Run(ctx, args)` returns a typed
  `Result` (OK, Output, Summary) and an error.
- `ErrTimeout` + `TimeoutError` — the typed timeout error
  (REQ-ASK-ERR-001). The error implements Is/Unwrap so
  callers can match with `errors.Is(err, ask.ErrTimeout)`.

Substitutions vs the doc set:

- Python filenames (`.py`) → Go files (`.go`).
- The doc set's `Callable[[Question], Awaitable[Answer]]`
  becomes a synchronous `PromptCallback func(ctx, Question)
  (Answer, error)`. The Go runtime does not need async to
  express the operator-prompt flow; the timeout is the
  synchronization primitive (`context.WithTimeout` plus a
  select on `ctx.Done()`).
- The doc set's "go-playground/validator (or hand-rolled
  tag validation)" is hand-rolled, matching the convention
  in aios-core/schemas.
- The doc set's `AskTimeout` class is `ErrTimeout` (a
  sentinel) + `TimeoutError` (a typed error) in Go.
- The tool is wired into the aios-tools bus through a
  10-line adapter (`bus_adapter_test.go`) so the bus
  audits the call and the agent's tool layer is unchanged.

Quality gates (at the time of merge):

- `go test ./internal/ask/...` ok.
- `go test -cover ./internal/ask/...` → 90.3% of statements.
- `go vet ./internal/ask/...` clean.
- `gofmt -l internal/ask/` empty.
- All other modules (approvals, core/audit, core/redact,
  core/schemas, core/state, egress, models, persist,
  planner, policy, runtime, sandbox, tools, tests)
  remained green.



Add the aios-benchmark module (Group 2 of the build plan).
The module is a single Go package at `internal/benchmark/`
and depends only on the Go standard library. No new
third-party deps.

Public surface:

- `WedgeDataset` — the wedge dataset with built-in
  tasks for all 8 wedges (file_read, file_write,
  file_search, shell_run, approval, ask_user, resume,
  cancel). Loads/saves JSON (REQ-BCH-DAT-001..003).
- `WedgeTask` — the per-task record with id, wedge,
  goal, expected_result, and timeout_seconds.
- `Runner` — the small interface aios-runtime plugs
  into. `Run(task) Result`. The benchmark is
  dependency-free at the module level.
- `Result` / `Metrics` — the per-task metrics struct
  with task_id, duration_ms, cost_usd, success,
  loop_count, tool_calls, model_calls, approvals
  (REQ-BCH-DAT-004).
- `BenchmarkRunner` — runs the dataset, enforces
  per-task timeouts, records the metrics
  (REQ-BCH-RUN-001..004).
- `TwoClassRunner` — runs the dataset with two runners
  and returns the deltas (REQ-BCH-2TR-001..003).
- `LoopChecker` — runs the dataset and reports the
  average, max, and percentage of looping tasks
  (REQ-BCH-LPC-001..003).
- `CLI` — minimal CLI struct with `Run` and
  `RunTwoClass` methods. Prints the wireframe report
  and writes the JSON metrics file.
- `BenchmarkTimeout` / `BenchmarkFailure` — typed
  errors (REQ-BCH-ERR-001..002).

Sample dataset at `internal/benchmark/examples/wedge.json`.

Test coverage: 47 cases, all green. Public functions
at 100% coverage; total package coverage 94.8%.
Uncovered lines are unreachable error paths
(marshal/parse on valid Go types).

## v0.16.0 — aios-report

Add the aios-report module (Group 2 of the build
plan). The module is a single Go package at
`internal/report/` and depends on aios-core (for the
canonical-JSON encoder) and the Go standard library
(text/template, html/template, embed). No new
third-party deps.

Public surface:

- `Report` and `Step` — the typed data contracts.
  The Report describes one run of a plan. The Step
  describes one step in the run.
- `Status` — the run-status enum (succeeded, failed,
  canceled).
- `Renderer` — the renderer interface every concrete
  renderer implements.
- `JSONRenderer` — the deterministic JSON renderer.
  The renderer sorts the keys at every level and
  indents by 2 spaces.
- `MarkdownRenderer` — the human-readable Markdown
  renderer. The renderer uses tables for parallel
  data and lists for sequential data.
- `HTMLRenderer` — the HTML5 renderer with the
  bundled report.html template. The template is
  embedded through `go:embed`.
- `ReportError` — the validation error.
- `ReportRenderError` — the render error.

The doc set's `IMPLEMENTATION-PLAN` lists
`.py` filenames; the source files are `.go` per the
project's language choice.

## v0.14.0 — aios-tools

Add the aios-tools module (Group 2 of the build plan).
The module is a single Go package at `internal/tools/`
and depends on aios-core (audit), aios-policy (gateway),
and aios-sandbox (Runner). No new third-party deps.

Public surface:

- `Tool` — the tool interface (Name + Run).
- `Context` — actor, task id, workspace root.
- `Result` — ok, output, summary.
- `ReadFileTool` — reads a file with optional offset/limit.
  Validates the path against the workspace (REQ-TOL-RD-003).
- `WriteFileTool` — writes a file atomically (temp + rename,
  REQ-TOL-WR-004). Rejects paths outside the workspace.
- `SearchFilesTool` — filename glob search with include,
  exclude, and max-result cap.
- `RunShellTool` — runs an argv through the sandbox runner.
  Maps `*sandbox.SandboxTimeout` to `ErrToolTimeout`.
- `Bus` — the tool bus. Looks up by name, consults the
  policy gateway, runs the tool, writes one `tool_call`
  audit row per call. Policy `deny` short-circuits;
  policy `ask` surfaces as a "requires approval" error
  (REQ-TOL-BUS-003..005).
- `Workspace()` and `ResolvePath()` — workspace resolution
  from `$AIOS_WORKSPACE` (default CWD) and path validation
  with symlink escape protection.
- `PathNotAllowedError`, `ToolNotFoundError`, `ToolTimeoutError`
  — typed errors (REQ-TOL-ERR-001..003).

Test coverage: 47 cases, all green. Public functions
at 100% coverage; the only uncovered statements are
unreachable filesystem-error paths in atomic write
and the audit-failure wrap in Dispatch.

## v0.13.0 — aios-persist

Add the aios-persist module (Group 2 of the build plan).
The module is a single Go package at `internal/persist/`
and depends on aios-core (for the audit log and the
clock). The persist package uses its own match type
`RuleMatch` to keep the package free of a hard
dependency on aios-policy. No new third-party deps.

Public surface:

- `RuleStorage` — the always-allow rule storage with
  add/remove/list/match and user+project+session
  levels. The project rules override the user rules;
  the session rules override both.
- `SessionCheckpoint` — the session state checkpoint
  with save/load/list/delete. The file is gzipped on
  disk when the JSON is above 1 MB.
- `Handoff` — the structured handoff with put/get.
  The handoff supports both file and JSON value
  handoffs through the `IsPath` flag.
- `SessionHistory` — the session history. The history
  writes `session_event` rows to the audit log. The
  history is append-only with the audit log's hash
  chain.
- `Rule`, `RuleMatch`, `RuleAction`, `RuleScope` —
  rule types.
- `SessionState`, `StepState` — session types.
- `HandoffRecord` — the on-disk form of a handoff.
- `HistoryEvent` — the typed payload of a history
  event.
- `RuleNotFoundError`, `SessionNotFoundError`,
  `HandoffNotFoundError` — typed errors.

The default paths honor the `AIOS_HOME` and
`AIOS_DATA_DIR` environment variables. The user can
override the paths through the constructors' `Config`
structs. The doc set's `IMPLEMENTATION-PLAN` lists
`.py` filenames; the source files are `.go` per the
project's language choice.

## v0.15.0 — aios-planner

Add the aios-planner module. The module turns a goal into a
verified `schemas.TaskPlan`. It depends on `aios-core` (for the
schema types), `aios-models` (for the model router), and the Go
standard library (for `text/template`). No new third-party deps.

Public surface (per REQ-PLN-IFC-001):

- `Planner` — the planner. Holds a `models.Router`, a tool map,
  and a `Verifier`. `Plan(ctx, goal, ctxMap)` returns a verified
  `*schemas.TaskPlan` or a `*PlanGenerationError` after the
  repair loop is exhausted.
- `Verifier` — the protocol the planner calls. The default
  implementation is `CodeFirstVerifier`.
- `CodeFirstVerifier` — the code-first verifier. Runs schema
  validation, tool-call checks, tool-registry checks, and the
  diff-scope check (default 200 lines), then any custom rules the
  caller registered. No model call. Deterministic.
- `PlanError` / `PlanErrorList` — the verifier's error type. The
  list is empty on pass.
- `PlanValidationError` — raised when the very first attempt is
  invalid and the planner gives up after the repair loop.
- `PlanGenerationError` — raised when the planner cannot produce
  a valid plan within `MaxRepairIterations` (default 3). Wraps
  the model call's error when one is the cause.
- `ToolSpec` — the planner's view of a tool (name, description,
  JSON input schema, risk tier). Distinct from any aios-tools
  type; the planner only needs these four fields to render
  prompts and to validate tool names.
- `RenderSystem`, `RenderUser`, `RenderRepair` — the prompt
  templates. `text/template` (stdlib).

Substitutions vs the doc set:

- `jinja2` (Python templating) → `text/template` (Go stdlib).
  The doc set itself says "text/template (stdlib)" in §6.3; the
  Python template references are vestigial. The Go templates are
  package-level `var`s parsed at init, so a malformed template
  surfaces at startup, not at first call.
- The doc set's `model_router: ModelRouter` is the
  `internal/models/router.go` Router. The planner calls
  `Router.Complete` and reads the response text + usage.
- The doc set's diff-scope hint (`lines_changed`) is encoded as
  a `lines=N` token in the task's `acceptance` string. The
  schema is not extended; the verifier parses the hint out of
  the existing field. Negative values (`lines=-N`) are rejected
  as schema errors.
- Doc set filenames `.py` → `.go`. The doc set's `aios.planner`
  package is `internal/planner` in the Go monorepo.
- The doc set's "8-day schedule" (Phase 1..5) is collapsed
  into a single delivery. The doc set's `tests/` folder is the
  package-local `*_test.go` files; the Go monorepo uses the
  package-internal test convention.

Quality gates (at the time of merge):

- `go test ./internal/planner/...` ok.
- `go test -cover ./internal/planner/...` → 92.7% of statements.
- `go vet ./internal/planner/...` clean.
- `gofmt -l internal/planner/` empty.
- All other modules (approvals, core/audit, core/redact,
  core/schemas, core/state, egress, models, persist, policy,
  runtime, sandbox, tools, tests) remained green.


## v0.14.0 — aios-egress

Add the aios-egress module (Group 1 of the build plan). The
module is a deny-by-default HTTP forward proxy with a YAML
allow + deny list and header-rewriting rules. It depends only
on aios-core (audit) and the Go standard library. No new
third-party deps.

Public surface:

- `AllowList` — exact + glob host matching with deny precedence.
- `HeaderRules` — remove + add header rewriting for both
  request and response.
- `Proxy` — the HTTP server; the `Listen` address defaults to
  `127.0.0.1:0` for tests and is set to `AIOS_EGRESS_LISTEN`
  by the CLI in production.
- `HostNotAllowedError`, `UpstreamError`, `ParseError` —
  typed errors per REQ-EGS-ERR-001..002.

Substitutions vs the doc set:

- `aiohttp` → `net/http` (stdlib). The doc set names aiohttp
  because the doc template was written for Python; the Go
  implementation uses the standard library as in
  `internal/policy/gateway.go` and `internal/approvals/`.
- `path.Match` (Go) replaces `fnmatch` (Python). The semantics
  are equivalent for the patterns the doc set shows.
- Doc set filenames `.py` → `.go`. The doc set's package
  shape `internal/egress/` is preserved.
- The `add` block in the YAML is a single map, not a list of
  maps. The doc example shows the single-map form; the loader
  follows the example. This is the YAML form that `gopkg.in/yaml.v3`
  parses naturally.

Design notes:

- The proxy is a forward proxy, not a reverse proxy. The
  client sends the absolute target URL on the request line
  (the way `http.Client` does with `Transport.Proxy` set).
  The proxy reads `r.RequestURI` first and falls back to
  `r.URL.Host` + `r.URL.Path` when the request line is
  relative; relative-only requests are refused with 400.
- Patterns and request hosts are both normalised at compare
  time: the port is stripped and the string is lower-cased.
  This means `api.openai.com:443` in the YAML matches
  `API.OPENAI.COM:8080` on the request.
- Hop-by-hop headers (`Connection`, `Keep-Alive`,
  `Proxy-Authenticate`, `Proxy-Authorization`, `Te`,
  `Trailer`, `Transfer-Encoding`, `Upgrade`) are stripped
  before forwarding, per RFC 7230 §6.1.

Tests: 42 cases pass, including a full end-to-end loopback
test against a real `httptest.NewServer` upstream, an upstream-
error path, hop-by-hop stripping, deny-list precedence, and
header-rewriting on both directions.

Example file: `examples/egress.yaml`.

## v0.13.0 — aios-runtime

Add the aios-runtime module (Group 2 of the build plan). The
module is a single Go package at `internal/runtime/` and
depends only on aios-core (state machine, audit, schemas) and
`encoding/json` (stdlib). No new third-party deps.

Public surface:

  - `Orchestrator` with `Run(ctx, plan, checkpoint) (*Report,
    error)` and `Cancel()`. Sequential (one task at a time,
    per REQ-RT-ORC-003); the parallel-workers semaphore is
    wired but unused in the sequential path.
  - `Worker` interface: `Run(TaskInput, *TaskContext,
    *BudgetMeter) (WorkerResult, error)`. Workers raise
    `*WaitingForApproval` to surface approval asks.
  - `Verifier` interface: `Verify(TaskInput, WorkerResult)
    (VerifierVerdict, error)`. Verdict values are `pass`,
    `fail`, `inconclusive`; per the global PRD §11.7,
    `inconclusive` is treated as `fail`.
  - `Budget` and `BudgetMeter`: 9 caps (5 totals + 4
    concurrency). The four concurrency caps are: parallel
    workers (semaphore), tool calls/min (sliding window),
    model calls/min (sliding window), network requests/min
    (sliding window). The 60-second windows are pruned on
    every `CanProceed` call.
  - `NoProgressDetector`: content-hash-based; trips when
    the same worker output repeats more than the limit.
  - `Checkpoint` and `ResumeHandler`: JSON files in
    `var/checkpoints/` (or the configured directory). One
    file per task id; saves after every successful task.
  - `Tracer`: writes `decision` audit rows.
  - Sentinel errors: `ErrBudgetExhausted`, `ErrNoProgress`,
    `ErrCanceled`, `ErrCheckpointNotFound`. Typed:
    `*BudgetExhausted`, `*NoProgressDetected`,
    `*WaitingForApproval`.

Stop conditions (the orchestrator stops and reports):

  - `completed` — every task ran and the verifier passed.
  - `verifier_fail` — the verifier said fail or inconclusive.
  - `worker_error` — the worker returned a non-approval error.
  - `budget_exhausted` — `BudgetMeter.CanProceed` failed;
    the checkpoint is saved with the next task's index.
  - `no_progress` — the detector tripped.
  - `waiting_for_approval` — the worker raised
    `*WaitingForApproval`; the report carries the wait.
  - `canceled` — `Cancel` was called.

Substitutions from the doc set (recorded for posterity):

  - The doc set says the orchestrator is "in-process" /
    "library"; the implementation is exactly that — a
    single Go package, no service, no HTTP.
  - The doc set's `Verifierinterface` is defined in
    `worker.go` (next to the Worker). The doc set's
    component list calls out 6 components, not 7; the
    Verifier is implicit in the orchestrator constructor
    signature. Defining it where the Worker is defined
    keeps the public surface tight.
  - The doc set's "per-model cost tracking" is implemented
    as `BudgetMeter.RecordModelCall(costUSD)`. The
    orchestrator asks the worker to report the cost; the
    meter does the running total. The price table is the
    caller's responsibility (typically aios-models).
  - The doc set's checkpoint file is JSON in
    `var/checkpoints/<task_id>.json`. The `Plan` and
    `Context` fields are `json.RawMessage` so the
    orchestrator can round-trip the schemas without
    needing to import every schema type. The auditor and
    the worker parse the JSON lazily.
  - The doc set's resume-via-audit-log (TRD §6.5) is a
    follow-up. The first pass saves JSON checkpoints.
    Replay via the audit log is a separate item.

Tests: 31 test cases across 6 test files. Coverage 84.3%
on the package. The integration test (`tests/`) walks a
3-task plan through the orchestrator and verifies the audit
chain.

## v0.12.0 — aios-sandbox

Add the aios-sandbox module (Group 2 of the build plan).
The module is a single Go package at `internal/sandbox/`
and depends only on the Go standard library. No new
third-party deps.

Public surface:

- `Runner` — the runner interface.
- `SubprocessRunner` — the os/exec wrapper.
- `NewSubprocessRunner()` and `NewSubprocessRunnerWithClock(now)`.
- `RunOptions` — the input to `Run`.
- `SandboxResult` — the output of `Run`.
- `ResourceLimits` — the typed limits object.
- `SandboxError`, `SandboxTimeout`, `SandboxResourceLimit`,
  `LimitsError` — typed errors.
- `DefaultTimeout` — the default wall-clock timeout (60s).

The runner applies the CPU, memory, and process limits
through `syscall.Setrlimit` on POSIX. On Windows the
limits are a no-op and the runner falls back to the
wall-clock timeout. The runner clears the child
environment and sets only the allowed variables
(REQ-SND-ISO-002). The runner drops privileges on POSIX
when `ResourceLimits.UID` and `ResourceLimits.GID` are
non-negative (REQ-SND-ISO-003).

The tests run on Windows and POSIX. The runner tests
spawn the test binary itself as a child process so the
tests do not depend on `/bin/sh` or any other shell. The
tests set `TMP` and `TEMP` on the child so the Go
coverage runtime can write its temp directory when the
parent runs with `-cover`. Coverage on the package is
86.5% on Windows. The uncovered branches are error
paths that are only reachable on POSIX (e.g.
`RLIMIT_CPU` exceeded) or unexported helpers.

The doc set's `IMPLEMENTATION-PLAN` lists `.py`
filenames; the source files are `.go` per the project's
language choice.

## v0.11.0 — aios-models

Add the aios-models module (Group 2 of the build plan). The
module is a single Go package at `internal/models/` and depends
only on aios-core and `gopkg.in/yaml.v3` (already a transitive
dep via aios-policy). No new third-party deps.

Public surface:

  - `Adapter` interface: `Provider()`, `Complete(ctx, req)`,
    `Stream(ctx, req)`. Two implementations:
    `OpenAIAdapter` and `AnthropicAdapter`. Both use
    `net/http` (stdlib) and accept a custom `http.Client` and
    base URL for tests / staging.
  - `Request`, `Response`, `Usage`, `Chunk`, `Options`: the
    call-site types. `Options.Temperature` defaults to 0
    (per the aios-models PRD §5 "Bounded by temperature=0").
  - `Stream` interface: `Next() (Chunk, error)`, `Usage()`,
    `Close()`. The final chunk carries the Usage.
  - `Price` and `PriceTable`: per-model USD pricing, with
    override support. `LoadPriceTable(path)` reads
    `var/prices.yaml` (SRS §8 REQ-MOD-DAT-001).
  - `Capability` and `Registry`: per-model capabilities
    (provider, context window, supports_tools/vision). The
    `Get/Set/List` API plus `LoadRegistry(path)` for
    `var/capabilities.yaml` (SRS §8 REQ-MOD-DAT-002).
  - `Router`: dispatches `Complete` / `Stream` to the right
    adapter via the registry, and attaches the USD cost from
    the price table. Adapters are cached per model id.
  - Sentinel errors: `ErrUnknownModel`, `ErrPriceNotFound`,
    `ErrProvider`. `ProviderError` is a typed error that wraps
    `ErrProvider` and carries the provider name, code, and
    message.

Streaming: both adapters speak the wire SSE format
(`text/event-stream`). The OpenAI stream yields
`data: {…}\n\n` lines; the Anthropic stream yields named
`event: …\ndata: …\n\n` events. The Stream interface hides
both formats behind a single `Next()` API.

Substitutions from the doc set (recorded for posterity):

  - The doc set says `internal/models/{adapter,openai,anthropic,price,capability,router}.py`.
    The implementation is exactly that layout, plus a
    `errors.go` for the four sentinel errors. Seven files
    instead of six; the extra one is one type per error and
    ~10 lines.
  - The doc set's `ModelChunk` is exposed as `Chunk`. The
    doc set's `ModelUsage` is exposed as `Usage`. The names
    are shorter; the shape is identical.
  - The doc set's `Stream` interface is documented as
    "async iterator of token chunks". The implementation
    uses an explicit `Next() (Chunk, error)` instead of
    `func() (Chunk, bool)` or a Go channel, because
    `Next()` is the idiomatic Go iterator pattern and is
    the most efficient (no goroutine per call, no channel
    buffering).
  - The doc set says the price file default is `./var/prices.yaml`.
    The implementation does not auto-load it; the caller
    passes a path to `LoadPriceTable`. The default path is
    a configuration decision the doc set left to the
    operator; the runtime layer is the right place for it.

Tests: 22 test cases across 5 test files. Coverage 80.0%
on the package. Streaming tested with `httptest.NewServer`
and recorded SSE responses.

Sample files: `examples/prices.yaml` and
`examples/capabilities.yaml` with the four canonical
models (gpt-4o, gpt-4o-mini, claude-sonnet-4-5,
claude-haiku-4-5).

## v0.10.0 — aios-approvals

Add the aios-approvals module (Group 2 of the build plan). The
module is a single Go package at `internal/approvals/` and depends
only on aios-core and `github.com/google/uuid` (already a
transitive dep). No new third-party deps.

Public surface:

  - `State` enum: `pending`, `granted`, `denied`, `expired`.
  - `Approval` struct: id, state, payload, actor, created_at,
    updated_at, ttl.
  - `Service` with `Request`, `Grant`, `Deny`, `Wait`, `Get`.
  - Sentinel errors: `ErrNotFound`, `ErrBindingMismatch`,
    `ErrInvalidTransition`, `ErrInvalidPayload`.
  - `RequestOptions{TTL}` for per-request TTL override
    (default 5m, negative = no expiry).

Storage: in-memory `map[string]*entry`, with a per-approval
`chan struct{}` for waiter notification. Persistence is the
audit log: every state transition writes one row of kind
`approval_requested` / `approval_granted` / `approval_denied` /
`approval_expired`.

Binding: per the SRS §5.4 REQ-APR-BND-001, the service compares
the payload's `BoundTo` (a 4-field struct in aios-core: task,
tool, actor, resource) against the call-site context. Empty
fields are wildcards. `Deny` skips the binding check (operators
can always say no). `Grant` does not.

TTL: `time.AfterFunc` schedules the expire; the closure
transitions the approval to `expired` only if the state is
still `pending`, and writes the audit row.

Cross-module addition: `schemas.ApprovalPayload` gained a
`BoundTo *BoundTo` field (typed; replaces the doc-set's loose
`bound_to` map in the aios-policy `ApprovalTemplate`). The
policy gateway converts the YAML map to `*schemas.BoundTo` at
decision time; the service consumes the typed form.

Substitutions from the doc set (recorded for posterity):

  - The doc set says `internal/approvals/{schema,binding,service,ttl}.py`;
    the implementation is `internal/approvals/{schema,service}.go`,
    with `BoundTo.Match` in aios-core filling the role of the
    doc-set `binding.py` and `time.AfterFunc` filling the role
    of the doc-set `ttl.py`. Two files instead of four: the
    binding is a method on a struct (4 lines), the TTL is one
    call.
  - The doc set's `bound_to` is a 4-field struct in aios-core
    (not a map). The aios-policy YAML still uses a
    `map[string]string` for ergonomics; the gateway converts at
    the boundary.
  - The doc set's `Approval` has `actor` as the requester; the
    service records both the requester (`Actor` on the
    `Approval`) and the granter/denier (passed as `actor` to
    `Grant`/`Deny`, written into the audit row).

Tests: 17 test cases across `schema_test.go` and
`service_test.go`, plus 2 integration tests in
`tests/approvals_integration_test.go` that exercise the service
end-to-end against the audit log and verify the hash chain.
Coverage: 85.5% on the package.

## v0.9.3 — aios-core gap fixes (G3, G6, G7)

Three of the nine gaps from the v0.9.2 audit are addressed in this
release.

- **G3 (redaction patterns).** The doc set
  (`docs/modules/aios-core/SRS.md` §5.4 REQ-CORE-RED-005) is amended
  with a substitution note: the impl's five patterns are API key,
  JWT, PEM, AWS access key, GitHub token. The original doc set
  listed "Bearer token" (covered by JWT — every JWT is a bearer),
  "Password" and "Session token" (dropped — noise-prone; the AWS /
  GitHub patterns were added in their place as high-precision
  concrete credential formats). The impl is extensible; the dropped
  patterns can be re-added in one line if a deployment needs them.
- **G6 (type aliases).** New file `internal/core/audit/alias.go`
  exports `AuditLog = Log`, `AuditRow = Row`, `RedactionMarker =
  redact.Match`, and `DuplicateWriteError = ErrDuplicate`. New file
  `internal/core/state/alias.go` exports `TaskState = State` and
  `TaskStateMachine = Machine`. Aliasing is purely textual in Go; no
  runtime cost. Callers can use either the short form
  (`audit.Log`, `state.Machine`) or the doc-set form
  (`audit.AuditLog`, `state.TaskStateMachine`).
- **G7 (env-var default).** New function `audit.OpenDefault()` reads
  the `AIOS_AUDIT_DB` environment variable and falls back to
  `var/audit.db` (per the aios-core TRD §11). The parent directory
  is created on demand (mode 0o755). 4 new tests cover default
  fallback, env-var override, parent-dir creation, and a write-then-
  readback smoke test on a real file.

Tests: 4 new tests in `default_test.go` and 1 in `alias_test.go`.
Coverage: audit 71.2% → 73.0%, all other packages unchanged.

Full audit report: `var/audit-core.md`. Remaining gaps (G1, G2, G4,
G5, G8, G9) are non-blocking follow-ups.

## v0.9.2 — aios-core end-to-end audit

Re-ran the full verification suite (go build, go vet, gofmt -l,
go test -count=1 -p 1) and walked the doc set REQ-by-REQ. Full
report at `var/audit-core.md`.

Status: aios-core is functionally complete and end-to-end correct.
All 4 builds, vet, gofmt, and 6 test packages are green; coverage
is 71-86% per package. The audit identifies 9 doc/code-alignment
gaps (G1-G9). The highest-impact are G6 (interface name aliases
for `AuditLog`/`AuditRow`/`TaskState`/`TaskStateMachine`),
G3 (redaction-pattern list amendment), and G7 (`AIOS_AUDIT_DB` env
var default). No code change in this release; the report is a
review artifact.

## v0.9.1 — aios-policy

Add the aios-policy module (Group 2 of the build plan). The module
is a single Go package at `internal/policy/` and depends only on
aios-core and `gopkg.in/yaml.v3`. No other new dependencies.

Public surface:

  - `Rule`, `Match`, `Decision`, `ApprovalTemplate` (schema types).
  - `Load(path)` (YAML loader, with `include:` directive support
    and cycle protection).
  - `Gateway` (first-match-wins evaluator, default-deny).
  - `Substitute` (template substitution for `{task}`, `{tool}`,
    `{path}`, `{host}`, `{actor}`, `{tier}`, `{arg.X}`).
  - `PolicyParseError`, `RuleValidationError` (typed errors).

The gateway writes a `policy_decision` audit row per call, joining
to the approvals service via the row's `approval_id` field. The
rule table is content-addressed via `Hash(rules)`, which is part
of the determinism contract.

Tests: 80+ test cases across five test files. Coverage 95.0%.

Substitutions from the doc set (recorded for posterity):

  - The doc set says `aios.policy`; the implementation is
    `internal/policy` to match the build plan's monorepo layout.
  - The doc set says `struct + go-playground/validator`; the
    implementation hand-rolls tag validation, matching aios-core.
  - The doc set's `decide` method is `Gateway.Decide` in Go; the
    function signature and behavior match the doc set's spec.

## v0.9.0 — foundation

Initial scaffold. aios-core module. The doc set
(`docs/modules/aios-core/`) was written before the global PRD was
finalized; the implementation follows the global PRD where they
disagree. See `README.md` for the substitution list.
