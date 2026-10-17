# Documentation Index

## Purpose

This document lists every file in the documentation
set. The list is for quick navigation. The reader
uses the list to find the file for a topic.

## Top-Level Files

| File | Description |
| --- | --- |
| `README.md` | The documentation overview. The file lists the layout, the module list, the module document set, and the merge order. |
| `INDEX.md` | This document. The file lists every file in the documentation set. |
| `TEMPLATE.md` | The module document template. The file shows the section order for the PRD, SRS, TRD, application flow, wireframes, and implementation plan. |
| `standards/STE-STYLE-GUIDE.md` | The ASD-STE100 writing standard. The file states the rules for prose. |
| `standards/CODING-STANDARD.md` | The coding standard. The file states the rules for code. The file is the code-side equivalent of STE for prose. |

## Global Documents

The global documents cover the project as a whole.
The documents are the umbrella for the per-module
documents. The documents cover the cross-module
requirements, the architecture, the data flow, the
deployment, the release plan, and the merge
sequence.

| File | Description |
| --- | --- |
| `global/PRD.md` | The global product requirements document. The file covers the product vision, the personas, the stories, and the success criteria. |
| `global/SRS.md` | The global software requirements specification. The file covers the module interface contract, the audit log contract, the policy contract, the cost contract, the approval contract, the retry contract, the verify contract, the role contract, the persist contract, the image contract, and the optimizer contract. |
| `global/TRD.md` | The global technical requirements document. The file covers the four-layer architecture, the module dependency graph, the cross-module data flow, the deployment topology, the build and test pipeline, the observability, and the security posture. |
| `global/APP-FLOW.md` | The global application flow. The file covers the run lifecycle, the user flows, the system flows, the error flows, and the sequence diagrams. |
| `global/WIREFRAMES.md` | The global wireframes. The file covers the CLI prompts, the report, the watcher event stream, the benchmark dashboard, and the audit log viewer. |
| `global/IMPLEMENTATION-PLAN.md` | The global implementation plan. The file covers the release phases, the cross-module test plan, the merge sequence, the release criteria, and the rollback plan. |

## Foundation Layer

The foundation layer is the lowest level. The layer
provides the typed schemas, the audit log, the
state machine, the redaction layer, the policy
gateway, the sandbox runner, the egress proxy, the
model adapter layer, and the approval service.

### aios-core

The foundation module. The module provides the typed
schemas, the audit log, the state machine, and the
redaction layer.

| File | Description |
| --- | --- |
| `modules/aios-core/PRD.md` | The product requirements document. |
| `modules/aios-core/SRS.md` | The software requirements specification. |
| `modules/aios-core/TRD.md` | The technical requirements document. |
| `modules/aios-core/APP-FLOW.md` | The application flow. |
| `modules/aios-core/WIREFRAMES.md` | The wireframes. The module has no user-facing surface. |
| `modules/aios-core/IMPLEMENTATION-PLAN.md` | The implementation plan. |

### aios-policy

The policy gateway. The module decides whether a
tool call is allowed, denied, or requires an
approval.

| File | Description |
| --- | --- |
| `modules/aios-policy/PRD.md` | The product requirements document. |
| `modules/aios-policy/SRS.md` | The software requirements specification. |
| `modules/aios-policy/TRD.md` | The technical requirements document. |
| `modules/aios-policy/APP-FLOW.md` | The application flow. |
| `modules/aios-policy/WIREFRAMES.md` | The wireframes. The module has no user-facing surface. |
| `modules/aios-policy/IMPLEMENTATION-PLAN.md` | The implementation plan. |

### aios-sandbox

The sandbox runner. The module runs untrusted code
in a safe environment.

| File | Description |
| --- | --- |
| `modules/aios-sandbox/PRD.md` | The product requirements document. |
| `modules/aios-sandbox/SRS.md` | The software requirements specification. |
| `modules/aios-sandbox/TRD.md` | The technical requirements document. |
| `modules/aios-sandbox/APP-FLOW.md` | The application flow. |
| `modules/aios-sandbox/WIREFRAMES.md` | The wireframes. The module has no user-facing surface. |
| `modules/aios-sandbox/IMPLEMENTATION-PLAN.md` | The implementation plan. |

### aios-egress

The egress proxy. The module controls the network
traffic from the agent.

| File | Description |
| --- | --- |
| `modules/aios-egress/PRD.md` | The product requirements document. |
| `modules/aios-egress/SRS.md` | The software requirements specification. |
| `modules/aios-egress/TRD.md` | The technical requirements document. |
| `modules/aios-egress/APP-FLOW.md` | The application flow. |
| `modules/aios-egress/WIREFRAMES.md` | The wireframes. The module has no user-facing surface. |
| `modules/aios-egress/IMPLEMENTATION-PLAN.md` | The implementation plan. |

### aios-models

The model adapter layer. The module connects to
model providers, routes requests, and tracks the
cost through the price table.

| File | Description |
| --- | --- |
| `modules/aios-models/PRD.md` | The product requirements document. |
| `modules/aios-models/SRS.md` | The software requirements specification. |
| `modules/aios-models/TRD.md` | The technical requirements document. |
| `modules/aios-models/APP-FLOW.md` | The application flow. |
| `modules/aios-models/WIREFRAMES.md` | The wireframes. The module has no user-facing surface. |
| `modules/aios-models/IMPLEMENTATION-PLAN.md` | The implementation plan. |

### aios-optimizer

The model router with cost and quality
optimization. The module picks the right model for
each task. The module tracks the cost. The module
adapts the routing strategy from the result of
every call.

| File | Description |
| --- | --- |
| `modules/aios-optimizer/PRD.md` | The product requirements document. |
| `modules/aios-optimizer/SRS.md` | The software requirements specification. |
| `modules/aios-optimizer/TRD.md` | The technical requirements document. |
| `modules/aios-optimizer/APP-FLOW.md` | The application flow. |
| `modules/aios-optimizer/WIREFRAMES.md` | The optimizer report wireframes. |
| `modules/aios-optimizer/IMPLEMENTATION-PLAN.md` | The implementation plan. |

### aios-approvals

The approval service. The module decides whether a
tier-3 action gets approved. The module persists
the decision. The module notifies the waiter.

| File | Description |
| --- | --- |
| `modules/aios-approvals/PRD.md` | The product requirements document. |
| `modules/aios-approvals/SRS.md` | The software requirements specification. |
| `modules/aios-approvals/TRD.md` | The technical requirements document. |
| `modules/aios-approvals/APP-FLOW.md` | The application flow. |
| `modules/aios-approvals/WIREFRAMES.md` | The approval prompt wireframes. |
| `modules/aios-approvals/IMPLEMENTATION-PLAN.md` | The implementation plan. |

### aios-persist

The session persistence layer. The module covers
the always-allow rule storage, the session state
checkpoint, the structured handoff between tasks,
and the session history.

| File | Description |
| --- | --- |
| `modules/aios-persist/PRD.md` | The product requirements document. |
| `modules/aios-persist/SRS.md` | The software requirements specification. |
| `modules/aios-persist/TRD.md` | The technical requirements document. |
| `modules/aios-persist/APP-FLOW.md` | The application flow. |
| `modules/aios-persist/WIREFRAMES.md` | The rule and session wireframes. |
| `modules/aios-persist/IMPLEMENTATION-PLAN.md` | The implementation plan. |

## Capability Layer

The capability layer is the middle level. The layer
provides the tool bus, the built-in tools, the
read and search tools, the edit tool, the git tool,
the memory tool, the watcher, the fanout, the
retry policy, the verify layer, the role registry,
the image tool, the ask tool, and the report
generator.

### aios-tools

The tool bus and the built-in tools. The module
covers file read, file write, file search, and
shell run.

| File | Description |
| --- | --- |
| `modules/aios-tools/PRD.md` | The product requirements document. |
| `modules/aios-tools/SRS.md` | The software requirements specification. |
| `modules/aios-tools/TRD.md` | The technical requirements document. |
| `modules/aios-tools/APP-FLOW.md` | The application flow. |
| `modules/aios-tools/WIREFRAMES.md` | The wireframes. The module has no user-facing surface. |
| `modules/aios-tools/IMPLEMENTATION-PLAN.md` | The implementation plan. |

### aios-search

The search and read tools. The module provides
glob search, content search, and file read at
scale.

| File | Description |
| --- | --- |
| `modules/aios-search/PRD.md` | The product requirements document. |
| `modules/aios-search/SRS.md` | The software requirements specification. |
| `modules/aios-search/TRD.md` | The technical requirements document. |
| `modules/aios-search/APP-FLOW.md` | The application flow. |
| `modules/aios-search/WIREFRAMES.md` | The wireframes. The module has no user-facing surface. |
| `modules/aios-search/IMPLEMENTATION-PLAN.md` | The implementation plan. |

### aios-edit

The edit tool. The module supports string-based
edit, line-based edit, and file write at scale.

| File | Description |
| --- | --- |
| `modules/aios-edit/PRD.md` | The product requirements document. |
| `modules/aios-edit/SRS.md` | The software requirements specification. |
| `modules/aios-edit/TRD.md` | The technical requirements document. |
| `modules/aios-edit/APP-FLOW.md` | The application flow. |
| `modules/aios-edit/WIREFRAMES.md` | The wireframes. The module has no user-facing surface. |
| `modules/aios-edit/IMPLEMENTATION-PLAN.md` | The implementation plan. |

### aios-git

The git integration. The module provides diff,
commit, branch, log, and merge.

| File | Description |
| --- | --- |
| `modules/aios-git/PRD.md` | The product requirements document. |
| `modules/aios-git/SRS.md` | The software requirements specification. |
| `modules/aios-git/TRD.md` | The technical requirements document. |
| `modules/aios-git/APP-FLOW.md` | The application flow. |
| `modules/aios-git/WIREFRAMES.md` | The wireframes. The module has no user-facing surface. |
| `modules/aios-git/IMPLEMENTATION-PLAN.md` | The implementation plan. |

### aios-memory

The in-session memory. The module provides a
key-value store with namespacing, TTL, and an
oplog.

| File | Description |
| --- | --- |
| `modules/aios-memory/PRD.md` | The product requirements document. |
| `modules/aios-memory/SRS.md` | The software requirements specification. |
| `modules/aios-memory/TRD.md` | The technical requirements document. |
| `modules/aios-memory/APP-FLOW.md` | The application flow. |
| `modules/aios-memory/WIREFRAMES.md` | The wireframes. The module has no user-facing surface. |
| `modules/aios-memory/IMPLEMENTATION-PLAN.md` | The implementation plan. |

### aios-watch

The file watcher. The module watches the
workspace and reports the file system events.

| File | Description |
| --- | --- |
| `modules/aios-watch/PRD.md` | The product requirements document. |
| `modules/aios-watch/SRS.md` | The software requirements specification. |
| `modules/aios-watch/TRD.md` | The technical requirements document. |
| `modules/aios-watch/APP-FLOW.md` | The application flow. |
| `modules/aios-watch/WIREFRAMES.md` | The watcher event wireframes. |
| `modules/aios-watch/IMPLEMENTATION-PLAN.md` | The implementation plan. |

### aios-fanout

The parallel sub-agent runner. The module spawns
sub-agents in parallel and merges the results.

| File | Description |
| --- | --- |
| `modules/aios-fanout/PRD.md` | The product requirements document. |
| `modules/aios-fanout/SRS.md` | The software requirements specification. |
| `modules/aios-fanout/TRD.md` | The technical requirements document. |
| `modules/aios-fanout/APP-FLOW.md` | The application flow. |
| `modules/aios-fanout/WIREFRAMES.md` | The fanout dashboard wireframes. |
| `modules/aios-fanout/IMPLEMENTATION-PLAN.md` | The implementation plan. |

### aios-retry

The retry policy layer. The module decides when
to retry a failed tool call. The module feeds the
error back to the model. The module caps the
retry count.

| File | Description |
| --- | --- |
| `modules/aios-retry/PRD.md` | The product requirements document. |
| `modules/aios-retry/SRS.md` | The software requirements specification. |
| `modules/aios-retry/TRD.md` | The technical requirements document. |
| `modules/aios-retry/APP-FLOW.md` | The application flow. |
| `modules/aios-retry/WIREFRAMES.md` | The wireframes. The module has no user-facing surface. |
| `modules/aios-retry/IMPLEMENTATION-PLAN.md` | The implementation plan. |

### aios-verify

The self-evaluation layer. The module lets the
agent check the result before the agent reports
the task as done. The module covers the
self-check, the test-runner tool, and the
verification prompt.

| File | Description |
| --- | --- |
| `modules/aios-verify/PRD.md` | The product requirements document. |
| `modules/aios-verify/SRS.md` | The software requirements specification. |
| `modules/aios-verify/TRD.md` | The technical requirements document. |
| `modules/aios-verify/APP-FLOW.md` | The application flow. |
| `modules/aios-verify/WIREFRAMES.md` | The wireframes. The module has no user-facing surface. |
| `modules/aios-verify/IMPLEMENTATION-PLAN.md` | The implementation plan. |

### aios-roles

The agent role registry. The module maps roles to
system prompts, tool sets, and model tiers. The
module lets the planner and the orchestrator use
the right configuration for every task.

| File | Description |
| --- | --- |
| `modules/aios-roles/PRD.md` | The product requirements document. |
| `modules/aios-roles/SRS.md` | The software requirements specification. |
| `modules/aios-roles/TRD.md` | The technical requirements document. |
| `modules/aios-roles/APP-FLOW.md` | The application flow. |
| `modules/aios-roles/WIREFRAMES.md` | The roles.yaml template wireframes. |
| `modules/aios-roles/IMPLEMENTATION-PLAN.md` | The implementation plan. |

### aios-image

The multimodal input and structured output
layer. The module covers the image read tool and
the schema-constrained generation.

| File | Description |
| --- | --- |
| `modules/aios-image/PRD.md` | The product requirements document. |
| `modules/aios-image/SRS.md` | The software requirements specification. |
| `modules/aios-image/TRD.md` | The technical requirements document. |
| `modules/aios-image/APP-FLOW.md` | The application flow. |
| `modules/aios-image/WIREFRAMES.md` | The wireframes. The module has no user-facing surface. |
| `modules/aios-image/IMPLEMENTATION-PLAN.md` | The implementation plan. |

### aios-ask

The ask-user tool. The module lets the agent ask
the operator a multiple-choice question.

| File | Description |
| --- | --- |
| `modules/aios-ask/PRD.md` | The product requirements document. |
| `modules/aios-ask/SRS.md` | The software requirements specification. |
| `modules/aios-ask/TRD.md` | The technical requirements document. |
| `modules/aios-ask/APP-FLOW.md` | The application flow. |
| `modules/aios-ask/WIREFRAMES.md` | The question prompt wireframes. |
| `modules/aios-ask/IMPLEMENTATION-PLAN.md` | The implementation plan. |

### aios-report

The report generator. The module turns a run into
a report in three formats.

| File | Description |
| --- | --- |
| `modules/aios-report/PRD.md` | The product requirements document. |
| `modules/aios-report/SRS.md` | The software requirements specification. |
| `modules/aios-report/TRD.md` | The technical requirements document. |
| `modules/aios-report/APP-FLOW.md` | The application flow. |
| `modules/aios-report/WIREFRAMES.md` | The report wireframes. |
| `modules/aios-report/IMPLEMENTATION-PLAN.md` | The implementation plan. |

## Agentic Layer

The agentic layer is the top level. The layer
provides the planner, the orchestrator, the
benchmark harness, and the CLI.

### aios-planner

The planner and the code-first verifier. The
module turns a goal into a TaskPlan.

| File | Description |
| --- | --- |
| `modules/aios-planner/PRD.md` | The product requirements document. |
| `modules/aios-planner/SRS.md` | The software requirements specification. |
| `modules/aios-planner/TRD.md` | The technical requirements document. |
| `modules/aios-planner/APP-FLOW.md` | The application flow. |
| `modules/aios-planner/WIREFRAMES.md` | The wireframes. The module has no user-facing surface. |
| `modules/aios-planner/IMPLEMENTATION-PLAN.md` | The implementation plan. |

### aios-runtime

The orchestrator. The module drives the execution
of a TaskPlan through the worker, the verifier, the
budget meter, and the decision tracer.

| File | Description |
| --- | --- |
| `modules/aios-runtime/PRD.md` | The product requirements document. |
| `modules/aios-runtime/SRS.md` | The software requirements specification. |
| `modules/aios-runtime/TRD.md` | The technical requirements document. |
| `modules/aios-runtime/APP-FLOW.md` | The application flow. |
| `modules/aios-runtime/WIREFRAMES.md` | The wireframes. The module has no user-facing surface. |
| `modules/aios-runtime/IMPLEMENTATION-PLAN.md` | The implementation plan. |

### aios-benchmark

The benchmark harness. The module runs the wedge
dataset against the orchestrator.

| File | Description |
| --- | --- |
| `modules/aios-benchmark/PRD.md` | The product requirements document. |
| `modules/aios-benchmark/SRS.md` | The software requirements specification. |
| `modules/aios-benchmark/TRD.md` | The technical requirements document. |
| `modules/aios-benchmark/APP-FLOW.md` | The application flow. |
| `modules/aios-benchmark/WIREFRAMES.md` | The benchmark report wireframes. |
| `modules/aios-benchmark/IMPLEMENTATION-PLAN.md` | The implementation plan. |

## Operator Layer

The operator layer is the top of the stack. The
layer provides the CLI and the approval daemon.
The layer is the entry point for the user.

### aios-cli

The first-run CLI and the approval daemon. The
module is the entry point for the user.

| File | Description |
| --- | --- |
| `modules/aios-cli/PRD.md` | The product requirements document. |
| `modules/aios-cli/SRS.md` | The software requirements specification. |
| `modules/aios-cli/TRD.md` | The technical requirements document. |
| `modules/aios-cli/APP-FLOW.md` | The application flow. |
| `modules/aios-cli/WIREFRAMES.md` | The CLI wireframes. |
| `modules/aios-cli/IMPLEMENTATION-PLAN.md` | The implementation plan. |

## Total Count

- Top-level: 2 files (`README.md`, `INDEX.md`)
- Module template: 1 file (`TEMPLATE.md`)
- Standards: 2 files
- Global documents: 6 files (PRD, SRS, TRD, application
  flow, wireframes, implementation plan)
- Modules: 25 modules
- Documents per module: 6 (PRD, SRS, TRD, application
  flow, wireframes, implementation plan)
- Module documents: 150 files
- Total: 161 files

## Module Merge Order

The project merges the modules in this order. The
order is from the lowest-level module to the
highest-level module.

1. aios-core
2. aios-policy
3. aios-sandbox
4. aios-egress
5. aios-models
6. aios-optimizer
7. aios-approvals
8. aios-persist
9. aios-tools
10. aios-search
11. aios-edit
12. aios-git
13. aios-memory
14. aios-watch
15. aios-fanout
16. aios-retry
17. aios-verify
18. aios-roles
19. aios-image
20. aios-ask
21. aios-report
22. aios-planner
23. aios-runtime
24. aios-benchmark
25. aios-cli

## Cross-Reference

- For the project overview, see `docs/README.md`.
- For the module document template, see
  `docs/modules/TEMPLATE.md`.
- For the STE writing standard, see
  `docs/standards/STE-STYLE-GUIDE.md`.
- For the coding standard, see
  `docs/standards/CODING-STANDARD.md`.
