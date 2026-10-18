package fanout

import (
	"context"
	"errors"
	"testing"

	"github.com/sadi/aios-system/internal/runtime"
)

// newTestScheduler constructs a Scheduler with
// a simple echo sub-agent. The helper is the
// shared test setup.
func newTestScheduler(t *testing.T, maxParallel int, cap BudgetCap) *Scheduler {
	t.Helper()
	tool, err := NewFanOutTool(FanOutConfig{
		Factory: SubAgentFactoryFunc(func(task FanOutTask) (runtime.Worker, error) {
			return &echoWorker{}, nil
		}),
		MaxParallel: maxParallel,
	})
	if err != nil {
		t.Fatalf("NewFanOutTool: %v", err)
	}
	sched, err := NewScheduler(SchedulerConfig{Tool: tool, Cap: cap})
	if err != nil {
		t.Fatalf("NewScheduler: %v", err)
	}
	return sched
}

// TestNewSchedulerRequiresTool covers the
// nil-tool guard.
func TestNewSchedulerRequiresTool(t *testing.T) {
	_, err := NewScheduler(SchedulerConfig{})
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestSchedulerEmpty covers the empty task
// list.
func TestSchedulerEmpty(t *testing.T) {
	sched := newTestScheduler(t, 4, BudgetCap{})
	merged, err := sched.Schedule(context.Background(), nil)
	if err != nil {
		t.Fatalf("Schedule: %v", err)
	}
	if len(merged.Results) != 0 {
		t.Fatalf("Results = %d, want 0", len(merged.Results))
	}
}

// TestSchedulerRunsAllTasks covers the basic
// path (REQ-FAN-SCH-001, -002).
func TestSchedulerRunsAllTasks(t *testing.T) {
	sched := newTestScheduler(t, 2, BudgetCap{})
	tasks := make([]FanOutTask, 5)
	for i := range tasks {
		tasks[i] = FanOutTask{ID: string(rune('a' + i)), Prompt: "p", AgentType: "x"}
	}
	merged, err := sched.Schedule(context.Background(), tasks)
	if err != nil {
		t.Fatalf("Schedule: %v", err)
	}
	if len(merged.Results) != 5 {
		t.Fatalf("Results = %d, want 5", len(merged.Results))
	}
	for _, r := range merged.Results {
		if r.Status != StatusSucceeded {
			t.Fatalf("task %s status = %q", r.ID, r.Status)
		}
	}
}

// TestSchedulerRespectsCostBudget covers
// REQ-FAN-SCH-003. The first batch exceeds the
// cost cap; the scheduler returns a partial
// result and a BudgetError.
func TestSchedulerRespectsCostBudget(t *testing.T) {
	sched := newTestScheduler(t, 1, BudgetCap{MaxCostUSD: 0.02})
	tasks := make([]FanOutTask, 5)
	for i := range tasks {
		tasks[i] = FanOutTask{ID: string(rune('a' + i)), Prompt: "p", AgentType: "x"}
	}
	merged, err := sched.Schedule(context.Background(), tasks)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrBudget) {
		t.Fatalf("err = %v, want ErrBudget", err)
	}
	// At least one task ran, but not all.
	if len(merged.Results) == 0 {
		t.Fatal("expected partial results")
	}
	if len(merged.Results) >= len(tasks) {
		t.Fatalf("Results = %d, want < %d (budget should have stopped early)", len(merged.Results), len(tasks))
	}
}

// TestSchedulerRespectsTaskBudget covers the
// MaxTasks cap.
func TestSchedulerRespectsTaskBudget(t *testing.T) {
	sched := newTestScheduler(t, 2, BudgetCap{MaxTasks: 3})
	tasks := make([]FanOutTask, 6)
	for i := range tasks {
		tasks[i] = FanOutTask{ID: string(rune('a' + i)), Prompt: "p", AgentType: "x"}
	}
	merged, err := sched.Schedule(context.Background(), tasks)
	if !errors.Is(err, ErrBudget) {
		t.Fatalf("err = %v, want ErrBudget", err)
	}
	if len(merged.Results) != 3 {
		t.Fatalf("Results = %d, want 3", len(merged.Results))
	}
}

// TestSchedulerTaskValidation covers the
// validate-up-front path.
func TestSchedulerTaskValidation(t *testing.T) {
	sched := newTestScheduler(t, 2, BudgetCap{})
	tasks := []FanOutTask{{ID: "", Prompt: "p", AgentType: "x"}}
	_, err := sched.Schedule(context.Background(), tasks)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestSchedulerGetToolAndCap covers the
// getters.
func TestSchedulerGetToolAndCap(t *testing.T) {
	sched := newTestScheduler(t, 4, BudgetCap{MaxCostUSD: 1.0})
	if sched.Tool() == nil {
		t.Fatal("Tool is nil")
	}
	if sched.Cap().MaxCostUSD != 1.0 {
		t.Fatalf("Cap.MaxCostUSD = %v", sched.Cap().MaxCostUSD)
	}
}

// TestSchedulerNilReceiver covers the nil
// receiver guard.
func TestSchedulerNilReceiver(t *testing.T) {
	var sched *Scheduler
	_, err := sched.Schedule(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error")
	}
}
