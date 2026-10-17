# aios — Global Implementation Plan

## 1. Purpose

This document states the integration plan for the
aios project. The per-module IMPLEMENTATION-PLAN
documents are the source of truth for the module
detail. This document is the source of truth for the
release plan, the cross-module test plan, and the
merge sequence.

## 2. Scope

The aios project is in scope for this document.

- The release phases.
- The cross-module test plan.
- The merge sequence.
- The release criteria.
- The rollback plan.

## 3. Reference Documents

- Global PRD. See `docs/global/PRD.md`.
- Global SRS. See `docs/global/SRS.md`.
- Global TRD. See `docs/global/TRD.md`.
- Global Application Flow. See
  `docs/global/APP-FLOW.md`.
- The per-module IMPLEMENTATION-PLAN under
  `docs/modules/<module>/IMPLEMENTATION-PLAN.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Phase | A unit of work in the release plan. |
| Release | A versioned set of modules. |

## 5. Release Phases

The aios project ships in four releases. The releases
are listed below.

### 5.1 Release 0.9: Foundation

The release ships the foundation layer. The layer
provides the cross-cutting concerns. The release
includes the following modules.

- `github.com/<org>/aios/internal/core`
- `github.com/<org>/aios/internal/policy`
- `github.com/<org>/aios/internal/sandbox`
- `github.com/<org>/aios/internal/egress`
- `github.com/<org>/aios/internal/models`
- `github.com/<org>/aios/internal/approvals`
- `github.com/<org>/aios/internal/persist`

The release establishes the typed schemas, the
audit log, the state machine, the redaction layer,
the policy gateway, the sandbox runner, the egress
proxy, the model adapter layer, the approval
service, and the session persistence layer.

### 5.2 Release 1.0: Capability

The release ships the capability layer. The layer
provides the tools and the policies. The release
includes the following modules.

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
- `github.com/<org>/aios/internal/optimizer`

The release establishes the tool bus, the built-in
tools, the search and read tools, the edit tools,
the git integration, the in-session memory, the
file watcher, the parallel sub-agent runner, the
retry policy, the verify layer, the role registry,
the image tool, the ask-user tool, the report
generator, and the model router with cost
optimization.

### 5.3 Release 1.1: Agentic

The release ships the agentic layer. The layer
provides the planner, the orchestrator, and the
benchmark harness. The release includes the
following modules.

- `github.com/<org>/aios/internal/planner`
- `github.com/<org>/aios/internal/runtime`
- `github.com/<org>/aios/internal/benchmark`

The release establishes the planner, the
orchestrator, and the benchmark harness.

### 5.4 Release 1.2: Operator

The release ships the operator layer. The layer
provides the CLI and the approval daemon. The
release includes the following module.

- `github.com/<org>/aios/internal/cli`

The release establishes the CLI as the entry point
for the user.

## 6. Cross-Module Test Plan

The cross-module test plan covers the requirements
that span more than one module. The plan has the
following test cases.

### 6.1 Module Interface Tests

The test cases for the module interface are listed
below.

- The module interface is exposed through
  `github.com/<org>/aios/internal/<module>`.
- The module interface is stable across minor
  versions.
- The module interface is tested in isolation.

### 6.2 Audit Log Tests

The test cases for the audit log are listed below.

- Every side effect produces an audit row.
- The audit row includes the required fields.
- The audit log is append-only.
- The audit log is hash-chained.
- The audit log is verifiable.

### 6.3 Policy Tests

The test cases for the policy are listed below.

- Every tool call goes through the policy gateway.
- The policy returns one of the three actions.
- The `ask` action routes to the approval service.
- The policy decision is recorded in the audit
  log.

### 6.4 Cost Tests

The test cases for the cost are listed below.

- Every model call records the cost.
- The cost is computed through the price table.
- The cost tracker enforces the per-task budget.
- The cost tracker enforces the per-call budget.
- The cost is recorded in the audit log.

### 6.5 Approval Tests

The test cases for the approval are listed below.

- Every tier-3 action routes to the approval
  service.
- The approval service blocks until the operator
  responds.
- The approval service persists the decision.
- The "always" option creates a standing rule.

### 6.6 Retry Tests

The test cases for the retry are listed below.

- Every tool call failure goes through the retry
  policy.
- The retry policy returns one of the three
  actions.
- The retry policy caps the retries.
- The retry policy supports per-tool limits.

### 6.7 Verify Tests

The test cases for the verify are listed below.

- Every task result goes through the self-check.
- The self-check returns one of the three
  statuses.
- The run_tests tool runs the test command.
- The run_tests tool returns the test result.

### 6.8 Role Tests

The test cases for the role are listed below.

- Every task resolves to a role.
- The role includes the system prompt and the
  tool set.
- The role is loaded from `var/roles.yaml`.
- The default roles are provided.

### 6.9 Persist Tests

The test cases for the persist are listed below.

- Every standing approval is persisted.
- The session state is checkpointed.
- The handoff store passes the artifacts.
- The session history is queryable.

### 6.10 Image Tests

The test cases for the image are listed below.

- The image read tool reads the image.
- The image read tool enforces the size limit.
- The structured output validator validates the
  response.

### 6.11 Optimizer Tests

The test cases for the optimizer are listed below.

- The model router picks the right tier.
- The cost tracker records the cost.
- The quality feedback updates the strategy.
- The cascade mode escalates on low confidence.

## 7. Merge Sequence

The merge sequence follows the dependency graph. The
foundation layer merges first. The capability layer
merges next. The agentic layer merges next. The
operator layer merges last.

The merge sequence is listed below.

1. `github.com/<org>/aios/internal/core` — the foundation module.
2. `github.com/<org>/aios/internal/policy` — the policy gateway.
3. `github.com/<org>/aios/internal/sandbox` — the sandbox runner.
4. `github.com/<org>/aios/internal/egress` — the egress proxy.
5. `github.com/<org>/aios/internal/models` — the model adapter layer.
6. `github.com/<org>/aios/internal/optimizer` — the model router.
7. `github.com/<org>/aios/internal/approvals` — the approval service.
8. `github.com/<org>/aios/internal/persist` — the session persistence.
9. `github.com/<org>/aios/internal/tools` — the tool bus.
10. `github.com/<org>/aios/internal/search` — the search tools.
11. `github.com/<org>/aios/internal/edit` — the edit tools.
12. `github.com/<org>/aios/internal/git` — the git integration.
13. `github.com/<org>/aios/internal/memory` — the in-session memory.
14. `github.com/<org>/aios/internal/watch` — the file watcher.
15. `github.com/<org>/aios/internal/fanout` — the parallel runner.
16. `github.com/<org>/aios/internal/retry` — the retry policy.
17. `github.com/<org>/aios/internal/verify` — the self-check.
18. `github.com/<org>/aios/internal/roles` — the role registry.
19. `github.com/<org>/aios/internal/image` — the image tool.
20. `github.com/<org>/aios/internal/ask` — the ask-user tool.
21. `github.com/<org>/aios/internal/report` — the report generator.
22. `github.com/<org>/aios/internal/planner` — the planner.
23. `github.com/<org>/aios/internal/runtime` — the orchestrator.
24. `github.com/<org>/aios/internal/benchmark` — the benchmark harness.
25. `github.com/<org>/aios/internal/cli` — the CLI.

## 8. Release Criteria

The aios project is releasable when the following
criteria are met.

- The test suite has a 100 percent pass rate.
- The test suite has 100 percent coverage for the
  public functions of every module.
- The `go vet + staticcheck` report is clean.
- The `gofmt + go vet check` report is clean.
- The benchmark passes at least 90 percent of the
  wedge dataset.
- The average cost per task is under $0.50.
- The average time per task is under 5 minutes.
- The audit log is hash-chained and verifiable.
- The CLI runs the end-to-end flow without
  crashes.

## 9. Rollback Plan

The rollback plan covers the case when a release
fails. The plan has the following steps.

1. The operator runs `aios rollback <version>`.
2. The CLI checks out the previous version of
   the source code.
3. The CLI checks out the previous version of
   the configuration.
4. The CLI restarts the service.
5. The CLI verifies the rollback.

## 10. Post-Release Plan

The post-release plan covers the work after the
release. The plan has the following items.

- The operator collects the metrics from the
  release.
- The operator compares the metrics to the
  release criteria.
- The operator writes the post-mortem if the
  release fails.
- The operator updates the documentation.

## 11. Schedule

| Phase | Release | Duration | Modules |
| --- | --- | --- | --- |
| Phase 1: Foundation | 0.9 | 30 days | 8 modules |
| Phase 2: Capability | 1.0 | 45 days | 14 modules |
| Phase 3: Agentic | 1.1 | 30 days | 3 modules |
| Phase 4: Operator | 1.2 | 15 days | 1 module |

The total duration is 120 days.

## 12. Version

This document is version 1.0. The previous version
was 0.9. The change is the addition of the
optimizer, retry, verify, roles, persist, and
image modules, and the closure of the remaining
v0.9 audit gaps.
