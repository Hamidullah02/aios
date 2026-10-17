# aios-tools — Technical Requirements Document

## 1. Purpose

This document states the technical approach for the
aios-tools module.

## 2. Scope

The scope of this document is the aios-tools module.
The module is a Go package.

## 3. Reference Documents

- aios-tools PRD. See `docs/modules/aios-tools/PRD.md`.
- aios-tools SRS. See `docs/modules/aios-tools/SRS.md`.
- aios-core TRD. See `docs/modules/aios-core/TRD.md`.
- aios-policy TRD. See `docs/modules/aios-policy/TRD.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Glob | A pattern for path matching with wildcards. |
| Atomic write | A write that creates a temp file and renames the temp file to the target. |

## 5. Architecture

The aios-tools module has six components.

- `aios.tools.interface` — the tool interface.
- `internal/tools/bus` — the tool bus.
- `internal/tools/read_file` — the file read tool.
- `internal/tools/write_file` — the file write tool.
- `aios.tools.search_files` — the file search tool.
- `aios.tools.run_shell` — the shell run tool.

The six components depend on aios-core, aios-policy,
and aios-sandbox. The components depend on each
other in this order. The interface has no
dependencies. The bus depends on the interface. The
tools depend on the interface.

## 6. Components

### 6.1 interface

The interface component defines the `Tool` interface.
The interface has the `name` property and the `run`
method. The `run` method accepts the arguments and
the context. The `run` method returns the result.

### 6.2 Tool Bus

The bus component dispatches tool calls. The bus
looks up the tool by name. The bus calls the policy
gateway. The bus calls the tool. The bus writes the
audit row.

### 6.3 Read File

The read file component reads a file. The component
validates the path. The component returns the file
content. The component supports line offsets and
line limits.

### 6.4 Write File

The write file component writes a file. The
component validates the path. The component creates
a temp file. The component writes the content to
the temp file. The component renames the temp file
to the target.

### 6.5 Search Files

The search files component searches files. The
component uses the `path/filepath (stdlib)` module to walk the
directory. The component matches the paths against
the glob pattern. The component returns the
matching paths.

### 6.6 Run Shell

The run shell component runs a shell command. The
component uses the sandbox runner. The component
returns the exit code, the stdout, and the stderr.

## 7. Data Structures

### 7.1 ToolCall

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `id` | string | yes | The unique identifier of the call. |
| `tool` | string | yes | The name of the tool. |
| `args` | dict | yes | The arguments of the call. |
| `actor` | string | yes | The actor that made the call. |
| `task_id` | string | yes | The task id. |

### 7.2 ToolResult

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `id` | string | yes | The id of the call. |
| `ok` | bool | yes | Whether the call succeeded. |
| `output` | any | no | The output of the call. |
| `error` | string | no | The error message. |
| `duration_ms` | int | yes | The duration in milliseconds. |

## 8. Interfaces

### 8.1 ToolBus

```go
type ToolBus:
    """The tool bus."""

    def __init__(
        self,
        tools: dict[str, Tool],
        policy: PolicyGateway,
        audit_log: AuditLog,
    ) -> nil:
        """Create the tool bus."""

    func dispatch(
        self,
        call: ToolCall,
    ) -> ToolResult:
        """Dispatch a tool call.

        Args:
            call: The tool call to dispatch.

        Returns:
            The result of the call.
        """
```

### 8.2 Tool

```go
type Tool(interface):
    """The tool interface."""

    @property
    def name(self) -> str:
        """The name of the tool."""

    func run(self, args: dict, context: ToolContext) -> ToolResult:
        """Run the tool.

        Args:
            args: The arguments of the call.
            context: The context for the call.

        Returns:
            The result of the call.
        """
```

## 9. Algorithms

### 9.1 Path Validation

The path validation algorithm works as follows.

1. Resolve the path to an absolute path.
2. Resolve the workspace root to an absolute path.
3. When the path does not start with the workspace
   root, raise `PathNotAllowedError`.
4. Return the resolved path.

### 9.2 Atomic Write

The atomic write algorithm works as follows.

1. Create a temp file in the same directory as the
   target.
2. Write the content to the temp file.
3. Flush the temp file.
4. Rename the temp file to the target.
5. On error, remove the temp file.

## 10. Dependencies

### 10.1 Runtime Dependencies

- Go 1.23
- aios-core
- aios-policy
- aios-sandbox

### 10.2 Module Dependencies

The aios-tools module depends on the following
internal modules.

- aios-core
- aios-policy
- aios-sandbox

The aios-tools module is depended on by the following
internal modules.

- aios-runtime
- aios-cli

## 11. Deployment

The module compiles into the monorepo under `internal/<module>/`. The
package name is `github.com/<org>/aios/internal/tools`. The workspace root is
configured through the `AIOS_WORKSPACE` environment
variable.

## 12. Risks and Mitigations

| Risk | Impact | Probability | Mitigation |
| --- | --- | --- | --- |
| A tool reads a file outside the workspace. | High | Low | The path validation blocks the path. The tool test covers the path. |
| A tool writes a file outside the workspace. | High | Low | The path validation blocks the path. The tool test covers the path. |
| A shell command runs forever. | Medium | Medium | The tool has a timeout. The sandbox has a timeout. |
| The atomic write fails. | Medium | Low | The temp file is removed on error. The tool returns an error. |
