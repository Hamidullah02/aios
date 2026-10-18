// Package state provides the task state machine.
//
// Per the doc set:
//
//   - REQ-CORE-STATE-001: 8 states (Pending, Planning, AwaitingApproval,
//     Running, Paused, Completed, Failed, Cancelled).
//   - REQ-CORE-STATE-002: every transition is allowed only by the
//     transition table. A transition that is not in the table is
//     rejected with an error that names the from-state, the to-state,
//     and the rejected reason.
//   - REQ-CORE-STATE-003: terminal states (Completed, Failed, Cancelled)
//     do not transition out. Resume() returns an error if called on a
//     terminal state.
//   - REQ-CORE-STATE-004: pause is allowed only from Running.
//     Resume is allowed from Paused and AwaitingApproval.
package state

import "fmt"

// State is a task state.
type State string

const (
	StatePending          State = "pending"
	StatePlanning         State = "planning"
	StateAwaitingApproval State = "awaiting_approval"
	StateRunning          State = "running"
	StatePaused           State = "paused"
	StateCompleted        State = "completed"
	StateFailed           State = "failed"
	StateCancelled        State = "cancelled"
)

// AllStates is the canonical list of all 8 states.
var AllStates = []State{
	StatePending,
	StatePlanning,
	StateAwaitingApproval,
	StateRunning,
	StatePaused,
	StateCompleted,
	StateFailed,
	StateCancelled,
}

// IsTerminal returns true if s is a terminal state.
func (s State) IsTerminal() bool {
	switch s {
	case StateCompleted, StateFailed, StateCancelled:
		return true
	}
	return false
}

// IsValid returns true if s is one of the 8 known states.
func (s State) IsValid() bool {
	for _, k := range AllStates {
		if k == s {
			return true
		}
	}
	return false
}

// transition holds the canonical set of allowed (from -> to) edges.
// The from-state is the key; the slice lists every legal destination.
//
// Notes on the table:
//
//   - Cancelled is reachable from any non-terminal state.
//   - Failed is reachable from any non-terminal state (a tool error
//     during Running, an invalid plan during Planning, etc.).
//   - Paused is reachable only from Running.
//   - Resume (Paused -> Running) is the only path back from Paused.
//   - AwaitingApproval can resume to Running or fail to Failed.
//   - Pending -> Planning is the only entry to planning.
//   - Planning -> Running, Planning -> AwaitingApproval, Planning ->
//     Failed are the planning exits.
var transitionTable = map[State][]State{
	StatePending: {
		StatePlanning,
		StateFailed,
		StateCancelled,
	},
	StatePlanning: {
		StateAwaitingApproval,
		StateRunning,
		StateFailed,
		StateCancelled,
	},
	StateAwaitingApproval: {
		StateRunning,
		StateFailed,
		StateCancelled,
	},
	StateRunning: {
		StatePaused,
		StateCompleted,
		StateFailed,
		StateCancelled,
	},
	StatePaused: {
		StateRunning,
		StateFailed,
		StateCancelled,
	},
	// Terminal states have no outgoing edges.
	StateCompleted: nil,
	StateFailed:    nil,
	StateCancelled: nil,
}

// TransitionError is the error type for a rejected transition.
type TransitionError struct {
	From   State
	To     State
	Reason string
}

func (e *TransitionError) Error() string {
	return fmt.Sprintf("state: cannot transition from %q to %q: %s", e.From, e.To, e.Reason)
}

// CanTransition returns true if the (from -> to) edge is in the
// transition table.
func CanTransition(from, to State) bool {
	if !from.IsValid() || !to.IsValid() {
		return false
	}
	for _, t := range transitionTable[from] {
		if t == to {
			return true
		}
	}
	return false
}

// Machine is a per-task state holder.
//
// A Machine is not safe for concurrent use. The audit writer (or any
// higher-level owner) is responsible for serializing access to a task's
// Machine.
type Machine struct {
	TaskID string
	State  State
}

// NewMachine returns a fresh Machine in the Pending state.
func NewMachine(taskID string) *Machine {
	return &Machine{TaskID: taskID, State: StatePending}
}

// Transition applies a (from -> to) edge. If the edge is not in the
// transition table, the Machine's state is unchanged and a
// TransitionError is returned.
func (m *Machine) Transition(to State) error {
	if m == nil {
		return fmt.Errorf("state: nil machine")
	}
	if !m.State.IsValid() {
		return &TransitionError{From: m.State, To: to, Reason: "current state is not a known state"}
	}
	if !to.IsValid() {
		return &TransitionError{From: m.State, To: to, Reason: "target state is not a known state"}
	}
	if m.State.IsTerminal() {
		return &TransitionError{From: m.State, To: to, Reason: "current state is terminal"}
	}
	if !CanTransition(m.State, to) {
		return &TransitionError{
			From:   m.State,
			To:     to,
			Reason: "transition is not in the transition table",
		}
	}
	m.State = to
	return nil
}

// Pause is a convenience for Running -> Paused.
func (m *Machine) Pause() error { return m.Transition(StatePaused) }

// Resume is a convenience for Paused -> Running and AwaitingApproval
// -> Running. It is an error to call Resume on a terminal state
// (REQ-CORE-STATE-003).
func (m *Machine) Resume() error {
	switch m.State {
	case StatePaused, StateAwaitingApproval:
		return m.Transition(StateRunning)
	}
	if m.State.IsTerminal() {
		return &TransitionError{From: m.State, To: StateRunning, Reason: "cannot resume a terminal task"}
	}
	return &TransitionError{From: m.State, To: StateRunning, Reason: "resume is only allowed from Paused or AwaitingApproval"}
}

// Cancel is a convenience for any non-terminal state -> Cancelled.
func (m *Machine) Cancel() error { return m.Transition(StateCancelled) }

// Fail is a convenience for any non-terminal state -> Failed.
func (m *Machine) Fail() error { return m.Transition(StateFailed) }

// Complete is a convenience for Running -> Completed.
func (m *Machine) Complete() error { return m.Transition(StateCompleted) }
