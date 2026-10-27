package schemas

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/Hamidullah02/aios/aios-system/internal/core/clock"
	"github.com/Hamidullah02/aios/aios-system/internal/core/state"
)

func TestEachType_MarshalUnmarshal_RoundTrip(t *testing.T) {
	cases := []struct {
		name string
		v    interface {
			json.Marshaler
			json.Unmarshaler
		}
	}{
		{"TaskPlan", &TaskPlan{ID: "p", Goal: "g", Tasks: []TaskSpec{{ID: "t", Description: "d", Acceptance: "a"}}}},
		{"AgentSpec", &AgentSpec{ID: "a", Role: "r", Tools: []string{"x"}, Model: "m"}},
		{"WorkflowSpec", &WorkflowSpec{ID: "w", Name: "n", Steps: []WorkflowStep{{ID: "s", Kind: "end"}}}},
		{"ScheduleSpec", &ScheduleSpec{ID: "s", Trigger: ScheduleTrigger{Kind: "cron", Cron: "* * * * *"}, Workflow: WorkflowSpec{ID: "w", Name: "n", Steps: []WorkflowStep{{ID: "s", Kind: "end"}}}}},
		{"DecisionTrace", &DecisionTrace{StepID: "s", Chosen: "c", Basis: "b", RiskTier: 1}},
		{"Envelope", &Envelope{Type: "t", Source: "s", CreatedAt: "2025-01-01T00:00:00Z"}},
		{"ApprovalPayload", &ApprovalPayload{ApprovalID: "550e8400-e29b-41d4-a716-446655440000", Action: "a", Target: "t", RiskTier: 0, Reason: "r", RequesterID: "op"}},
		{"ModelCapability", &ModelCapability{ModelID: "m", Class: "mid", MaxTokens: 1, Supports: []string{"tools"}}},
		{"ToolCall", &ToolCall{CallID: "c", Tool: "t", RiskTier: 0}},
		{"ToolResult", &ToolResult{CallID: "c", OK: true}},
		{"Budget", &Budget{}},
		{"Alternative", &Alternative{Action: "a", Rejected: "r"}},
		{"WorkflowStep", &WorkflowStep{ID: "s", Kind: "end"}},
		{"WorkflowTransition", &WorkflowTransition{From: "a", To: "b"}},
		{"ScheduleTrigger", &ScheduleTrigger{Kind: "cron", Cron: "*"}},
		{"ScheduleLimits", &ScheduleLimits{}},
	}
	for _, c := range cases {
		data, err := json.Marshal(c.v)
		if err != nil {
			t.Fatalf("%s: marshal: %v", c.name, err)
		}
		// Re-decode into a generic map to ensure the bytes are
		// well-formed and sortable.
		var generic any
		if err := json.Unmarshal(data, &generic); err != nil {
			t.Fatalf("%s: unmarshal generic: %v", c.name, err)
		}
	}
}

func TestValidationError_Error(t *testing.T) {
	e := &ValidationError{Path: "x", Reason: "y"}
	if e.Error() != "x: y" {
		t.Fatalf("got %q", e.Error())
	}
	e2 := &ValidationError{Reason: "y"}
	if e2.Error() != "y" {
		t.Fatalf("got %q", e2.Error())
	}
}

func TestNonEmptyList_Generic(t *testing.T) {
	if err := NonEmptyList("p", []int{}); err == nil {
		t.Fatal("expected error")
	}
	if err := NonEmptyList("p", []int{1}); err != nil {
		t.Fatal(err)
	}
}

func TestEmail_And_MinLen(t *testing.T) {
	if err := Email("e", ""); err != nil {
		t.Fatal("empty email OK")
	}
	if err := Email("e", "not-an-email"); err == nil {
		t.Fatal("expected error for bad email")
	}
	if err := MinLen("p", "ab", 3); err == nil {
		t.Fatal("expected error for short string")
	}
}

func TestWorkflowSpec_DuplicateStepID(t *testing.T) {
	w := &WorkflowSpec{
		ID:   "w",
		Name: "n",
		Steps: []WorkflowStep{
			{ID: "s", Kind: "tool"},
			{ID: "s", Kind: "end"},
		},
	}
	if err := w.Validate(); err == nil {
		t.Fatal("expected duplicate step id error")
	}
}

func TestScheduleSpec_OnceValid(t *testing.T) {
	s := &ScheduleSpec{
		ID:      "s",
		Trigger: ScheduleTrigger{Kind: "once", At: "2025-01-01T00:00:00Z"},
		Workflow: WorkflowSpec{
			ID: "w", Name: "n",
			Steps: []WorkflowStep{{ID: "s", Kind: "end"}},
		},
	}
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestScheduleSpec_BadKind(t *testing.T) {
	s := &ScheduleSpec{
		ID:      "s",
		Trigger: ScheduleTrigger{Kind: "bogus"},
		Workflow: WorkflowSpec{
			ID: "w", Name: "n",
			Steps: []WorkflowStep{{ID: "s", Kind: "end"}},
		},
	}
	if err := s.Validate(); err == nil {
		t.Fatal("expected error for bad trigger kind")
	}
}

func TestScheduleLimits_Negative(t *testing.T) {
	cases := []ScheduleLimits{
		{MaxRuns: -1},
		{MaxWallSec: -1},
		{MaxCostUSD: -1},
	}
	for _, l := range cases {
		if err := l.Validate(); err == nil {
			t.Fatal("expected error for negative limit")
		}
	}
}

func TestTaskPlan_BudgetNegative(t *testing.T) {
	p := &TaskPlan{
		ID: "p", Goal: "g",
		Tasks:  []TaskSpec{{ID: "t", Description: "d", Acceptance: "a"}},
		Budget: &Budget{CostUSD: -1},
	}
	if err := p.Validate(); err == nil {
		t.Fatal("expected error for negative cost")
	}
}

func TestDecisionTrace_EmptyAlternatives(t *testing.T) {
	d := &DecisionTrace{StepID: "s", Chosen: "c", Basis: "b", RiskTier: 1}
	if err := d.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestTransitionError_Error(t *testing.T) {
	e := &state.TransitionError{From: state.StatePending, To: state.StateRunning, Reason: "x"}
	if e.Error() == "" {
		t.Fatal("non-empty error expected")
	}
}

func TestNewMachine_DoesNotMutate(t *testing.T) {
	// A Machine value (not pointer) is fine; check no surprise.
	m := state.NewMachine("t1")
	if m.TaskID != "t1" || m.State != state.StatePending {
		t.Fatalf("got %+v", m)
	}
}

func TestFail_Helper(t *testing.T) {
	m := state.NewMachine("t1")
	_ = m.Transition(state.StatePlanning)
	if err := m.Fail(); err != nil {
		t.Fatal(err)
	}
	if m.State != state.StateFailed {
		t.Fatalf("state: %s", m.State)
	}
}

func TestClock_AllNows(t *testing.T) {
	var sysDec clock.Decision = clock.SystemDecision{}
	if sysDec.Now().IsZero() {
		t.Fatal("SystemDecision should not be zero")
	}
	var sysDdl clock.Deadline = clock.SystemDeadline{}
	if sysDdl.Now().IsZero() {
		t.Fatal("SystemDeadline should not be zero")
	}
	fixDec := clock.FixedDecision{T: time.Unix(1, 0)}
	if !fixDec.Now().Equal(time.Unix(1, 0)) {
		t.Fatal("FixedDecision")
	}
	fixDdl := clock.FixedDeadline{T: time.Unix(1, 0)}
	if !fixDdl.Now().Equal(time.Unix(1, 0)) {
		t.Fatal("FixedDeadline")
	}
}

func TestTaskPlan_Valid(t *testing.T) {
	p := &TaskPlan{
		ID:   "p1",
		Goal: "ship aios",
		Tasks: []TaskSpec{
			{ID: "t1", Description: "design", Acceptance: "design doc exists"},
		},
	}
	if err := p.Validate(); err != nil {
		t.Fatalf("valid plan: %v", err)
	}
}

func TestTaskPlan_MissingGoal(t *testing.T) {
	p := &TaskPlan{
		ID: "p1",
		Tasks: []TaskSpec{
			{ID: "t1", Description: "design", Acceptance: "doc"},
		},
	}
	if err := p.Validate(); err == nil {
		t.Fatal("expected error for missing goal")
	}
}

func TestTaskPlan_NoTasks(t *testing.T) {
	p := &TaskPlan{ID: "p1", Goal: "g"}
	if err := p.Validate(); err == nil {
		t.Fatal("expected error for empty tasks")
	}
}

func TestTaskPlan_CanonicalMarshal(t *testing.T) {
	p := &TaskPlan{
		ID:   "p1",
		Goal: "ship",
		Tasks: []TaskSpec{
			{ID: "t1", Description: "design", Acceptance: "ok"},
		},
		Budget: &Budget{CostUSD: 1.5, WallClockSec: 60},
	}
	a, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	// Keys must be sorted.
	const want = `{"budget":{"cost_usd":1.5,"wall_clock_sec":60},"goal":"ship","id":"p1","tasks":[{"acceptance":"ok","description":"design","id":"t1"}]}`
	if string(a) != want {
		t.Fatalf("canonical mismatch:\n got  %s\n want %s", a, want)
	}
}

func TestTaskPlan_StrictUnmarshal_RejectsUnknown(t *testing.T) {
	const data = `{"id":"p1","goal":"g","tasks":[{"id":"t1","description":"d","acceptance":"a"}],"extra":1}`
	var p TaskPlan
	if err := json.Unmarshal([]byte(data), &p); err == nil {
		t.Fatal("expected error for unknown field")
	}
}

func TestAgentSpec_Valid(t *testing.T) {
	a := &AgentSpec{ID: "a1", Role: "planner", Tools: []string{"plan"}, Model: "m1"}
	if err := a.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestAgentSpec_NoTools(t *testing.T) {
	a := &AgentSpec{ID: "a1", Role: "planner", Model: "m1"}
	if err := a.Validate(); err == nil {
		t.Fatal("expected error for no tools")
	}
}

func TestWorkflowSpec_Valid(t *testing.T) {
	w := &WorkflowSpec{
		ID:   "w1",
		Name: "ship",
		Steps: []WorkflowStep{
			{ID: "s1", Kind: "tool", Tool: "edit"},
			{ID: "s2", Kind: "end"},
		},
		Transitions: []WorkflowTransition{
			{From: "s1", To: "s2"},
		},
	}
	if err := w.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestWorkflowSpec_UnknownTransitionTarget(t *testing.T) {
	w := &WorkflowSpec{
		ID:   "w1",
		Name: "ship",
		Steps: []WorkflowStep{
			{ID: "s1", Kind: "tool"},
		},
		Transitions: []WorkflowTransition{
			{From: "s1", To: "nope"},
		},
	}
	if err := w.Validate(); err == nil {
		t.Fatal("expected error for unknown transition target")
	}
}

func TestScheduleSpec_CronValid(t *testing.T) {
	s := &ScheduleSpec{
		ID:      "sc1",
		Trigger: ScheduleTrigger{Kind: "cron", Cron: "*/5 * * * *"},
		Workflow: WorkflowSpec{
			ID:   "w1",
			Name: "ship",
			Steps: []WorkflowStep{
				{ID: "s1", Kind: "end"},
			},
		},
	}
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestDecisionTrace_Valid(t *testing.T) {
	d := &DecisionTrace{
		StepID:   "st1",
		Chosen:   "edit",
		Basis:    "because",
		RiskTier: 2,
		Alternatives: []Alternative{
			{Action: "skip", Score: 0.1, Rejected: "low score"},
		},
	}
	if err := d.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestDecisionTrace_BadRiskTier(t *testing.T) {
	d := &DecisionTrace{
		StepID: "st1", Chosen: "edit", Basis: "b", RiskTier: 7,
	}
	if err := d.Validate(); err == nil {
		t.Fatal("expected error for risk_tier 7")
	}
}

func TestEnvelope_Valid(t *testing.T) {
	e := &Envelope{
		Type:      "task_plan",
		Source:    "cli",
		CreatedAt: "2025-01-01T00:00:00Z",
	}
	if err := e.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestApprovalPayload_Valid(t *testing.T) {
	a := &ApprovalPayload{
		ApprovalID:  "550e8400-e29b-41d4-a716-446655440000",
		Action:      "edit",
		Target:      "/tmp/file",
		RiskTier:    2,
		Reason:      "user request",
		RequesterID: "operator:alice",
	}
	if err := a.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestApprovalPayload_BadUUID(t *testing.T) {
	a := &ApprovalPayload{
		ApprovalID:  "not-a-uuid",
		Action:      "edit",
		Target:      "/tmp/file",
		RiskTier:    2,
		Reason:      "x",
		RequesterID: "operator:alice",
	}
	if err := a.Validate(); err == nil {
		t.Fatal("expected error for bad UUID")
	}
}

func TestModelCapability_Valid(t *testing.T) {
	m := &ModelCapability{
		ModelID:   "m1",
		Class:     "mid",
		MaxTokens: 8000,
		Supports:  []string{"tools"},
	}
	if err := m.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestModelCapability_BadClass(t *testing.T) {
	m := &ModelCapability{
		ModelID:   "m1",
		Class:     "huge",
		MaxTokens: 8000,
		Supports:  []string{"tools"},
	}
	if err := m.Validate(); err == nil {
		t.Fatal("expected error for class huge")
	}
}

func TestToolCall_Valid(t *testing.T) {
	c := &ToolCall{CallID: "c1", Tool: "edit", RiskTier: 1}
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestToolResult_OK(t *testing.T) {
	r := &ToolResult{CallID: "c1", OK: true}
	if err := r.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestToolResult_NotOK_NoError(t *testing.T) {
	r := &ToolResult{CallID: "c1", OK: false}
	if err := r.Validate(); err == nil {
		t.Fatal("expected error: !ok requires error")
	}
}

func TestToolResult_OK_WithError(t *testing.T) {
	r := &ToolResult{CallID: "c1", OK: true, Error: "wat"}
	if err := r.Validate(); err == nil {
		t.Fatal("expected error: ok forbids error")
	}
}
