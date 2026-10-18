// Package runtime is the orchestrator for aios.
//
// The package exports the components that drive a TaskPlan from
// start to finish:
//
//   - Orchestrator: the for-loop that walks the plan's tasks
//     and dispatches each to a Worker, then to a Verifier.
//   - Worker: the interface an agent implements.
//   - BudgetMeter: tracks cost, time, and rate caps; gates
//     whether the orchestrator can proceed.
//   - NoProgressDetector: trips when the worker's output
//     repeats.
//   - ResumeHandler: saves / loads checkpoints (JSON files in
//     var/checkpoints/).
//   - Tracer: writes DecisionTrace audit rows.
//
// The orchestrator is sequential (one task at a time, per
// REQ-RT-ORC-003). The semaphore in BudgetMeter is a no-op in
// the sequential path but is wired so a future parallel path
// can take advantage of it without re-plumbing.
package runtime

import (
	"errors"
	"fmt"
)

// Sentinel errors. The doc set names them in §9.
var (
	// ErrBudgetExhausted is returned when the BudgetMeter refuses
	// to let the orchestrator proceed (REQ-RT-ERR-001).
	ErrBudgetExhausted = errors.New("runtime: budget exhausted")

	// ErrNoProgress is returned when the NoProgressDetector
	// trips (REQ-RT-ERR-002).
	ErrNoProgress = errors.New("runtime: no progress detected")

	// ErrCanceled is returned when Cancel is called while a
	// task is in flight.
	ErrCanceled = errors.New("runtime: task canceled")
)

// BudgetExhausted is the typed error returned by BudgetMeter
// when a specific cap is hit. The Kind is the cap name
// (e.g. "max_cost", "max_tool_calls_per_minute").
type BudgetExhausted struct {
	Kind   string
	Limit  float64
	Used   float64
	Reason string
}

func (e *BudgetExhausted) Error() string {
	return fmt.Sprintf("runtime: budget exhausted: %s (used %v of %v) — %s",
		e.Kind, e.Used, e.Limit, e.Reason)
}
func (e *BudgetExhausted) Is(target error) bool { return target == ErrBudgetExhausted }
func (e *BudgetExhausted) Unwrap() error        { return ErrBudgetExhausted }

// NoProgressDetected is the typed error returned by
// NoProgressDetector. The Count is the number of repeated
// outputs seen in a row.
type NoProgressDetected struct {
	Count int
}

func (e *NoProgressDetected) Error() string {
	return fmt.Sprintf("runtime: no progress: %d identical outputs in a row", e.Count)
}
func (e *NoProgressDetected) Is(target error) bool { return target == ErrNoProgress }
func (e *NoProgressDetected) Unwrap() error        { return ErrNoProgress }

// WaitingForApproval is the "exception" the worker raises when
// a tool call needs human approval (REQ-RT-ERR-003,
// REQ-RT-IFC-002). The Payload and ApprovalID come from the
// policy gateway's `ask` action.
type WaitingForApproval struct {
	Payload    any // schemas.ApprovalPayload in the policy-aware path
	ApprovalID string
	Decision   string // "ask" (from the policy gateway)
}

func (e *WaitingForApproval) Error() string {
	return fmt.Sprintf("runtime: waiting for approval %s", e.ApprovalID)
}
