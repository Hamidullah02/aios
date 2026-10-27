package git

import (
	"testing"

	"github.com/Hamidullah02/aios/aios-system/internal/sandbox"
	"github.com/Hamidullah02/aios/aios-system/internal/tools"
)

// Empty-Workspace tests. The tools call gitWorkspace which
// falls back to tools.Workspace(); that returns the cwd.
// These tests verify the fallback path succeeds with a
// fake runner.

func TestStatus_EmptyWorkspaceFallback(t *testing.T) {
	g := GitStatusTool{
		Runner: &fakeRunner{
			commands: map[string]sandbox.SandboxResult{
				"git": {ExitCode: 0, Stdout: "## main\x00"},
			},
		},
	}
	res, err := g.Run(nil, tools.Context{Workspace: ""})
	assertOK(t, res, err)
}

func TestDiff_EmptyWorkspaceFallback(t *testing.T) {
	g := GitDiffTool{
		Runner: &fakeRunner{commands: map[string]sandbox.SandboxResult{"git": {ExitCode: 0, Stdout: "diff"}}},
	}
	res, err := g.Run(nil, tools.Context{Workspace: ""})
	assertOK(t, res, err)
}

func TestLog_EmptyWorkspaceFallback(t *testing.T) {
	g := GitLogTool{
		Runner: &fakeRunner{commands: map[string]sandbox.SandboxResult{"git": {ExitCode: 0, Stdout: ""}}},
	}
	res, err := g.Run(nil, tools.Context{Workspace: ""})
	assertOK(t, res, err)
}

func TestCommit_EmptyWorkspaceFallback(t *testing.T) {
	g := GitCommitTool{
		Runner: &scriptedRunner{
			fn: func(argv []string) sandbox.SandboxResult {
				if len(argv) >= 3 && argv[1] == "diff" {
					return sandbox.SandboxResult{ExitCode: 1}
				}
				return sandbox.SandboxResult{ExitCode: 0}
			},
		},
	}
	_, _ = g.Run(map[string]any{"message": "msg"}, tools.Context{Workspace: ""})
}

func TestBranch_EmptyWorkspaceFallback(t *testing.T) {
	g := GitBranchTool{
		Runner: &fakeRunner{commands: map[string]sandbox.SandboxResult{"git": {ExitCode: 0, Stdout: "main"}}},
	}
	res, err := g.Run(map[string]any{"action": "list"}, tools.Context{Workspace: ""})
	assertOK(t, res, err)
}

func TestWorktree_EmptyWorkspaceFallback(t *testing.T) {
	g := GitWorktreeTool{
		Runner: &fakeRunner{commands: map[string]sandbox.SandboxResult{"git": {ExitCode: 0, Stdout: ""}}},
	}
	res, err := g.Run(map[string]any{"action": "list"}, tools.Context{Workspace: ""})
	assertOK(t, res, err)
}

// Default-runner integration tests. The tool's runner()
// method returns the package-level newRunner() when
// g.Runner is nil. We exercise this with real git on PATH.
func TestStatus_NilRunnerAndLookPath_Integration(t *testing.T) {
	if _, err := execLookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	dir := initRepo(t)
	g := GitStatusTool{}
	res, err := g.Run(nil, tools.Context{Workspace: dir})
	assertOK(t, res, err)
}

func TestDiff_NilRunnerAndLookPath_Integration(t *testing.T) {
	if _, err := execLookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	dir := initRepo(t)
	g := GitDiffTool{}
	res, err := g.Run(nil, tools.Context{Workspace: dir})
	assertOK(t, res, err)
}

func TestLog_NilRunnerAndLookPath_Integration(t *testing.T) {
	if _, err := execLookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	dir := initRepo(t)
	g := GitLogTool{}
	res, err := g.Run(nil, tools.Context{Workspace: dir})
	assertOK(t, res, err)
}

func TestCommit_NilRunnerAndLookPath_Integration(t *testing.T) {
	if _, err := execLookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	dir := initRepo(t)
	g := GitCommitTool{}
	_, _ = g.Run(map[string]any{"message": "x"}, tools.Context{Workspace: dir})
}

func TestBranch_NilRunnerAndLookPath_Integration(t *testing.T) {
	if _, err := execLookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	dir := initRepo(t)
	g := GitBranchTool{}
	res, err := g.Run(map[string]any{"action": "list"}, tools.Context{Workspace: dir})
	assertOK(t, res, err)
}

func TestWorktree_NilRunnerAndLookPath_Integration(t *testing.T) {
	if _, err := execLookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	dir := initRepo(t)
	g := GitWorktreeTool{}
	res, err := g.Run(map[string]any{"action": "list"}, tools.Context{Workspace: dir})
	assertOK(t, res, err)
}

// parseStatus short-line guard.
func TestParseStatus_ShortLine(t *testing.T) {
	st := parseStatus("## main\x00ab\x00")
	if len(st.Staged) != 0 || len(st.Modified) != 0 || len(st.Untracked) != 0 {
		t.Fatalf("got %+v", st)
	}
}

// parseLog short-record guard.
func TestParseLog_ShortRecord(t *testing.T) {
	out := "abc|Alice|2024\x1e"
	if got := parseLog(out); len(got) != 0 {
		t.Fatalf("got %v", got)
	}
}
