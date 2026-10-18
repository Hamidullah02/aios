# Changelog

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
