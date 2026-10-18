package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/sadi/aios-system/internal/core/audit"
	"github.com/sadi/aios-system/internal/core/clock"
	"github.com/sadi/aios-system/internal/core/schemas"
	"github.com/sadi/aios-system/internal/core/state"
)

// fakeWorker is a programmable Worker for the orchestrator tests.
type fakeWorker struct {
	// runFn is the per-task body. The slice index is the task index.
	runFn func(idx int, input TaskInput, meter *BudgetMeter) (WorkerResult, error)
}

func (f *fakeWorker) Run(input TaskInput, _ *TaskContext, meter *BudgetMeter) (WorkerResult, error) {
	// The orchestrator passes a TaskInput without an index. The
	// fake worker uses the id as a hint.
	return f.runFn(0, input, meter)
}

// fakeVerifier returns a verdict from a programmable function.
type fakeVerifier struct {
	verdict VerifierVerdict
	err     error
}

func (f *fakeVerifier) Verify(_ TaskInput, _ WorkerResult) (VerifierVerdict, error) {
	return f.verdict, f.err
}

func newTestAuditLog(t *testing.T) *audit.Log {
	t.Helper()
	log, err := audit.Open(audit.Config{
		Path:  ":memory:",
		Clock: clock.FixedDecision{T: time.Unix(1700000000, 0)},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { log.Close() })
	return log
}

func newTestPlan() *schemas.TaskPlan {
	return &schemas.TaskPlan{
		ID: "p1", Goal: "ship",
		Tasks: []schemas.TaskSpec{
			{ID: "st1", Description: "design", Acceptance: "doc"},
			{ID: "st2", Description: "implement", Acceptance: "tests pass"},
		},
	}
}

func TestOrchestrator_Run_HappyPath(t *testing.T) {
	log := newTestAuditLog(t)
	meter := NewBudgetMeter(Budget{})
	worker := &fakeWorker{runFn: func(idx int, _ TaskInput, _ *BudgetMeter) (WorkerResult, error) {
		return WorkerResult{Output: "ok", Data: map[string]any{"k": "v"}}, nil
	}}
	verifier := &fakeVerifier{verdict: VerdictPass}
	o := New(Config{
		Worker:   worker,
		Verifier: verifier,
		Meter:    meter,
		Audit:    log,
		Resume:   NewResumeHandler(t.TempDir()),
	})
	report, err := o.Run(context.Background(), newTestPlan(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if report.StoppedReason != "completed" {
		t.Fatalf("StoppedReason = %q", report.StoppedReason)
	}
	if report.TaskStates["st1"] != state.StateCompleted {
		t.Fatalf("st1 state = %q", report.TaskStates["st1"])
	}
	if report.TaskStates["st2"] != state.StateCompleted {
		t.Fatalf("st2 state = %q", report.TaskStates["st2"])
	}
}

func TestOrchestrator_Run_VerifierFail(t *testing.T) {
	log := newTestAuditLog(t)
	meter := NewBudgetMeter(Budget{})
	worker := &fakeWorker{runFn: func(_ int, _ TaskInput, _ *BudgetMeter) (WorkerResult, error) {
		return WorkerResult{Output: "ok"}, nil
	}}
	verifier := &fakeVerifier{verdict: VerdictFail}
	o := New(Config{
		Worker:   worker,
		Verifier: verifier,
		Meter:    meter,
		Audit:    log,
		Resume:   NewResumeHandler(t.TempDir()),
	})
	report, _ := o.Run(context.Background(), newTestPlan(), nil)
	if report.StoppedReason != "verifier_fail" {
		t.Fatalf("StoppedReason = %q", report.StoppedReason)
	}
	if report.TaskStates["st1"] != state.StateFailed {
		t.Fatalf("st1 state = %q", report.TaskStates["st1"])
	}
}

func TestOrchestrator_Run_InconclusiveTreatedAsFail(t *testing.T) {
	log := newTestAuditLog(t)
	worker := &fakeWorker{runFn: func(_ int, _ TaskInput, _ *BudgetMeter) (WorkerResult, error) {
		return WorkerResult{Output: "ok"}, nil
	}}
	verifier := &fakeVerifier{verdict: VerdictInconclusive}
	o := New(Config{
		Worker:   worker,
		Verifier: verifier,
		Meter:    NewBudgetMeter(Budget{}),
		Audit:    log,
		Resume:   NewResumeHandler(t.TempDir()),
	})
	report, _ := o.Run(context.Background(), newTestPlan(), nil)
	if report.StoppedReason != "verifier_fail" {
		t.Fatalf("inconclusive should be fail: %q", report.StoppedReason)
	}
}

func TestOrchestrator_Run_WaitingForApproval(t *testing.T) {
	log := newTestAuditLog(t)
	worker := &fakeWorker{runFn: func(_ int, _ TaskInput, _ *BudgetMeter) (WorkerResult, error) {
		return WorkerResult{}, &WaitingForApproval{
			ApprovalID: "apr-1",
			Decision:   "ask",
		}
	}}
	o := New(Config{
		Worker:   worker,
		Verifier: &fakeVerifier{verdict: VerdictPass},
		Meter:    NewBudgetMeter(Budget{}),
		Audit:    log,
		Resume:   NewResumeHandler(t.TempDir()),
	})
	_, err := o.Run(context.Background(), newTestPlan(), nil)
	var wfa *WaitingForApproval
	if !errors.As(err, &wfa) {
		t.Fatalf("err type = %T, want *WaitingForApproval", err)
	}
	if wfa.ApprovalID != "apr-1" {
		t.Fatalf("approval id = %q", wfa.ApprovalID)
	}
}

func TestOrchestrator_Run_BudgetExhausted(t *testing.T) {
	log := newTestAuditLog(t)
	meter := NewBudgetMeter(Budget{MaxToolCalls: 1})
	calls := 0
	worker := &fakeWorker{runFn: func(_ int, _ TaskInput, m *BudgetMeter) (WorkerResult, error) {
		calls++
		m.RecordToolCall()
		return WorkerResult{Output: "ok"}, nil
	}}
	o := New(Config{
		Worker:   worker,
		Verifier: &fakeVerifier{verdict: VerdictPass},
		Meter:    meter,
		Audit:    log,
		Resume:   NewResumeHandler(t.TempDir()),
	})
	report, _ := o.Run(context.Background(), newTestPlan(), nil)
	if report.StoppedReason != "budget_exhausted" {
		t.Fatalf("StoppedReason = %q", report.StoppedReason)
	}
	// Only the first task ran; the second was gated by the
	// budget check at the top of the loop.
	if calls != 1 {
		t.Fatalf("calls = %d, want 1", calls)
	}
	// Checkpoint was saved with the second task's index.
	cp, err := o.resume.Load("p1")
	if err != nil {
		t.Fatal(err)
	}
	if cp.TaskIndex != 1 {
		t.Fatalf("checkpoint.TaskIndex = %d, want 1", cp.TaskIndex)
	}
}

func TestOrchestrator_Run_NoProgress(t *testing.T) {
	log := newTestAuditLog(t)
	worker := &fakeWorker{runFn: func(_ int, _ TaskInput, _ *BudgetMeter) (WorkerResult, error) {
		return WorkerResult{Output: "same"}, nil
	}}
	o := New(Config{
		Worker:   worker,
		Verifier: &fakeVerifier{verdict: VerdictPass},
		Meter:    NewBudgetMeter(Budget{}),
		NoProg:   NewNoProgressDetector(1), // trip on the second identical output
		Audit:    log,
		Resume:   NewResumeHandler(t.TempDir()),
	})
	report, _ := o.Run(context.Background(), newTestPlan(), nil)
	if report.StoppedReason != "no_progress" {
		t.Fatalf("StoppedReason = %q", report.StoppedReason)
	}
}

func TestOrchestrator_Cancel(t *testing.T) {
	log := newTestAuditLog(t)
	started := make(chan struct{})
	release := make(chan struct{})
	worker := &fakeWorker{runFn: func(_ int, _ TaskInput, _ *BudgetMeter) (WorkerResult, error) {
		close(started)
		<-release
		return WorkerResult{Output: "ok"}, nil
	}}
	o := New(Config{
		Worker:   worker,
		Verifier: &fakeVerifier{verdict: VerdictPass},
		Meter:    NewBudgetMeter(Budget{}),
		Audit:    log,
		Resume:   NewResumeHandler(t.TempDir()),
	})
	plan := newTestPlan()
	ctx := context.Background()
	done := make(chan *Report, 1)
	go func() { r, _ := o.Run(ctx, plan, nil); done <- r }()
	<-started
	o.Cancel()
	close(release)
	report := <-done
	if report.StoppedReason != "canceled" {
		t.Fatalf("StoppedReason = %q", report.StoppedReason)
	}
}

func TestOrchestrator_ResumeFromCheckpoint(t *testing.T) {
	log := newTestAuditLog(t)
	resumeDir := t.TempDir()
	resume := NewResumeHandler(resumeDir)

	// Seed a checkpoint that says we already finished task 0
	// and are about to start task 1.
	cp := &Checkpoint{
		TaskID:    "p1",
		TaskIndex: 1,
		Plan:      mustJSON(newTestPlan()),
		Context:   mustJSON(&TaskContext{Vars: map[string]any{"k": "v"}}),
		Budget:    Budget{},
	}
	if err := resume.Save(cp); err != nil {
		t.Fatal(err)
	}

	worker := &fakeWorker{runFn: func(_ int, _ TaskInput, _ *BudgetMeter) (WorkerResult, error) {
		return WorkerResult{Output: "ok"}, nil
	}}
	o := New(Config{
		Worker:   worker,
		Verifier: &fakeVerifier{verdict: VerdictPass},
		Meter:    NewBudgetMeter(Budget{}),
		Audit:    log,
		Resume:   resume,
	})
	loaded, err := resume.Load("p1")
	if err != nil {
		t.Fatal(err)
	}
	plan := newTestPlan()
	report, err := o.Run(context.Background(), plan, loaded)
	if err != nil {
		t.Fatal(err)
	}
	if report.FromIndex != 1 {
		t.Fatalf("FromIndex = %d, want 1", report.FromIndex)
	}
	// Both tasks should be marked completed (st1 from the
	// checkpoint, st2 from this run).
	if report.TaskStates["st2"] != state.StateCompleted {
		t.Fatalf("st2 state = %q", report.TaskStates["st2"])
	}
}

func TestOrchestrator_Run_NilPlan(t *testing.T) {
	o := New(Config{Worker: &fakeWorker{}, Verifier: &fakeVerifier{}, Meter: NewBudgetMeter(Budget{}), Resume: NewResumeHandler(t.TempDir())})
	if _, err := o.Run(context.Background(), nil, nil); err == nil {
		t.Fatal("expected error for nil plan")
	}
}

func mustJSON(v any) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}
