# aios-memory — Application Flow

## 1. Purpose

This document states the application flow for the
aios-memory module. The flow covers the full
lifecycle: the agent remembers, the developer
inspects, the memory persists, the agent
retrieves, the memory compacts.

## 2. Scope

The scope of this document is the aios-memory
module.

## 3. Reference Documents

- aios-memory PRD. See `docs/modules/aios-memory/PRD.md`.
- aios-memory TRD. See `docs/modules/aios-memory/TRD.md`.
- aios-cli APP-FLOW. See `docs/modules/aios-cli/APP-FLOW.md`.
- aios-runtime APP-FLOW. See `docs/modules/aios-runtime/APP-FLOW.md`.

## 4. Flows

### 4.1 Remember Flow

**Trigger** The agent finishes a unit of work.

**Steps**

1. The agent calls `memory.Remember(artifact)`.
2. The memory stores the artifact in the
   raw layer. The raw layer is content-
   addressed. The memory returns the content
   hash.
3. The memory adds the artifact to the
   indexed layer. The indexed layer is
   in-memory and persisted to disk.
4. The memory returns the content hash to the
   agent.

**Result** The artifact is in the memory. The
artifact is queryable. The artifact is
pinnable.

### 4.2 Query Flow

**Trigger** The agent needs a piece of context.

**Steps**

1. The agent calls
   `memory.Query(query)`.
2. The query runs against the indexed layer.
3. The query returns a list of content
   hashes. The list is sorted by the
   timestamp in descending order.
4. The agent calls
   `memory.Get(hash)` for each hash to load
   the artifacts.
5. The agent uses the artifacts to construct
   the prompt.

**Result** The agent has the context it needs.

### 4.3 Retrieval Flow

**Trigger** The orchestrator prepares the
prompt for a turn.

**Steps**

1. The orchestrator calls
   `retrieval.Retrieve(query, currentTaskID)`.
2. The retrieval runs the query against the
   indexed layer.
3. The retrieval ranks the matching artifacts.
4. The retrieval enforces the budget.
5. The retrieval returns a
   `CompactedView`.
6. The orchestrator adds the `CompactedView`
   to the prompt.

**Result** The prompt has the right context. The
prompt fits the budget.

### 4.4 Compaction Flow

**Trigger** The agent exceeds the context
budget.

**Steps**

1. The agent calls
   `compaction.Compact(query, currentTaskID,
   budget)`.
2. The compaction runs the query against the
   indexed layer.
3. The compaction truncates the result to
   fit the budget.
4. The compaction always includes the pinned
   artifacts.
5. The compaction returns a
   `CompactedView`.
6. The agent uses the `CompactedView` to
   construct the prompt.

**Result** The agent has a context that fits the
budget. The agent did not lose the important
context.

### 4.5 Pin Flow

**Trigger** The developer wants to keep an
artifact across all compacted views.

**Steps**

1. The developer runs
   `aios memory pin <hash>`.
2. The CLI calls `pinSet.Add(hash)`.
3. The pin set is saved to disk. The pin set
   id is updated.
4. The next retrieval and compaction
   includes the pinned artifact.

**Result** The artifact is always in the
compacted view. The artifact is never dropped.

### 4.6 Inspect Flow

**Trigger** The developer wants to see what the
agent has in its head.

**Steps**

1. The developer runs `aios memory show`.
2. The CLI loads the indexed layer.
3. The CLI prints the indexed layer. The
   output is JSON.
4. The developer pipes to `jq` and filters.

**Result** The developer sees the indexed layer.
The developer can filter and sort.

### 4.7 Grep Flow

**Trigger** The developer wants to find a
decision from last week.

**Steps**

1. The developer runs
   `aios memory grep <pattern>`.
2. The CLI runs a query against the indexed
   layer. The query pattern is the glob.
3. The CLI prints the matching artifacts. The
   output is JSON.
4. The developer pipes to `jq` and filters.

**Result** The developer finds the artifact. The
developer can `aios memory show <hash>` to see
the full content.

### 4.8 Diff Flow

**Trigger** The developer wants to see what
differed between two sessions.

**Steps**

1. The developer runs
   `aios memory diff <session_a> <session_b>`.
2. The CLI loads the indexed layers for both
   sessions.
3. The CLI computes the diff. The diff is the
   set of artifacts that are in one session
   but not the other.
4. The CLI prints the diff. The output is
   JSON.

**Result** The developer sees what differed.
The developer can `aios memory show <hash>` to
see the full content of any artifact.

### 4.9 Prune Flow

**Trigger** The developer wants to drop noise.

**Steps**

1. The developer runs
   `aios memory prune <hash>`.
2. The CLI marks the artifact as pruned in
   the indexed layer.
3. The indexed layer is saved to disk.
4. The next retrieval and compaction
   excludes the pruned artifact.

**Result** The artifact is not in the compacted
view. The raw layer still has the artifact. The
developer can `aios memory unprune` to restore.

### 4.10 GC Flow

**Trigger** The raw layer grows too large.

**Steps**

1. The developer runs `aios memory gc`.
2. The CLI loads the indexed layer.
3. The CLI compacts the raw layer. The
   compaction deduplicates by content hash.
4. The CLI rewrites the raw layer to a single
   file. The CLI updates the metadata index.
5. The raw layer is smaller. The indexed
   layer is unchanged.

**Result** The raw layer is smaller. The
indexed layer is unchanged. The retrieval and
compaction still work.

## 5. Sequence Diagram

```
Agent            Memory             Index           Retrieval
  |                |                 |                 |
  |--Remember----->|                 |                 |
  |                |--Put(raw)----->|                 |
  |                |--Add(idx)----->|                 |
  |                |<--hash----------|                 |
  |<--hash---------|                 |                 |
  |                |                 |                 |
  |  (later)       |                 |                 |
  |                |                 |                 |
  |--Retrieve----->|                 |                 |
  |                |--Query--------->|                 |
  |                |<--hashes--------|                 |
  |                |--Retrieve--------------------->    |
  |                |<--CompactedView------------------ |
  |<--View---------|                 |                 |
  |                |                 |                 |
  | (budget tight) |                 |                 |
  |                |                 |                 |
  |--Compact------>|                 |                 |
  |                |--Compact------------------------> |
  |                |<--CompactedView------------------ |
  |<--View---------|                 |                 |
```

## 6. State Diagram

```
                      +-----------+
                      |  empty    |
                      +-----------+
                            |
                       Remember
                            |
                            v
                      +-----------+
                      |  raw      |
                      |  indexed  |
                      +-----------+
                            |
                          Query
                            |
                            v
                      +-----------+
                      |  queried  |
                      +-----------+
                            |
                       Retrieve
                            |
                            v
                      +-----------+
                      |  retrieved|
                      +-----------+
                            |
                       Compact
                            |
                            v
                      +-----------+
                      |  compacted|
                      +-----------+
                            |
                          Pin
                            |
                            v
                      +-----------+
                      |  pinned   |
                      +-----------+
```

## 7. Failure Modes

| Failure | Detection | Recovery |
| --- | --- | --- |
| The raw layer is corrupted. | The store fails to read the file. | The store returns `ErrStoreCorrupt`. The CLI prints an error. The developer can `aios memory gc` to recover. |
| The indexed layer is corrupted. | The index fails to parse. | The index returns `ErrIndexCorrupt`. The CLI prints an error. The developer can `aios memory rebuild-index` to rebuild. |
| The pin set is corrupted. | The pin set fails to parse. | The pin set returns `ErrPinSetCorrupt`. The CLI prints an error. The developer can `aios memory reset-pins` to reset. |
| The retrieval times out. | The retrieval takes longer than 50 ms. | The retrieval returns `ErrTimeout`. The orchestrator falls back to the most recent compacted view. |
| The compaction drops important context. | The developer notices a missing artifact. | The developer can `aios memory show-compaction` to see what was dropped. The developer can `aios memory pin` to keep important artifacts. |

## 8. Performance Considerations

- The raw layer is content-addressed. The
  write throughput is bounded by the disk.
- The indexed layer is in-memory. The query
  latency is O(n) over the index. The index
  is bounded by the budget.
- The retrieval algorithm is a pure function.
  The retrieval can be parallelized.
- The compaction algorithm is a pure function.
  The compaction can be parallelized.
- The pin lookup is O(1). The pin set is a
  map.
- The `gc` operation is slow. The developer
  can run `gc` on a schedule.

## 9. Security Considerations

- The user scope is encrypted at rest through
  `crypto/aes`. The key is derived from the
  user's passphrase through `argon2`.
- The project scope is not encrypted. The
  project scope is `git`-trackable. The
  developer should not put secrets in the
  project scope.
- The session scope is in-memory only. The
  session scope dies when the session ends.
- The pin set is not encrypted. The pin set
  is a JSON object. The pin set does not
  contain the artifact content. The pin set
  contains the content hashes.
- The `aios memory` CLI is local. The CLI
  does not make network calls.
