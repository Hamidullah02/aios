package cli

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sadi/aios-system/internal/core/audit"
	"github.com/sadi/aios-system/internal/models"
	"github.com/sadi/aios-system/internal/tools"
)

// TestWireVerifyTools_AlwaysReturnsRunTests asserts the
// unconditional part of the wiring: the bus has a
// run_tests tool after wireVerifyTools is called with a
// nil router.
func TestWireVerifyTools_AlwaysReturnsRunTests(t *testing.T) {
	log, err := audit.Open(audit.Config{Path: ":memory:"})
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()
	tools := wireVerifyTools(nil, log)
	if _, ok := tools["run_tests"]; !ok {
		t.Fatal("wireVerifyTools(nil) did not return run_tests")
	}
	// self_check is NOT in the map when router is nil.
	if _, ok := tools["self_check"]; ok {
		t.Error("wireVerifyTools(nil) returned self_check; expected nil")
	}
}

// TestWireVerifyTools_RouterGetsSelfCheck asserts the
// conditional part: when a router is provided, the
// self_check tool is included.
func TestWireVerifyTools_RouterGetsSelfCheck(t *testing.T) {
	log, err := audit.Open(audit.Config{Path: ":memory:"})
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()
	router := newStubRouter(t)
	tools := wireVerifyTools(router, log)
	if _, ok := tools["self_check"]; !ok {
		t.Error("wireVerifyTools(router) did not return self_check")
	}
	if _, ok := tools["run_tests"]; !ok {
		t.Error("wireVerifyTools(router) dropped run_tests")
	}
}

// TestWireSelfCheckOnly_NilRouterReturnsNil asserts the
// deferred-wiring path: no router, no self_check tool.
func TestWireSelfCheckOnly_NilRouterReturnsNil(t *testing.T) {
	log, err := audit.Open(audit.Config{Path: ":memory:"})
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()
	if got := wireSelfCheckOnly(nil, log); got != nil {
		t.Errorf("wireSelfCheckOnly(nil) = %v, want nil", got)
	}
}

// TestWireSelfCheckOnly_RouterReturnsTool asserts the
// happy path: a router produces a working self_check
// tool.
func TestWireSelfCheckOnly_RouterReturnsTool(t *testing.T) {
	log, err := audit.Open(audit.Config{Path: ":memory:"})
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()
	tool := wireSelfCheckOnly(newStubRouter(t), log)
	if tool == nil {
		t.Fatal("wireSelfCheckOnly returned nil")
	}
	if tool.Name() != "self_check" {
		t.Errorf("name: %q, want self_check", tool.Name())
	}
}

// TestRouterJudge_CallsRouterAndReturnsText asserts the
// routerJudge adapter: it calls router.Complete and
// returns the response text. The test uses a stub
// router whose adapter records the request and
// returns a canned response.
func TestRouterJudge_CallsRouterAndReturnsText(t *testing.T) {
	stub := &fakeAdapter{text: "ok"}
	router := newStubRouterWith(t, stub)
	j := routerJudge{router: router, model: "gpt-4o"}
	got, err := j.Judge(context.Background(), "is this ok?")
	if err != nil {
		t.Fatal(err)
	}
	if got != stub.text {
		t.Errorf("Judge text: got %q, want %q", got, stub.text)
	}
	if stub.lastPrompt != "is this ok?" {
		t.Errorf("Judge prompt: got %q, want %q", stub.lastPrompt, "is this ok?")
	}
	if stub.lastModel != "gpt-4o" {
		t.Errorf("Judge model: got %q, want gpt-4o", stub.lastModel)
	}
}

// TestRouterJudge_NilRouterReturnsError asserts the
// nil-router case: the adapter returns an error so the
// SelfCheck tool surfaces a clear message instead of
// panicking.
func TestRouterJudge_NilRouterReturnsError(t *testing.T) {
	j := routerJudge{router: nil, model: "gpt-4o"}
	_, err := j.Judge(context.Background(), "x")
	if err == nil {
		t.Fatal("expected error for nil router")
	}
}

// TestBuildDeps_RunSubcommandHasRunTestsTool asserts the
// end-to-end wiring: a `run` subcommand gets a bus
// with run_tests available. The tool call itself may
// fail (the go-test parser needs `go test -json`), but
// the dispatch must not return ToolNotFoundError.
func TestBuildDeps_RunSubcommandHasRunTestsTool(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfg, []byte("rules:\n  - id: allow-tests\n    match:\n      tool: run_tests\n    action: allow\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	args := &Args{
		Command:    CmdRun,
		ConfigPath: cfg,
		LogPath:    filepath.Join(dir, "aios.db"),
		Workspace:  dir,
	}
	deps, err := BuildDeps(args)
	if err != nil {
		t.Fatal(err)
	}
	defer deps.Close()
	_, err = deps.Bus.Dispatch("run_tests", map[string]any{
		"command": "go test -count=1 -run=NONE ./doesnotexist",
		"runner":  "go-test",
	}, tools.Context{Actor: "t", TaskID: "t1"})
	// The error must not be a ToolNotFoundError. It
	// can be anything else (subprocess error, parse
	// error, policy error).
	if err != nil {
		var tnf *tools.ToolNotFoundError
		if errors.As(err, &tnf) {
			t.Errorf("run_tests is not registered in the bus: %v", err)
		}
		if strings.Contains(err.Error(), "tool not found") {
			t.Errorf("run_tests is not registered in the bus: %v", err)
		}
	}
}

// TestBuildDeps_RunSubcommandNoSelfCheck asserts the
// inverse: a `run` subcommand (no router) does NOT get
// self_check. Calling it returns ToolNotFoundError.
func TestBuildDeps_RunSubcommandNoSelfCheck(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfg, []byte("rules:\n  - id: allow-self-check\n    match:\n      tool: self_check\n    action: allow\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	args := &Args{
		Command:    CmdRun,
		ConfigPath: cfg,
		LogPath:    filepath.Join(dir, "aios.db"),
		Workspace:  dir,
	}
	deps, err := BuildDeps(args)
	if err != nil {
		t.Fatal(err)
	}
	defer deps.Close()
	_, err = deps.Bus.Dispatch("self_check", map[string]any{
		"result": "ok",
		"goal":   "test",
	}, tools.Context{Actor: "t", TaskID: "t1"})
	if err == nil {
		t.Fatal("expected error: self_check should not be wired for run")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected tool-not-found error; got %v", err)
	}
}

// --- helpers ---

// newStubRouter builds a *models.Router with the
// minimal fixture: one capability (gpt-4o -> openai),
// one adapter, and an empty price table. The Judge
// path only needs Router.Complete, which routes via
// the capability map.
func newStubRouter(t *testing.T) *models.Router {
	t.Helper()
	return newStubRouterWith(t, &fakeAdapter{text: "ok"})
}

// newStubRouterWith is the same as newStubRouter but
// lets the caller inject the adapter.
func newStubRouterWith(t *testing.T, adapter *fakeAdapter) *models.Router {
	t.Helper()
	reg, err := models.NewRegistry([]models.Capability{
		{Model: "gpt-4o", Provider: "openai", ContextWindow: 128000},
	})
	if err != nil {
		t.Fatal(err)
	}
	// Router.Complete reads prices to attach a USD
	// cost to the response. The Judge path does not
	// care about the cost, but the router errors
	// without a price row. We register a zero price
	// so the test is independent of the actual
	// pricing.
	return models.NewRouter(
		map[string]models.Adapter{"openai": adapter},
		reg,
		models.NewPriceTable(
			[]models.Price{
				{Model: "gpt-4o", InputPer1K: 0, OutputPer1K: 0},
			},
			nil,
		),
	)
}

// fakeAdapter is a tiny models.Adapter for tests.
// It implements Provider, Complete, and Stream.
type fakeAdapter struct {
	text       string
	lastPrompt string
	lastModel  string
}

// Provider is the Adapter method.
func (a *fakeAdapter) Provider() string { return "openai" }

// Complete is the Adapter method. It records the
// prompt and model, then returns a canned response.
func (a *fakeAdapter) Complete(_ context.Context, req models.Request) (models.Response, error) {
	a.lastPrompt = req.Prompt
	a.lastModel = req.Model
	return models.Response{Text: a.text, Usage: models.Usage{Model: req.Model}}, nil
}

// Stream is the Adapter method. The Judge path does
// not use streaming, so this returns an error.
func (a *fakeAdapter) Stream(_ context.Context, _ models.Request) (models.Stream, error) {
	return nil, errors.New("fakeAdapter: stream not supported")
}
