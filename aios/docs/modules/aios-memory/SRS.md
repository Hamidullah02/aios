# aios-memory — Software Requirements Specification

## 1. Purpose

This document states the software requirements for
the aios-memory module.

## 2. Scope

The scope of this document is the aios-memory
module. The module is a Go package.

## 3. Reference Documents

- aios-memory PRD. See `docs/modules/aios-memory/PRD.md`.
- aios-core SRS. See `docs/modules/aios-core/SRS.md`.
- aios-persist SRS. See `docs/modules/aios-persist/SRS.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Artifact | A typed object in the memory. |
| ArtifactKind | A typed string: task, file, decision, observation, tool_result, model_output. |
| Scope | A typed string: session, project, user. |
| Layer | A typed string: raw, indexed, compacted. |
| Query | A typed request for artifacts. |
| Retrieval | A typed operation that runs a query. |
| Compaction | A typed operation that produces a compacted view. |
| Pin | A typed directive that forces inclusion. |

## 5. Functional Requirements

### 5.1 Content-Addressed Storage

**REQ-MEM-STO-001** The module shall provide the
`Store` type. The type shall hold the raw layer.

**REQ-MEM-STO-002** Every artifact shall be
identified by its content hash. The content hash
shall be the SHA-256 of the artifact payload.

**REQ-MEM-STO-003** The store shall provide the
`Put` method. The method shall accept an
artifact. The method shall return the content
hash. The method shall be idempotent. Two puts of
the same content return the same hash.

**REQ-MEM-STO-004** The store shall provide the
`Get` method. The method shall accept a content
hash. The method shall return the artifact. The
method shall return `ErrNotFound` when the
artifact is not in the store.

**REQ-MEM-STO-005** The store shall be
append-only. The store shall not provide a
`Delete` method. The developer can `aios memory
gc` to compact.

**REQ-MEM-STO-006** The store shall be persisted
to disk. The default path is
`var/memory/raw/`. The store shall be
content-addressed on disk. The file name is the
content hash.

### 5.2 Typed Index

**REQ-MEM-IDX-001** The module shall provide the
`Index` type. The type shall hold the indexed
layer.

**REQ-MEM-IDX-002** The index shall categorize
artifacts as one of the following kinds.

- `task`. A unit of work in the plan.
- `file`. A file read or written by the agent.
- `decision`. A decision the agent made.
- `observation`. A fact the agent noticed.
- `tool_result`. The result of a tool call.
- `model_output`. The output of a model call.

**REQ-MEM-IDX-003** The index shall be keyed by
content hash. The index shall be a map. The key
is the content hash. The value is the index
entry.

**REQ-MEM-IDX-004** The index entry shall include
the kind, the scope, the session id, the task
id, the timestamp, the tags, and the content
hash.

**REQ-MEM-IDX-005** The index shall be persisted
to disk. The default path is
`var/memory/index.json`. The file is a JSON
object.

### 5.3 Scopes

**REQ-MEM-SCO-001** The module shall support
three scopes: `session`, `project`, `user`.

**REQ-MEM-SCO-002** The `session` scope shall
die when the session ends. The session scope
shall be stored in memory only.

**REQ-MEM-SCO-003** The `project` scope shall
persist across sessions in the same project.
The project scope shall be stored at
`var/memory/project/`. The project scope shall
be `git`-trackable.

**REQ-MEM-SCO-004** The `user` scope shall
persist across projects. The user scope shall
be stored at `~/.aios/memory/`. The user scope
shall be encrypted at rest.

### 5.4 Query Language

**REQ-MEM-QUE-001** The module shall provide the
`Query` type. The type shall accept the
following fields.

- `Kinds`. A list of artifact kinds. The query
  matches artifacts of the listed kinds.
- `Scopes`. A list of scopes. The query matches
  artifacts in the listed scopes.
- `Tags`. A list of tags. The query matches
  artifacts tagged with any of the listed tags.
- `Pattern`. A glob pattern. The query matches
  artifacts whose content matches the pattern.
- `Since`. A timestamp. The query matches
  artifacts seen since the timestamp.
- `Limit`. An int. The query returns at most
  `Limit` artifacts.

**REQ-MEM-QUE-002** The query shall return a
list of content hashes. The list shall be sorted
by the timestamp in descending order.

**REQ-MEM-QUE-003** The query shall support
compound filters. The query shall support
`AND` and `OR` operators. The default is `AND`.

### 5.5 Retrieval Algorithm

**REQ-MEM-RET-001** The module shall provide the
`Retrieval` type. The type shall accept the
index, the pin set, and the budget.

**REQ-MEM-RET-002** The retrieval shall provide
the `Retrieve` method. The method shall accept
a query. The method shall return a compacted
view.

**REQ-MEM-RET-003** The retrieval shall run the
query against the index. The retrieval shall
rank the matching artifacts by the following
priority.

1. Pinned artifacts. Always include.
2. Artifacts from the current task. Always
   include.
3. Artifacts matching the current role's
   default context. Include if budget allows.
4. Artifacts from sibling tasks in the current
   plan. Include if budget allows.
5. Artifacts from past sessions matching the
   query. Include if budget allows.

**REQ-MEM-RET-004** The retrieval shall enforce
the budget. The budget is the number of tokens
or the number of artifacts. The retrieval shall
truncate the result to fit the budget.

**REQ-MEM-RET-005** The retrieval shall be a
pure function. Two retrievals with the same
input return the same compacted view.

### 5.6 Compaction Algorithm

**REQ-MEM-COM-001** The module shall provide the
`Compaction` type. The type shall accept the
index, the pin set, and the budget.

**REQ-MEM-COM-002** The compaction shall provide
the `Compact` method. The method shall accept a
query. The method shall return a compacted
view.

**REQ-MEM-COM-003** The compaction shall
truncate the indexed layer to the budget. The
truncation shall keep the most relevant
artifacts first.

**REQ-MEM-COM-004** The compaction shall support
task-scoped truncation. The agent can say
"compact the memory for this task, keeping the
goal, the plan, and the last 10 actions."

**REQ-MEM-COM-005** The compaction shall be
observable. The developer can see what was kept
and what was dropped.

**REQ-MEM-COM-006** The compaction shall be a
pure function. Two compactions with the same
input return the same compacted view.

### 5.7 Pinning

**REQ-MEM-PIN-001** The module shall provide the
`PinSet` type. The type shall hold the set of
pinned content hashes.

**REQ-MEM-PIN-002** The pin set shall be
content-addressed. The pin set id shall be the
SHA-256 of the pin set payload.

**REQ-MEM-PIN-003** The pin set shall be
persisted to disk. The default path is
`var/memory/pins.json`. The file is a JSON
object.

**REQ-MEM-PIN-004** The pin set shall be
honored. Pinned artifacts shall always be
included in the compacted view.

**REQ-MEM-PIN-005** The pin set shall be
editable. The developer can `aios memory pin
<hash>` and `aios memory unpin <hash>`.

### 5.8 `aios memory` CLI

**REQ-MEM-CLI-001** The CLI shall provide the
`aios memory` subcommand.

**REQ-MEM-CLI-002** The subcommand shall
provide the `show` subcommand. The subcommand
shall print the current compacted view. The
subcommand shall accept the `--hash` flag to
print a single artifact.

**REQ-MEM-CLI-003** The subcommand shall
provide the `grep` subcommand. The subcommand
shall accept a pattern. The subcommand shall
return the matching artifacts.

**REQ-MEM-CLI-004** The subcommand shall
provide the `pin` and `unpin` subcommands. The
subcommands shall accept a content hash.

**REQ-MEM-CLI-005** The subcommand shall
provide the `prune` subcommand. The subcommand
shall accept a content hash. The subcommand
shall mark the artifact as pruned.

**REQ-MEM-CLI-006** The subcommand shall
provide the `diff` subcommand. The subcommand
shall accept two session ids. The subcommand
shall return the artifacts that differ.

**REQ-MEM-CLI-007** The subcommand shall
provide the `gc` subcommand. The subcommand
shall compact the raw layer to a single file.
The subcommand shall preserve the index.

**REQ-MEM-CLI-008** The subcommand shall
provide the `stats` subcommand. The subcommand
shall print the raw layer size, the indexed
layer size, the pin set size, and the number
of artifacts per kind.

**REQ-MEM-CLI-009** The subcommand shall
print output as JSON. The output shall be
grep-friendly. The developer can pipe to `jq`.

## 6. Non-Functional Requirements

**REQ-MEM-NFR-001** The module shall run on Go
1.23.

**REQ-MEM-NFR-002** The module shall have 100
percent coverage for the public functions.

**REQ-MEM-NFR-003** The retrieval algorithm
shall have a maximum latency of 50
milliseconds for a 10,000-artifact indexed
layer.

**REQ-MEM-NFR-004** The compaction algorithm
shall have a maximum latency of 100
milliseconds for a 10,000-artifact indexed
layer.

**REQ-MEM-NFR-005** The memory shall be
content-addressed on disk. The file name shall
be the content hash.

## 7. Interface Requirements

**REQ-MEM-IFC-001** The module shall export the
following items from
`github.com/<org>/aios/internal/memory`.

- `Store`
- `Index`
- `Query`
- `Retrieval`
- `Compaction`
- `PinSet`
- `Artifact`
- `ArtifactKind`
- `Scope`
- `NewArtifact`
- `NewQuery`
- `NewRetrieval`
- `NewCompaction`
- `NewPinSet`

## 8. Data Requirements

**REQ-MEM-DAT-001** The `Artifact` schema shall
include the fields below.

- `Kind` (string: task, file, decision,
  observation, tool_result, model_output)
- `Scope` (string: session, project, user)
- `SessionID` (string)
- `TaskID` (string, optional)
- `Timestamp` (string, ISO 8601)
- `Tags` (list of string)
- `Content` (bytes)
- `ContentHash` (string, SHA-256 hex)

**REQ-MEM-DAT-002** The `Query` schema shall
include the fields below.

- `Kinds` (list of string)
- `Scopes` (list of string)
- `Tags` (list of string)
- `Pattern` (string, glob)
- `Since` (string, ISO 8601, optional)
- `Limit` (int)
- `Operator` (string: AND, OR)

**REQ-MEM-DAT-003** The `CompactedView` schema
shall include the fields below.

- `Artifacts` (list of Artifact, content-
  addressed)
- `TotalTokens` (int)
- `Truncated` (bool)
- `PinSetID` (string, SHA-256 hex)
- `IndexID` (string, SHA-256 hex)

## 9. Error Handling

**REQ-MEM-ERR-001** The store shall return
`ErrNotFound` when the artifact is not in the
store.

**REQ-MEM-ERR-002** The index shall return
`ErrIndexCorrupt` when the index file fails to
parse.

**REQ-MEM-ERR-003** The retrieval shall return
`ErrBudgetTooSmall` when the budget is smaller
than the smallest artifact.

## 10. Acceptance Criteria

1. Every REQ in this document has a test case.
2. The test suite has a 100 percent pass rate.
3. The test suite has 100 percent coverage for
   the public functions.
4. Every artifact is identified by its content
   hash.
5. The retrieval algorithm is a pure function.
6. The compaction algorithm is a pure function.
7. The pin set is honored.
8. The memory persists across sessions.
9. The `aios memory` CLI is grep-friendly.
