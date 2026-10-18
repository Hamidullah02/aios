package fanout

import (
	"errors"
	"fmt"
)

// ErrFanOut is the sentinel error the package
// returns when a fan-out operation fails. The
// sentinel lets callers match with errors.Is
// regardless of the concrete value.
var ErrFanOut = errors.New("fanout: operation failed")

// ErrWorktree is the sentinel error the package
// returns when a worktree operation fails
// (REQ-FAN-ERR-002).
var ErrWorktree = errors.New("fanout: worktree operation failed")

// ErrBudget is the sentinel error the package
// returns when the scheduler's budget is
// exhausted.
var ErrBudget = errors.New("fanout: budget exhausted")

// FanOutError is the typed error the FanOutTool
// raises when a sub-agent fails (REQ-FAN-ERR-001).
// The error carries the failing task id, the
// underlying error, and the index of the task in
// the input list.
type FanOutError struct {
	TaskID string
	Index  int
	Op     string
	Reason string
	Err    error
}

func (e *FanOutError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("fanout: %s: task %q (index %d): %s: %s.", e.Op, e.TaskID, e.Index, e.Reason, e.Err.Error())
	}
	return fmt.Sprintf("fanout: %s: task %q (index %d): %s.", e.Op, e.TaskID, e.Index, e.Reason)
}

// Unwrap returns the underlying error. The method
// lets callers use errors.Is and errors.As on the
// wrapped error.
func (e *FanOutError) Unwrap() error { return e.Err }

// Is reports whether target is also a FanOutError.
// The method lets callers use errors.Is to detect
// a fan-out failure regardless of the concrete
// value.
func (e *FanOutError) Is(target error) bool {
	_, ok := target.(*FanOutError)
	return ok
}

// WorktreeError is the typed error the
// WorktreeIsolation raises when the worktree
// operation fails (REQ-FAN-ERR-002). The error
// carries the failing operation, the path, and
// the underlying error.
type WorktreeError struct {
	Op     string
	Path   string
	Reason string
	Err    error
}

func (e *WorktreeError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("fanout: worktree: %s: %s: %s: %s.", e.Op, e.Path, e.Reason, e.Err.Error())
	}
	return fmt.Sprintf("fanout: worktree: %s: %s: %s.", e.Op, e.Path, e.Reason)
}

// Unwrap returns the underlying error. The method
// lets callers use errors.Is and errors.As on the
// wrapped error.
func (e *WorktreeError) Unwrap() error { return e.Err }

// Is reports whether target is also a
// WorktreeError or the ErrWorktree sentinel. The
// method lets callers use errors.Is to detect a
// worktree failure regardless of the concrete
// value.
func (e *WorktreeError) Is(target error) bool {
	if target == ErrWorktree {
		return true
	}
	_, ok := target.(*WorktreeError)
	return ok
}

// BudgetError is the typed error the Scheduler
// raises when the budget is exhausted. The error
// carries the budget cap and the amount used.
type BudgetError struct {
	Kind   string
	Limit  float64
	Used   float64
	Reason string
}

func (e *BudgetError) Error() string {
	return fmt.Sprintf("fanout: budget exhausted: %s (used %v of %v) — %s", e.Kind, e.Used, e.Limit, e.Reason)
}

// Is reports whether target is also a budget
// error. The method lets callers use errors.Is
// to detect a budget failure regardless of the
// concrete value.
func (e *BudgetError) Is(target error) bool {
	return target == ErrBudget
}
