# aios-verify — Technical Requirements Document

## 1. Purpose

This document states the technical approach for the
aios-verify module.

## 2. Scope

The scope of this document is the aios-verify module.
The module is a Go package.

## 3. Reference Documents

- aios-verify PRD. See
  `docs/modules/aios-verify/PRD.md`.
- aios-verify SRS. See
  `docs/modules/aios-verify/SRS.md`.
- aios-core TRD. See `docs/modules/aios-core/TRD.md`.
- aios-tools TRD. See `docs/modules/aios-tools/TRD.md`.
- aios-models TRD. See `docs/modules/aios-models/TRD.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| text/template (stdlib) | The Go standard-library template engine. |
| go test | The Go standard-library test framework. |
| Jest | A JavaScript test framework. |

## 5. Architecture

The aios-verify module has four components.

- `internal/verify/self_check` — the self-check tool.
- `internal/verify/run_tests` — the run_tests tool.
- `internal/verify/prompt` — the verification prompt.
- `internal/verify/result` — the verification result.

The four components depend on aios-core, aios-tools,
and aios-models. The components depend on each
other in this order. The prompt has no
dependencies. The result has no dependencies. The
self-check tool depends on the prompt, the result,
and the model router. The run_tests tool depends on
the os/exec wrapper and the result.

## 6. Components

### 6.1 Self-Check Tool

The self-check tool accepts a result. The tool
builds the verification prompt through the
`VerificationPrompt` class. The tool calls the
model router. The tool parses the response into the
`VerificationResult`.

The tool supports a `goal` argument. The tool uses
the goal in the prompt. The tool returns the
verification result.

### 6.2 run_tests Tool

The run_tests tool accepts a `test_command`
argument. The tool runs the command through the
os/exec wrapper. The tool parses the output
through a per-runner parser.

The tool supports the following runners.

- go test — parses the JUnit XML output.
- jest — parses the JSON output.
- mocha — parses the JSON output.
- cargo test — parses the JSON output.
- go test — parses the JSON output.
- npm test — parses the JSON output.

The tool returns the `TestResult` object.

### 6.3 Verification Prompt

The prompt component uses text/template (stdlib). The component
loads the template from the default location. The
component renders the template with the goal and
the result.

### 6.4 Verification Result

The result component uses struct + go-playground/validator (or hand-rolled tag validation). The component
defines the `VerificationResult` and the
`TestResult` schemas. The component validates the
result through struct + go-playground/validator (or hand-rolled tag validation).

## 7. Data Structures

### 7.1 VerificationResult

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `status` | string | yes | The status: pass, fail, inconclusive. |
| `reason` | string | yes | The reason for the status. |
| `suggestions` | list of string | no | The suggestions for the fix. |

### 7.2 TestResult

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `passed` | int | yes | The number of passed tests. |
| `failed` | int | yes | The number of failed tests. |
| `skipped` | int | yes | The number of skipped tests. |
| `duration_ms` | int | yes | The duration in milliseconds. |
| `failures` | list of TestFailure | yes | The list of failures. |

### 7.3 TestFailure

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `name` | string | yes | The test name. |
| `message` | string | yes | The failure message. |
| `traceback` | string | no | The traceback. |

## 8. Interfaces

### 8.1 SelfCheckTool

```go
type SelfCheckTool:
    """The self-check tool."""

    def __init__(
        self,
        model_router: ModelRouter,
        prompt: VerificationPrompt,
    ) -> nil:
        """Create the self-check tool."""

    func run(self, args: dict, context: ToolContext) -> ToolResult:
        """Run the self-check tool.

        Args:
            args: The arguments of the call.
            context: The context for the call.

        Returns:
            The result of the call.
        """
```

### 8.2 RunTestsTool

```go
type RunTestsTool:
    """The run tests tool."""

    def __init__(self, sandbox: SubprocessRunner) -> nil:
        """Create the run tests tool."""

    func run(self, args: dict, context: ToolContext) -> ToolResult:
        """Run the run tests tool.

        Args:
            args: The arguments of the call.
            context: The context for the call.

        Returns:
            The result of the call.
        """
```

## 9. Algorithms

### 9.1 Self-Check Algorithm

The self-check algorithm works as follows.

1. Build the verification prompt with the goal and
   the result.
2. Call the model router.
3. Parse the response.
4. Validate the response through struct + go-playground/validator (or hand-rolled tag validation).
5. Return the verification result.

### 9.2 Test Run Algorithm

The test run algorithm works as follows.

1. Validate the arguments.
2. Determine the runner from the test command.
3. Run the test command through the subprocess
   runner.
4. Parse the output through the per-runner parser.
5. Return the test result.

## 10. Dependencies

### 10.1 Runtime Dependencies

- Go 1.23
- text/template (stdlib)
- aios-core
- aios-tools
- aios-models
- aios-sandbox

### 10.2 Module Dependencies

The aios-verify module depends on the following
internal modules.

- aios-core
- aios-tools
- aios-models
- aios-sandbox

The aios-verify module is depended on by the
following internal modules.

- aios-runtime
- aios-planner

## 11. Deployment

The module compiles into the monorepo under `internal/<module>/`. The
package name is `github.com/<org>/aios/internal/verify`. The default
verification prompt is bundled with the package.

## 12. Risks and Mitigations

| Risk | Impact | Probability | Mitigation |
| --- | --- | --- | --- |
| The self-check model is wrong. | High | Medium | The tool supports a custom prompt. The operator can override the model. |
| The test runner output changes. | Medium | Medium | The tool supports a custom parser. The test suite covers every default parser. |
| The verification result is not typed. | Medium | Low | The component uses struct + go-playground/validator (or hand-rolled tag validation). The result is validated. |
