package report

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/Hamidullah02/aios/aios-system/internal/core/schemas"
)

// JSONRenderer is the JSON renderer. The renderer
// serializes the report through Go's encoding/json
// with sorted keys (per the schemas package's
// canonical-JSON encoder) and 2-space indentation.
//
// Per REQ-RPT-JSN-002 the output is deterministic:
// the same report produces the same bytes. The
// determinism comes from the schemas package's
// canonical encoder, which sorts keys at every level.
type JSONRenderer struct {
	// Indent is the indentation string. Defaults to
	// two spaces when empty.
	Indent string
}

// NewJSONRenderer constructs a JSONRenderer with the
// default 2-space indentation.
func NewJSONRenderer() *JSONRenderer {
	return &JSONRenderer{Indent: "  "}
}

// Render serializes the report as canonical JSON. The
// renderer validates the report first; an invalid
// report returns a *ReportRenderError.
//
// REQ-RPT-JSN-001, REQ-RPT-JSN-002.
func (r *JSONRenderer) Render(report *Report) (string, error) {
	if report == nil {
		return "", &ReportRenderError{Format: "json", Err: fmt.Errorf("report is nil")}
	}
	if err := report.Validate(); err != nil {
		return "", &ReportRenderError{Format: "json", Err: err}
	}
	indent := r.Indent
	if indent == "" {
		indent = "  "
	}
	// Use schemas.MarshalCanonical for the
	// deterministic sort. The function sorts keys at
	// every level, which is what REQ-RPT-JSN-002 asks
	// for. Then re-marshal the canonical output with
	// the desired indentation.
	canonical, err := schemas.MarshalCanonical(report)
	if err != nil {
		return "", &ReportRenderError{Format: "json", Err: err}
	}
	var buf bytes.Buffer
	if err := json.Indent(&buf, canonical, "", indent); err != nil {
		return "", &ReportRenderError{Format: "json", Err: err}
	}
	return buf.String(), nil
}
