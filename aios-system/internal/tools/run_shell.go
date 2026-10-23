package tools

import (
	"errors"
	"fmt"

	"github.com/sadi/aios-system/internal/sandbox"
)

// RunShellTool runs a shell command through the sandbox
// runner (REQ-TOL-SH-001 to 004).
//
// The tool accepts:
//   - command (required, []string): the argv. The first
//     element is the executable; the rest are arguments.
//     We do not run a shell — the args go straight to
//     os/exec via the sandbox runner.
//   - timeout (optional, int, seconds, default 60).
//   - cwd (optional): the working directory.
//   - env (optional, map[string]string): the environment.
//
// The runner is supplied at construction time so the tool
// can be tested with a fake runner.
type RunShellTool struct {
	Runner sandbox.Runner
}

func (RunShellTool) Name() string { return "run_shell" }

func (r RunShellTool) Run(args map[string]any, ctx Context) (Result, error) {
	cmd, _ := args["command"].([]string)
	if len(cmd) == 0 {
		// Allow the caller to pass a single string for
		// "command" and split it themselves. The doc set
		// says argv, so this is a strict check.
		return Result{OK: false, Summary: "missing command"}, fmt.Errorf("tools: run_shell: command is required (argv)")
	}
	timeout, _ := toInt(args["timeout"])
	cwd, _ := args["cwd"].(string)
	var env map[string]string
	if e, ok := args["env"].(map[string]any); ok {
		env = make(map[string]string, len(e))
		for k, v := range e {
			env[k] = fmt.Sprint(v)
		}
	} else if e, ok := args["env"].(map[string]string); ok {
		env = e
	}

	runner := r.Runner
	if runner == nil {
		runner = sandbox.DefaultRunner()
	}
	res, err := runner.Run(sandbox.RunOptions{
		Command: cmd,
		Env:     env,
		Cwd:     cwd,
		Timeout: timeout,
	})
	if err != nil {
		// SandboxTimeout from the runner maps to ToolTimeout
		// so callers can use errors.Is(err, ErrToolTimeout).
		var st *sandbox.SandboxTimeout
		if errors.As(err, &st) {
			return Result{
				OK:      false,
				Output:  map[string]any{"exit_code": res.ExitCode, "stdout": res.Stdout, "stderr": res.Stderr},
				Summary: "timed out",
			}, &ToolTimeoutError{Tool: "run_shell", Timeout: fmt.Sprintf("%ds", st.TimeoutSec)}
		}
		return Result{
			OK:      false,
			Output:  map[string]any{"exit_code": res.ExitCode, "stdout": res.Stdout, "stderr": res.Stderr},
			Summary: "run failed",
		}, err
	}
	return Result{
		OK: res.ExitCode == 0,
		Output: map[string]any{
			"exit_code":   res.ExitCode,
			"stdout":      res.Stdout,
			"stderr":      res.Stderr,
			"duration_ms": res.DurationMs,
		},
		Summary: fmt.Sprintf("exit %d in %dms", res.ExitCode, res.DurationMs),
	}, nil
}
