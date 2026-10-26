package tools

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

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
	cmd := resolveCommandArg(args["command"])
	if len(cmd) == 0 {
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

// resolveCommandArg turns the `command` argument into an argv.
//
// It accepts four shapes, because a plan can express a command
// in whichever of them is most natural:
//
//   - a []string, which is the canonical form the schema
//     describes;
//   - a []any of strings, which is what a JSON decode produces;
//   - a JSON array written as a string, which is what a plan
//     produces when it writes `command=["ls", "-la"]`;
//   - a plain string, which is what the plan's `key=value`
//     input format gives, because a value cannot carry a list.
//
// The string forms matter more than they look. The plan format
// is a list of `key=value` strings, so a plan cannot express a
// list value without encoding it in the string. The tool
// accepted only []string and rejected the string, so a plan
// saying `command=ls -la` failed with "command is required
// (argv)" -- an error that named neither the real problem nor
// the way out of it.
//
// A JSON array is decoded, not split on whitespace. Splitting
// would tear `["ls", "-la"]` into two fragments at the space
// after the comma and try to run a program called `["ls",`.
func resolveCommandArg(raw any) []string {
	switch v := raw.(type) {
	case []string:
		return v
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	case string:
		return splitCommandString(v)
	default:
		return nil
	}
}

// splitCommandString turns a string into an argv.
//
// A string that is a JSON array is decoded: that is how a plan
// writes a command whose arguments need quoting or spaces. Any
// other string is split on whitespace, which is enough for the
// ordinary case and is deliberately not a shell parse -- no
// quoting, no pipes, no redirection -- because the runner takes
// an argv and never goes through a shell.
func splitCommandString(s string) []string {
	trimmed := strings.TrimSpace(s)
	if strings.HasPrefix(trimmed, "[") {
		var list []string
		if err := json.Unmarshal([]byte(trimmed), &list); err == nil && len(list) > 0 {
			return list
		}
	}
	return strings.Fields(s)
}
