# aios-report

The aios-report module turns a run of a plan into a
report. The package is a single Go package at
`internal/report/` and depends only on aios-core (for
the canonical-JSON encoder) and the Go standard
library (text/template, html/template, embed).

## Public surface

- `Report` and `Step` — the typed data contracts.
  The Report describes one run. The Step describes
  one step in the run.
- `Status` — the run-status enum (succeeded, failed,
  canceled).
- `Renderer` — the renderer interface every concrete
  renderer implements.
- `JSONRenderer` — the deterministic JSON renderer.
- `MarkdownRenderer` — the human-readable Markdown
  renderer.
- `HTMLRenderer` — the HTML5 renderer with the
  bundled report.html template.
- `ReportError` — the validation error.
- `ReportRenderError` — the render error.

## Usage

```go
import "github.com/Hamidullah02/aios/aios-system/internal/report"

r := report.NewJSONRenderer()
out, err := r.Render(&report.Report{ ... })

m := report.NewMarkdownRenderer()
md, err := m.Render(&report.Report{ ... })

h := report.NewHTMLRenderer()
html, err := h.Render(&report.Report{ ... })
```

The JSON renderer sorts keys at every level and
indents by 2 spaces. The Markdown renderer uses
tables for parallel data (steps) and lists for
sequential data (approvals). The HTML renderer loads
the bundled `templates/report.html` template and
escapes user-controlled fields through
`html/template`.

## Algorithm

The render algorithm is:

1. Validate the report.
2. Serialize the report through the renderer's
   strategy.
3. Return the rendered string.

The renderers are pure functions of the report. The
JSON renderer is deterministic: the same report
produces the same bytes.

## HTML template

The HTML template is bundled through
`go:embed`. The template lives at
`internal/report/templates/report.html`. The template
is parseable by `html/template` and renders the
report in valid HTML5.

## Tests

Coverage on the package is 94.9%. All public
functions are at 100%; the small gaps are the
default-template parse-error path (unreachable from
outside the package) and the `Unwrap` method on
`ReportRenderError` (only reachable through
`errors.Unwrap`, which the package's tests do not
call).
