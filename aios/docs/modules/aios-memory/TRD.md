# aios-memory — Technical Requirements Document

## 1. Purpose

This document states the technical approach for the
aios-memory module.

## 2. Scope

The scope of this document is the aios-memory module.
The module is a Go package.

## 3. Reference Documents

- aios-memory PRD. See `docs/modules/aios-memory/PRD.md`.
- aios-memory SRS. See `docs/modules/aios-memory/SRS.md`.
- aios-core TRD. See `docs/modules/aios-core/TRD.md`.
- aios-models TRD. See `docs/modules/aios-models/TRD.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| internal/pathspec | A gitignore-style pattern matcher (home-grown). |
| mtime | The last modification time of a file. |

## 5. Architecture

The aios-memory module has four components.

- `aios.memory.project_memory` — the project memory
  loader.
- `aios.memory.conventions` — the conventions
  loader.
- `aios.memory.ignore` — the ignore loader.
- `aios.memory.compaction` — the context
  compactor.

The four components depend on aios-core and
aios-models. The components depend on each other in
this order. The project memory loader has no
dependencies. The conventions loader has no
dependencies. The ignore loader has no
dependencies. The compactor depends on the model
router.

## 6. Components

### 6.1 Project Memory Loader

The loader reads the AGENTS.md files from the
workspace root to the file system root. The loader
merges the files. The closer file overrides the
parent file.

The loader caches the merged result. The loader
tracks the mtime of every file. The loader reloads
the result when any mtime changes.

### 6.2 Conventions Loader

The loader reads the CONVENTIONS.md file from the
workspace root. The loader reads the user-level
CONVENTIONS.md file from `~/.aios/`. The loader
merges the files. The project-level file overrides
the user-level file.

### 6.3 Ignore Loader

The loader reads the .aiosignore file. The loader
uses the `pathspec` library to parse the file. The
loader returns the patterns.

The loader provides the `match` method. The method
checks the path against the patterns. The method
returns true when the path matches.

### 6.4 Context Compactor

The compactor accepts a list of messages. The
compactor keeps the last N messages verbatim. The
compactor summarizes the older messages through the
model router.

The compactor uses a sliding window. The compactor
counts the tokens of the current context. When
the count exceeds the budget, the compactor
summarizes the older messages.

The compactor writes an audit row for every
compaction. The row includes the original token
count and the compacted token count.

## 7. Data Structures

### 7.1 ProjectMemory

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `project` | string | no | The project description. |
| `commands` | dict | no | The build, test, lint commands. |
| `conventions` | string | no | The code style rules. |
| `architecture` | string | no | The high-level architecture. |
| `notes` | string | no | The notes for the agent. |

### 7.2 IgnorePattern

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `pattern` | string | yes | The glob pattern. |
| `negation` | bool | no | Whether the pattern is a negation. |

## 8. Interfaces

### 8.1 ContextCompactor

```go
type ContextCompactor:
    """The context compactor."""

    def __init__(
        self,
        model_router: ModelRouter,
        keep_last: int = 5,
    ) -> nil:
        """Create the context compactor."""

    func compact(
        self,
        messages: list[Message],
        token_budget: int,
    ) -> list[Message]:
        """Compact the messages.

        Args:
            messages: The messages to compact.
            token_budget: The maximum token count.

        Returns:
            The compacted messages.
        """
```

## 9. Algorithms

### 9.1 Memory Merge Algorithm

The memory merge algorithm works as follows.

1. Start with the parent file content.
2. Read the child file content.
3. For each section in the child file, override
   the section in the parent content.
4. Return the merged content.

### 9.2 Compaction Algorithm

The compaction algorithm works as follows.

1. Count the tokens of the messages.
2. When the count is within the budget, return the
   messages.
3. Split the messages into the kept set and the
   older set.
4. Build the compaction prompt with the older
   messages.
5. Call the model router.
6. Replace the older messages with the summary.
7. Return the compacted messages.

## 10. Dependencies

### 10.1 Runtime Dependencies

- Go 1.23
- pathspec
- aios-core
- aios-models

### 10.2 Module Dependencies

The aios-memory module depends on the following
internal modules.

- aios-core
- aios-models

The aios-memory module is depended on by the
following internal modules.

- aios-runtime
- aios-cli
- aios-search
- aios-tools
- aios-edit

## 11. Deployment

The module compiles into the monorepo under `internal/<module>/`. The
package name is `github.com/<org>/aios/internal/memory`. The user-level
conventions file is at `~/.aios/CONVENTIONS.md`.

## 12. Risks and Mitigations

| Risk | Impact | Probability | Mitigation |
| --- | --- | --- | --- |
| The AGENTS.md file is too large. | Medium | Medium | The loader has a size limit. The loader raises an error on overflow. |
| The compaction changes the meaning. | High | Low | The compactor keeps the last N messages verbatim. The compactor has a test case. |
| The ignore pattern blocks a needed file. | Medium | Low | The loader supports the `!` negation. The loader has a test case. |
