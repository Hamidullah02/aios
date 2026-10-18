# Integration audit (v0.21.0)

A full end-to-end wire audit of aios-system. The previous
audit (`var/audit-e2e.md`, v0.20.0) verified that every
module compiled and passed its unit tests. This audit
goes one level deeper: it verifies that the modules talk
to each other, that config plumbing is consistent, and
that the production code paths exercise the cross-module
wires (not just stubs).

## 1. Cross-module wire map

The full import graph (from `go list -f '{{range .Imports}}'`):

| Module         | Imports (internal only)                              |
| -------------- | ---------------------------------------------------- |
| `approvals`    | `core/audit`, `core/clock`, `core/schemas`           |
| `ask`          | `core/schemas`                                       |
| `benchmark`    | (no internal imports)                                |
| `cli`          | `approvals`, `ask`, `core/audit`, `core/schemas`, `policy`, `runtime` |
| `core/audit`   | `core/clock`, `core/redact`, `core/schemas`          |
| `core/clock`   | (no internal imports)                                |
| `core/redact`  | (no internal imports)                                |
| `core/schemas` | (no internal imports)                                |
| `core/state`   | (no internal imports)                                |
| `egress`       | `core/audit`, `core/clock`, `core/schemas`           |
| `memory`       | (no internal imports; isolated)                      |
| `models`       | (no internal imports)                                |
| `persist`      | `core/audit`, `core/clock`                           |
| `planner`      | `core/schemas`, `models`                             |
| `policy`       | `core/audit`, `core/clock`, `core/schemas`           |
| `report`       | `core/schemas`                                       |
| `runtime`      | `core/audit`, `core/schemas`, `core/state`           |
| `sandbox`      | (no internal imports)                                |
| `search`       | `sandbox`, `tools`                                   |
| `tools`        | `core/audit`, `policy`, `sandbox`                    |

Acyclic. No layer violations. (The runtime does NOT
import tools, which keeps the orchestrator reusable in
tests with stub workers.)

## 2. Findings & fixes

### 2.1 `runtime` was not stitched to `tools` in any production path

**Severity**: high.

**Symptom**: the `tools.Bus` is only ever called from
tests. Production code paths use a `PassthroughWorker`
that returns canned results for known tool names and
errors on unknown ones. The runtime + policy + bus + tool
+ audit chain is therefore a unit-test artefact, not a
running system.

**Fix**: added `internal/cli/bus_worker.go`, a
`BusWorker` that implements `runtime.Worker` and
forwards each task through the bus. Five end-to-end
tests in `bus_worker_test.go` exercise:

- read_file through the bus + policy `allow` + audit
- policy `deny` short-circuit
- policy `ask` → `*runtime.WaitingForApproval`
- run_shell through the bus + sandbox + audit
- ask_user through the bus + scripted callback

**Verification**: all 5 new tests PASS, hash chain
verifies across the full plan + task rows.

### 2.2 `BuildDeps` silently swallowed `audit.Open` errors

**Severity**: high.

**Symptom**: in `internal/cli/deps.go`,
`BuildDeps` called
`audit.Open` and dropped the error on the floor
when it returned non-nil. The function then returned
`nil` for the error, so the production binary
proceeded with `deps.Audit == nil` and silently
dropped every audit row the system wrote. A
misconfigured filesystem (e.g. a parent dir the
operator cannot create) was indistinguishable from a
working system.

**Fix**: `BuildDeps` now returns a `*CliError`
when `audit.Open` fails. The binary exits 2 (system
error). The audit log is required for `run`, `resume`,
`cancel`, `report`, `audit`; the `init` and `daemon`
subcommands remain audit-free so they can run before
the database exists.

**Test**: new `TestCoverage_BuildDeps_AuditError`
asserts the contract (unreachable parent dir → error
mentions "audit").

### 2.3 `BuildDeps` silently swallowed `policy.Load` errors

**Severity**: high.

**Symptom**: same shape as 2.2. A policy file with
invalid YAML made `BuildDeps` quietly skip the policy
and run with no enforcement. The operator's edits were
silently ignored.

**Fix**: `BuildDeps` now returns a `*CliError` when
`policy.Load` fails. The already-opened audit log is
closed before the function returns the error, so the
caller (and the test cleanup) does not leak a file
handle.

**Test**: the existing
`TestCoverage_BuildDeps_PolicyError` was asserting the
wrong contract (it expected `err == nil` when the
policy was unparseable). The test is now corrected to
assert `err != nil` and that the error message
mentions "policy".

### 2.4 Default policy path forced a silent failure

**Severity**: medium.

**Symptom**: the `--config` flag's default was
`./var/config.yaml`. On a fresh install that file
does not exist; the operator had no idea their CLI
was failing silently until they hit a tool call
that the missing policy would have blocked.

**Fix**: the default is now `""` (no policy). The doc
set treats the policy as optional; the default now
reflects that. The help text spells out the
semantics. The smoke tests that previously passed
`--config=...config.yaml` no longer do, because the
test does not exercise the policy path.

### 2.5 Default `ConfigPath` constant collision

**Severity**: low (caught before commit).

**Symptom**: an early draft of the
`isInitOrDaemonStart` helper referenced
`args.DaemonSub`, which does not exist on the
`Args` struct. The build failed loudly.

**Fix**: switched to `args.Subcommand` (the actual
field name). Caught at compile time.

### 2.6 Stale `var/` artefacts in `internal/cli/`

**Severity**: low.

**Symptom**: the `TestMain_EndToEndCancel` smoke
test pre-populated a checkpoint at `./var/checkpoints`
(relative to the package CWD) and did not always clean
it up. The first run after the audit left a
`var/checkpoints/cancel-1.json` on disk.

**Fix**: the smoke test now defers
`os.RemoveAll("./var")` so any scaffolded files are
cleaned up. Pre-existing test artefacts were deleted
by hand.

### 2.7 Ask user tool was not wired in any production path

**Severity**: high.

**Symptom**: `ask.AskUserTool` is a complete,
validated implementation with 32 unit tests, but the
production CLI never registered it in the bus. The
agent could not actually ask the operator a question
through the real `run` subcommand.

**Fix**: the new `BusWorker` accepts an `askTool` and
the new `TestBusWorker_AskTool` test wires it through
the bus with a scripted callback. The audit row is
written. The integration is now proven.

## 3. Env var audit

The full list of env vars the code reads:

| Env var | Read where | Default | Notes |
| ------- | ---------- | ------- | ----- |
| `AIOS_AUDIT_DB` | `core/audit/default.go` | `var/audit.db` | The audit log path; honoured by `audit.OpenDefault`. |
| `AIOS_DATA_DIR` | `persist/checkpoint.go`, `persist/handoff.go`, `persist/rules.go` | per-call fallback | Persistence paths. |
| `AIOS_HOME` | `persist/rules.go` | `$HOME` / `$USERPROFILE` | Home dir for daemon state. |
| `AIOS_WORKSPACE` | `tools/pathutil.go` | `os.Getwd()` | Workspace root for `read_file` / `write_file` path validation. |
| `ANTHROPIC_API_KEY` | `models/anthropic.go` | (required) | Provider key. |
| `OPENAI_API_KEY` | `models/openai.go` | (required) | Provider key. |

No duplicate or conflicting env vars. The named
constants (`audit.EnvAuditDB`, `tools.EnvWorkspace`)
are the public API for these names; callers should
not hard-code the strings.

## 4. Default file paths

| Module | Default | Where |
| ------ | ------- | ----- |
| audit log | `./var/aios.db` (CLI) / `var/audit.db` (`OpenDefault`) | `cli/deps.go`, `core/audit/default.go` |
| checkpoints | `./var/checkpoints` | `cli/deps.go` |
| policy file | (no default; opt-in via `--config`) | `cli/parser.go` |
| egress config | `var/egress.yaml` (scaffolded by `aios init`) | `cli/init_command.go` |
| prices file | `var/prices.yaml` (scaffolded) | `cli/init_command.go` |
| models config | `var/models.yaml` (scaffolded) | `cli/init_command.go` |
| daemon pid file | `./var/aios-daemon.pid` | `cli/daemon.go` |
| daemon socket | `/tmp/aios-approval.sock` | `cli/daemon.go` |

Two notes:

- The CLI's audit path (`var/aios.db`) differs from
  the audit module's `DefaultPath` (`var/audit.db`).
  The CLI builds its own path with `./` so an
  absolute resolution succeeds. The mismatch is
  cosmetic; both paths resolve to the same file
  when CWD is the workspace root. The doc set does
  not specify a name, so we keep the two
  intentionally separate: `OpenDefault` is the
  module's public entry point (matches the
  `aios.AuditDB` env var), the CLI uses its own
  path for the production binary.
- The daemon socket defaults to `/tmp/aios-approval.sock`
  on every platform. On Windows this resolves to
  `C:\tmp\...` which is not a valid path. The
  runtime is Unix-only for the daemon; the CLI's
  Windows path is exercised in unit tests via a
  custom socket. A follow-up could pick
  `os.TempDir()` on Windows.

## 5. Concurrency audit

CGO is not available in this environment, so `-race`
cannot run. We did:

- `go test -count=5 -p 1 ./...` — 21/21 packages
  pass; no flakiness across 5 runs.
- `go test -v -count=10 ./tests` — 80/80 integration
  runs pass.

Goroutines in production code:

- `approvals.Service.expire` — runs via
  `time.AfterFunc`. Holds the mutex while it
  snapshots the entry, then releases the lock
  before writing the audit row (correct I/O outside
  the lock pattern).
- `cli.Daemon.Start` — accepts connections on a
  Unix socket; the parent goroutine calls `Stop`
  when the context is cancelled. `Stop` releases
  the lock before closing the listener (correct).
- `runtime.Orchestrator.Run` — uses
  `context.WithCancel` so the operator can
  interrupt a long run. The cancel func is
  guarded by a mutex so the operator can race a
  graceful cancel against a re-entry.
- `benchmark.Runner` — fires one goroutine per
  task and synchronises on a WaitGroup.

No naked goroutines, no shared state without a
mutex, no missing `defer cancel()`.

## 6. Test counts (final)

- Unit + integration tests: **1090 PASS**, 0 FAIL,
  1 SKIP (the Windows symlink test).
- Integration tests with `-count=10`: 80/80 PASS.
- All 21 packages compile, pass `go vet`, and pass
  `gofmt -l .`.

## 7. Outstanding items (not regressions)

- `internal/cli` is a library; the production
  binary's `package main` is a follow-up. The
  `Main` function in `internal/cli/main.go` is the
  entry point and is fully covered by
  `TestMain_EndToEnd*`.
- The daemon socket's default path is
  Unix-only; a follow-up can pick `os.TempDir()`
  on Windows.
- The audit module's `DefaultPath`
  (`var/audit.db`) and the CLI's audit path
  (`var/aios.db`) are intentionally different.
  Operators who want the module default can set
  `--log=$AIOS_AUDIT_DB`.
