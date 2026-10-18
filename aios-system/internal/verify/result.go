package verify

import (
	"fmt"
	"strings"
)

// Status is the typed enum for the
// verification status (REQ-VRF-RES-002).
type Status string

const (
	// StatusPass is the success status.
	StatusPass Status = "pass"
	// StatusFail is the failure status.
	StatusFail Status = "fail"
	// StatusInconclusive is the
	// "cannot decide" status.
	StatusInconclusive Status = "inconclusive"
)

// IsValid reports whether s is one of the
// three known statuses.
func (s Status) IsValid() bool {
	switch s {
	case StatusPass, StatusFail, StatusInconclusive:
		return true
	}
	return false
}

// AllStatuses returns the three known statuses
// in deterministic order.
func AllStatuses() []Status {
	return []Status{StatusPass, StatusFail, StatusInconclusive}
}

// VerificationResult is the typed result of
// the SelfCheck tool (REQ-VRF-RES-001).
type VerificationResult struct {
	// Status is the verification status
	// (pass, fail, inconclusive).
	Status Status `json:"status"`
	// Reason is the human-readable reason
	// the model gave for the status.
	Reason string `json:"reason"`
	// Suggestions is the list of suggested
	// fixes. The list is empty for pass
	// results; non-empty for fail results.
	Suggestions []string `json:"suggestions,omitempty"`
}

// Validate checks the result is well-formed.
// The function is the gate the parser and the
// self-check tool call.
func (r *VerificationResult) Validate() error {
	if r == nil {
		return &ResultError{Field: "result", Reason: "is nil"}
	}
	if !r.Status.IsValid() {
		return &ResultError{
			Field:  "status",
			Reason: fmt.Sprintf("must be one of %v", allStatusNames()),
		}
	}
	if strings.TrimSpace(r.Reason) == "" {
		return &ResultError{
			Field:  "reason",
			Reason: "is required",
		}
	}
	// The Suggestions list, when present,
	// must contain non-empty strings.
	for i, s := range r.Suggestions {
		if strings.TrimSpace(s) == "" {
			return &ResultError{
				Field:  fmt.Sprintf("suggestions[%d]", i),
				Reason: "must be a non-empty string",
			}
		}
	}
	return nil
}

// allStatusNames is the helper the
// validation-error message uses.
func allStatusNames() []string {
	ss := AllStatuses()
	out := make([]string, len(ss))
	for i, s := range ss {
		out[i] = string(s)
	}
	return out
}

// TestFailure is the typed record of a single
// test failure (REQ-VRF-DAT-002).
type TestFailure struct {
	// Name is the test name (e.g.
	// "TestFoo/TestBar").
	Name string `json:"name"`
	// Message is the failure message.
	Message string `json:"message"`
	// Traceback is the optional stack
	// trace.
	Traceback string `json:"traceback,omitempty"`
}

// Validate checks the failure is well-formed.
func (f *TestFailure) Validate() error {
	if f == nil {
		return &ResultError{Field: "failure", Reason: "is nil"}
	}
	if strings.TrimSpace(f.Name) == "" {
		return &ResultError{Field: "name", Reason: "is required"}
	}
	if strings.TrimSpace(f.Message) == "" {
		return &ResultError{Field: "message", Reason: "is required"}
	}
	return nil
}

// TestResult is the typed result of the
// RunTests tool (REQ-VRF-DAT-001).
type TestResult struct {
	// Passed is the number of passing
	// tests.
	Passed int `json:"passed"`
	// Failed is the number of failing
	// tests.
	Failed int `json:"failed"`
	// Skipped is the number of skipped
	// tests.
	Skipped int `json:"skipped"`
	// DurationMs is the total duration in
	// milliseconds.
	DurationMs int64 `json:"duration_ms"`
	// Failures is the list of failing
	// tests.
	Failures []TestFailure `json:"failures"`
	// Command is the test command the
	// tool ran. The field is informational;
	// the tool echoes it back to the
	// caller.
	Command string `json:"command,omitempty"`
	// Runner is the name of the parser
	// the tool used (e.g. "go-test",
	// "jest"). The field is informational.
	Runner string `json:"runner,omitempty"`
}

// Validate checks the result is well-formed.
// The function is the gate the per-runner
// parsers and the run-tests tool call.
func (r *TestResult) Validate() error {
	if r == nil {
		return &ResultError{Field: "result", Reason: "is nil"}
	}
	if r.Passed < 0 {
		return &ResultError{Field: "passed", Reason: "must be non-negative"}
	}
	if r.Failed < 0 {
		return &ResultError{Field: "failed", Reason: "must be non-negative"}
	}
	if r.Skipped < 0 {
		return &ResultError{Field: "skipped", Reason: "must be non-negative"}
	}
	if r.DurationMs < 0 {
		return &ResultError{Field: "duration_ms", Reason: "must be non-negative"}
	}
	for i := range r.Failures {
		if err := r.Failures[i].Validate(); err != nil {
			// The failure's own error
			// already names the field; we
			// wrap it for context.
			return &ResultError{
				Field:  fmt.Sprintf("failures[%d]", i),
				Reason: err.Error(),
			}
		}
	}
	return nil
}
