package clock

import (
	"testing"
	"time"
)

func TestSystemDecision_ReadsWallClock(t *testing.T) {
	before := time.Now()
	got := SystemDecision{}.Now()
	after := time.Now()
	if got.Before(before) || got.After(after) {
		t.Fatalf("SystemDecision.Now = %v, not in [%v, %v]", got, before, after)
	}
}

func TestSystemDeadline_ReadsWallClock(t *testing.T) {
	before := time.Now()
	got := SystemDeadline{}.Now()
	after := time.Now()
	if got.Before(before) || got.After(after) {
		t.Fatalf("SystemDeadline.Now = %v, not in [%v, %v]", got, before, after)
	}
}

func TestFixedDecision_ZeroIsEpoch(t *testing.T) {
	// The doc-set contract: zero FixedDecision returns
	// the Unix epoch. This is the "no clock injected"
	// default for deterministic tests.
	if got := (FixedDecision{}).Now(); !got.Equal(time.Unix(0, 0)) {
		t.Fatalf("zero FixedDecision = %v, want epoch", got)
	}
}

func TestFixedDecision_NonZero(t *testing.T) {
	fixed := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	if got := (FixedDecision{T: fixed}).Now(); !got.Equal(fixed) {
		t.Fatalf("FixedDecision = %v, want %v", got, fixed)
	}
}

func TestFixedDeadline_ZeroIsEpoch(t *testing.T) {
	if got := (FixedDeadline{}).Now(); !got.Equal(time.Unix(0, 0)) {
		t.Fatalf("zero FixedDeadline = %v, want epoch", got)
	}
}

func TestFixedDeadline_NonZero(t *testing.T) {
	fixed := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	if got := (FixedDeadline{T: fixed}).Now(); !got.Equal(fixed) {
		t.Fatalf("FixedDeadline = %v, want %v", got, fixed)
	}
}

// TestDecisionInterface makes sure the four impls
// satisfy the interface. Compile-time.
var (
	_ Decision = SystemDecision{}
	_ Decision = FixedDecision{}
	_ Deadline = SystemDeadline{}
	_ Deadline = FixedDeadline{}
)
