# aios-ask — Technical Requirements Document

## 1. Purpose

This document states the technical approach for the
aios-ask module.

## 2. Scope

The scope of this document is the aios-ask module.
The module is a Go package.

## 3. Reference Documents

- aios-ask PRD. See `docs/modules/aios-ask/PRD.md`.
- aios-ask SRS. See `docs/modules/aios-ask/SRS.md`.
- aios-core TRD. See `docs/modules/aios-core/TRD.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Callback | A function passed as a parameter. |
| Prompt | A text-based question for the user. |

## 5. Architecture

The aios-ask module has two components.

- `internal/ask/schema` — the question and the answer
  schemas.
- `internal/ask/tool` — the ask-user tool.

The two components depend on aios-core. The
components depend on each other in this order. The
schema has no dependencies. The tool depends on the
schema.

## 6. Components

### 6.1 Schema

The schema component uses struct + go-playground/validator (or hand-rolled tag validation). The component
defines the `Question`, the `Answer`, and the
`AskOption` classes.

### 6.2 Tool

The tool component uses the prompt callback. The
callback is an `async` function. The callback
accepts the question. The callback returns the
answer.

The tool provides the `run` method. The method
validates the arguments. The method calls the
callback. The method returns the answer.

The tool supports a timeout. The tool uses
`time.AfterFunc + a select on a context.Done channel` to enforce the timeout. The tool
returns the default answer when the timeout
expires.

## 7. Data Structures

### 7.1 Question

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `id` | string | yes | The unique identifier. |
| `prompt` | string | yes | The prompt for the operator. |
| `options` | list of AskOption | yes | The options for the question. |
| `timeout_seconds` | int | no | The timeout in seconds. |
| `default_option` | string | no | The id of the default option. |

### 7.2 AskOption

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `id` | string | yes | The unique identifier. |
| `label` | string | yes | The label of the option. |
| `description` | string | no | The description of the option. |

### 7.3 Answer

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `question_id` | string | yes | The id of the question. |
| `selected_option` | string | yes | The id of the selected option. |
| `timestamp` | string | yes | The ISO 8601 timestamp of the answer. |

## 8. Interfaces

### 8.1 AskUserTool

```go
type AskUserTool:
    """The ask-user tool."""

    def __init__(
        self,
        prompt_callback: Callable[[Question], Awaitable[Answer]],
    ) -> nil:
        """Create the ask-user tool.

        Args:
            prompt_callback: The callback that prompts the
                operator.
        """

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

### 9.1 Prompt Algorithm

The prompt algorithm works as follows.

1. Validate the arguments.
2. Build the `Question` from the arguments.
3. Call the prompt callback with the question.
4. When the callback returns, return the answer.
5. When the timeout expires, return the default
   answer.

## 10. Dependencies

### 10.1 Runtime Dependencies

- Go 1.23
- aios-core

### 10.2 Module Dependencies

The aios-ask module depends on the following
internal modules.

- aios-core

The aios-ask module is depended on by the following
internal modules.

- aios-tools (through the tool bus)
- aios-cli

## 11. Deployment

The module compiles into the monorepo under `internal/<module>/`. The
package name is `github.com/<org>/aios/internal/ask`. The module is registered
in the tool bus.

## 12. Risks and Mitigations

| Risk | Impact | Probability | Mitigation |
| --- | --- | --- | --- |
| The operator does not answer. | High | Medium | The tool has a timeout. The tool returns a default answer. |
| The question is too long. | Medium | Low | The tool truncates the prompt. The tool supports multiple screens. |
| The question has no options. | High | Low | The tool validates the options. The tool raises an error when the options are empty. |
