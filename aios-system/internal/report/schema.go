// Package report — schema. The schema component
// defines the Report and Step types and their
// validation. The types are content addressed; the
// canonical JSON of a Report is what the JSON renderer
// uses.
package report

import (
	"fmt"
	"time"
)

// Status is the run-status enum. The values are the
// three states the orchestrator can leave the run in:
// succeeded, failed, or canceled. The renderer does
// not enforce the enum, but the schema does.
type Status string

const (
	StatusSucceeded Status = "succeeded"
	StatusFailed    Status = "failed"
	StatusCanceled  Status = "canceled"
)

// IsValid reports whether s is one of the three known
// statuses.
func (s Status) IsValid() bool {
	switch s {
	case StatusSucceeded, StatusFailed, StatusCanceled:
		return true
	}
	return false
}

// Report is the typed object that describes one run
// of a plan. The report is what the orchestrator hands
// the renderer after the run is done.
//
// Per REQ-RPT-DAT-001 the report carries the task id,
// the plan id, the start and end times, the status,
// the steps, the cost, the token usage, and the
// approvals. All fields except Approvals are required.
type Report struct {
	// TaskID is the unique identifier of the run. The
	// field is required.
	TaskID string `json:"task_id"`
	// PlanID is the plan that produced the run. The
	// field is required.
	PlanID string `json:"plan_id"`
	// StartedAt is the ISO 8601 start time. The field
	// is required.
	StartedAt string `json:"started_at"`
	// EndedAt is the ISO 8601 end time. The field is
	// required.
	EndedAt string `json:"ended_at"`
	// Status is the run status. The field is required
	// and must be one of the three known values.
	Status Status `json:"status"`
	// Steps is the ordered list of steps. The field is
	// required and may be empty.
	Steps []Step `json:"steps"`
	// CostUSD is the total cost in USD. The field is
	// required.
	CostUSD float64 `json:"cost_usd"`
	// InputTokens is the input token count. The field
	// is required.
	InputTokens int `json:"input_tokens"`
	// OutputTokens is the output token count. The field
	// is required.
	OutputTokens int `json:"output_tokens"`
	// Approvals is the list of approval ids that were
	// granted or denied during the run. The field is
	// optional.
	Approvals []string `json:"approvals,omitempty"`
}

// Validate checks the report is well-formed. The
// function returns the first validation error
// encountered. The validation rules are the
// "required" and "value" checks below; the renderer
// performs the format-specific checks (e.g. the JSON
// renderer checks the canonical encoding).
func (r *Report) Validate() error {
	if r == nil {
		return &ReportError{Op: "validate", Reason: "report is nil"}
	}
	if r.TaskID == "" {
		return &ReportError{Op: "validate", Reason: "task_id is required"}
	}
	if r.PlanID == "" {
		return &ReportError{Op: "validate", Reason: "plan_id is required"}
	}
	if r.StartedAt == "" {
		return &ReportError{Op: "validate", Reason: "started_at is required"}
	}
	if r.EndedAt == "" {
		return &ReportError{Op: "validate", Reason: "ended_at is required"}
	}
	if !r.Status.IsValid() {
		return &ReportError{Op: "validate", Reason: fmt.Sprintf("status must be one of succeeded, failed, canceled; got %q", r.Status)}
	}
	if r.CostUSD < 0 {
		return &ReportError{Op: "validate", Reason: "cost_usd must be non-negative"}
	}
	if r.InputTokens < 0 {
		return &ReportError{Op: "validate", Reason: "input_tokens must be non-negative"}
	}
	if r.OutputTokens < 0 {
		return &ReportError{Op: "validate", Reason: "output_tokens must be non-negative"}
	}
	for i := range r.Steps {
		if err := r.Steps[i].Validate(); err != nil {
			return &ReportError{Op: "validate", Reason: fmt.Sprintf("steps[%d]: %s", i, err.Error())}
		}
	}
	return nil
}

// Step is the typed object that describes one step in
// the run. Per the data requirements, the step has an
// id, a tool, a start and end time, a status, an
// input, and an optional output.
type Step struct {
	// ID is the unique identifier of the step. The
	// field is required.
	ID string `json:"id"`
	// Tool is the tool name. The field is required.
	Tool string `json:"tool"`
	// StartedAt is the ISO 8601 start time. The field
	// is required.
	StartedAt string `json:"started_at"`
	// EndedAt is the ISO 8601 end time. The field is
	// required.
	EndedAt string `json:"ended_at"`
	// Status is the step status. The field is required
	// and must be one of the three known values.
	Status Status `json:"status"`
	// Input is the input of the step. The field is
	// required and may be empty.
	Input map[string]any `json:"input"`
	// Output is the output of the step. The field is
	// optional.
	Output any `json:"output,omitempty"`
}

// Validate checks the step is well-formed.
func (s *Step) Validate() error {
	if s == nil {
		return &ReportError{Op: "validate", Reason: "step is nil"}
	}
	if s.ID == "" {
		return &ReportError{Op: "validate", Reason: "id is required"}
	}
	if s.Tool == "" {
		return &ReportError{Op: "validate", Reason: "tool is required"}
	}
	if s.StartedAt == "" {
		return &ReportError{Op: "validate", Reason: "started_at is required"}
	}
	if s.EndedAt == "" {
		return &ReportError{Op: "validate", Reason: "ended_at is required"}
	}
	if !s.Status.IsValid() {
		return &ReportError{Op: "validate", Reason: fmt.Sprintf("status must be one of succeeded, failed, canceled; got %q", s.Status)}
	}
	if s.Input == nil {
		return &ReportError{Op: "validate", Reason: "input is required (use an empty map for no input)"}
	}
	return nil
}

// DurationSeconds returns the duration of the run in
// seconds. The function parses the start and end
// timestamps. The timestamps are expected to be in a
// format Go's time.Parse can read; the renderer does
// not enforce a specific layout, so the function
// returns the raw string when parsing fails.
//
// The function is best-effort: it returns 0 when the
// timestamps cannot be parsed.
func (r *Report) DurationSeconds() float64 {
	// The renderer does not require RFC 3339, but the
	// orchestrator always produces RFC 3339. Try RFC
	// 3339 with nanoseconds first, then without.
	start, err := parseTimestamp(r.StartedAt)
	if err != nil {
		return 0
	}
	end, err := parseTimestamp(r.EndedAt)
	if err != nil {
		return 0
	}
	return end.Sub(start).Seconds()
}

// parseTimestamp parses an ISO 8601 timestamp. The
// function tries RFC 3339 with and without
// nanoseconds.
func parseTimestamp(s string) (time.Time, error) {
	layouts := []string{time.RFC3339Nano, time.RFC3339}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognized timestamp: %q", s)
}

// ReportError is the typed error for the report
// component. The error message includes the failing
// operation and the reason.
type ReportError struct {
	Op     string
	Reason string
}

func (e *ReportError) Error() string {
	return fmt.Sprintf("report: %s: %s.", e.Op, e.Reason)
}
