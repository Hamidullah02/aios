# aios-watch — Application Flow

## 1. Purpose

This document describes the application flow for the
aios-watch module.

## 2. Scope

The scope of this document is the aios-watch module.
The module is a library.

## 3. Reference Documents

- aios-watch PRD. See `docs/modules/aios-watch/PRD.md`.
- aios-watch SRS. See `docs/modules/aios-watch/SRS.md`.
- aios-watch TRD. See `docs/modules/aios-watch/TRD.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Caller | The code that calls the channel or the tool. |

## 5. User Flow

The aios-watch module has no user-facing flow. The
stream channel and the file watcher update the CLI
output in real time.

## 6. System Flow

### 6.1 Stream Flow

The stream flow runs as follows.

1. The caller opens the stream channel.
2. The caller subscribes a callback.
3. The caller iterates the channel.
4. The channel calls the model router.
5. The model emits chunks.
6. The channel calls every subscriber.
7. The channel yields the chunks.
8. When the model is done, the channel yields the
   usage.

### 6.2 File Watch Flow

The file watch flow runs as follows.

1. The caller calls the file watcher tool.
2. The tool starts the watcher.
3. The watcher detects an event.
4. The tool filters the event through the ignore
   patterns.
5. The tool emits the matching event.
6. The tool returns the events.

### 6.3 Sub-agent Flow

The sub-agent flow runs as follows.

1. The caller calls the Task tool.
2. The tool builds the sub-agent.
3. The tool runs the sub-agent orchestrator.
4. The sub-agent returns the result.
5. The tool returns the result.

## 7. Error Flows

### 7.1 Watch Error

The watch error flow runs as follows.

1. The watcher fails.
2. The tool raises `WatchError`.
3. The caller catches the error.
4. The caller reports the error.

### 7.2 Sub-agent Error

The sub-agent error flow runs as follows.

1. The sub-agent fails.
2. The tool raises `SubAgentError`.
3. The caller catches the error.
4. The caller reports the error.

## 8. Sequence Diagrams

### 8.1 Stream Sequence

The stream sequence has the steps below.

1. The caller opens the stream channel.
2. The caller subscribes a callback.
3. The channel calls the model.
4. The model emits a chunk.
5. The channel calls the callback.
6. The channel yields the chunk.

### 8.2 Sub-agent Sequence

The sub-agent sequence has the steps below.

1. The caller calls the Task tool.
2. The tool builds the sub-agent.
3. The tool runs the sub-agent.
4. The sub-agent returns the result.
5. The tool returns the result.
