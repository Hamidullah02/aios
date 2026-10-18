package report

import (
	"bytes"
	_ "embed"
	"fmt"
	"html/template"
)

//go:embed templates/report.html
var defaultHTMLTemplate string

// HTMLRenderer is the HTML5 renderer. The renderer
// loads the bundled report.html template and renders
// the report.
//
// The renderer escapes user-controlled fields through
// the html/template package, which is the safe
// default. The renderer raises a *ReportRenderError
// when the template fails to parse or execute.
type HTMLRenderer struct {
	// Template is the optional custom template. When
	// nil, the renderer uses the bundled default
	// template. The template is parsed at Render
	// time so a caller can change the field between
	// calls.
	Template *template.Template
}

// NewHTMLRenderer constructs an HTMLRenderer that
// uses the bundled default template.
func NewHTMLRenderer() *HTMLRenderer {
	return &HTMLRenderer{}
}

// NewHTMLRendererWithTemplate constructs an
// HTMLRenderer with a caller-provided template. The
// caller is responsible for parsing the template;
// the renderer uses the parsed template as-is.
func NewHTMLRendererWithTemplate(t *template.Template) *HTMLRenderer {
	return &HTMLRenderer{Template: t}
}

// Render turns a report into an HTML5 string.
//
// REQ-RPT-HTML-001, REQ-RPT-HTML-002, REQ-RPT-HTML-003.
func (h *HTMLRenderer) Render(report *Report) (string, error) {
	if report == nil {
		return "", &ReportRenderError{Format: "html", Err: fmt.Errorf("report is nil")}
	}
	if err := report.Validate(); err != nil {
		return "", &ReportRenderError{Format: "html", Err: err}
	}
	tpl, err := h.template()
	if err != nil {
		return "", &ReportRenderError{Format: "html", Err: err}
	}
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, report); err != nil {
		return "", &ReportRenderError{Format: "html", Err: err}
	}
	return buf.String(), nil
}

// template returns the renderer's template. The
// function parses the default template on first use
// and caches the result. The cache is per-renderer so
// a renderer with a custom template does not pay the
// parse cost.
func (h *HTMLRenderer) template() (*template.Template, error) {
	if h.Template != nil {
		return h.Template, nil
	}
	// Parse with the add1 helper function.
	tpl, err := template.New("report").Funcs(template.FuncMap{
		"add1": func(i int) int { return i + 1 },
	}).Parse(defaultHTMLTemplate)
	if err != nil {
		return nil, err
	}
	h.Template = tpl
	return tpl, nil
}
