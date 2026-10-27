package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/Hamidullah02/aios/aios-system/internal/ask"
	"github.com/Hamidullah02/aios/aios-system/internal/runtime"
)

func TestPassthroughWorker_KnownTools(t *testing.T) {
	w := PassthroughWorker{}
	for _, tool := range []string{"read_file", "edit_file", "write_file", "search_files", "run_shell", "ask_user"} {
		res, err := w.Run(runtime.TaskInput{ID: "t1", Tool: tool}, &runtime.TaskContext{}, &runtime.BudgetMeter{})
		if err != nil {
			t.Errorf("tool %s: %v", tool, err)
		}
		if res.Output == "" {
			t.Errorf("tool %s: empty output", tool)
		}
	}
}

func TestPassthroughWorker_EmptyTool(t *testing.T) {
	w := PassthroughWorker{}
	res, err := w.Run(runtime.TaskInput{ID: "t1"}, &runtime.TaskContext{}, &runtime.BudgetMeter{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Output == "" {
		t.Fatal("empty output")
	}
}

func TestPassthroughWorker_UnknownTool(t *testing.T) {
	w := PassthroughWorker{}
	_, err := w.Run(runtime.TaskInput{ID: "t1", Tool: "rm_rf"}, &runtime.TaskContext{}, &runtime.BudgetMeter{})
	if err == nil {
		t.Fatal("expected error for unknown tool")
	}
}

func TestSmokeVerifier_AlwaysPass(t *testing.T) {
	v := SmokeVerifier{}
	verdict, err := v.Verify(runtime.TaskInput{}, runtime.WorkerResult{})
	if err != nil {
		t.Fatal(err)
	}
	if verdict != runtime.VerdictPass {
		t.Fatalf("verdict: %v", verdict)
	}
}

func TestStdinPromptCallback(t *testing.T) {
	in := strings.NewReader("2\n")
	out := &bytes.Buffer{}
	cb := &StdinPromptCallback{In: in, Out: out}
	prompt := cb.StdinPrompt()
	q := ask.Question{
		ID:     "q-1",
		Prompt: "Which?",
		Options: []ask.AskOption{
			{ID: "a", Label: "A"},
			{ID: "b", Label: "B"},
		},
	}
	ans, err := prompt(context.Background(), q)
	if err != nil {
		t.Fatal(err)
	}
	if ans.SelectedOption != "b" {
		t.Errorf("selected: %q", ans.SelectedOption)
	}
	if !strings.Contains(out.String(), "Which?") {
		t.Errorf("output should contain question: %s", out.String())
	}
}

func TestStdinPromptCallback_InvalidChoice(t *testing.T) {
	in := strings.NewReader("5\n")
	out := &bytes.Buffer{}
	cb := &StdinPromptCallback{In: in, Out: out}
	prompt := cb.StdinPrompt()
	q := ask.Question{
		ID:     "q-1",
		Prompt: "Which?",
		Options: []ask.AskOption{
			{ID: "a", Label: "A"},
		},
	}
	_, err := prompt(context.Background(), q)
	if err == nil {
		t.Fatal("expected error for invalid choice")
	}
}

func TestStdinPromptCallback_Default(t *testing.T) {
	// Output should mention the default option.
	in := strings.NewReader("1\n")
	out := &bytes.Buffer{}
	cb := &StdinPromptCallback{In: in, Out: out}
	prompt := cb.StdinPrompt()
	q := ask.Question{
		ID:            "q-1",
		Prompt:        "Which?",
		Options:       []ask.AskOption{{ID: "a", Label: "A"}},
		DefaultOption: "a",
	}
	_, _ = prompt(context.Background(), q)
	if !strings.Contains(out.String(), "Default: a") {
		t.Errorf("output should mention default: %s", out.String())
	}
}
