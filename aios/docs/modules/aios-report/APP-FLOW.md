# aios-report — Application Flow

## 1. Purpose

This document describes the application flow for the
aios-report module.

## 2. Scope

The scope of this document is the aios-report module.
The module is a library.

## 3. Reference Documents

- aios-report PRD. See `docs/modules/aios-report/PRD.md`.
- aios-report SRS. See `docs/modules/aios-report/SRS.md`.
- aios-report TRD. See `docs/modules/aios-report/TRD.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Caller | The code that calls the renderer. |

## 5. User Flow

The aios-report module has no user-facing flow. The
user views the report through a file or a browser.

## 6. System Flow

### 6.1 JSON Render Flow

The JSON render flow runs as follows.

1. The caller builds the `Report`.
2. The caller calls `json_renderer.render`.
3. The renderer serializes the report.
4. The renderer returns the JSON string.

### 6.2 Markdown Render Flow

The Markdown render flow runs as follows.

1. The caller builds the `Report`.
2. The caller calls `markdown_renderer.render`.
3. The renderer builds the Markdown.
4. The renderer returns the Markdown string.

### 6.3 HTML Render Flow

The HTML render flow runs as follows.

1. The caller builds the `Report`.
2. The caller calls `html_renderer.render`.
3. The renderer loads the template.
4. The renderer renders the template.
5. The renderer returns the HTML string.

## 7. Error Flows

### 7.1 Report Render Error

The report render error flow runs as follows.

1. The renderer fails.
2. The renderer raises `ReportRenderError`.
3. The caller catches the error.
4. The caller reports the error.

## 8. Sequence Diagrams

### 8.1 Render Sequence

The render sequence has the steps below.

1. The CLI builds the report.
2. The CLI calls the renderer.
3. The renderer returns the string.
4. The CLI writes the string to a file.
