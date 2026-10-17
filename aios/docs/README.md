# Project Documentation

## Purpose

This directory contains the product, requirements, and
design documents for the project. The documents follow
the ASD-STE100 writing standard. See the Standards
folder for the writing standard and the coding standard.

The whole point of aios is to make complex coding
and large systems deterministic. Every part of the
system that is not the model, the clock, the network,
or the operator is a pure function of its inputs.
The system is replayable, diffable, and reproducible
on a second host. The audit log is the proof.

## Layout

The documentation has three top-level folders.

- `standards/` — the writing and coding standards.
- `global/` — the project-level documents that
  cover all 25 modules.
- `modules/` — the documents for each project module.

The `global/` folder contains the project-level
PRD, SRS, TRD, application flow, wireframes, and
implementation plan. The documents cover the
cross-module requirements, the architecture, the
data flow, the deployment, the release plan, and
the merge sequence.

The `modules/` folder contains one folder per module.
Each module folder has the same document set. The set
lets a developer understand, build, test, and review the
module without reading the rest of the project.

## Module List

The project has 25 modules. Each module is an independent
project that ships in its own folder. The modules are
merged into the main project after each module is finished.

### Foundation Layer

| Module | Folder | Purpose |
| --- | --- | --- |
| aios-core | `modules/aios-core/` | Typed schemas, audit log, state machine |
| aios-policy | `modules/aios-policy/` | Policy gateway with rule table |
| aios-sandbox | `modules/aios-sandbox/` | Sandbox runner |
| aios-egress | `modules/aios-egress/` | Egress proxy |
| aios-models | `modules/aios-models/` | Model adapter, router, price table |
| aios-optimizer | `modules/aios-optimizer/` | Tiered model router with cost optimization |
| aios-approvals | `modules/aios-approvals/` | Approval service |
| aios-persist | `modules/aios-persist/` | Always-allow rules, checkpoint, handoff |

### Capability Layer

| Module | Folder | Purpose |
| --- | --- | --- |
| aios-tools | `modules/aios-tools/` | Tool bus and file tools |
| aios-search | `modules/aios-search/` | Glob search, content search, file read at scale |
| aios-edit | `modules/aios-edit/` | String edit, line edit, file write at scale |
| aios-git | `modules/aios-git/` | Diff, commit, branch, log, merge |
| aios-memory | `modules/aios-memory/` | Key-value store with namespacing and TTL |
| aios-watch | `modules/aios-watch/` | File system watcher |
| aios-fanout | `modules/aios-fanout/` | Parallel sub-agent runner |
| aios-retry | `modules/aios-retry/` | Retry policy with error feedback |
| aios-verify | `modules/aios-verify/` | Self-check and run_tests tool |
| aios-roles | `modules/aios-roles/` | Agent role registry |
| aios-image | `modules/aios-image/` | Multimodal input and structured output |
| aios-ask | `modules/aios-ask/` | Ask-user tool |
| aios-report | `modules/aios-report/` | Report generator |

### Agentic Layer

| Module | Folder | Purpose |
| --- | --- | --- |
| aios-planner | `modules/aios-planner/` | Planner and code-first verifier |
| aios-runtime | `modules/aios-runtime/` | Orchestrator, protocols, resume |
| aios-benchmark | `modules/aios-benchmark/` | Benchmark harness |

### Operator Layer

| Module | Folder | Purpose |
| --- | --- | --- |
| aios-cli | `modules/aios-cli/` | First-run CLI and approval daemon |

## Module Document Set

Each module folder has the same six documents. The
documents are written in this order.

1. `PRD.md` — Product Requirements Document
2. `SRS.md` — Software Requirements Specification
3. `TRD.md` — Technical Requirements Document
4. `APP-FLOW.md` — Application Flow
5. `WIREFRAMES.md` — Wireframes (only for modules with
   a user-facing surface)
6. `IMPLEMENTATION-PLAN.md` — Implementation Plan

The Implementation Plan has the test plan, the merge
plan, and the risk register.

## Reading Order

A new reader starts with the standards, then the module
list above, then the module of interest.

1. `standards/STE-STYLE-GUIDE.md`
2. `standards/CODING-STANDARD.md`
3. The PRD of the module
4. The SRS of the module
5. The TRD of the module
6. The Application Flow of the module
7. The Wireframes of the module (when present)
8. The Implementation Plan of the module

## Module Independence

Each module is an independent project. The module
ships in a folder under `modules/`. The module has its
own tests, its own documentation, and its own
implementation plan. The module does not depend on the
other modules for code.

The module declares its dependencies on the other
modules in the TRD. The dependency is on the module
interface, not on the module code.

## Merge Order

The project merges the modules in this order. The order
is from the lowest-level module to the highest-level
module.

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

## Total Count

- Top-level files: 2 (`README.md`, `INDEX.md`)
- Module template: 1 (`TEMPLATE.md`)
- Standards: 2
- Global documents: 6 (PRD, SRS, TRD, application
  flow, wireframes, implementation plan)
- Modules: 25
- Documents per module: 6
- Module documents: 150
- Total: 161 files

## Version

This documentation set is version 1.0. The previous
version was 0.9. The change is the addition of the
optimizer, retry, verify, roles, persist, and image
modules, and the closure of the remaining v0.9 audit
gaps. The full module set is now 25 modules across
the foundation, capability, agentic, and operator
layers.
