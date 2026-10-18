package git

import (
	"fmt"
	"strings"

	"github.com/sadi/aios-system/internal/sandbox"
	"github.com/sadi/aios-system/internal/tools"
)

// GitCommitTool stages files and creates a commit
// (REQ-GIT-CMT-001 to 005).
//
// Args:
//
//	message (string, required) — the commit message.
//	files   ([]string, optional) — paths to stage. When
//	         empty, the tool commits whatever is already
//	         staged. When the working tree is clean and no
//	         files are staged, the tool raises EmptyCommitError.
//
// Note: the policy gateway is consulted by the bus, not by
// this tool. The bus returns a deny/ask result for tier-3
// tools; the tool itself just runs git.
type GitCommitTool struct {
	Runner runner
}

func (GitCommitTool) Name() string { return "git_commit" }

func (g GitCommitTool) Run(args map[string]any, ctx tools.Context) (tools.Result, error) {
	ws, err := gitWorkspace(ctx.Workspace)
	if err != nil {
		return tools.Result{OK: false, Summary: "no workspace"}, err
	}
	if err := requireGitFunc(); err != nil {
		return tools.Result{OK: false, Summary: "git missing"}, err
	}
	message, _ := args["message"].(string)
	if strings.TrimSpace(message) == "" {
		return tools.Result{OK: false, Summary: "missing message"},
			fmt.Errorf("git: commit: message is required")
	}
	files, _ := args["files"].([]any)
	paths := make([]string, 0, len(files))
	for _, f := range files {
		if s, ok := f.(string); ok {
			paths = append(paths, s)
		}
	}

	runner := g.runner()

	// Stage explicit files when given.
	if len(paths) > 0 {
		addArgv := append([]string{"git", "add", "--"}, paths...)
		res, err := runner.Run(sandbox.RunOptions{
			Command: addArgv, Cwd: ws, Timeout: 30,
		})
		if err != nil {
			return tools.Result{OK: false, Summary: "git add failed"},
				&GitCommandError{Command: addArgv, Stderr: res.Stderr, Exit: res.ExitCode}
		}
	}

	// Make sure there is something to commit. `git diff
	// --cached --quiet` exits 0 when there is nothing to
	// commit, 1 when there is.
	checkRes, _ := runner.Run(sandbox.RunOptions{
		Command: []string{"git", "diff", "--cached", "--quiet"},
		Cwd:     ws, Timeout: 30,
	})
	if checkRes.ExitCode == 0 {
		return tools.Result{OK: false, Summary: "empty commit"},
			&EmptyCommitError{Reason: "no staged changes"}
	}

	// Commit. The bus has already passed policy.
	commitArgv := []string{"git", "commit", "-m", message}
	res, err := runner.Run(sandbox.RunOptions{
		Command: commitArgv, Cwd: ws, Timeout: 60,
	})
	if err != nil {
		return tools.Result{OK: false, Summary: "git commit failed"},
			&GitCommandError{Command: commitArgv, Stderr: res.Stderr, Exit: res.ExitCode}
	}

	// Read the new commit's hash and metadata.
	logRes, logErr := runner.Run(sandbox.RunOptions{
		Command: []string{"git", "log", "-n", "1", "--format=%H|%an|%aI|%s"},
		Cwd:     ws, Timeout: 30,
	})
	var commit GitCommit
	if logErr == nil && logRes.ExitCode == 0 {
		rows := parseLog(logRes.Stdout)
		if len(rows) > 0 {
			commit = rows[0]
		}
	}
	return tools.Result{
		OK:      true,
		Output:  map[string]any{"commit": commit, "raw": strings.TrimSpace(res.Stdout)},
		Summary: fmt.Sprintf("commit %s", commit.Hash),
	}, nil
}

func (g GitCommitTool) runner() runner {
	if g.Runner != nil {
		return g.Runner
	}
	return newRunner()
}
