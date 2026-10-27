package git

import (
	"fmt"
	"strings"

	"github.com/Hamidullah02/aios/aios-system/internal/sandbox"
	"github.com/Hamidullah02/aios/aios-system/internal/tools"
)

// GitLogTool returns the recent commits (REQ-GIT-LOG-001
// to 005). The tool is read-only.
//
// Args:
//
//	max  (int,    optional, default 10) — number of commits.
//	path (string, optional) — restrict to commits that
//	      touched this file.
type GitLogTool struct {
	Runner runner
}

func (GitLogTool) Name() string { return "git_log" }

func (g GitLogTool) Run(args map[string]any, ctx tools.Context) (tools.Result, error) {
	ws, err := gitWorkspace(ctx.Workspace)
	if err != nil {
		return tools.Result{OK: false, Summary: "no workspace"}, err
	}
	if err := requireGitFunc(); err != nil {
		return tools.Result{OK: false, Summary: "git missing"}, err
	}
	max := 10
	if v, ok := args["max"].(int); ok && v > 0 {
		max = v
	}
	path, _ := args["path"].(string)

	// --format produces a delimiter-separated row. We use
	// "|<key>=<value>" pairs so the parser is a tiny
	// split on '|'. The trailing %x1e is the record
	// separator (we strip it).
	format := "%H|%an|%aI|%s%x1e"
	argv := []string{"git", "log", "-n", fmt.Sprintf("%d", max), "--format=" + format}
	if path != "" {
		argv = append(argv, "--", path)
	}
	res, err := g.runner().Run(sandbox.RunOptions{
		Command: argv, Cwd: ws, Timeout: 30,
	})
	if err != nil {
		return tools.Result{OK: false, Summary: "git log failed"},
			&GitCommandError{Command: argv, Stderr: res.Stderr, Exit: res.ExitCode}
	}
	commits := parseLog(res.Stdout)
	return tools.Result{
		OK:      true,
		Output:  map[string]any{"commits": commits, "count": len(commits)},
		Summary: fmt.Sprintf("%d commit(s)", len(commits)),
	}, nil
}

func (g GitLogTool) runner() runner {
	if g.Runner != nil {
		return g.Runner
	}
	return newRunner()
}

// parseLog splits the --format output on the record
// separator and maps each row to a GitCommit.
func parseLog(out string) []GitCommit {
	records := strings.Split(out, "\x1e")
	out2 := make([]GitCommit, 0, len(records))
	for _, rec := range records {
		rec = strings.TrimRight(rec, "\n")
		if rec == "" {
			continue
		}
		parts := strings.SplitN(rec, "|", 4)
		if len(parts) < 4 {
			continue
		}
		out2 = append(out2, GitCommit{
			Hash:    parts[0],
			Author:  parts[1],
			Date:    parts[2],
			Subject: parts[3],
		})
	}
	return out2
}
