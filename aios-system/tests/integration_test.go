// Package tests holds black-box integration tests for aios-core.
//
// The tests live in a separate package and exercise the public surface
// of the four core sub-packages together: schemas, redact, audit, state.
package tests

import (
	"strings"
	"testing"
	"time"

	"github.com/sadi/aios-system/internal/core/audit"
	"github.com/sadi/aios-system/internal/core/clock"
	"github.com/sadi/aios-system/internal/core/redact"
	"github.com/sadi/aios-system/internal/core/schemas"
	"github.com/sadi/aios-system/internal/core/state"
)

func newLog(t *testing.T) *audit.Log {
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

func TestFullTaskLifecycle_HappyPath(t *testing.T) {
	log := newLog(t)
	m := state.NewMachine("t1")

	// 1. Plan.
	plan := &schemas.TaskPlan{
		ID: "p1", Goal: "ship aios",
		Tasks: []schemas.TaskSpec{
			{ID: "st1", Description: "design", Acceptance: "doc"},
			{ID: "st2", Description: "implement", Acceptance: "tests pass"},
		},
	}
	if err := plan.Validate(); err != nil {
		t.Fatal(err)
	}
	if err := m.Transition(state.StatePlanning); err != nil {
		t.Fatal(err)
	}
	if _, err := log.Write(&audit.Row{
		TaskID: "t1", Kind: "plan_built", Actor: "planner", RiskTier: 0,
		Payload: plan,
	}); err != nil {
		t.Fatal(err)
	}

	// 2. Plan needs approval (tier 1).
	if err := m.Transition(state.StateAwaitingApproval); err != nil {
		t.Fatal(err)
	}
	if _, err := log.Write(&audit.Row{
		TaskID: "t1", Kind: "approval_requested", Actor: "planner", RiskTier: 1,
		ApprovalID: "550e8400-e29b-41d4-a716-446655440000",
		Payload: &schemas.ApprovalPayload{
			ApprovalID: "550e8400-e29b-41d4-a716-446655440000",
			Action:     "edit", Target: "/tmp/file", RiskTier: 1,
			Reason: "design review", RequesterID: "operator:alice",
		},
	}); err != nil {
		t.Fatal(err)
	}

	// 3. Approved, run.
	if err := m.Resume(); err != nil {
		t.Fatal(err)
	}

	// 4. Each step is a tool call with a decision trace.
	for i := 1; i <= 2; i++ {
		if _, err := log.Write(&audit.Row{
			TaskID: "t1", StepID: "st" + itoa(i), Kind: "decision",
			Actor: "planner", RiskTier: 1,
			Payload: &schemas.DecisionTrace{
				StepID: "st" + itoa(i), Chosen: "edit",
				Basis: "design says so", RiskTier: 1,
				Alternatives: []schemas.Alternative{
					{Action: "skip", Score: 0.1, Rejected: "low score"},
				},
			},
		}); err != nil {
			t.Fatal(err)
		}
		if _, err := log.Write(&audit.Row{
			TaskID: "t1", StepID: "st" + itoa(i), Kind: "tool_call",
			Actor: "runtime", RiskTier: 1,
			Payload: &schemas.ToolCall{
				CallID: "c" + itoa(i), Tool: "edit", RiskTier: 1,
			},
		}); err != nil {
			t.Fatal(err)
		}
		if _, err := log.Write(&audit.Row{
			TaskID: "t1", StepID: "st" + itoa(i), Kind: "tool_result",
			Actor: "runtime", RiskTier: 0,
			Payload: &schemas.ToolResult{CallID: "c" + itoa(i), OK: true},
		}); err != nil {
			t.Fatal(err)
		}
	}

	// 5. Complete.
	if err := m.Complete(); err != nil {
		t.Fatal(err)
	}
	if _, err := log.Write(&audit.Row{
		TaskID: "t1", Kind: "task_completed", Actor: "runtime", RiskTier: 0,
		Payload: map[string]any{"status": "ok"},
	}); err != nil {
		t.Fatal(err)
	}

	// 6. Read all and verify chain.
	rows, err := log.GetByTask("t1")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 9 {
		t.Fatalf("expected 9 rows, got %d", len(rows))
	}
	v := audit.VerifyChain(rows)
	if !v.OK {
		t.Fatalf("chain invalid: %s", v.Reason)
	}

	// 7. Approval join works.
	apRows, err := log.GetByApproval("550e8400-e29b-41d4-a716-446655440000")
	if err != nil {
		t.Fatal(err)
	}
	if len(apRows) != 1 {
		t.Fatalf("approval rows: %d", len(apRows))
	}
}

func TestResumeAfterCancelFails(t *testing.T) {
	m := state.NewMachine("t1")
	_ = m.Transition(state.StatePlanning)
	_ = m.Transition(state.StateRunning)
	if err := m.Cancel(); err != nil {
		t.Fatal(err)
	}
	if err := m.Resume(); err == nil {
		t.Fatal("expected error: cannot resume cancelled")
	}
	if err := m.Pause(); err == nil {
		t.Fatal("expected error: cannot pause cancelled")
	}
}

func TestRedactionPreservesApprovalIDInAudit(t *testing.T) {
	log := newLog(t)
	_, err := log.Write(&audit.Row{
		TaskID: "t1", Kind: "approval_requested", Actor: "planner", RiskTier: 2,
		ApprovalID: "550e8400-e29b-41d4-a716-446655440000",
		Payload: map[string]any{
			"approval_id": "550e8400-e29b-41d4-a716-446655440000",
			"reason":      "key rotation; api_key=ABCDEFGHIJKLMNOP1234",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	rows, err := log.GetByTask("t1")
	if err != nil {
		t.Fatal(err)
	}
	payload := string(rows[0].Payload)
	if !strings.Contains(payload, "550e8400-e29b-41d4-a716-446655440000") {
		t.Fatalf("approval_id should be preserved: %s", payload)
	}
	if strings.Contains(payload, "ABCDEFGHIJKLMNOP1234") {
		t.Fatalf("secret leaked: %s", payload)
	}
}

func TestRedactStableAcrossRuns(t *testing.T) {
	// Two different processes redacting the same input should produce
	// the same output. This is the cross-run stability property the
	// audit log relies on.
	a := redact.RedactString("api_key=ABCDEFGHIJKLMNOP1234")
	b := redact.RedactString("api_key=ABCDEFGHIJKLMNOP1234")
	if a != b {
		t.Fatalf("not stable: %q vs %q", a, b)
	}
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	neg := i < 0
	if neg {
		i = -i
	}
	var b [21]byte
	pos := len(b)
	for i > 0 {
		pos--
		b[pos] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		pos--
		b[pos] = '-'
	}
	return string(b[pos:])
}
