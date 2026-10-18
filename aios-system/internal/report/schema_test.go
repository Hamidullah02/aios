package report

import (
	"strings"
	"testing"
)

// sampleReport returns a valid report the tests can
// adapt. The function is the canonical fixture for
// the report package.
func sampleReport() Report {
	return Report{
		TaskID:    "task-001",
		PlanID:    "plan-abc",
		StartedAt: "2025-01-15T10:30:00Z",
		EndedAt:   "2025-01-15T10:35:12.4Z",
		Status:    StatusSucceeded,
		Steps: []Step{
			{
				ID:        "step-1",
				Tool:      "read_file",
				StartedAt: "2025-01-15T10:30:01Z",
				EndedAt:   "2025-01-15T10:30:02.2Z",
				Status:    StatusSucceeded,
				Input:     map[string]any{"path": "internal/core/foo.go"},
			},
		},
		CostUSD:      1.234,
		InputTokens:  12345,
		OutputTokens: 4567,
		Approvals:    []string{"apr-9f2a", "apr-3a1b"},
	}
}

// TestStatusIsValid covers the status enum.
func TestStatusIsValid(t *testing.T) {
	for _, s := range []Status{StatusSucceeded, StatusFailed, StatusCanceled} {
		if !s.IsValid() {
			t.Fatalf("expected %q to be valid", s)
		}
	}
	for _, bad := range []Status{"", "pending", "SUCCEEDED", "Succeeded"} {
		if bad.IsValid() {
			t.Fatalf("expected %q to be invalid", bad)
		}
	}
}

// TestReportValidate covers the validation rules.
func TestReportValidate(t *testing.T) {
	cases := []struct {
		name    string
		report  Report
		wantErr bool
	}{
		{
			name:    "valid",
			report:  sampleReport(),
			wantErr: false,
		},
		{
			name:    "missing task id",
			report:  func() Report { r := sampleReport(); r.TaskID = ""; return r }(),
			wantErr: true,
		},
		{
			name:    "missing plan id",
			report:  func() Report { r := sampleReport(); r.PlanID = ""; return r }(),
			wantErr: true,
		},
		{
			name:    "missing started_at",
			report:  func() Report { r := sampleReport(); r.StartedAt = ""; return r }(),
			wantErr: true,
		},
		{
			name:    "missing ended_at",
			report:  func() Report { r := sampleReport(); r.EndedAt = ""; return r }(),
			wantErr: true,
		},
		{
			name:    "bad status",
			report:  func() Report { r := sampleReport(); r.Status = "pending"; return r }(),
			wantErr: true,
		},
		{
			name:    "negative cost",
			report:  func() Report { r := sampleReport(); r.CostUSD = -1; return r }(),
			wantErr: true,
		},
		{
			name:    "negative input tokens",
			report:  func() Report { r := sampleReport(); r.InputTokens = -1; return r }(),
			wantErr: true,
		},
		{
			name:    "negative output tokens",
			report:  func() Report { r := sampleReport(); r.OutputTokens = -1; return r }(),
			wantErr: true,
		},
		{
			name: "invalid step",
			report: func() Report {
				r := sampleReport()
				r.Steps = []Step{{}}
				return r
			}(),
			wantErr: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := tc.report
			err := r.Validate()
			if tc.wantErr && err == nil {
				t.Fatalf("expected error")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

// TestReportValidateNil covers the nil-receiver
// branch.
func TestReportValidateNil(t *testing.T) {
	var r *Report
	err := r.Validate()
	if err == nil {
		t.Fatalf("expected error for nil report")
	}
}

// TestStepValidate covers the step validation rules.
func TestStepValidate(t *testing.T) {
	cases := []struct {
		name    string
		step    Step
		wantErr bool
	}{
		{
			name:    "valid",
			step:    sampleReport().Steps[0],
			wantErr: false,
		},
		{
			name:    "missing id",
			step:    func() Step { s := sampleReport().Steps[0]; s.ID = ""; return s }(),
			wantErr: true,
		},
		{
			name:    "missing tool",
			step:    func() Step { s := sampleReport().Steps[0]; s.Tool = ""; return s }(),
			wantErr: true,
		},
		{
			name:    "missing started_at",
			step:    func() Step { s := sampleReport().Steps[0]; s.StartedAt = ""; return s }(),
			wantErr: true,
		},
		{
			name:    "missing ended_at",
			step:    func() Step { s := sampleReport().Steps[0]; s.EndedAt = ""; return s }(),
			wantErr: true,
		},
		{
			name:    "bad status",
			step:    func() Step { s := sampleReport().Steps[0]; s.Status = "pending"; return s }(),
			wantErr: true,
		},
		{
			name:    "nil input",
			step:    func() Step { s := sampleReport().Steps[0]; s.Input = nil; return s }(),
			wantErr: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := tc.step
			err := s.Validate()
			if tc.wantErr && err == nil {
				t.Fatalf("expected error")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

// TestStepValidateNil covers the nil-receiver branch
// for steps.
func TestStepValidateNil(t *testing.T) {
	var s *Step
	err := s.Validate()
	if err == nil {
		t.Fatalf("expected error for nil step")
	}
}

// TestDurationSeconds covers the duration computation.
// The function parses the timestamps and returns the
// elapsed seconds.
func TestDurationSeconds(t *testing.T) {
	r := sampleReport()
	// 5 minutes 12.4 seconds = 312.4 seconds.
	got := r.DurationSeconds()
	if got < 312 || got > 313 {
		t.Fatalf("expected ~312.4, got %v", got)
	}
}

// TestDurationSecondsBadTimestamps covers the
// bad-timestamp branch. The function returns 0.
func TestDurationSecondsBadTimestamps(t *testing.T) {
	r := sampleReport()
	r.StartedAt = "not a timestamp"
	if got := r.DurationSeconds(); got != 0 {
		t.Fatalf("expected 0, got %v", got)
	}
}

// TestReportErrorMessage covers the error format.
func TestReportErrorMessage(t *testing.T) {
	err := (&ReportError{Op: "validate", Reason: "boom"}).Error()
	if !strings.HasPrefix(err, "report:") {
		t.Fatalf("unexpected prefix: %q", err)
	}
	if !strings.HasSuffix(err, ".") {
		t.Fatalf("missing trailing period: %q", err)
	}
	if !strings.Contains(err, "validate") {
		t.Fatalf("missing op: %q", err)
	}
}
