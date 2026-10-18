package state

import "testing"

func TestAllStatesValid(t *testing.T) {
	if len(AllStates) != 8 {
		t.Fatalf("expected 8 states, got %d", len(AllStates))
	}
	for _, s := range AllStates {
		if !s.IsValid() {
			t.Fatalf("state %q not valid", s)
		}
	}
}

func TestTerminals(t *testing.T) {
	if !StateCompleted.IsTerminal() {
		t.Fatal("Completed not terminal")
	}
	if !StateFailed.IsTerminal() {
		t.Fatal("Failed not terminal")
	}
	if !StateCancelled.IsTerminal() {
		t.Fatal("Cancelled not terminal")
	}
	if StateRunning.IsTerminal() {
		t.Fatal("Running should not be terminal")
	}
}

func TestHappyPath(t *testing.T) {
	m := NewMachine("t1")
	if m.State != StatePending {
		t.Fatalf("initial state: %q", m.State)
	}
	if err := m.Transition(StatePlanning); err != nil {
		t.Fatal(err)
	}
	if err := m.Transition(StateRunning); err != nil {
		t.Fatal(err)
	}
	if err := m.Complete(); err != nil {
		t.Fatal(err)
	}
	if m.State != StateCompleted {
		t.Fatalf("final state: %q", m.State)
	}
}

func TestCannotLeaveTerminal(t *testing.T) {
	m := NewMachine("t1")
	if err := m.Transition(StateFailed); err != nil {
		t.Fatal(err)
	}
	if err := m.Transition(StateRunning); err == nil {
		t.Fatal("expected error: cannot leave terminal")
	}
}

func TestCancelFromAnyNonTerminal(t *testing.T) {
	// pathTo: sequence of transitions to reach a target state.
	cases := map[State][]State{
		StatePending:          {},
		StatePlanning:         {StatePlanning},
		StateAwaitingApproval: {StatePlanning, StateAwaitingApproval},
		StateRunning:          {StatePlanning, StateRunning},
		StatePaused:           {StatePlanning, StateRunning, StatePaused},
	}
	for target, path := range cases {
		m := NewMachine("t1")
		for _, step := range path {
			if err := m.Transition(step); err != nil {
				t.Fatalf("setup to %s: %v on step %s", target, err, step)
			}
		}
		if m.State != target {
			t.Fatalf("setup to %s: ended at %s", target, m.State)
		}
		if err := m.Cancel(); err != nil {
			t.Fatalf("cancel from %s: %v", target, err)
		}
	}
}

func TestResumeFromPaused(t *testing.T) {
	m := NewMachine("t1")
	_ = m.Transition(StatePlanning)
	_ = m.Transition(StateRunning)
	if err := m.Pause(); err != nil {
		t.Fatal(err)
	}
	if err := m.Resume(); err != nil {
		t.Fatal(err)
	}
	if m.State != StateRunning {
		t.Fatalf("after resume: %q", m.State)
	}
}

func TestResumeFromAwaitingApproval(t *testing.T) {
	m := NewMachine("t1")
	_ = m.Transition(StatePlanning)
	_ = m.Transition(StateAwaitingApproval)
	if err := m.Resume(); err != nil {
		t.Fatal(err)
	}
	if m.State != StateRunning {
		t.Fatalf("after resume: %q", m.State)
	}
}

func TestResumeFromTerminalFails(t *testing.T) {
	m := NewMachine("t1")
	_ = m.Transition(StateFailed)
	if err := m.Resume(); err == nil {
		t.Fatal("expected error: resume from terminal")
	}
}

func TestResumeFromPendingFails(t *testing.T) {
	m := NewMachine("t1")
	if err := m.Resume(); err == nil {
		t.Fatal("expected error: resume from pending")
	}
}

func TestInvalidTransition(t *testing.T) {
	m := NewMachine("t1")
	// Pending -> Running is not in the table.
	if err := m.Transition(StateRunning); err == nil {
		t.Fatal("expected error: Pending->Running not in table")
	}
}

func TestUnknownStateRejected(t *testing.T) {
	m := NewMachine("t1")
	if err := m.Transition(State("bogus")); err == nil {
		t.Fatal("expected error for bogus state")
	}
}
