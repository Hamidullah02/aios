package cli

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Hamidullah02/aios/aios-system/internal/core/schemas"
	"github.com/Hamidullah02/aios/aios-system/internal/models"
	"github.com/Hamidullah02/aios/aios-system/internal/planner"
)

// TestPlannerCanPlanAGitTool is the end-to-end proof that
// the doc-set tools are reachable from a plan.
//
// Before this change the planner carried a hand-written table
// of five tools, and the verifier rejected any plan naming
// another one. So a goal like "commit the change" could not be
// planned at all: the model was never told git_commit existed,
// and even if it had guessed the name the verifier would have
// rejected the plan as naming an unknown tool.
//
// The test drives the real planner with a real model stub and
// asserts the plan survives both the planner and the verifier.
func TestPlannerCanPlanAGitTool(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)

	// A plan that commits the staged change.
	plan := schemas.TaskPlan{
		ID:   "fix-git",
		Goal: "commit the change",
		Tasks: []schemas.TaskSpec{
			{
				ID:          "t1",
				Description: "check the working tree",
				Tool:        "git_status",
				Acceptance:  "ok",
			},
			{
				ID:          "t2",
				Description: "commit the change",
				Tool:        "git_commit",
				Acceptance:  "ok",
				Inputs:      []string{"message=add a note"},
			},
		},
		Tools: []string{"git_status", "git_commit"},
		Model: "stub-1",
	}
	raw, _ := json.Marshal(plan)

	stub := &stubAdapter{provider: "openai", resp: []models.Response{
		{Text: string(raw)},
	}}
	reg, _ := models.NewRegistry([]models.Capability{
		{Model: "stub-1", Provider: "openai", ContextWindow: 128000, SupportsTools: true},
	})
	prices := models.NewPriceTable([]models.Price{
		{Model: "stub-1", InputPer1K: 0.001, OutputPer1K: 0.002},
	}, nil)
	router := models.NewRouter(map[string]models.Adapter{"openai": stub}, reg, prices)

	toolSpecs := plannerToolSpecs(testBus(t))
	pl, err := planner.NewPlanner(planner.Config{
		Router:       router,
		Tools:        toolSpecs,
		Verifier:     planner.NewCodeFirstVerifier(toolSpecs),
		DefaultModel: "stub-1",
	})
	if err != nil {
		t.Fatal(err)
	}

	got, err := pl.Plan(context.Background(), "commit the change", nil)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if len(got.Tasks) != 2 {
		t.Fatalf("tasks = %d, want 2", len(got.Tasks))
	}
	if got.Tasks[0].Tool != "git_status" {
		t.Errorf("task 1 tool = %q, want git_status", got.Tasks[0].Tool)
	}
	if got.Tasks[1].Tool != "git_commit" {
		t.Errorf("task 2 tool = %q, want git_commit", got.Tasks[1].Tool)
	}
}

// TestPlannerPromptOffersTheDocSetTools checks the prompt the
// model actually receives.
//
// The plan above is a proof that the verifier accepts these
// tools. This checks the other half: the model has to be told
// they exist, or it cannot choose them. A tool that is
// dispatchable but absent from the prompt is a tool the agent
// does not have.
func TestPlannerPromptOffersTheDocSetTools(t *testing.T) {
	toolSpecs := plannerToolSpecs(testBus(t))

	prompt, err := planner.RenderUser(planner.PromptData{
		Goal:  "commit the change",
		Tools: toolSpecSlice(toolSpecs),
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{
		"git_commit", "git_status", "edit", "apply_patch", "grep", "code_search",
	} {
		if !strings.Contains(prompt, name) {
			t.Errorf("prompt does not mention %q:\n%s", name, prompt)
		}
	}
}

// TestPlannerPromptCarriesTheInputSchema checks that the
// prompt includes each tool's JSON schema.
//
// The schema is how the model learns the argument names. A
// prompt with the tool name but no schema produces a plan
// whose inputs the tool rejects at run time.
func TestPlannerPromptCarriesTheInputSchema(t *testing.T) {
	toolSpecs := plannerToolSpecs(testBus(t))

	prompt, err := planner.RenderUser(planner.PromptData{
		Goal:  "commit the change",
		Tools: toolSpecSlice(toolSpecs),
	})
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(prompt, "old_string") {
		t.Errorf("prompt is missing the edit tool's argument names:\n%s", prompt)
	}
	if !strings.Contains(prompt, "message") {
		t.Errorf("prompt is missing the git_commit argument names:\n%s", prompt)
	}
}

// toolSpecSlice converts the planner's map into the slice the
// prompt renderer takes.
func toolSpecSlice(specs map[string]planner.ToolSpec) []planner.ToolSpec {
	out := make([]planner.ToolSpec, 0, len(specs))
	for _, spec := range specs {
		out = append(out, spec)
	}
	return out
}

// initGitRepo makes dir a git repository with one commit, so
// the git tools have something to work against.
func initGitRepo(t *testing.T, dir string) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@example.com",
			"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@example.com",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
		}
	}
	run("init", "-q", "-b", "main")
	run("config", "user.email", "test@example.com")
	run("config", "user.name", "test")
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", "README.md")
	run("commit", "-q", "-m", "initial")
}
