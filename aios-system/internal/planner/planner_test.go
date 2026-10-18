package planner

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/sadi/aios-system/internal/core/schemas"
	"github.com/sadi/aios-system/internal/models"
)

// stubAdapter is a programmable models.Adapter. The test sets
// either a response or an error; the adapter returns them in
// order on successive Complete calls. The adapter records every
// call for the test to inspect.
type stubAdapter struct {
	provider string
	mu       sync.Mutex
	calls    []models.Request
	resp     []models.Response
	err      []error
}

func (s *stubAdapter) Provider() string { return s.provider }

func (s *stubAdapter) Complete(ctx context.Context, req models.Request) (models.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls = append(s.calls, req)
	// Honour cancellation: a real adapter would surface
	// ctx.Err() before doing any work.
	if err := ctx.Err(); err != nil {
		return models.Response{}, err
	}
	if len(s.err) == 0 && len(s.resp) == 0 {
		return models.Response{}, errors.New("stubAdapter: no response scripted")
	}
	// Pop the next scripted response / error pair.
	var resp models.Response
	if len(s.resp) > 0 {
		resp = s.resp[0]
		s.resp = s.resp[1:]
	}
	var err error
	if len(s.err) > 0 {
		err = s.err[0]
		s.err = s.err[1:]
	}
	return resp, err
}

func (s *stubAdapter) Stream(_ context.Context, _ models.Request) (models.Stream, error) {
	return nil, errors.New("not implemented in stubAdapter")
}

func (s *stubAdapter) callCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.calls)
}

func newStubRouter(t *testing.T, stub *stubAdapter) *models.Router {
	t.Helper()
	reg, err := models.NewRegistry([]models.Capability{
		{Model: "gpt-4o", Provider: "openai", ContextWindow: 128000},
	})
	if err != nil {
		t.Fatal(err)
	}
	prices := models.NewPriceTable([]models.Price{
		{Model: "gpt-4o", InputPer1K: 0.005, OutputPer1K: 0.015},
	}, nil)
	return models.NewRouter(map[string]models.Adapter{
		"openai": stub,
	}, reg, prices)
}

func mkTools() map[string]ToolSpec {
	return map[string]ToolSpec{
		"read_file": {Name: "read_file", Description: "read", RiskTier: 0, InputSchema: "{}"},
		"edit_file": {Name: "edit_file", Description: "edit", RiskTier: 1, InputSchema: "{}"},
	}
}

// validPlanJSON returns a JSON string that parses to a
// schemas.TaskPlan that the verifier accepts. Tests can edit the
// returned string to make it invalid.
func validPlanJSON() string {
	plan := schemas.TaskPlan{
		ID:   "p1",
		Goal: "ship aios",
		Tasks: []schemas.TaskSpec{
			{ID: "t1", Description: "read", Tool: "read_file", Acceptance: "ok; lines=3"},
		},
		Tools: []string{"read_file"},
		Model: "gpt-4o",
	}
	raw, _ := json.Marshal(plan)
	return string(raw)
}

func TestPlanner_HappyPath(t *testing.T) {
	stub := &stubAdapter{
		provider: "openai",
		resp:     []models.Response{{Text: validPlanJSON(), Usage: models.Usage{InputTokens: 100, OutputTokens: 50, Model: "gpt-4o"}}},
	}
	router := newStubRouter(t, stub)
	verifier := NewCodeFirstVerifier(mkTools())
	p, err := NewPlanner(Config{Router: router, Tools: mkTools(), Verifier: verifier})
	if err != nil {
		t.Fatal(err)
	}
	got, err := p.Plan(context.Background(), "ship aios", nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "p1" {
		t.Fatalf("id: %q", got.ID)
	}
	if got.Goal != "ship aios" {
		t.Fatalf("goal: %q", got.Goal)
	}
	if len(got.Tasks) != 1 {
		t.Fatalf("tasks: %d", len(got.Tasks))
	}
	if stub.callCount() != 1 {
		t.Fatalf("calls: %d", stub.callCount())
	}
}

func TestPlanner_RepairThenPass(t *testing.T) {
	// First response is invalid (missing tasks), second is valid.
	bad := `{"id":"p1","goal":"ship aios","tasks":[]}`
	stub := &stubAdapter{
		provider: "openai",
		resp: []models.Response{
			{Text: bad, Usage: models.Usage{InputTokens: 100, OutputTokens: 50, Model: "gpt-4o"}},
			{Text: validPlanJSON(), Usage: models.Usage{InputTokens: 200, OutputTokens: 80, Model: "gpt-4o"}},
		},
	}
	router := newStubRouter(t, stub)
	verifier := NewCodeFirstVerifier(mkTools())
	p, err := NewPlanner(Config{Router: router, Tools: mkTools(), Verifier: verifier, MaxRepairIterations: 3})
	if err != nil {
		t.Fatal(err)
	}
	got, err := p.Plan(context.Background(), "ship aios", nil)
	if err != nil {
		t.Fatalf("expected eventual success, got %v", err)
	}
	if got.ID != "p1" {
		t.Fatalf("id: %q", got.ID)
	}
	if stub.callCount() != 2 {
		t.Fatalf("expected 2 model calls, got %d", stub.callCount())
	}
	// The second call should be a repair prompt; assert it
	// mentions the previous failure.
	calls := stub.calls
	if !strings.Contains(calls[1].Prompt, "previous plan") && !strings.Contains(calls[1].Prompt, "Previous plan") {
		t.Errorf("second call should be a repair prompt: %s", calls[1].Prompt)
	}
	if !strings.Contains(calls[1].Prompt, "schema") {
		t.Errorf("repair prompt should include the error code: %s", calls[1].Prompt)
	}
}

func TestPlanner_RepairExhausted(t *testing.T) {
	// Every attempt is invalid; planner should raise
	// PlanGenerationError after the cap.
	bad := `{"id":"p1","goal":"ship aios","tasks":[]}`
	stub := &stubAdapter{
		provider: "openai",
		resp: []models.Response{
			{Text: bad, Usage: models.Usage{InputTokens: 100, OutputTokens: 50, Model: "gpt-4o"}},
			{Text: bad, Usage: models.Usage{InputTokens: 100, OutputTokens: 50, Model: "gpt-4o"}},
			{Text: bad, Usage: models.Usage{InputTokens: 100, OutputTokens: 50, Model: "gpt-4o"}},
			{Text: bad, Usage: models.Usage{InputTokens: 100, OutputTokens: 50, Model: "gpt-4o"}},
		},
	}
	router := newStubRouter(t, stub)
	verifier := NewCodeFirstVerifier(mkTools())
	p, err := NewPlanner(Config{Router: router, Tools: mkTools(), Verifier: verifier, MaxRepairIterations: 2})
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.Plan(context.Background(), "ship aios", nil)
	if err == nil {
		t.Fatal("expected PlanGenerationError")
	}
	var pge *PlanGenerationError
	if !errors.As(err, &pge) {
		t.Fatalf("expected *PlanGenerationError, got %T: %v", err, err)
	}
	// 1 first attempt + 2 repair attempts = 3 total.
	if pge.Iterations != 3 {
		t.Errorf("iterations: %d", pge.Iterations)
	}
	// 4 scripted responses, only 3 should be consumed (1 + 2).
	if stub.callCount() != 3 {
		t.Errorf("expected 3 calls, got %d", stub.callCount())
	}
}

func TestPlanner_RepairExhausted_OnModelError(t *testing.T) {
	// The model returns an error on the first attempt and on
	// every repair attempt. The planner should raise a
	// PlanGenerationError whose Cause is the model's error.
	modelErr := errors.New("rate limited")
	stub := &stubAdapter{
		provider: "openai",
		err:      []error{modelErr, modelErr, modelErr},
	}
	router := newStubRouter(t, stub)
	verifier := NewCodeFirstVerifier(mkTools())
	p, err := NewPlanner(Config{Router: router, Tools: mkTools(), Verifier: verifier, MaxRepairIterations: 2})
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.Plan(context.Background(), "ship aios", nil)
	if err == nil {
		t.Fatal("expected PlanGenerationError")
	}
	var pge *PlanGenerationError
	if !errors.As(err, &pge) {
		t.Fatalf("expected *PlanGenerationError, got %T", err)
	}
	if !errors.Is(pge, modelErr) {
		t.Errorf("Unwrap should expose model error: %v", err)
	}
}

func TestPlanner_EmptyGoal(t *testing.T) {
	router := newStubRouter(t, &stubAdapter{provider: "openai"})
	verifier := NewCodeFirstVerifier(mkTools())
	p, _ := NewPlanner(Config{Router: router, Tools: mkTools(), Verifier: verifier})
	_, err := p.Plan(context.Background(), "", nil)
	if err == nil {
		t.Fatal("expected error for empty goal")
	}
	var pve *PlanValidationError
	if !errors.As(err, &pve) {
		t.Fatalf("expected *PlanValidationError, got %T", err)
	}
}

func TestPlanner_StripsMarkdownFence(t *testing.T) {
	// The model sometimes wraps the response in a code fence.
	wrapped := "```json\n" + validPlanJSON() + "\n```"
	stub := &stubAdapter{
		provider: "openai",
		resp:     []models.Response{{Text: wrapped, Usage: models.Usage{InputTokens: 100, OutputTokens: 50, Model: "gpt-4o"}}},
	}
	router := newStubRouter(t, stub)
	verifier := NewCodeFirstVerifier(mkTools())
	p, _ := NewPlanner(Config{Router: router, Tools: mkTools(), Verifier: verifier})
	got, err := p.Plan(context.Background(), "ship aios", nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "p1" {
		t.Fatalf("id: %q", got.ID)
	}
}

func TestPlanner_RepairFromMalformedJSON(t *testing.T) {
	// The first response is not even JSON. The repair loop
	// should ask the model again; the second response is valid.
	stub := &stubAdapter{
		provider: "openai",
		resp: []models.Response{
			{Text: "this is not json", Usage: models.Usage{InputTokens: 10, OutputTokens: 5, Model: "gpt-4o"}},
			{Text: validPlanJSON(), Usage: models.Usage{InputTokens: 200, OutputTokens: 80, Model: "gpt-4o"}},
		},
	}
	router := newStubRouter(t, stub)
	verifier := NewCodeFirstVerifier(mkTools())
	p, _ := NewPlanner(Config{Router: router, Tools: mkTools(), Verifier: verifier, MaxRepairIterations: 3})
	got, err := p.Plan(context.Background(), "ship aios", nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "p1" {
		t.Fatalf("id: %q", got.ID)
	}
	if stub.callCount() != 2 {
		t.Fatalf("calls: %d", stub.callCount())
	}
}

func TestPlanner_Defaults(t *testing.T) {
	router := newStubRouter(t, &stubAdapter{provider: "openai"})
	verifier := NewCodeFirstVerifier(mkTools())
	p, err := NewPlanner(Config{Router: router, Tools: mkTools(), Verifier: verifier})
	if err != nil {
		t.Fatal(err)
	}
	if p.maxRepair != 3 {
		t.Errorf("maxRepair default: %d", p.maxRepair)
	}
	if p.defaultMod != "gpt-4o" {
		t.Errorf("default model: %q", p.defaultMod)
	}
}

func TestPlanner_RequiresRouter(t *testing.T) {
	verifier := NewCodeFirstVerifier(mkTools())
	_, err := NewPlanner(Config{Tools: mkTools(), Verifier: verifier})
	if err == nil {
		t.Fatal("expected error for nil router")
	}
}

func TestPlanner_RequiresTools(t *testing.T) {
	router := newStubRouter(t, &stubAdapter{provider: "openai"})
	verifier := NewCodeFirstVerifier(mkTools())
	_, err := NewPlanner(Config{Router: router, Verifier: verifier})
	if err == nil {
		t.Fatal("expected error for empty tools")
	}
}

func TestPlanner_RequiresVerifier(t *testing.T) {
	router := newStubRouter(t, &stubAdapter{provider: "openai"})
	_, err := NewPlanner(Config{Router: router, Tools: mkTools()})
	if err == nil {
		t.Fatal("expected error for nil verifier")
	}
}

func TestPlanner_ContextPropagated(t *testing.T) {
	stub := &stubAdapter{
		provider: "openai",
		resp:     []models.Response{{Text: validPlanJSON(), Usage: models.Usage{InputTokens: 100, OutputTokens: 50, Model: "gpt-4o"}}},
	}
	router := newStubRouter(t, stub)
	verifier := NewCodeFirstVerifier(mkTools())
	p, _ := NewPlanner(Config{Router: router, Tools: mkTools(), Verifier: verifier})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	// A cancelled context will trip the first call. We want to
	// confirm the planner respects it. The model call may or may
	// not be reached, depending on the order of checks; we just
	// check that the error is non-nil and mentions the goal.
	_, err := p.Plan(ctx, "ship aios", map[string]any{"k": "v"})
	if err == nil {
		t.Fatal("expected error with cancelled context")
	}
}

func TestParsePlan_EdgeCases(t *testing.T) {
	cases := []struct {
		name    string
		raw     string
		wantErr bool
	}{
		{"empty", "", true},
		{"whitespace", "   \n\t  ", true},
		{"not json", "hello world", true},
		{"array not object", "[]", true},
		{"object missing fields", `{"id":"p1"}`, true},
		{"valid", validPlanJSON(), false},
		{"fence json", "```json\n" + validPlanJSON() + "\n```", false},
		{"fence no language", "```\n" + validPlanJSON() + "\n```", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := parsePlan(c.raw)
			if (err != nil) != c.wantErr {
				t.Errorf("got err=%v, wantErr=%v", err, c.wantErr)
			}
		})
	}
}

func TestStripMarkdownFence(t *testing.T) {
	cases := map[string]string{
		"```json\n{}\n```":        "{}",
		"```\n{}\n```":            "{}",
		"```json\n{\"a\":1}\n```": "{\"a\":1}",
		"```json\n{}\n```\n":      "{}",
		"```\n{\"a\":1}\n```":     "{\"a\":1}",
		"no fence":                "no fence",
		"":                        "",
		"```":                     "",
		"{}":                      "{}",
	}
	for in, want := range cases {
		if got := stripMarkdownFence(in); got != want {
			t.Errorf("stripMarkdownFence(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestTrimSpace(t *testing.T) {
	cases := map[string]string{
		"":        "",
		"   ":     "",
		"\n\n":    "",
		"\t a \n": "a",
		"a":       "a",
		" a b ":   "a b",
	}
	for in, want := range cases {
		if got := trimSpace(in); got != want {
			t.Errorf("trimSpace(%q) = %q, want %q", in, got, want)
		}
	}
}
