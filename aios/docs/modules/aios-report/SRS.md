# aios-report — Software Requirements Specification

## 1. Purpose

This document states the software requirements for the
aios-report module.

## 2. Scope

The scope of this document is the aios-report module.
The module is a library.

## 3. Reference Documents

- aios-report PRD. See `docs/modules/aios-report/PRD.md`.
- aios-core SRS. See `docs/modules/aios-core/SRS.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Report | A typed object that describes the run. |
| Renderer | A typed component that turns a report into a format. |

## 5. Functional Requirements

### 5.1 Report Schema

**REQ-RPT-SCH-001** The module shall provide the
`Report` schema. The schema shall validate the
task id, the plan, the start time, the end time, the
status, the steps, the cost, the token usage, and the
approvals.

**REQ-RPT-SCH-002** The module shall provide the
`Step` schema. The schema shall validate the step
id, the tool, the start time, the end time, the
status, the input, and the output.

### 5.2 JSON Renderer

**REQ-RPT-JSN-001** The module shall provide the
`JsonRenderer` class. The type shall implement the
`Renderer` protocol.

**REQ-RPT-JSN-002** The renderer shall return the
JSON-serialized report. The JSON shall be
deterministic.

### 5.3 Markdown Renderer

**REQ-RPT-MD-001** The module shall provide the
`MarkdownRenderer` class. The type shall implement
the `Renderer` protocol.

**REQ-RPT-MD-002** The renderer shall return the
Markdown-serialized report. The Markdown shall be
human-readable.

**REQ-RPT-MD-003** The renderer shall use tables
for parallel data. The renderer shall use lists for
sequential data.

### 5.4 HTML Renderer

**REQ-RPT-HTML-001** The module shall provide the
`HtmlRenderer` class. The type shall implement the
`Renderer` protocol.

**REQ-RPT-HTML-002** The renderer shall return the
HTML-serialized report. The HTML shall be valid
HTML5.

**REQ-RPT-HTML-003** The renderer shall support a
template. The default template is `report.html`.

## 6. Non-Functional Requirements

**REQ-RPT-NFR-001** The module shall run on Go 1.23
3.12.

**REQ-RPT-NFR-002** The module shall have 100 percent
coverage for the public functions.

## 7. Interface Requirements

**REQ-RPT-IFC-001** The module shall export the
following items from `aios.report`.

- `Report`
- `Step`
- `Renderer`
- `JsonRenderer`
- `MarkdownRenderer`
- `HtmlRenderer`

## 8. Data Requirements

**REQ-RPT-DAT-001** The `Report` schema shall
include the fields below.

- `task_id` (string)
- `plan_id` (string)
- `started_at` (string, ISO 8601)
- `ended_at` (string, ISO 8601)
- `status` (string: succeeded, failed, canceled)
- `steps` (list of Step)
- `cost_usd` (float)
- `input_tokens` (int)
- `output_tokens` (int)
- `approvals` (list of string, approval ids)

## 9. Error Handling

**REQ-RPT-ERR-001** The module shall raise
`ReportRenderError` when the renderer fails.

## 10. Acceptance Criteria

1. Every REQ in this document has a test case.
2. The test suite has a 100 percent pass rate.
3. The test suite has 100 percent coverage for the
   public functions.
4. The JSON renderer returns valid JSON.
5. The Markdown renderer returns valid Markdown.
6. The HTML renderer returns valid HTML.
