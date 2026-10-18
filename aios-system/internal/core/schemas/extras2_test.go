package schemas

import (
	"errors"
	"strings"
	"testing"
)

func TestAlternative_Validate(t *testing.T) {
	if err := (&Alternative{}).Validate(); err == nil {
		t.Fatal("empty should fail")
	}
	a := &Alternative{Action: "a", Rejected: "r"}
	if err := a.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestEnvelope_Validate_NonEmptyTrace(t *testing.T) {
	v := Envelope{Type: "t", Source: "s", CreatedAt: "2024-01-01T00:00:00Z", Trace: []string{"a", "", "b"}}
	if err := v.Validate(); err == nil {
		t.Fatal("empty trace element should fail")
	}
}

func TestScheduleTrigger_Validate_CronAndOnce(t *testing.T) {
	// cron with empty cron expression
	if err := (&ScheduleTrigger{Kind: "cron"}).Validate(); err == nil {
		t.Fatal("cron with empty expr should fail")
	}
	// once with empty at
	if err := (&ScheduleTrigger{Kind: "once"}).Validate(); err == nil {
		t.Fatal("once with empty at should fail")
	}
	// valid cron
	if err := (&ScheduleTrigger{Kind: "cron", Cron: "* * *"}).Validate(); err != nil {
		t.Fatal(err)
	}
	// valid once
	if err := (&ScheduleTrigger{Kind: "once", At: "x"}).Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestTaskPlan_UnmarshalJSON_InvalidJSON(t *testing.T) {
	var v TaskPlan
	if err := v.UnmarshalJSON([]byte("not json")); err == nil {
		t.Fatal("expected error")
	}
}

func TestTaskPlan_UnmarshalJSON_InvalidValue(t *testing.T) {
	// known field, wrong type
	var v TaskPlan
	if err := v.UnmarshalJSON([]byte(`{"id":1}`)); err == nil {
		t.Fatal("expected error for wrong type")
	}
}

func TestWorkflowSpec_UnmarshalJSON_InvalidJSON(t *testing.T) {
	var v WorkflowSpec
	if err := v.UnmarshalJSON([]byte("not json")); err == nil {
		t.Fatal("expected error")
	}
}

func TestWorkflowStep_Validate(t *testing.T) {
	if err := (&WorkflowStep{}).Validate(); err == nil {
		t.Fatal("empty should fail")
	}
	if err := (&WorkflowStep{ID: "s", Kind: "tool"}).Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestWorkflowStep_UnmarshalJSON_Invalid(t *testing.T) {
	var v WorkflowStep
	if err := v.UnmarshalJSON([]byte("not json")); err == nil {
		t.Fatal("expected error")
	}
}

func TestWorkflowTransition_Validate(t *testing.T) {
	if err := (&WorkflowTransition{}).Validate(); err == nil {
		t.Fatal("empty should fail")
	}
	if err := (&WorkflowTransition{From: "a", To: "b"}).Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestWorkflowTransition_UnmarshalJSON_Invalid(t *testing.T) {
	var v WorkflowTransition
	if err := v.UnmarshalJSON([]byte("not json")); err == nil {
		t.Fatal("expected error")
	}
}

func TestItoa_Zero(t *testing.T) {
	// The doc-set spec for itoa is "0" for i==0. We test
	// the function directly because the only consumer
	//(Envelope.Validate) skips the empty-trace path in
	// the existing tests.
	if got := itoa(0); got != "0" {
		t.Fatalf("itoa(0) = %q", got)
	}
	if got := itoa(10); got != "10" {
		t.Fatalf("itoa(10) = %q", got)
	}
	if got := itoa(-1); got != "-1" {
		t.Fatalf("itoa(-1) = %q", got)
	}
}

func TestOptionalString_NonEmpty(t *testing.T) {
	// The 80% gap is the non-empty path. The empty path
	// is exercised in TestOptionalString_EmptyAllowed.
	if err := OptionalString("x", "abc"); err != nil {
		t.Fatal("non-empty should be valid")
	}
}

func TestValidateAlternatives_EmptyAndNonEmpty(t *testing.T) {
	if err := validateAlternatives(nil); err != nil {
		t.Fatalf("nil should be valid: %v", err)
	}
	if err := validateAlternatives([]Alternative{}); err != nil {
		t.Fatalf("zero-len should be valid: %v", err)
	}
	bad := []Alternative{{Action: "", Rejected: "r"}}
	if err := validateAlternatives(bad); err == nil {
		t.Fatal("invalid element should fail")
	}
}

func TestNewValidationError_EmptyPath(t *testing.T) {
	// Base path: empty path returns just the reason.
	e := NewValidationError("", "boom")
	if got := e.Error(); got != "boom" {
		t.Fatalf("got %q", got)
	}
}

func TestMarshalCanonical_MarshalErrorPropagates(t *testing.T) {
	// Channels cannot be marshaled, so this exercises
	// the json.Marshal error path.
	c := make(chan int)
	if _, err := MarshalCanonical(c); err == nil {
		t.Fatal("expected marshal error")
	}
}

func TestUnmarshalStrict_NotAMap(t *testing.T) {
	// Top-level array, not object — exercises the
	// json.Unmarshal-into-map error branch.
	var v map[string]any
	if err := UnmarshalStrict([]byte(`[1,2,3]`), &v, []string{"x"}); err == nil {
		t.Fatal("expected error")
	}
}

// keep the errors import alive
var _ = errors.New

func TestBudget_StrictRejectsUnknown(t *testing.T) {
	var v Budget
	err := v.UnmarshalJSON([]byte(`{"cost_usd":1,"x":1}`))
	if err == nil || !strings.Contains(err.Error(), "x") {
		t.Fatalf("err = %v", err)
	}
}

func TestBudget_UnmarshalHappy(t *testing.T) {
	var v Budget
	if err := v.UnmarshalJSON([]byte(`{"cost_usd":1,"wall_clock_sec":60}`)); err != nil {
		t.Fatal(err)
	}
}

func TestBudget_UnmarshalBadJSON(t *testing.T) {
	var v Budget
	if err := v.UnmarshalJSON([]byte("not json")); err == nil {
		t.Fatal("expected error")
	}
}

func TestBudget_ValidateNegative(t *testing.T) {
	if err := (&Budget{CostUSD: -1}).Validate(); err == nil {
		t.Fatal("negative cost should fail")
	}
	if err := (&Budget{WallClockSec: -1}).Validate(); err == nil {
		t.Fatal("negative wall_clock_sec should fail")
	}
}

func TestValidateOptionalStringList(t *testing.T) {
	if err := validateOptionalStringList("p", nil); err != nil {
		t.Fatal("nil list should be valid")
	}
	if err := validateOptionalStringList("p", []string{}); err != nil {
		t.Fatal("empty list should be valid")
	}
	// OptionalString returns nil for "", but rejects NUL
	// bytes in non-empty entries. validateOptionalStringList
	// only flags the NUL-byte case.
	if err := validateOptionalStringList("p", []string{"a", "\x00"}); err == nil {
		t.Fatal("NUL byte should fail")
	}
	if err := validateOptionalStringList("p", []string{"a", "b"}); err != nil {
		t.Fatal(err)
	}
}

func TestEnvelope_Validate_BadTrace(t *testing.T) {
	v := Envelope{Type: "t", Source: "s", CreatedAt: "x", Trace: []string{""}}
	if err := v.Validate(); err == nil {
		t.Fatal("empty trace element should fail")
	}
}

func TestValidateTaskSpecs_NilAndEmpty(t *testing.T) {
	// empty (nil or len-0) is rejected (at-least-one)
	if err := validateTaskSpecs(nil); err == nil {
		t.Fatal("nil should fail")
	}
	if err := validateTaskSpecs([]TaskSpec{}); err == nil {
		t.Fatal("empty should fail")
	}
	// valid spec passes
	good := []TaskSpec{{ID: "t", Description: "d", Acceptance: "a"}}
	if err := validateTaskSpecs(good); err != nil {
		t.Fatal(err)
	}
	// invalid spec is rejected
	bad := []TaskSpec{{ID: "t"}}
	if err := validateTaskSpecs(bad); err == nil {
		t.Fatal("invalid spec should fail")
	}
}

func TestValidateWorkflowSteps(t *testing.T) {
	// empty is rejected
	if err := validateWorkflowSteps(nil); err == nil {
		t.Fatal("nil should fail")
	}
	if err := validateWorkflowSteps([]WorkflowStep{}); err == nil {
		t.Fatal("empty should fail")
	}
	// valid passes
	good := []WorkflowStep{{ID: "s", Kind: "tool"}}
	if err := validateWorkflowSteps(good); err != nil {
		t.Fatal(err)
	}
}

func TestValidateWorkflowTransitions(t *testing.T) {
	steps := []WorkflowStep{{ID: "a", Kind: "tool"}}
	if err := validateWorkflowTransitions(nil, steps); err != nil {
		t.Fatal(err)
	}
	if err := validateWorkflowTransitions([]WorkflowTransition{}, steps); err != nil {
		t.Fatal(err)
	}
}

func TestToolCall_Validate_BadRisk(t *testing.T) {
	if err := (&ToolCall{CallID: "c", Tool: "t", RiskTier: 4}).Validate(); err == nil {
		t.Fatal("risk_tier 4 should fail")
	}
	if err := (&ToolCall{CallID: "c", Tool: "t", RiskTier: -1}).Validate(); err == nil {
		t.Fatal("risk_tier -1 should fail")
	}
}

func TestOptionalString_NonEmptyValue(t *testing.T) {
	if err := OptionalString("x", "v"); err != nil {
		t.Fatal("non-empty should be valid")
	}
}

func TestTaskPlan_UnmarshalJSON_NoKnown(t *testing.T) {
	var v TaskPlan
	// empty JSON object is valid; the schema defaults
	if err := v.UnmarshalJSON([]byte(`{}`)); err != nil {
		t.Fatal(err)
	}
}
