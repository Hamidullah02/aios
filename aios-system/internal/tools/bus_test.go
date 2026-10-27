package tools

import (
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Hamidullah02/aios/aios-system/internal/core/audit"
	"github.com/Hamidullah02/aios/aios-system/internal/core/clock"
	"github.com/Hamidullah02/aios/aios-system/internal/policy"
)

// countingTool is a Tool for tests. It returns the args
// unchanged as the result, and increments a counter so
// tests can confirm the tool ran.
type countingTool struct {
	count *atomic.Int32
}

func (c countingTool) Name() string { return "ok" }
func (c countingTool) Run(args map[string]any, _ Context) (Result, error) {
	c.count.Add(1)
	return Result{OK: true, Output: args, Summary: "ok"}, nil
}

// newAudit opens an in-memory audit log.
func newAudit(t *testing.T) *audit.Log {
	t.Helper()
	log, err := audit.Open(audit.Config{
		Path:  ":memory:",
		Clock: clock.FixedDecision{T: time.Unix(1700000000, 0)},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = log.Close() })
	return log
}

func TestBus_Dispatch_Happy(t *testing.T) {
	cnt := &atomic.Int32{}
	bus := New(Config{Tools: map[string]Tool{"ok": countingTool{count: cnt}}})
	res, err := bus.Dispatch("ok", map[string]any{"x": 1}, Context{Actor: "a", TaskID: "t1"})
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK {
		t.Fatal("OK = false")
	}
	if cnt.Load() != 1 {
		t.Fatalf("count = %d", cnt.Load())
	}
}

func TestBus_ToolNotFound(t *testing.T) {
	bus := New(Config{Tools: map[string]Tool{}})
	_, err := bus.Dispatch("nope", nil, Context{})
	var tnf *ToolNotFoundError
	if !errors.As(err, &tnf) {
		t.Fatalf("err = %v, want *ToolNotFoundError", err)
	}
	if tnf.Name != "nope" {
		t.Fatalf("name = %s", tnf.Name)
	}
}

func TestBus_PolicyDeny(t *testing.T) {
	// No rules — default deny.
	gw := policy.New(nil, nil)
	cnt := &atomic.Int32{}
	bus := New(Config{
		Tools:  map[string]Tool{"ok": countingTool{count: cnt}},
		Policy: gw,
		Audit:  newAudit(t),
	})
	_, err := bus.Dispatch("ok", nil, Context{Actor: "a", TaskID: "t1"})
	if err == nil || !strings.Contains(err.Error(), "denied by policy") {
		t.Fatalf("err = %v", err)
	}
	if cnt.Load() != 0 {
		t.Fatalf("count = %d (tool ran on deny)", cnt.Load())
	}
}

func TestBus_PolicyAllow_PlusAuditRow(t *testing.T) {
	gw := policy.New([]policy.Rule{
		{ID: "t", Match: policy.Match{Tool: "ok"}, Action: policy.ActionAllow},
	}, nil)
	log := newAudit(t)
	cnt := &atomic.Int32{}
	bus := New(Config{
		Tools:  map[string]Tool{"ok": countingTool{count: cnt}},
		Policy: gw,
		Audit:  log,
	})
	_, err := bus.Dispatch("ok", map[string]any{"x": "y"}, Context{Actor: "a", TaskID: "task-1"})
	if err != nil {
		t.Fatal(err)
	}
	rows, _ := log.GetByTask("task-1")
	if len(rows) != 1 {
		t.Fatalf("rows = %d", len(rows))
	}
	if rows[0].Kind != "tool_call" {
		t.Fatalf("kind = %s", rows[0].Kind)
	}
}

func TestBus_PolicyAsk(t *testing.T) {
	gw := policy.New([]policy.Rule{
		{
			ID:     "ask-w",
			Match:  policy.Match{Tool: "ok"},
			Action: policy.ActionAsk,
			Approval: &policy.ApprovalTemplate{
				Action:   "write",
				Resource: "mem",
				Reason:   "ask",
			},
		},
	}, nil)
	cnt := &atomic.Int32{}
	bus := New(Config{
		Tools:  map[string]Tool{"ok": countingTool{count: cnt}},
		Policy: gw,
		Audit:  newAudit(t),
	})
	_, err := bus.Dispatch("ok", nil, Context{})
	if err == nil || !strings.Contains(err.Error(), "requires approval") {
		t.Fatalf("err = %v", err)
	}
	if cnt.Load() != 0 {
		t.Fatalf("count = %d (tool ran on ask)", cnt.Load())
	}
}

func TestBus_NoPolicy_NoAudit_StillWorks(t *testing.T) {
	cnt := &atomic.Int32{}
	bus := New(Config{Tools: map[string]Tool{"ok": countingTool{count: cnt}}})
	if _, err := bus.Dispatch("ok", nil, Context{}); err != nil {
		t.Fatal(err)
	}
}
