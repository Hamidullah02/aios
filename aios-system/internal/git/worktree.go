package git

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Hamidullah02/aios/aios-system/internal/sandbox"
	"github.com/Hamidullah02/aios/aios-system/internal/tools"
)

// GitWorktreeTool manages worktrees (REQ-GIT-WRT-001 to 004).
//
// Args:
//
//	action (string, required) — "list", "add", or "remove".
//	path   (string, required for add/remove) — the worktree path.
//	branch (string, required for add) — the branch to check out.
type GitWorktreeTool struct {
	Runner runner
}

func (GitWorktreeTool) Name() string { return "git_worktree" }

func (g GitWorktreeTool) Run(args map[string]any, ctx tools.Context) (tools.Result, error) {
	ws, err := gitWorkspace(ctx.Workspace)
	if err != nil {
		return tools.Result{OK: false, Summary: "no workspace"}, err
	}
	if err := requireGitFunc(); err != nil {
		return tools.Result{OK: false, Summary: "git missing"}, err
	}
	action, _ := args["action"].(string)
	path, _ := args["path"].(string)
	branch, _ := args["branch"].(string)
	runner := g.runner()

	var argv []string
	switch action {
	case "list", "":
		argv = []string{"git", "worktree", "list", "--porcelain"}
	case "add":
		if path == "" || branch == "" {
			return tools.Result{OK: false, Summary: "missing path/branch"},
				fmt.Errorf("git: worktree add: path and branch are required")
		}
		// Resolve the path against the workspace root and
		// reject paths that escape it.
		abs, err := tools.ResolvePath(ws, path)
		if err != nil {
			return tools.Result{OK: false, Summary: "bad path"}, err
		}
		path = abs
		argv = []string{"git", "worktree", "add", path, branch}
	case "remove":
		if path == "" {
			return tools.Result{OK: false, Summary: "missing path"},
				fmt.Errorf("git: worktree remove: path is required")
		}
		abs, err := tools.ResolvePath(ws, path)
		if err != nil {
			return tools.Result{OK: false, Summary: "bad path"}, err
		}
		path = abs
		argv = []string{"git", "worktree", "remove", path}
	default:
		return tools.Result{OK: false, Summary: "bad action"},
			fmt.Errorf("git: worktree: unknown action %q", action)
	}
	res, err := runner.Run(sandbox.RunOptions{
		Command: argv, Cwd: ws, Timeout: 60,
	})
	if err != nil {
		return tools.Result{OK: false, Summary: "git worktree " + action + " failed"},
			&GitCommandError{Command: argv, Stderr: res.Stderr, Exit: res.ExitCode}
	}
	// For "add", return the resolved path. For "list", parse
	// the porcelain output (each worktree separated by a
	// blank line; first line is "worktree <path>").
	if action == "add" {
		return tools.Result{
			OK:      true,
			Output:  map[string]any{"action": "add", "path": path, "branch": branch},
			Summary: "worktree added at " + path,
		}, nil
	}
	if action == "remove" {
		return tools.Result{
			OK:      true,
			Output:  map[string]any{"action": "remove", "path": path},
			Summary: "worktree removed at " + filepath.Base(path),
		}, nil
	}
	worktrees := parseWorktreeList(res.Stdout)
	return tools.Result{
		OK:      true,
		Output:  map[string]any{"worktrees": worktrees, "count": len(worktrees)},
		Summary: fmt.Sprintf("%d worktree(s)", len(worktrees)),
	}, nil
}

func (g GitWorktreeTool) runner() runner {
	if g.Runner != nil {
		return g.Runner
	}
	return newRunner()
}

// worktreeRow is one parsed worktree entry.
type worktreeRow struct {
	Path   string `json:"path"`
	Commit string `json:"commit"`
	Branch string `json:"branch"`
}

// parseWorktreeList parses the porcelain output. Each
// record is a block of key/value lines followed by a
// blank line.
func parseWorktreeList(out string) []worktreeRow {
	var rows []worktreeRow
	var cur worktreeRow
	flush := func() {
		if cur.Path != "" {
			rows = append(rows, cur)
		}
		cur = worktreeRow{}
	}
	for _, line := range strings.Split(out, "\n") {
		switch {
		case strings.HasPrefix(line, "worktree "):
			flush()
			cur.Path = strings.TrimPrefix(line, "worktree ")
		case strings.HasPrefix(line, "HEAD "):
			cur.Commit = strings.TrimPrefix(line, "HEAD ")
		case strings.HasPrefix(line, "branch "):
			cur.Branch = strings.TrimPrefix(line, "branch ")
		case line == "":
			flush()
		}
	}
	flush()
	return rows
}
