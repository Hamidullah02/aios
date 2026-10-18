package policy

import (
	"testing"
)

func ptrInt(n int) *int { return &n }

func TestAction_IsValid(t *testing.T) {
	cases := map[Action]bool{
		ActionAllow: true,
		ActionDeny:  true,
		ActionAsk:   true,
		"":          false,
		"maybe":     false,
	}
	for a, want := range cases {
		if got := a.IsValid(); got != want {
			t.Errorf("Action(%q).IsValid() = %v, want %v", a, got, want)
		}
	}
}

func TestMatch_HasAny(t *testing.T) {
	if (&Match{}).HasAny() {
		t.Error("empty match should have no conditions")
	}
	if !(Match{Tool: "x"}).HasAny() {
		t.Error("tool set should have a condition")
	}
}

func TestRule_Valid_Allow(t *testing.T) {
	r := Rule{ID: "r1", Match: Match{Tool: "read"}, Action: ActionAllow}
	if err := r.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestRule_Valid_Ask(t *testing.T) {
	r := Rule{
		ID:     "r2",
		Match:  Match{ToolGlob: "write_*"},
		Action: ActionAsk,
		Approval: &ApprovalTemplate{
			Action:   "write",
			Resource: "{path}",
			Reason:   "write to a project file",
		},
	}
	if err := r.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestRule_MissingID(t *testing.T) {
	r := Rule{Match: Match{Tool: "x"}, Action: ActionAllow}
	err := r.Validate()
	if err == nil {
		t.Fatal("expected error")
	}
	if _, ok := err.(*RuleValidationError); !ok {
		t.Fatalf("expected *RuleValidationError, got %T", err)
	}
}

func TestRule_BadAction(t *testing.T) {
	r := Rule{ID: "r", Action: "maybe"}
	if err := r.Validate(); err == nil {
		t.Fatal("expected error")
	}
}

func TestRule_AskWithoutApproval(t *testing.T) {
	r := Rule{ID: "r", Action: ActionAsk}
	if err := r.Validate(); err == nil {
		t.Fatal("expected error: ask requires approval")
	}
}

func TestRule_AllowWithApproval(t *testing.T) {
	r := Rule{
		ID:       "r",
		Action:   ActionAllow,
		Approval: &ApprovalTemplate{Action: "x", Resource: "y", Reason: "z"},
	}
	if err := r.Validate(); err == nil {
		t.Fatal("expected error: approval only for ask")
	}
}

func TestRule_AskMissingApprovalField(t *testing.T) {
	cases := []struct {
		name string
		ap   ApprovalTemplate
	}{
		{"no action", ApprovalTemplate{Resource: "r", Reason: "b"}},
		{"no resource", ApprovalTemplate{Action: "a", Reason: "b"}},
		{"no reason", ApprovalTemplate{Action: "a", Resource: "r"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := Rule{ID: "r", Action: ActionAsk, Approval: &c.ap}
			if err := r.Validate(); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestValidateAll_DuplicateID(t *testing.T) {
	rules := []Rule{
		{ID: "r1", Match: Match{Tool: "a"}, Action: ActionAllow},
		{ID: "r1", Match: Match{Tool: "b"}, Action: ActionDeny},
	}
	if err := ValidateAll(rules); err == nil {
		t.Fatal("expected duplicate id error")
	}
}

func TestValidateAll_Empty(t *testing.T) {
	if err := ValidateAll(nil); err != nil {
		t.Fatal(err)
	}
	if err := ValidateAll([]Rule{}); err != nil {
		t.Fatal(err)
	}
}

func TestHash_Deterministic(t *testing.T) {
	r1 := []Rule{
		{ID: "r1", Match: Match{Tool: "a"}, Action: ActionAllow},
		{ID: "r2", Match: Match{Tool: "b"}, Action: ActionDeny},
	}
	r2 := []Rule{
		{ID: "r1", Match: Match{Tool: "a"}, Action: ActionAllow},
		{ID: "r2", Match: Match{Tool: "b"}, Action: ActionDeny},
	}
	if Hash(r1) != Hash(r2) {
		t.Fatal("same rules should hash equally")
	}
}

func TestHash_OrderMatters(t *testing.T) {
	a := []Rule{
		{ID: "r1", Match: Match{Tool: "a"}, Action: ActionAllow},
		{ID: "r2", Match: Match{Tool: "b"}, Action: ActionDeny},
	}
	b := []Rule{
		{ID: "r2", Match: Match{Tool: "b"}, Action: ActionDeny},
		{ID: "r1", Match: Match{Tool: "a"}, Action: ActionAllow},
	}
	if Hash(a) == Hash(b) {
		t.Fatal("different order should hash differently")
	}
}

func TestDecision_ToMap(t *testing.T) {
	d := &Decision{Action: ActionDeny, Reason: "x", RuleID: "r1"}
	m := d.ToMap()
	if m["action"] != "deny" {
		t.Fatalf("action: %v", m["action"])
	}
	if m["rule_id"] != "r1" {
		t.Fatalf("rule_id: %v", m["rule_id"])
	}
	if _, ok := m["approval"]; ok {
		t.Fatal("approval should be omitted when nil")
	}
}

func ptrMatchTier(t *testing.T) {
	m := Match{Tier: ptrInt(2)}
	if !m.HasAny() {
		t.Fatal("tier should count as a set condition")
	}
}
