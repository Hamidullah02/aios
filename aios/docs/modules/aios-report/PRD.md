# aios-report — Product Requirements Document

## 1. Purpose

This document states the product requirements for the
aios-report module. The module is the report
generator. The generator turns a run into a report.
The report is in three formats: JSON, Markdown, and
HTML.

## 2. Scope

The aios-report module is in scope for this document.

- The report schema.
- The JSON renderer.
- The Markdown renderer.
- The HTML renderer.

The aios-report module is not in scope for this
document.

- The orchestrator. The orchestrator is in aios-runtime.
- The CLI. The CLI is in aios-cli.

## 3. Reference Documents

- Project design document, sections 8, 15.
- aios-core PRD. See `docs/modules/aios-core/PRD.md`.
- aios-runtime PRD. See `docs/modules/aios-runtime/PRD.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Report | A typed object that describes the run. |
| Renderer | A component that turns a report into a format. |
| Run | A single execution of a plan. |


## 5. Determinism

The whole point of aios is to make complex coding and large systems deterministic. The determinism stance of this module is **Highly deterministic**.

The report generator is a pure function of the audit log and the report template.

The module provides the following determinism guarantees.

- Reports are content addressed.

- The report format is a pure function of the audit log and the template.
## 6. User Personas

The user persona is a developer who needs to review
the run results.

## 7. User Stories

A developer wants to view the report as JSON. The
developer calls the JSON renderer. The renderer
returns the JSON.

A developer wants to view the report as Markdown.
The developer calls the Markdown renderer. The
renderer returns the Markdown.

A developer wants to view the report as HTML. The
developer calls the HTML renderer. The renderer
returns the HTML.

## 8. Functional Requirements Summary

The aios-report module provides the following
functions.

- The report schema.
- The JSON renderer.
- The Markdown renderer.
- The HTML renderer.

## 9. Non-Functional Requirements Summary

- The module runs on Go 1.23.
- The module depends on aios-core.
- The module has 100 percent test coverage for the
  public functions.

## 10. Success Criteria

- The JSON renderer returns valid JSON.
- The Markdown renderer returns valid Markdown.
- The HTML renderer returns valid HTML.
- The test suite has a 100 percent pass rate.

## 11. Out of Scope

- The user interface for the report.
- The remote access to the report.
- The historical reports.
