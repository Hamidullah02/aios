package planner

import (
	"errors"
	"testing"
)

func TestPlanError_Error(t *testing.T) {
	e := &PlanError{Code: CodeSchema, Field: "goal", Message: "required"}
	got := e.Error()
	if got == "" {
		t.Fatal("non-empty")
	}
	if !contains(got, "schema") || !contains(got, "goal") {
		t.Errorf("got %q", got)
	}
}

func TestPlanError_NoField(t *testing.T) {
	e := &PlanError{Code: CodeSchema, Message: "x"}
	got := e.Error()
	if got == "" {
		t.Fatal("non-empty")
	}
}

func TestPlanError_NilSafe(t *testing.T) {
	var e *PlanError
	if e.Error() == "" {
		t.Fatal("nil plan error should still produce a string")
	}
}

func TestPlanValidationError_FormatsList(t *testing.T) {
	e := &PlanValidationError{
		Errors: PlanErrorList{
			{Code: CodeSchema, Field: "goal", Message: "required"},
			{Code: CodeToolCallSchema, Field: "tasks.0.tool", Message: "blank"},
		},
	}
	got := e.Error()
	if got == "" {
		t.Fatal("non-empty")
	}
	if !contains(got, "schema") {
		t.Errorf("got %q", got)
	}
}

func TestPlanValidationError_EmptyList(t *testing.T) {
	e := &PlanValidationError{}
	if e.Error() == "" {
		t.Fatal("non-empty even when no errors")
	}
}

func TestPlanGenerationError_WithCause(t *testing.T) {
	cause := errors.New("router failed")
	e := &PlanGenerationError{Iterations: 3, Cause: cause}
	if !errors.Is(e, cause) {
		t.Fatal("Unwrap should expose the cause")
	}
	if e.Error() == "" {
		t.Fatal("non-empty")
	}
}

func TestPlanGenerationError_WithLastErrors(t *testing.T) {
	e := &PlanGenerationError{
		Iterations: 4,
		LastErrors: PlanErrorList{{Code: CodeDiffScope, Message: "x"}},
	}
	got := e.Error()
	if got == "" {
		t.Fatal("non-empty")
	}
	if !contains(got, "diff_scope") {
		t.Errorf("got %q", got)
	}
}

func TestPlanGenerationError_Plain(t *testing.T) {
	e := &PlanGenerationError{Iterations: 1}
	if e.Error() == "" {
		t.Fatal("non-empty")
	}
}

func TestMarshalErrors(t *testing.T) {
	cases := []struct {
		name string
		in   PlanErrorList
		want string
	}{
		{"nil", nil, "[]"},
		{"empty", PlanErrorList{}, "[]"},
		{"one", PlanErrorList{{Code: CodeSchema, Message: "x"}}, `[{"code":"schema","message":"x"}]`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := marshalErrors(c.in)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != c.want {
				t.Errorf("got %q, want %q", string(got), c.want)
			}
		})
	}
}

func TestJoinStrings(t *testing.T) {
	if got := joinStrings(nil, ";"); got != "" {
		t.Errorf("nil: %q", got)
	}
	if got := joinStrings([]string{"a"}, ";"); got != "a" {
		t.Errorf("one: %q", got)
	}
	if got := joinStrings([]string{"a", "b", "c"}, ";"); got != "a;b;c" {
		t.Errorf("many: %q", got)
	}
	// Single-element slice that contains a comma is returned as-is
	// (the comma is part of the element, not a separator).
	if got := joinStrings([]string{"a,b,c"}, ";"); got != "a,b,c" {
		t.Errorf("single-element comma: %q", got)
	}
}

func contains(s, sub string) bool {
	if len(sub) == 0 {
		return true
	}
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
