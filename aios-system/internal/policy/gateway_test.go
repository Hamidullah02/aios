package policy

import (
	"testing"
	"time"

	"github.com/sadi/aios-system/internal/core/audit"
	"github.com/sadi/aios-system/internal/core/clock"
)

func newTestGateway(t *testing.T, rules []Rule) (*Gateway, *audit.Log) {
	t.Helper()
	log, err := audit.Open(audit.Config{
		Path:  ":memory:",
		Clock: clock.FixedDecision{T: time.Unix(1700000000, 0)},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { log.Close() })
	return New(rules, log), log
}

func TestDecide_Allow(t *testing.T) {
	gw, _ := newTestGateway(t, []Rule{
		{ID: "r1", Match: Match{Tool: "read"}, Action: ActionAllow},
	})
	d, err := gw.Decide(Call{Tool: "read", TaskID: "t1", Actor: "op"})
	if err != nil {
		t.Fatal(err)
	}
	if d.Action != ActionAllow || d.RuleID != "r1" {
		t.Fatalf("got %+v", d)
	}
}

func TestDecide_Deny(t *testing.T) {
	gw, _ := newTestGateway(t, []Rule{
		{ID: "r1", Match: Match{Tool: "read"}, Action: ActionAllow},
		{ID: "r2", Match: Match{PathPrefix: "/etc/"}, Action: ActionDeny},
	})
	d, err := gw.Decide(Call{Tool: "edit", Path: "/etc/passwd", TaskID: "t1", Actor: "op"})
	if err != nil {
		t.Fatal(err)
	}
	if d.Action != ActionDeny || d.RuleID != "r2" {
		t.Fatalf("got %+v", d)
	}
}

func TestDecide_DefaultDeny(t *testing.T) {
	gw, _ := newTestGateway(t, []Rule{
		{ID: "r1", Match: Match{Tool: "read"}, Action: ActionAllow},
	})
	d, err := gw.Decide(Call{Tool: "edit", TaskID: "t1", Actor: "op"})
	if err != nil {
		t.Fatal(err)
	}
	if d.Action != ActionDeny {
		t.Fatalf("action: %s", d.Action)
	}
	if d.RuleID != "" {
		t.Fatalf("rule_id should be empty for default deny: %q", d.RuleID)
	}
	if d.Reason == "" {
		t.Fatal("reason should be set")
	}
}

func TestDecide_FirstMatchWins(t *testing.T) {
	gw, _ := newTestGateway(t, []Rule{
		{ID: "r1", Match: Match{Tool: "edit"}, Action: ActionAllow},
		{ID: "r2", Match: Match{Tool: "edit"}, Action: ActionDeny},
	})
	d, _ := gw.Decide(Call{Tool: "edit", TaskID: "t1", Actor: "op"})
	if d == nil {
		t.Fatal("nil decision")
	}
	if d.RuleID != "r1" {
		t.Fatalf("first match should win, got %q", d.RuleID)
	}
}

func TestDecide_AskWithApproval(t *testing.T) {
	gw, _ := newTestGateway(t, []Rule{
		{
			ID:     "r1",
			Match:  Match{ToolGlob: "write_*"},
			Action: ActionAsk,
			Approval: &ApprovalTemplate{
				Action:   "write",
				Resource: "{path}",
				Reason:   "Write to {path}",
				BoundTo:  map[string]string{"task": "{task}", "tool": "{tool}"},
			},
		},
	})
	c := Call{Tool: "write_file", Path: "/tmp/x", TaskID: "t1", Actor: "op:alice"}
	d, err := gw.Decide(c)
	if err != nil {
		t.Fatal(err)
	}
	if d.Action != ActionAsk {
		t.Fatalf("action: %s", d.Action)
	}
	if d.Approval == nil {
		t.Fatal("approval should be set")
	}
	if d.Approval.Action != "write" {
		t.Errorf("approval.action: %s", d.Approval.Action)
	}
	if d.Approval.Target != "/tmp/x" {
		t.Errorf("approval.target: %s", d.Approval.Target)
	}
	if d.Approval.Reason != "Write to /tmp/x" {
		t.Errorf("approval.reason: %s", d.Approval.Reason)
	}
	if d.Approval.ApprovalID == "" {
		t.Error("approval.approval_id should be a uuid")
	}
	if d.Approval.RequesterID != "op:alice" {
		t.Errorf("approval.requester_id: %s", d.Approval.RequesterID)
	}
}

func TestDecide_AskTemplateError(t *testing.T) {
	gw, _ := newTestGateway(t, []Rule{
		{
			ID:     "r1",
			Match:  Match{Tool: "edit"},
			Action: ActionAsk,
			Approval: &ApprovalTemplate{
				Action:   "write",
				Resource: "{unknown}",
				Reason:   "r",
			},
		},
	})
	_, err := gw.Decide(Call{Tool: "edit"})
	if err == nil {
		t.Fatal("expected template error")
	}
}

func TestDecide_AuditRowWritten(t *testing.T) {
	gw, log := newTestGateway(t, []Rule{
		{ID: "r1", Match: Match{Tool: "read"}, Action: ActionAllow},
	})
	if _, err := gw.Decide(Call{Tool: "read", TaskID: "t1", Actor: "op"}); err != nil {
		t.Fatal(err)
	}
	rows, err := log.GetByTask("t1")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows: %d", len(rows))
	}
	if rows[0].Kind != "policy_decision" {
		t.Fatalf("kind: %s", rows[0].Kind)
	}
}

func TestDecide_AuditRowForAsk(t *testing.T) {
	gw, log := newTestGateway(t, []Rule{
		{
			ID:     "r1",
			Match:  Match{Tool: "edit"},
			Action: ActionAsk,
			Approval: &ApprovalTemplate{
				Action: "edit", Resource: "{path}", Reason: "r",
			},
		},
	})
	d, err := gw.Decide(Call{Tool: "edit", TaskID: "t1", Path: "/tmp/x", Actor: "op"})
	if err != nil {
		t.Fatal(err)
	}
	rows, err := log.GetByApproval(d.Approval.ApprovalID)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows: %d", len(rows))
	}
}

func TestDecide_NoAudit(t *testing.T) {
	// Construct a gateway with a nil audit log; Decide should
	// still return a decision.
	gw := New([]Rule{
		{ID: "r1", Match: Match{Tool: "read"}, Action: ActionAllow},
	}, nil)
	d, err := gw.Decide(Call{Tool: "read"})
	if err != nil {
		t.Fatal(err)
	}
	if d.Action != ActionAllow {
		t.Fatalf("action: %s", d.Action)
	}
}

func TestNew_RulesCopied(t *testing.T) {
	src := []Rule{
		{ID: "r1", Match: Match{Tool: "read"}, Action: ActionAllow},
	}
	gw := New(src, nil)
	src[0].ID = "mutated"
	if gw.Rules()[0].ID != "r1" {
		t.Fatal("gateway should hold a copy")
	}
}

func TestGateway_RulesAndHash(t *testing.T) {
	rules := []Rule{
		{ID: "r1", Match: Match{Tool: "read"}, Action: ActionAllow},
	}
	gw := New(rules, nil)
	if got := gw.Rules(); len(got) != 1 || got[0].ID != "r1" {
		t.Fatalf("Rules: %+v", got)
	}
	if gw.Hash() == "" {
		t.Fatal("Hash should not be empty")
	}
	if gw.Hash() != Hash(rules) {
		t.Fatal("gateway hash should match Hash(rules)")
	}
}

func TestFlattenBoundTo(t *testing.T) {
	got := flattenBoundTo(map[string]string{
		"task": "t1",
		"tool": "edit",
	})
	// Keys are sorted for determinism (Scope is part of the
	// row's payload and therefore part of the audit content hash).
	if got != "task=t1;tool=edit" {
		t.Fatalf("got %q", got)
	}
}

func TestFlattenBoundTo_Empty(t *testing.T) {
	if got := flattenBoundTo(nil); got != "" {
		t.Fatalf("got %q", got)
	}
	if got := flattenBoundTo(map[string]string{}); got != "" {
		t.Fatalf("got %q", got)
	}
}
