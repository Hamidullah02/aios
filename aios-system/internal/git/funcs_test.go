package git

import (
	"errors"
	"testing"

	"github.com/Hamidullah02/aios/aios-system/internal/sandbox"
	"github.com/Hamidullah02/aios/aios-system/internal/tools"
)

// Tests that override the package-level workspaceFunc and
// requireGitFunc to drive the early-return blocks in each
// tool's Run method.

func withWorkspaceFunc(t *testing.T, fn func(string) (string, error)) {
	t.Helper()
	prev := workspaceFunc
	workspaceFunc = fn
	t.Cleanup(func() { workspaceFunc = prev })
}

func withRequireGitFunc(t *testing.T, fn func() error) {
	t.Helper()
	prev := requireGitFunc
	requireGitFunc = fn
	t.Cleanup(func() { requireGitFunc = prev })
}

func TestStatus_WorkspaceError(t *testing.T) {
	withWorkspaceFunc(t, func(string) (string, error) { return "", errors.New("no ws") })
	g := GitStatusTool{Runner: &fakeRunner{commands: map[string]sandbox.SandboxResult{"git": {ExitCode: 0}}}}
	_, err := g.Run(nil, tools.Context{Workspace: ""})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestStatus_RequireGitError(t *testing.T) {
	withRequireGitFunc(t, func() error { return errors.New("missing") })
	g := GitStatusTool{Runner: &fakeRunner{commands: map[string]sandbox.SandboxResult{"git": {ExitCode: 0}}}}
	_, err := g.Run(nil, tools.Context{Workspace: t.TempDir()})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDiff_WorkspaceError(t *testing.T) {
	withWorkspaceFunc(t, func(string) (string, error) { return "", errors.New("no ws") })
	g := GitDiffTool{Runner: &fakeRunner{}}
	_, err := g.Run(nil, tools.Context{Workspace: ""})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDiff_RequireGitError(t *testing.T) {
	withRequireGitFunc(t, func() error { return errors.New("missing") })
	g := GitDiffTool{Runner: &fakeRunner{}}
	_, err := g.Run(nil, tools.Context{Workspace: t.TempDir()})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLog_WorkspaceError(t *testing.T) {
	withWorkspaceFunc(t, func(string) (string, error) { return "", errors.New("no ws") })
	g := GitLogTool{Runner: &fakeRunner{}}
	_, err := g.Run(nil, tools.Context{Workspace: ""})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLog_RequireGitError(t *testing.T) {
	withRequireGitFunc(t, func() error { return errors.New("missing") })
	g := GitLogTool{Runner: &fakeRunner{}}
	_, err := g.Run(nil, tools.Context{Workspace: t.TempDir()})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCommit_WorkspaceError(t *testing.T) {
	withWorkspaceFunc(t, func(string) (string, error) { return "", errors.New("no ws") })
	g := GitCommitTool{Runner: &fakeRunner{}}
	_, err := g.Run(nil, tools.Context{Workspace: ""})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCommit_RequireGitError(t *testing.T) {
	withRequireGitFunc(t, func() error { return errors.New("missing") })
	g := GitCommitTool{Runner: &fakeRunner{}}
	_, err := g.Run(nil, tools.Context{Workspace: t.TempDir()})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestBranch_WorkspaceError(t *testing.T) {
	withWorkspaceFunc(t, func(string) (string, error) { return "", errors.New("no ws") })
	g := GitBranchTool{Runner: &fakeRunner{}}
	_, err := g.Run(nil, tools.Context{Workspace: ""})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestBranch_RequireGitError(t *testing.T) {
	withRequireGitFunc(t, func() error { return errors.New("missing") })
	g := GitBranchTool{Runner: &fakeRunner{}}
	_, err := g.Run(nil, tools.Context{Workspace: t.TempDir()})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestWorktree_WorkspaceError(t *testing.T) {
	withWorkspaceFunc(t, func(string) (string, error) { return "", errors.New("no ws") })
	g := GitWorktreeTool{Runner: &fakeRunner{}}
	_, err := g.Run(nil, tools.Context{Workspace: ""})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestWorktree_RequireGitError(t *testing.T) {
	withRequireGitFunc(t, func() error { return errors.New("missing") })
	g := GitWorktreeTool{Runner: &fakeRunner{}}
	_, err := g.Run(nil, tools.Context{Workspace: t.TempDir()})
	if err == nil {
		t.Fatal("expected error")
	}
}

// exerciseRunnerError covers the `if err != nil` after
// runner.Run in each tool. The runner returns an error.
func TestStatus_RunnerError(t *testing.T) {
	g := GitStatusTool{Runner: &fakeRunner{err: errors.New("boom")}}
	_, err := g.Run(nil, tools.Context{Workspace: t.TempDir()})
	if !errors.Is(err, ErrGitCommand) {
		t.Fatalf("err = %v", err)
	}
}

func TestDiff_RunnerError(t *testing.T) {
	g := GitDiffTool{Runner: &fakeRunner{err: errors.New("boom")}}
	_, err := g.Run(nil, tools.Context{Workspace: t.TempDir()})
	if !errors.Is(err, ErrGitCommand) {
		t.Fatalf("err = %v", err)
	}
}

func TestLog_RunnerError(t *testing.T) {
	g := GitLogTool{Runner: &fakeRunner{err: errors.New("boom")}}
	_, err := g.Run(nil, tools.Context{Workspace: t.TempDir()})
	if !errors.Is(err, ErrGitCommand) {
		t.Fatalf("err = %v", err)
	}
}

func TestWorktree_RunnerError(t *testing.T) {
	g := GitWorktreeTool{Runner: &fakeRunner{err: errors.New("boom")}}
	_, err := g.Run(nil, tools.Context{Workspace: t.TempDir()})
	if !errors.Is(err, ErrGitCommand) {
		t.Fatalf("err = %v", err)
	}
}
