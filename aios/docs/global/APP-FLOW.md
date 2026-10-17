# aios — Global Application Flow

## 1. Purpose

This document describes the application flow for the
aios project at the cross-module level. The per-module
APP-FLOW documents are the source of truth for the
module detail. This document is the source of truth for
the run lifecycle, the user flows, and the system
flows that span more than one module.

## 2. Scope

The aios project is in scope for this document.

- The user flows.
- The system flows.
- The error flows.
- The sequence diagrams.

## 3. Reference Documents

- Global PRD. See `docs/global/PRD.md`.
- Global SRS. See `docs/global/SRS.md`.
- Global TRD. See `docs/global/TRD.md`.
- The per-module APP-FLOW under
  `docs/modules/<module>/APP-FLOW.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Run | A single execution of the agent. |
| Plan | The output of the planner. |
| Task | A unit of work in the plan. |
| Step | A single tool call within a task. |

## 5. User Flows

### 5.1 First Run

The first run flow runs as follows.

1. The operator installs the project through
   `go mod download && go mod tidy`.
2. The operator runs `aios init`. The CLI creates
   `var/`, `var/sessions/`, `var/handoffs/`, and
   `var/reports/`. The CLI creates
   `var/optimizer.yaml`, `var/roles.yaml`,
   `var/policy.yaml`, and `var/models.yaml`.
3. The operator edits the config files. The
   operator sets the model, the budget, the
   policy, and the roles.
4. The operator runs `aios run "<goal>"`. The
   CLI starts the run.

### 5.2 Subsequent Run

The subsequent run flow runs as follows.

1. The operator runs `aios run "<goal>"`. The
   CLI starts the run.
2. The CLI loads the config files. The CLI
   loads the rules from `github.com/<org>/aios/internal/persist`. The CLI
   applies the rules to the policy.

### 5.3 Approval Flow

The approval flow runs as follows.

1. The agent calls a tier-3 action.
2. The policy gateway returns `ask`.
3. The approval service shows the approval
   prompt.
4. The operator chooses one of the four options.
5. The approval service persists the decision.
6. The run continues or stops based on the
   decision.

### 5.4 Report Flow

The report flow runs as follows.

1. The operator runs `aios report`. The CLI shows
   the latest report.
2. The operator runs `aios report <session_id>`.
   The CLI shows the report for the session.
3. The operator runs `aios report --format html`.
   The CLI shows the report in HTML.

### 5.5 Replay Flow

The replay flow runs as follows.

1. The operator runs `aios replay <session_id>`.
2. The CLI loads the session from
   `github.com/<org>/aios/internal/persist`.
3. The CLI re-executes the plan.
4. The CLI writes the new report.

## 6. System Flows

### 6.1 Run Lifecycle

The run lifecycle has the following phases.

1. **Init** — the CLI initializes the run. The
   CLI loads the config. The CLI loads the rules.
   The CLI writes the `run_started` audit row.
2. **Plan** — the planner turns the goal into a
   `TaskPlan`. The planner writes the
   `plan_generated` audit row.
3. **Dispatch** — the orchestrator dispatches the
   first task. The orchestrator writes the
   `task_started` audit row.
4. **Resolve Role** — the orchestrator resolves
   the role. The role resolver returns the role
   configuration.
5. **Route** — the model router picks the model
   tier. The router writes the
   `routing_decision` audit row.
6. **Call** — the model adapter sends the
   request. The adapter writes the `model_call`
   audit row.
7. **Tool Call** — the tool bus runs the tool
   call. The bus calls the policy gateway. The
   bus calls the retry policy. The bus calls the
   sandbox. The bus writes the `tool_call` audit
   row.
8. **Approve** — when the policy returns `ask`,
   the approval service pauses the run. The
   service writes the `approval_decision` audit
   row.
9. **Retry** — when the tool fails, the retry
   policy decides. The policy writes the
   `retry_decision` audit row.
10. **Handoff** — when the task produces an
    artifact, the handoff store saves the
    artifact. The store writes the `handoff`
    audit row.
11. **Verify** — the self-check verifies the
    result. The check writes the `self_check`
    audit row.
12. **Checkpoint** — the session checkpoint
    saves the state. The checkpoint writes the
    `checkpoint` audit row.
13. **Dispatch Next** — the orchestrator
    dispatches the next task. The cycle continues
    until the plan is complete.
14. **Report** — the report generator writes
    the report. The generator writes the
    `report_generated` audit row.
15. **Done** — the CLI shows the result. The CLI
    writes the `run_completed` audit row.

### 6.2 Crash Recovery

The crash recovery flow runs as follows.

1. The agent crashes.
2. The operator runs `aios resume <session_id>`.
3. The CLI loads the session from
   `github.com/<org>/aios/internal/persist`.
4. The CLI resumes the run from the last
   checkpoint.
5. The cycle continues.

### 6.3 Audit Verification

The audit verification flow runs as follows.

1. The operator runs `aios audit verify`.
2. The audit log iterates over the rows.
3. The audit log verifies the hash chain.
4. The audit log reports the result.

## 7. Error Flows

### 7.1 Policy Deny

The policy deny flow runs as follows.

1. The agent calls a tier-3 action.
2. The policy gateway returns `deny`.
3. The tool bus raises `PolicyDenied`.
4. The orchestrator catches the error.
5. The orchestrator writes the
   `policy_denied` audit row.
6. The orchestrator returns the error to the
   planner.

### 7.2 Budget Exhausted

The budget exhausted flow runs as follows.

1. The cost tracker exhausts the budget.
2. The tracker raises `BudgetExhausted`.
3. The orchestrator catches the error.
4. The orchestrator writes the
   `budget_exhausted` audit row.
5. The orchestrator stops the run.

### 7.3 Approval Timeout

The approval timeout flow runs as follows.

1. The agent calls a tier-3 action.
2. The approval service waits for the operator.
3. The operator does not respond within the
   timeout.
4. The approval service raises
   `ApprovalTimeout`.
5. The orchestrator catches the error.
6. The orchestrator writes the
   `approval_timeout` audit row.
7. The orchestrator stops the run.

### 7.4 Retry Exhausted

The retry exhausted flow runs as follows.

1. The tool fails.
2. The retry policy returns `stop`.
3. The tool bus raises `RetryExhausted`.
4. The orchestrator catches the error.
5. The orchestrator writes the
   `retry_exhausted` audit row.
6. The orchestrator returns the error to the
   planner.

### 7.5 Verification Failed

The verification failed flow runs as follows.

1. The self-check returns `fail`.
2. The orchestrator catches the result.
3. The orchestrator writes the
   `verification_failed` audit row.
4. The orchestrator returns the result to the
   planner.
5. The planner revises the plan.

## 8. Determinism Flows

The determinism flows are the flows that protect
the determinism thesis. The flows are listed
below.

### 8.1 Replay Flow

The replay flow runs as follows.

1. The operator runs `aios replay <session_id>`.
2. The CLI loads the session from
   `internal/persist`.
3. The CLI loads the checkpoint from
   `internal/persist`.
4. The CLI passes the checkpoint to the replay
   engine.
5. The replay engine re-executes the plan from
   the checkpoint.
6. The replay engine produces a new audit log.
7. The CLI shows the new audit log.

### 8.2 Diff Flow

The diff flow runs as follows.

1. The operator runs `aios diff <session_id_a>
   <session_id_b>`.
2. The CLI loads both sessions from
   `internal/persist`.
3. The CLI loads both audit logs.
4. The CLI computes the unified diff.
5. The CLI shows the diff.

### 8.3 Plan Revision Flow

The plan revision flow runs as follows.

1. The agent revises the plan.
2. The orchestrator detects the revision.
3. The orchestrator rejects the revision because
   the plan is immutable.
4. The orchestrator writes the
   `plan_revision_rejected` audit row.
5. The orchestrator stops the run.

### 8.4 Policy Revision Flow

The policy revision flow runs as follows.

1. The operator revises the policy.
2. The orchestrator detects the revision.
3. The orchestrator rejects the revision because
   the policy is immutable.
4. The orchestrator writes the
   `policy_revision_rejected` audit row.
5. The orchestrator stops the run.

## 9. Sequence Diagrams

### 9.1 Run Sequence

The run sequence has the steps below.

1. The operator runs `aios run "<goal>"`.
2. The CLI passes the goal to the planner.
3. The planner returns the plan.
4. The CLI passes the plan to the orchestrator.
5. The orchestrator dispatches the tasks.
6. The orchestrator calls the report generator.
7. The CLI shows the report.

### 9.2 Task Sequence

The task sequence has the steps below.

1. The orchestrator calls the role resolver.
2. The orchestrator calls the model router.
3. The orchestrator calls the model adapter.
4. The model adapter returns the response.
5. The orchestrator calls the tool bus.
6. The tool bus calls the policy gateway.
7. The tool bus runs the tool.
8. The orchestrator calls the self-check.
9. The orchestrator writes the audit row.
10. The orchestrator dispatches the next task.

### 9.3 Approval Sequence

The approval sequence has the steps below.

1. The tool bus calls the policy gateway.
2. The policy gateway returns `ask`.
3. The tool bus calls the approval service.
4. The approval service shows the prompt.
5. The operator responds.
6. The approval service persists the decision.
7. The approval service returns the decision.
8. The tool bus runs the tool or raises
   `PolicyDenied`.
