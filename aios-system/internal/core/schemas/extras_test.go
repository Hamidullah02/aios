package schemas

import (
	"encoding/json"
	"strings"
	"testing"
)

// This file drives the UnmarshalJSON paths and the
// auxiliary Validate methods. The high-value targets
// are: (a) the strict-rejection on every type, and
// (b) the canonical sort. Round-trip tests already
// live in schemas_test.go.

// --- base.go helpers ---

func TestJoinValidationErrors_AllNil(t *testing.T) {
	if err := JoinValidationErrors(nil, nil); err != nil {
		t.Fatalf("all nil should be nil, got %v", err)
	}
}

func TestJoinValidationErrors_Single(t *testing.T) {
	e := NewValidationError("x", "y")
	if err := JoinValidationErrors(e); err != e {
		t.Fatalf("single should be returned as-is, got %v", err)
	}
}

func TestJoinValidationErrors_Multiple(t *testing.T) {
	e1 := NewValidationError("a", "1")
	e2 := NewValidationError("b", "2")
	got := JoinValidationErrors(e1, e2)
	if got == nil {
		t.Fatal("expected joined error")
	}
	if !strings.Contains(got.Error(), "a: 1") || !strings.Contains(got.Error(), "b: 2") {
		t.Fatalf("got %q", got.Error())
	}
}

func TestMarshalCanonical_NilAndValues(t *testing.T) {
	if b, err := MarshalCanonical(nil); err != nil || string(b) != "null" {
		t.Fatalf("nil = %q, err=%v", b, err)
	}
	if b, err := MarshalCanonical("hi"); err != nil || string(b) != `"hi"` {
		t.Fatalf("string = %q, err=%v", b, err)
	}
}

func TestMarshalCanonical_MapSortsKeys(t *testing.T) {
	m := map[string]any{"z": 1, "a": 2, "m": 3}
	a, _ := MarshalCanonical(m)
	b, _ := MarshalCanonical(m)
	if string(a) != string(b) {
		t.Fatalf("not stable: %q vs %q", a, b)
	}
	if !strings.HasPrefix(string(a), `{"a":`) {
		t.Fatalf("keys not sorted: %q", a)
	}
}

func TestMarshalCanonical_NestedSlice(t *testing.T) {
	v := map[string]any{"list": []any{3, 2, 1}}
	b, err := MarshalCanonical(v)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `{"list":[3,2,1]}` {
		t.Fatalf("got %q", b)
	}
}

func TestUnmarshalStrict_BadJSON(t *testing.T) {
	var v map[string]any
	err := UnmarshalStrict([]byte("not json"), &v, []string{"x"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "invalid JSON") {
		t.Fatalf("err = %v", err)
	}
}

func TestUnmarshalStrict_InvalidValue(t *testing.T) {
	var v struct {
		N int `json:"n"`
	}
	err := UnmarshalStrict([]byte(`{"n":"not int"}`), &v, []string{"n"})
	if err == nil {
		t.Fatal("expected type error")
	}
	if !strings.Contains(err.Error(), "invalid value") {
		t.Fatalf("err = %v", err)
	}
}

func TestUnmarshalStrict_UnknownFieldMessage(t *testing.T) {
	var v map[string]any
	err := UnmarshalStrict([]byte(`{"known":1,"evil":2}`), &v, []string{"known"})
	if err == nil {
		t.Fatal("expected unknown-field error")
	}
	if !strings.Contains(err.Error(), "evil") || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("err = %v", err)
	}
}

func TestUnmarshalStrict_AllKnown(t *testing.T) {
	var v map[string]any
	if err := UnmarshalStrict([]byte(`{"a":1,"b":2}`), &v, []string{"a", "b"}); err != nil {
		t.Fatal(err)
	}
}

// --- per-type UnmarshalJSON strict-rejection ---
// The strict-rejection is the heart of REQ-CORE-SCH-009:
// "The type rejects extra fields." We round-trip JSON
// with a single extra field and assert the error.

func TestAgentSpec_StrictRejectsUnknown(t *testing.T) {
	var v AgentSpec
	err := v.UnmarshalJSON([]byte(`{"id":"a","role":"r","tools":[],"model":"m","weird":1}`))
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "weird") {
		t.Fatalf("err = %v", err)
	}
}

func TestAgentSpec_UnmarshalHappy(t *testing.T) {
	var v AgentSpec
	if err := v.UnmarshalJSON([]byte(`{"id":"a","role":"r","tools":[],"model":"m"}`)); err != nil {
		t.Fatal(err)
	}
	if v.ID != "a" || v.Role != "r" {
		t.Fatalf("v = %+v", v)
	}
}

func TestApprovalPayload_StrictRejectsUnknown(t *testing.T) {
	var v ApprovalPayload
	err := v.UnmarshalJSON([]byte(`{"approval_id":"00000000-0000-0000-0000-000000000000","extra":1}`))
	if err == nil || !strings.Contains(err.Error(), "extra") {
		t.Fatalf("err = %v", err)
	}
}

func TestApprovalPayload_UnmarshalHappy(t *testing.T) {
	var v ApprovalPayload
	if err := v.UnmarshalJSON([]byte(`{"approval_id":"00000000-0000-0000-0000-000000000000"}`)); err != nil {
		t.Fatal(err)
	}
}

func TestDecisionTrace_StrictRejectsUnknown(t *testing.T) {
	var v DecisionTrace
	err := v.UnmarshalJSON([]byte(`{"step_id":"s","chosen":"c","alternatives":[],"basis":"b","weird":1}`))
	if err == nil || !strings.Contains(err.Error(), "weird") {
		t.Fatalf("err = %v", err)
	}
}

func TestDecisionTrace_UnmarshalHappy(t *testing.T) {
	var v DecisionTrace
	if err := v.UnmarshalJSON([]byte(`{"step_id":"s","chosen":"c","alternatives":[],"basis":"b"}`)); err != nil {
		t.Fatal(err)
	}
}

func TestAlternative_StrictRejectsUnknown(t *testing.T) {
	var v Alternative
	err := v.UnmarshalJSON([]byte(`{"action":"a","score":0.5,"rejected":"r","x":1}`))
	if err == nil || !strings.Contains(err.Error(), "x") {
		t.Fatalf("err = %v", err)
	}
}

func TestAlternative_UnmarshalHappy(t *testing.T) {
	var v Alternative
	if err := v.UnmarshalJSON([]byte(`{"action":"a","score":0.5,"rejected":"r"}`)); err != nil {
		t.Fatal(err)
	}
}

func TestEnvelope_StrictRejectsUnknown(t *testing.T) {
	var v Envelope
	err := v.UnmarshalJSON([]byte(`{"type":"x","source":"s","created_at":"2024-01-01T00:00:00Z","weird":1}`))
	if err == nil || !strings.Contains(err.Error(), "weird") {
		t.Fatalf("err = %v", err)
	}
}

func TestEnvelope_UnmarshalHappy(t *testing.T) {
	var v Envelope
	if err := v.UnmarshalJSON([]byte(`{"type":"x","source":"s","created_at":"2024-01-01T00:00:00Z"}`)); err != nil {
		t.Fatal(err)
	}
}

func TestModelCapability_StrictRejectsUnknown(t *testing.T) {
	var v ModelCapability
	err := v.UnmarshalJSON([]byte(`{"model_id":"m","class":"cheap","supports":[],"x":1}`))
	if err == nil || !strings.Contains(err.Error(), "x") {
		t.Fatalf("err = %v", err)
	}
}

func TestModelCapability_UnmarshalHappy(t *testing.T) {
	var v ModelCapability
	if err := v.UnmarshalJSON([]byte(`{"model_id":"m","class":"cheap","supports":[]}`)); err != nil {
		t.Fatal(err)
	}
}

func TestScheduleSpec_StrictRejectsUnknown(t *testing.T) {
	var v ScheduleSpec
	err := v.UnmarshalJSON([]byte(`{"id":"s","trigger":{"kind":"once","at":"2024-01-01T00:00:00Z"},"workflow":{"id":"w","steps":[],"transitions":[]},"limits":{},"x":1}`))
	if err == nil || !strings.Contains(err.Error(), "x") {
		t.Fatalf("err = %v", err)
	}
}

func TestScheduleSpec_UnmarshalHappy(t *testing.T) {
	var v ScheduleSpec
	if err := v.UnmarshalJSON([]byte(`{"id":"s","trigger":{"kind":"once","at":"2024-01-01T00:00:00Z"},"workflow":{"id":"w","steps":[],"transitions":[]},"limits":{}}`)); err != nil {
		t.Fatal(err)
	}
}

func TestScheduleTrigger_StrictRejectsUnknown(t *testing.T) {
	var v ScheduleTrigger
	err := v.UnmarshalJSON([]byte(`{"kind":"once","at":"x","y":1}`))
	if err == nil || !strings.Contains(err.Error(), "y") {
		t.Fatalf("err = %v", err)
	}
}

func TestScheduleTrigger_UnmarshalHappy(t *testing.T) {
	var v ScheduleTrigger
	if err := v.UnmarshalJSON([]byte(`{"kind":"once","at":"x"}`)); err != nil {
		t.Fatal(err)
	}
}

func TestScheduleLimits_StrictRejectsUnknown(t *testing.T) {
	var v ScheduleLimits
	err := v.UnmarshalJSON([]byte(`{"max_runs":1,"x":1}`))
	if err == nil || !strings.Contains(err.Error(), "x") {
		t.Fatalf("err = %v", err)
	}
}

func TestScheduleLimits_UnmarshalHappy(t *testing.T) {
	var v ScheduleLimits
	if err := v.UnmarshalJSON([]byte(`{"max_runs":1}`)); err != nil {
		t.Fatal(err)
	}
}

func TestTaskSpec_StrictRejectsUnknown(t *testing.T) {
	var v TaskSpec
	err := v.UnmarshalJSON([]byte(`{"id":"t","description":"d","acceptance":"a","x":1}`))
	if err == nil || !strings.Contains(err.Error(), "x") {
		t.Fatalf("err = %v", err)
	}
}

func TestTaskSpec_UnmarshalHappy(t *testing.T) {
	var v TaskSpec
	if err := v.UnmarshalJSON([]byte(`{"id":"t","description":"d","acceptance":"a"}`)); err != nil {
		t.Fatal(err)
	}
}

func TestTaskSpec_Validate_MissingFields(t *testing.T) {
	if err := (&TaskSpec{}).Validate(); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestBudget_Validate(t *testing.T) {
	if err := (&Budget{}).Validate(); err != nil {
		t.Fatalf("zero budget should be valid: %v", err)
	}
	if err := (&Budget{}).Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestToolCall_StrictRejectsUnknown(t *testing.T) {
	var v ToolCall
	err := v.UnmarshalJSON([]byte(`{"call_id":"c","tool":"t","args":{},"risk_tier":0,"x":1}`))
	if err == nil || !strings.Contains(err.Error(), "x") {
		t.Fatalf("err = %v", err)
	}
}

func TestToolCall_UnmarshalHappy(t *testing.T) {
	var v ToolCall
	if err := v.UnmarshalJSON([]byte(`{"call_id":"c","tool":"t","args":{},"risk_tier":0}`)); err != nil {
		t.Fatal(err)
	}
}

func TestToolResult_StrictRejectsUnknown(t *testing.T) {
	var v ToolResult
	err := v.UnmarshalJSON([]byte(`{"call_id":"c","ok":true,"duration_ms":0,"x":1}`))
	if err == nil || !strings.Contains(err.Error(), "x") {
		t.Fatalf("err = %v", err)
	}
}

func TestToolResult_UnmarshalHappy(t *testing.T) {
	var v ToolResult
	if err := v.UnmarshalJSON([]byte(`{"call_id":"c","ok":true,"duration_ms":0}`)); err != nil {
		t.Fatal(err)
	}
}

func TestWorkflowSpec_StrictRejectsUnknown(t *testing.T) {
	var v WorkflowSpec
	err := v.UnmarshalJSON([]byte(`{"id":"w","name":"n","steps":[],"transitions":[],"x":1}`))
	if err == nil || !strings.Contains(err.Error(), "x") {
		t.Fatalf("err = %v", err)
	}
}

func TestWorkflowSpec_UnmarshalHappy(t *testing.T) {
	var v WorkflowSpec
	if err := v.UnmarshalJSON([]byte(`{"id":"w","name":"n","steps":[],"transitions":[]}`)); err != nil {
		t.Fatal(err)
	}
}

func TestWorkflowStep_StrictRejectsUnknown(t *testing.T) {
	var v WorkflowStep
	err := v.UnmarshalJSON([]byte(`{"id":"s","tool":"t","x":1}`))
	if err == nil || !strings.Contains(err.Error(), "x") {
		t.Fatalf("err = %v", err)
	}
}

func TestWorkflowStep_UnmarshalHappy(t *testing.T) {
	var v WorkflowStep
	if err := v.UnmarshalJSON([]byte(`{"id":"s","tool":"t"}`)); err != nil {
		t.Fatal(err)
	}
}

func TestWorkflowTransition_StrictRejectsUnknown(t *testing.T) {
	var v WorkflowTransition
	err := v.UnmarshalJSON([]byte(`{"from":"a","to":"b","when":"x","y":1}`))
	if err == nil || !strings.Contains(err.Error(), "y") {
		t.Fatalf("err = %v", err)
	}
}

func TestWorkflowTransition_UnmarshalHappy(t *testing.T) {
	var v WorkflowTransition
	if err := v.UnmarshalJSON([]byte(`{"from":"a","to":"b","when":"x"}`)); err != nil {
		t.Fatal(err)
	}
}

// BoundTo.Validate is a marker that always returns nil;
// the type's only contract is well-formed JSON (handled
// by UnmarshalJSON). The Match method enforces the
// semantic constraints.
func TestBoundTo_Validate_AlwaysNil(t *testing.T) {
	if err := (*BoundTo)(nil).Validate(); err != nil {
		t.Fatalf("nil BoundTo: %v", err)
	}
	if err := (&BoundTo{}).Validate(); err != nil {
		t.Fatalf("empty BoundTo: %v", err)
	}
	if err := (&BoundTo{Task: "t", Tool: "edit", Actor: "a", Resource: "r"}).Validate(); err != nil {
		t.Fatal(err)
	}
}

// --- validators in validate.go ---

func TestOptionalString_EmptyAllowed(t *testing.T) {
	if err := OptionalString("x", ""); err != nil {
		t.Fatal("empty should be valid")
	}
}

func TestListOfStrings_BadElement(t *testing.T) {
	if err := ListOfStrings("x", []string{"a", "", "b"}); err == nil {
		t.Fatal("empty element should fail")
	}
}

func TestEmail_EdgeCases(t *testing.T) {
	for _, c := range []struct {
		in   string
		want bool
	}{
		// Empty is allowed (Optional semantics).
		{"", true},
		// RFC-valid forms.
		{"a@b.c", true},
		{"a@b", true}, // net.ParseAddress accepts single-label hosts
		// Rejected.
		{"no-at", false},
		{"@b.c", false},
		{"a@", false},
	} {
		err := Email("e", c.in)
		got := err == nil
		if got != c.want {
			t.Errorf("Email(%q) ok=%v, want %v (err=%v)", c.in, got, c.want, err)
		}
	}
}

func TestUUIDv4_EdgeCases(t *testing.T) {
	for _, c := range []struct {
		in   string
		want bool
	}{
		// Rejected.
		{"", false},
		{"00000000-0000-0000-0000-00000000000", false},   // too short
		{"00000000-0000-0000-0000-0000000000000", false}, // too long
		{"not-a-uuid", false},
		{"00000000-0000-0000-0000-000000000000", false}, // version 0, not 4
		// Accepted: version 4 (the "4" in 3rd group) and
		// variant 8/9/a/b in 4th group.
		{"12345678-1234-4234-8234-123456789abc", true},
	} {
		err := UUIDv4("u", c.in)
		got := err == nil
		if got != c.want {
			t.Errorf("UUIDv4(%q) ok=%v, want %v (err=%v)", c.in, got, c.want, err)
		}
	}
}

func TestMinLen_Boundary(t *testing.T) {
	if err := MinLen("x", "ab", 3); err == nil {
		t.Fatal("len 2 < 3 should fail")
	}
	if err := MinLen("x", "abc", 3); err != nil {
		t.Fatal("len 3 == 3 should pass")
	}
}

// keep the json import alive in case future tests need it
var _ = json.Marshal
