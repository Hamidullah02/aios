# aios-report — Technical Requirements Document

## 1. Purpose

This document states the technical approach for the
aios-report module.

## 2. Scope

The scope of this document is the aios-report module.
The module is a Go package.

## 3. Reference Documents

- aios-report PRD. See `docs/modules/aios-report/PRD.md`.
- aios-report SRS. See `docs/modules/aios-report/SRS.md`.
- aios-core TRD. See `docs/modules/aios-core/TRD.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| JSON | The JavaScript Object Notation. |
| Markdown | A lightweight markup language. |
| HTML | The HyperText Markup Language. |
| text/template (stdlib) | The Go standard-library template engine. |

## 5. Architecture

The aios-report module has four components.

- `aios.report.schema` — the report schema.
- `aios.report.json_renderer` — the JSON renderer.
- `aios.report.markdown_renderer` — the Markdown
  renderer.
- `aios.report.html_renderer` — the HTML renderer.

The four components depend on aios-core. The
components depend on each other in this order. The
schema has no dependencies. The renderers depend on
the schema.

## 6. Components

### 6.1 Schema

The schema component uses struct + go-playground/validator (or hand-rolled tag validation). The component
defines the `Report` and the `Step` classes.

### 6.2 JSON Renderer

The JSON renderer uses struct + go-playground/validator (or hand-rolled tag validation) serialization. The
renderer sorts the keys. The renderer uses 2-space
indentation. The renderer returns the JSON string.

### 6.3 Markdown Renderer

The Markdown renderer uses string templates. The
renderer builds the Markdown through f-strings. The
renderer uses tables for parallel data. The renderer
uses lists for sequential data.

### 6.4 HTML Renderer

The HTML renderer uses text/template (stdlib). The renderer loads
the template. The renderer renders the template with
the report. The renderer returns the HTML string.

## 7. Data Structures

### 7.1 Report

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `task_id` | string | yes | The unique identifier. |
| `plan_id` | string | yes | The plan id. |
| `started_at` | string | yes | The ISO 8601 start time. |
| `ended_at` | string | yes | The ISO 8601 end time. |
| `status` | string | yes | The status of the run. |
| `steps` | list of Step | yes | The steps of the run. |
| `cost_usd` | float | yes | The cost in USD. |
| `input_tokens` | int | yes | The input token count. |
| `output_tokens` | int | yes | The output token count. |
| `approvals` | list of string | no | The approval ids. |

### 7.2 Step

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `id` | string | yes | The unique identifier. |
| `tool` | string | yes | The tool name. |
| `started_at` | string | yes | The ISO 8601 start time. |
| `ended_at` | string | yes | The ISO 8601 end time. |
| `status` | string | yes | The status of the step. |
| `input` | dict | yes | The input of the step. |
| `output` | any | no | The output of the step. |

## 8. Interfaces

### 8.1 Renderer

```go
type Renderer(interface):
    """The renderer interface."""

    def render(self, report: Report) -> str:
        """Render the report.

        Args:
            report: The report to render.

        Returns:
            The rendered report.
        """
```

## 9. Algorithms

### 9.1 JSON Render Algorithm

The JSON render algorithm works as follows.

1. Serialize the report through struct + go-playground/validator (or hand-rolled tag validation).
2. Sort the keys.
3. Use 2-space indentation.
4. Return the JSON string.

### 9.2 Markdown Render Algorithm

The Markdown render algorithm works as follows.

1. Build the header with the task id, the status,
   the duration, and the cost.
2. Build the steps table.
3. Build the approvals list.
4. Return the Markdown string.

### 9.3 HTML Render Algorithm

The HTML render algorithm works as follows.

1. Load the template.
2. Render the template with the report.
3. Return the HTML string.

## 10. Dependencies

### 10.1 Runtime Dependencies

- Go 1.23
- text/template (stdlib)
- aios-core

### 10.2 Module Dependencies

The aios-report module depends on the following
internal modules.

- aios-core

The aios-report module is depended on by the
following internal modules.

- aios-cli

## 11. Deployment

The module compiles into the monorepo under `internal/<module>/`. The
package name is `github.com/<org>/aios/internal/report`. The HTML template is
loaded from `internal/report/templates/report.html`.

## 12. Risks and Mitigations

| Risk | Impact | Probability | Mitigation |
| --- | --- | --- | --- |
| The JSON output is not deterministic. | Medium | Low | The renderer sorts the keys. The renderer has a test case. |
| The HTML template is missing. | Medium | Low | The renderer raises an error. The template is bundled with the package. |
| The Markdown has special characters. | Medium | Low | The renderer escapes the special characters. The renderer has a test case. |
