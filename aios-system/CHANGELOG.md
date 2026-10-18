# Changelog

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

#