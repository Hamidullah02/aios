package state

import (
	"strings"
	"testing"
)

func TestTransitionError_Error(t *testing.T) {
	e := &TransitionError{From: StatePending, To: StateCancelled, Reason: "not allowed"}
	got := e.Error()
	if !strings.Contains(got, "pending") || !strings.Contains(got, "cancelled") || !strings.Contains(got, "not allowed") {
		t.Fatalf("got %q", got)
	}
}

func TestCanTransition_InvalidState(t *testing.T) {
	if CanTransition(State("nope"), StatePending) {
		t.Fatal("invalid 'from' should not transition")
	}
	if CanTransition(StatePending, State("nope")) {
		t.Fatal("invalid 'to' should not transition")
	}
}

func TestTransition_NilMachine(t *testing.T) {
	var m *Machine
	if err := m.Transition(StateRunning); err == nil {
		t.Fatal("expected error for nil machine")
	}
}

func TestFail_FromPending(t *testing.T) {
	// Pending -> Failed is a valid edge.
	m := NewMachine("t1")
	if err := m.Fail(); err != nil {
		t.Fatal(err)
	}
	if m.State != StateFailed {
		t.Fatalf("State = %v", m.State)
	}
}

func TestFail_FromTerminal(t *testing.T) {
	// Completed is terminal; Fail should fail.
	m := &Machine{TaskID: "t1", State: StateCompleted}
	if err := m.Fail(); err == nil {
		t.Fatal("expected error transitioning from terminal")
	}
}

func TestComplete_FromPending(t *testing.T) {
	// Pending -> Completed is not a valid edge; only
	// Running -> Completed is.
	m := NewMachine("t1")
	if err := m.Complete(); err == nil {
		t.Fatal("expected error from Pending to Completed")
	}
}

func TestComplete_FromRunning(t *testing.T) {
	m := &Machine{TaskID: "t1", State: StateRunning}
	if err := m.Complete(); err != nil {
		t.Fatal(err)
	}
	if m.State != StateCompleted {
		t.Fatalf("State = %v", m.State)
	}
}

func TestTransition_CurrentStateInvalid(t *testing.T) {
	// Force a machine into an unknown state.
	m := &Machine{TaskID: "t1", State: State("nope")}
	err := m.Transition(StateRunning)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "current state is not a known state") {
		t.Fatalf("err = %v", err)
	}
}

func TestTransition_TargetStateInvalid(t *testing.T) {
	m := NewMachine("t1")
	err := m.Transition(State("nope"))
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "target state is not a known state") {
		t.Fatalf("err = %v", err)
	}
}
