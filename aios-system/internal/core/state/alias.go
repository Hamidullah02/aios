package state

// Doc-set names. The doc set (SRS §7 IFC-003) names the state types
// `TaskState` and `TaskStateMachine`. The implementation uses the
// package-qualified `State` / `Machine` because every type in this
// package is implicitly a task type and the package name carries
// the qualifier. These aliases give downstream modules a choice:
//
//   - use the short form:    state.State, state.Machine
//   - use the doc-set form:  state.TaskState, state.TaskStateMachine
//
// Aliasing in Go is purely textual. No runtime cost.

// TaskState is the doc-set name for State.
type TaskState = State

// TaskStateMachine is the doc-set name for Machine.
type TaskStateMachine = Machine
