package report

import (
	"fmt"
	"strings"
)

// MarkdownRenderer is the Markdown renderer. The
// renderer builds the Markdown through string
// concatenation. The renderer uses tables for
// parallel data (steps) and lists for sequential data
// (approvals), per REQ-RPT-MD-003.
//
// The renderer escapes special Markdown characters
// in user-controlled fields (task id, plan id,
// approval ids, tool names). The escape is minimal:
// backticks and pipes are escaped because they would
// break the steps table.
type MarkdownRenderer struct{}

// NewMarkdownRenderer constructs a MarkdownRenderer.
func NewMarkdownRenderer() *MarkdownRenderer {
	return &MarkdownRenderer{}
}

// Render turns a report into a Markdown string.
//
// REQ-RPT-MD-001, REQ-RPT-MD-002, REQ-RPT-MD-003.
func (m *MarkdownRenderer) Render(report *Report) (string, error) {
	if report == nil {
		return "", &ReportRenderError{Format: "markdown", Err: fmt.Errorf("report is nil")}
	}
	if err := report.Validate(); err != nil {
		return "", &ReportRenderError{Format: "markdown", Err: err}
	}
	var b strings.Builder
	m.writeHeader(&b, report)
	m.writeSteps(&b, report)
	m.writeApprovals(&b, report)
	return b.String(), nil
}

// writeHeader writes the report header. The header
// carries the task id, the status, the duration, the
// cost, and the token usage.
func (m *MarkdownRenderer) writeHeader(b *strings.Builder, r *Report) {
	fmt.Fprintf(b, "# Run Report — %s\n\n", escapeMarkdown(r.TaskID))
	fmt.Fprintf(b, "**Status:** %s\n", strings.ToUpper(string(r.Status)))
	fmt.Fprintf(b, "**Duration:** %.1fs\n", r.DurationSeconds())
	fmt.Fprintf(b, "**Cost:** $%.3f\n", r.CostUSD)
	fmt.Fprintf(b, "**Tokens:** %d in / %d out\n\n", r.InputTokens, r.OutputTokens)
	fmt.Fprintf(b, "## Plan\n\n")
	fmt.Fprintf(b, "- Plan id: %s\n", escapeMarkdown(r.PlanID))
	fmt.Fprintf(b, "- Started: %s\n", escapeMarkdown(r.StartedAt))
	fmt.Fprintf(b, "- Ended: %s\n\n", escapeMarkdown(r.EndedAt))
}

// writeSteps writes the steps table. The table uses a
// fixed-width header and a row per step.
func (m *MarkdownRenderer) writeSteps(b *strings.Builder, r *Report) {
	fmt.Fprintf(b, "## Steps\n\n")
	if len(r.Steps) == 0 {
		fmt.Fprintf(b, "_No steps._\n\n")
		return
	}
	fmt.Fprintf(b, "| # | Tool | Status | Started | Ended |\n")
	fmt.Fprintf(b, "|---|------|--------|---------|-------|\n")
	for i, s := range r.Steps {
		fmt.Fprintf(b, "| %d | %s | %s | %s | %s |\n",
			i+1,
			escapeMarkdown(s.Tool),
			strings.ToLower(string(s.Status)),
			escapeMarkdown(s.StartedAt),
			escapeMarkdown(s.EndedAt),
		)
	}
	fmt.Fprintf(b, "\n")
}

// writeApprovals writes the approvals list. The list
// is a bulleted list; an empty list is omitted.
func (m *MarkdownRenderer) writeApprovals(b *strings.Builder, r *Report) {
	if len(r.Approvals) == 0 {
		return
	}
	fmt.Fprintf(b, "## Approvals\n\n")
	for _, a := range r.Approvals {
		fmt.Fprintf(b, "- %s\n", escapeMarkdown(a))
	}
	fmt.Fprintf(b, "\n")
}

// escapeMarkdown escapes characters that would
// otherwise break the Markdown structure. The
// function escapes pipes and backticks because they
// would break the steps table; newlines are replaced
// with spaces because the renderer emits one row per
// step. The escape is intentionally minimal — full
// Markdown escaping would make the report
// unreadable.
func escapeMarkdown(s string) string {
	s = strings.ReplaceAll(s, "|", "\\|")
	s = strings.ReplaceAll(s, "`", "\\`")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	return s
}
