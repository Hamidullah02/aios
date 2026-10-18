# aios-memory — Implementation Plan

## 1. Purpose

This document states the implementation plan for
the aios-memory module. The plan covers the
order in which the components are built, the
tests, the milestones, and the deliverables.

## 2. Scope

The scope of this document is the aios-memory
module.

## 3. Reference Documents

- aios-memory PRD. See `docs/modules/aios-memory/PRD.md`.
- aios-memory SRS. See `docs/modules/aios-memory/SRS.md`.
- aios-memory TRD. See `docs/modules/aios-memory/TRD.md`.
- aios-memory APP-FLOW. See `docs/modules/aios-memory/APP-FLOW.md`.
- aios-memory WIREFRAMES. See `docs/modules/aios-memory/WIREFRAMES.md`.
- Project IMPLEMENTATION-PLAN. See
  `docs/global/IMPLEMENTATION-PLAN.md`.

## 4. Build Order

The build order follows the dependency graph in
the TRD. The store is built first. The index is
built second. The scope, query, retrieval,
compaction, and pin are built third. The CLI is
built last.

### 4.1 Milestone 1 — Store

**Goal** The content-addressed storage works.

**Tasks**

1. Implement the `Store` type.
2. Implement the `Put`, `Get`, and `Has`
   methods.
3. Implement the metadata index with
   `modernc.org/sqlite`.
4. Implement the file system layout. The file
   name is the content hash. The file content
   is the artifact payload.
5. Write the unit tests for the store.

**Deliverable** The store is working. The store
passes the unit tests. The store is content-
addressed on disk.

### 4.2 Milestone 2 — Index

**Goal** The typed index works.

**Tasks**

1. Implement the `Index` type.
2. Implement the `Add`, `Get`, `Query`, and
   `Save` methods.
3. Implement the index file format. The file
   is a JSON object.
4. Implement the atomic save through a
   `rename` syscall.
5. Write the unit tests for the index.

**Deliverable** The index is working. The index
passes the unit tests. The index is persisted
to disk.

### 4.3 Milestone 3 — Scopes

**Goal** The three scopes work.

**Tasks**

1. Implement the `Scope` type.
2. Implement the `session` scope. The scope
   is in-memory only.
3. Implement the `project` scope. The scope
   is at `var/memory/project/`. The scope is
   `git`-trackable.
4. Implement the `user` scope. The scope is
   at `~/.aios/memory/`. The scope is
   encrypted at rest.
5. Write the unit tests for the scopes.

**Deliverable** The three scopes are working. The
scopes pass the unit tests.

### 4.4 Milestone 4 — Query

**Goal** The query language works.

**Tasks**

1. Implement the `Query` type.
2. Implement the query parser. The parser
   accepts the query as a value type.
3. Implement the query runner. The runner
   runs the query against the in-memory index.
4. Write the unit tests for the query.

**Deliverable** The query is working. The query
passes the unit tests.

### 4.5 Milestone 5 — Retrieval

**Goal** The retrieval algorithm works.

**Tasks**

1. Implement the `Retrieval` type.
2. Implement the `Retrieve` method.
3. Implement the ranking priority. The
   ranking priority is listed in the TRD.
4. Implement the budget enforcement.
5. Write the unit tests for the retrieval.

**Deliverable** The retrieval is working. The
retrieval passes the unit tests. The retrieval
is a pure function.

### 4.6 Milestone 6 — Compaction

**Goal** The compaction algorithm works.

**Tasks**

1. Implement the `Compaction` type.
2. Implement the `Compact` method.
3. Implement the task-scoped truncation.
4. Implement the observability. The
   `CompactedView` includes what was kept
   and what was dropped.
5. Write the unit tests for the compaction.

**Deliverable** The compaction is working. The
compaction passes the unit tests. The compaction
is a pure function.

### 4.7 Milestone 7 — Pin

**Goal** The pinning mechanism works.

**Tasks**

1. Implement the `PinSet` type.
2. Implement the `Add`, `Remove`, and `Has`
   methods.
3. Implement the pin set file format. The
   file is a JSON object.
4. Implement the pin set id. The id is the
   SHA-256 of the pin set payload.
5. Write the unit tests for the pin set.

**Deliverable** The pin set is working. The pin
set passes the unit tests. The pin set is
content-addressed.

### 4.8 Milestone 8 — CLI

**Goal** The `aios memory` CLI works.

**Tasks**

1. Implement the `aios memory` subcommand.
2. Implement the `show` subcommand.
3. Implement the `grep` subcommand.
4. Implement the `pin` and `unpin`
   subcommands.
5. Implement the `prune` subcommand.
6. Implement the `diff` subcommand.
7. Implement the `gc` subcommand.
8. Implement the `stats` subcommand.
9. Write the unit tests for the CLI.
10. Write the integration tests for the CLI.

**Deliverable** The CLI is working. The CLI
passes the unit tests. The CLI passes the
integration tests. The output is grep-friendly
JSON.

## 5. Test Plan

### 5.1 Unit Tests

The unit tests cover the public functions. The
unit tests are in the same package as the
implementation. The unit tests use the standard
`go test` framework.

| Test | Requirement | Description |
| --- | --- | --- |
| `TestStorePut` | REQ-MEM-STO-003 | Verify that two puts of the same content return the same hash. |
| `TestStoreGet` | REQ-MEM-STO-004 | Verify that the `Get` method returns the artifact. |
| `TestStoreGetNotFound` | REQ-MEM-STO-004 | Verify that the `Get` method returns `ErrNotFound`. |
| `TestStoreHas` | REQ-MEM-STO-004 | Verify that the `Has` method returns true. |
| `TestIndexAdd` | REQ-MEM-IDX-003 | Verify that the `Add` method is idempotent. |
| `TestIndexQuery` | REQ-MEM-IDX-003 | Verify that the `Query` method returns the matching artifacts. |
| `TestIndexSave` | REQ-MEM-IDX-005 | Verify that the `Save` method is atomic. |
| `TestScopeSession` | REQ-MEM-SCO-002 | Verify that the session scope dies when the session ends. |
| `TestScopeProject` | REQ-MEM-SCO-003 | Verify that the project scope persists across sessions. |
| `TestScopeUser` | REQ-MEM-SCO-004 | Verify that the user scope is encrypted at rest. |
| `TestQueryKinds` | REQ-MEM-QUE-001 | Verify that the query filters by kind. |
| `TestQueryScopes` | REQ-MEM-QUE-001 | Verify that the query filters by scope. |
| `TestQueryTags` | REQ-MEM-QUE-001 | Verify that the query filters by tags. |
| `TestQueryPattern` | REQ-MEM-QUE-001 | Verify that the query matches the glob pattern. |
| `TestQuerySince` | REQ-MEM-QUE-001 | Verify that the query filters by timestamp. |
| `TestQueryLimit` | REQ-MEM-QUE-001 | Verify that the query truncates to the limit. |
| `TestRetrievalPinned` | REQ-MEM-RET-003 | Verify that pinned artifacts are always included. |
| `TestRetrievalCurrentTask` | REQ-MEM-RET-003 | Verify that artifacts from the current task are always included. |
| `TestRetrievalBudget` | REQ-MEM-RET-004 | Verify that the retrieval truncates to the budget. |
| `TestRetrievalPure` | REQ-MEM-RET-005 | Verify that two retrievals with the same input return the same output. |
| `TestCompactionBudget` | REQ-MEM-COM-003 | Verify that the compaction truncates to the budget. |
| `TestCompactionTaskScoped` | REQ-MEM-COM-004 | Verify that the task-scoped truncation works. |
| `TestCompactionObservable` | REQ-MEM-COM-005 | Verify that the compaction shows what was kept and what was dropped. |
| `TestCompactionPure` | REQ-MEM-COM-006 | Verify that two compactions with the same input return the same output. |
| `TestPinSetAdd` | REQ-MEM-PIN-001 | Verify that the `Add` method is idempotent. |
| `TestPinSetRemove` | REQ-MEM-PIN-001 | Verify that the `Remove` method works. |
| `TestPinSetID` | REQ-MEM-PIN-002 | Verify that the pin set id is the SHA-256 of the payload. |
| `TestPinHonored` | REQ-MEM-PIN-004 | Verify that pinned artifacts are always in the compacted view. |

### 5.2 Integration Tests

The integration tests cover the CLI. The
integration tests run the CLI as a subprocess.
The integration tests use the standard `go test`
framework.

| Test | Requirement | Description |
| --- | --- | --- |
| `TestCLIShow` | REQ-MEM-CLI-002 | Verify that `aios memory show` prints the compacted view. |
| `TestCLIGrep` | REQ-MEM-CLI-003 | Verify that `aios memory grep` returns the matching artifacts. |
| `TestCLIPin` | REQ-MEM-CLI-004 | Verify that `aios memory pin` adds the artifact to the pin set. |
| `TestCLIUnpin` | REQ-MEM-CLI-004 | Verify that `aios memory unpin` removes the artifact from the pin set. |
| `TestCLIPrune` | REQ-MEM-CLI-005 | Verify that `aios memory prune` marks the artifact as pruned. |
| `TestCLIDiff` | REQ-MEM-CLI-006 | Verify that `aios memory diff` returns the differing artifacts. |
| `TestCLIGC` | REQ-MEM-CLI-007 | Verify that `aios memory gc` compacts the raw layer. |
| `TestCLIStats` | REQ-MEM-CLI-008 | Verify that `aios memory stats` prints the statistics. |

### 5.3 Determinism Tests

The determinism tests verify the determinism
guarantees in the PRD. The determinism tests are
the highest-priority acceptance gate.

| Test | Requirement | Description |
| --- | --- | --- |
| `TestDeterminismContentHash` | REQ-MEM-DET-101 | Verify that the content hash is deterministic. Two artifacts with the same content have the same hash. |
| `TestDeterminismRetrieval` | REQ-MEM-DET-102 | Verify that the retrieval is deterministic. Two retrievals with the same input return the same output. |
| `TestDeterminismCompaction` | REQ-MEM-DET-103 | Verify that the compaction is deterministic. Two compactions with the same input return the same output. |
| `TestDeterminismPinSet` | REQ-MEM-DET-104 | Verify that the pin set id is deterministic. Two pin sets with the same content have the same id. |
| `TestDeterminismReplay` | REQ-MEM-DET-105 | Verify that the memory can be replayed. The replay produces the same compacted views. |

### 5.4 Coverage Requirements

The test suite shall have 100 percent coverage
for the public functions. The coverage is
measured with `go test -cover`.

## 6. Milestones Summary

| Milestone | Goal | Duration | Dependencies |
| --- | --- | --- | --- |
| M1 | Store | 1 week | aios-core, aios-persist |
| M2 | Index | 1 week | M1 |
| M3 | Scopes | 1 week | M2 |
| M4 | Query | 1 week | M2 |
| M5 | Retrieval | 1 week | M2, M4, M7 |
| M6 | Compaction | 1 week | M5, M7 |
| M7 | Pin | 1 week | M1 |
| M8 | CLI | 1 week | M5, M6 |

## 7. Dependencies

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

## 8. Risks and Mitigations

| Risk | Impact | Probability | Mitigation |
| --- | --- | --- | --- |
| The memory grows without bound. | High | High | The memory is append-only. The `gc` operation compacts the raw layer. The developer can run `gc` on a schedule. |
| The retrieval is slow. | Medium | Medium | The index is in-memory. The query is O(n) over the index. The index is bounded by the budget. |
| The compaction drops important context. | High | Low | The compaction is observable. The developer can `aios memory show-compaction` to see what was dropped. The developer can `aios memory pin` to keep important artifacts. |
| The pin set grows without bound. | Medium | Medium | The pin set is editable. The developer can `aios memory unpin` to drop pins. |
| The cross-session memory is wrong. | High | Low | The memory is content-addressed. The developer can `aios memory show <hash>` to inspect any artifact. The developer can `aios memory prune` to drop wrong artifacts. |

## 9. Acceptance Criteria

1. Every REQ in the SRS has a test case.
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
10. The determinism tests pass.
