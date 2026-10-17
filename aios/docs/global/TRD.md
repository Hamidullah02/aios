# aios — Global Technical Requirements Document

## 1. Purpose

This document states the technical approach for the
aios project at the cross-module level. The per-module
TRD documents are the source of truth for the module
detail. This document is the source of truth for the
cross-module architecture, the data flow, and the
deployment.

## 2. Scope

The aios project is in scope for this document.

- The four-layer architecture.
- The module dependency graph.
- The data flow across layers.
- The deployment topology.
- The shared data structures.

## 3. Reference Documents

- Global PRD. See `docs/global/PRD.md`.
- Global SRS. See `docs/global/SRS.md`.
- `DESIGN.md` at the project root. The document is
  the source of truth for the architecture and the
  competitive research.
- The per-module TRD under
  `docs/modules/<module>/TRD.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Layer | A horizontal band of modules. |
| Foundation | The lowest layer. The layer provides the cross-cutting concerns. |
| Capability | The middle layer. The layer provides the tools and the policies. |
| Agentic | The top layer. The layer provides the planner, the orchestrator, and the benchmark. |
| Operator | The top of the stack. The layer provides the CLI and the approval daemon. |

## 5. Four-Layer Architecture

The aios project has four layers. The layers are
listed from the lowest to the highest.

### 5.1 Foundation Layer

The foundation layer provides the cross-cutting
concerns. The concerns are the typed schemas, the
audit log, the state machine, the redaction layer,
the policy gateway, the sandbox runner, the egress
proxy, the model adapter layer, the model router
with cost optimization, the approval service, and
the session persistence layer.

The modules in this layer are listed below.

- `github.com/<org>/aios/internal/core`
- `github.com/<org>/aios/internal/policy`
- `github.com/<org>/aios/internal/sandbox`
- `github.com/<org>/aios/internal/egress`
- `github.com/<org>/aios/internal/models`
- `github.com/<org>/aios/internal/optimizer`
- `github.com/<org>/aios/internal/approvals`
- `github.com/<org>/aios/internal/persist`

### 5.2 Capability Layer

The capability layer provides the tools and the
policies that the agent uses to interact with the
world. The layer is the most populous layer. The
layer includes the tool bus, the built-in tools, the
search and read tools, the edit tools, the git
integration, the in-session memory, the file
watcher, the parallel sub-agent runner, the retry
policy, the verify layer, the role registry, the
image tool, the ask-user tool, and the report
generator.

The modules in this layer are listed below.

- `github.com/<org>/aios/internal/tools`
- `github.com/<org>/aios/internal/search`
- `github.com/<org>/aios/internal/edit`
- `github.com/<org>/aios/internal/git`
- `github.com/<org>/aios/internal/memory`
- `github.com/<org>/aios/internal/watch`
- `github.com/<org>/aios/internal/fanout`
- `github.com/<org>/aios/internal/retry`
- `github.com/<org>/aios/internal/verify`
- `github.com/<org>/aios/internal/roles`
- `github.com/<org>/aios/internal/image`
- `github.com/<org>/aios/internal/ask`
- `github.com/<org>/aios/internal/report`

### 5.3 Agentic Layer

The agentic layer provides the planner, the
orchestrator, and the benchmark harness. The layer
is where the agent's intelligence lives.

The modules in this layer are listed below.

- `github.com/<org>/aios/internal/planner`
- `github.com/<org>/aios/internal/runtime`
- `github.com/<org>/aios/internal/benchmark`

### 5.4 Operator Layer

The operator layer provides the CLI and the approval
daemon. The layer is the entry point for the user.

The modules in this layer are listed below.

- `github.com/<org>/aios/internal/cli`

## 4.5 Go Stack

The project is implemented in Go 1.23. The stack
is listed below.

- The language is Go 1.23.
- The module path is `github.com/<org>/aios`.
- The internal packages are under
  `internal/<module>/`.
- The CLI entry point is `cmd/aios/main.go`.
- The CLI is built on
  `github.com/spf13/cobra`.
- The TUI is built on
  `github.com/charmbracelet/bubbletea` and
  `github.com/charmbracelet/lipgloss`.
- The config files use `gopkg.in/yaml.v3`.
- The persistence layer uses
  `modernc.org/sqlite` (pure Go, no CGo).
- The HTTP client uses `net/http` (stdlib).
- The file watcher uses
  `github.com/fsnotify/fsnotify`.
- The JSON serialization uses `encoding/json`
  (stdlib).
- The structured logging uses `log/slog` (stdlib).
- The templates use `text/template` (stdlib).
- The image decoding uses `image/jpeg`,
  `image/png`, `image/gif` (stdlib).
- The JSON Schema validation uses
  `github.com/santhosh-tekuri/github.com/santhosh-tekuri/jsonschema/v5/v5`.
- The UUIDs use `github.com/google/uuid`.
- The test framework is the stdlib `testing`
  package.
- The assertions use
  `github.com/stretchr/testify`.

## 6. Module Dependency Graph

The module dependency graph is a directed acyclic
graph. The graph flows from the foundation layer to
the operator layer. The graph has no cycles.

The high-level edges are listed below.

- `github.com/<org>/aios/internal/core` has no dependencies on other modules.
- `github.com/<org>/aios/internal/policy` depends on `github.com/<org>/aios/internal/core`.
- `github.com/<org>/aios/internal/sandbox` depends on `github.com/<org>/aios/internal/core`.
- `github.com/<org>/aios/internal/egress` depends on `github.com/<org>/aios/internal/core` and
  `github.com/<org>/aios/internal/policy`.
- `github.com/<org>/aios/internal/models` depends on `github.com/<org>/aios/internal/core`.
- `github.com/<org>/aios/internal/optimizer` depends on `github.com/<org>/aios/internal/core` and
  `github.com/<org>/aios/internal/models`.
- `github.com/<org>/aios/internal/approvals` depends on `github.com/<org>/aios/internal/core` and
  `github.com/<org>/aios/internal/persist`.
- `github.com/<org>/aios/internal/persist` depends on `github.com/<org>/aios/internal/core` and
  `github.com/<org>/aios/internal/policy`.
- `github.com/<org>/aios/internal/tools` depends on `github.com/<org>/aios/internal/core`,
  `github.com/<org>/aios/internal/sandbox`, and `github.com/<org>/aios/internal/policy`.
- `github.com/<org>/aios/internal/search` depends on `github.com/<org>/aios/internal/core` and
  `github.com/<org>/aios/internal/tools`.
- `github.com/<org>/aios/internal/edit` depends on `github.com/<org>/aios/internal/core` and
  `github.com/<org>/aios/internal/tools`.
- `github.com/<org>/aios/internal/git` depends on `github.com/<org>/aios/internal/core` and
  `github.com/<org>/aios/internal/tools`.
- `github.com/<org>/aios/internal/memory` depends on `github.com/<org>/aios/internal/core`.
- `github.com/<org>/aios/internal/watch` depends on `github.com/<org>/aios/internal/core`.
- `github.com/<org>/aios/internal/fanout` depends on `github.com/<org>/aios/internal/core`,
  `github.com/<org>/aios/internal/models`, and `github.com/<org>/aios/internal/roles`.
- `github.com/<org>/aios/internal/retry` depends on `github.com/<org>/aios/internal/core` and
  `github.com/<org>/aios/internal/tools`.
- `github.com/<org>/aios/internal/verify` depends on `github.com/<org>/aios/internal/core`,
  `github.com/<org>/aios/internal/tools`, `github.com/<org>/aios/internal/models`, and `github.com/<org>/aios/internal/sandbox`.
- `github.com/<org>/aios/internal/roles` depends on `github.com/<org>/aios/internal/core` and
  `github.com/<org>/aios/internal/models`.
- `github.com/<org>/aios/internal/image` depends on `github.com/<org>/aios/internal/core`, `github.com/<org>/aios/internal/tools`,
  and `github.com/<org>/aios/internal/models`.
- `github.com/<org>/aios/internal/ask` depends on `github.com/<org>/aios/internal/core`.
- `github.com/<org>/aios/internal/report` depends on `github.com/<org>/aios/internal/core`.
- `github.com/<org>/aios/internal/planner` depends on `github.com/<org>/aios/internal/core`,
  `github.com/<org>/aios/internal/models`, `github.com/<org>/aios/internal/roles`, and `github.com/<org>/aios/internal/verify`.
- `github.com/<org>/aios/internal/runtime` depends on `github.com/<org>/aios/internal/core`,
  `github.com/<org>/aios/internal/tools`, `github.com/<org>/aios/internal/models`, `github.com/<org>/aios/internal/policy`,
  `github.com/<org>/aios/internal/approvals`, `github.com/<org>/aios/internal/retry`, `github.com/<org>/aios/internal/verify`,
  `github.com/<org>/aios/internal/planner`, `github.com/<org>/aios/internal/persist`, `github.com/<org>/aios/internal/report`,
  and `github.com/<org>/aios/internal/watch`.
- `github.com/<org>/aios/internal/benchmark` depends on `github.com/<org>/aios/internal/core`,
  `github.com/<org>/aios/internal/runtime`, and `github.com/<org>/aios/internal/report`.
- `github.com/<org>/aios/internal/cli` depends on every module.

The merge order follows the dependency graph. The
foundation layer merges first. The capability layer
merges next. The agentic layer merges next. The
operator layer merges last.

## 7. Cross-Module Data Flow

The cross-module data flow has the following shape.

1. The user types a goal at the CLI. The CLI is in
   `github.com/<org>/aios/internal/cli`.
2. The CLI passes the goal to the planner. The
   planner is in `github.com/<org>/aios/internal/planner`.
3. The planner turns the goal into a typed task
   plan. The plan is a `TaskPlan` object.
4. The CLI passes the plan to the orchestrator. The
   orchestrator is in `github.com/<org>/aios/internal/runtime`.
5. The orchestrator dispatches the first task. The
   task is a `Task` object.
6. The orchestrator calls the role resolver. The
   resolver is in `github.com/<org>/aios/internal/roles`. The resolver
   returns the role configuration.
7. The orchestrator calls the model router. The
   router is in `github.com/<org>/aios/internal/optimizer`. The router picks
   the model tier. The router returns the model id.
8. The orchestrator calls the model adapter. The
   adapter is in `github.com/<org>/aios/internal/models`. The adapter sends
   the request. The adapter returns the response.
9. The orchestrator calls the tool bus. The bus is
   in `github.com/<org>/aios/internal/tools`. The bus routes the tool call
   to the tool.
10. The tool bus calls the policy gateway. The
    gateway is in `github.com/<org>/aios/internal/policy`. The gateway
    returns the policy decision.
11. When the policy decision is `ask`, the bus
    routes to the approval service. The service is
    in `github.com/<org>/aios/internal/approvals`. The service blocks until
    the operator responds.
12. When the policy decision is `allow`, the bus
    runs the tool. The tool returns the result.
13. When the tool fails, the bus routes to the
    retry policy. The policy is in `github.com/<org>/aios/internal/retry`.
    The policy returns the retry decision.
14. The orchestrator calls the self-check. The
    check is in `github.com/<org>/aios/internal/verify`. The check returns
    the verification result.
15. The orchestrator writes the audit row. The row
    is in `github.com/<org>/aios/internal/core`. The row includes the
    timestamp, the actor, the action, the result,
    and the hash chain.
16. The orchestrator publishes the artifact to the
    handoff store. The store is in `github.com/<org>/aios/internal/persist`.
    The next task reads the artifact.
17. The orchestrator calls the next task. The
    cycle continues until the plan is complete.
18. The orchestrator writes the report. The report
    is in `github.com/<org>/aios/internal/report`.

## 8. Determinism Architecture

The aios project is built so that every part of
the system that is not the model, the clock, the
network, or the operator is a pure function of
its inputs. The cross-cutting mechanism is
described in this section.

### 8.1 The Non-Determinism Budget

The system has exactly four sources of
non-determinism. Every other part of the system
is deterministic by construction. The four sources
are listed below.

- The model output. The model is a stochastic
  sampler. The non-determinism is bounded by
  `temperature=0`. The call fingerprint and the
  cost/quality table handle the residual variance.
- The wall clock. The clock is non-deterministic
  for ordering and for timeouts. The clock is
  injected through a typed interface. The audit
  log records every clock read.
- The network. The network is non-deterministic
  for latency and for failure. The retry policy
  and the policy gateway handle the variance.
- The operator. The operator is non-deterministic
  for approvals. The auto-allow rules and the
  approval templates handle the variance.

### 8.2 Content-Addressed Values

The system uses content addressing for the values
that must be stable across runs. The values are
listed below.

- The plan is content addressed. The plan id is
  the SHA-256 of the plan payload.
- The policy is content addressed. The policy id
  is the SHA-256 of the policy payload.
- The checkpoint is content addressed. The
  checkpoint id is the SHA-256 of the checkpoint
  payload.
- The role is content addressed. The role id is
  the SHA-256 of the role payload.
- The audit row is content addressed. The row
  hash is the SHA-256 of the row payload.

### 8.3 The Clock Interface

The clock is a typed interface. The interface is
listed below.

```go
// Clock returns the current time.
type Clock interface {
    Now() time.Time
}

// wallClock is the production implementation.
type wallClock struct{}

func (wallClock) Now() time.Time { return time.Now() }

// fixedClock is the test implementation.
type fixedClock struct{ t time.Time }

func (f fixedClock) Now() time.Time { return f.t }
```

Every module that reads the clock accepts a
`Clock` argument. The default constructor
wires in `wallClock`. The test constructor wires
in `fixedClock` with a fixed timestamp.

### 8.4 The Random Interface

The random source is a typed interface. The
interface is listed below.

```go
// Random returns a seeded random source.
type Random interface {
    Intn(n int) int
    Float64() float64
}
```

Every module that uses randomness accepts a
`Random` argument. The default constructor
wires in a source seeded from the run id. The
test constructor wires in a source with a fixed
seed.

### 8.5 The Replay Engine

The replay engine re-executes a plan from a
checkpoint. The replay engine is a function that
takes a checkpoint id and a model adapter. The
function returns the new audit log.

The replay engine is a pure function. The
function is deterministic for every input that
does not involve the model. The function
re-executes the plan in the same order. The
function produces the same audit log entries
for the same tool calls.

The replay engine is the unit of determinism.
The user can replay any run by passing the
checkpoint id. The user can diff two runs by
passing two checkpoint ids.

## 9. Shared Data Structures

The shared data structures are the types that flow
between modules. The types are listed below.

### 8.1 TaskPlan

The `TaskPlan` is the output of the planner. The
type is in `github.com/<org>/aios/internal/planner`. The type includes the
list of `Task` objects, the `TaskPlan` id, the
goal, the budget, and the metadata.

### 8.2 Task

The `Task` is the unit of work. The type is in
`github.com/<org>/aios/internal/core`. The type includes the task id, the
description, the role, the tools, the input, the
expected output, the budget, and the status.

### 8.3 ToolCall

The `ToolCall` is the call to a tool. The type is in
`github.com/<org>/aios/internal/core`. The type includes the tool name, the
arguments, and the context.

### 8.4 ToolResult

The `ToolResult` is the result of a tool call. The
type is in `github.com/<org>/aios/internal/core`. The type includes the
status, the output, the error, and the duration.

### 8.5 AuditRow

The `AuditRow` is the row in the audit log. The
type is in `github.com/<org>/aios/internal/core`. The type includes the
fields listed in REQ-GLB-AUD-002.

### 8.6 PolicyDecision

The `PolicyDecision` is the output of the policy
gateway. The type is in `github.com/<org>/aios/internal/policy`. The type
includes the action, the reason, and the metadata.

### 8.7 RetryDecision

The `RetryDecision` is the output of the retry
policy. The type is in `github.com/<org>/aios/internal/retry`. The type
includes the action, the backoff, the fix prompt,
and the reason.

### 8.8 VerificationResult

The `VerificationResult` is the output of the
self-check. The type is in `github.com/<org>/aios/internal/verify`. The type
includes the status, the reason, and the
suggestions.

### 8.9 Handoff

The `Handoff` is the artifact passed between
tasks. The type is in `github.com/<org>/aios/internal/persist`. The type
includes the task id, the key, the value, and the
`is_path` flag.

### 8.10 Role

The `Role` is the agent configuration. The type
is in `github.com/<org>/aios/internal/roles`. The type includes the system
prompt, the tools, the model tier, the output
format, the temperature, the max tokens, and the
stop sequences.

## 10. Deployment Topology

The aios project is currently in the design phase.
The current layout is the documentation set. The
build phase will introduce the source tree, the
test tree, and the configuration tree. The target
layout is listed below.

The current layout is listed below.

```
aiagent/
  DESIGN.md
aios/
  AUDIT.md
  .gitignore
  docs/
    INDEX.md
    README.md
    global/
      PRD.md
      SRS.md
      TRD.md
      APP-FLOW.md
      WIREFRAMES.md
      IMPLEMENTATION-PLAN.md
    standards/
      STE-STYLE-GUIDE.md
      CODING-STANDARD.md
    modules/
      TEMPLATE.md
      <module_name>/
        PRD.md
        SRS.md
        TRD.md
        APP-FLOW.md
        WIREFRAMES.md
        IMPLEMENTATION-PLAN.md
```

The target layout after the build phase is listed
below.

```
aios/
  go.mod
  go.sum
  README.md
  AUDIT.md
  cmd/
    aios/
      main.go
  internal/
    <module_name>/
      <source files>
  docs/
    INDEX.md
    README.md
    global/
      ...
    standards/
      ...
    modules/
      ...
  var/
    optimizer.yaml
    roles.yaml
    policy.yaml
    models.yaml
    aios.db
    sessions/
    handoffs/
    reports/
  benchmark/
    wedges/
      v1/
        dataset.json
        tasks/
```

The deployment is a single Linux host. The host runs
Docker. The host runs the Go monorepo. The host
runs the CLI.

## 11. Build and Test Pipeline

The build and test pipeline has the following steps.

1. The pipeline resolves the dependencies through
   `go mod download`.
2. The pipeline runs the formatter through
   `gofmt -l .`.
3. The pipeline runs the linter through
   `go vet ./...` and `staticcheck ./...`.
4. The pipeline runs the test suite through
   `go test ./...`.
5. The pipeline runs the coverage report through
   `go test -cover ./...`.
6. The pipeline runs the benchmark through
   `github.com/<org>/aios/internal/benchmark`.

The pipeline runs in a CI environment. The CI
environment is GitHub Actions or a self-hosted
runner.

## 12. Observability

The aios project has three observability surfaces.

### 11.1 Audit Log

The audit log is the primary observability surface.
The log is append-only and hash-chained. The log
is queryable through `aios audit`.

### 11.2 Report

The report is the human-readable summary of a run.
The report is in three formats: text, JSON, and
HTML. The report is generated through
`aios report`.

### 11.3 Watcher

The watcher is the live observability surface. The
watcher streams the events to the CLI. The watcher
is in `github.com/<org>/aios/internal/watch`.

## 13. Security Posture

The aios project has the following security posture.

### 13.1 Enforcement Below the Model

Every tool call goes through the policy gateway.
The policy gateway is ordinary code outside the
model. The model cannot bypass the policy gateway.

### 13.2 Sandboxing

Every task runs in a sandbox. The sandbox isolates
the file system, the network, and the process. The
sandbox is in `github.com/<org>/aios/internal/sandbox`.

### 13.3 Egress Control

The egress proxy controls the network traffic. The
proxy blocks the network by default. The proxy
allows the network through the policy.

### 13.4 Approval Queue

The approval queue pauses the run on tier-3
actions. The queue blocks until the operator
responds.

### 13.5 Redaction

The redaction layer strips secrets before any log
write. The layer is in `github.com/<org>/aios/internal/core`.

### 13.6 Audit Log

The audit log is append-only and hash-chained.
The log is verifiable through the hash chain.

## 14. Risks and Mitigations

| Risk | Impact | Probability | Mitigation |
| --- | --- | --- | --- |
| The model quality is below the threshold. | High | Medium | The model router picks the cheapest tier that the verify layer accepts. The cost/quality table tracks the empirical pass rate. |
| The cost is above the budget. | High | Medium | The cost tracker enforces the budget. The router picks the cheapest tier. |
| The approval fatigue is too high. | Medium | High | The auto-allow rules reduce the prompts. The policy templates cover the common cases. |
| The audit log is too large. | Medium | Medium | The report summarizes the run. The search tool finds the events. |
| The hash chain breaks. | High | Low | The re-anchor tool repairs the chain. The audit log is backed up. |
| The sandbox is bypassed. | High | Low | The sandbox is tested in isolation. The escape attempts are logged. |
| The egress proxy is bypassed. | High | Low | The egress proxy is tested in isolation. The DNS is monitored. |
| The model drifts between runs. | High | Medium | The temperature is 0. The call fingerprint and the cost/quality table learn the empirical pass rate. The drift is recorded in the audit log. |
| The replay diverges from the original. | High | Low | The checkpoints are content addressed. The plan and the policy are immutable. The replay is re-execution from the same input. |
| The diff is too large to inspect. | Medium | Medium | The diff is in unified format. The report summarizes the diff. The search tool filters the diff. |

## 15. Version

This document is version 1.0. The previous version
was 0.9. The change is the addition of the
optimizer, retry, verify, roles, persist, and
image modules, and the closure of the remaining
v0.9 audit gaps.
