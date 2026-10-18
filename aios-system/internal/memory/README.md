# aios-memory

The aios-memory module is the content-addressed
memory and context manager for the agent. The
package treats context as a query against a
content-addressed knowledge base, not as a
fixed-size buffer of recent messages.

The package is a single Go package at
`internal/memory/` and depends only on the Go
standard library and `crypto/aes` (for the
optional user-scope encryption). No new
third-party deps.

## Public surface

- `Artifact`, `ArtifactKind`, `Scope`, `Operator` —
  the typed data contracts. The Artifact is the
  unit of memory; the others are enums.
- `NewArtifact`, `NewArtifactWithMeta` — the typed
  constructors. Both fill the content hash and the
  timestamp.
- `HashBytes` — the SHA-256 hex content hash.
- `Store` — the content-addressed raw layer. The
  store is append-only; the file name is the
  content hash. `Put` / `Get` / `Has` / `List` /
  `Size` / `Count` / `Compact`.
- `Index` — the typed indexed layer. The index is
  a JSON file at `<path>/index.json` with
  `Add` / `Get` / `Has` / `Prune` / `Unprune` /
  `Size` / `All` / `ID`. Saves are atomic.
- `Query` — the typed query language. Filters by
  kinds, scopes, tags, glob pattern, since
  timestamp, and limit. Combines with AND or OR.
  `RunIndex` applies the query to an entry set.
- `PinSet` — the content-addressed pin set. The
  pin set id is the SHA-256 of the pin set
  payload. `Add` / `Remove` / `Has` / `Entry` /
  `Size` / `List` / `ID`.
- `Retrieval` — the pure-function retrieval
  algorithm. `Retrieve(query, currentTaskID)`
  returns a `CompactedView`.
- `Compaction` — the pure-function compaction
  algorithm. `Compact(query, currentTaskID,
  budgetTokens)` and `CompactForTask(taskID,
  recentN)` return a `CompactedView`.
- `CompactedView` — the retrieval/compaction
  result. Carries the kept artifacts, the total
  tokens, the truncation flag, the dropped list,
  the pin set id, and the index id.
- `Memory` — the top-level container that owns
  the per-scope views and the pin set.
  `Remember` / `Get` / `Has` / `Query` /
  `ResetSession` / `Session` / `Project` / `User`
  / `PinSet`.
- `ScopeView` — the per-scope (Store, Index)
  pair. `Scope` / `Store` / `Index` / `putLocked`.
- `Encryptor` — the symmetric-encryption
  interface. `Encrypt` / `Decrypt`.
- `AESGCMEncryptor` — the production AES-256-GCM
  encryptor. `Encrypt` / `Decrypt` round-trip
  through `nonce || ciphertext`.
- `NoopEncryptor` — the no-op encryptor used by
  tests.
- `ErrNotFound`, `ErrIndexCorrupt`, `ErrPinSetCorrupt`,
  `ErrInvalidArtifact`, `ErrStoreCorrupt`,
  `ErrBudgetTooSmall` — sentinel errors. The
  callers match with `errors.Is`.
- `MemoryError`, `ErrBudgetTooSmallError` — typed
  errors with `Op` and `Reason` fields.

## Three layers

- **Raw layer** — the `Store`. The file name is
  the content hash; the file content is the
  artifact payload. The store has no `Delete`
  method (append-only).
- **Indexed layer** — the `Index`. A JSON object
  keyed by content hash; the value is the
  `IndexEntry` (kind, scope, session id, task
  id, timestamp, tags, pruned flag).
- **Compacted layer** — the `CompactedView`. A
  transient, budget-bounded view of the indexed
  layer, computed by `Retrieval.Retrieve` or
  `Compaction.Compact`. The view is not stored
  on disk.

## Three scopes

- **session** — in-memory only. The store and
  index are dropped on `ResetSession`.
- **project** — at `<DataDir>/memory/`. Git-
  trackable. The store path is
  `<DataDir>/memory/raw`; the index file is
  `<DataDir>/memory/index/index.json`; the pin
  set file is `<DataDir>/memory/pins.json`.
- **user** — at `<Home>/memory/`. The user scope
  is optionally encrypted at rest through
  AES-GCM. The encryption key is supplied
  through `MemoryConfig.UserEncryptor`; the
  library does not derive the key from a
  passphrase (the CLI is the right layer for
  that).

## Determinism

The retrieval and compaction are pure functions
of their inputs. The pin set id is the SHA-256
of the pin set payload. The index id is the
SHA-256 of the canonical JSON of the index.
Two retrievals with the same input return the
same `CompactedView`.

## Quality gates

- `go test ./internal/memory/...` ok.
- `go test -cover ./internal/memory/...` → 80.2%
  of statements. Public functions are at 100%
  except a few error-path branches that the
  tests do not exercise (e.g. corrupt pin set
  recovery, partial-pinned-budget branch).
- `go vet ./internal/memory/...` clean.
- `gofmt -l internal/memory/` empty.

## Substitutions vs the doc set

- **modernc.org/sqlite for the metadata index →
  plain files.** The doc set's "metadata index
  maps the content hash to the file path" is
  exactly the directory listing of a content-
  addressed file system. The directory walk IS
  the metadata index. SQLite is not needed.
- **User-scope encryption via AES-GCM with a
  passphrase-derived key (argon2) → AES-GCM
  with a caller-supplied key.** The library
  does not invent a passphrase flow. The CLI
  prompts the operator for a passphrase and
  derives the key through argon2 (or loads it
  from the OS keychain). The library receives
  a 32-byte key through `MemoryConfig.UserEncryptor`.
- **`aios memory` CLI → deferred to aios-cli.**
  The library provides the operations; the CLI
  wires them to `aios memory {show, grep, pin,
  unpin, prune, diff, gc, stats}`.
- **Doc set filenames `.py` → `.go`** per the
  project's language choice.

## Tests

The tests use `t.TempDir()` for the project and
user roots, a fixed clock for the timestamp
helpers, and `t.Setenv` for the env-var
fallback. The memory tests do not depend on
`/bin/sh` or any other shell.

The retrieval and compaction tests pass a
`storeResolver` that joins the index with the
project store so the token budget is computed
from the real payload. The default resolver
(`entryResolverFromIndex`) joins the index
alone and returns artifacts with empty content;
the default is intended for tests that do not
need the token budget.
