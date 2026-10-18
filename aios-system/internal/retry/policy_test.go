package retry

import (
	"errors"
	"strings"
	"testing"
)

func newPolicy(mode StrategyMode, maxRetries int) *RetryPolicy {
	return NewPolicy(
		NewClassifier(),
		NewStrategy(mode),
		NewBudget(maxRetries),
	)
}

func TestPolicy_Transient(t *testing.T) {
	p := newPolicy(StrategyExponential, 3)
	dec := p.Decide("t1", "edit", errors.New("connection timeout"), 1)
	if dec.Action != ActionRetry {
		t.Fatalf("action = %s", dec.Action)
	}
	if dec.BackoffMs == 0 {
		t.Fatal("expected a non-zero backoff for exponential")
	}
}

func TestPolicy_Permanent(t *testing.T) {
	p := newPolicy(StrategyExponential, 3)
	dec := p.Decide("t1", "edit", errors.New("permission denied"), 0)
	if dec.Action != ActionStop {
		t.Fatalf("action = %s", dec.Action)
	}
	if dec.FixPrompt != "" {
		t.Fatal("permanent should not have a fix prompt")
	}
}

func TestPolicy_Fixable(t *testing.T) {
	p := newPolicy(StrategyImmediate, 3)
	dec := p.Decide("t1", "edit", errors.New("edit not unique: 3 matches"), 0)
	if dec.Action != ActionRetryWithFix {
		t.Fatalf("action = %s", dec.Action)
	}
	if dec.FixPrompt == "" {
		t.Fatal("expected a fix prompt")
	}
	if !strings.Contains(dec.FixPrompt, "unique") {
		t.Fatalf("fix prompt = %q", dec.FixPrompt)
	}
}

func TestPolicy_BudgetExhausted(t *testing.T) {
	p := newPolicy(StrategyImmediate, 2)
	// First two transient failures consume the budget.
	p.Decide("t1", "edit", errors.New("timeout"), 0)
	p.Decide("t1", "edit", errors.New("timeout"), 1)
	// Third call must stop.
	dec := p.Decide("t1", "edit", errors.New("timeout"), 2)
	if dec.Action != ActionStop {
		t.Fatalf("action = %s, want stop", dec.Action)
	}
}

func TestPolicy_NilStrategy(t *testing.T) {
	p := NewPolicy(NewClassifier(), nil, NewBudget(3))
	dec := p.Decide("t1", "edit", errors.New("timeout"), 1)
	if dec.Action != ActionRetry {
		t.Fatalf("action = %s", dec.Action)
	}
	// nil strategy → backoff 0
	if dec.BackoffMs != 0 {
		t.Fatalf("backoff = %d", dec.BackoffMs)
	}
}

func TestPolicy_NilClassifierAndBudget(t *testing.T) {
	p := NewPolicy(nil, nil, nil)
	dec := p.Decide("t1", "edit", errors.New("permission denied"), 0)
	if dec.Action != ActionStop {
		t.Fatalf("action = %s", dec.Action)
	}
}

func TestPolicy_FixPrompts_NotFound(t *testing.T) {
	p := newPolicy(StrategyImmediate, 3)
	dec := p.Decide("t1", "edit", errors.New("edit not found"), 0)
	if dec.Action != ActionRetryWithFix {
		t.Fatalf("action = %s", dec.Action)
	}
	if !strings.Contains(dec.FixPrompt, "Read the file") {
		t.Fatalf("fix prompt = %q", dec.FixPrompt)
	}
}

func TestPolicy_FixPrompts_Conflict(t *testing.T) {
	p := newPolicy(StrategyImmediate, 3)
	dec := p.Decide("t1", "edit", errors.New("merge conflict"), 0)
	if dec.Action != ActionRetryWithFix {
		t.Fatalf("action = %s", dec.Action)
	}
}

func TestPolicy_FixPrompts_Syntax(t *testing.T) {
	p := newPolicy(StrategyImmediate, 3)
	dec := p.Decide("t1", "edit", errors.New("syntax error in foo"), 0)
	if dec.Action != ActionRetryWithFix {
		t.Fatalf("action = %s", dec.Action)
	}
}

func TestPolicy_UnknownErrorStops(t *testing.T) {
	p := newPolicy(StrategyImmediate, 3)
	// "something went wrong" matches no rule → permanent → stop.
	dec := p.Decide("t1", "edit", errors.New("something went wrong"), 0)
	if dec.Action != ActionStop {
		t.Fatalf("action = %s, want stop", dec.Action)
	}
}

func TestPolicy_FixPrompts_Generic(t *testing.T) {
	// When a fixable error matches the catch-all fixable
	// bucket, the prompt is the generic "previous call
	// failed" text. This is reached when the classifier
	// is custom-configured with a fixable rule for a
	// non-standard error.
	c := NewClassifier()
	c.Add(ClassFixable, "weird thing")
	p := NewPolicy(c, NewStrategy(StrategyImmediate), NewBudget(3))
	dec := p.Decide("t1", "edit", errors.New("weird thing happened"), 0)
	if dec.Action != ActionRetryWithFix {
		t.Fatalf("action = %s", dec.Action)
	}
	if !strings.Contains(dec.FixPrompt, "previous call") {
		t.Fatalf("fix prompt = %q", dec.FixPrompt)
	}
}

func TestPolicy_ReasonIncludesError(t *testing.T) {
	p := newPolicy(StrategyImmediate, 3)
	dec := p.Decide("t1", "edit", errors.New("permission denied"), 0)
	if !strings.Contains(dec.Reason, "permission denied") {
		t.Fatalf("reason = %q", dec.Reason)
	}
}

func TestPolicy_NilError(t *testing.T) {
	// A nil error is classified as permanent → stop with
	// an empty reason segment.
	p := newPolicy(StrategyImmediate, 3)
	dec := p.Decide("t1", "edit", nil, 0)
	if dec.Action != ActionStop {
		t.Fatalf("action = %s", dec.Action)
	}
}
