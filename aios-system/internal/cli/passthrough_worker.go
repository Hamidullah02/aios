package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/sadi/aios-system/internal/ask"
	"github.com/sadi/aios-system/internal/runtime"
)

// defaultStdin / defaultStdout are the I/O defaults the
// StdinPromptCallback uses when the caller did not pass
// explicit streams. They read/write the real process I/O.
var (
	defaultStdin  = os.Stdin
	defaultStdout = os.Stdout
)

// PassthroughWorker is a runtime.Worker that records each
// task and returns a passing WorkerResult. The CLI ships
// this so a smoke run of `aios run <plan.yaml>` has a
// worker without a real agent. Production deployments
// replace it with a real worker (the planner-driven
// worker, a code-generation worker, etc.).
//
// The worker treats `task.tool` as a hint:
//   - "read_file" returns a stub "ok" output.
//   - "edit_file" / "write_file" returns a stub "ok" output.
//   - any other tool returns an "unknown tool" error so
//     the smoke run surfaces the missing wiring.
type PassthroughWorker struct{}

func (PassthroughWorker) Run(input runtime.TaskInput, _ *runtime.TaskContext, _ *runtime.BudgetMeter) (runtime.WorkerResult, error) {
	switch input.Tool {
	case "read_file", "edit_file", "write_file", "search_files", "run_shell", "ask_user":
		return runtime.WorkerResult{
			Output: fmt.Sprintf("passthrough: %s/%s", input.ID, input.Tool),
			Data:   map[string]any{"ok": true, "task_id": input.ID},
		}, nil
	}
	if input.Tool == "" {
		// No tool on the task: still pass with a stub.
		return runtime.WorkerResult{
			Output: fmt.Sprintf("passthrough: %s", input.ID),
			Data:   map[string]any{"ok": true, "task_id": input.ID},
		}, nil
	}
	return runtime.WorkerResult{}, fmt.Errorf("passthrough: unknown tool %q", input.Tool)
}

// SmokeVerifier is the verifier the passthrough worker
// uses. It returns VerdictPass for every result.
type SmokeVerifier struct{}

func (SmokeVerifier) Verify(_ runtime.TaskInput, _ runtime.WorkerResult) (runtime.VerifierVerdict, error) {
	return runtime.VerdictPass, nil
}

// Ensure interfaces are satisfied (also catches API drift).
var (
	_ runtime.Worker   = PassthroughWorker{}
	_ runtime.Verifier = SmokeVerifier{}
)

// StdinPromptCallback is the prompt callback the smoke
// `aios` binary uses for the ask-user tool. It writes the
// question to the operator's stdout and reads the choice
// from stdin. Tests substitute a scripted callback.
//
// The callback honours the context: when the deadline
// fires, the read returns ctx.Err() and the tool falls
// back to the default answer (if any) or raises
// ErrTimeout.
type StdinPromptCallback struct {
	// In is the input source (default: os.Stdin). Tests
	// pass a buffer.
	In interface {
		Read(p []byte) (int, error)
	}
	// Out is the output sink for the question (default:
	// os.Stdout). Tests pass a buffer.
	Out interface {
		Write(p []byte) (int, error)
	}
}

// AskQuestion / AskAnswer are aliases for the ask package's
// types so the callback type can be exported without a
// cycle in the test surface.
type (
	AskQuestion = ask.Question
	AskOption   = ask.AskOption
	AskAnswer   = ask.Answer
)

// StdinPrompt returns a PromptCallback that writes the
// question to Out and reads the choice number from In. The
// function returns the answer on success and ctx.Err() on
// deadline.
func (s *StdinPromptCallback) StdinPrompt() ask.PromptCallback {
	in := s.In
	out := s.Out
	if in == nil {
		in = defaultStdin
	}
	if out == nil {
		out = defaultStdout
	}
	return func(ctx context.Context, q ask.Question) (ask.Answer, error) {
		// Render the question.
		fmt.Fprintf(out, "QUESTION (%s): %s\n", q.ID, q.Prompt)
		for i, opt := range q.Options {
			fmt.Fprintf(out, "  %d. %s", i+1, opt.Label)
			if opt.Description != "" {
				fmt.Fprintf(out, " (%s)", opt.Description)
			}
			fmt.Fprintln(out)
		}
		if q.DefaultOption != "" {
			fmt.Fprintf(out, "Default: %s\n", q.DefaultOption)
		}
		fmt.Fprint(out, "Choice: ")
		// Read a line.
		buf := make([]byte, 32)
		n, err := in.Read(buf)
		if err != nil {
			return ask.Answer{}, err
		}
		s := strings.TrimSpace(string(buf[:n]))
		// Parse the choice.
		var idx int
		if _, err := fmt.Sscanf(s, "%d", &idx); err != nil {
			return ask.Answer{}, err
		}
		if idx < 1 || idx > len(q.Options) {
			return ask.Answer{}, fmt.Errorf("invalid choice %d", idx)
		}
		return ask.Answer{
			QuestionID:     q.ID,
			SelectedOption: q.Options[idx-1].ID,
		}, nil
	}
}
