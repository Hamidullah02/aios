# aios-planner — Technical Requirements Document

## 1. Purpose

This document states the technical approach for the
aios-planner module.

## 2. Scope

The scope of this document is the aios-planner module.
The module is a Go package.

## 3. Reference Documents

- aios-planner PRD. See
  `docs/modules/aios-planner/PRD.md`.
- aios-planner SRS. See
  `docs/modules/aios-planner/SRS.md`.
- aios-core TRD. See `docs/modules/aios-core/TRD.md`.
- aios-runtime TRD. See `docs/modules/aios-runtime/TRD.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| struct + go-playground/validator (or hand-rolled tag validation) | The Go struct validator used for typed cross-module records. |
| Repair loop | A loop that calls the model to fix the errors. |

## 5. Architecture

The aios-planner module has three components.

- `aios.planner.planner` — the planner.
- `internal/planner/verifier` — the verifier.
- `aios.planner.prompts` — the prompt templates.

The three components depend on aios-core, aios-runtime,
and aios-models. The components depend on each other
in this order. The prompts have no dependencies. The
verifier has no dependencies. The planner depends on
the model router, the tool map, the verifier, and
the prompts.

## 6. Components

### 6.1 Planner

The planner component uses the model router. The
planner builds the prompt from the goal, the tool
map, the context, and the prompts. The planner calls
the model router. The model returns the plan as
JSON. The planner validates the JSON against the
schema. The planner runs the verifier. The planner
repairs the plan on fail.

The planner supports a maximum repair iteration
count. The default is 3. The planner raises
`PlanGenerationError` when the count is exceeded.

### 6.2 Verifier

The verifier component runs the following checks.

1. Schema validation. The verifier validates the
   plan against the `TaskPlan` schema.
2. Tool call schema validation. The verifier
   validates every tool call against the `ToolCall`
   schema.
3. Diff scope check. The verifier compares the
   number of changed lines to the limit. The default
   limit is 200 lines.
4. Custom rules. The verifier runs the custom rules
   on the plan.

The verifier returns a list of `PlanError`. The
verifier returns an empty list on pass.

### 6.3 Prompts

The prompts component holds the prompt templates.
The component uses text/template (stdlib). The templates are the
following.

- `plan_system.txt` — the system prompt.
- `plan_user.txt` — the user prompt.
- `repair_user.txt` — the repair prompt.

## 7. Data Structures

### 7.1 PlanError

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `code` | string | yes | The error code. |
| `field` | string | no | The field with the error. |
| `message` | string | yes | The error message. |

## 8. Interfaces

### 8.1 Planner

```go
type Planner:
    """The planner."""

    def __init__(
        self,
        model_router: ModelRouter,
        tool_map: dict[str, Tool],
        verifier: Verifier,
        max_repair_iterations: int = 3,
    ) -> nil:
        """Create the planner."""

    func plan(
        self,
        goal: str,
        context: dict,
    ) -> TaskPlan:
        """Plan a goal.

        Args:
            goal: The goal to plan.
            context: The context for the plan.

        Returns:
            The plan.
        """
```

### 8.2 CodeFirstVerifier

```go
type CodeFirstVerifier:
    """The code-first verifier."""

    def __init__(
        self,
        tool_map: dict[str, Tool],
        diff_scope_limit: int = 200,
    ) -> nil:
        """Create the verifier."""

    def verify(self, plan: TaskPlan) -> list[PlanError]:
        """Verify the plan.

        Args:
            plan: The plan to verify.

        Returns:
            The list of errors.
        """
```

## 9. Algorithms

### 9.1 Planning Algorithm

The planning algorithm works as follows.

1. Build the prompt from the goal, the tool map,
   and the context.
2. Call the model router with the prompt.
3. Parse the response as JSON.
4. Validate the JSON against the schema.
5. Run the verifier.
6. On pass, return the plan.
7. On fail, call the model with the errors.
8. Repeat from step 3 for the maximum repair
   iterations.
9. On exceed, raise `PlanGenerationError`.

### 9.2 Diff Scope Check

The diff scope check algorithm works as follows.

1. For each task, compute the number of changed
   lines from the tool call arguments.
2. Sum the counts.
3. When the sum exceeds the limit, return an error.

## 10. Dependencies

### 10.1 Runtime Dependencies

- Go 1.23
- text/template (stdlib)
- aios-core
- aios-runtime
- aios-models

### 10.2 Module Dependencies

The aios-planner module depends on the following
internal modules.

- aios-core
- aios-runtime
- aios-models

The aios-planner module is depended on by the
following internal modules.

- aios-cli

## 11. Deployment

The module compiles into the monorepo under `internal/<module>/`. The
package name is `github.com/<org>/aios/internal/planner`.

## 12. Risks and Mitigations

| Risk | Impact | Probability | Mitigation |
| --- | --- | --- | --- |
| The model returns invalid JSON. | High | Medium | The planner parses the response. The planner raises an error on parse failure. |
| The repair loop runs forever. | Medium | Low | The planner has a maximum iteration count. The planner raises an error on exceed. |
| The diff scope is too small. | Medium | Low | The limit is configurable. The limit is documented. |
