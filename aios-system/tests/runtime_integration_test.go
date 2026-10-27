// Package tests holds black-box integration tests for aios
// modules.
package tests

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Hamidullah02/aios/aios-system/internal/core/audit"
	"github.com/Hamidullah02/aios/aios-system/internal/core/clock"
	"github.com/Hamidullah02/aios/aios-system/internal/core/schemas"
	"github.com/Hamidullah02/aios/aios-system/internal/runtime"
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
	//
	// The chain is global and links every row to its
	// predecessor regardless of kind or task, so it can only
	// be verified over the COMPLETE log. v0.41.0 added
	// task-scoped task_started / task_passed rows alongside
	// the plan-scoped ones, which makes a per-task slice a
	// non-contiguous view of the chain: row 2's prev_hash
	// points at a task-scoped row this query does not return.
	rows := allAuditRows(t, log)
	if len(rows) < 3 {
		t.Fatalf("audit rows = %d, want >= 3", len(rows))
	}
	if v := audit.VerifyChain(rows); !v.OK {
		t.Fatalf("chain invalid: %s", v.Reason)
	}
}

// allAuditRows returns every row in the log in chain order.
//
// The kind list must stay complete: VerifyChain hashes each
// row against its predecessor, so omitting a kind checks a
// fiction. A newly added kind that is forgotten here fails
// this test, which is the alarm working rather than a false
// pass.
func allAuditRows(t *testing.T, log *audit.Log) []audit.PersistedRow {
	t.Helper()
	kinds := []string{
		"task_started", // v0.41.0
		"task_passed",  // v0.41.0
		"task_failed",
		"task_paused",
		"task_awaiting_approval",
		"task_cancelled",
		"policy_decision",
		"decision",
		"tool_call",
	}
	seen := make(map[int64]bool)
	var rows []audit.PersistedRow
	for _, kind := range kinds {
		got, err := log.GetByKind(kind, 1000)
		if err != nil {
			t.Fatalf("GetByKind(%q): %v", kind, err)
		}
		for _, r := range got {
			if !seen[r.SequenceNumber] {
				seen[r.SequenceNumber] = true
				rows = append(rows, r)
			}
		}
	}
	for i := 1; i < len(rows); i++ {
		for j := i; j > 0 && rows[j-1].SequenceNumber > rows[j].SequenceNumber; j-- {
			rows[j-1], rows[j] = rows[j], rows[j-1]
		}
	}
	return rows
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
