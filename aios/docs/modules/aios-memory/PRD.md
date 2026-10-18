# aios-memory — Product Requirements Document

## 1. Purpose

This document states the product requirements for the
aios-memory module. The module is the
content-addressed memory and context manager for
the agent. The module gives the agent a memory
that is bigger, better, and more honest than the
competition.

The competition treats context as a fixed-size
buffer of the most recent messages. The buffer
fills up, gets truncated, the agent loses the
goal, the agent re-reads the same file five
times, the agent forgets what it decided an hour
ago, and the agent dies when the buffer is full.

aios-memory treats context as a query against a
content-addressed knowledge base. The agent
queries the memory for the artifacts it needs.
The memory returns the right slice, ranked by
relevance, bounded by the context budget. The
memory persists across sessions. The memory is
observable. The memory is editable. The memory
is the moat.

## 2. Scope

The aios-memory module is in scope for this
document.

- The content-addressed storage.
- The typed index.
- The three scopes: session, project, user.
- The three layers: raw, indexed, compacted.
- The query language.
- The retrieval algorithm.
- The compaction algorithm.
- The pinning mechanism.
- The `aios memory` CLI.

The aios-memory module is not in scope for this
document.

- The model prompt. The prompt is in
  aios-planner.
- The conversation history. The history is in
  aios-persist.
- The file system. The file system is in
  aios-tools.

## 3. Reference Documents

- aios-core PRD. See `docs/modules/aios-core/PRD.md`.
- aios-persist PRD. See `docs/modules/aios-persist/PRD.md`.
- aios-runtime PRD. See `docs/modules/aios-runtime/PRD.md`.
- aios-planner PRD. See `docs/modules/aios-planner/PRD.md`.
- aios-cli PRD. See `docs/modules/aios-cli/PRD.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Artifact | A typed object stored in the memory. Every artifact has a content hash. |
| Raw layer | The unindexed content-addressed storage. Every artifact is here. Nothing is deleted. |
| Indexed layer | A typed view of the raw layer. Artifacts are categorized as tasks, files, decisions, observations. |
| Compacted layer | A lossy view of the indexed layer. Regenerated on demand. Sized to fit the context budget. |
| Session scope | The memory that dies when the session ends. |
| Project scope | The memory that persists across sessions in the same project. |
| User scope | The memory that persists across projects. |
| Pin | A directive that forces an artifact to be included in every compacted view. |
| Query | A typed request for artifacts. The query has a type, filters, and a budget. |
| Retrieval | The act of running a query and producing a compacted view. |
| Compaction | The act of reducing the indexed layer to a compacted view. |
| Fingerprint | The SHA-256 of (artifact type, scope, content). Used for dedup and lookup. |

## 5. Determinism

The whole point of aios is to make complex coding and large systems deterministic. The determinism stance of this module is **Highly deterministic**.

The memory is content addressed. The retrieval algorithm is a pure function of the indexed layer, the query, and the pin set. The compaction algorithm is a pure function of the indexed layer, the budget, and the pin set. The memory has no clock and no random source.

The module provides the following determinism guarantees.

- Every artifact is identified by its content hash. Two artifacts with the same content have the same hash.
- The retrieval algorithm is a pure function. Two queries with the same input return the same compacted view.
- The compaction algorithm is a pure function. Two compactions with the same input return the same compacted view.
- The pin set is content addressed. The pin set id is the SHA-256 of the pin set payload.
- The memory is append-only. Artifacts are never deleted. Compaction views are regenerated, not stored.

## 6. User Personas

The user persona is the developer who runs the
agent. The developer wants the agent to remember
what it learned yesterday. The developer wants to
see what the agent has in its head. The developer
wants to pin the parts the agent must not
forget. The developer wants to drop the parts the
agent should forget.

## 7. User Stories

A developer wants the agent to remember a
decision. The agent calls `memory.remember` with
the decision. The memory stores the decision in
the project scope. The next session retrieves
the decision through a query.

A developer wants to see what the agent has in
its head. The developer runs `aios memory show`.
The CLI prints the current compacted view. The
CLI prints the indexed layer. The CLI prints the
raw layer sizes.

A developer wants the agent to never forget a
fact. The developer runs `aios memory pin <hash>`.
The memory adds the artifact to the pin set. The
compacted view always includes pinned artifacts.

A developer wants the agent to drop noise. The
developer runs `aios memory prune <hash>`. The
memory marks the artifact as pruned. The
compacted view excludes pruned artifacts.

A developer wants the agent to find a decision
from last week. The developer runs `aios memory
grep <pattern>`. The CLI returns the matching
artifacts. The CLI shows the session, the task,
and the content hash.

A developer wants to diff two sessions. The
developer runs `aios memory diff <session_a>
<session_b>`. The CLI returns the artifacts that
differ.

## 8. Functional Requirements Summary

The aios-memory module provides the following
functions.

- The content-addressed storage.
- The typed index.
- The three scopes: session, project, user.
- The three layers: raw, indexed, compacted.
- The query language.
- The retrieval algorithm.
- The compaction algorithm.
- The pinning mechanism.
- The `aios memory` CLI.

## 9. Non-Functional Requirements Summary

- The module runs on Go 1.23.
- The module depends on aios-core and
  aios-persist.
- The module has 100 percent test coverage for
  the public functions.
- The retrieval algorithm has a maximum latency
  of 50 milliseconds for a 10,000-artifact
  indexed layer.
- The compaction algorithm has a maximum
  latency of 100 milliseconds for a 10,000-
  artifact indexed layer.

## 10. Success Criteria

- Every artifact is identified by its content
  hash. The developer can `aios memory show
  <hash>` and see the artifact.
- The retrieval algorithm returns the same
  compacted view for the same query, the same
  indexed layer, and the same pin set.
- The compaction algorithm returns the same
  compacted view for the same indexed layer,
  the same budget, and the same pin set.
- The pin set is honored. Pinned artifacts are
  always in the compacted view.
- The memory persists across sessions. The
  developer can `aios memory grep` and find
  decisions from previous sessions.
- The `aios memory` CLI is a developer tool. The
  output is grep-friendly JSON. The developer
  can pipe to `jq`.

## 11. Out of Scope

- The vector embeddings. The memory is
  content-addressed, not embedding-based. The
  developer can build a vector index on top of
  the raw layer if they want.
- The semantic search. The memory is
  keyword-based and exact-match. The developer
  can build a semantic index on top of the raw
  layer if they want.
- The cross-host replication. The memory is
  local. The developer can `git`-track the
  project scope and push it to a remote.
- The garbage collection. The memory is
  append-only. The developer can `aios memory
  gc` to compact the raw layer to a single
  file.
