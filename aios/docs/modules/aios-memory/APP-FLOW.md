# aios-memory — Application Flow

## 1. Purpose

This document describes the application flow for the
aios-memory module.

## 2. Scope

The scope of this document is the aios-memory module.
The module is a library.

## 3. Reference Documents

- aios-memory PRD. See `docs/modules/aios-memory/PRD.md`.
- aios-memory SRS. See `docs/modules/aios-memory/SRS.md`.
- aios-memory TRD. See `docs/modules/aios-memory/TRD.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Caller | The code that calls the loader. |

## 5. User Flow

The aios-memory module has no user-facing flow.

## 6. System Flow

### 6.1 Project Memory Load Flow

The load flow runs as follows.

1. The caller calls the loader.
2. The loader checks the cache.
3. When the cache is fresh, the loader returns
   the cached value.
4. When the cache is stale, the loader reads the
   files.
5. The loader merges the files.
6. The loader updates the cache.
7. The loader returns the merged value.

### 6.2 Compaction Flow

The compaction flow runs as follows.

1. The caller calls the compactor.
2. The compactor counts the tokens.
3. When the count is within the budget, the
   compactor returns the messages.
4. When the count exceeds the budget, the
   compactor splits the messages.
5. The compactor summarizes the older messages.
6. The compactor returns the compacted messages.

## 7. Error Flows

### 7.1 Memory Parse Error

The memory parse error flow runs as follows.

1. The file is invalid.
2. The loader raises `MemoryParseError`.
3. The caller catches the error.
4. The caller reports the error.

### 7.2 Compaction Error

The compaction error flow runs as follows.

1. The model call fails.
2. The compactor raises `CompactionError`.
3. The caller catches the error.
4. The caller uses the messages without
   compaction.

## 8. Sequence Diagrams

### 8.1 Load Sequence

The load sequence has the steps below.

1. The caller calls the loader.
2. The loader reads the files.
3. The loader merges the files.
4. The loader returns the merged value.

### 8.2 Compaction Sequence

The compaction sequence has the steps below.

1. The caller calls the compactor.
2. The compactor counts the tokens.
3. The compactor splits the messages.
4. The compactor calls the model.
5. The compactor returns the compacted messages.
