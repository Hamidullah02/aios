package report

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

// TestJSONRendererRender covers the happy path. The
// output is a valid indented JSON document.
func TestJSONRendererRender(t *testing.T) {
	r := NewJSONRenderer()
	out, err := r.Render(sampleReportForRenderer())
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	// The output must be valid JSON.
	var decoded map[string]any
	if err := json.Unmarshal([]byte(out), &decoded); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, out)
	}
	// The output must use 2-space indentation.
	if !strings.Contains(out, "\n  \"") {
		t.Fatalf("expected 2-space indent, got %q", out)
	}
}

// TestJSONRendererDeterministic covers the
// determinism guarantee. Two renders of the same
// report produce the same bytes.
func TestJSONRendererDeterministic(t *testing.T) {
	r := NewJSONRenderer()
	rep := sampleReportForRenderer()
	a, err := r.Render(rep)
	if err != nil {
		t.Fatalf("Render a: %v", err)
	}
	b, err := r.Render(rep)
	if err != nil {
		t.Fatalf("Render b: %v", err)
	}
	if a != b {
		t.Fatalf("non-deterministic output:\n%s\nvs\n%s", a, b)
	}
}

// TestJSONRendererSortedKeys covers the sort-keys
// guarantee. The keys are in alphabetical order at
// every level.
func TestJSONRendererSortedKeys(t *testing.T) {
	r := NewJSONRenderer()
	out, err := r.Render(sampleReportForRenderer())
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	// The top-level keys appear in the output in
	// alphabetical order. The first keys should be
	// "approvals" (or "cost_usd") depending on
	// omitempty. The point is that adjacent keys are
	// in order: we expect "cost_usd" before
	// "ended_at", "ended_at" before "input_tokens",
	// etc.
	wantOrder := []string{
		"\"approvals\"",
		"\"cost_usd\"",
		"\"ended_at\"",
		"\"input_tokens\"",
		"\"output_tokens\"",
		"\"plan_id\"",
		"\"started_at\"",
		"\"status\"",
		"\"steps\"",
		"\"task_id\"",
	}
	prev := -1
	for _, key := range wantOrder {
		idx := strings.Index(out, key)
		if idx < 0 {
			t.Fatalf("expected %s in output, got:\n%s", key, out)
		}
		if idx <= prev {
			t.Fatalf("expected %s after previous key, got idx=%d prev=%d", key, idx, prev)
		}
		prev = idx
	}
}

// TestJSONRendererNil covers the nil-report branch.
func TestJSONRendererNil(t *testing.T) {
	r := NewJSONRenderer()
	_, err := r.Render(nil)
	if err == nil {
		t.Fatalf("expected error for nil report")
	}
	var re *ReportRenderError
	if !errors.As(err, &re) {
		t.Fatalf("expected *ReportRenderError, got %T", err)
	}
}

// TestJSONRendererInvalid covers the invalid-report
// branch. The renderer rejects a report that fails
// validation.
func TestJSONRendererInvalid(t *testing.T) {
	r := NewJSONRenderer()
	rep := sampleReportForRenderer()
	rep.TaskID = ""
	_, err := r.Render(rep)
	if err == nil {
		t.Fatalf("expected error for invalid report")
	}
	var re *ReportRenderError
	if !errors.As(err, &re) {
		t.Fatalf("expected *ReportRenderError, got %T", err)
	}
}

// TestJSONRendererCustomIndent covers the
// custom-indent branch. The renderer honors the
// Indent field.
func TestJSONRendererCustomIndent(t *testing.T) {
	r := &JSONRenderer{Indent: "    "}
	out, err := r.Render(sampleReportForRenderer())
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(out, "\n    \"") {
		t.Fatalf("expected 4-space indent, got %q", out)
	}
}

// TestJSONRendererEmptyIndent covers the empty-indent
// fallback. The renderer uses 2 spaces when Indent is
// empty.
func TestJSONRendererEmptyIndent(t *testing.T) {
	r := &JSONRenderer{}
	out, err := r.Render(sampleReportForRenderer())
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(out, "\n  \"") {
		t.Fatalf("expected 2-space indent fallback, got %q", out)
	}
}

// TestJSONRendererOmitsEmptyApprovals covers the
// omitempty branch. The renderer omits the approvals
// field when the list is empty.
func TestJSONRendererOmitsEmptyApprovals(t *testing.T) {
	r := NewJSONRenderer()
	rep := sampleReportForRenderer()
	rep.Approvals = nil
	out, err := r.Render(rep)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if strings.Contains(out, "approvals") {
		t.Fatalf("expected no 'approvals' in output, got:\n%s", out)
	}
}

// TestJSONRendererImplementsRenderer covers the
// interface assertion.
func TestJSONRendererImplementsRenderer(t *testing.T) {
	var _ Renderer = NewJSONRenderer()
}

// TestReportRenderErrorIs covers the errors.Is match
// for the typed error.
func TestReportRenderErrorIs(t *testing.T) {
	var err error = &ReportRenderError{Format: "json", Err: errors.New("boom")}
	if !errors.Is(err, &ReportRenderError{}) {
		t.Fatalf("expected errors.Is to match")
	}
}

// TestReportRenderErrorMessage covers the error
// format.
func TestReportRenderErrorMessage(t *testing.T) {
	err := (&ReportRenderError{Format: "json", Err: errors.New("boom")}).Error()
	if !strings.HasPrefix(err, "report: render:") {
		t.Fatalf("unexpected prefix: %q", err)
	}
	if !strings.HasSuffix(err, ".") {
		t.Fatalf("missing trailing period: %q", err)
	}
}

// TestReportRenderErrorUnwrapNil covers the nil-err
// branch. The error message ends with a period.
func TestReportRenderErrorUnwrapNil(t *testing.T) {
	err := (&ReportRenderError{Format: "json"}).Error()
	if !strings.HasSuffix(err, ".") {
		t.Fatalf("missing trailing period: %q", err)
	}
}

// sampleReportForRenderer is a small fixture for the
// renderer tests. The function returns a fresh copy
// so callers can mutate it without affecting other
// tests.
func sampleReportForRenderer() *Report {
	r := sampleReport()
	return &r
}
