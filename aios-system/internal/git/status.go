package git

import (
	"fmt"
	"strings"

	"github.com/sadi/aios-system/internal/sandbox"
	"github.com/sadi/aios-system/internal/tools"
)

// GitStatusTool reports the working tree status
// (REQ-GIT-STA-001 to 003). The tool is read-only.
type GitStatusTool struct {
	// Runner runs the git command. nil means default.
	Runner runner
}

func (GitStatusTool) Name() string { return "git_status" }

func (g GitStatusTool) Run(args map[string]any, ctx tools.Context) (tools.Result, error) {
	ws, err := gitWorkspace(ctx.Workspace)
	if err != nil {
		return tools.Result{OK: false, Summary: "no workspace"}, err
	}
	if err := requireGitFunc(); err != nil {
		return tools.Result{OK: false, Summary: "git missing"}, err
	}
	res, err := g.runner().Run(sandbox.RunOptions{
		Command: []string{"git", "status", "--porcelain", "-b", "-z"},
		Cwd:     ws,
		Timeout: 30,
	})
	if err != nil {
		return tools.Result{OK: false, Summary: "git status failed"},
			&GitCommandError{Command: []string{"git", "status"}, Stderr: res.Stderr, Exit: res.ExitCode}
	}
	status := parseStatus(res.Stdout)
	return tools.Result{
		OK:      true,
		Output:  map[string]any{"status": status},
		Summary: fmt.Sprintf("on %s", status.Branch),
	}, nil
}

func (g GitStatusTool) runner() runner {
	if g.Runner != nil {
		return g.Runner
	}
	return newRunner()
}

// parseStatus parses the output of `git status --porcelain
// -b -z`. The -z flag uses NUL as the line separator so
// filenames with spaces or quotes are preserved.
func parseStatus(out string) GitStatus {
	st := GitStatus{Branch: "HEAD", Staged: []string{}, Modified: []string{}, Untracked: []string{}}
	// -z uses NUL separators; the header (## branch)
	// comes first, then each entry is exactly 3 bytes
	// (XY) + space + path + NUL, or "?? " + path + NUL.
	parts := strings.Split(out, "\x00")
	for _, p := range parts {
		if p == "" {
			continue
		}
		if strings.HasPrefix(p, "## ") {
			// "## branch" or "## HEAD...origin/main [ahead 3]"
			line := strings.TrimPrefix(p, "## ")
			if i := strings.IndexAny(line, "...["); i >= 0 {
				line = line[:i]
			}
			line = strings.TrimSpace(line)
			if line != "" {
				st.Branch = line
			}
			continue
		}
		if len(p) < 4 {
			continue
		}
		xy := p[:2]
		name := p[3:]
		if xy == "??" {
			st.Untracked = append(st.Untracked, name)
			continue
		}
		// Index (X) and worktree (Y).
		if xIsStaged(xy[0]) {
			st.Staged = append(st.Staged, name)
		}
		if yIsModified(xy[1]) {
			st.Modified = append(st.Modified, name)
		}
	}
	return st
}

func xIsStaged(b byte) bool {
	return b != ' ' && b != '?'
}

func yIsModified(b byte) bool {
	return b == 'M' || b == 'D'
}
