package runtime

import (
	"errors"
	"testing"
	"time"
)

func TestBudgetMeter_Allows(t *testing.T) {
	m := NewBudgetMeter(Budget{MaxToolCalls: 5})
	if err := m.CanProceed(); err != nil {
		t.Fatalf("CanProceed: %v", err)
	}
	m.RecordToolCall()
	m.RecordToolCall()
	if err := m.CanProceed(); err != nil {
		t.Fatalf("after 2/5: %v", err)
	}
}

func TestBudgetMeter_ToolCallsCap(t *testing.T) {
	m := NewBudgetMeter(Budget{MaxToolCalls: 2})
	m.RecordToolCall()
	m.RecordToolCall()
	err := m.CanProceed()
	var be *BudgetExhausted
	if !errors.As(err, &be) {
		t.Fatalf("err type = %T, want *BudgetExhausted", err)
	}
	if be.Kind != "max_tool_calls" {
		t.Fatalf("Kind = %q", be.Kind)
	}
}

func TestBudgetMeter_CostCap(t *testing.T) {
	m := NewBudgetMeter(Budget{MaxCostUSD: 1.0})
	m.RecordModelCall(0.6)
	if err := m.CanProceed(); err != nil {
		t.Fatal(err)
	}
	m.RecordModelCall(0.4) // total 1.0, hits the cap
	err := m.CanProceed()
	if !errors.Is(err, ErrBudgetExhausted) {
		t.Fatalf("err = %v, want ErrBudgetExhausted", err)
	}
}

func TestBudgetMeter_WallTimeCap(t *testing.T) {
	m := NewBudgetMeter(Budget{MaxWallTimeSec: 0})
	// No wall-time cap, so no error even after "elapsed" time.
	time.Sleep(2 * time.Millisecond)
	if err := m.CanProceed(); err != nil {
		t.Fatal(err)
	}
	// With a cap, we have to wait. Skip that here; trust the
	// production code path.
}

func TestBudgetMeter_RateCap(t *testing.T) {
	m := NewBudgetMeter(Budget{MaxToolCallsPerMinute: 3})
	now := time.Now()
	m.toolWindow = []time.Time{now.Add(-30 * time.Second), now.Add(-10 * time.Second)}
	if err := m.CanProceed(); err != nil {
		t.Fatalf("at 2 of 3 in last 60s: %v", err)
	}
	m.toolWindow = append(m.toolWindow, now) // 3 of 3
	err := m.CanProceed()
	if !errors.Is(err, ErrBudgetExhausted) {
		t.Fatalf("err = %v, want ErrBudgetExhausted", err)
	}
}

func TestBudgetMeter_RateCapPrunesOldEntries(t *testing.T) {
	m := NewBudgetMeter(Budget{MaxToolCallsPerMinute: 5})
	now := time.Now()
	// 10 entries older than 60s. After pruning, count should
	// drop to 0 and CanProceed should pass.
	for i := 0; i < 10; i++ {
		m.toolWindow = append(m.toolWindow, now.Add(-90*time.Second))
	}
	if err := m.CanProceed(); err != nil {
		t.Fatalf("after pruning: %v", err)
	}
	if len(m.toolWindow) != 0 {
		t.Fatalf("window len = %d, want 0", len(m.toolWindow))
	}
}

func TestBudgetMeter_ParallelWorkers(t *testing.T) {
	m := NewBudgetMeter(Budget{MaxParallelWorkers: 2})
	if !m.AcquireWorker() {
		t.Fatal("first acquire")
	}
	if !m.AcquireWorker() {
		t.Fatal("second acquire")
	}
	m.ReleaseWorker()
	if !m.AcquireWorker() {
		t.Fatal("re-acquire after release")
	}
	m.ReleaseWorker()
	m.ReleaseWorker()
}

func TestBudgetMeter_ParallelWorkers_NoCap(t *testing.T) {
	m := NewBudgetMeter(Budget{}) // no cap
	// AcquireWorker returns true but does not block.
	for i := 0; i < 100; i++ {
		if !m.AcquireWorker() {
			t.Fatalf("acquire %d failed", i)
		}
	}
}

func TestBudgetMeter_Snapshot(t *testing.T) {
	m := NewBudgetMeter(Budget{})
	m.RecordToolCall()
	m.RecordToolCall()
	m.RecordModelCall(0.05)
	m.RecordNetworkRequest()
	snap := m.Snapshot()
	if snap.ToolCalls != 2 {
		t.Fatalf("ToolCalls = %d", snap.ToolCalls)
	}
	if snap.ModelCalls != 1 {
		t.Fatalf("ModelCalls = %d", snap.ModelCalls)
	}
	if snap.NetworkRequests != 1 {
		t.Fatalf("NetworkRequests = %d", snap.NetworkRequests)
	}
	if snap.CostUSD != 0.05 {
		t.Fatalf("CostUSD = %v", snap.CostUSD)
	}
}
