package planner

import (
	"testing"

	"github.com/Hamidullah02/aios/aios-system/internal/core/schemas"
)

func goodPlan() *schemas.TaskPlan {
	return &schemas.TaskPlan{
		ID:   "p1",
		Goal: "ship aios",
		Tasks: []schemas.TaskSpec{
			{
				ID:          "t1",
				Description: "read the file",
				Tool:        "read_file",
				Acceptance:  "file contents returned; lines=3",
			},
		},
		Tools: []string{"read_file"},
		Model: "gpt-4o",
	}
}

func toolsForReadAndEdit() map[string]ToolSpec {
	return map[string]ToolSpec{
		"read_file": {Name: "read_file", Description: "read", RiskTier: 0, InputSchema: "{}"},
		"edit_file": {Name: "edit_file", Description: "edit", RiskTier: 1, InputSchema: "{}"},
	}
}

func TestVerifier_GoodPlan(t *testing.T) {
	v := NewCodeFirstVerifier(toolsForReadAndEdit())
	errs := v.Verify(goodPlan())
	if len(errs) != 0 {
		t.Fatalf("expected no errors, got: %+v", errs)
	}
}

func TestVerifier_MissingID(t *testing.T) {
	p := goodPlan()
	p.ID = ""
	v := NewCodeFirstVerifier(toolsForReadAndEdit())
	errs := v.Verify(p)
	if !hasCode(errs, CodeSchema) {
		t.Fatalf("want schema error, got: %+v", errs)
	}
}

func TestVerifier_MissingGoal(t *testing.T) {
	p := goodPlan()
	p.Goal = ""
	v := NewCodeFirstVerifier(toolsForReadAndEdit())
	errs := v.Verify(p)
	if !hasCode(errs, CodeSchema) {
		t.Fatalf("want schema error, got: %+v", errs)
	}
}

func TestVerifier_NoTasks(t *testing.T) {
	p := goodPlan()
	p.Tasks = nil
	v := NewCodeFirstVerifier(toolsForReadAndEdit())
	errs := v.Verify(p)
	if !hasCode(errs, CodeSchema) {
		t.Fatalf("want schema error, got: %+v", errs)
	}
}

func TestVerifier_BlankToolName(t *testing.T) {
	p := goodPlan()
	p.Tasks[0].Tool = "   "
	v := NewCodeFirstVerifier(toolsForReadAndEdit())
	errs := v.Verify(p)
	if !hasCode(errs, CodeToolCallSchema) {
		t.Fatalf("want tool_call_schema error, got: %+v", errs)
	}
}

func TestVerifier_BlankInput(t *testing.T) {
	p := goodPlan()
	p.Tasks[0].Inputs = []string{"first", "  "}
	v := NewCodeFirstVerifier(toolsForReadAndEdit())
	errs := v.Verify(p)
	if !hasCode(errs, CodeToolCallSchema) {
		t.Fatalf("want tool_call_schema error, got: %+v", errs)
	}
}

func TestVerifier_UnknownTool(t *testing.T) {
	p := goodPlan()
	p.Tasks[0].Tool = "rm_rf" // not in the map
	v := NewCodeFirstVerifier(toolsForReadAndEdit())
	errs := v.Verify(p)
	if !hasCode(errs, CodeUnknownTool) {
		t.Fatalf("want unknown_tool error, got: %+v", errs)
	}
}

func TestVerifier_UnknownTool_EmptyMapIsPermissive(t *testing.T) {
	p := goodPlan()
	p.Tasks[0].Tool = "anything" // not in the map, but map is empty
	v := NewCodeFirstVerifier(nil)
	errs := v.Verify(p)
	if hasCode(errs, CodeUnknownTool) {
		t.Fatalf("empty tool map should be permissive: %+v", errs)
	}
}

func TestVerifier_DiffScope(t *testing.T) {
	p := goodPlan()
	// One task with lines=250, default limit is 200.
	p.Tasks[0].Acceptance = "tests pass; lines=250"
	v := NewCodeFirstVerifier(toolsForReadAndEdit())
	errs := v.Verify(p)
	if !hasCode(errs, CodeDiffScope) {
		t.Fatalf("want diff_scope error, got: %+v", errs)
	}
}

func TestVerifier_DiffScope_Acceptable(t *testing.T) {
	p := goodPlan()
	p.Tasks[0].Acceptance = "tests pass; lines=120"
	v := NewCodeFirstVerifier(toolsForReadAndEdit())
	errs := v.Verify(p)
	if hasCode(errs, CodeDiffScope) {
		t.Fatalf("under limit should pass: %+v", errs)
	}
}

func TestVerifier_DiffScope_SumAcrossTasks(t *testing.T) {
	p := goodPlan()
	p.Tasks = []schemas.TaskSpec{
		{ID: "t1", Description: "a", Tool: "read_file", Acceptance: "ok; lines=150"},
		{ID: "t2", Description: "b", Tool: "edit_file", Acceptance: "ok; lines=80"},
	}
	v := NewCodeFirstVerifier(toolsForReadAndEdit())
	errs := v.Verify(p)
	if !hasCode(errs, CodeDiffScope) {
		t.Fatalf("sum over limit should reject: %+v", errs)
	}
}

func TestVerifier_DiffScope_ZeroLimitSkipsCheck(t *testing.T) {
	p := goodPlan()
	p.Tasks[0].Acceptance = "ok; lines=999999"
	v := NewCodeFirstVerifier(toolsForReadAndEdit()).WithDiffScopeLimit(0)
	errs := v.Verify(p)
	if hasCode(errs, CodeDiffScope) {
		t.Fatalf("zero limit should skip: %+v", errs)
	}
}

func TestVerifier_DiffScope_NegativeHint(t *testing.T) {
	p := goodPlan()
	p.Tasks[0].Acceptance = "ok; lines=-5"
	v := NewCodeFirstVerifier(toolsForReadAndEdit())
	errs := v.Verify(p)
	if !hasCode(errs, CodeDiffScope) {
		t.Fatalf("negative hint should reject: %+v", errs)
	}
}

func TestVerifier_DiffScope_NoHintInAcceptance(t *testing.T) {
	p := goodPlan()
	p.Tasks[0].Acceptance = "tests pass"
	v := NewCodeFirstVerifier(toolsForReadAndEdit())
	errs := v.Verify(p)
	if hasCode(errs, CodeDiffScope) {
		t.Fatalf("no hint should pass: %+v", errs)
	}
}

func TestVerifier_NilPlan(t *testing.T) {
	v := NewCodeFirstVerifier(toolsForReadAndEdit())
	errs := v.Verify(nil)
	if !hasCode(errs, CodeSchema) {
		t.Fatalf("nil plan should produce schema error: %+v", errs)
	}
}

func TestVerifier_CustomRule(t *testing.T) {
	p := goodPlan()
	// Custom rule: reject any plan whose goal contains "ship".
	v := NewCodeFirstVerifier(toolsForReadAndEdit()).WithRule(func(plan *schemas.TaskPlan, out PlanErrorList) PlanErrorList {
		if plan != nil && contains(plan.Goal, "ship") {
			out = append(out, PlanError{Code: CodeCustomRuleFailed, Message: "no shipping"})
		}
		return out
	})
	errs := v.Verify(p)
	if !hasCode(errs, CodeCustomRuleFailed) {
		t.Fatalf("custom rule should fire: %+v", errs)
	}
}

func TestParseLinesHint(t *testing.T) {
	cases := []struct {
		s    string
		want int
		ok   bool
	}{
		{"lines=12", 12, true},
		{"tests pass; lines=12", 12, true},
		{"lines=0", 0, true},
		{"lines=999999", 999999, true},
		{"lines=", 0, false},
		{"lines=abc", 0, false},
		{"no hint", 0, false},
		{"", 0, false},
		{"lines=-5", 0, false}, // negative digits start with '-'
		{"lines=12 more stuff", 12, true},
	}
	for _, c := range cases {
		n, ok := parseLinesHint(c.s)
		if ok != c.ok || n != c.want {
			t.Errorf("parseLinesHint(%q) = (%d, %v), want (%d, %v)", c.s, n, ok, c.want, c.ok)
		}
	}
}

func TestLooksBlank(t *testing.T) {
	cases := map[string]bool{
		"":     true,
		"  ":   true,
		"\t\n": true,
		"a":    false,
		" a ":  false,
	}
	for in, want := range cases {
		if got := looksBlank(in); got != want {
			t.Errorf("looksBlank(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestEncodePlan(t *testing.T) {
	s, err := EncodePlan(goodPlan())
	if err != nil {
		t.Fatal(err)
	}
	if s == "" {
		t.Fatal("non-empty")
	}
	// Round-trip: encoding nil returns "", nil.
	s, err = EncodePlan(nil)
	if err != nil || s != "" {
		t.Fatalf("nil plan: (%q, %v)", s, err)
	}
}

// hasCode reports whether errs contains a PlanError with the given code.
func hasCode(errs PlanErrorList, code string) bool {
	for _, e := range errs {
		if e.Code == code {
			return true
		}
	}
	return false
}
