package verify

import (
	"errors"
	"fmt"
)

// ErrVerification is the sentinel error the
// package returns when the self-check tool's
// model call or response parse fails
// (REQ-VRF-ERR-002).
var ErrVerification = errors.New("verify: verification failed")

// ErrTestRunner is the sentinel error the
// package returns when the test runner fails
// (REQ-VRF-ERR-001).
var ErrTestRunner = errors.New("verify: test runner failed")

// ErrResult is the sentinel error the
// package returns when a result schema
// fails validation.
var ErrResult = errors.New("verify: invalid result")

// VerificationError is the typed error the
// SelfCheck tool raises when the model call
// or response parse fails. The Op is the
// failing operation; the Err is the
// underlying error.
type VerificationError struct {
	Op     string
	Reason string
	Err    error
}

func (e *VerificationError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("verify: %s: %s: %s.", e.Op, e.Reason, e.Err.Error())
	}
	return fmt.Sprintf("verify: %s: %s.", e.Op, e.Reason)
}

// Unwrap returns the underlying error.
func (e *VerificationError) Unwrap() error { return e.Err }

// Is reports whether target is the
// ErrVerification sentinel.
func (e *VerificationError) Is(target error) bool {
	return target == ErrVerification
}

// TestRunnerError is the typed error the
// RunTests tool raises when the test runner
// fails. The Op is the failing operation
// (e.g. "spawn", "parse"); the Command is
// the test command; the Err is the
// underlying error.
type TestRunnerError struct {
	Op      string
	Command string
	Reason  string
	Err     error
}

func (e *TestRunnerError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("verify: %s: %s: %s: %s.", e.Op, e.Command, e.Reason, e.Err.Error())
	}
	return fmt.Sprintf("verify: %s: %s: %s.", e.Op, e.Command, e.Reason)
}

// Unwrap returns the underlying error.
func (e *TestRunnerError) Unwrap() error { return e.Err }

// Is reports whether target is the
// ErrTestRunner sentinel.
func (e *TestRunnerError) Is(target error) bool {
	return target == ErrTestRunner
}

// ResultError is the typed error the schema
// validation raises. The Field is the
// offending field; the Reason is the
// validator's reason.
type ResultError struct {
	Field  string
	Reason string
}

func (e *ResultError) Error() string {
	return fmt.Sprintf("verify: result: %s: %s.", e.Field, e.Reason)
}

// Is reports whether target is the
// ErrResult sentinel.
func (e *ResultError) Is(target error) bool {
	return target == ErrResult
}
