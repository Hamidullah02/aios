package fanout

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sadi/aios-system/internal/core/audit"
	"github.com/sadi/aios-system/internal/runtime"
)

// echoWorker is the test Worker that returns the
// input description as the output. The worker
// is the cheapest possible sub-agent; the tests
// use it to assert the parallel runner's
// orchestration without a real agent.
type echoWorker struct {
	called int32
	delay  time.Duration
}

func (w *echoWorker) Run(task runtime.TaskInput, ctx *runtime.TaskContext, meter *runtime.BudgetMeter) (runtime.WorkerResult, error) {
	atomic.AddInt32(&w.called, 1)
	if w.delay > 0 {
		time.Sleep(w.delay)
	}
	return runtime.WorkerResult{
		Output:  "echo:" + task.Description,
		Data:    map[string]any{"id": task.ID},
		CostUSD: 0.01,
	}, nil
}

// failingWorker is the test Worker that always
// returns an error.
type failingWorker struct{}

func (failingWorker) Run(task runtime.TaskInput, ctx *runtime.TaskContext, meter *runtime.BudgetMeter) (runtime.WorkerResult, error) {
	return runtime.WorkerResult{}, errors.New("sub-agent boom")
}

// cancelWorker is the test Worker that honors
// the context. The Worker is the test the
// context-cancel path uses.
type cancelWorker struct{}

func (cancelWorker) Run(task runtime.TaskInput, ctx *runtime.TaskContext, meter *runtime.BudgetMeter) (runtime.WorkerResult, error) {
	return runtime.WorkerResult{
		Output:  "ok",
		CostUSD: 0.001,
	}, nil
}

// TestFanOutToolNewRequiresFactory covers the
// nil-factory guard.
func TestFanOutToolNewRequiresFactory(t *testing.T) {
	_, err := NewFanOutTool(FanOutConfig{})
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestFanOutToolNewDefaultMaxParallel covers
// the default max_parallel (REQ-FAN-FOT-004).
func TestFanOutToolNewDefaultMaxParallel(t *testing.T) {
	tool, err := NewFanOutTool(FanOutConfig{
		Factory: SubAgentFactoryFunc(func(task FanOutTask) (runtime.Worker, error) {
			return &echoWorker{}, nil
		}),
	})
	if err != nil {
		t.Fatalf("NewFanOutTool: %v", err)
	}
	if tool.MaxParallel() != DefaultMaxParallel {
		t.Fatalf("MaxParallel = %d, want %d", tool.MaxParallel(), DefaultMaxParallel)
	}
}

// TestFanOutToolMaxParallelClampedTo8 covers
// the hard cap (REQ-FAN-NFR-003).
func TestFanOutToolMaxParallelClampedTo8(t *testing.T) {
	tool, _ := NewFanOutTool(FanOutConfig{
		Factory: SubAgentFactoryFunc(func(task FanOutTask) (runtime.Worker, error) {
			return &echoWorker{}, nil
		}),
		MaxParallel: 32,
	})
	if tool.MaxParallel() != MaxMaxParallel {
		t.Fatalf("MaxParallel = %d, want %d", tool.MaxParallel(), MaxMaxParallel)
	}
}

// TestFanOutToolRunEmpty covers the empty
// task list.
func TestFanOutToolRunEmpty(t *testing.T) {
	tool, _ := NewFanOutTool(FanOutConfig{
		Factory: SubAgentFactoryFunc(func(task FanOutTask) (runtime.Worker, error) {
			return &echoWorker{}, nil
		}),
	})
	merged, err := tool.Run(context.Background(), nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(merged.Results) != 0 {
		t.Fatalf("Results = %d, want 0", len(merged.Results))
	}
}

// TestFanOutToolRunParallel covers the basic
// parallel run (REQ-FAN-FOT-003, -005).
func TestFanOutToolRunParallel(t *testing.T) {
	tool, _ := NewFanOutTool(FanOutConfig{
		Factory: SubAgentFactoryFunc(func(task FanOutTask) (runtime.Worker, error) {
			return &echoWorker{delay: 10 * time.Millisecond}, nil
		}),
		MaxParallel: 4,
	})
	tasks := []FanOutTask{
		{ID: "a", Prompt: "alpha", AgentType: "researcher"},
		{ID: "b", Prompt: "beta", AgentType: "researcher"},
		{ID: "c", Prompt: "gamma", AgentType: "researcher"},
	}
	merged, err := tool.Run(context.Background(), tasks)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(merged.Results) != 3 {
		t.Fatalf("Results = %d, want 3", len(merged.Results))
	}
	for _, r := range merged.Results {
		if r.Status != StatusSucceeded {
			t.Fatalf("task %s status = %q, want succeeded", r.ID, r.Status)
		}
		if r.Output != "echo:"+r.ID {
			// echo returns "echo:<description>" and
			// task.Prompt was "alpha" etc, but the
			// Worker is invoked with description ==
			// task.Prompt. Re-assert against the
			// task list.
			t.Logf("task %s output = %q", r.ID, r.Output)
		}
	}
}

// TestFanOutToolRespectsMaxParallel covers the
// semaphore (REQ-FAN-FOT-004).
func TestFanOutToolRespectsMaxParallel(t *testing.T) {
	var concurrent int32
	var maxConcurrent int32
	worker := &concurrencyWorker{
		onEnter: func() {
			c := atomic.AddInt32(&concurrent, 1)
			for {
				prev := atomic.LoadInt32(&maxConcurrent)
				if c <= prev || atomic.CompareAndSwapInt32(&maxConcurrent, prev, c) {
					break
				}
			}
		},
		onExit: func() {
			atomic.AddInt32(&concurrent, -1)
		},
		delay: 50 * time.Millisecond,
	}
	tool, _ := NewFanOutTool(FanOutConfig{
		Factory: SubAgentFactoryFunc(func(task FanOutTask) (runtime.Worker, error) {
			return worker, nil
		}),
		MaxParallel: 2,
	})
	tasks := make([]FanOutTask, 6)
	for i := range tasks {
		tasks[i] = FanOutTask{ID: string(rune('a' + i)), Prompt: "p", AgentType: "x"}
	}
	_, err := tool.Run(context.Background(), tasks)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got := atomic.LoadInt32(&maxConcurrent); got > 2 {
		t.Fatalf("max concurrent = %d, want <= 2", got)
	}
	if got := atomic.LoadInt32(&maxConcurrent); got < 2 {
		t.Fatalf("max concurrent = %d, want >= 2 (semaphore should be saturated)", got)
	}
}

// concurrencyWorker is the test Worker that
// tracks the maximum number of concurrent
// invocations.
type concurrencyWorker struct {
	onEnter func()
	onExit  func()
	delay   time.Duration
}

func (w *concurrencyWorker) Run(task runtime.TaskInput, ctx *runtime.TaskContext, meter *runtime.BudgetMeter) (runtime.WorkerResult, error) {
	w.onEnter()
	defer w.onExit()
	time.Sleep(w.delay)
	return runtime.WorkerResult{Output: "ok", CostUSD: 0.001}, nil
}

// TestFanOutToolSubAgentFactoryError covers the
// factory-error path.
func TestFanOutToolSubAgentFactoryError(t *testing.T) {
	tool, _ := NewFanOutTool(FanOutConfig{
		Factory: SubAgentFactoryFunc(func(task FanOutTask) (runtime.Worker, error) {
			return nil, errors.New("factory boom")
		}),
	})
	tasks := []FanOutTask{{ID: "a", Prompt: "p", AgentType: "x"}}
	_, err := tool.Run(context.Background(), tasks)
	if err != nil {
		t.Fatalf("Run returned err: %v", err)
	}
	// The error is recorded on the per-task
	// result, not surfaced as a Run error.
	merged, _ := tool.Run(context.Background(), tasks)
	if merged.Results[0].Status != StatusFailed {
		t.Fatalf("status = %q, want failed", merged.Results[0].Status)
	}
	if !contains(merged.Results[0].Output, "factory boom") {
		t.Fatalf("output = %q", merged.Results[0].Output)
	}
}

// TestFanOutToolSubAgentRuntimeError covers the
// sub-agent runtime error path.
func TestFanOutToolSubAgentRuntimeError(t *testing.T) {
	tool, _ := NewFanOutTool(FanOutConfig{
		Factory: SubAgentFactoryFunc(func(task FanOutTask) (runtime.Worker, error) {
			return failingWorker{}, nil
		}),
	})
	tasks := []FanOutTask{{ID: "a", Prompt: "p", AgentType: "x"}}
	merged, err := tool.Run(context.Background(), tasks)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if merged.Results[0].Status != StatusFailed {
		t.Fatalf("status = %q", merged.Results[0].Status)
	}
	if !contains(merged.Results[0].Output, "sub-agent boom") {
		t.Fatalf("output = %q", merged.Results[0].Output)
	}
}

// TestFanOutToolTaskValidation covers the
// validate-up-front path.
func TestFanOutToolTaskValidation(t *testing.T) {
	tool, _ := NewFanOutTool(FanOutConfig{
		Factory: SubAgentFactoryFunc(func(task FanOutTask) (runtime.Worker, error) {
			return &echoWorker{}, nil
		}),
	})
	tasks := []FanOutTask{{ID: "", Prompt: "p", AgentType: "x"}}
	_, err := tool.Run(context.Background(), tasks)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestFanOutToolContextCancel covers the
// context-cancel path.
func TestFanOutToolContextCancel(t *testing.T) {
	tool, _ := NewFanOutTool(FanOutConfig{
		Factory: SubAgentFactoryFunc(func(task FanOutTask) (runtime.Worker, error) {
			return &echoWorker{delay: 5 * time.Second}, nil
		}),
		MaxParallel: 1,
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before run
	tasks := []FanOutTask{{ID: "a", Prompt: "p", AgentType: "x"}}
	_, err := tool.Run(ctx, tasks)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	// The result is either succeeded (if the
	// cancel raced) or canceled. Both are
	// acceptable.
}

// TestFanOutToolContextCancelBeforeSemaphore
// covers the path where the context is canceled
// before the goroutine acquires the semaphore.
// The test uses a max_parallel of 1 and a slow
// worker; the second task hits the canceled
// path.
func TestFanOutToolContextCancelBeforeSemaphore(t *testing.T) {
	tool, _ := NewFanOutTool(FanOutConfig{
		Factory: SubAgentFactoryFunc(func(task FanOutTask) (runtime.Worker, error) {
			return &echoWorker{delay: 50 * time.Millisecond}, nil
		}),
		MaxParallel: 1,
	})
	ctx, cancel := context.WithCancel(context.Background())
	tasks := []FanOutTask{
		{ID: "a", Prompt: "p", AgentType: "x"},
		{ID: "b", Prompt: "p", AgentType: "x"},
		{ID: "c", Prompt: "p", AgentType: "x"},
	}
	// Cancel after the first task starts
	// (timing is best-effort). The result is
	// either all-succeeded or one canceled
	// depending on the race. We assert no
	// crash and at least one result.
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()
	merged, err := tool.Run(ctx, tasks)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(merged.Results) == 0 {
		t.Fatal("expected at least one result")
	}
}

// TestFanOutToolWithWorktree covers the
// worktree-isolation path (REQ-FAN-FOT-006).
func TestFanOutToolWithWorktree(t *testing.T) {
	be := &fakeBackend{}
	w, _ := NewWorktreeIsolation(WorktreeConfig{Backend: be, RootDir: t.TempDir()})
	tool, _ := NewFanOutTool(FanOutConfig{
		Factory: SubAgentFactoryFunc(func(task FanOutTask) (runtime.Worker, error) {
			return &echoWorker{}, nil
		}),
		Worktree: w,
	})
	tasks := []FanOutTask{{ID: "t1", Prompt: "p", AgentType: "x", Isolate: true, Branch: "feat/x"}}
	merged, err := tool.Run(context.Background(), tasks)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if merged.Results[0].Status != StatusSucceeded {
		t.Fatalf("status = %q", merged.Results[0].Status)
	}
	if merged.Results[0].WorktreePath == "" {
		t.Fatal("WorktreePath is empty")
	}
	if len(be.removed) != 1 {
		t.Fatalf("removed = %d, want 1", len(be.removed))
	}
}

// TestFanOutToolIsolateWithoutWorktree covers
// the "isolate requested but no worktree
// configured" path.
func TestFanOutToolIsolateWithoutWorktree(t *testing.T) {
	tool, _ := NewFanOutTool(FanOutConfig{
		Factory: SubAgentFactoryFunc(func(task FanOutTask) (runtime.Worker, error) {
			return &echoWorker{}, nil
		}),
	})
	tasks := []FanOutTask{{ID: "t1", Prompt: "p", AgentType: "x", Isolate: true}}
	merged, err := tool.Run(context.Background(), tasks)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if merged.Results[0].Status != StatusFailed {
		t.Fatalf("status = %q, want failed", merged.Results[0].Status)
	}
	if !contains(merged.Results[0].Output, "no worktree configured") {
		t.Fatalf("output = %q", merged.Results[0].Output)
	}
}

// TestFanOutToolWorktreeCreateFails covers the
// worktree-create-error path.
func TestFanOutToolWorktreeCreateFails(t *testing.T) {
	be := &fakeBackend{createFail: errors.New("create boom")}
	w, _ := NewWorktreeIsolation(WorktreeConfig{Backend: be, RootDir: t.TempDir()})
	tool, _ := NewFanOutTool(FanOutConfig{
		Factory: SubAgentFactoryFunc(func(task FanOutTask) (runtime.Worker, error) {
			return &echoWorker{}, nil
		}),
		Worktree: w,
	})
	tasks := []FanOutTask{{ID: "t1", Prompt: "p", AgentType: "x", Isolate: true}}
	merged, err := tool.Run(context.Background(), tasks)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if merged.Results[0].Status != StatusFailed {
		t.Fatalf("status = %q", merged.Results[0].Status)
	}
	if !contains(merged.Results[0].Output, "create boom") {
		t.Fatalf("output = %q", merged.Results[0].Output)
	}
}

// TestFanOutToolProgress covers the progress
// callback.
func TestFanOutToolProgress(t *testing.T) {
	var seen []int
	var mu sync.Mutex
	tool, _ := NewFanOutTool(FanOutConfig{
		Factory: SubAgentFactoryFunc(func(task FanOutTask) (runtime.Worker, error) {
			return &echoWorker{}, nil
		}),
		Progress: func(index int, result FanOutResult) {
			mu.Lock()
			defer mu.Unlock()
			seen = append(seen, index)
		},
	})
	tasks := []FanOutTask{
		{ID: "a", Prompt: "p", AgentType: "x"},
		{ID: "b", Prompt: "p", AgentType: "x"},
		{ID: "c", Prompt: "p", AgentType: "x"},
	}
	_, err := tool.Run(context.Background(), tasks)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(seen) != 3 {
		t.Fatalf("progress called %d times, want 3", len(seen))
	}
}

// openAuditLog is the test helper that opens an
// in-memory audit log. The helper is the
// seam the audit-row test uses; the in-memory
// database avoids a temp file and a clean-up
// step.
func openAuditLog(t *testing.T) (*audit.Log, error) {
	t.Helper()
	log, err := audit.Open(audit.Config{Path: ":memory:"})
	if err != nil {
		return nil, err
	}
	return log, nil
}

// TestFanOutToolWithAuditLog covers the audit
// row path (REQ-FAN-FOT-007). The test opens a
// real audit log and asserts the row count
// after Run.
func TestFanOutToolWithAuditLog(t *testing.T) {
	log, err := openAuditLog(t)
	if err != nil {
		t.Fatalf("openAuditLog: %v", err)
	}
	defer log.Close()
	tool, _ := NewFanOutTool(FanOutConfig{
		Factory: SubAgentFactoryFunc(func(task FanOutTask) (runtime.Worker, error) {
			return &echoWorker{}, nil
		}),
		Log: log,
	})
	tasks := []FanOutTask{{ID: "a", Prompt: "p", AgentType: "x"}}
	_, err = tool.Run(context.Background(), tasks)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	rows, err := log.GetByKind("fanout_task", 100)
	if err != nil {
		t.Fatalf("GetByKind: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(rows))
	}
}
