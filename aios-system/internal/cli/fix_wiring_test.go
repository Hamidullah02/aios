package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Hamidullah02/aios/aios-system/internal/approvals"
	"github.com/Hamidullah02/aios/aios-system/internal/core/schemas"
	"github.com/Hamidullah02/aios/aios-system/internal/models"
	"github.com/Hamidullah02/aios/aios-system/internal/planner"
	"github.com/Hamidullah02/aios/aios-system/internal/policy"
	"github.com/Hamidullah02/aios/aios-system/internal/runtime"
)

// stubAdapter is a models.Adapter that returns a fixed
// Response. The planner uses it to drive a deterministic
// model client. Mirrors the stub in planner_test.go.
type stubAdapter struct {
	provider string
	resp     []models.Response
	idx      int
}

func (s *stubAdapter) Provider() string { return s.provider }
func (s *stubAdapter) Complete(_ context.Context, _ models.Request) (models.Response, error) {
	if s.idx >= len(s.resp) {
		// Replay the last response; the planner's repair
		// loop may call Complete more times than we
		// scripted.
		return s.resp[len(s.resp)-1], nil
	}
	r := s.resp[s.idx]
	s.idx++
	return r, nil
}
func (s *stubAdapter) Stream(_ context.Context, _ models.Request) (models.Stream, error) {
	return nil, nil
}

// validFixPlanJSON returns a JSON string the planner
// verifier accepts. The plan reads a single file. Used
// by the happy-path test below.
func validFixPlanJSON() string {
	plan := schemas.TaskPlan{
		ID:   "fix-1",
		Goal: "read the design doc",
		Tasks: []schemas.TaskSpec{
			{
				ID:          "t1",
				Description: "read design.md",
				Tool:        "read_file",
				Acceptance:  "ok; lines=3",
				Inputs:      []string{"path=design.md"},
			},
		},
		Tools: []string{"read_file"},
		Model: "stub-1",
	}
	raw, _ := json.Marshal(plan)
	return string(raw)
}

// newFixTestDeps builds a Deps with a stub router and
// a planner wired for the production tool set. The
// helper mirrors the production wiring
// (buildRouter + wirePlanner) but with a stub adapter
// so tests do not require an API key.
func newFixTestDeps(t *testing.T, dir string, resp []models.Response) *Deps {
	t.Helper()
	// Write a real file the plan will read.
	if err := os.WriteFile(filepath.Join(dir, "design.md"),
		[]byte("# design\n\nhello aios\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	log := openTestAudit(t)
	gw := policy.New([]policy.Rule{
		{ID: "allow-read", Match: policy.Match{Tool: "read_file"}, Action: policy.ActionAllow},
	}, log)
	bus, worker, verifier := buildTestAgentLoop(t, dir, gw, log)
	// Construct the stub router.
	stub := &stubAdapter{provider: "openai", resp: resp}
	reg, _ := models.NewRegistry([]models.Capability{
		{Model: "stub-1", Provider: "openai", ContextWindow: 128000, SupportsTools: true},
	})
	prices := models.NewPriceTable([]models.Price{
		{Model: "stub-1", InputPer1K: 0.001, OutputPer1K: 0.002},
	}, nil)
	router := models.NewRouter(map[string]models.Adapter{
		"openai": stub,
	}, reg, prices)
	tools := plannerToolSpecs(testBus(t))
	pl, err := planner.NewPlanner(planner.Config{
		Router:       router,
		Tools:        tools,
		Verifier:     planner.NewCodeFirstVerifier(tools),
		DefaultModel: "stub-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	return &Deps{
		Audit:     log,
		Policy:    gw,
		Resume:    runtime.NewResumeHandler(filepath.Join(dir, "checkpoints")),
		Bus:       bus,
		Worker:    worker,
		Verifier:  verifier,
		Approvals: approvals.New(approvals.Config{Audit: log}),
		Router:    router,
		Prices:    prices,
		Planner:   pl,
	}
}

// --- tests ---

// TestBuildDeps_WiresPlannerForFix asserts that the
// production BuildDeps constructs a planner for the
// `fix` subcommand but not for other non-bootstrap
// subcommands. This is the v0.33.0 wiring: the planner
// is added on top of v0.32.0's BusWorker wiring.
func TestBuildDeps_WiresPlannerForFix(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfg, []byte("rules:\n  - id: allow-read\n    match:\n      tool: read_file\n    action: allow\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// fix: planner must be wired.
	args := &Args{
		Command:    CmdFix,
		ConfigPath: cfg,
		LogPath:    filepath.Join(dir, "fix.db"),
		Workspace:  dir,
		Objective:  "do something",
	}
	deps, err := BuildDeps(args)
	if err != nil {
		t.Fatal(err)
	}
	defer deps.Close()
	if deps.Planner == nil {
		t.Error("Planner is nil for fix; BuildDeps did not wire it")
	}
	if deps.Router == nil {
		t.Error("Router is nil for fix; BuildDeps did not wire it")
	}

	// run: planner must NOT be wired.
	args2 := &Args{
		Command:    CmdRun,
		ConfigPath: cfg,
		LogPath:    filepath.Join(dir, "run.db"),
		Workspace:  dir,
	}
	deps2, err := BuildDeps(args2)
	if err != nil {
		t.Fatal(err)
	}
	defer deps2.Close()
	if deps2.Planner != nil {
		t.Error("Planner is set for run; it should be nil (only fix wires it)")
	}
}

// TestRunFix_RequiresObjective asserts runFix returns a
// user error when the objective is empty.
func TestRunFix_RequiresObjective(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runFix(context.Background(), &Args{Command: CmdFix}, &Deps{}, &stdout, &stderr)
	if code != ExitUserError {
		t.Errorf("code: %d, want %d (ExitUserError); stderr: %s", code, ExitUserError, stderr.String())
	}
}

// TestRunFix_RealChain runs the full chain end-to-end:
// the stub model returns a valid plan JSON, the planner
// produces a verified TaskPlan, the BusWorker dispatches
// to the bus, the bus consults the policy gateway, runs
// the real read_file tool, writes the audit row, and the
// CommandVerifier passes. The test asserts the audit log
// has a tool_call row.
func TestRunFix_RealChain(t *testing.T) {
	dir := t.TempDir()
	deps := newFixTestDeps(t, dir, []models.Response{
		{Text: validFixPlanJSON(), Usage: models.Usage{InputTokens: 100, OutputTokens: 80, Model: "stub-1"}},
	})
	defer deps.Close()
	args := &Args{
		Command:   CmdFix,
		Workspace: dir,
		Objective: "read the design doc",
	}
	var stdout, stderr bytes.Buffer
	code := runFix(context.Background(), args, deps, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("code: %d, stderr: %s", code, stderr.String())
	}
	// The audit log must have at least one tool_call
	// row (the read_file call from the BusWorker).
	rows, _ := deps.Audit.GetByKind("tool_call", 100)
	if len(rows) < 1 {
		t.Fatalf("expected tool_call rows; got 0")
	}
	// The plan was completed.
	if !strings.Contains(stdout.String(), "fix-1") {
		t.Errorf("output should mention plan id; got %q", stdout.String())
	}
}

// TestRunFix_NoAPIKey_FailsClosed asserts that a missing
// API key produces a clear error and a non-zero exit
// when the production BuildDeps wires the real adapters.
//
// The test sets OPENAI_API_KEY and ANTHROPIC_API_KEY to
// empty values so the production adapters' newRequest
// helper returns "OPENAI_API_KEY is not set". The
// planner propagates the error to runFix.
func TestRunFix_NoAPIKey_FailsClosed(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("ANTHROPIC_API_KEY", "")
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfg, []byte("rules:\n  - id: allow-read\n    match:\n      tool: read_file\n    action: allow\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := Main([]string{
		"aios",
		"--config=" + cfg,
		"--log=" + filepath.Join(dir, "aios.db"),
		"--workspace=" + dir,
		"fix", "do something",
	}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("expected non-zero exit for missing API key; got 0. stdout: %s", stdout.String())
	}
	// The error must mention the missing key.
	if !strings.Contains(stderr.String(), "OPENAI_API_KEY") &&
		!strings.Contains(stderr.String(), "ANTHROPIC_API_KEY") &&
		!strings.Contains(stderr.String(), "no_api_key") {
		t.Errorf("stderr should mention the missing API key; got %q", stderr.String())
	}
}

// TestRunFix_EndToEndWithHTTPServer runs the full chain
// against a real httptest server that mimics the OpenAI
// chat completions endpoint. The test asserts the
// planner's call hits the server, parses the response,
// and the resulting plan runs through the real loop.
//
// This is the integration test the v0.31.0 review
// demanded: a real model call (to a fake server) drives
// a real plan that runs through a real worker, bus,
// policy gateway, tool, audit log, and verifier.
func TestRunFix_EndToEndWithHTTPServer(t *testing.T) {
	// Point the production OpenAI adapter at the test
	// server. The adapter reads the API key from the
	// config; we set it to a non-empty value so the
	// adapter does not short-circuit on "no API key".
	t.Setenv("OPENAI_API_KEY", "test-key-not-real")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Mimic the OpenAI chat completions response.
		w.Header().Set("Content-Type", "application/json")
		body := `{
			"model": "stub-1",
			"choices": [{"message": {"role": "assistant", "content": ` + jsonString(validFixPlanJSON()) + `}}],
			"usage": {"prompt_tokens": 200, "completion_tokens": 100}
		}`
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()
	// Build a router whose OpenAI adapter points at the
	// test server. The adapter reads OPENAI_API_KEY
	// from the env (set above) and uses the custom Base
	// from the constructor.
	stub := models.NewOpenAI(models.OpenAIConfig{Base: srv.URL, APIKey: "test-key-not-real"})
	reg, _ := models.NewRegistry([]models.Capability{
		{Model: "stub-1", Provider: "openai", ContextWindow: 128000, SupportsTools: true},
	})
	prices := models.NewPriceTable([]models.Price{
		{Model: "stub-1", InputPer1K: 0.001, OutputPer1K: 0.002},
	}, nil)
	router := models.NewRouter(map[string]models.Adapter{"openai": stub}, reg, prices)
	tools := plannerToolSpecs(testBus(t))
	pl, _ := planner.NewPlanner(planner.Config{
		Router:       router,
		Tools:        tools,
		Verifier:     planner.NewCodeFirstVerifier(tools),
		DefaultModel: "stub-1",
	})
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "design.md"),
		[]byte("# design\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	log := openTestAudit(t)
	gw := policy.New([]policy.Rule{
		{ID: "allow-read", Match: policy.Match{Tool: "read_file"}, Action: policy.ActionAllow},
	}, log)
	bus, worker, verifier := buildTestAgentLoop(t, dir, gw, log)
	deps := &Deps{
		Audit:     log,
		Policy:    gw,
		Resume:    runtime.NewResumeHandler(filepath.Join(dir, "checkpoints")),
		Bus:       bus,
		Worker:    worker,
		Verifier:  verifier,
		Approvals: approvals.New(approvals.Config{Audit: log}),
		Router:    router,
		Prices:    prices,
		Planner:   pl,
	}
	defer deps.Close()
	args := &Args{
		Command:   CmdFix,
		Workspace: dir,
		Objective: "read the design doc",
	}
	var stdout, stderr bytes.Buffer
	code := runFix(context.Background(), args, deps, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("code: %d, stderr: %s", code, stderr.String())
	}
	// The audit log has a tool_call row.
	rows, _ := deps.Audit.GetByKind("tool_call", 100)
	if len(rows) < 1 {
		t.Fatalf("expected tool_call rows; got 0")
	}
}

// jsonString JSON-encodes a string with surrounding
// quotes. Used to embed a JSON-encoded plan into a
// mock OpenAI response.
func jsonString(s string) string {
	raw, _ := json.Marshal(s)
	return string(raw)
}
