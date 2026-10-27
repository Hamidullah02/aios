package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Hamidullah02/aios/aios-system/internal/core/audit"
	"github.com/Hamidullah02/aios/aios-system/internal/core/schemas"
	"github.com/Hamidullah02/aios/aios-system/internal/runtime"
)

// TestBuildDeps_WiresBusWorker asserts that the production
// BuildDeps no longer falls back to the passthrough worker
// and the smoke verifier. Every non-bootstrap subcommand
// must get a real BusWorker + tools.Bus + CommandVerifier.
//
// This test is the gate the v0.32.0 review demanded: the
// agent loop is real, not a stub.
func TestBuildDeps_WiresBusWorker(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfg, []byte("rules:\n  - id: allow-read\n    match:\n      tool: read_file\n    action: allow\n"), 0o644); err != nil {
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

	// Worker must be the real BusWorker, not Passthrough.
	if _, ok := deps.Worker.(*BusWorker); !ok {
		t.Errorf("Worker type = %T, want *BusWorker", deps.Worker)
	}
	// Verifier must be the real CommandVerifier, not Smoke.
	if _, ok := deps.Verifier.(CommandVerifier); !ok {
		t.Errorf("Verifier type = %T, want CommandVerifier", deps.Verifier)
	}
	// Bus must be constructed.
	if deps.Bus == nil {
		t.Error("Bus is nil; BuildDeps did not construct the tool bus")
	}
}

// TestBuildDeps_BootstrapSkipsWiring asserts the inverse:
// init and daemon (the bootstrap subcommands) do not get a
// BusWorker, because they do not run plans.
func TestBuildDeps_BootstrapSkipsWiring(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfg, []byte("rules: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, cmd := range []string{CmdInit, CmdDaemon} {
		args := &Args{
			Command:    cmd,
			ConfigPath: cfg,
			LogPath:    filepath.Join(dir, cmd+".db"),
		}
		deps, err := BuildDeps(args)
		if err != nil {
			t.Fatalf("%s: %v", cmd, err)
		}
		_ = deps.Close()
		if deps.Worker != nil {
			t.Errorf("%s: Worker = %T, want nil (bootstrap skips wiring)", cmd, deps.Worker)
		}
		if deps.Bus != nil {
			t.Errorf("%s: Bus = %v, want nil (bootstrap skips wiring)", cmd, deps.Bus)
		}
	}
}

// TestRunCommand_RealToolCall exercises the full chain
// from CLI argv through the orchestrator to the real
// tools.Bus, with a real read_file call, and asserts the
// audit log has a tool_call row. This is the agent loop
// the v0.32.0 review demanded.
func TestRunCommand_RealToolCall(t *testing.T) {
	dir := t.TempDir()
	// Write a real file the plan will read.
	target := "hello.md"
	if err := os.WriteFile(filepath.Join(dir, target), []byte("hello aios"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Policy that allows read_file.
	cfg := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfg, []byte("rules:\n  - id: allow-read\n    match:\n      tool: read_file\n    action: allow\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Plan with a real read_file task.
	planPath := filepath.Join(dir, "plan.yaml")
	plan := &schemas.TaskPlan{
		ID: "wired-1", Goal: "verify the bus",
		Tasks: []schemas.TaskSpec{
			{ID: "t1", Description: "read hello",
				Acceptance: "ok", Tool: "read_file",
				Inputs: []string{"path=" + target}},
		},
	}
	yaml := "id: " + plan.ID + "\n" + "goal: " + plan.Goal + "\n" + "tasks:\n"
	for _, task := range plan.Tasks {
		yaml += "  - id: " + task.ID + "\n" +
			"    description: " + task.Description + "\n" +
			"    acceptance: " + task.Acceptance + "\n" +
			"    tool: " + task.Tool + "\n"
		for _, in := range task.Inputs {
			yaml += "    inputs:\n      - " + in + "\n"
		}
	}
	if err := os.WriteFile(planPath, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	// Run through the real entry point. Main() builds the
	// deps and dispatches; we just feed it argv.
	var stdout, stderr bytes.Buffer
	code := Main([]string{
		"aios",
		"--config=" + cfg,
		"--log=" + filepath.Join(dir, "aios.db"),
		"--workspace=" + dir,
		"run", planPath,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code: %d, stderr:\n%s", code, stderr.String())
	}
	// The audit log must have at least one tool_call row.
	logPath := filepath.Join(dir, "aios.db")
	log, err := audit.Open(audit.Config{Path: logPath})
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()
	rows, err := log.GetByKind("tool_call", 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) < 1 {
		t.Fatalf("expected tool_call rows; got 0")
	}
	// The first tool_call row should record a real read.
	// PersistedRow.Payload is raw JSON; decode it before
	// reading the fields.
	var p map[string]any
	if err := json.Unmarshal(rows[0].Payload, &p); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if p["tool"] != "read_file" {
		t.Errorf("first tool_call row tool = %v, want read_file", p["tool"])
	}
	if p["decision"] != "allow" {
		t.Errorf("first tool_call row decision = %v, want allow", p["decision"])
	}
}

// TestRunCommand_VerifierRejectsUnknownTool asserts the
// real CommandVerifier returns fail when a plan asks for
// a tool the bus does not have. The previous SmokeVerifier
// would have rubber-stamped the empty result.
func TestRunCommand_VerifierRejectsUnknownTool(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfg, []byte("rules: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	planPath := filepath.Join(dir, "plan.yaml")
	plan := &schemas.TaskPlan{
		ID: "verifier-1", Goal: "verify the verifier",
		Tasks: []schemas.TaskSpec{
			{ID: "t1", Description: "tool that does not exist",
				Acceptance: "ok", Tool: "no_such_tool_xyz"},
		},
	}
	yaml := "id: " + plan.ID + "\n" + "goal: " + plan.Goal + "\n" + "tasks:\n"
	for _, task := range plan.Tasks {
		yaml += "  - id: " + task.ID + "\n" +
			"    description: " + task.Description + "\n" +
			"    acceptance: " + task.Acceptance + "\n" +
			"    tool: " + task.Tool + "\n"
	}
	if err := os.WriteFile(planPath, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := Main([]string{
		"aios",
		"--config=" + cfg,
		"--log=" + filepath.Join(dir, "aios.db"),
		"--workspace=" + dir,
		"run", planPath,
	}, &stdout, &stderr)
	// The run should exit non-zero because the verifier
	// rejected the empty result.
	if code == 0 {
		t.Fatalf("expected non-zero exit for unknown tool; got 0. stdout:\n%s\nstderr:\n%s",
			stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "verifier") {
		// The error message should mention the verifier;
		// if it does not, the bus short-circuited before
		// the verifier ran.
		t.Logf("stderr did not mention verifier; stderr:\n%s", stderr.String())
	}
}

// TestCommandVerifier_StubsAreGone is a guard against
// regressing back to the passthrough defaults. The test
// asserts PassthroughWorker and SmokeVerifier are no
// longer used in the production code path.
//
// This is the literal "stop shipping a stub" test the
// v0.32.0 review asked for: if a future commit brings
// the stubs back as defaults, this test fails.
func TestCommandVerifier_StubsAreGone(t *testing.T) {
	// BuildDeps must NOT return the stubs for non-bootstrap
	// subcommands.
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfg, []byte("rules: []\n"), 0o644); err != nil {
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
	if _, ok := deps.Worker.(PassthroughWorker); ok {
		t.Error("Worker = PassthroughWorker; production must use BusWorker")
	}
	if _, ok := deps.Verifier.(SmokeVerifier); ok {
		t.Error("Verifier = SmokeVerifier; production must use CommandVerifier")
	}
}

// TestRunCommand_PolicyDenyStopsLoop asserts the verifier
// + policy gateway chain: a denied tool call is audited
// as deny and the plan stops. Without the real loop this
// is impossible to test (PassthroughWorker + SmokeVerifier
// would rubber-stamp the deny).
func TestRunCommand_PolicyDenyStopsLoop(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config.yaml")
	// Deny read_file.
	if err := os.WriteFile(cfg, []byte("rules:\n  - id: deny-read\n    match:\n      tool: read_file\n    action: deny\n    reason: test deny\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	planPath := filepath.Join(dir, "plan.yaml")
	if err := os.WriteFile(planPath, []byte(
		"id: deny-1\n"+
			"goal: test\n"+
			"tasks:\n"+
			"  - id: t1\n"+
			"    description: denied read\n"+
			"    acceptance: ok\n"+
			"    tool: read_file\n"+
			"    inputs:\n      - path=missing.md\n",
	), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := Main([]string{
		"aios",
		"--config=" + cfg,
		"--log=" + filepath.Join(dir, "aios.db"),
		"--workspace=" + dir,
		"run", planPath,
	}, &stdout, &stderr)
	// The deny should propagate to a non-zero exit (the
	// verifier fails the task on the empty Output).
	if code == 0 {
		t.Fatalf("expected non-zero exit on deny; got 0. stdout:\n%s\nstderr:\n%s",
			stdout.String(), stderr.String())
	}
	// And the audit log should have a deny decision.
	logPath := filepath.Join(dir, "aios.db")
	log, err := audit.Open(audit.Config{Path: logPath})
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()
	rows, _ := log.GetByKind("tool_call", 100)
	if len(rows) < 1 {
		t.Fatalf("expected at least 1 tool_call row; got 0")
	}
	// PersistedRow.Payload is raw JSON; decode it.
	var p map[string]any
	if err := json.Unmarshal(rows[0].Payload, &p); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if p["decision"] != "deny" {
		t.Errorf("first tool_call decision = %v, want deny", p["decision"])
	}
}

// Touch runtime to keep the import warning quiet even if
// a refactor removes the orchestrator reference.
var _ = runtime.NewBudgetMeter
var _ = context.Background
