package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sadi/aios-system/internal/core/audit"
	"github.com/sadi/aios-system/internal/core/schemas"
	"github.com/sadi/aios-system/internal/models"
	"github.com/sadi/aios-system/internal/policy"
	"github.com/sadi/aios-system/internal/runtime"
)

// TestWritePlanYAML_RoundTrip writes a plan with every
// optional field set, reads it back with LoadPlan, and
// asserts the round-trip is exact. This is the contract
// `--plan-out` promises: a file a user can re-run with
// `aios run <path>` after editing.
func TestWritePlanYAML_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "plan.yaml")
	plan := &schemas.TaskPlan{
		ID:    "roundtrip-1",
		Goal:  "ship aios",
		Model: "gpt-4o",
		Tools: []string{"read_file", "search_files"},
		Budget: &schemas.Budget{
			CostUSD:      0.5,
			WallClockSec: 60,
		},
		Tasks: []schemas.TaskSpec{
			{
				ID:          "t1",
				Description: "read the design",
				Tool:        "read_file",
				Acceptance:  "doc exists",
				Inputs:      []string{"path=design.md", "limit=100"},
			},
			{
				ID:          "t2",
				Description: "search the code",
				Tool:        "search_files",
				Acceptance:  "matches found",
			},
		},
	}
	if err := WritePlanYAML(plan, path); err != nil {
		t.Fatal(err)
	}
	// The file must exist and be non-empty.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Fatal("wrote empty plan file")
	}
	// Read it back with the runtime's LoadPlan.
	got, err := LoadPlan(path)
	if err != nil {
		t.Fatalf("LoadPlan: %v", err)
	}
	if got.ID != plan.ID {
		t.Errorf("ID: got %q, want %q", got.ID, plan.ID)
	}
	if got.Goal != plan.Goal {
		t.Errorf("Goal: got %q, want %q", got.Goal, plan.Goal)
	}
	if got.Model != plan.Model {
		t.Errorf("Model: got %q, want %q", got.Model, plan.Model)
	}
	if len(got.Tools) != len(plan.Tools) {
		t.Errorf("Tools length: got %d, want %d", len(got.Tools), len(plan.Tools))
	}
	for i, tool := range plan.Tools {
		if got.Tools[i] != tool {
			t.Errorf("Tools[%d]: got %q, want %q", i, got.Tools[i], tool)
		}
	}
	if got.Budget == nil {
		t.Fatal("Budget is nil after round-trip")
	}
	if got.Budget.CostUSD != plan.Budget.CostUSD {
		t.Errorf("Budget.CostUSD: got %v, want %v", got.Budget.CostUSD, plan.Budget.CostUSD)
	}
	if got.Budget.WallClockSec != plan.Budget.WallClockSec {
		t.Errorf("Budget.WallClockSec: got %d, want %d", got.Budget.WallClockSec, plan.Budget.WallClockSec)
	}
	if len(got.Tasks) != len(plan.Tasks) {
		t.Fatalf("Tasks length: got %d, want %d", len(got.Tasks), len(plan.Tasks))
	}
	for i, want := range plan.Tasks {
		g := got.Tasks[i]
		if g.ID != want.ID || g.Description != want.Description ||
			g.Tool != want.Tool || g.Acceptance != want.Acceptance {
			t.Errorf("Tasks[%d] mismatch: got %+v, want %+v", i, g, want)
		}
		if len(g.Inputs) != len(want.Inputs) {
			t.Errorf("Tasks[%d].Inputs length: got %d, want %d", i, len(g.Inputs), len(want.Inputs))
		}
		for j, in := range want.Inputs {
			if g.Inputs[j] != in {
				t.Errorf("Tasks[%d].Inputs[%d]: got %q, want %q", i, j, g.Inputs[j], in)
			}
		}
	}
}

// TestWritePlanYAML_EmptyPath asserts fail-closed on
// empty path. The CLI parser rejects empty --plan-out
// at the boundary; the writer itself also rejects it
// for callers that bypass the parser.
func TestWritePlanYAML_EmptyPath(t *testing.T) {
	plan := &schemas.TaskPlan{ID: "x", Goal: "y", Tasks: []schemas.TaskSpec{
		{ID: "t1", Description: "d", Acceptance: "a"},
	}}
	if err := WritePlanYAML(plan, ""); err == nil {
		t.Fatal("expected error for empty path")
	}
}

// TestWritePlanYAML_NilPlan asserts fail-closed on nil plan.
func TestWritePlanYAML_NilPlan(t *testing.T) {
	if err := WritePlanYAML(nil, "x.yaml"); err == nil {
		t.Fatal("expected error for nil plan")
	}
}

// TestWritePlanYAML_InvalidPlan asserts fail-closed on
// a plan that fails validation. The writer calls
// plan.Validate as a defence-in-depth check so a bad
// file is never written.
func TestWritePlanYAML_InvalidPlan(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "plan.yaml")
	plan := &schemas.TaskPlan{ID: "", Goal: "y", Tasks: []schemas.TaskSpec{}} // both invalid
	if err := WritePlanYAML(plan, path); err == nil {
		t.Fatal("expected validation error")
	}
	// And the file must not exist on disk.
	if _, err := os.Stat(path); err == nil {
		t.Errorf("file was written despite validation error: %s", path)
	}
}

// TestWritePlanYAML_CreatesParentDir asserts that the
// writer creates the parent directory when it does
// not exist. `aios fix "..." --plan-out var/plans/x.yaml`
// is the canonical case: a fresh `aios init` does not
// create var/plans/, so the writer must.
func TestWritePlanYAML_CreatesParentDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "deep", "nested", "plan.yaml")
	plan := &schemas.TaskPlan{ID: "x", Goal: "y", Tasks: []schemas.TaskSpec{
		{ID: "t1", Description: "d", Acceptance: "a"},
	}}
	if err := WritePlanYAML(plan, path); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("file not created: %v", err)
	}
}

// TestRunFix_PlanOutWritesFile asserts the end-to-end
// path: runFix with --plan-out writes the planner's
// plan to the requested path, prints the path, and the
// file is loadable by LoadPlan. This is the v0.34.0
// feature: a user can `aios fix "ship aios"
// --plan-out var/plans/ship.yaml` and then `aios run
// var/plans/ship.yaml` to re-run the same plan (or
// edit it first).
func TestRunFix_PlanOutWritesFile(t *testing.T) {
	dir := t.TempDir()
	planOut := filepath.Join(dir, "plans", "ship.yaml")
	deps := newFixTestDeps(t, dir, []models.Response{
		{Text: validFixPlanJSON(), Usage: models.Usage{InputTokens: 100, OutputTokens: 80, Model: "stub-1"}},
	})
	defer deps.Close()
	args := &Args{
		Command:   CmdFix,
		Workspace: dir,
		Objective: "read the design doc",
		PlanOut:   planOut,
	}
	var stdout, stderr bytes.Buffer
	code := runFix(context.Background(), args, deps, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("code: %d, stderr: %s", code, stderr.String())
	}
	// The plan file must exist and be loadable.
	if _, err := os.Stat(planOut); err != nil {
		t.Fatalf("plan file not created: %v", err)
	}
	got, err := LoadPlan(planOut)
	if err != nil {
		t.Fatalf("LoadPlan on the saved file: %v", err)
	}
	if got.ID != "fix-1" {
		t.Errorf("saved plan ID: got %q, want fix-1", got.ID)
	}
	// stdout should mention the path so the user can
	// copy-paste it.
	if !strings.Contains(stdout.String(), planOut) {
		t.Errorf("stdout should mention plan path %q; got %q", planOut, stdout.String())
	}
	// The run also wrote audit rows.
	rows, _ := deps.Audit.GetByKind("tool_call", 100)
	if len(rows) < 1 {
		t.Errorf("expected tool_call rows; got 0")
	}
}

// TestRunFix_PlanOutThenRunAgain is the full round-trip:
// runFix writes a plan, then a fresh runRun executes
// the saved plan. The two runs produce independent
// audit chains, but the plan file is the same.
//
// This is the v0.34.0 use case the user asked for:
// "I want to re-run a planned plan."
func TestRunFix_PlanOutThenRunAgain(t *testing.T) {
	dir := t.TempDir()
	planOut := filepath.Join(dir, "plans", "ship.yaml")
	deps := newFixTestDeps(t, dir, []models.Response{
		{Text: validFixPlanJSON(), Usage: models.Usage{InputTokens: 100, OutputTokens: 80, Model: "stub-1"}},
	})
	defer deps.Close()
	// First run: fix with --plan-out.
	args := &Args{
		Command:   CmdFix,
		Workspace: dir,
		Objective: "read the design doc",
		PlanOut:   planOut,
	}
	var stdout, stderr bytes.Buffer
	code := runFix(context.Background(), args, deps, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("first run: code: %d, stderr: %s", code, stderr.String())
	}
	// Second run: aios run on the saved plan. Use a
	// fresh audit log so we can count this run's rows
	// independently.
	log2, err := audit.Open(audit.Config{Path: filepath.Join(dir, "aios2.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer log2.Close()
	gw := policy.New([]policy.Rule{
		{ID: "allow-read", Match: policy.Match{Tool: "read_file"}, Action: policy.ActionAllow},
	}, log2)
	bus, worker, verifier := buildTestAgentLoop(t, dir, gw, log2)
	deps2 := &Deps{
		Audit:    log2,
		Policy:   gw,
		Resume:   runtime.NewResumeHandler(filepath.Join(dir, "checkpoints2")),
		Bus:      bus,
		Worker:   worker,
		Verifier: verifier,
	}
	runArgs := &Args{
		Command:   CmdRun,
		Workspace: dir,
		PlanPath:  planOut,
	}
	stdout2, stderr2 := &bytes.Buffer{}, &bytes.Buffer{}
	code2 := runRun(context.Background(), runArgs, deps2, stdout2, stderr2)
	if code2 != ExitOK {
		t.Fatalf("second run: code: %d, stderr: %s", code2, stderr2.String())
	}
	// The second run produced tool_call rows.
	rows, _ := log2.GetByKind("tool_call", 100)
	if len(rows) < 1 {
		t.Errorf("second run: expected tool_call rows; got 0")
	}
}
