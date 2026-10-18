package planner

import (
	"strings"
	"testing"
)

func TestRenderSystem(t *testing.T) {
	s := RenderSystem()
	if s == "" {
		t.Fatal("non-empty")
	}
	if !strings.Contains(s, "aios-planner") {
		t.Errorf("system prompt should identify the planner: %s", s)
	}
}

func TestRenderUser_WithGoalOnly(t *testing.T) {
	got, err := RenderUser(PromptData{Goal: "ship aios"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "ship aios") {
		t.Errorf("missing goal: %s", got)
	}
}

func TestRenderUser_WithContext(t *testing.T) {
	got, err := RenderUser(PromptData{
		Goal:    "ship aios",
		Context: map[string]any{"repo": "acme/aios", "branch": "main"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "repo: acme/aios") {
		t.Errorf("missing context entry: %s", got)
	}
}

func TestRenderUser_WithTools(t *testing.T) {
	got, err := RenderUser(PromptData{
		Goal: "ship aios",
		Tools: []ToolSpec{
			{Name: "read_file", Description: "read a file", RiskTier: 0, InputSchema: "{}"},
			{Name: "edit_file", Description: "edit a file", RiskTier: 1, InputSchema: "{}"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "read_file") || !strings.Contains(got, "edit_file") {
		t.Errorf("missing tool entries: %s", got)
	}
	// Tools should be sorted alphabetically. Ascending order
	// places "edit_file" before "read_file" (e < r).
	if strings.Index(got, "edit_file") > strings.Index(got, "read_file") {
		t.Errorf("tools not sorted: read_file appeared before edit_file")
	}
}

func TestRenderUser_DeterministicToolsOrder(t *testing.T) {
	mkTools := func() []ToolSpec {
		return []ToolSpec{
			{Name: "zeta", Description: "z", RiskTier: 0, InputSchema: "{}"},
			{Name: "alpha", Description: "a", RiskTier: 0, InputSchema: "{}"},
			{Name: "mu", Description: "m", RiskTier: 0, InputSchema: "{}"},
		}
	}
	a, err := RenderUser(PromptData{Goal: "g", Tools: mkTools()})
	if err != nil {
		t.Fatal(err)
	}
	b, err := RenderUser(PromptData{Goal: "g", Tools: mkTools()})
	if err != nil {
		t.Fatal(err)
	}
	if a != b {
		t.Errorf("not deterministic:\n%s\n---\n%s", a, b)
	}
}

func TestRenderRepair(t *testing.T) {
	got, err := RenderRepair(PromptData{
		Goal:     "ship aios",
		Previous: `{"id":"p1","goal":"ship aios","tasks":[]}`,
		Errors:   PlanErrorList{{Code: CodeSchema, Field: "tasks", Message: "required"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "ship aios") {
		t.Errorf("missing goal: %s", got)
	}
	if !strings.Contains(got, "schema") {
		t.Errorf("missing error code: %s", got)
	}
	if !strings.Contains(got, "previous plan") && !strings.Contains(got, "Previous plan") {
		t.Errorf("missing previous plan: %s", got)
	}
}

func TestRenderRepair_WithContext(t *testing.T) {
	got, err := RenderRepair(PromptData{
		Goal:    "g",
		Context: map[string]any{"k": "v"},
		Tools:   []ToolSpec{{Name: "t", Description: "d", RiskTier: 1, InputSchema: "{}"}},
		Errors:  PlanErrorList{{Code: CodeSchema, Message: "x"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "k: v") {
		t.Errorf("missing context in repair: %s", got)
	}
	if !strings.Contains(got, "t (risk_tier=1)") {
		t.Errorf("missing tool in repair: %s", got)
	}
}

func TestSortedTools(t *testing.T) {
	in := []ToolSpec{
		{Name: "b"}, {Name: "a"}, {Name: "c"},
	}
	out := sortedTools(in)
	if out[0].Name != "a" || out[1].Name != "b" || out[2].Name != "c" {
		t.Errorf("not sorted: %+v", out)
	}
	// Empty and single-element inputs are returned untouched.
	if sortedTools(nil) != nil {
		t.Error("nil should return nil")
	}
	if sortedTools([]ToolSpec{{Name: "x"}})[0].Name != "x" {
		t.Error("single element should be unchanged")
	}
}
