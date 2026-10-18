// Package retry is the retry-policy layer.
//
// The package provides four components: ErrorClassifier
// (decides the error class), RetryStrategy (computes the
// backoff), RetryBudget (caps retries), and RetryPolicy
// (puts the three together). The package has no third-
// party dependencies and uses only the Go standard
// library.
package retry

import (
	"errors"
	"fmt"
)

// ErrorClass is the category of a tool error
// (REQ-RTR-CLS-002).
type ErrorClass string

const (
	// ClassTransient: temporary failure. Retry with backoff.
	ClassTransient ErrorClass = "transient"
	// ClassPermanent: non-recoverable failure. Stop.
	ClassPermanent ErrorClass = "permanent"
	// ClassFixable: the model can fix the call. Retry with a fix prompt.
	ClassFixable ErrorClass = "fixable"
)

// StrategyMode is the backoff mode (REQ-RTR-STR-002).
type StrategyMode string

const (
	StrategyNone        StrategyMode = "none"
	StrategyImmediate   StrategyMode = "immediate"
	StrategyLinear      StrategyMode = "linear"
	StrategyExponential StrategyMode = "exponential"
)

// Action is the retry decision's action.
type Action string

const (
	ActionRetry        Action = "retry"
	ActionRetryWithFix Action = "retry_with_fix"
	ActionStop         Action = "stop"
)

// RetryDecision is the policy's output (REQ-RTR-DAT-001).
type RetryDecision struct {
	Action    Action `json:"action"`
	BackoffMs int    `json:"backoff_ms,omitempty"`
	FixPrompt string `json:"fix_prompt,omitempty"`
	Reason    string `json:"reason"`
}

// MaxBackoffMs is the strategy cap (REQ-RTR-STR-003).
const MaxBackoffMs = 30_000

// DefaultBaseDelayMs is the strategy's default base delay.
const DefaultBaseDelayMs = 1_000

// DefaultMaxRetries is the budget's default (REQ-RTR-DAT-002).
const DefaultMaxRetries = 3

// Sentinel errors. Per the aios-retry SRS §9.
var (
	// ErrRetryBudgetExceeded is raised when the budget is
	// exhausted (REQ-RTR-ERR-001).
	ErrRetryBudgetExceeded = errors.New("retry: budget exceeded")
)

// RetryBudgetExceeded is the typed error for an exhausted budget.
type RetryBudgetExceeded struct {
	TaskID string
	Tool   string
	Used   int
	Limit  int
}

func (e *RetryBudgetExceeded) Error() string {
	return fmt.Sprintf("retry: budget exceeded (task=%s tool=%s used=%d limit=%d)", e.TaskID, e.Tool, e.Used, e.Limit)
}

func (e *RetryBudgetExceeded) Is(target error) bool { return target == ErrRetryBudgetExceeded }
