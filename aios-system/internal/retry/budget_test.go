package retry

import (
	"errors"
	"testing"
)

func TestBudget_DefaultLimit(t *testing.T) {
	b := NewBudget(0)
	if b.Max != DefaultMaxRetries {
		t.Fatalf("default = %d", b.Max)
	}
}

func TestBudget_RemainingAndAllow(t *testing.T) {
	b := NewBudget(3)
	if got := b.Remaining("t1", "edit", ClassTransient); got != 3 {
		t.Fatalf("remaining = %d", got)
	}
	if !b.Allow("t1", "edit", ClassTransient) {
		t.Fatal("first allow should be true")
	}
	if got := b.Remaining("t1", "edit", ClassTransient); got != 2 {
		t.Fatalf("remaining = %d", got)
	}
	if !b.Allow("t1", "edit", ClassTransient) {
		t.Fatal("second allow should be true")
	}
	if !b.Allow("t1", "edit", ClassTransient) {
		t.Fatal("third allow should be true")
	}
	if b.Allow("t1", "edit", ClassTransient) {
		t.Fatal("fourth allow should be false")
	}
}

func TestBudget_PerTaskIsolation(t *testing.T) {
	b := NewBudget(2)
	b.Allow("t1", "edit", ClassTransient)
	b.Allow("t1", "edit", ClassTransient)
	// t1 is exhausted, but t2 still has full budget.
	if b.Allow("t1", "edit", ClassTransient) {
		t.Fatal("t1 should be exhausted")
	}
	if !b.Allow("t2", "edit", ClassTransient) {
		t.Fatal("t2 should be allowed")
	}
}

func TestBudget_PerToolLimit(t *testing.T) {
	b := NewBudget(10)
	b.SetToolLimit("edit", 2)
	if got := b.Remaining("t1", "edit", ClassTransient); got != 2 {
		t.Fatalf("per-tool remaining = %d", got)
	}
	b.Allow("t1", "edit", ClassTransient)
	b.Allow("t1", "edit", ClassTransient)
	if b.Allow("t1", "edit", ClassTransient) {
		t.Fatal("per-tool cap should fire")
	}
	// Other tool still allowed.
	if !b.Allow("t1", "read", ClassTransient) {
		t.Fatal("other tool should be allowed")
	}
}

func TestBudget_PerClassLimit(t *testing.T) {
	b := NewBudget(10)
	b.SetClassLimit(ClassFixable, 1)
	if got := b.Remaining("t1", "edit", ClassFixable); got != 1 {
		t.Fatalf("per-class remaining = %d", got)
	}
	b.Allow("t1", "edit", ClassFixable)
	if b.Allow("t1", "edit", ClassFixable) {
		t.Fatal("per-class cap should fire")
	}
	if !b.Allow("t1", "edit", ClassTransient) {
		t.Fatal("other class should be allowed")
	}
}

func TestBudget_MinOfLimits(t *testing.T) {
	b := NewBudget(10)
	b.SetToolLimit("edit", 5)
	b.SetClassLimit(ClassTransient, 3)
	if got := b.Remaining("t1", "edit", ClassTransient); got != 3 {
		t.Fatalf("min = %d, want 3 (class limit is the minimum)", got)
	}
}

func TestBudget_SetZeroRemovesOverride(t *testing.T) {
	b := NewBudget(10)
	b.SetToolLimit("edit", 5)
	if _, ok := b.PerTool["edit"]; !ok {
		t.Fatal("expected override")
	}
	b.SetToolLimit("edit", 0)
	if _, ok := b.PerTool["edit"]; ok {
		t.Fatal("zero should remove the override")
	}
}

func TestBudget_SetClassLimitZeroRemoves(t *testing.T) {
	b := NewBudget(10)
	b.SetClassLimit(ClassFixable, 1)
	if _, ok := b.PerClass[ClassFixable]; !ok {
		t.Fatal("expected override")
	}
	b.SetClassLimit(ClassFixable, 0)
	if _, ok := b.PerClass[ClassFixable]; ok {
		t.Fatal("zero should remove the override")
	}
}

func TestBudget_Used_ScopeMax(t *testing.T) {
	// A single Allow charges all three scopes. Used should
	// return that count.
	b := NewBudget(10)
	b.Allow("t1", "edit", ClassTransient)
	if got := b.Used("t1", "edit", ClassTransient); got != 1 {
		t.Fatalf("used = %d", got)
	}
}

func TestBudget_MustAllow_ReportsToolLimit(t *testing.T) {
	b := NewBudget(10)
	b.SetToolLimit("edit", 1)
	_ = b.MustAllow("t1", "edit", ClassTransient)
	err := b.MustAllow("t1", "edit", ClassTransient)
	if !errors.Is(err, ErrRetryBudgetExceeded) {
		t.Fatal("expected budget exceeded")
	}
	be, ok := err.(*RetryBudgetExceeded)
	if !ok {
		t.Fatal("expected typed error")
	}
	if be.Tool != "edit" || be.Limit != 1 {
		t.Fatalf("err = %+v", be)
	}
}

func TestBudget_MustAllow_ReportsClassLimit(t *testing.T) {
	b := NewBudget(10)
	b.SetClassLimit(ClassFixable, 1)
	_ = b.MustAllow("t1", "edit", ClassFixable)
	err := b.MustAllow("t1", "edit", ClassFixable)
	be, ok := err.(*RetryBudgetExceeded)
	if !ok || be.Limit != 1 {
		t.Fatalf("err = %+v", err)
	}
}

func TestBudget_Used_SkewedScopes(t *testing.T) {
	// Call with tool="read" and class=ClassFixable to skew
	// the per-scope counts: global and class bump, but the
	// "edit" tool scope stays at 0. Then ask for "edit" and
	// verify Used still returns the max.
	b := NewBudget(10)
	b.Allow("t1", "read", ClassFixable)
	if got := b.Used("t1", "edit", ClassFixable); got != 1 {
		t.Fatalf("used = %d, want 1 (max across scopes)", got)
	}
}

func TestBudget_Remaining_ClampsAtZero(t *testing.T) {
	// Over-spend the budget; remaining must clamp to 0.
	b := NewBudget(1)
	for i := 0; i < 5; i++ {
		b.Allow("t1", "edit", ClassTransient)
	}
	if got := b.Remaining("t1", "edit", ClassTransient); got != 0 {
		t.Fatalf("remaining = %d", got)
	}
}

func TestBudget_MustAllow(t *testing.T) {
	b := NewBudget(1)
	if err := b.MustAllow("t1", "edit", ClassTransient); err != nil {
		t.Fatalf("first must-allow: %v", err)
	}
	err := b.MustAllow("t1", "edit", ClassTransient)
	if !errors.Is(err, ErrRetryBudgetExceeded) {
		t.Fatalf("err = %v, want ErrRetryBudgetExceeded", err)
	}
}

func TestBudget_Used(t *testing.T) {
	b := NewBudget(5)
	b.Allow("t1", "edit", ClassTransient)
	b.Allow("t1", "edit", ClassTransient)
	if got := b.Used("t1", "edit", ClassTransient); got != 2 {
		t.Fatalf("used = %d", got)
	}
}
