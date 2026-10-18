package report

import (
	"errors"
	"strings"
	"testing"
)

// TestMarkdownRendererRender covers the happy path.
// The output is a Markdown document with the expected
// sections.
func TestMarkdownRendererRender(t *testing.T) {
	r := NewMarkdownRenderer()
	out, err := r.Render(sampleReportForRenderer())
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	// Required sections.
	wantSections := []string{
		"# Run Report — task-001",
		"**Status:** SUCCEEDED",
		"**Cost:** $1.234",
		"**Tokens:** 12345 in / 4567 out",
		"## Plan",
		"## Steps",
		"## Approvals",
		"| # | Tool | Status | Started | Ended |",
		"| 1 | read_file | succeeded |",
		"- apr-9f2a",
		"- apr-3a1b",
	}
	for _, s := range wantSections {
		if !strings.Contains(out, s) {
			t.Fatalf("expected %q in output, got:\n%s", s, out)
		}
	}
}

// TestMarkdownRendererNil covers the nil-report
// branch.
func TestMarkdownRendererNil(t *testing.T) {
	r := NewMarkdownRenderer()
	_, err := r.Render(nil)
	if err == nil {
		t.Fatalf("expected error for nil report")
	}
	var re *ReportRenderError
	if !errors.As(err, &re) {
		t.Fatalf("expected *ReportRenderError, got %T", err)
	}
	if re.Format != "markdown" {
		t.Fatalf("expected format=markdown, got %q", re.Format)
	}
}

// TestMarkdownRendererInvalid covers the invalid-
// report branch.
func TestMarkdownRendererInvalid(t *testing.T) {
	r := NewMarkdownRenderer()
	rep := sampleReportForRenderer()
	rep.TaskID = ""
	_, err := r.Render(rep)
	if err == nil {
		t.Fatalf("expected error")
	}
	var re *ReportRenderError
	if !errors.As(err, &re) {
		t.Fatalf("expected *ReportRenderError, got %T", err)
	}
}

// TestMarkdownRendererNoApprovals covers the
// empty-approvals branch. The renderer omits the
// Approvals section.
func TestMarkdownRendererNoApprovals(t *testing.T) {
	r := NewMarkdownRenderer()
	rep := sampleReportForRenderer()
	rep.Approvals = nil
	out, err := r.Render(rep)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if strings.Contains(out, "## Approvals") {
		t.Fatalf("expected no Approvals section, got:\n%s", out)
	}
}

// TestMarkdownRendererNoSteps covers the empty-steps
// branch. The renderer emits a "no steps" placeholder.
func TestMarkdownRendererNoSteps(t *testing.T) {
	r := NewMarkdownRenderer()
	rep := sampleReportForRenderer()
	rep.Steps = nil
	out, err := r.Render(rep)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(out, "_No steps._") {
		t.Fatalf("expected no-steps placeholder, got:\n%s", out)
	}
}

// TestMarkdownRendererEscapesPipes covers the
// pipe-escape branch. A pipe in the task id is
// escaped so it does not break the table.
func TestMarkdownRendererEscapesPipes(t *testing.T) {
	r := NewMarkdownRenderer()
	rep := sampleReportForRenderer()
	rep.TaskID = "task|x"
	out, err := r.Render(rep)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(out, `task\|x`) {
		t.Fatalf("expected escaped pipe, got:\n%s", out)
	}
}

// TestMarkdownRendererEscapesBackticks covers the
// backtick-escape branch.
func TestMarkdownRendererEscapesBackticks(t *testing.T) {
	r := NewMarkdownRenderer()
	rep := sampleReportForRenderer()
	rep.TaskID = "task`x"
	out, err := r.Render(rep)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(out, "task\\`x") {
		t.Fatalf("expected escaped backtick, got:\n%s", out)
	}
}

// TestMarkdownRendererDuration covers the duration
// computation in the header.
func TestMarkdownRendererDuration(t *testing.T) {
	r := NewMarkdownRenderer()
	rep := sampleReportForRenderer()
	rep.StartedAt = "2025-01-15T10:30:00Z"
	rep.EndedAt = "2025-01-15T10:30:42.5Z"
	out, err := r.Render(rep)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(out, "**Duration:** 42.5s") {
		t.Fatalf("expected duration in output, got:\n%s", out)
	}
}

// TestMarkdownRendererImplementsRenderer covers the
// interface assertion.
func TestMarkdownRendererImplementsRenderer(t *testing.T) {
	var _ Renderer = NewMarkdownRenderer()
}

// TestEscapeMarkdown covers the escape function in
// isolation.
func TestEscapeMarkdown(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"plain", "plain"},
		{"with|pipe", `with\|pipe`},
		{"with`tick", "with\\`tick"},
		{"with\nnewline", "with newline"},
		{"with\rcr", "with cr"},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			if got := escapeMarkdown(tc.in); got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}
}
