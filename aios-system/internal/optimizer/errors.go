package optimizer

import "errors"

// Sentinel errors (REQ-OPT-ERR-001 to 003).
var (
	// ErrBudgetExhausted is raised when the budget
	// cannot cover any class.
	ErrBudgetExhausted = errors.New("optimizer: budget exhausted")
	// ErrUnknownClass is raised when the override is
	// not a known class.
	ErrUnknownClass = errors.New("optimizer: unknown class")
	// ErrTableCorrupt is raised when the cost/quality
	// table file fails to parse.
	ErrTableCorrupt = errors.New("optimizer: table corrupt")
)
