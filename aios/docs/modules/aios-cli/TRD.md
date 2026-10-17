# aios-cli — Technical Requirements Document

## 1. Purpose

This document states the technical approach for the
aios-cli module.

## 2. Scope

The scope of this document is the aios-cli module.
The module is the entry point for the user.

## 3. Reference Documents

- aios-cli PRD. See `docs/modules/aios-cli/PRD.md`.
- aios-cli SRS. See `docs/modules/aios-cli/SRS.md`.
- aios-core TRD. See `docs/modules/aios-core/TRD.md`.
- aios-runtime TRD. See `docs/modules/aios-runtime/TRD.md`.
- aios-approvals TRD. See
  `docs/modules/aios-approvals/TRD.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| flag (stdlib) | The Go standard-library command-line flag parser. |
| Unix socket | A local inter-process communication channel. |
| goroutines + channels + errgroup | The Go concurrency primitives for async I/O. |

## 5. Architecture

The aios-cli module has three components.

- `aios.cli.parser` — the command parser.
- `internal/cli/runner` — the run subcommand.
- `internal/cli/daemon` — the approval daemon.

The three components depend on every other module.
The components depend on each other in this order.
The parser has no dependencies. The runner depends
on the parser, the orchestrator, the approval
service, the report generator, and the audit log.
The daemon depends on the parser, the approval
service, and the audit log.

## 6. Components

### 6.1 Parser

The parser component uses `argparse`. The component
defines the subcommands. The component returns the
parsed arguments.

### 6.2 Runner

The runner component runs the plan. The component
loads the plan. The component starts the
orchestrator. The component handles the
`WaitingForApproval` exception. The component shows
the approval prompt. The component waits for the
grant.

The component uses the approval daemon. The
component sends the approval request to the daemon.
The daemon shows the prompt. The daemon returns
the answer.

### 6.3 Daemon

The daemon component uses a Unix socket. The
component listens on the socket. The component
accepts messages. The component routes the
messages to the approval service.

The component starts on `aios daemon start`. The
component stops on `aios daemon stop`. The
component uses `atexit` to clean up on exit.

## 7. Data Structures

### 7.1 CliArgs

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `command` | string | yes | The subcommand. |
| `args` | dict | yes | The subcommand arguments. |
| `debug` | bool | no | The debug flag. |
| `config` | string | no | The config file path. |

## 8. Interfaces

### 8.1 Main

```go
def main(argv: list[str] | nil = nil) -> int:
    """The CLI entry point.

    Args:
        argv: The command-line arguments.

    Returns:
        The exit code.
    """
```

### 8.2 Approval Daemon

```go
type ApprovalDaemon:
    """The approval daemon."""

    def __init__(
        self,
        approval_service: ApprovalService,
        audit_log: AuditLog,
        socket_path: str = "/tmp/aios-approval.sock",
    ) -> nil:
        """Create the approval daemon."""

    func start(self) -> nil:
        """Start the daemon."""

    func stop(self) -> nil:
        """Stop the daemon."""
```

## 9. Algorithms

### 9.1 Run Algorithm

The run algorithm works as follows.

1. Load the plan from the file.
2. Build the orchestrator.
3. Start the orchestrator.
4. When the orchestrator raises
   `WaitingForApproval`, send the approval request
   to the daemon.
5. Wait for the daemon to return the answer.
6. Resume the orchestrator.
7. When the orchestrator finishes, build the
   report.
8. Print the report.

### 9.2 Daemon Algorithm

The daemon algorithm works as follows.

1. Listen on the Unix socket.
2. Accept a connection.
3. Read the message.
4. Route the message to the approval service.
5. Write the response to the connection.
6. Close the connection.

## 10. Dependencies

### 10.1 Runtime Dependencies

- Go 1.23
- gopkg.in/yaml.v3
- aios-core
- aios-policy
- aios-runtime
- aios-models
- aios-tools
- aios-sandbox
- aios-egress
- aios-approvals
- aios-ask
- aios-planner
- aios-report

### 10.2 Module Dependencies

The aios-cli module depends on every other module.
The aios-cli module is not depended on by any
internal module.

## 11. Deployment

The module compiles into the monorepo under `internal/<module>/`. The
package name is `github.com/<org>/aios/internal/cli`. The package provides a
console script named `aios`.

The console script is registered through the
`go.mod` file. The entry point is
`aios.cli.main:main`.

## 12. Risks and Mitigations

| Risk | Impact | Probability | Mitigation |
| --- | --- | --- | --- |
| The CLI fails to start. | High | Low | The CLI has a health check. The CLI returns a clear error. |
| The daemon crashes. | High | Low | The daemon is restarted by the process manager. The daemon writes the crash to the log. |
| The approval prompt is not shown. | High | Low | The CLI checks the daemon is running. The CLI starts the daemon when needed. |
| The plan file is invalid. | High | Medium | The CLI validates the plan. The CLI returns a clear error. |
