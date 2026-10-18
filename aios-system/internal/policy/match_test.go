package policy

import "testing"

func TestMatch_Tool(t *testing.T) {
	m := Match{Tool: "edit"}
	if !m.Matches(Call{Tool: "edit"}) {
		t.Fatal("exact tool should match")
	}
	if m.Matches(Call{Tool: "Edit"}) {
		t.Fatal("case-sensitive; should not match")
	}
}

func TestMatch_ToolGlob(t *testing.T) {
	m := Match{ToolGlob: "write_*"}
	if !m.Matches(Call{Tool: "write_file"}) {
		t.Fatal("glob should match")
	}
	if m.Matches(Call{Tool: "read_file"}) {
		t.Fatal("glob should not match different tool")
	}
}

func TestMatch_ToolGlob_Invalid(t *testing.T) {
	m := Match{ToolGlob: "["} // invalid glob
	if m.Matches(Call{Tool: "x"}) {
		t.Fatal("invalid glob should not match")
	}
}

func TestMatch_PathPrefix(t *testing.T) {
	m := Match{PathPrefix: "/etc/"}
	if !m.Matches(Call{Path: "/etc/passwd"}) {
		t.Fatal("prefix should match")
	}
	if m.Matches(Call{Path: "/var/log"}) {
		t.Fatal("prefix should not match different path")
	}
	if m.Matches(Call{Path: ""}) {
		t.Fatal("prefix should not match empty path")
	}
}

func TestMatch_PathGlob(t *testing.T) {
	m := Match{PathGlob: "/tmp/*.log"}
	if !m.Matches(Call{Path: "/tmp/x.log"}) {
		t.Fatal("glob should match")
	}
	if m.Matches(Call{Path: "/tmp/x.txt"}) {
		t.Fatal("glob should not match .txt")
	}
}

func TestMatch_HostGlob(t *testing.T) {
	m := Match{HostGlob: "*.example.com"}
	if !m.Matches(Call{Host: "api.example.com"}) {
		t.Fatal("glob should match")
	}
	if m.Matches(Call{Host: "example.com"}) {
		t.Fatal("glob should not match the apex")
	}
}

func TestMatch_Tier(t *testing.T) {
	m := Match{Tier: ptrInt(2)}
	if !m.Matches(Call{RiskTier: 2}) {
		t.Fatal("tier should match")
	}
	if m.Matches(Call{RiskTier: 3}) {
		t.Fatal("tier should not match different")
	}
}

func TestMatch_Actor(t *testing.T) {
	m := Match{Actor: "operator:alice"}
	if !m.Matches(Call{Actor: "operator:alice"}) {
		t.Fatal("actor should match")
	}
	if m.Matches(Call{Actor: "operator:bob"}) {
		t.Fatal("actor should not match different")
	}
}

func TestMatch_Task(t *testing.T) {
	m := Match{Task: "t1"}
	if !m.Matches(Call{TaskID: "t1"}) {
		t.Fatal("task should match")
	}
	if m.Matches(Call{TaskID: "t2"}) {
		t.Fatal("task should not match different")
	}
}

func TestMatch_EmptyMatchesEverything(t *testing.T) {
	if !(Match{}).Matches(Call{}) {
		t.Fatal("empty match should match everything")
	}
}

func TestMatch_Conjunction(t *testing.T) {
	m := Match{Tool: "edit", Tier: ptrInt(2)}
	if !m.Matches(Call{Tool: "edit", RiskTier: 2}) {
		t.Fatal("both conditions met")
	}
	if m.Matches(Call{Tool: "edit", RiskTier: 3}) {
		t.Fatal("tier mismatch")
	}
	if m.Matches(Call{Tool: "read", RiskTier: 2}) {
		t.Fatal("tool mismatch")
	}
}

func TestNewCall_ExtractsPathHostTier(t *testing.T) {
	c := NewCall("edit", map[string]any{
		"path": "/tmp/x",
		"host": "api.example.com",
		"tier": 2,
	}, "operator:alice", "t1")
	if c.Path != "/tmp/x" || c.Host != "api.example.com" || c.RiskTier != 2 {
		t.Fatalf("got %+v", c)
	}
}

func TestNewCall_TierFromInt64(t *testing.T) {
	c := NewCall("edit", map[string]any{"tier": int64(3)}, "op", "t")
	if c.RiskTier != 3 {
		t.Fatalf("tier: %d", c.RiskTier)
	}
}

func TestNewCall_TierFromFloat64(t *testing.T) {
	c := NewCall("edit", map[string]any{"tier": float64(3)}, "op", "t")
	if c.RiskTier != 3 {
		t.Fatalf("tier: %d", c.RiskTier)
	}
}

func TestNewCall_TierFromString(t *testing.T) {
	c := NewCall("edit", map[string]any{"tier": "3"}, "op", "t")
	if c.RiskTier != 3 {
		t.Fatalf("tier: %d", c.RiskTier)
	}
}

func TestNewCall_NoArgs(t *testing.T) {
	c := NewCall("edit", nil, "op", "t")
	if c.RiskTier != 0 || c.Path != "" || c.Host != "" {
		t.Fatalf("got %+v", c)
	}
}
