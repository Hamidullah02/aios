# aios-cli — Application Flow

## 1. Purpose

This document describes the application flow for the
aios-cli module.

## 2. Scope

The scope of this document is the aios-cli module.
The module is the user-facing entry point.

## 3. Reference Documents

- aios-cli PRD. See `docs/modules/aios-cli/PRD.md`.
- aios-cli SRS. See `docs/modules/aios-cli/SRS.md`.
- aios-cli TRD. See `docs/modules/aios-cli/TRD.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Operator | The user that runs the CLI. |
| Daemon | The approval daemon process. |

## 5. User Flow

### 5.1 First Run Flow

The first run flow runs as follows.

1. The operator types `aios init`.
2. The CLI creates the workspace.
3. The CLI prints the next steps.
4. The operator types `aios run plan.yaml`.
5. The CLI loads the plan.
6. The CLI runs the plan.
7. The CLI shows the approval prompt.
8. The operator answers the prompt.
9. The CLI prints the report.

### 5.2 Daily Run Flow

The daily run flow runs as follows.

1. The operator types `aios run plan.yaml`.
2. The CLI loads the plan.
3. The CLI runs the plan.
4. The CLI shows the approval prompt when needed.
5. The CLI prints the report.

## 6. System Flow

### 6.1 Init Flow

The init flow runs as follows.

1. The CLI creates the `var/` folder.
2. The CLI creates the default policy file.
3. The CLI creates the default egress file.
4. The CLI creates the default price file.
5. The CLI creates the default config file.
6. The CLI prints the next steps.

### 6.2 Run Flow

The run flow runs as follows.

1. The CLI loads the plan from the file.
2. The CLI validates the plan.
3. The CLI starts the orchestrator.
4. The orchestrator runs the tasks.
5. When a tier-3 action needs approval, the
   orchestrator raises `WaitingForApproval`.
6. The CLI catches the exception.
7. The CLI sends the approval request to the
   daemon.
8. The daemon shows the prompt to the operator.
9. The operator answers the prompt.
10. The daemon returns the answer.
11. The CLI sends the grant to the approval
    service.
12. The CLI resumes the orchestrator.
13. The orchestrator continues the tasks.
14. The orchestrator finishes the plan.
15. The CLI builds the report.
16. The CLI prints the report.

### 6.3 Daemon Start Flow

The daemon start flow runs as follows.

1. The operator types `aios daemon start`.
2. The CLI starts the daemon process.
3. The daemon opens the Unix socket.
4. The daemon waits for connections.

### 6.4 Daemon Request Flow

The daemon request flow runs as follows.

1. The CLI sends the request to the daemon.
2. The daemon accepts the connection.
3. The daemon reads the request.
4. The daemon shows the prompt.
5. The operator answers the prompt.
6. The daemon sends the answer to the approval
   service.
7. The daemon closes the connection.

## 7. Error Flows

### 7.1 Plan Load Error

The plan load error flow runs as follows.

1. The CLI loads an invalid plan.
2. The CLI raises `PlanLoadError`.
3. The CLI prints the error.
4. The CLI returns exit code 1.

### 7.2 Config Error

The config error flow runs as follows.

1. The CLI loads an invalid config.
2. The CLI raises `ConfigError`.
3. The CLI prints the error.
4. The CLI returns exit code 1.

### 7.3 System Error

The system error flow runs as follows.

1. The orchestrator raises an unhandled error.
2. The CLI prints the error.
3. The CLI prints the traceback when `--debug` is
   set.
4. The CLI returns exit code 2.

## 8. Sequence Diagrams

### 8.1 Run Sequence with Approval

The run sequence with approval has the steps below.

1. The operator types `aios run plan.yaml`.
2. The CLI loads the plan.
3. The CLI starts the orchestrator.
4. The orchestrator runs a task.
5. The worker raises `WaitingForApproval`.
6. The orchestrator pauses the task.
7. The CLI catches the exception.
8. The CLI sends the request to the daemon.
9. The daemon shows the prompt.
10. The operator answers `y`.
11. The daemon sends the grant to the service.
12. The daemon returns the answer.
13. The CLI resumes the orchestrator.
14. The orchestrator continues the task.
15. The orchestrator finishes the plan.
16. The CLI builds the report.
17. The CLI prints the report.
