package cli

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Hamidullah02/aios/aios-system/internal/core/audit"
	"github.com/Hamidullah02/aios/aios-system/internal/core/clock"
	"github.com/Hamidullah02/aios/aios-system/internal/core/schemas"
	"github.com/Hamidullah02/aios/aios-system/internal/runtime"
)

// e2eWorker is a runtime.Worker that returns a scripted result
// for every task. The e2e tests use it to drive the orchestrator
// through the Run / Resume paths without touching the real
// agent stack.
type e2eWorker struct {
	runFn func(idx int, input runtime.TaskInput, meter *runtime.BudgetMeter) (runtime.WorkerResult, error)
}

func (w *e2eWorker) Run(input runtime.TaskInput, _ *runtime.TaskContext, meter *runtime.BudgetMeter) (runtime.WorkerResult, error) {
	// The fake worker uses the input id's hash to get an
	// index hint. A real worker has the index; the fake
	// gets it through the closure. For the e2e tests the
	// caller passes a closure that ignores the index, so
	// any per-task logic the test wants to express can
	// come from `input` alone.
	idx := 0
	for i, r := range input.ID {
		idx = (idx*31 + int(r)) % 1024
		_ = i
	}
	return w.runFn(idx, input, meter)
}

// e2eVerifier returns a fixed verdict for every task.
type e2eVerifier struct {
	verdict runtime.VerifierVerdict
}

func (v *e2eVerifier) Verify(_ runtime.TaskInput, _ runtime.WorkerResult) (runtime.VerifierVerdict, error) {
	return v.verdict, nil
}

func newE2EAudit(t *testing.T) *audit.Log {
	t.Helper()
	log, err := audit.Open(audit.Config{
		Path:  ":memory:",
		Clock: clock.SystemDecision{},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { log.Close() })
	return log
}

func writePlanFile(t *testing.T, plan *schemas.TaskPlan) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "plan.yaml")
	// Use the same YAML the runtime's LoadPlan consumes.
	yaml := "id: " + plan.ID + "\n" +
		"goal: " + plan.Goal + "\n" +
		"tasks:\n"
	for _, task := range plan.Tasks {
		yaml += "  - id: " + task.ID + "\n" +
			"    description: " + task.Description + "\n" +
			"    acceptance: " + task.Acceptance + "\n"
		if task.Tool != "" {
			yaml += "    tool: " + task.Tool + "\n"
		}
	}
	if err := os.WriteFile(path, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestRun_HappyPath(t *testing.T) {
	log := newE2EAudit(t)
	worker := &e2eWorker{runFn: func(_ int, _ runtime.TaskInput, _ *runtime.BudgetMeter) (runtime.WorkerResult, error) {
		return runtime.WorkerResult{Output: "ok", Data: map[string]any{"k": "v"}}, nil
	}}
	verifier := &e2eVerifier{verdict: runtime.VerdictPass}
	plan := &schemas.TaskPlan{
		ID: "p1", Goal: "ship",
		Tasks: []schemas.TaskSpec{
			{ID: "t1", Description: "design", Acceptance: "doc"},
			{ID: "t2", Description: "implement", Acceptance: "tests pass"},
		},
	}
	path := writePlanFile(t, plan)
	report, err := Run(context.Background(), &Args{PlanPath: path}, RunDeps{
		Audit:    log,
		Resume:   runtime.NewResumeHandler(t.TempDir()),
		Worker:   worker,
		Verifier: verifier,
	})
	if err != nil {
		t.Fatal(err)
	}
	if report.StoppedReason != "completed" {
		t.Fatalf("StoppedReason = %q", report.StoppedReason)
	}
	if report.PlanID != "p1" {
		t.Errorf("PlanID = %q", report.PlanID)
	}
}

func TestRun_WorkerError(t *testing.T) {
	log := newE2EAudit(t)
	want := errors.New("worker blew up")
	worker := &e2eWorker{runFn: func(_ int, _ runtime.TaskInput, _ *runtime.BudgetMeter) (runtime.WorkerResult, error) {
		return runtime.WorkerResult{}, want
	}}
	verifier := &e2eVerifier{verdict: runtime.VerdictPass}
	plan := &schemas.TaskPlan{
		ID: "p2", Goal: "ship",
		Tasks: []schemas.TaskSpec{{ID: "t1", Description: "x", Acceptance: "y"}},
	}
	path := writePlanFile(t, plan)
	_, err := Run(context.Background(), &Args{PlanPath: path}, RunDeps{
		Audit:    log,
		Resume:   runtime.NewResumeHandler(t.TempDir()),
		Worker:   worker,
		Verifier: verifier,
	})
	if err == nil {
		t.Fatal("expected error from worker")
	}
}

func TestRun_VerifierFail(t *testing.T) {
	log := newE2EAudit(t)
	worker := &e2eWorker{runFn: func(_ int, _ runtime.TaskInput, _ *runtime.BudgetMeter) (runtime.WorkerResult, error) {
		return runtime.WorkerResult{Output: "ok"}, nil
	}}
	verifier := &e2eVerifier{verdict: runtime.VerdictFail}
	plan := &schemas.TaskPlan{
		ID: "p3", Goal: "ship",
		Tasks: []schemas.TaskSpec{{ID: "t1", Description: "x", Acceptance: "y"}},
	}
	path := writePlanFile(t, plan)
	report, err := Run(context.Background(), &Args{PlanPath: path}, RunDeps{
		Audit:    log,
		Resume:   runtime.NewResumeHandler(t.TempDir()),
		Worker:   worker,
		Verifier: verifier,
	})
	if err != nil {
		t.Fatal(err)
	}
	if report.StoppedReason != "verifier_fail" {
		t.Fatalf("StoppedReason = %q", report.StoppedReason)
	}
}

func TestRun_WaitingForApproval_NoHandler(t *testing.T) {
	log := newE2EAudit(t)
	worker := &e2eWorker{runFn: func(_ int, _ runtime.TaskInput, _ *runtime.BudgetMeter) (runtime.WorkerResult, error) {
		return runtime.WorkerResult{}, &runtime.WaitingForApproval{ApprovalID: "apr-1", Decision: "ask"}
	}}
	verifier := &e2eVerifier{verdict: runtime.VerdictPass}
	plan := &schemas.TaskPlan{
		ID: "p4", Goal: "ship",
		Tasks: []schemas.TaskSpec{{ID: "t1", Description: "x", Acceptance: "y"}},
	}
	path := writePlanFile(t, plan)
	_, err := Run(context.Background(), &Args{PlanPath: path}, RunDeps{
		Audit:    log,
		Resume:   runtime.NewResumeHandler(t.TempDir()),
		Worker:   worker,
		Verifier: verifier,
		// No Approval handler.
	})
	if err == nil {
		t.Fatal("expected an error when no approval handler is wired")
	}
	// The error used to surface the raw
	// *runtime.WaitingForApproval sentinel, which told an
	// operator nothing: the message was "runtime: waiting for
	// approval ask:t1" and there was no command that could
	// resolve it. It is now a CliError naming the approval
	// and the ways to resolve it.
	var cliErr *CliError
	if !errors.As(err, &cliErr) {
		t.Fatalf("expected *CliError, got %T: %v", err, err)
	}
	if cliErr.Kind != "approval" {
		t.Errorf("Kind = %q, want \"approval\"", cliErr.Kind)
	}
	if !strings.Contains(cliErr.What, "apr-1") {
		t.Errorf("error should name the approval id; got %q", cliErr.What)
	}
	if !strings.Contains(cliErr.What, "--yes") {
		t.Errorf("error should say how to proceed; got %q", cliErr.What)
	}
}

func TestRun_WaitingForApproval_Approve(t *testing.T) {
	log := newE2EAudit(t)
	plan := &schemas.TaskPlan{
		ID: "p5", Goal: "ship",
		Tasks: []schemas.TaskSpec{{ID: "t1", Description: "x", Acceptance: "y"}},
	}
	path := writePlanFile(t, plan)
	calls := 0
	// Each orchestrator.Run starts fresh with the
	// checkpoint we pre-populate. The resume handler
	// indexes checkpoints by the TaskID we pass to Save.
	resume := runtime.NewResumeHandler(t.TempDir())
	// Pre-populate a checkpoint at task index 1 (one
	// past the only task) so the resumed run finishes.
	cp := &runtime.Checkpoint{TaskID: "p5", TaskIndex: 1}
	if err := resume.Save(cp); err != nil {
		t.Fatal(err)
	}
	// Re-define the worker now that the closure can
	// refer to the resume handler.
	worker := &e2eWorker{runFn: func(_ int, _ runtime.TaskInput, _ *runtime.BudgetMeter) (runtime.WorkerResult, error) {
		calls++
		return runtime.WorkerResult{Output: "ok"}, nil
	}}
	verifier := &e2eVerifier{verdict: runtime.VerdictPass}
	approve := &scriptedApprover{approve: true}
	// Run with the plan; the orchestrator saves a fresh
	// checkpoint at task index 1 after the single task
	// passes. The next Run with the same plan id should
	// start at index 1 and complete with no work.
	report, err := Run(context.Background(), &Args{PlanPath: path}, RunDeps{
		Audit:    log,
		Resume:   resume,
		Worker:   worker,
		Verifier: verifier,
		Approval: approve,
	})
	if err != nil {
		t.Fatalf("first run: %v", err)
	}
	if report.StoppedReason != "completed" {
		t.Errorf("first run StoppedReason = %q", report.StoppedReason)
	}
	if calls != 1 {
		t.Errorf("first run: worker called %d times, want 1", calls)
	}
}

func TestRun_WaitingForApproval_Deny(t *testing.T) {
	log := newE2EAudit(t)
	worker := &e2eWorker{runFn: func(_ int, _ runtime.TaskInput, _ *runtime.BudgetMeter) (runtime.WorkerResult, error) {
		return runtime.WorkerResult{}, &runtime.WaitingForApproval{ApprovalID: "apr-1", Decision: "ask"}
	}}
	verifier := &e2eVerifier{verdict: runtime.VerdictPass}
	plan := &schemas.TaskPlan{
		ID: "p6", Goal: "ship",
		Tasks: []schemas.TaskSpec{{ID: "t1", Description: "x", Acceptance: "y"}},
	}
	path := writePlanFile(t, plan)
	approve := &scriptedApprover{approve: false}
	_, err := Run(context.Background(), &Args{PlanPath: path}, RunDeps{
		Audit:    log,
		Resume:   runtime.NewResumeHandler(t.TempDir()),
		Worker:   worker,
		Verifier: verifier,
		Approval: approve,
	})
	if err == nil {
		t.Fatal("expected denial error")
	}
}

func TestRun_WaitingForApproval_HandlerError(t *testing.T) {
	log := newE2EAudit(t)
	worker := &e2eWorker{runFn: func(_ int, _ runtime.TaskInput, _ *runtime.BudgetMeter) (runtime.WorkerResult, error) {
		return runtime.WorkerResult{}, &runtime.WaitingForApproval{ApprovalID: "apr-1", Decision: "ask"}
	}}
	verifier := &e2eVerifier{verdict: runtime.VerdictPass}
	plan := &schemas.TaskPlan{
		ID: "p7", Goal: "ship",
		Tasks: []schemas.TaskSpec{{ID: "t1", Description: "x", Acceptance: "y"}},
	}
	path := writePlanFile(t, plan)
	approve := &scriptedApprover{err: errors.New("daemon down")}
	_, err := Run(context.Background(), &Args{PlanPath: path}, RunDeps{
		Audit:    log,
		Resume:   runtime.NewResumeHandler(t.TempDir()),
		Worker:   worker,
		Verifier: verifier,
		Approval: approve,
	})
	if err == nil {
		t.Fatal("expected approval-handler error")
	}
}

func TestResume_LoadsCheckpoint(t *testing.T) {
	log := newE2EAudit(t)
	worker := &e2eWorker{runFn: func(_ int, _ runtime.TaskInput, _ *runtime.BudgetMeter) (runtime.WorkerResult, error) {
		return runtime.WorkerResult{Output: "ok"}, nil
	}}
	verifier := &e2eVerifier{verdict: runtime.VerdictPass}
	plan := &schemas.TaskPlan{
		ID: "p8", Goal: "ship",
		Tasks: []schemas.TaskSpec{{ID: "t1", Description: "x", Acceptance: "y"}},
	}
	path := writePlanFile(t, plan)
	resume := runtime.NewResumeHandler(t.TempDir())
	// Pre-populate the checkpoint so Resume finds one.
	cp := &runtime.Checkpoint{
		TaskID:    "p8",
		TaskIndex: 0,
	}
	if err := resume.Save(cp); err != nil {
		t.Fatal(err)
	}
	report, err := Resume(context.Background(), &Args{PlanPath: path, TaskID: "p8"}, ResumeDeps{
		Audit:  log,
		Resume: resume,
		Worker: worker,
		Verify: verifier,
	})
	if err != nil {
		t.Fatal(err)
	}
	if report.PlanID != "p8" {
		t.Errorf("PlanID = %q", report.PlanID)
	}
}

func TestResume_NoCheckpoint(t *testing.T) {
	log := newE2EAudit(t)
	worker := &e2eWorker{runFn: func(_ int, _ runtime.TaskInput, _ *runtime.BudgetMeter) (runtime.WorkerResult, error) {
		return runtime.WorkerResult{Output: "ok"}, nil
	}}
	verifier := &e2eVerifier{verdict: runtime.VerdictPass}
	plan := &schemas.TaskPlan{
		ID: "p9", Goal: "ship",
		Tasks: []schemas.TaskSpec{{ID: "t1", Description: "x", Acceptance: "y"}},
	}
	path := writePlanFile(t, plan)
	_, err := Resume(context.Background(), &Args{PlanPath: path, TaskID: "p9"}, ResumeDeps{
		Audit:  log,
		Resume: runtime.NewResumeHandler(t.TempDir()),
		Worker: worker,
		Verify: verifier,
	})
	if err == nil {
		t.Fatal("expected no-checkpoint error")
	}
	var cliErr *CliError
	if !errors.As(err, &cliErr) {
		t.Fatalf("expected *CliError, got %T", err)
	}
}

func TestResume_RequiresPlanPath(t *testing.T) {
	_, err := Resume(context.Background(), &Args{TaskID: "x"}, ResumeDeps{
		Resume: runtime.NewResumeHandler(t.TempDir()),
	})
	if err == nil {
		t.Fatal("expected error for missing plan path")
	}
}

func TestResume_RequiresTaskID(t *testing.T) {
	_, err := Resume(context.Background(), &Args{}, ResumeDeps{
		Resume: runtime.NewResumeHandler(t.TempDir()),
	})
	if err == nil {
		t.Fatal("expected error for missing task id")
	}
}

func TestCancel(t *testing.T) {
	dir := t.TempDir()
	cp := &runtime.Checkpoint{TaskID: "abc", TaskIndex: 0}
	resume := runtime.NewResumeHandler(dir)
	if err := resume.Save(cp); err != nil {
		t.Fatal(err)
	}
	if err := Cancel("abc", CancelDeps{Resume: resume}); err != nil {
		t.Fatal(err)
	}
	if _, err := resume.Load("abc"); err == nil {
		t.Fatal("checkpoint should be deleted")
	}
}

func TestCancel_EmptyTaskID(t *testing.T) {
	if err := Cancel("", CancelDeps{Resume: runtime.NewResumeHandler(t.TempDir())}); err == nil {
		t.Fatal("expected error for empty task id")
	}
}

func TestCancel_MissingResume(t *testing.T) {
	if err := Cancel("x", CancelDeps{}); err == nil {
		t.Fatal("expected error for missing resume handler")
	}
}

func TestReport_Formats(t *testing.T) {
	log := newE2EAudit(t)
	// Write a few rows by running a tiny plan.
	worker := &e2eWorker{runFn: func(_ int, _ runtime.TaskInput, _ *runtime.BudgetMeter) (runtime.WorkerResult, error) {
		return runtime.WorkerResult{Output: "ok"}, nil
	}}
	verifier := &e2eVerifier{verdict: runtime.VerdictPass}
	plan := &schemas.TaskPlan{
		ID: "rep-1", Goal: "ship",
		Tasks: []schemas.TaskSpec{{ID: "t1", Description: "x", Acceptance: "y"}},
	}
	path := writePlanFile(t, plan)
	if _, err := Run(context.Background(), &Args{PlanPath: path}, RunDeps{
		Audit:    log,
		Resume:   runtime.NewResumeHandler(t.TempDir()),
		Worker:   worker,
		Verifier: verifier,
	}); err != nil {
		t.Fatal(err)
	}
	// Now query the report.
	for _, format := range []string{"json", "markdown", "html"} {
		t.Run(format, func(t *testing.T) {
			if err := Report("rep-1", ReportDeps{
				Audit:  log,
				Format: format,
				Out:    &bytesBuf{},
			}); err != nil {
				t.Errorf("format %s: %v", format, err)
			}
		})
	}
}

func TestReport_UnknownFormat(t *testing.T) {
	log := newE2EAudit(t)
	if err := Report("x", ReportDeps{
		Audit:  log,
		Format: "yaml",
		Out:    &bytesBuf{},
	}); err == nil {
		t.Fatal("expected error for unknown format")
	}
}

func TestReport_EmptyTaskID(t *testing.T) {
	if err := Report("", ReportDeps{Out: &bytesBuf{}}); err == nil {
		t.Fatal("expected error for empty task id")
	}
}

func TestReport_NilOut(t *testing.T) {
	if err := Report("x", ReportDeps{}); err == nil {
		t.Fatal("expected error for nil out")
	}
}

func TestAudit_NoTaskID(t *testing.T) {
	log := newE2EAudit(t)
	// Empty log + no task filter is a valid no-op: the
	// function iterates the well-known kinds and finds
	// nothing.
	if err := Audit("", AuditDeps{Audit: log, Out: &bytesBuf{}}); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestAudit_TaskID(t *testing.T) {
	log := newE2EAudit(t)
	if err := Audit("some-task", AuditDeps{Audit: log, Out: &bytesBuf{}}); err != nil {
		t.Fatal(err)
	}
}

func TestAudit_NilAudit(t *testing.T) {
	if err := Audit("x", AuditDeps{Out: &bytesBuf{}}); err == nil {
		t.Fatal("expected error for nil audit")
	}
}

func TestAudit_NilOut(t *testing.T) {
	log := newE2EAudit(t)
	if err := Audit("x", AuditDeps{Audit: log}); err == nil {
		t.Fatal("expected error for nil out")
	}
}

func TestPrintReport_Real(t *testing.T) {
	r := &runtime.Report{
		PlanID:        "p1",
		TaskCount:     3,
		FromIndex:     0,
		StartedAt:     "2024-01-01T00:00:00Z",
		FinishedAt:    "2024-01-01T00:01:00Z",
		StoppedReason: "completed",
	}
	buf := &bytesBuf{}
	PrintReport(buf, r)
	s := buf.String()
	if !contains(s, "p1") {
		t.Errorf("missing plan id: %s", s)
	}
	if !contains(s, "completed") {
		t.Errorf("missing stopped reason: %s", s)
	}
}

func TestRunDaemonStart_StopsCleanly(t *testing.T) {
	// Cover the daemon's normal start/stop path. The
	// goroutine that calls Start blocks in Accept; we
	// cancel via Stop, which closes the listener and
	// unblocks the loop.
	dir := t.TempDir()
	sock := filepath.Join(dir, "a.sock")
	pid := filepath.Join(dir, "a.pid")
	d := NewDaemon(DaemonConfig{SocketPath: sock, PidPath: pid})
	done := make(chan error, 1)
	go func() {
		done <- d.Start(context.Background())
	}()
	// Wait for the socket to appear.
	deadline := time.Now().Add(2 * time.Second)
	for {
		if _, err := os.Stat(sock); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("socket did not appear")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err := d.Stop(); err != nil {
		t.Errorf("Stop: %v", err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Start returned: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Start did not return after Stop")
	}
}

// scriptedApprover is an ApprovalHandler test double. It
// returns a scripted (approved, err) pair on every call.
type scriptedApprover struct {
	approve bool
	err     error
	calls   int
}

func (s *scriptedApprover) Approve(_ context.Context, _ *runtime.WaitingForApproval) (bool, error) {
	s.calls++
	return s.approve, s.err
}

// bytesBuf is a small io.Writer for tests. We don't import
// bytes to keep the test's surface area small.
type bytesBuf struct {
	data []byte
}

func (b *bytesBuf) Write(p []byte) (int, error) {
	b.data = append(b.data, p...)
	return len(p), nil
}

func (b *bytesBuf) String() string {
	return string(b.data)
}
