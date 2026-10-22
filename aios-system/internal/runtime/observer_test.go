package runtime

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/sadi/aios-system/internal/core/state"
)

// recordingObserver captures every event it receives.
type recordingObserver struct {
	mu     sync.Mutex
	events []TaskEvent
}

func (r *recordingObserver) add(e TaskEvent) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, e)
}

func (r *recordingObserver) TaskStarted(e TaskEvent)  { r.add(e) }
func (r *recordingObserver) TaskFinished(e TaskEvent) { r.add(e) }
func (r *recordingObserver) TaskFailed(e TaskEvent)   { r.add(e) }

func (r *recordingObserver) all() []TaskEvent {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]TaskEvent(nil), r.events...)
}

// TestObserver_SeesTaskLifecycle is the core contract: an
// observer attached to the orchestrator sees a start and a
// finish for a plan that succeeds.
func TestObserver_SeesTaskLifecycle(t *testing.T) {
	o := New(Config{
		Worker: &fakeWorker{runFn: func(int, TaskInput, *BudgetMeter) (WorkerResult, error) {
			return WorkerResult{Output: "ok", Data: map[string]any{"k": "v"}}, nil
		}},
		Verifier: &fakeVerifier{verdict: VerdictPass},
		Meter:    NewBudgetMeter(Budget{}),
		Audit:    newTestAuditLog(t),
		Resume:   NewResumeHandler(t.TempDir()),
	})
	rec := &recordingObserver{}
	o.AddObserver(rec)

	if _, err := o.Run(context.Background(), newTestPlan(), nil); err != nil {
		t.Fatalf("Run: %v", err)
	}

	// The shared test plan has two tasks, so a successful run
	// reports two start/finish pairs. Asserting the pairs
	// rather than a total count keeps the test honest if the
	// fixture plan changes.
	ev := rec.all()
	if len(ev) != 4 {
		t.Fatalf("events = %d, want 4 (2 tasks x start+finish): %+v", len(ev), ev)
	}
	if ev[0].State != state.StateRunning {
		t.Errorf("first state = %q, want running", ev[0].State)
	}
	if ev[0].TaskID == "" {
		t.Error("first event has no task id")
	}
	if ev[0].PlanID == "" {
		t.Error("first event has no plan id")
	}
	if ev[0].At.IsZero() {
		t.Error("event has no timestamp; a UI could not order it")
	}
	if ev[1].State != state.StateCompleted {
		t.Errorf("second state = %q, want completed", ev[1].State)
	}
	// Every task must report its own start and finish.
	for i := 0; i+1 < len(ev); i += 2 {
		if ev[i].TaskID != ev[i+1].TaskID {
			t.Errorf("events %d/%d are for different tasks (%q, %q)",
				i, i+1, ev[i].TaskID, ev[i+1].TaskID)
		}
	}
}

// TestObserver_WorkerErrorIsReported covers the failure
// path, including the detail the UI shows.
func TestObserver_WorkerErrorIsReported(t *testing.T) {
	o := New(Config{
		Worker: &fakeWorker{runFn: func(int, TaskInput, *BudgetMeter) (WorkerResult, error) {
			return WorkerResult{}, errors.New("boom")
		}},
		Verifier: &fakeVerifier{verdict: VerdictPass},
		Meter:    NewBudgetMeter(Budget{}),
		Audit:    newTestAuditLog(t),
		Resume:   NewResumeHandler(t.TempDir()),
	})
	rec := &recordingObserver{}
	o.AddObserver(rec)

	if _, err := o.Run(context.Background(), newTestPlan(), nil); err == nil {
		t.Fatal("expected the worker error to propagate")
	}

	// The worker fails on the first task, so the run stops
	// there; look for a failure event rather than a fixed
	// position.
	ev := rec.all()
	var fail *TaskEvent
	for i := range ev {
		if ev[i].State == state.StateFailed {
			fail = &ev[i]
		}
	}
	if fail == nil {
		t.Fatalf("no failed event in %+v", ev)
	}
	if fail.Err == nil {
		t.Error("failure event carries no error")
	}
	if fail.Detail == "" {
		t.Error("failure event carries no detail for the UI")
	}
}

// TestObserver_AwaitingApprovalIsReported covers the state
// the UI renders as "blocked": the operator's next action is
// to grant an approval, not to investigate a fault.
func TestObserver_AwaitingApprovalIsReported(t *testing.T) {
	o := New(Config{
		Worker: &fakeWorker{runFn: func(int, TaskInput, *BudgetMeter) (WorkerResult, error) {
			return WorkerResult{}, &WaitingForApproval{ApprovalID: "apr-1", Decision: "ask"}
		}},
		Verifier: &fakeVerifier{verdict: VerdictPass},
		Meter:    NewBudgetMeter(Budget{}),
		Audit:    newTestAuditLog(t),
		Resume:   NewResumeHandler(t.TempDir()),
	})
	rec := &recordingObserver{}
	o.AddObserver(rec)

	if _, err := o.Run(context.Background(), newTestPlan(), nil); err == nil {
		t.Fatal("expected the approval wait to propagate")
	}

	ev := rec.all()
	found := false
	for i := range ev {
		if ev[i].State == state.StateAwaitingApproval {
			found = true
			if ev[i].Detail != "waiting_for_approval" {
				t.Errorf("detail = %q, want waiting_for_approval", ev[i].Detail)
			}
		}
	}
	if !found {
		t.Errorf("no awaiting-approval event in %+v", ev)
	}
}

// TestObserver_BudgetExhaustedIsReported covers the pause
// path, the third reason a task can leave the running
// state.
func TestObserver_BudgetExhaustedIsReported(t *testing.T) {
	o := New(Config{
		Worker: &fakeWorker{runFn: func(_ int, _ TaskInput, m *BudgetMeter) (WorkerResult, error) {
			m.RecordToolCall()
			return WorkerResult{Output: "ok"}, nil
		}},
		Verifier: &fakeVerifier{verdict: VerdictPass},
		Meter:    NewBudgetMeter(Budget{MaxToolCalls: 1}),
		Audit:    newTestAuditLog(t),
		Resume:   NewResumeHandler(t.TempDir()),
	})
	rec := &recordingObserver{}
	o.AddObserver(rec)

	if _, err := o.Run(context.Background(), newTestPlan(), nil); err == nil {
		t.Fatal("expected budget exhaustion to propagate")
	}

	found := false
	for _, e := range rec.all() {
		if e.State == state.StatePaused {
			found = true
			if e.Detail != "budget_exhausted" {
				t.Errorf("detail = %q, want budget_exhausted", e.Detail)
			}
		}
	}
	if !found {
		t.Error("no paused event was reported")
	}
}

// TestObserver_NoObserverIsANoOp asserts the common case --
// nobody watching -- behaves exactly as before.
func TestObserver_NoObserverIsANoOp(t *testing.T) {
	o := New(Config{
		Worker: &fakeWorker{runFn: func(int, TaskInput, *BudgetMeter) (WorkerResult, error) {
			return WorkerResult{Output: "ok"}, nil
		}},
		Verifier: &fakeVerifier{verdict: VerdictPass},
		Meter:    NewBudgetMeter(Budget{}),
		Audit:    newTestAuditLog(t),
		Resume:   NewResumeHandler(t.TempDir()),
	})
	report, err := o.Run(context.Background(), newTestPlan(), nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if report.StoppedReason != "completed" {
		t.Errorf("StoppedReason = %q, want completed", report.StoppedReason)
	}
}

// TestObserver_ConcurrentRegistrationAndNotify is the race
// gate: an observer registered while a plan runs must not
// corrupt the set the run loop is reading. Meaningful under
// -race; without it, still catches a panic from a torn map.
func TestObserver_ConcurrentRegistrationAndNotify(t *testing.T) {
	o := New(Config{
		Worker: &fakeWorker{runFn: func(int, TaskInput, *BudgetMeter) (WorkerResult, error) {
			return WorkerResult{Output: "ok"}, nil
		}},
		Verifier: &fakeVerifier{verdict: VerdictPass},
		Meter:    NewBudgetMeter(Budget{}),
		Audit:    newTestAuditLog(t),
		Resume:   NewResumeHandler(t.TempDir()),
	})
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			o.AddObserver(&recordingObserver{})
		}()
	}
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			o.notify(Observer.TaskStarted, TaskEvent{TaskID: "t1"})
		}()
	}
	wg.Wait()
}

// TestObserver_NilObserverIgnored asserts a nil registration
// is a no-op rather than a panic on the next task.
func TestObserver_NilObserverIgnored(t *testing.T) {
	o := New(Config{
		Worker: &fakeWorker{runFn: func(int, TaskInput, *BudgetMeter) (WorkerResult, error) {
			return WorkerResult{Output: "ok"}, nil
		}},
		Verifier: &fakeVerifier{verdict: VerdictPass},
		Meter:    NewBudgetMeter(Budget{}),
		Audit:    newTestAuditLog(t),
		Resume:   NewResumeHandler(t.TempDir()),
	})
	o.AddObserver(nil)
	if _, err := o.Run(context.Background(), newTestPlan(), nil); err != nil {
		t.Fatalf("Run: %v", err)
	}
}

// TestObserverFuncs_SkipsNilFields covers the function
// adapter: a nil field must not be called.
func TestObserverFuncs_SkipsNilFields(t *testing.T) {
	calls := 0
	var f Observer = ObserverFuncs{
		Finished: func(TaskEvent) { calls++ },
	}
	f.TaskStarted(TaskEvent{})
	if calls != 0 {
		t.Error("a nil Started callback was invoked")
	}
	f.TaskFinished(TaskEvent{})
	if calls != 1 {
		t.Errorf("Finished called %d times, want 1", calls)
	}
}
