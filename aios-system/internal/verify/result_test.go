package verify

import (
	"errors"
	"testing"
)

func TestStatus_IsValid(t *testing.T) {
	for _, s := range AllStatuses() {
		if !s.IsValid() {
			t.Errorf("expected %q to be valid", s)
		}
	}
	if Status("nope").IsValid() {
		t.Errorf("expected unknown status to be invalid")
	}
}

func TestAllStatuses_Deterministic(t *testing.T) {
	want := []string{"pass", "fail", "inconclusive"}
	got := AllStatuses()
	if len(got) != len(want) {
		t.Fatalf("len mismatch: %d", len(got))
	}
	for i := range want {
		if string(got[i]) != want[i] {
			t.Errorf("position %d: got %q want %q", i, got[i], want[i])
		}
	}
}

func TestVerificationResult_Validate(t *testing.T) {
	tests := []struct {
		name   string
		result VerificationResult
		ok     bool
	}{
		{"pass", VerificationResult{Status: StatusPass, Reason: "looks good"}, true},
		{"fail with suggestions", VerificationResult{Status: StatusFail, Reason: "missing field", Suggestions: []string{"add field"}}, true},
		{"inconclusive", VerificationResult{Status: StatusInconclusive, Reason: "cannot decide"}, true},
		{"bad status", VerificationResult{Status: "nope", Reason: "x"}, false},
		{"empty reason", VerificationResult{Status: StatusPass, Reason: ""}, false},
		{"whitespace reason", VerificationResult{Status: StatusPass, Reason: "   "}, false},
		{"empty suggestion", VerificationResult{Status: StatusFail, Reason: "x", Suggestions: []string{""}}, false},
		{"whitespace suggestion", VerificationResult{Status: StatusFail, Reason: "x", Suggestions: []string{"  "}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.result.Validate()
			if tt.ok && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !tt.ok && err == nil {
				t.Errorf("expected error, got nil")
			}
		})
	}
}

func TestVerificationResult_Validate_Nil(t *testing.T) {
	var r *VerificationResult
	if err := r.Validate(); err == nil {
		t.Fatalf("expected error for nil")
	}
}

func TestTestResult_Validate(t *testing.T) {
	tests := []struct {
		name   string
		result TestResult
		ok     bool
	}{
		{"all zero", TestResult{}, true},
		{"normal", TestResult{Passed: 5, Failed: 1, Skipped: 2, DurationMs: 100, Failures: []TestFailure{{Name: "T", Message: "M"}}}, true},
		{"negative passed", TestResult{Passed: -1}, false},
		{"negative failed", TestResult{Failed: -1}, false},
		{"negative skipped", TestResult{Skipped: -1}, false},
		{"negative duration", TestResult{DurationMs: -1}, false},
		{"invalid failure", TestResult{Failures: []TestFailure{{Name: "", Message: "M"}}}, false},
		{"invalid failure message", TestResult{Failures: []TestFailure{{Name: "T", Message: ""}}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.result.Validate()
			if tt.ok && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !tt.ok && err == nil {
				t.Errorf("expected error, got nil")
			}
		})
	}
}

func TestTestResult_Validate_Nil(t *testing.T) {
	var r *TestResult
	if err := r.Validate(); err == nil {
		t.Fatalf("expected error for nil")
	}
}

func TestTestFailure_Validate(t *testing.T) {
	f := &TestFailure{Name: "TestFoo", Message: "boom"}
	if err := f.Validate(); err != nil {
		t.Errorf("unexpected: %v", err)
	}
	if err := (&TestFailure{Name: "", Message: "x"}).Validate(); err == nil {
		t.Errorf("expected name error")
	}
	if err := (&TestFailure{Name: "x", Message: ""}).Validate(); err == nil {
		t.Errorf("expected message error")
	}
}

func TestResultError_IsTarget(t *testing.T) {
	e := &ResultError{Field: "status", Reason: "bad"}
	if !errors.Is(e, ErrResult) {
		t.Errorf("expected Is ErrResult")
	}
	if e.Error() == "" {
		t.Errorf("expected non-empty error")
	}
}
