package cli

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Hamidullah02/aios/aios-system/internal/ask"
	"github.com/Hamidullah02/aios/aios-system/internal/core/audit"
	"github.com/Hamidullah02/aios/aios-system/internal/core/schemas"
	"github.com/Hamidullah02/aios/aios-system/internal/policy"
	"github.com/Hamidullah02/aios/aios-system/internal/runtime"
	"github.com/Hamidullah02/aios/aios-system/internal/sandbox"
	"github.com/Hamidullah02/aios/aios-system/internal/tools"
)

// TestBusWorker_EndToEnd exercises the full chain:
//
//	task → BusWorker → bus → policy → tool → audit
//
// and the orchestrator that drives the loop. This is the
// production wire the doc set describes; the unit tests
// of bus, policy, and runtime verify each piece
// separately. This test verifies they fit together.
func TestBusWorker_EndToEnd(t *testing.T) {
	log := openTestAudit(t)
	defer log.Close()

	gw := policy.New([]policy.Rule{
		{ID: "allow-read", Match: policy.Match{Tool: "read_file"},
			Action: policy.ActionAllow},
	}, log)
	bus := tools.New(tools.Config{
		Tools:  map[string]tools.Tool{"read_file": tools.ReadFileTool{}},
		Policy: gw,
		Audit:  log,
	})
	dir := t.TempDir()
	target := filepath.Join(dir, "hello.txt")
	if err := os.WriteFile(target, []byte("hi from bus"), 0o644); err != nil {
		t.Fatal(err)
	}
	w := NewBusWorker(bus, nil, dir)

	o := runtime.New(runtime.Config{
		Worker:   w,
		Verifier: busPassVerifier{},
		Meter:    runtime.NewBudgetMeter(runtime.Budget{}),
		Audit:    log,
		Resume:   runtime.NewResumeHandler(t.TempDir()),
	})

	plan := &schemas.TaskPlan{
		ID: "e2e-1", Goal: "smoke",
		Tasks: []schemas.TaskSpec{
			{ID: "t1", Description: "read hello", Acceptance: "ok",
				Tool: "read_file", Inputs: []string{"path=" + target}},
		},
	}

	report, err := o.Run(context.Background(), plan, nil)
	if err != nil {
		t.Fatal(err)
	}
	if report.StoppedReason != "completed" {
		t.Fatalf("StoppedReason = %q", report.StoppedReason)
	}

	rows, _ := log.GetByTask("e2e-1")
	if !kindExists(rows, "decision") {
		t.Errorf("expected decision row, got: %v", kindsOf(rows))
	}
	// The bus writes its tool_call row against the
	// task id (input.ID), not the plan id. Look up
	// the task-level rows separately and verify the
	// bus audited the dispatch.
	taskRows, _ := log.GetByTask("t1")
	if !kindExists(taskRows, "tool_call") {
		t.Errorf("expected tool_call row for t1, got: %v", kindsOf(taskRows))
	}
	// The chain is global (sequence_number is per-log, not
	// per-task), and every row links to the one before it,
	// whatever its kind. So the chain can only be verified
	// over the COMPLETE log: selecting a subset of kinds
	// skips rows the hash still depends on and yields a
	// spurious mismatch.
	//
	// v0.41.0 added task_started / task_passed rows, which
	// is what surfaced this: the old filter listed three
	// kinds and silently dropped the new ones, so VerifyChain
	// saw row 2's prev_hash pointing at a row it did not
	// have. Every kind the orchestrator or bus writes must
	// be listed here; a new kind that is forgotten will fail
	// this test loudly, which is the intended alarm.
	allRows := mustGetAllRows(t, log)
	if v := audit.VerifyChain(allRows); !v.OK {
		t.Errorf("global chain invalid: %s (rows=%d)", v.Reason, len(allRows))
	}
}

func TestBusWorker_PolicyDeny(t *testing.T) {
	log := openTestAudit(t)
	defer log.Close()
	gw := policy.New([]policy.Rule{
		{ID: "deny-read", Match: policy.Match{Tool: "read_file"},
			Action: policy.ActionDeny, Reason: "no reads"},
	}, log)
	bus := tools.New(tools.Config{
		Tools:  map[string]tools.Tool{"read_file": tools.ReadFileTool{}},
		Policy: gw,
		Audit:  log,
	})
	w := NewBusWorker(bus, nil, "")
	_, err := w.Run(runtime.TaskInput{ID: "t1", Tool: "read_file"},
		&runtime.TaskContext{Vars: map[string]any{}}, nil)
	if err == nil {
		t.Fatal("expected deny error")
	}
}

func TestBusWorker_PolicyAsk(t *testing.T) {
	log := openTestAudit(t)
	defer log.Close()
	gw := policy.New([]policy.Rule{
		{ID: "ask-read", Match: policy.Match{Tool: "read_file"},
			Action: policy.ActionAsk, Reason: "operator review",
			Approval: &policy.ApprovalTemplate{Action: "read", Resource: "{tool}"}},
	}, log)
	bus := tools.New(tools.Config{
		Tools:  map[string]tools.Tool{"read_file": tools.ReadFileTool{}},
		Policy: gw,
		Audit:  log,
	})
	w := NewBusWorker(bus, nil, "")
	_, err := w.Run(runtime.TaskInput{ID: "t1", Tool: "read_file"},
		&runtime.TaskContext{Vars: map[string]any{}}, nil)
	var wfa *runtime.WaitingForApproval
	if !errors.As(err, &wfa) {
		t.Fatalf("err type = %T, want *WaitingForApproval", err)
	}
}

func TestBusWorker_ShellViaSandbox(t *testing.T) {
	log := openTestAudit(t)
	defer log.Close()

	runner := sandbox.NewSubprocessRunner()
	bus := tools.New(tools.Config{
		Tools: map[string]tools.Tool{
			"run_shell": tools.RunShellTool{Runner: runner},
		},
		Audit: log,
	})
	w := NewBusWorker(bus, nil, "")

	o := runtime.New(runtime.Config{
		Worker:   w,
		Verifier: busPassVerifier{},
		Meter:    runtime.NewBudgetMeter(runtime.Budget{}),
		Audit:    log,
		Resume:   runtime.NewResumeHandler(t.TempDir()),
	})

	// run_shell expects argv. The command must be a real
	// executable; on Windows the echo is a shell builtin
	// so we route through cmd.exe.
	plan := &schemas.TaskPlan{
		ID: "sh-1", Goal: "echo",
		Tasks: []schemas.TaskSpec{
			{ID: "t1", Description: "echo", Acceptance: "ok",
				Tool:   "run_shell",
				Inputs: []string{"command=cmd,/c,echo,hello"}},
		},
	}
	report, err := o.Run(context.Background(), plan, nil)
	if err != nil {
		t.Fatal(err)
	}
	if report.StoppedReason != "completed" {
		t.Fatalf("StoppedReason = %q", report.StoppedReason)
	}
	taskRows, _ := log.GetByTask("t1")
	if !kindExists(taskRows, "tool_call") {
		t.Errorf("expected tool_call row for t1, got: %v", kindsOf(taskRows))
	}
}

// TestBusWorker_AskTool exercises the ask-user tool when
// the agent dispatches it. The test wires the ask tool
// into the bus as "ask_user" through the adapter and
// verifies the bus returns the answer via a tool_call
// audit row.
func TestBusWorker_AskTool(t *testing.T) {
	log := openTestAudit(t)
	defer log.Close()

	askTool, _ := ask.New(func(_ context.Context, q ask.Question) (ask.Answer, error) {
		return ask.Answer{QuestionID: q.ID, SelectedOption: q.Options[0].ID}, nil
	}, nil)

	bus := tools.New(tools.Config{
		Tools: map[string]tools.Tool{
			"ask_user": askAdapterForTest{tool: askTool},
		},
		Audit: log,
	})
	// Use a bus that accepts pre-built args. The bus
	// worker is the entry point; for the ask tool the
	// caller should pass the question as a typed struct
	// in `Inputs[0]` so the comma-in-JSON case is
	// handled.
	w := NewBusWorker(bus, askTool, "")

	// Pass the question via a JSON string the ask
	// decoder accepts (it is a `case string` branch).
	_, err := w.Run(runtime.TaskInput{
		ID:     "t1",
		Tool:   "ask_user",
		Inputs: []string{`question=` + singleQJSON},
	}, &runtime.TaskContext{Vars: map[string]any{}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	taskRows, _ := log.GetByTask("t1")
	if !kindExists(taskRows, "tool_call") {
		t.Errorf("expected tool_call row for t1, got: %v", kindsOf(taskRows))
	}
}

// singleQJSON is a JSON-encoded Question the ask decoder
// accepts. It is package-private because the test is the
// only consumer; production code builds the question
// through the ask package's typed API.
const singleQJSON = `{"id":"q1","prompt":"pick","options":[{"id":"a","label":"A"}]}`

// --- helpers ---

// askAdapterForTest adapts an ask.AskUserTool to the
// tools.Tool interface without creating an import cycle
// (tools does not import ask, and ask does not import
// tools). It mirrors the production askAdapter in
// ask/bus_adapter_test.go.
type askAdapterForTest struct{ tool *ask.AskUserTool }

func (a askAdapterForTest) Name() string { return a.tool.Name() }
func (a askAdapterForTest) Run(args map[string]any, _ tools.Context) (tools.Result, error) {
	res, err := a.tool.Run(context.Background(), args)
	return tools.Result{OK: res.OK, Output: res.Output, Summary: res.Summary}, err
}

type busPassVerifier struct{}

func (busPassVerifier) Verify(_ runtime.TaskInput, _ runtime.WorkerResult) (runtime.VerifierVerdict, error) {
	return runtime.VerdictPass, nil
}

func kindExists(rows []audit.PersistedRow, kind string) bool {
	for _, r := range rows {
		if r.Kind == kind {
			return true
		}
	}
	return false
}

func kindsOf(rows []audit.PersistedRow) []string {
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.Kind)
	}
	return out
}

// mustGetByKind is a small wrapper that ignores the
// "no rows" case and returns an empty slice. VerifyChain
// handles the empty input correctly.
func mustGetByKind(log *audit.Log, kind string, limit int) []audit.PersistedRow {
	rows, _ := log.GetByKind(kind, limit)
	return rows
}

// auditChainKinds is every row kind the orchestrator and the
// bus write in a plain run.
//
// VerifyChain hashes each row against its predecessor, so a
// chain check that omits a kind is checking a fiction. This
// list must be kept complete: a new kind that is not added
// here makes the end-to-end chain tests fail, which is the
// alarm working as intended rather than a false pass.
var auditChainKinds = []string{
	"tool_call",
	"policy_decision",
	"decision",
	"task_started", // v0.41.0
	"task_passed",  // v0.41.0
}

// mustGetAllRows returns every row in the log, in chain
// order.
func mustGetAllRows(t *testing.T, log *audit.Log) []audit.PersistedRow {
	t.Helper()
	seen := make(map[int64]bool)
	var rows []audit.PersistedRow
	for _, kind := range auditChainKinds {
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
	sortBySeq(rows)
	return rows
}

// sortBySeq sorts rows in ascending sequence_number
// order so the chain check sees them in write order.
// The sort is in place; the input slice is mutated.
func sortBySeq(rows []audit.PersistedRow) {
	for i := 1; i < len(rows); i++ {
		for j := i; j > 0 && rows[j-1].SequenceNumber > rows[j].SequenceNumber; j-- {
			rows[j-1], rows[j] = rows[j], rows[j-1]
		}
	}
}
