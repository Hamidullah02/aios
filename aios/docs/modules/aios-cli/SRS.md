# aios-cli — Software Requirements Specification

## 1. Purpose

This document states the software requirements for the
aios-cli module.

## 2. Scope

The scope of this document is the aios-cli module.
The module is the entry point for the user.

## 3. Reference Documents

- aios-cli PRD. See `docs/modules/aios-cli/PRD.md`.
- aios-core SRS. See `docs/modules/aios-core/SRS.md`.
- aios-runtime SRS. See `docs/modules/aios-runtime/SRS.md`.
- aios-approvals SRS. See
  `docs/modules/aios-approvals/SRS.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| CLI | The command-line interface. |
| Subcommand | A command under the main command. |

## 5. Functional Requirements

### 5.1 First-Run CLI

**REQ-CLI-FST-001** The module shall provide the
`aios` command. The command shall be the entry
point for the user.

**REQ-CLI-FST-002** The command shall support the
following subcommands.

- `aios run <plan>` — run a plan.
- `aios fix <objective>` — plan an objective and run it.
- `aios resume <task>` — resume a task.
- `aios cancel <task>` — cancel a task.
- `aios report <task>` — print the report.
- `aios daemon start` — start the daemon.
- `aios daemon stop` — stop the daemon.
- `aios approve <approval-id>` — grant a pending
  approval from another terminal.
- `aios deny <approval-id>` — deny a pending approval
  from another terminal.
- `aios audit <task>` — show the audit log.
- `aios init` — initialize the workspace.
- `aios tools` — list the tools the agent can dispatch.
- `aios web` — serve the operator UI.

**REQ-CLI-FST-003** The command shall use
`argparse` from the Go standard library.

**REQ-CLI-FST-004** The command shall print a
help message when the user types `aios --help`.

### 5.2 Run Subcommand

**REQ-CLI-RUN-001** The `run` subcommand shall
load the plan from a file. The plan is a YAML file.

**REQ-CLI-RUN-002** The `run` subcommand shall
start the orchestrator. The orchestrator shall run
the plan.

**REQ-CLI-RUN-003** The `run` subcommand shall
handle the `WaitingForApproval` exception. The
subcommand shall show the approval prompt. The
subcommand shall wait for the grant.

**REQ-CLI-RUN-004** The `run` subcommand shall
print the report when the plan is finished.

**REQ-CLI-RUN-005** The approval prompt shall display
the action, the target, and the reason from the approval
payload. A prompt that cannot name what it is asking to
authorize does not let a human decide.

**REQ-CLI-RUN-006** An approval shall authorize exactly
one tool call, bound to the task, action, and target the
operator was shown. A grant shall not carry over to a
different task, action, or target, and shall not persist
beyond the run that recorded it.

**REQ-CLI-RUN-007** After an approval is granted, the
retried tool call shall proceed without prompting again.
Re-asking for one action once per retry is a defect.

**REQ-CLI-RUN-008** When the subcommand cannot obtain an
answer (closed stdin, no operator, no daemon), it shall
treat the request as denied and shall not block
indefinitely. An unanswered request must never be treated
as approved.

**REQ-CLI-RUN-009** The `--yes` flag shall
auto-approve gated calls in place of prompting. Each
auto-approval shall be announced on the output stream, so
a transcript shows that no human decided.

**REQ-CLI-RUN-010** When no approval handler can be
reached, the subcommand shall report an actionable error
naming the approval and the ways to resolve it.

### 5.3 Resume Subcommand

**REQ-CLI-RSM-001** The `resume` subcommand shall
load the checkpoint. The subcommand shall call the
orchestrator to resume.

### 5.4 Cancel Subcommand

**REQ-CLI-CXL-001** The `cancel` subcommand shall
call the orchestrator to cancel. The subcommand
shall print the status.

### 5.5 Report Subcommand

**REQ-CLI-RPT-001** The `report` subcommand shall
load the report. The subcommand shall print the
report in the requested format. The supported
formats are JSON, Markdown, and HTML.

### 5.6 Daemon Subcommand

**REQ-CLI-DAE-001** The `daemon start` subcommand
shall start the approval daemon. The daemon shall
listen on a Unix socket.

**REQ-CLI-DAE-002** The `daemon stop` subcommand
shall stop the approval daemon.

**REQ-CLI-DAE-003** The daemon shall accept the
following messages.

- `request` — a request for an approval.
- `grant` — a grant for an approval.
- `deny` — a deny for an approval.
- `wait` — a wait for an approval.

**REQ-CLI-DAE-004** The daemon shall write the
approval events to the audit log.

### 5.7 Approve / Deny Subcommands

**REQ-CLI-APV-001** The `approve` subcommand shall
take an approval id and shall send a `grant` message to
the approval daemon on the configured socket.

**REQ-CLI-APV-002** The `deny` subcommand shall take
an approval id and shall send a `deny` message to the
approval daemon on the configured socket.

**REQ-CLI-APV-003** Both subcommands shall require the
approval id as a positional argument and shall fail with
a usage error when it is absent.

**REQ-CLI-APV-004** Both subcommands shall work
without a local policy file. Resolving what is pending
must remain possible when the policy is the thing that is
broken.

**REQ-CLI-APV-005** When the daemon is unreachable,
both subcommands shall report the socket path and name
the `daemon start` command as the remedy.

### 5.8 Audit Subcommand

**REQ-CLI-AUD-001** The `audit` subcommand shall
print the audit log for a task. The subcommand
shall support the `tail` flag.

### 5.9 Init Subcommand

**REQ-CLI-INI-001** The `init` subcommand shall
create the workspace. The subcommand shall create
the `var/` folder. The subcommand shall create the
default policy file. The subcommand shall create
the default egress file. The subcommand shall
create the default price file.

### 5.10 Error Display

**REQ-CLI-ERR-001** The CLI shall print a clear
error message when an error occurs. The message
shall include the error type and the error reason.

**REQ-CLI-ERR-002** The CLI shall print the
traceback when the `--debug` flag is set.

**REQ-CLI-ERR-003** The CLI shall return exit
code 0 on success. The CLI shall return exit
code 1 on user error. The CLI shall return exit
code 2 on system error.

## 6. Non-Functional Requirements

**REQ-CLI-NFR-001** The module shall run on Go 1.23
3.12.

**REQ-CLI-NFR-002** The module shall have 100 percent
coverage for the public functions.

**REQ-CLI-NFR-003** The module shall start within
1 second on a modern laptop.

## 7. Interface Requirements

**REQ-CLI-IFC-001** The module shall export the
following items from `aios.cli`.

- `main`
- `run_command`
- `resume_command`
- `cancel_command`
- `report_command`
- `daemon_command`
- `audit_command`
- `init_command`

## 8. Data Requirements

**REQ-CLI-DAT-001** The config file shall be a YAML
file. The default path is `./var/config.yaml`. The
file is loaded by the `init` subcommand.

**REQ-CLI-DAT-002** The log file shall be a text
file. The default path is `./var/aios.log`. The
file is written through the standard `logging`
module.

## 9. Error Handling

**REQ-CLI-ERR-100** The module shall raise
`CliError` when a CLI error occurs.

**REQ-CLI-ERR-101** The module shall raise
`ConfigError` when the config file is invalid.

**REQ-CLI-ERR-102** The module shall raise
`PlanLoadError` when the plan file is invalid.

## 10. Acceptance Criteria

1. Every REQ in this document has a test case.
2. The test suite has a 100 percent pass rate.
3. The test suite has 100 percent coverage for the
   public functions.
4. The CLI loads a plan.
5. The CLI runs a plan.
6. The CLI shows the approval prompt.
7. The CLI sends the grant.
8. The CLI prints the report.
9. The CLI starts the daemon.
10. The CLI initializes the workspace.
