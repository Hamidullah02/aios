# aios-report — Implementation Plan

## 1. Purpose

This document states the implementation plan for the
aios-report module.

## 2. Scope

The scope of this document is the aios-report module.
The module depends on aios-core.

## 3. Reference Documents

- aios-report PRD. See `docs/modules/aios-report/PRD.md`.
- aios-report SRS. See `docs/modules/aios-report/SRS.md`.
- aios-report TRD. See `docs/modules/aios-report/TRD.md`.
- aios-report Application Flow. See
  `docs/modules/aios-report/APP-FLOW.md`.
- aios-report Wireframes. See
  `docs/modules/aios-report/WIREFRAMES.md`.

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

### Phase 2: Schema

1. Create the file `internal/report/schema.py`.
2. Create the file `tests/test_schema.py`.
3. Implement the `Report` class.
4. Implement the `Step` class.
5. Write the test cases.

The deliverable is the schema with tests.

### Phase 3: JSON Renderer

1. Create the file `internal/report/json_renderer.py`.
2. Create the file `tests/test_json_renderer.py`.
3. Implement the `JsonRenderer` class.
4. Write the test cases.

The deliverable is the JSON renderer with tests.

### Phase 4: Markdown Renderer

1. Create the file
   `internal/report/markdown_renderer.py`.
2. Create the file
   `tests/test_markdown_renderer.py`.
3. Implement the `MarkdownRenderer` class.
4. Write the test cases.

The deliverable is the Markdown renderer with tests.

### Phase 5: HTML Renderer

1. Create the file `internal/report/html_renderer.py`.
2. Create the file `tests/test_html_renderer.py`.
3. Implement the `HtmlRenderer` class.
4. Implement the HTML template.
5. Write the test cases.

The deliverable is the HTML renderer with tests.

### Phase 6: Quality Gates

1. Run the test suite with `go test`.
2. Run the coverage report with `go test -cover`.
3. Run the type check with `go vet ./internal/report`.
4. Run the linter with `gofmt + go vet check`.
5. Fix every failure.

The deliverable is a green test suite.

## 6. Deliverables

- The source files in `internal/report/`.
- The test files in `tests/`.
- The `go.mod` file.
- The `README.md` file.
- The HTML template in
  `internal/report/templates/report.html`.

## 7. Test Plan

### 7.1 Unit Tests

- `tests/test_schema.py` covers
  `internal/report/schema.py`.
- `tests/test_json_renderer.py` covers
  `internal/report/json_renderer.py`.
- `tests/test_markdown_renderer.py` covers
  `internal/report/markdown_renderer.py`.
- `tests/test_html_renderer.py` covers
  `internal/report/html_renderer.py`.

### 7.2 Test Cases

The test cases include the following items.

- The report schema validates.
- The JSON renderer returns valid JSON.
- The JSON renderer is deterministic.
- The Markdown renderer returns valid Markdown.
- The Markdown renderer uses tables.
- The HTML renderer returns valid HTML.
- The HTML renderer uses the template.

## 8. Merge Plan

### 8.1 Pre-Merge Checklist

1. The test suite has a 100 percent pass rate.
2. The coverage report shows 100 percent coverage for
   the public functions.
3. The `go vet` report shows no errors.
4. The `gofmt + go vet` report shows no errors.

### 8.2 Merge Steps

1. Create a branch named `merge/aios-report`.
2. Move the files from `modules/aios-report/` to the
   target locations in the main repository.
3. Run the test suite on the main repository.
4. Open a pull request.
5. Merge after the review is approved.

## 9. Risk Register

| Risk | Impact | Probability | Mitigation |
| --- | --- | --- | --- |
| The JSON output is not deterministic. | Medium | Low | The renderer sorts the keys. The renderer has a test case. |
| The HTML template is missing. | Medium | Low | The renderer raises an error. The template is bundled with the package. |
| The Markdown has special characters. | Medium | Low | The renderer escapes the special characters. The renderer has a test case. |

## 10. Schedule

| Phase | Duration | Dependency |
| --- | --- | --- |
| Phase 1: Project Skeleton | 1 day | nil |
| Phase 2: Schema | 1 day | Phase 1 |
| Phase 3: JSON Renderer | 1 day | Phase 2 |
| Phase 4: Markdown Renderer | 1 day | Phase 2 |
| Phase 5: HTML Renderer | 2 days | Phase 2 |
| Phase 6: Quality Gates | 1 day | All phases |

The total duration is 7 days.
