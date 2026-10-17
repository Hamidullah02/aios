# aios-watch — Technical Requirements Document

## 1. Purpose

This document states the technical approach for the
aios-watch module.

## 2. Scope

The scope of this document is the aios-watch module.
The module is a Go package.

## 3. Reference Documents

- aios-watch PRD. See `docs/modules/aios-watch/PRD.md`.
- aios-watch SRS. See `docs/modules/aios-watch/SRS.md`.
- aios-core TRD. See `docs/modules/aios-core/TRD.md`.
- aios-models TRD. See `docs/modules/aios-models/TRD.md`.
- aios-runtime TRD. See `docs/modules/aios-runtime/TRD.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| goroutines + channels + errgroup | The Go concurrency primitives for async I/O. |
| github.com/fsnotify/fsnotify | The cross-platform file-system event library. |

## 5. Architecture

The aios-watch module has three components.

- `aios.watch.stream` — the stream channel.
- `aios.watch.file_watcher` — the file watcher
  tool.
- `aios.watch.task` — the sub-agent Task tool.

The three components depend on aios-core, aios-models,
and aios-runtime. The components depend on each
other in this order. The stream channel depends on
the model router. The file watcher tool depends on
the github.com/fsnotify/fsnotify library and the ignore loader. The
Task tool depends on the orchestrator.

## 6. Components

### 6.1 Stream Channel

The stream channel wraps the model router. The
channel calls `router.stream`. The channel yields
the model chunks. The channel supports subscribers
through a list of async callbacks.

The channel closes when the model is done. The
channel closes on error. The channel raises the
error to the subscribers.

### 6.2 File Watcher Tool

The file watcher tool uses the `github.com/fsnotify/fsnotify` library.
The tool starts a watcher on the path. The tool
filters the events through the ignore patterns.
The tool returns the matching events.

The tool supports the `created`, `modified`,
`deleted`, and `moved` event types.

### 6.3 Sub-agent Task Tool

The Task tool spawns a sub-agent. The tool builds
a sub-agent orchestrator. The orchestrator runs in
its own context. The orchestrator returns the
result.

The tool supports the `general`, `explore`, `plan`,
and `code` agent types. Each type has a default
system prompt and a default tool set.

The tool supports parallel calls. The tool uses
`errgroup.Group (golang.org/x/sync/errgroup)` to run multiple calls in
parallel.

## 7. Data Structures

### 7.1 WatchEvent

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `path` | string | yes | The path of the file. |
| `event_type` | string | yes | The event type. |
| `timestamp` | string | yes | The ISO 8601 timestamp. |

### 7.2 AgentType

| Value | Description |
| --- | --- |
| general | The general-purpose sub-agent. |
| explore | The exploration sub-agent. The agent reads files and searches the codebase. The agent cannot edit files. |
| plan | The planning sub-agent. The agent plans a change. The agent cannot edit files. |
| code | The coding sub-agent. The agent edits files and runs commands. |

## 8. Interfaces

### 8.1 StreamChannel

```go
type StreamChannel:
    """The stream channel."""

    def __init__(self, model_router: ModelRouter) -> nil:
        """Create the stream channel."""

    def subscribe(
        self,
        callback: Callable[[ModelChunk], Awaitable[nil]],
    ) -> nil:
        """Subscribe to the stream.

        Args:
            callback: The callback to call on every chunk.
        """

    func stream(
        self,
        prompt: str,
        model: str,
    ) -> AsyncIterator[ModelChunk]:
        """Stream the model output.

        Args:
            prompt: The prompt to stream.
            model: The model id.

        Yields:
            The chunks of the model output.
        """
```

### 8.2 TaskTool

```go
type TaskTool:
    """The sub-agent task tool."""

    def __init__(self, orchestrator_factory: OrchestratorFactory) -> nil:
        """Create the task tool."""

    func run(self, args: dict, context: ToolContext) -> ToolResult:
        """Run the task tool.

        Args:
            args: The arguments of the call.
            context: The context for the call.

        Returns:
            The result of the call.
        """
```

## 9. Algorithms

### 9.1 Stream Algorithm

The stream algorithm works as follows.

1. Call `router.stream` with the prompt and the
   model.
2. For each chunk, call every subscriber.
3. Yield the chunk to the iterator.
4. When the model is done, yield the usage.
5. On error, close the channel and raise the
   error.

### 9.2 File Watch Algorithm

The file watch algorithm works as follows.

1. Start a `github.com/fsnotify/fsnotify` watcher on the path.
2. For each event, check the path against the
   ignore patterns.
3. When the path does not match an ignore pattern,
   emit the event.
4. Return the matching events.

### 9.3 Sub-agent Spawn Algorithm

The sub-agent spawn algorithm works as follows.

1. Build the sub-agent system prompt.
2. Build the sub-agent tool set.
3. Build a new orchestrator for the sub-agent.
4. Run the sub-agent orchestrator with the
   prompt.
5. Return the sub-agent result.

## 10. Dependencies

### 10.1 Runtime Dependencies

- Go 1.23
- github.com/fsnotify/fsnotify
- aios-core
- aios-models
- aios-runtime
- aios-memory

### 10.2 Module Dependencies

The aios-watch module depends on the following
internal modules.

- aios-core
- aios-models
- aios-runtime
- aios-memory

The aios-watch module is depended on by the
following internal modules.

- aios-runtime
- aios-cli

## 11. Deployment

The module compiles into the monorepo under `internal/<module>/`. The
package name is `github.com/<org>/aios/internal/watch`.

## 12. Risks and Mitigations

| Risk | Impact | Probability | Mitigation |
| --- | --- | --- | --- |
| The stream channel is slow. | Medium | Low | The channel is async. The channel buffers the chunks. |
| The file watcher misses an event. | Medium | Low | The watcher uses `github.com/fsnotify/fsnotify`. The watcher has a test case. |
| The sub-agent runs forever. | Medium | Low | The sub-agent has a timeout. The sub-agent has a budget. |
