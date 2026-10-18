package fanout

import (
	"errors"
	"testing"
)

// TestMergeStrategyIsValid covers the strategy
// enum.
func TestMergeStrategyIsValid(t *testing.T) {
	for _, s := range []MergeStrategy{MergeConcat, MergeSummary, MergeVote} {
		if !s.IsValid() {
			t.Fatalf("expected %q to be valid", s)
		}
	}
	for _, bad := range []MergeStrategy{"", "merge", "Concat", "VOTE"} {
		if bad.IsValid() {
			t.Fatalf("expected %q to be invalid", bad)
		}
	}
}

// TestFanOutStatusIsValid covers the status enum.
func TestFanOutStatusIsValid(t *testing.T) {
	for _, s := range []FanOutStatus{StatusSucceeded, StatusFailed, StatusCanceled} {
		if !s.IsValid() {
			t.Fatalf("expected %q to be valid", s)
		}
	}
	for _, bad := range []FanOutStatus{"", "ok", "Succeeded", "DONE"} {
		if bad.IsValid() {
			t.Fatalf("expected %q to be invalid", bad)
		}
	}
}

// TestFanOutTaskValidate covers the required-
// field guards (REQ-FAN-DAT-001).
func TestFanOutTaskValidate(t *testing.T) {
	cases := []struct {
		name  string
		task  FanOutTask
		valid bool
	}{
		{"all good", FanOutTask{ID: "t1", Prompt: "do x", AgentType: "researcher"}, true},
		{"empty id", FanOutTask{ID: "", Prompt: "do x", AgentType: "researcher"}, false},
		{"empty prompt", FanOutTask{ID: "t1", Prompt: "", AgentType: "researcher"}, false},
		{"empty agent_type", FanOutTask{ID: "t1", Prompt: "do x", AgentType: ""}, false},
		{"with isolate", FanOutTask{ID: "t1", Prompt: "do x", AgentType: "researcher", Isolate: true, Branch: "feat/x"}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.task.Validate()
			if c.valid && err != nil {
				t.Fatalf("expected valid, got %v", err)
			}
			if !c.valid && err == nil {
				t.Fatal("expected error")
			}
		})
	}
	// Nil receiver.
	var nilTask *FanOutTask
	if err := nilTask.Validate(); err == nil {
		t.Fatal("expected error for nil task")
	}
}

// TestFanOutResultValidate covers the result
// validation (REQ-FAN-DAT-002).
func TestFanOutResultValidate(t *testing.T) {
	cases := []struct {
		name   string
		result FanOutResult
		valid  bool
	}{
		{"all good", FanOutResult{ID: "t1", Status: StatusSucceeded, DurationMs: 10, CostUSD: 0.5}, true},
		{"empty id", FanOutResult{ID: "", Status: StatusSucceeded}, false},
		{"bad status", FanOutResult{ID: "t1", Status: "ok"}, false},
		{"negative duration", FanOutResult{ID: "t1", Status: StatusSucceeded, DurationMs: -1}, false},
		{"negative cost", FanOutResult{ID: "t1", Status: StatusSucceeded, CostUSD: -1.0}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.result.Validate()
			if c.valid && err != nil {
				t.Fatalf("expected valid, got %v", err)
			}
			if !c.valid && err == nil {
				t.Fatal("expected error")
			}
		})
	}
	// Nil receiver.
	var nilResult *FanOutResult
	if err := nilResult.Validate(); err == nil {
		t.Fatal("expected error for nil result")
	}
}

// TestFanOutErrorIsCovers the typed error
// Is method.
func TestFanOutErrorIsCovers(t *testing.T) {
	e1 := &FanOutError{Op: "x", TaskID: "t1", Reason: "y"}
	e2 := &FanOutError{Op: "a", TaskID: "t2", Reason: "b"}
	if !errors.Is(e1, e2) {
		t.Fatal("FanOutError should match itself")
	}
	var plain error = errors.New("plain")
	if errors.Is(e1, plain) {
		t.Fatal("FanOutError should not match a plain error")
	}
}

// TestWorktreeErrorIsCovers the typed error
// Is method.
func TestWorktreeErrorIsCovers(t *testing.T) {
	e1 := &WorktreeError{Op: "x", Path: "p", Reason: "y"}
	e2 := &WorktreeError{Op: "a", Path: "q", Reason: "b"}
	if !errors.Is(e1, e2) {
		t.Fatal("WorktreeError should match itself")
	}
	if !errors.Is(e1, ErrWorktree) {
		t.Fatal("WorktreeError should match ErrWorktree")
	}
}

// TestBudgetErrorIsCovers the typed error
// Is method.
func TestBudgetErrorIsCovers(t *testing.T) {
	e := &BudgetError{Kind: "max_cost", Limit: 1.0, Used: 2.0, Reason: "r"}
	if !errors.Is(e, ErrBudget) {
		t.Fatal("BudgetError should match ErrBudget")
	}
	if e.Error() == "" {
		t.Fatal("Error() should not be empty")
	}
}

// TestFanOutErrorWithAndWithoutCause covers the
// Error() branches.
func TestFanOutErrorWithAndWithoutCause(t *testing.T) {
	e := &FanOutError{Op: "run", TaskID: "t1", Index: 0, Reason: "boom"}
	if e.Error() == "" {
		t.Fatal("Error() should not be empty")
	}
	e2 := &FanOutError{Op: "run", TaskID: "t1", Index: 0, Reason: "boom", Err: errors.New("inner")}
	if e2.Error() == "" {
		t.Fatal("Error() should not be empty")
	}
	if !contains(e2.Error(), "inner") {
		t.Fatalf("Error() should contain inner: %q", e2.Error())
	}
}

// TestWorktreeErrorWithAndWithoutCause covers
// the Error() branches.
func TestWorktreeErrorWithAndWithoutCause(t *testing.T) {
	e := &WorktreeError{Op: "create", Path: "p", Reason: "boom"}
	if e.Error() == "" {
		t.Fatal("Error() should not be empty")
	}
	e2 := &WorktreeError{Op: "create", Path: "p", Reason: "boom", Err: errors.New("inner")}
	if e2.Error() == "" {
		t.Fatal("Error() should not be empty")
	}
	if !contains(e2.Error(), "inner") {
		t.Fatalf("Error() should contain inner: %q", e2.Error())
	}
}

// TestNewTimerAndElapsed covers the small timer
// helper.
func TestNewTimerAndElapsed(t *testing.T) {
	tmr := newTimer()
	if tmr == nil {
		t.Fatal("newTimer returned nil")
	}
	_ = tmr.elapsed()
	// Nil receiver.
	var nilTimer *timer
	if got := nilTimer.elapsed(); got != 0 {
		t.Fatalf("nil timer elapsed = %d, want 0", got)
	}
}
