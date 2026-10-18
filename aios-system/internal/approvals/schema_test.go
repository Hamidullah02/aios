package approvals

import "testing"

func TestState_IsValid(t *testing.T) {
	for _, s := range []State{StatePending, StateGranted, StateDenied, StateExpired} {
		if !s.IsValid() {
			t.Fatalf("%q should be valid", s)
		}
	}
	if State("garbage").IsValid() {
		t.Fatal("garbage should not be valid")
	}
}

func TestState_IsTerminal(t *testing.T) {
	if StatePending.IsTerminal() {
		t.Fatal("pending is not terminal")
	}
	for _, s := range []State{StateGranted, StateDenied, StateExpired} {
		if !s.IsTerminal() {
			t.Fatalf("%q should be terminal", s)
		}
	}
}
