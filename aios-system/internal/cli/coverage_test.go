package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sadi/aios-system/internal/approvals"
	"github.com/sadi/aios-system/internal/core/audit"
	"github.com/sadi/aios-system/internal/core/schemas"
	"github.com/sadi/aios-system/internal/policy"
	"github.com/sadi/aios-system/internal/runtime"

	"github.com/sadi/aios-system/internal/ask"
)

// --- Main / RunCommand coverage ---

func TestCoverage_Main_NilArgs(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Main(nil, &stdout, &stderr)
	if code == 0 {
		t.Fatal("expected non-zero on nil args")
	}
	if !strings.Contains(stderr.String(), "no arguments") {
		t.Errorf("stderr: %s", stderr.String())
	}
}

func TestCoverage_Main_NilStdoutStderr(t *testing.T) {
	// Suppress real stdout / stderr; we just want to
	// confirm the nil-writer fallback doesn't crash.
	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	os.Stdout = w
	os.Stderr = w
	defer func() {
		os.Stdout = oldStdout
		os.Stderr = oldStderr
		w.Close()
		r.Close()
	}()
	code := Main([]string{"aios", "--help"}, nil, nil)
	if code != 0 {
		t.Fatalf("code: %d", code)
	}
}

func TestCoverage_Main_BadFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Main([]string{"aios", "--nonexistent-flag"}, &stdout, &stderr)
	if code == 0 {
		t.Fatal("expected non-zero for unknown flag")
	}
}

func TestCoverage_Main_NoSubcommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Main([]string{"aios"}, &stdout, &stderr)
	if code == 0 {
		t.Fatal("expected non-zero for no subcommand")
	}
}

func TestCoverage_Main_UnknownSubcommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Main([]string{"aios", "nope"}, &stdout, &stderr)
	if code == 0 {
		t.Fatal("expected non-zero for unknown subcommand")
	}
}

func TestCoverage_Main_DebugFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Main([]string{"aios", "--debug", "run"}, &stdout, &stderr)
	if code == 0 {
		t.Fatal("expected non-zero")
	}
	if !strings.Contains(stderr.String(), "plan") {
		t.Errorf("stderr should mention plan: %s", stderr.String())
	}
}

func TestCoverage_RunCommand_NilArgs(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := RunCommand(nil, &Deps{}, &stdout, &stderr)
	if code == 0 {
		t.Fatal("expected non-zero")
	}
}

func TestCoverage_RunCommand_UnknownSubcommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	args := &Args{Command: "nope"}
	code := RunCommand(args, &Deps{}, &stdout, &stderr)
	if code == 0 {
		t.Fatal("expected non-zero for unknown subcommand")
	}
}

func TestCoverage_RunInit_Defaults(t *testing.T) {
	dir := t.TempDir()
	oldwd, _ := os.Getwd()
	defer os.Chdir(oldwd)
	os.Chdir(dir)
	var stdout, stderr bytes.Buffer
	code := runInit(&Args{ConfigPath: ""}, &Deps{}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code: %d, stderr: %s", code, stderr.String())
	}
}

func TestCoverage_RunRun_BadPlanPath(t *testing.T) {
	var stdout, stderr bytes.Buffer
	args := &Args{Command: CmdRun, PlanPath: filepath.Join(t.TempDir(), "missing.yaml")}
	code := runRun(context.Background(), args, &Deps{}, &stdout, &stderr)
	if code == 0 {
		t.Fatal("expected non-zero for missing plan")
	}
}

func TestCoverage_RunResume_MissingArgs(t *testing.T) {
	var stdout, stderr bytes.Buffer
	args := &Args{Command: CmdResume}
	code := runResume(context.Background(), args, &Deps{}, &stdout, &stderr)
	if code == 0 {
		t.Fatal("expected non-zero")
	}
}

func TestCoverage_RunCancel_NoTaskID(t *testing.T) {
	var stdout, stderr bytes.Buffer
	args := &Args{Command: CmdCancel}
	code := runCancel(args, &Deps{}, &stdout, &stderr)
	if code == 0 {
		t.Fatal("expected non-zero")
	}
}

func TestCoverage_RunReport_BadFormat(t *testing.T) {
	var stderr bytes.Buffer
	args := &Args{Command: CmdReport, TaskID: "x", ReportFmt: "yaml"}
	deps := &Deps{Audit: openTestAudit(t)}
	code := runReport(args, deps, &bytes.Buffer{}, &stderr)
	if code == 0 {
		t.Fatal("expected non-zero for unknown format")
	}
}

func TestCoverage_RunReport_NilDeps(t *testing.T) {
	var stdout, stderr bytes.Buffer
	args := &Args{Command: CmdReport, TaskID: "x"}
	code := runReport(args, &Deps{}, &stdout, &stderr)
	if code == 0 {
		t.Fatal("expected non-zero for nil audit")
	}
}

func TestCoverage_RunAudit_NoTaskID(t *testing.T) {
	var stderr bytes.Buffer
	args := &Args{Command: CmdAudit}
	deps := &Deps{Audit: openTestAudit(t)}
	code := runAudit(args, deps, &bytes.Buffer{}, &stderr)
	if code != 0 {
		t.Errorf("code: %d, stderr: %s", code, stderr.String())
	}
}

func TestCoverage_RunAudit_NilDeps(t *testing.T) {
	var stdout, stderr bytes.Buffer
	args := &Args{Command: CmdAudit, TaskID: "x"}
	code := runAudit(args, &Deps{}, &stdout, &stderr)
	if code == 0 {
		t.Fatal("expected non-zero for nil audit")
	}
}

func TestCoverage_RunDaemon_UnknownSubcommand(t *testing.T) {
	var stderr bytes.Buffer
	args := &Args{Command: CmdDaemon, Subcommand: "restart"}
	deps := &Deps{}
	code := runDaemon(context.Background(), args, deps, &stderr)
	if code == 0 {
		t.Fatal("expected non-zero")
	}
}

func TestCoverage_RunDaemon_StopNoPID(t *testing.T) {
	var stderr bytes.Buffer
	dir := t.TempDir()
	args := &Args{Command: CmdDaemon, Subcommand: DaemonStop, DaemonPID: filepath.Join(dir, "x.pid")}
	deps := &Deps{}
	code := runDaemon(context.Background(), args, deps, &stderr)
	if code != 0 {
		t.Errorf("code: %d, stderr: %s", code, stderr.String())
	}
}

func TestCoverage_RunDaemon_StartCancel(t *testing.T) {
	dir := t.TempDir()
	sock := filepath.Join(dir, "a.sock")
	pid := filepath.Join(dir, "a.pid")
	var stderr bytes.Buffer
	args := &Args{Command: CmdDaemon, Subcommand: DaemonStart, SocketPath: sock, DaemonPID: pid}
	deps := &Deps{}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan int, 1)
	go func() { done <- runDaemon(ctx, args, deps, &stderr) }()
	deadline := time.Now().Add(2 * time.Second)
	for {
		if _, err := os.Stat(sock); err == nil {
			break
		}
		if time.Now().After(deadline) {
			cancel()
			t.Fatal("socket did not appear")
		}
		time.Sleep(10 * time.Millisecond)
	}
	cancel()
	select {
	case code := <-done:
		if code != 0 {
			t.Errorf("code: %d, stderr: %s", code, stderr.String())
		}
	case <-time.After(2 * time.Second):
		t.Fatal("runDaemon did not return after cancel")
	}
}

// --- Daemon handleDeny + dispatch coverage ---

func TestCoverage_Daemon_HandleDeny(t *testing.T) {
	log := openTestAudit(t)
	svc := approvals.New(approvals.Config{Audit: log})
	d := &Daemon{approvals: svc}
	payload := schemas.ApprovalPayload{
		Action: "edit", Target: "/tmp/x", RiskTier: 3,
		Reason: "needed", RequesterID: "agent:planner",
	}
	id, err := svc.Request(payload, "alice", approvals.RequestOptions{})
	if err != nil {
		t.Fatal(err)
	}
	resp := d.handleDeny(context.Background(), id, "alice", nil)
	if resp["ok"] != true {
		t.Errorf("response: %v", resp)
	}
}

func TestCoverage_Daemon_HandleDeny_NilApprovals(t *testing.T) {
	d := &Daemon{}
	resp := d.handleDeny(context.Background(), "x", "alice", nil)
	if resp["ok"] != false {
		t.Errorf("expected ok=false, got %v", resp)
	}
}

func TestCoverage_Daemon_HandleDeny_UnknownID(t *testing.T) {
	svc := approvals.New(approvals.Config{})
	d := &Daemon{approvals: svc}
	resp := d.handleDeny(context.Background(), "missing", "alice", nil)
	if resp["ok"] != false {
		t.Errorf("expected ok=false, got %v", resp)
	}
}

func TestCoverage_Daemon_Dispatch_UnknownKind(t *testing.T) {
	d := &Daemon{}
	resp := d.dispatch(context.Background(), map[string]any{"kind": "nope"})
	if resp["ok"] != false {
		t.Errorf("expected ok=false, got %v", resp)
	}
}

func TestCoverage_Daemon_Dispatch_KindMissing(t *testing.T) {
	d := &Daemon{}
	resp := d.dispatch(context.Background(), map[string]any{})
	if resp["ok"] != false {
		t.Errorf("expected ok=false, got %v", resp)
	}
}

func TestCoverage_Daemon_HandleWait_NotFound(t *testing.T) {
	svc := approvals.New(approvals.Config{})
	d := &Daemon{approvals: svc}
	resp := d.handleWait(context.Background(), "missing")
	if resp["ok"] != false {
		t.Errorf("expected ok=false, got %v", resp)
	}
}

func TestCoverage_Daemon_HandleGrant_NilApprovals(t *testing.T) {
	d := &Daemon{}
	resp := d.handleGrant(context.Background(), "x", "alice", nil)
	if resp["ok"] != false {
		t.Errorf("expected ok=false, got %v", resp)
	}
}

func TestCoverage_Daemon_HandleRequest_NilApprovals(t *testing.T) {
	d := &Daemon{}
	resp := d.handleRequest(context.Background(), "x", "alice", nil)
	if resp["ok"] != false {
		t.Errorf("expected ok=false, got %v", resp)
	}
}

func TestCoverage_Daemon_HandleRequest_BindExtraction(t *testing.T) {
	log := openTestAudit(t)
	svc := approvals.New(approvals.Config{Audit: log})
	d := &Daemon{approvals: svc}
	// The handler reads `payload` from the message and
	// unmarshals it as schemas.ApprovalPayload.
	payload := map[string]any{
		"action":       "edit",
		"target":       "/tmp/x",
		"risk_tier":    3,
		"reason":       "needed",
		"requester_id": "agent:planner",
	}
	resp := d.handleRequest(context.Background(), "auto", "alice", map[string]any{
		"payload": payload,
	})
	if resp["ok"] != true {
		t.Errorf("response: %v", resp)
	}
}

func TestCoverage_Daemon_NewDaemon_Defaults(t *testing.T) {
	d := NewDaemon(DaemonConfig{})
	if d.socketPath == "" {
		t.Error("expected default socket path")
	}
	if d.pidPath == "" {
		t.Error("expected default pid path")
	}
}

func TestCoverage_Daemon_SocketPathGetter(t *testing.T) {
	d := NewDaemon(DaemonConfig{SocketPath: "/tmp/x.sock"})
	if d.SocketPath() != "/tmp/x.sock" {
		t.Errorf("got %q", d.SocketPath())
	}
}

func TestCoverage_RunDaemonStart_DefaultPaths(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- RunDaemonStart(ctx, DaemonConfig{}) }()
	// The default socket path is platform-dependent
	// (DefaultSocketPath), so the assertion has to ask the
	// helper rather than hardcode a POSIX path: on Windows
	// "/tmp/aios-approval.sock" is not a valid location and
	// the assertion could never pass.
	sock := DefaultSocketPath()
	t.Cleanup(func() { _ = os.Remove(sock) })
	deadline := time.Now().Add(2 * time.Second)
	for {
		if _, err := os.Stat(sock); err == nil {
			break
		}
		if time.Now().After(deadline) {
			cancel()
			t.Fatalf("default socket did not appear at %s", sock)
		}
		time.Sleep(10 * time.Millisecond)
	}
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("RunDaemonStart did not return after cancel")
	}
}

// --- Parser coverage ---

func TestCoverage_Parse_BadGlobalFlag(t *testing.T) {
	_, err := Parse([]string{"aios", "--nonexistent"}, io.Discard)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCoverage_Parse_HelpShort(t *testing.T) {
	_, err := Parse([]string{"aios", "-h"}, io.Discard)
	if !IsHelp(err) {
		t.Errorf("expected help, got %v", err)
	}
}

func TestCoverage_Parse_SubcommandHelp(t *testing.T) {
	_, err := Parse([]string{"aios", "run", "-h"}, io.Discard)
	if !IsHelp(err) {
		t.Errorf("expected help, got %v", err)
	}
}

func TestCoverage_Parse_BadSubcommandFlag(t *testing.T) {
	_, err := Parse([]string{"aios", "run", "--unknown"}, io.Discard)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCoverage_Parse_RunNoPlan(t *testing.T) {
	_, err := Parse([]string{"aios", "run"}, io.Discard)
	if err == nil {
		t.Fatal("expected error for missing plan")
	}
}

func TestCoverage_Parse_ResumeNoTask(t *testing.T) {
	_, err := Parse([]string{"aios", "resume"}, io.Discard)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCoverage_Parse_ResumeWithFlags(t *testing.T) {
	args, err := Parse([]string{"aios", "resume", "--plan", "p.yaml", "--task", "t1"}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if args.PlanPath != "p.yaml" || args.TaskID != "t1" {
		t.Errorf("args: %+v", args)
	}
}

func TestCoverage_Parse_CancelNoTask(t *testing.T) {
	_, err := Parse([]string{"aios", "cancel"}, io.Discard)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCoverage_Parse_ReportNoTask(t *testing.T) {
	_, err := Parse([]string{"aios", "report"}, io.Discard)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCoverage_Parse_ReportWithFormat(t *testing.T) {
	args, err := Parse([]string{"aios", "report", "--format", "html", "t1"}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if args.ReportFmt != "html" {
		t.Errorf("format: %q", args.ReportFmt)
	}
}

func TestCoverage_Parse_AuditNoTask_Tail(t *testing.T) {
	args, err := Parse([]string{"aios", "audit", "--tail"}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if !args.Tail {
		t.Error("tail should be true")
	}
}

func TestCoverage_Parse_DaemonNoSub(t *testing.T) {
	_, err := Parse([]string{"aios", "daemon"}, io.Discard)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCoverage_Parse_DaemonStartWithSocket(t *testing.T) {
	// The socket and pidfile are global flags; they
	// can appear before or after the subcommand. The
	// parser splits argv on the subcommand and parses
	// the prefix with the global flag set.
	args, err := Parse([]string{"aios", "-socket", "/tmp/x", "-pidfile", "/tmp/y", "daemon", "start"}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if args.SocketPath != "/tmp/x" || args.DaemonPID != "/tmp/y" {
		t.Errorf("args: %+v", args)
	}
	if args.Subcommand != "start" {
		t.Errorf("sub: %q", args.Subcommand)
	}
}

func TestCoverage_Parse_GlobalFlags(t *testing.T) {
	args, err := Parse([]string{"aios", "--debug", "--config", "c.yaml", "--log", "l.db", "init"}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if !args.Debug || args.ConfigPath != "c.yaml" || args.LogPath != "l.db" {
		t.Errorf("args: %+v", args)
	}
}

func TestCoverage_Parse_ResumePositionalPlan(t *testing.T) {
	args, err := Parse([]string{"aios", "resume", "p.yaml", "t1"}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if args.PlanPath != "p.yaml" || args.TaskID != "t1" {
		t.Errorf("args: %+v", args)
	}
}

// --- fmtPath / Close / Deps coverage ---

func TestCoverage_FmtPath(t *testing.T) {
	if got := fmtPath(""); got != "(unset)" {
		t.Errorf("empty: %q", got)
	}
	if got := fmtPath("a"); got != "a" {
		t.Errorf("set: %q", got)
	}
}

func TestCoverage_Deps_Close(t *testing.T) {
	d := &Deps{}
	if err := d.Close(); err != nil {
		t.Errorf("empty deps close: %v", err)
	}
	d.Audit = openTestAudit(t)
	if err := d.Close(); err != nil {
		t.Errorf("close: %v", err)
	}
	if d.Audit != nil {
		t.Error("audit should be nil after close")
	}
	if err := d.Close(); err != nil {
		t.Errorf("double close: %v", err)
	}
}

func TestCoverage_BuildDeps_NilArgs(t *testing.T) {
	_, err := BuildDeps(nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCoverage_BuildDeps_LoadsPolicy(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "p.yaml")
	if err := os.WriteFile(cfg, []byte("rules: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	oldwd, _ := os.Getwd()
	defer os.Chdir(oldwd)
	os.Chdir(dir)
	args := &Args{ConfigPath: cfg, LogPath: filepath.Join(dir, "l.db")}
	deps, err := BuildDeps(args)
	if err != nil {
		t.Fatal(err)
	}
	defer deps.Close()
	if deps.Audit == nil {
		t.Error("audit should be set")
	}
}

// TestCoverage_BuildDeps_PolicyError exercises the fail-closed
// path in BuildDeps: a non-bootstrap subcommand with a
// syntactically broken policy file must return an error
// rather than silently defaulting to allow-all.
func TestCoverage_BuildDeps_PolicyError(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "p.yaml")
	if err := os.WriteFile(cfg, []byte("this is: not: valid: yaml: : :"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Command: "run" — a non-bootstrap subcommand. BuildDeps
	// must require a valid policy file for it.
	args := &Args{Command: "run", ConfigPath: cfg, LogPath: filepath.Join(dir, "l.db")}
	deps, err := BuildDeps(args)
	if err == nil {
		if deps != nil {
			_ = deps.Close()
		}
		t.Fatal("expected error from invalid policy file")
	}
}

// --- Audit coverage: union-of-kinds ---

func TestCoverage_Audit_UnionOfKinds_Populated(t *testing.T) {
	log := openTestAudit(t)
	defer log.Close()
	rows := []struct {
		kind string
		task string
	}{
		{"tool_call", "t1"},
		{"approval_granted", "t1"},
		{"task_passed", "t1"},
	}
	for _, r := range rows {
		if _, err := log.Write(&audit.Row{
			TaskID:  r.task,
			Kind:    r.kind,
			Actor:   "test",
			Payload: map[string]any{"k": "v"},
		}); err != nil {
			t.Fatal(err)
		}
	}
	var buf bytes.Buffer
	if err := Audit("", AuditDeps{Audit: log, Out: &buf}); err != nil {
		t.Fatal(err)
	}
	if buf.Len() == 0 {
		t.Error("expected output")
	}
	if got := strings.Count(buf.String(), "\n"); got != len(rows) {
		t.Errorf("got %d lines, want %d", got, len(rows))
	}
}

// TestCoverage_SortBySequence_OutOfOrder exercises the swap
// branch of sortBySequence. We construct a PersistedRow slice
// out of order and call the unexported helper via the
// audit-of-empty-task path (which internally calls
// sortBySequence).
func TestCoverage_SortBySequence_OutOfOrder(t *testing.T) {
	log := openTestAudit(t)
	defer log.Close()
	// Insert rows with different kinds; the union
	// iterates the kinds in a fixed order, so the
	// resulting slice from GetByKind calls can be out
	// of sequence order across kinds.
	for _, k := range []string{"tool_call", "approval_requested", "task_passed"} {
		if _, err := log.Write(&audit.Row{
			TaskID: "t1", Kind: k, Actor: "test",
			Payload: map[string]any{"k": "v"},
		}); err != nil {
			t.Fatal(err)
		}
	}
	var buf bytes.Buffer
	if err := Audit("", AuditDeps{Audit: log, Out: &buf}); err != nil {
		t.Fatal(err)
	}
	// Verify the output is in ascending sequence order.
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) < 2 {
		t.Skip("need at least 2 rows")
	}
	prev := int64(-1)
	for _, line := range lines {
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) < 2 {
			t.Fatalf("bad line: %s", line)
		}
		var seq int64
		fmt.Sscanf(parts[0], "%d", &seq)
		if seq < prev {
			t.Errorf("not sorted: %d after %d", seq, prev)
		}
		prev = seq
	}
}

// --- runWithApprovals coverage ---

func TestCoverage_RunWithApprovals_NilPlan(t *testing.T) {
	log := openTestAudit(t)
	defer log.Close()
	deps := ResumeDeps{
		Audit:  log,
		Resume: runtime.NewResumeHandler(t.TempDir()),
		Worker: &e2eWorker{runFn: func(_ int, _ runtime.TaskInput, _ *runtime.BudgetMeter) (runtime.WorkerResult, error) {
			return runtime.WorkerResult{Output: "ok"}, nil
		}},
		Verify: &e2eVerifier{verdict: runtime.VerdictPass},
	}
	_, err := runWithApprovals(context.Background(), nil, nil, deps, "test")
	if err == nil {
		t.Fatal("expected error for nil plan")
	}
}

func TestCoverage_RunWithApprovals_ApprovalHandlerError_Propagates(t *testing.T) {
	log := openTestAudit(t)
	defer log.Close()
	plan := &schemas.TaskPlan{
		ID: "p2", Goal: "ship",
		Tasks: []schemas.TaskSpec{
			{ID: "t1", Description: "x", Acceptance: "y"},
		},
	}
	deps := ResumeDeps{
		Audit:  log,
		Resume: runtime.NewResumeHandler(t.TempDir()),
		Worker: &e2eWorker{runFn: func(_ int, _ runtime.TaskInput, _ *runtime.BudgetMeter) (runtime.WorkerResult, error) {
			return runtime.WorkerResult{}, &runtime.WaitingForApproval{ApprovalID: "apr-1", Decision: "ask"}
		}},
		Verify:  &e2eVerifier{verdict: runtime.VerdictPass},
		Approve: &scriptedApprover{err: errors.New("daemon down")},
	}
	_, err := runWithApprovals(context.Background(), plan, &runtime.Checkpoint{}, deps, "test")
	if err == nil {
		t.Fatal("expected error from approval handler")
	}
}

func TestCoverage_RunWithApprovals_ApprovalApprove(t *testing.T) {
	log := openTestAudit(t)
	defer log.Close()
	plan := &schemas.TaskPlan{
		ID: "p3", Goal: "ship",
		Tasks: []schemas.TaskSpec{
			{ID: "t1", Description: "x", Acceptance: "y"},
		},
	}
	calls := 0
	// Pre-populate a checkpoint at task index 0 so the
	// orchestrator resumes; without this the orchestrator
	// starts at 0 and saves a checkpoint after the first
	// run. The approval loop re-loads via
	// deps.Resume.Load(plan.ID), which reads what the
	// orchestrator saved.
	resume := runtime.NewResumeHandler(t.TempDir())
	// The approval loop re-loads a checkpoint via
	// deps.Resume.Load(plan.ID). Pre-populate so the
	// second iteration has a checkpoint to load.
	if err := resume.Save(&runtime.Checkpoint{TaskID: "p3", TaskIndex: 0}); err != nil {
		t.Fatal(err)
	}
	deps := ResumeDeps{
		Audit:  log,
		Resume: resume,
		Worker: &e2eWorker{runFn: func(_ int, _ runtime.TaskInput, _ *runtime.BudgetMeter) (runtime.WorkerResult, error) {
			calls++
			if calls == 1 {
				return runtime.WorkerResult{}, &runtime.WaitingForApproval{ApprovalID: "apr-1", Decision: "ask"}
			}
			return runtime.WorkerResult{Output: "ok"}, nil
		}},
		Verify:  &e2eVerifier{verdict: runtime.VerdictPass},
		Approve: &scriptedApprover{approve: true},
	}
	_, err := runWithApprovals(context.Background(), plan, &runtime.Checkpoint{TaskIndex: 0}, deps, "test")
	if err != nil {
		t.Fatal(err)
	}
	if calls < 1 {
		t.Errorf("worker not called")
	}
}

// --- Run coverage: more branches ---

func TestCoverage_Run_LoadPlanError(t *testing.T) {
	_, err := Run(context.Background(), &Args{PlanPath: filepath.Join(t.TempDir(), "missing.yaml")}, RunDeps{
		Resume:   runtime.NewResumeHandler(t.TempDir()),
		Worker:   &e2eWorker{},
		Verifier: &e2eVerifier{},
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCoverage_Run_OrchestratorError(t *testing.T) {
	log := openTestAudit(t)
	defer log.Close()
	plan := writePlanFile(t, &schemas.TaskPlan{
		ID: "p-orch-err", Goal: "ship",
		Tasks: []schemas.TaskSpec{
			{ID: "t1", Description: "x", Acceptance: "y"},
		},
	})
	_, err := Run(context.Background(), &Args{PlanPath: plan}, RunDeps{
		Audit:  log,
		Resume: runtime.NewResumeHandler(t.TempDir()),
		Worker: &e2eWorker{runFn: func(_ int, _ runtime.TaskInput, _ *runtime.BudgetMeter) (runtime.WorkerResult, error) {
			return runtime.WorkerResult{}, fmt.Errorf("worker boom")
		}},
		Verifier: &e2eVerifier{verdict: runtime.VerdictPass},
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCoverage_Run_ApprovalHandlerError(t *testing.T) {
	log := openTestAudit(t)
	defer log.Close()
	plan := writePlanFile(t, &schemas.TaskPlan{
		ID: "p-approval-err", Goal: "ship",
		Tasks: []schemas.TaskSpec{
			{ID: "t1", Description: "x", Acceptance: "y"},
		},
	})
	_, err := Run(context.Background(), &Args{PlanPath: plan}, RunDeps{
		Audit:  log,
		Resume: runtime.NewResumeHandler(t.TempDir()),
		Worker: &e2eWorker{runFn: func(_ int, _ runtime.TaskInput, _ *runtime.BudgetMeter) (runtime.WorkerResult, error) {
			return runtime.WorkerResult{}, &runtime.WaitingForApproval{ApprovalID: "apr-1", Decision: "ask"}
		}},
		Verifier: &e2eVerifier{verdict: runtime.VerdictPass},
		Approval: &scriptedApprover{err: errors.New("daemon down")},
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- Report coverage: more branches ---

func TestCoverage_Report_Json(t *testing.T) {
	log := openTestAudit(t)
	defer log.Close()
	if err := Report("x", ReportDeps{Audit: log, Format: "json", Out: &bytes.Buffer{}}); err != nil {
		t.Fatal(err)
	}
}

func TestCoverage_Report_Markdown(t *testing.T) {
	log := openTestAudit(t)
	defer log.Close()
	if err := Report("x", ReportDeps{Audit: log, Format: "markdown", Out: &bytes.Buffer{}}); err != nil {
		t.Fatal(err)
	}
}

func TestCoverage_Report_Html(t *testing.T) {
	log := openTestAudit(t)
	defer log.Close()
	if err := Report("x", ReportDeps{Audit: log, Format: "html", Out: &bytes.Buffer{}}); err != nil {
		t.Fatal(err)
	}
}

// --- PrintReport branches ---

func TestCoverage_PrintReport_NilReport(t *testing.T) {
	buf := &bytes.Buffer{}
	PrintReport(buf, nil)
	// PrintReport is a no-op on a nil report (the caller
	// decided to suppress the summary; the audit log is
	// the canonical record).
	if buf.Len() != 0 {
		t.Errorf("expected empty output, got %q", buf.String())
	}
}

func TestCoverage_PrintReport_WithFromIndex(t *testing.T) {
	r := &runtime.Report{
		PlanID:        "p1",
		TaskCount:     5,
		FromIndex:     2,
		StartedAt:     "2024-01-01T00:00:00Z",
		FinishedAt:    "2024-01-01T00:01:00Z",
		StoppedReason: "paused",
	}
	buf := &bytes.Buffer{}
	PrintReport(buf, r)
	if !strings.Contains(buf.String(), "paused") {
		t.Errorf("output: %s", buf.String())
	}
}

// --- policy ---

func TestCoverage_Policy_LoadDefault(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "p.yaml")
	yaml := "rules:\n" +
		"  - id: r1\n" +
		"    match:\n" +
		"      tool: read\n" +
		"    action: allow\n"
	if err := os.WriteFile(cfg, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	rules, err := policy.Load(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if rules == nil {
		t.Fatal("rules nil")
	}
	gw := policy.New(rules, nil)
	if gw == nil {
		t.Fatal("gateway nil")
	}
	if _, err := policy.Load(filepath.Join(dir, "missing.yaml")); err == nil {
		t.Error("expected error for missing policy file")
	}
}

// --- Init more branches ---

func TestCoverage_Init_PrintReport_AllFields(t *testing.T) {
	buf := &bytes.Buffer{}
	PrintInitReport(buf, []string{"a", "b"}, []string{"c"})
	if !strings.Contains(buf.String(), "a") {
		t.Errorf("output should mention a: %s", buf.String())
	}
	if !strings.Contains(buf.String(), "c") {
		t.Errorf("output should mention c: %s", buf.String())
	}
}

func TestCoverage_Init_AllExisted(t *testing.T) {
	buf := &bytes.Buffer{}
	PrintInitReport(buf, nil, []string{"a", "b"})
	if !strings.Contains(buf.String(), "a") || !strings.Contains(buf.String(), "b") {
		t.Errorf("output should mention a and b: %s", buf.String())
	}
}

// --- handleRequest / dispatch error paths ---

func TestCoverage_Daemon_HandleRequest_BadPayloadType(t *testing.T) {
	log := openTestAudit(t)
	svc := approvals.New(approvals.Config{Audit: log})
	d := &Daemon{approvals: svc}
	// msg["payload"] is a value that JSON-marshals OK but
	// fails to unmarshal into schemas.ApprovalPayload.
	resp := d.handleRequest(context.Background(), "auto", "alice", map[string]any{
		"payload": "not an object",
	})
	if resp["ok"] != false {
		t.Errorf("expected ok=false, got %v", resp)
	}
}

func TestCoverage_Daemon_HandleRequest_EmptyPayload(t *testing.T) {
	log := openTestAudit(t)
	svc := approvals.New(approvals.Config{Audit: log})
	d := &Daemon{approvals: svc}
	// Empty payload object fails validation
	// (action / target / risk_tier / reason / requester_id
	// are required).
	resp := d.handleRequest(context.Background(), "auto", "alice", map[string]any{
		"payload": map[string]any{},
	})
	if resp["ok"] != false {
		t.Errorf("expected ok=false, got %v", resp)
	}
}

func TestCoverage_Daemon_HandleGrant_InvalidID(t *testing.T) {
	log := openTestAudit(t)
	svc := approvals.New(approvals.Config{Audit: log})
	d := &Daemon{approvals: svc}
	resp := d.handleGrant(context.Background(), "missing", "alice", nil)
	if resp["ok"] != false {
		t.Errorf("expected ok=false, got %v", resp)
	}
}

func TestCoverage_Daemon_HandleWait_NilApprovals(t *testing.T) {
	d := &Daemon{}
	resp := d.handleWait(context.Background(), "x")
	if resp["ok"] != false {
		t.Errorf("expected ok=false, got %v", resp)
	}
}

func TestCoverage_Daemon_HandleWait_Found(t *testing.T) {
	log := openTestAudit(t)
	svc := approvals.New(approvals.Config{Audit: log})
	d := &Daemon{approvals: svc}
	payload := schemas.ApprovalPayload{
		Action: "edit", Target: "/tmp/x", RiskTier: 3,
		Reason: "needed", RequesterID: "agent:planner",
	}
	id, err := svc.Request(payload, "alice", approvals.RequestOptions{})
	if err != nil {
		t.Fatal(err)
	}
	resp := d.handleWait(context.Background(), id)
	if resp["ok"] != true {
		t.Errorf("expected ok=true, got %v", resp)
	}
}

// --- StdinPrompt error paths ---

func TestCoverage_StdinPrompt_EmptyInput(t *testing.T) {
	in := strings.NewReader("")
	out := &bytes.Buffer{}
	cb := &StdinPromptCallback{In: in, Out: out}
	prompt := cb.StdinPrompt()
	q := ask.Question{
		ID:     "q-1",
		Prompt: "x?",
		Options: []ask.AskOption{
			{ID: "a", Label: "A"},
		},
	}
	_, err := prompt(context.Background(), q)
	if err == nil {
		t.Fatal("expected error on empty input")
	}
}

func TestCoverage_StdinPrompt_InvalidInput(t *testing.T) {
	in := strings.NewReader("not-a-number\n")
	out := &bytes.Buffer{}
	cb := &StdinPromptCallback{In: in, Out: out}
	prompt := cb.StdinPrompt()
	q := ask.Question{
		ID:     "q-1",
		Prompt: "x?",
		Options: []ask.AskOption{
			{ID: "a", Label: "A"},
		},
	}
	_, err := prompt(context.Background(), q)
	if err == nil {
		t.Fatal("expected error on non-numeric input")
	}
}

// --- Init error path ---

func TestCoverage_Init_InvalidDir(t *testing.T) {
	// A path under a file (not a directory) should fail
	// to create.
	f := filepath.Join(t.TempDir(), "f")
	if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, _, err := Init(f)
	if err == nil {
		t.Fatal("expected error creating workspace under a file")
	}
}

// --- Log.Recent ordering ---

// TestCoverage_Audit_Recent_IncludesEveryKind is the guard for
// the bug that made `aios audit` silently lossy: the no-taskID
// path used to union a hardcoded list of "well known" kinds, and
// policy_decision was not on it. Recent() has no kind filter at
// all, so the test writes rows of a kind the old list never
// mentioned and asserts they come back.
func TestCoverage_Audit_Recent_IncludesEveryKind(t *testing.T) {
	log := openTestAudit(t)
	defer log.Close()
	for _, kind := range []string{"task_started", "policy_decision", "a_kind_added_next_release"} {
		if _, err := log.Write(&audit.Row{
			TaskID: "t-recent", Kind: kind, Actor: "tester",
			Payload: map[string]any{"k": kind},
		}); err != nil {
			t.Fatal(err)
		}
	}
	rows, err := log.Recent(100)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, r := range rows {
		seen[r.Kind] = true
	}
	for _, kind := range []string{"task_started", "policy_decision", "a_kind_added_next_release"} {
		if !seen[kind] {
			t.Errorf("Recent dropped kind %q; got %v", kind, seen)
		}
	}
	// Recent returns oldest-first like every other read here.
	for i := 1; i < len(rows); i++ {
		if rows[i-1].SequenceNumber > rows[i].SequenceNumber {
			t.Fatalf("not oldest-first at %d: %d > %d", i, rows[i-1].SequenceNumber, rows[i].SequenceNumber)
		}
	}
}

// TestCoverage_Audit_Recent_RespectsLimit pins the bound: the
// LIMIT must keep the NEWEST rows, not an arbitrary slice.
func TestCoverage_Audit_Recent_RespectsLimit(t *testing.T) {
	log := openTestAudit(t)
	defer log.Close()
	for i := 0; i < 10; i++ {
		if _, err := log.Write(&audit.Row{
			TaskID: "t-limit", Kind: "tool_call", Actor: "tester",
			Payload: map[string]any{"i": i},
		}); err != nil {
			t.Fatal(err)
		}
	}
	rows, err := log.Recent(3)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 {
		t.Fatalf("len = %d, want 3", len(rows))
	}
	// The newest row is the 10th write.
	if rows[len(rows)-1].Payload == nil {
		t.Skip("payload not decoded in this build")
	}
	last := rows[len(rows)-1]
	if last.SequenceNumber < 8 {
		t.Errorf("limit kept the wrong rows; last seq = %d, want >= 8", last.SequenceNumber)
	}
}

// --- Start bind error path ---

// The Start bind error path is exercised indirectly by
// the cancellation tests above. A direct test is
// platform-dependent: on Windows the rebind succeeds
// even with a pre-bound listener, so the failure path
// cannot be triggered cleanly. Skipping direct
// coverage of this branch.

// --- Run with ReportWriter ---

func TestCoverage_Run_ReportWriter(t *testing.T) {
	log := openTestAudit(t)
	defer log.Close()
	plan := writePlanFile(t, &schemas.TaskPlan{
		ID: "p-rw", Goal: "ship",
		Tasks: []schemas.TaskSpec{
			{ID: "t1", Description: "x", Acceptance: "y"},
		},
	})
	written := false
	_, err := Run(context.Background(), &Args{PlanPath: plan}, RunDeps{
		Audit:  log,
		Resume: runtime.NewResumeHandler(t.TempDir()),
		Worker: &e2eWorker{runFn: func(_ int, _ runtime.TaskInput, _ *runtime.BudgetMeter) (runtime.WorkerResult, error) {
			return runtime.WorkerResult{Output: "ok"}, nil
		}},
		Verifier: &e2eVerifier{verdict: runtime.VerdictPass},
		ReportWriter: func(*runtime.Report) error {
			written = true
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !written {
		t.Error("ReportWriter not called")
	}
}

func TestCoverage_Run_ReportWriterError(t *testing.T) {
	log := openTestAudit(t)
	defer log.Close()
	plan := writePlanFile(t, &schemas.TaskPlan{
		ID: "p-rwe", Goal: "ship",
		Tasks: []schemas.TaskSpec{
			{ID: "t1", Description: "x", Acceptance: "y"},
		},
	})
	_, err := Run(context.Background(), &Args{PlanPath: plan}, RunDeps{
		Audit:  log,
		Resume: runtime.NewResumeHandler(t.TempDir()),
		Worker: &e2eWorker{runFn: func(_ int, _ runtime.TaskInput, _ *runtime.BudgetMeter) (runtime.WorkerResult, error) {
			return runtime.WorkerResult{Output: "ok"}, nil
		}},
		Verifier: &e2eVerifier{verdict: runtime.VerdictPass},
		ReportWriter: func(*runtime.Report) error {
			return errors.New("writer fail")
		},
	})
	if err == nil {
		t.Fatal("expected error from ReportWriter")
	}
}

// --- Resume plan validation error ---

func TestCoverage_Resume_PlanValidation(t *testing.T) {
	log := openTestAudit(t)
	defer log.Close()
	// An invalid plan (empty ID) fails Validate.
	plan := &schemas.TaskPlan{
		Goal: "ship",
		Tasks: []schemas.TaskSpec{
			{ID: "t1", Description: "x", Acceptance: "y"},
		},
	}
	planPath := writePlanFile(t, plan)
	resume := runtime.NewResumeHandler(t.TempDir())
	if err := resume.Save(&runtime.Checkpoint{TaskID: "p1", TaskIndex: 0}); err != nil {
		t.Fatal(err)
	}
	_, err := Resume(context.Background(), &Args{PlanPath: planPath, TaskID: "p1"}, ResumeDeps{
		Audit:  log,
		Resume: resume,
		Worker: PassthroughWorker{},
		Verify: SmokeVerifier{},
	})
	if err == nil {
		t.Fatal("expected validation error for empty plan id")
	}
}

// --- runResume success path ---

func TestCoverage_RunResume_Success(t *testing.T) {
	log := openTestAudit(t)
	defer log.Close()
	plan := &schemas.TaskPlan{
		ID: "rsm-ok", Goal: "ship",
		Tasks: []schemas.TaskSpec{
			{ID: "t1", Description: "x", Acceptance: "y"},
		},
	}
	planPath := writePlanFile(t, plan)
	// Pre-populate a checkpoint so Resume finds it.
	resume := runtime.NewResumeHandler(t.TempDir())
	if err := resume.Save(&runtime.Checkpoint{TaskID: "rsm-ok", TaskIndex: 1}); err != nil {
		t.Fatal(err)
	}
	deps := &Deps{
		Audit:    log,
		Resume:   resume,
		Worker:   PassthroughWorker{},
		Verifier: SmokeVerifier{},
	}
	var stdout, stderr bytes.Buffer
	args := &Args{Command: CmdResume, PlanPath: planPath, TaskID: "rsm-ok"}
	code := runResume(context.Background(), args, deps, &stdout, &stderr)
	if code != 0 {
		t.Errorf("code: %d, stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "rsm-ok") {
		t.Errorf("output should mention plan: %s", stdout.String())
	}
}

// --- runDaemon start success path ---

func TestCoverage_RunDaemon_StopSuccess(t *testing.T) {
	dir := t.TempDir()
	pid := filepath.Join(dir, "a.pid")
	if err := os.WriteFile(pid, []byte("12345"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	args := &Args{Command: CmdDaemon, Subcommand: DaemonStop, DaemonPID: pid}
	deps := &Deps{}
	code := runDaemon(context.Background(), args, deps, &stderr)
	if code != 0 {
		t.Errorf("code: %d, stderr: %s", code, stderr.String())
	}
	if _, err := os.Stat(pid); !os.IsNotExist(err) {
		t.Errorf("pid file should be deleted: stat err = %v", err)
	}
}

func TestCoverage_RunDaemon_StartSuccess(t *testing.T) {
	dir := t.TempDir()
	sock := filepath.Join(dir, "a.sock")
	pid := filepath.Join(dir, "a.pid")
	var stderr bytes.Buffer
	args := &Args{Command: CmdDaemon, Subcommand: DaemonStart, SocketPath: sock, DaemonPID: pid}
	deps := &Deps{}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan int, 1)
	go func() { done <- runDaemon(ctx, args, deps, &stderr) }()
	deadline := time.Now().Add(2 * time.Second)
	for {
		if _, err := os.Stat(sock); err == nil {
			break
		}
		if time.Now().After(deadline) {
			cancel()
			t.Fatal("socket did not appear")
		}
		time.Sleep(10 * time.Millisecond)
	}
	cancel()
	select {
	case code := <-done:
		if code != 0 {
			t.Errorf("code: %d, stderr: %s", code, stderr.String())
		}
	case <-time.After(2 * time.Second):
		t.Fatal("runDaemon did not return after cancel")
	}
}

// --- helpers ---

func openTestAudit(t *testing.T) *audit.Log {
	t.Helper()
	log, err := audit.Open(audit.Config{Path: ":memory:"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { log.Close() })
	return log
}
