// Package tests holds black-box integration tests for aios
// modules.
package tests

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sadi/aios-system/internal/core/audit"
	"github.com/sadi/aios-system/internal/core/clock"
	"github.com/sadi/aios-system/internal/core/schemas"
	"github.com/sadi/aios-system/internal/runtime"
)

type recordingWorker struct {
	calls int
}

func (r *recordingWorker) Run(_ runtime.TaskInput, _ *runtime.TaskContext, _ *runtime.BudgetMeter) (runtime.WorkerResult, error) {
	r.calls++
	return runtime.WorkerResult{Output: "ok", Data: map[string]any{"step": r.calls}}, nil
}

type passVerifier struct{}

func (passVerifier) Verify(_ runtime.TaskInput, _ runtime.WorkerResult) (runtime.VerifierVerdict, error) {
	return runtime.VerdictPass, nil
}

func TestRuntime_Orchestrator_EndToEnd(t *testing.T) {
	log, err := audit.Open(audit.Config{
		Path:  ":memory:",
		Clock: clock.FixedDecision{T: time.Unix(1700000000, 0)},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { log.Close() })

	w := &recordingWorker{}
	o := runtime.New(runtime.Config{
		Worker:   w,
		Verifier: passVerifier{},
		Meter:    runtime.NewBudgetMeter(runtime.Budget{}),
		Audit:    log,
		Resume:   runtime.NewResumeHandler(t.TempDir()),
	})

	plan := &schemas.TaskPlan{
		ID: "p1", Goal: "ship",
		Tasks: []schemas.TaskSpec{
			{ID: "st1", Description: "design", Acceptance: "doc"},
			{ID: "st2", Description: "implement", Acceptance: "tests"},
			{ID: "st3", Description: "verify", Acceptance: "all green"},
		},
	}
	report, err := o.Run(context.Background(), plan, nil)
	if err != nil {
		t.Fatal(err)
	}
	if report.StoppedReason != "completed" {
		t.Fatalf("StoppedReason = %q", report.StoppedReason)
	}
	if w.calls != 3 {
		t.Fatalf("calls = %d, want 3", w.calls)
	}

	// Verify the audit chain.
	rows, _ := log.GetByTask("p1")
	if len(rows) < 3 {
		t.Fatalf("audit rows = %d, want >= 3", len(rows))
	}
	if v := audit.VerifyChain(rows); !v.OK {
		t.Fatalf("chain invalid: %s", v.Reason)
	}
}

type approvalWorker struct{}

func (approvalWorker) Run(_ runtime.TaskInput, _ *runtime.TaskContext, _ *runtime.BudgetMeter) (runtime.WorkerResult, error) {
	return runtime.WorkerResult{}, &runtime.WaitingForApproval{ApprovalID: "apr-1", Decision: "ask"}
}

func TestRuntime_Orchestrator_WaitingForApproval(t *testing.T) {
	log, err := audit.Open(audit.Config{
		Path:  ":memory:",
		Clock: clock.FixedDecision{T: time.Unix(1700000000, 0)},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { log.Close() })

	o := runtime.New(runtime.Config{
		Worker:   approvalWorker{},
		Verifier: passVerifier{},
		Meter:    runtime.NewBudgetMeter(runtime.Budget{}),
		Audit:    log,
		Resume:   runtime.NewResumeHandler(t.TempDir()),
	})
	plan := &schemas.TaskPlan{
		ID: "p1", Goal: "ship",
		Tasks: []schemas.TaskSpec{
			{ID: "st1", Description: "x", Acceptance: "y"},
		},
	}
	_, err = o.Run(context.Background(), plan, nil)
	var wfa *runtime.WaitingForApproval
	if !errors.As(err, &wfa) {
		t.Fatalf("err type = %T, want *WaitingForApproval", err)
	}
}
