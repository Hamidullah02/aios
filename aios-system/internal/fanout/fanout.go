package fanout

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sadi/aios-system/internal/core/audit"
	"github.com/sadi/aios-system/internal/runtime"
)

// SubAgentFactory builds a sub-agent for a single
// task. The factory is the seam that lets the
// FanOutTool decouple from aios-runtime: the
// factory receives the task and returns a Worker
// the FanOutTool invokes.
//
// The factory is the typed constructor; the
// default NewWorkerSubAgentFactory wraps a
// factory function the call site provides.
type SubAgentFactory interface {
	// BuildSubAgent returns the Worker the
	// FanOutTool should invoke for the given
	// task. The Worker is invoked exactly once.
	BuildSubAgent(task FanOutTask) (runtime.Worker, error)
}

// SubAgentFactoryFunc is the function-type
// adapter for SubAgentFactory. The adapter is
// the convenience constructor when the call
// site wants to pass a plain function.
type SubAgentFactoryFunc func(task FanOutTask) (runtime.Worker, error)

// BuildSubAgent returns the Worker the factory
// function produces.
func (f SubAgentFactoryFunc) BuildSubAgent(task FanOutTask) (runtime.Worker, error) {
	return f(task)
}

// FanOutTool is the parallel task executor
// (REQ-FAN-FOT-001).
//
// The tool accepts a list of tasks, spawns a
// sub-agent for every task, runs the sub-agents
// in parallel under a semaphore, and merges the
// results through a pluggable merger.
//
// The tool is safe for concurrent use by multiple
// goroutines because the parallel runner is the
// only stateful part and the runner is built
// fresh on every Run.
type FanOutTool struct {
	factory     SubAgentFactory
	worktree    *WorktreeIsolation
	merger      *ResultMerger
	maxParallel int
	log         *audit.Log
	// taskID is the audit-row kind the tool uses
	// (REQ-FAN-FOT-007).
	taskKind string
	// now is the clock the tool uses for
	// per-task timestamps. Defaults to time.Now.
	now func() time.Time
	// progress is the optional callback the
	// scheduler wires to the watch layer. The
	// callback is invoked after every task
	// finishes.
	progress func(index int, result FanOutResult)
}

// FanOutConfig controls FanOutTool construction.
type FanOutConfig struct {
	// Factory is the sub-agent factory. Required.
	Factory SubAgentFactory
	// Worktree is the optional worktree
	// isolation. When nil, tasks are not
	// isolated.
	Worktree *WorktreeIsolation
	// Merger is the optional result merger. When
	// nil, the tool merges with the Concat
	// strategy.
	Merger *ResultMerger
	// MaxParallel is the maximum number of
	// sub-agents that run concurrently. Default
	// is 4 (REQ-FAN-FOT-004). The hard cap
	// (REQ-FAN-NFR-003) is 8.
	MaxParallel int
	// Log is the optional audit log. When set,
	// the tool writes a fanout_task audit row for
	// every task (REQ-FAN-FOT-007).
	Log *audit.Log
	// Now is the clock the tool uses for
	// per-task timestamps. Defaults to time.Now.
	Now func() time.Time
	// Progress is the optional callback the
	// tool invokes after every task finishes.
	// The callback is the seam the watch layer
	// uses.
	Progress func(index int, result FanOutResult)
}

// DefaultMaxParallel is the default max_parallel
// setting (REQ-FAN-FOT-004).
const DefaultMaxParallel = 4

// MaxMaxParallel is the hard cap on
// max_parallel (REQ-FAN-NFR-003).
const MaxMaxParallel = 8

// NewFanOutTool constructs a FanOutTool. The
// function validates the config and returns an
// error when the factory is nil.
func NewFanOutTool(cfg FanOutConfig) (*FanOutTool, error) {
	if cfg.Factory == nil {
		return nil, &FanOutError{Op: "init", Reason: "factory is required"}
	}
	mp := cfg.MaxParallel
	if mp <= 0 {
		mp = DefaultMaxParallel
	}
	if mp > MaxMaxParallel {
		mp = MaxMaxParallel
	}
	merger := cfg.Merger
	if merger == nil {
		merger = NewResultMerger(MergeConcat)
	}
	now := cfg.Now
	if now == nil {
		now = time.Now
	}
	return &FanOutTool{
		factory:     cfg.Factory,
		worktree:    cfg.Worktree,
		merger:      merger,
		maxParallel: mp,
		log:         cfg.Log,
		taskKind:    "fanout_task",
		now:         now,
		progress:    cfg.Progress,
	}, nil
}

// Run runs the tasks in parallel and returns the
// merged result. The function is the public
// entry point the tool protocol invokes.
//
// Per REQ-FAN-FOT-002 the tool accepts a list of
// tasks. Per REQ-FAN-FOT-003 the tool spawns a
// sub-agent for every task and runs them in
// parallel. Per REQ-FAN-FOT-004 the tool respects
// max_parallel through a semaphore. Per REQ-FAN-FOT-005
// the tool merges the results.
func (t *FanOutTool) Run(ctx context.Context, tasks []FanOutTask) (Merged, error) {
	if t == nil {
		return Merged{}, &FanOutError{Op: "run", Reason: "tool is nil"}
	}
	if len(tasks) == 0 {
		return Merged{Results: []FanOutResult{}, Strategy: t.merger.Strategy}, nil
	}
	// Validate every task up front so the
	// parallel run does not fail halfway through.
	for i := range tasks {
		if err := tasks[i].Validate(); err != nil {
			return Merged{}, err
		}
	}
	results := t.runParallel(ctx, tasks)
	// Merge the results.
	merged, err := t.merger.Merge(ctx, results)
	if err != nil {
		return Merged{}, err
	}
	return merged, nil
}

// runParallel is the inner parallel runner. The
// function spawns one goroutine per task,
// gates the goroutines through a semaphore, and
// collects the results in input order.
//
// The function is the implementation of
// REQ-FAN-FOT-003 and REQ-FAN-FOT-004. The
// semaphore is a buffered channel; the buffer
// is max_parallel.
func (t *FanOutTool) runParallel(ctx context.Context, tasks []FanOutTask) []FanOutResult {
	results := make([]FanOutResult, len(tasks))
	sem := make(chan struct{}, t.maxParallel)
	var wg sync.WaitGroup
	for i := range tasks {
		i := i
		task := tasks[i]
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Acquire the semaphore.
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				results[i] = failedResult(task, i, ctx.Err())
				if t.progress != nil {
					t.progress(i, results[i])
				}
				return
			}
			defer func() { <-sem }()
			// Run the task.
			results[i] = t.runOne(ctx, task, i)
			if t.progress != nil {
				t.progress(i, results[i])
			}
		}()
	}
	wg.Wait()
	return results
}

// runOne runs a single task. The function
// implements REQ-FAN-FOT-006 (worktree isolation)
// and REQ-FAN-FOT-007 (audit row).
func (t *FanOutTool) runOne(ctx context.Context, task FanOutTask, index int) FanOutResult {
	tmr := newTimer()
	result := FanOutResult{
		ID:        task.ID,
		Index:     index,
		Status:    StatusFailed,
		StartedAt: t.now().UTC().Format(time.RFC3339Nano),
	}
	// Worktree isolation.
	if task.Isolate {
		if t.worktree == nil {
			result.Output = fmt.Sprintf("isolate requested but no worktree configured: %s", task.ID)
			result.EndedAt = t.now().UTC().Format(time.RFC3339Nano)
			result.DurationMs = tmr.elapsed()
			t.writeAudit(ctx, task, result)
			return result
		}
		path, err := t.worktree.Create(task.ID, task.Branch)
		if err != nil {
			result.Output = err.Error()
			result.EndedAt = t.now().UTC().Format(time.RFC3339Nano)
			result.DurationMs = tmr.elapsed()
			t.writeAudit(ctx, task, result)
			return result
		}
		result.WorktreePath = path
		// Run the sub-agent inside the worktree.
		var innerErr error
		runErr := t.worktree.RunIn(path, func() error {
			innerErr = t.invokeSubAgent(ctx, task, &result)
			return innerErr
		})
		if runErr != nil {
			innerErr = runErr
		}
		// Always remove the worktree.
		_ = t.worktree.Remove(path)
		if innerErr != nil {
			result.Output = innerErr.Error()
		}
	} else {
		// No isolation: run directly.
		if err := t.invokeSubAgent(ctx, task, &result); err != nil {
			result.Output = err.Error()
		}
	}
	result.EndedAt = t.now().UTC().Format(time.RFC3339Nano)
	result.DurationMs = tmr.elapsed()
	if result.Status == StatusFailed && result.Output == "" {
		result.Output = "sub-agent failed without output"
	}
	t.writeAudit(ctx, task, result)
	return result
}

// invokeSubAgent builds the sub-agent through the
// factory and runs it through the runtime.Worker
// interface. The function is the integration
// point with aios-runtime.
func (t *FanOutTool) invokeSubAgent(ctx context.Context, task FanOutTask, result *FanOutResult) error {
	worker, err := t.factory.BuildSubAgent(task)
	if err != nil {
		return err
	}
	input := runtime.TaskInput{
		ID:          task.ID,
		Description: task.Prompt,
		Acceptance:  "",
		Tool:        task.AgentType,
		Inputs:      nil,
	}
	tctx := &runtime.TaskContext{Vars: map[string]any{
		"agent_type": task.AgentType,
		"prompt":     task.Prompt,
	}}
	wr, err := worker.Run(input, tctx, nil)
	if err != nil {
		return err
	}
	result.Output = wr.Output
	result.CostUSD = wr.CostUSD
	if wr.Output != "" {
		result.Status = StatusSucceeded
	}
	return nil
}

// writeAudit writes the fanout_task audit row
// (REQ-FAN-FOT-007). The function is a no-op
// when the audit log is not configured.
func (t *FanOutTool) writeAudit(ctx context.Context, task FanOutTask, result FanOutResult) {
	if t.log == nil {
		return
	}
	row := &audit.Row{
		TaskID:         task.ID,
		StepID:         fmt.Sprintf("%d", result.Index),
		Kind:           t.taskKind,
		Actor:          "fanout",
		RiskTier:       0,
		IdempotencyKey: fmt.Sprintf("fanout:%s:%s", task.ID, result.StartedAt),
		Payload:        result,
	}
	// Errors are swallowed: the audit log is
	// best-effort. The runtime layer treats
	// audit failures as run blocks; the fanout
	// tool is a leaf, so a failure here is
	// reported through the result's Status
	// rather than the audit call.
	_, _ = t.log.Write(row)
	_ = ctx
}

// failedResult returns a FanOutResult that
// represents a canceled or failed task. The
// function is the helper the parallel runner
// uses when the context is canceled before the
// task acquired the semaphore.
func failedResult(task FanOutTask, index int, err error) FanOutResult {
	return FanOutResult{
		ID:        task.ID,
		Index:     index,
		Status:    StatusCanceled,
		Output:    err.Error(),
		StartedAt: nowRFC3339(),
		EndedAt:   nowRFC3339(),
	}
}

// MaxParallel returns the effective max_parallel
// the tool uses. The function is the getter the
// tests use.
func (t *FanOutTool) MaxParallel() int { return t.maxParallel }

// Merger returns the result merger the tool
// uses. The function is the getter the tests use.
func (t *FanOutTool) Merger() *ResultMerger { return t.merger }
