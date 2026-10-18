package tools

import (
	"errors"
	"strings"
	"testing"

	"github.com/sadi/aios-system/internal/sandbox"
)

// fakeRunner is a deterministic stand-in for sandbox.Runner.
// The "commands" map returns the desired SandboxResult for a
// given first argument; otherwise the run returns the
// default error.
type fakeRunner struct {
	commands map[string]sandbox.SandboxResult
	err      error
}

func (f *fakeRunner) Run(opts sandbox.RunOptions) (sandbox.SandboxResult, error) {
	if len(opts.Command) == 0 {
		return sandbox.SandboxResult{}, errors.New("no command")
	}
	if f.err != nil {
		return sandbox.SandboxResult{}, f.err
	}
	if r, ok := f.commands[opts.Command[0]]; ok {
		return r, nil
	}
	return sandbox.SandboxResult{}, errors.New("unknown command: " + opts.Command[0])
}

func TestRunShell_Happy(t *testing.T) {
	r := RunShellTool{Runner: &fakeRunner{
		commands: map[string]sandbox.SandboxResult{
			"echo": {ExitCode: 0, Stdout: "hi\n", DurationMs: 5},
		},
	}}
	res, err := r.Run(map[string]any{"command": []string{"echo", "hi"}}, Context{})
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK {
		t.Fatalf("OK = false, summary = %s", res.Summary)
	}
	if res.Output["stdout"] != "hi\n" {
		t.Fatalf("stdout = %v", res.Output["stdout"])
	}
}

func TestRunShell_NonZero(t *testing.T) {
	r := RunShellTool{Runner: &fakeRunner{
		commands: map[string]sandbox.SandboxResult{
			"false": {ExitCode: 1, Stderr: "boom"},
		},
	}}
	res, err := r.Run(map[string]any{"command": []string{"false"}}, Context{})
	if err != nil {
		t.Fatal(err)
	}
	if res.OK {
		t.Fatal("OK = true for exit 1")
	}
	if !strings.Contains(res.Summary, "exit 1") {
		t.Fatalf("summary = %s", res.Summary)
	}
}

func TestRunShell_Timeout(t *testing.T) {
	runner := &fakeRunner{err: &sandbox.SandboxTimeout{TimeoutSec: 5, Wrapped: errors.New("deadline")}}
	r := RunShellTool{Runner: runner}
	_, err := r.Run(map[string]any{"command": []string{"sleep"}}, Context{})
	if !errors.Is(err, ErrToolTimeout) {
		t.Fatalf("err = %v, want ErrToolTimeout", err)
	}
}

func TestRunShell_MissingCommand(t *testing.T) {
	r := RunShellTool{Runner: &fakeRunner{}}
	_, err := r.Run(map[string]any{}, Context{})
	if err == nil {
		t.Fatal("expected error for missing command")
	}
}
