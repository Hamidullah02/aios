package verify

import (
	"errors"
	"testing"
)

func TestVerificationError_Branches(t *testing.T) {
	e := &VerificationError{Op: "x", Reason: "y"}
	if got := e.Error(); got != "verify: x: y." {
		t.Errorf("no-err branch: %q", got)
	}
	e2 := &VerificationError{Op: "x", Reason: "y", Err: errors.New("z")}
	if got := e2.Error(); got != "verify: x: y: z." {
		t.Errorf("with-err branch: %q", got)
	}
}

func TestVerificationError_Unwrap(t *testing.T) {
	inner := errors.New("inner")
	e := &VerificationError{Op: "x", Reason: "y", Err: inner}
	if !errors.Is(e, inner) {
		t.Errorf("Unwrap mismatch")
	}
	if !errors.Is(e, ErrVerification) {
		t.Errorf("Is ErrVerification failed")
	}
	if errors.Is(e, ErrTestRunner) {
		t.Errorf("Is ErrTestRunner must be false")
	}
}

func TestTestRunnerError_Branches(t *testing.T) {
	e := &TestRunnerError{Op: "x", Command: "c", Reason: "y"}
	if got := e.Error(); got != "verify: x: c: y." {
		t.Errorf("no-err branch: %q", got)
	}
	e2 := &TestRunnerError{Op: "x", Command: "c", Reason: "y", Err: errors.New("z")}
	if got := e2.Error(); got != "verify: x: c: y: z." {
		t.Errorf("with-err branch: %q", got)
	}
}

func TestTestRunnerError_Unwrap(t *testing.T) {
	inner := errors.New("inner")
	e := &TestRunnerError{Op: "x", Command: "c", Reason: "y", Err: inner}
	if !errors.Is(e, inner) {
		t.Errorf("Unwrap mismatch")
	}
	if !errors.Is(e, ErrTestRunner) {
		t.Errorf("Is ErrTestRunner failed")
	}
	if errors.Is(e, ErrVerification) {
		t.Errorf("Is ErrVerification must be false")
	}
}

func TestResultError_Is(t *testing.T) {
	e := &ResultError{Field: "x", Reason: "y"}
	if !errors.Is(e, ErrResult) {
		t.Errorf("Is ErrResult failed")
	}
}

func TestBufferCloser(t *testing.T) {
	bc := bufferCloser{Buffer: nil}
	if err := bc.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}
