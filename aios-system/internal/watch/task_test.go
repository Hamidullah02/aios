package watch

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Hamidullah02/aios/aios-system/internal/core/audit"
	"github.com/Hamidullah02/aios/aios-system/internal/core/clock"
	"github.com/Hamidullah02/aios/aios-system/internal/runtime"
	"github.com/Hamidullah02/aios/aios-system/internal/tools"
)

// echoWorker is a runtime.Worker that returns its prompt
// verbatim. Used in the parallel-calls test.
type echoWorker struct {
	delay   time.Duration
	got     []string
	mu      sync.Mutex
	active  int32
	maxSeen int32
}

func (e *echoWorker) Run(in runtime.TaskInput, _ *runtime.TaskContext, _ *runtime.BudgetMeter) (runtime.WorkerResult, error) {
	atomic.AddInt32(&e.active, 1)
	for {
		ms := atomic.LoadInt32(&e.maxSeen)
		if atomic.LoadInt32(&e.active) <= ms {
			break
		}
		if atomic.CompareAndSwapInt32(&e.maxSeen, ms, atomic.LoadInt32(&e.active)) {
			break
		}
	}
	defer atomic.AddInt32(&e.active, -1)
	if e.delay > 0 {
		time.Sleep(e.delay)
	}
	e.mu.Lock()
	e.got = append(e.got, in.Description)
	e.mu.Unlock()
	return runtime.WorkerResult{Output: "echo: " + in.Description}, nil
}

// failingWorker always returns an error.
type failingWorker struct{ msg string }

func (f failingWorker) Run(runtime.TaskInput, *runtime.TaskContext, *runtime.BudgetMeter) (runtime.WorkerResult, error) {
	return runtime.WorkerResult{}, errors.New(f.msg)
}

func openTestAudit2(t *testing.T) *audit.Log {
	t.Helper()
	log, err := audit.Open(audit.Config{Path: ":memory:", Clock: clock.FixedDecision{}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { log.Close() })
	return log
}

func makeTool(t *testing.T, w *echoWorker) *TaskTool {
	t.Helper()
	return &TaskTool{
		Factory: func(agentType AgentType) (runtime.Worker, error) {
			if w == nil {
				return nil, errors.New("no worker")
			}
			return w, nil
		},
	}
}

// TestTaskTool_SingleCall covers REQ-WTC-SBT-002 / 004.
func TestTaskTool_SingleCall(t *testing.T) {
	w := &echoWorker{}
	tool := makeTool(t, w)
	res, err := tool.Run(map[string]any{
		"prompt":     "hi",
		"agent_type": "explore",
	}, tools.Context{})
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK {
		t.Fatalf("res = %+v", res)
	}
	rs, _ := res.Output["results"].([]string)
	if len(rs) != 1 || rs[0] != "echo: hi" {
		t.Fatalf("results = %+v", rs)
	}
	if len(w.got) != 1 || w.got[0] != "hi" {
		t.Fatalf("worker got = %+v", w.got)
	}
}

// TestTaskTool_ParallelCalls covers REQ-WTC-SBT-007.
func TestTaskTool_ParallelCalls(t *testing.T) {
	w := &echoWorker{delay: 30 * time.Millisecond}
	tool := &TaskTool{
		Factory:     func(AgentType) (runtime.Worker, error) { return w, nil },
		MaxParallel: 2,
	}
	start := time.Now()
	res, err := tool.Run(map[string]any{
		"calls": []any{
			map[string]any{"prompt": "a", "agent_type": "general"},
			map[string]any{"prompt": "b", "agent_type": "general"},
			map[string]any{"prompt": "c", "agent_type": "general"},
			map[string]any{"prompt": "d", "agent_type": "general"},
		},
	}, tools.Context{})
	if err != nil {
		t.Fatal(err)
	}
	dur := time.Since(start)
	// 4 calls, 2 in parallel, 30ms each => ~60ms total.
	// Allow generous slack for CI.
	if dur > 250*time.Millisecond {
		t.Fatalf("expected parallel runs (~60ms), took %v", dur)
	}
	rs, _ := res.Output["results"].([]string)
	if len(rs) != 4 {
		t.Fatalf("results = %d, want 4", len(rs))
	}
}

// TestTaskTool_ParallelCalls_RespectsMaxParallel covers the
// semaphore cap.
func TestTaskTool_ParallelCalls_RespectsMaxParallel(t *testing.T) {
	var (
		maxActive  int32
		currentAct int32
	)
	w := &parallelWorker{
		beforeRun: func() {
			n := atomic.AddInt32(&currentAct, 1)
			for {
				m := atomic.LoadInt32(&maxActive)
				if n <= m || atomic.CompareAndSwapInt32(&maxActive, m, n) {
					break
				}
			}
		},
		afterRun: func() { atomic.AddInt32(&currentAct, -1) },
		delay:    20 * time.Millisecond,
	}
	tool := &TaskTool{
		Factory:     func(AgentType) (runtime.Worker, error) { return w, nil },
		MaxParallel: 2,
	}
	_, err := tool.Run(map[string]any{
		"calls": []any{
			map[string]any{"prompt": "a", "agent_type": "general"},
			map[string]any{"prompt": "b", "agent_type": "general"},
			map[string]any{"prompt": "c", "agent_type": "general"},
			map[string]any{"prompt": "d", "agent_type": "general"},
		},
	}, tools.Context{})
	if err != nil {
		t.Fatal(err)
	}
	if got := atomic.LoadInt32(&maxActive); got > 2 {
		t.Fatalf("max active = %d, want <= 2", got)
	}
}

// parallelWorker is a runtime.Worker that records the
// concurrent call count. Used in the semaphore test.
type parallelWorker struct {
	beforeRun func()
	afterRun  func()
	delay     time.Duration
}

func (p *parallelWorker) Run(in runtime.TaskInput, _ *runtime.TaskContext, _ *runtime.BudgetMeter) (runtime.WorkerResult, error) {
	if p.beforeRun != nil {
		p.beforeRun()
	}
	if p.afterRun != nil {
		defer p.afterRun()
	}
	if p.delay > 0 {
		time.Sleep(p.delay)
	}
	return runtime.WorkerResult{Output: in.Description}, nil
}

// TestTaskTool_DefaultsToGeneral covers the missing agent_type
// path.
func TestTaskTool_DefaultsToGeneral(t *testing.T) {
	w := &echoWorker{}
	tool := makeTool(t, w)
	if _, err := tool.Run(map[string]any{"prompt": "hi"}, tools.Context{}); err != nil {
		t.Fatal(err)
	}
	if len(w.got) != 1 {
		t.Fatalf("worker called %d times", len(w.got))
	}
}

// TestTaskTool_MissingPrompt covers the validation path.
func TestTaskTool_MissingPrompt(t *testing.T) {
	tool := makeTool(t, &echoWorker{})
	_, err := tool.Run(map[string]any{"agent_type": "explore"}, tools.Context{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !errorsIs(err, ErrSubAgent) {
		t.Fatalf("err = %v, want SubAgentError", err)
	}
}

// TestTaskTool_BadAgentType covers the validation path.
func TestTaskTool_BadAgentType(t *testing.T) {
	tool := makeTool(t, &echoWorker{})
	_, err := tool.Run(map[string]any{"prompt": "hi", "agent_type": "nope"}, tools.Context{})
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestTaskTool_FactoryError covers the factory-error path.
func TestTaskTool_FactoryError(t *testing.T) {
	tool := &TaskTool{
		Factory: func(AgentType) (runtime.Worker, error) {
			return nil, errors.New("nope")
		},
	}
	_, err := tool.Run(map[string]any{"prompt": "hi"}, tools.Context{})
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestTaskTool_WorkerError covers the sub-agent error path.
func TestTaskTool_WorkerError(t *testing.T) {
	tool := &TaskTool{
		Factory: func(AgentType) (runtime.Worker, error) {
			return failingWorker{msg: "boom"}, nil
		},
	}
	_, err := tool.Run(map[string]any{"prompt": "hi"}, tools.Context{})
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestTaskTool_AuditRow covers REQ-WTC-SBT-006.
func TestTaskTool_AuditRow(t *testing.T) {
	log := openTestAudit2(t)
	tool := &TaskTool{
		Factory: func(AgentType) (runtime.Worker, error) { return &echoWorker{}, nil },
		Audit:   log,
	}
	_, err := tool.Run(map[string]any{
		"prompt":      "find x",
		"agent_type":  "explore",
		"description": "find the audit log call sites",
	}, tools.Context{TaskID: "task-y", Actor: "alice"})
	if err != nil {
		t.Fatal(err)
	}
	rows, err := log.GetByKind("subagent_task", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows = %d", len(rows))
	}
	if rows[0].TaskID != "task-y" || rows[0].Actor != "alice" {
		t.Fatalf("row = %+v", rows[0])
	}
}

// TestAgentType_IsValid covers the enum gate.
func TestAgentType_IsValid(t *testing.T) {
	for _, a := range AllAgentTypes {
		if !a.IsValid() {
			t.Fatalf("%s should be valid", a)
		}
	}
	if AgentType("nope").IsValid() {
		t.Fatal("nope should not be valid")
	}
	if AgentType("").IsValid() {
		t.Fatal("empty should not be valid")
	}
}

// TestNormalizeCalls covers the single-vs-list dispatch.
func TestNormalizeCalls(t *testing.T) {
	single := normalizeCalls(map[string]any{"prompt": "p", "agent_type": "explore"})
	if len(single) != 1 || single[0].Prompt != "p" || single[0].AgentType != AgentExplore {
		t.Fatalf("single = %+v", single)
	}
	list := normalizeCalls(map[string]any{
		"calls": []any{
			map[string]any{"prompt": "a", "agent_type": "general"},
			map[string]any{"prompt": "b", "agent_type": "code"},
		},
	})
	if len(list) != 2 || list[0].Prompt != "a" || list[1].AgentType != AgentCode {
		t.Fatalf("list = %+v", list)
	}
}

// TestSubAgentError_IsCovers covers the errors.Is match.
func TestSubAgentError_IsCovers(t *testing.T) {
	e := &SubAgentError{Reason: "x"}
	if !errorsIs(e, ErrSubAgent) {
		t.Fatal("SubAgentError should match ErrSubAgent")
	}
}

func errorsIs(err, target error) bool {
	for err != nil {
		if err == target {
			return true
		}
		if se, ok := err.(*SubAgentError); ok && se.Is(target) {
			return true
		}
		type unwrapper interface{ Unwrap() error }
		if u, ok := err.(unwrapper); ok {
			err = u.Unwrap()
			continue
		}
		break
	}
	return false
}
