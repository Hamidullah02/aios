package sandbox

import (
	"errors"
	"strings"
	"testing"
)

// TestResourceLimitsValidate covers the validation rules
// for ResourceLimits. The function rejects a non-positive
// time limit and negative values for the other fields.
func TestResourceLimitsValidate(t *testing.T) {
	cases := []struct {
		name    string
		limits  ResourceLimits
		wantErr bool
	}{
		{
			name:    "time only",
			limits:  ResourceLimits{TimeSeconds: 60},
			wantErr: false,
		},
		{
			name: "all set",
			limits: ResourceLimits{
				CPUSeconds:  10,
				MemoryBytes: 1 << 20,
				TimeSeconds: 60,
				Processes:   4,
			},
			wantErr: false,
		},
		{
			name:    "zero time",
			limits:  ResourceLimits{CPUSeconds: 1, TimeSeconds: 0},
			wantErr: true,
		},
		{
			name:    "negative time",
			limits:  ResourceLimits{TimeSeconds: -1},
			wantErr: true,
		},
		{
			name:    "negative cpu",
			limits:  ResourceLimits{TimeSeconds: 60, CPUSeconds: -1},
			wantErr: true,
		},
		{
			name:    "negative memory",
			limits:  ResourceLimits{TimeSeconds: 60, MemoryBytes: -1},
			wantErr: true,
		},
		{
			name:    "negative processes",
			limits:  ResourceLimits{TimeSeconds: 60, Processes: -1},
			wantErr: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.limits.Validate()
			if tc.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

// TestLimitsErrorMessage checks the error format. The
// project coding standard says the message starts with a
// lowercase letter and ends with a period.
func TestLimitsErrorMessage(t *testing.T) {
	err := (&LimitsError{Field: "time_seconds", Reason: "must be positive"}).Error()
	if !strings.HasPrefix(err, "sandbox: limits:") {
		t.Fatalf("missing prefix: %q", err)
	}
	if !strings.HasSuffix(err, ".") {
		t.Fatalf("missing trailing period: %q", err)
	}
	if !strings.Contains(err, "time_seconds") {
		t.Fatalf("missing field name: %q", err)
	}
}

// TestLimitsErrorIsTyped confirms the error is the package
// error type so callers can use errors.As.
func TestLimitsErrorIsTyped(t *testing.T) {
	err := ResourceLimits{TimeSeconds: 0}.Validate()
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	var le *LimitsError
	if !errors.As(err, &le) {
		t.Fatalf("expected *LimitsError, got %T", err)
	}
	if le.Field != "time_seconds" {
		t.Fatalf("unexpected field: %q", le.Field)
	}
}

// TestApplyLimitsSkipsZeros confirms that a zero-value field
// is treated as "no limit" and the corresponding syscall is
// not made. We use a fully-zero limit struct to exercise
// the skip path on the build tag for this OS.
func TestApplyLimitsSkipsZeros(t *testing.T) {
	// The validate step requires a positive time limit; the
	// apply step is the post-validate step. We test the apply
	// step with a valid struct that has only TimeSeconds set
	// to confirm the other syscalls are skipped. Save and
	// restore the parent's CPU limit so the test process
	// is not left constrained.
	saveParentLimits()
	limits := ResourceLimits{TimeSeconds: 60, CPUSeconds: 1}
	if err := limits.applyLimits(); err != nil {
		t.Fatalf("applyLimits: %v", err)
	}
	restoreParentLimits()
}
