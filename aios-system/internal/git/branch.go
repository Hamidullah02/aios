package git

import (
	"fmt"
	"strings"

	"github.com/Hamidullah02/aios/aios-system/internal/sandbox"
	"github.com/Hamidullah02/aios/aios-system/internal/tools"
)

// GitBranchTool manages branches (REQ-GIT-BRH-001 to 004).
//
// Args:
//
//	action (string, required) — "list", "create", "checkout", or "delete".
//	name   (string, required for create/checkout/delete) — the branch name.
type GitBranchTool struct {
	Runner runner
}

func (GitBranchTool) Name() string { return "git_branch" }

func (g GitBranchTool) Run(args map[string]any, ctx tools.Context) (tools.Result, error) {
	ws, err := gitWorkspace(ctx.Workspace)
	if err != nil {
		return tools.Result{OK: false, Summary: "no workspace"}, err
	}
	if err := requireGitFunc(); err != nil {
		return tools.Result{OK: false, Summary: "git missing"}, err
	}
	action, _ := args["action"].(string)
	name, _ := args["name"].(string)
	runner := g.runner()

	var argv []string
	switch action {
	case "list", "":
		argv = []string{"git", "branch", "--no-color", "--format=%(refname:short)"}
		res, err := runner.Run(sandbox.RunOptions{
			Command: argv, Cwd: ws, Timeout: 30,
		})
		if err != nil {
			return tools.Result{OK: false, Summary: "git branch failed"},
				&GitCommandError{Command: argv, Stderr: res.Stderr, Exit: res.ExitCode}
		}
		branches := splitNonEmpty(res.Stdout, "\n")
		return tools.Result{
			OK:      true,
			Output:  map[string]any{"branches": branches, "count": len(branches)},
			Summary: fmt.Sprintf("%d branches", len(branches)),
		}, nil
	case "create":
		if name == "" {
			return tools.Result{OK: false, Summary: "missing name"},
				fmt.Errorf("git: branch create: name is required")
		}
		argv = []string{"git", "branch", name}
	case "checkout":
		if name == "" {
			return tools.Result{OK: false, Summary: "missing name"},
				fmt.Errorf("git: branch checkout: name is required")
		}
		argv = []string{"git", "checkout", name}
	case "delete":
		if name == "" {
			return tools.Result{OK: false, Summary: "missing name"},
				fmt.Errorf("git: branch delete: name is required")
		}
		argv = []string{"git", "branch", "-d", name}
	default:
		return tools.Result{OK: false, Summary: "bad action"},
			fmt.Errorf("git: branch: unknown action %q", action)
	}
	res, err := runner.Run(sandbox.RunOptions{
		Command: argv, Cwd: ws, Timeout: 30,
	})
	if err != nil {
		return tools.Result{OK: false, Summary: "git branch " + action + " failed"},
			&GitCommandError{Command: argv, Stderr: res.Stderr, Exit: res.ExitCode}
	}
	// Read the current branch for the result.
	headRes, _ := runner.Run(sandbox.RunOptions{
		Command: []string{"git", "rev-parse", "--abbrev-ref", "HEAD"},
		Cwd:     ws, Timeout: 30,
	})
	current := strings.TrimSpace(headRes.Stdout)
	return tools.Result{
		OK:      true,
		Output:  map[string]any{"action": action, "name": name, "current": current},
		Summary: fmt.Sprintf("branch %s %s", action, name),
	}, nil
}

func (g GitBranchTool) runner() runner {
	if g.Runner != nil {
		return g.Runner
	}
	return newRunner()
}

// splitNonEmpty splits s on sep and drops empty rows.
func splitNonEmpty(s, sep string) []string {
	parts := strings.Split(s, sep)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if strings.TrimSpace(p) == "" {
			continue
		}
		out = append(out, p)
	}
	return out
}
