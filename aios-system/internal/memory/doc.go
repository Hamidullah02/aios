// Package memory is the content-addressed memory and
// context manager for aios.
//
// The package treats context as a query against a
// content-addressed knowledge base, not as a fixed-size
// buffer of recent messages. The agent remembers an
// artifact with a content hash; the agent queries the
// memory for the slice it needs; the memory returns the
// slice ranked by relevance, bounded by the context
// budget.
//
// The package provides seven things:
//
//   - Store: the content-addressed raw layer. The store
//     is an append-only file system. The file name is
//     the content hash; the file content is the artifact
//     payload.
//   - Index: the typed indexed layer. The index is a
//     JSON file at <path>/index.json. Artifacts are
//     categorized as task, file, decision, observation,
//     tool_result, or model_output.
//   - Scope: the three scopes — session, project, user.
//     The session scope is in-memory only. The project
//     scope is git-trackable at <path>/project/. The
//     user scope lives at <home>/.aios/memory/ and is
//     optionally encrypted at rest through AES-GCM.
//   - Query: the typed query language. A query filters
//     by kinds, scopes, tags, pattern, since, and limit
//     with an AND or OR operator.
//   - Retrieval: the pure-function retrieval algorithm.
//     The algorithm runs a query against the index,
//     ranks the matches, and enforces a budget.
//   - Compaction: the pure-function compaction
//     algorithm. The algorithm truncates the indexed
//     layer to a budget and is observable: the
//     CompactedView reports what was kept and what was
//     dropped.
//   - PinSet: the content-addressed pin set. The pin
//     set id is the SHA-256 of the pin set payload.
//     Pinned artifacts are always in the compacted view.
//
// The package depends only on aios-core (for the
// canonical-JSON encoder) and the Go standard library.
// The package is depended on by aios-runtime,
// aios-planner, aios-verify, aios-report, aios-fanout,
// and aios-cli.
package memory
