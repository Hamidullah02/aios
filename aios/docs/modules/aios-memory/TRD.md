# aios-memory — Technical Requirements Document

## 1. Purpose

This document states the technical approach for the
aios-memory module.

## 2. Scope

The scope of this document is the aios-memory
module. The module is a Go package.

## 3. Reference Documents

- aios-memory PRD. See `docs/modules/aios-memory/PRD.md`.
- aios-memory SRS. See `docs/modules/aios-memory/SRS.md`.
- aios-core TRD. See `docs/modules/aios-core/TRD.md`.
- aios-persist TRD. See `docs/modules/aios-persist/TRD.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| SHA-256 | A cryptographic hash function. |
| Glob | A pattern for filename matching. |
| SHA-256 hex | A SHA-256 hash encoded as a lowercase hex string of 64 characters. |

## 5. Architecture

The aios-memory module has seven components.

- `internal/memory/store` — the content-addressed
  storage.
- `internal/memory/index` — the typed index.
- `internal/memory/scope` — the three scopes.
- `internal/memory/query` — the query language.
- `internal/memory/retrieval` — the retrieval
  algorithm.
- `internal/memory/compaction` — the compaction
  algorithm.
- `internal/memory/pin` — the pinning mechanism.

The seven components depend on aios-core. The
components depend on each other in this order.
The store has no dependencies. The index depends
on the store. The scope wraps the index. The
query uses the index. The retrieval uses the
index, the query, and the pin set. The compaction
uses the index, the query, the pin set, and the
retrieval. The pin uses the store.

## 6. Components

### 6.1 Content-Addressed Storage

The store holds the raw layer. The store is a
content-addressed file system. The file name is
the content hash. The file content is the
artifact payload.

The store provides the `Put`, `Get`, and
`Has` methods. The `Put` method is idempotent.
The `Get` method returns `ErrNotFound` when the
artifact is not in the store.

The store uses `modernc.org/sqlite` for the
metadata index. The metadata index maps the
content hash to the file path. The metadata
index is rebuilt on startup from the file
system.

The store is append-only. The store does not
provide a `Delete` method. The `gc` operation
compacts the store to a single file.

### 6.2 Typed Index

The index holds the indexed layer. The index is
a JSON object. The keys are the content hashes.
The values are the index entries.

The index categorizes artifacts. The categories
are listed below.

- `task`. A unit of work in the plan.
- `file`. A file read or written by the agent.
- `decision`. A decision the agent made.
- `observation`. A fact the agent noticed.
- `tool_result`. The result of a tool call.
- `model_output`. The output of a model call.

The index provides the `Add`, `Get`, `Query`,
and `Save` methods. The `Add` method is
idempotent. The `Query` method runs a query
against the in-memory index.

The index is loaded from disk on startup. The
index is saved to disk on every `Add` call.
The save is atomic through a `rename` syscall.

### 6.3 Scopes

The scope wraps the index with a scope filter.
The scope provides the `Put`, `Get`, and
`Query` methods. The methods filter by the
scope.

The three scopes are listed below.

- `session`. The session scope is in-memory
  only. The session scope dies when the
  session ends.
- `project`. The project scope is at
  `var/memory/project/`. The project scope is
  `git`-trackable.
- `user`. The user scope is at
  `~/.aios/memory/`. The user scope is
  encrypted at rest through `crypto/aes`.

### 6.4 Query Language

The query is a typed object. The query accepts
the following fields.

- `Kinds`. A list of artifact kinds.
- `Scopes`. A list of scopes.
- `Tags`. A list of tags.
- `Pattern`. A glob pattern. The query matches
  artifacts whose content matches the pattern.
- `Since`. A timestamp.
- `Limit`. An int.
- `Operator`. A string: `AND` or `OR`.

The query is a value type. The query has no
methods. The query is passed to the retrieval
and the compaction.

### 6.5 Retrieval Algorithm

The retrieval algorithm is a pure function. The
retrieval algorithm runs the query against the
index. The retrieval algorithm ranks the
matching artifacts.

The ranking priority is listed below.

1. Pinned artifacts. Always include.
2. Artifacts from the current task. Always
   include.
3. Artifacts matching the current role's
   default context. Include if budget allows.
4. Artifacts from sibling tasks in the current
   plan. Include if budget allows.
5. Artifacts from past sessions matching the
   query. Include if budget allows.

The retrieval enforces the budget. The budget
is the number of tokens or the number of
artifacts. The retrieval truncates the result
to fit the budget.

The retrieval returns a `CompactedView`. The
view includes the artifacts, the total tokens,
the truncation flag, the pin set id, and the
index id.

### 6.6 Compaction Algorithm

The compaction algorithm is a pure function. The
compaction algorithm truncates the indexed
layer to the budget. The truncation keeps the
most relevant artifacts first.

The compaction supports task-scoped truncation.
The agent can say "compact the memory for this
task, keeping the goal, the plan, and the last
10 actions." The compaction returns a
`CompactedView` that satisfies the constraint.

The compaction is observable. The
`CompactedView` includes the artifacts that
were kept and the artifacts that were dropped.
The developer can `aios memory show-compaction`
to see the result.

### 6.7 Pinning

The pin set is a JSON object. The keys are the
content hashes. The values are the pin
metadata.

The pin set is content-addressed. The pin set
id is the SHA-256 of the pin set payload.

The pin set is honored. Pinned artifacts are
always included in the compacted view. The
retrieval algorithm checks the pin set first.

The pin set is editable. The developer can
`aios memory pin <hash>` and `aios memory
unpin <hash>`. The pin set is saved to disk on
every change.

## 7. Data Structures

### 7.1 Artifact

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `Kind` | string | yes | The artifact kind. |
| `Scope` | string | yes | The artifact scope. |
| `SessionID` | string | yes | The session id. |
| `TaskID` | string | no | The task id. |
| `Timestamp` | string | yes | The ISO 8601 timestamp. |
| `Tags` | []string | no | The tags. |
| `Content` | []byte | yes | The content. |
| `ContentHash` | string | yes | The SHA-256 hex. |

### 7.2 Query

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `Kinds` | []string | no | The kinds. |
| `Scopes` | []string | no | The scopes. |
| `Tags` | []string | no | The tags. |
| `Pattern` | string | no | The glob pattern. |
| `Since` | string | no | The ISO 8601 timestamp. |
| `Limit` | int | no | The limit. |
| `Operator` | string | no | The operator: AND or OR. |

### 7.3 CompactedView

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `Artifacts` | []Artifact | yes | The artifacts. |
| `TotalTokens` | int | yes | The total tokens. |
| `Truncated` | bool | yes | Whether the view was truncated. |
| `PinSetID` | string | yes | The pin set id. |
| `IndexID` | string | yes | The index id. |

## 8. Interfaces

### 8.1 Store

```go
// Store is the content-addressed storage.
type Store struct {
    path string
}

// Put stores an artifact. The method is
// idempotent. Two puts of the same content
// return the same hash.
func (s *Store) Put(content []byte) (string, error)

// Get returns the artifact for the content hash.
// The method returns ErrNotFound when the
// artifact is not in the store.
func (s *Store) Get(hash string) ([]byte, error)

// Has returns true when the artifact is in the
// store.
func (s *Store) Has(hash string) (bool, error)
```

### 8.2 Retrieval

```go
// Retrieval runs a query against the index and
// returns a compacted view.
type Retrieval struct {
    index   *Index
    pinSet  *PinSet
    budget  int
}

// Retrieve returns a compacted view. The method
// is a pure function. Two retrievals with the
// same input return the same compacted view.
func (r *Retrieval) Retrieve(
    query Query,
    currentTaskID string,
) (CompactedView, error)
```

### 8.3 Compaction

```go
// Compaction truncates the indexed layer to the
// budget.
type Compaction struct {
    index  *Index
    pinSet *PinSet
}

// Compact returns a compacted view that fits the
// budget. The method is a pure function. Two
// compactions with the same input return the same
// compacted view.
func (c *Compaction) Compact(
    query Query,
    currentTaskID string,
    budgetTokens int,
) (CompactedView, error)
```

## 9. Algorithms

### 9.1 Retrieval Algorithm

The retrieval algorithm works as follows.

1. Load the pin set. The pin set is content-
   addressed.
2. Run the query against the index. The query
   returns a list of content hashes.
3. Partition the list by the ranking priority.
4. Truncate the list to fit the budget.
5. Return the `CompactedView`.

### 9.2 Compaction Algorithm

The compaction algorithm works as follows.

1. Run the query against the index.
2. Sort the matching artifacts by relevance.
3. Truncate the list to fit the budget.
4. Always include the pinned artifacts.
5. Return the `CompactedView`.

### 9.3 Pin Lookup Algorithm

The pin lookup algorithm works as follows.

1. Check the pin set for the content hash.
2. When the hash is pinned, return the
   artifact.
3. When the hash is not pinned, return nil.

## 10. Dependencies

### 10.1 Runtime Dependencies

- Go 1.23
- modernc.org/sqlite
- aios-core
- aios-persist

### 10.2 Module Dependencies

The aios-memory module depends on the following
internal modules.

- aios-core
- aios-persist

The aios-memory module is depended on by the
following internal modules.

- aios-runtime
- aios-planner
- aios-verify
- aios-cli
- aios-report
- aios-fanout

## 11. Deployment

The module is installed as a Go package. The
package path is
`github.com/<org>/aios/internal/memory`. The
default paths are configurable through the
`AIOS_DATA_DIR` environment variable.

## 12. Risks and Mitigations

| Risk | Impact | Probability | Mitigation |
| --- | --- | --- | --- |
| The memory grows without bound. | High | High | The memory is append-only. The `gc` operation compacts the raw layer. The developer can run `gc` on a schedule. |
| The retrieval is slow. | Medium | Medium | The index is in-memory. The query is O(n) over the index. The index is bounded by the budget. |
| The compaction drops important context. | High | Low | The compaction is observable. The developer can `aios memory show-compaction` to see what was dropped. The developer can `aios memory pin` to keep important artifacts. |
| The pin set grows without bound. | Medium | Medium | The pin set is editable. The developer can `aios memory unpin` to drop pins. |
| The cross-session memory is wrong. | High | Low | The memory is content-addressed. The developer can `aios memory show <hash>` to inspect any artifact. The developer can `aios memory prune` to drop wrong artifacts. |
