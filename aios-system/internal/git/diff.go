package git

import (
	"fmt"

	"github.com/Hamidullah02/aios/aios-system/internal/sandbox"
	"github.com/Hamidullah02/aios/aios-system/internal/tools"
)

// GitDiffTool returns the diff for a path (REQ-GIT-DIF-001
// to 005). The tool is read-only.
//
// Args:
//
//	path    (string, optional) — restrict the diff to this file.
//	staged  (bool,   optional) — show the staged diff instead.
//	commit  (string, optional) — diff this commit against the
//	          working tree. With "HEAD~1" or a hash.
type GitDiffTool struct {
	Runner runner
}

func (GitDiffTool) Name() string { return "git_diff" }

func (g GitDiffTool) Run(args map[string]any, ctx tools.Context) (tools.Result, error) {
	ws, err := gitWorkspace(ctx.Workspace)
	if err != nil {
		return tools.Result{OK: false, Summary: "no workspace"}, err
	}
	if err := requireGitFunc(); err != nil {
		return tools.Result{OK: false, Summary: "git missing"}, err
	}
	path, _ := args["path"].(string)
	staged, _ := args["staged"].(bool)
	commit, _ := args["commit"].(string)

	argv := []string{"git", "diff", "--no-color"}
	if staged {
		argv = append(argv, "--staged")
	}
	if commit != "" {
		argv = append(argv, commit)
	}
	if path != "" {
		argv = append(argv, "--", path)
	}
	res, err := g.runner().Run(sandbox.RunOptions{
		Command: argv, Cwd: ws, Timeout: 30,
	})
	if err != nil {
		return tools.Result{OK: false, Summary: "git diff failed"},
			&GitCommandError{Command: argv, Stderr: res.Stderr, Exit: res.ExitCode}
	}
	return tools.Result{
		OK:      true,
		Output:  map[string]any{"diff": res.Stdout, "bytes": len(res.Stdout)},
		Summary: fmt.Sprintf("%d bytes of diff", len(res.Stdout)),
	}, nil
}

func (g GitDiffTool) runner() runner {
	if g.Runner != nil {
		return g.Runner
	}
	return newRunner()
}
