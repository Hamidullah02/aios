# aios-cli — Implementation Plan

## 1. Purpose

This document states the implementation plan for the
aios-cli module.

## 2. Scope

The scope of this document is the aios-cli module.
The module depends on every other module.

## 3. Reference Documents

- aios-cli PRD. See `docs/modules/aios-cli/PRD.md`.
- aios-cli SRS. See `docs/modules/aios-cli/SRS.md`.
- aios-cli TRD. See `docs/modules/aios-cli/TRD.md`.
- aios-cli Application Flow. See
  `docs/modules/aios-cli/APP-FLOW.md`.
- aios-cli Wireframes. See
  `docs/modules/aios-cli/WIREFRAMES.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Phase | A unit of work in the implementation plan. |

## 5. Phases

### Phase 1: Project Skeleton

1. Create the project folders.
2. Create the `go.mod` file.
3. Create the `README.md` file.

The deliverable is the project skeleton.

### Phase 2: Parser

1. Create the file `internal/cli/parser.py`.
2. Create the file `tests/test_parser.py`.
3. Implement the `argparse` setup.
4. Implement the subcommand parsers.
5. Write the test cases.

The deliverable is the parser with tests.

### Phase 3: Init Subcommand

1. Create the file `internal/cli/init_command.py`.
2. Create the file `tests/test_init_command.py`.
3. Implement the workspace creation.
4. Implement the default files.
5. Write the test cases.

The deliverable is the init subcommand with tests.

### Phase 4: Run Subcommand

1. Create the file `internal/cli/run_command.py`.
2. Create the file `tests/test_run_command.py`.
3. Implement the plan loading.
4. Implement the orchestrator startup.
5. Implement the `WaitingForApproval` handler.
6. Implement the daemon communication.
7. Implement the report printing.
8. Write the test cases.

The deliverable is the run subcommand with tests.

### Phase 5: Resume Subcommand

1. Create the file `internal/cli/resume_command.py`.
2. Create the file `tests/test_resume_command.py`.
3. Implement the checkpoint loading.
4. Implement the orchestrator resume.
5. Write the test cases.

The deliverable is the resume subcommand with tests.

### Phase 6: Cancel Subcommand

1. Create the file `internal/cli/cancel_command.py`.
2. Create the file `tests/test_cancel_command.py`.
3. Implement the orchestrator cancel.
4. Write the test cases.

The deliverable is the cancel subcommand with tests.

### Phase 7: Report Subcommand

1. Create the file `internal/cli/report_command.py`.
2. Create the file `tests/test_report_command.py`.
3. Implement the report loading.
4. Implement the report printing.
5. Write the test cases.

The deliverable is the report subcommand with tests.

### Phase 8: Audit Subcommand

1. Create the file `internal/cli/audit_command.py`.
2. Create the file `tests/test_audit_command.py`.
3. Implement the audit log reading.
4. Implement the audit log printing.
5. Write the test cases.

The deliverable is the audit subcommand with tests.

### Phase 9: Daemon

1. Create the file `internal/cli/daemon.py`.
2. Create the file `tests/test_daemon.py`.
3. Implement the daemon class.
4. Implement the Unix socket server.
5. Implement the message routing.
6. Write the test cases.

The deliverable is the daemon with tests.

### Phase 10: Daemon Subcommand

1. Create the file `internal/cli/daemon_command.py`.
2. Create the file `tests/test_daemon_command.py`.
3. Implement the daemon start.
4. Implement the daemon stop.
5. Write the test cases.

The deliverable is the daemon subcommand with tests.

### Phase 11: Main Entry Point

1. Create the file `internal/cli/main.py`.
2. Create the file `tests/test_main.py`.
3. Implement the `main` function.
4. Implement the error display.
5. Write the test cases.

The deliverable is the main entry point with tests.

### Phase 12: Quality Gates

1. Run the test suite with `go test`.
2. Run the coverage report with `go test -cover`.
3. Run the type check with `go vet ./internal/cli`.
4. Run the linter with `gofmt + go vet check`.
5. Fix every failure.

The deliverable is a green test suite.

## 6. Deliverables

- The source files in `internal/cli/`.
- The test files in `tests/`.
- The `go.mod` file.
- The `README.md` file.
- The console script entry point.

## 7. Test Plan

### 7.1 Unit Tests

- `tests/test_parser.py` covers
  `internal/cli/parser.py`.
- `tests/test_init_command.py` covers
  `internal/cli/init_command.py`.
- `tests/test_run_command.py` covers
  `internal/cli/run_command.py`.
- `tests/test_resume_command.py` covers
  `internal/cli/resume_command.py`.
- `tests/test_cancel_command.py` covers
  `internal/cli/cancel_command.py`.
- `tests/test_report_command.py` covers
  `internal/cli/report_command.py`.
- `tests/test_audit_command.py` covers
  `internal/cli/audit_command.py`.
- `tests/test_daemon.py` covers
  `internal/cli/daemon.py`.
- `tests/test_daemon_command.py` covers
  `internal/cli/daemon_command.py`.
- `tests/test_main.py` covers
  `internal/cli/main.py`.

### 7.2 Test Cases

The test cases include the following items.

- The parser parses the arguments.
- The init subcommand creates the workspace.
- The run subcommand loads the plan.
- The run subcommand starts the orchestrator.
- The run subcommand handles
  `WaitingForApproval`.
- The run subcommand resumes on grant.
- The run subcommand prints the report.
- The resume subcommand loads the checkpoint.
- The cancel subcommand cancels the task.
- The report subcommand prints the report.
- The audit subcommand prints the audit log.
- The daemon accepts connections.
- The daemon routes messages.
- The daemon start subcommand starts the daemon.
- The daemon stop subcommand stops the daemon.
- The main function returns the exit code.
- The error display prints the error.

## 8. Merge Plan

### 8.1 Pre-Merge Checklist

1. The test suite has a 100 percent pass rate.
2. The coverage report shows 100 percent coverage for
   the public functions.
3. The `go vet` report shows no errors.
4. The `gofmt + go vet` report shows no errors.
5. The console script is registered in
   `go.mod`.
6. The end-to-end smoke test passes. The smoke test
   runs `aios init`, `aios run plan.yaml`, and
   `aios report task-001`.

### 8.2 Merge Steps

1. Create a branch named `merge/aios-cli`.
2. Move the files from `modules/aios-cli/` to the
   target locations in the main repository.
3. Run the test suite on the main repository.
4. Open a pull request.
5. Merge after the review is approved.

### 8.3 Post-Merge Steps

1. Update the project README with the CLI usage.
2. Publish the package to PyPI.
3. Update the documentation site.

## 9. Risk Register

| Risk | Impact | Probability | Mitigation |
| --- | --- | --- | --- |
| The CLI fails to start. | High | Low | The CLI has a health check. The CLI returns a clear error. |
| The daemon crashes. | High | Low | The daemon is restarted by the process manager. The daemon writes the crash to the log. |
| The approval prompt is not shown. | High | Low | The CLI checks the daemon is running. The CLI starts the daemon when needed. |
| The plan file is invalid. | High | Medium | The CLI validates the plan. The CLI returns a clear error. |
| The console script is not on the PATH. | High | Medium | The installation is documented. The smoke test verifies the script is on the PATH. |

## 10. Schedule

| Phase | Duration | Dependency |
| --- | --- | --- |
| Phase 1: Project Skeleton | 1 day | nil |
| Phase 2: Parser | 1 day | Phase 1 |
| Phase 3: Init Subcommand | 1 day | Phase 2 |
| Phase 4: Run Subcommand | 4 days | Phase 2, all other modules |
| Phase 5: Resume Subcommand | 1 day | Phase 4 |
| Phase 6: Cancel Subcommand | 1 day | Phase 4 |
| Phase 7: Report Subcommand | 1 day | Phase 4 |
| Phase 8: Audit Subcommand | 1 day | Phase 4 |
| Phase 9: Daemon | 2 days | Phase 2 |
| Phase 10: Daemon Subcommand | 1 day | Phase 9 |
| Phase 11: Main Entry Point | 1 day | All previous phases |
| Phase 12: Quality Gates | 2 days | All phases |

The total duration is 17 days. The schedule assumes
that every other module is already merged.
