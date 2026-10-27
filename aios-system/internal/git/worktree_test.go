package git

import (
	"errors"
	"testing"

	"github.com/Hamidullah02/aios/aios-system/internal/sandbox"
	"github.com/Hamidullah02/aios/aios-system/internal/tools"
)

func TestWorktree_List(t *testing.T) {
	out := "worktree /tmp/main\nHEAD 111\nbranch refs/heads/main\n\n"
	g := GitWorktreeTool{
		Runner: &fakeRunner{
			commands: map[string]sandbox.SandboxResult{
				"git": {ExitCode: 0, Stdout: out},
			},
		},
	}
	res, err := g.Run(map[string]any{"action": "list"}, tools.Context{Workspace: t.TempDir()})
	assertOK(t, res, err)
	wts, _ := res.Output["worktrees"].([]worktreeRow)
	if len(wts) != 1 || wts[0].Path != "/tmp/main" {
		t.Fatalf("wts = %v", wts)
	}
}

func TestWorktree_Add(t *testing.T) {
	rec := &recordingRunner{res: sandbox.SandboxResult{ExitCode: 0}}
	g := GitWorktreeTool{Runner: rec}
	ws := t.TempDir()
	_, err := g.Run(map[string]any{
		"action": "add", "path": "wt", "branch": "feat",
	}, tools.Context{Workspace: ws})
	if err != nil {
		t.Fatal(err)
	}
	opts, _ := rec.firstCall()
	argv := opts.Command
	if len(argv) < 5 || argv[1] != "worktree" || argv[2] != "add" {
		t.Fatalf("argv = %v", argv)
	}
}

func TestWorktree_Add_MissingArgs(t *testing.T) {
	g := GitWorktreeTool{Runner: &fakeRunner{}}
	_, err := g.Run(map[string]any{"action": "add"}, tools.Context{Workspace: t.TempDir()})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestWorktree_Remove(t *testing.T) {
	rec := &recordingRunner{res: sandbox.SandboxResult{ExitCode: 0}}
	g := GitWorktreeTool{Runner: rec}
	ws := t.TempDir()
	_, err := g.Run(map[string]any{
		"action": "remove", "path": "wt",
	}, tools.Context{Workspace: ws})
	if err != nil {
		t.Fatal(err)
	}
	opts, _ := rec.firstCall()
	argv := opts.Command
	if len(argv) < 4 || argv[1] != "worktree" || argv[2] != "remove" {
		t.Fatalf("argv = %v", argv)
	}
}

func TestWorktree_Remove_MissingPath(t *testing.T) {
	g := GitWorktreeTool{Runner: &fakeRunner{}}
	_, err := g.Run(map[string]any{"action": "remove"}, tools.Context{Workspace: t.TempDir()})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestWorktree_UnknownAction(t *testing.T) {
	g := GitWorktreeTool{Runner: &fakeRunner{}}
	_, err := g.Run(map[string]any{"action": "merge"}, tools.Context{Workspace: t.TempDir()})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestWorktree_CommandFailure(t *testing.T) {
	g := GitWorktreeTool{
		Runner: &fakeRunner{err: errors.New("boom")},
	}
	_, err := g.Run(map[string]any{"action": "list"}, tools.Context{Workspace: t.TempDir()})
	if !errors.Is(err, ErrGitCommand) {
		t.Fatalf("err = %v", err)
	}
}

func TestWorktree_OutsideWorkspace(t *testing.T) {
	g := GitWorktreeTool{
		Runner: &recordingRunner{res: sandbox.SandboxResult{ExitCode: 0}},
	}
	ws := t.TempDir()
	outside := t.TempDir()
	_, err := g.Run(map[string]any{
		"action": "add", "path": outside, "branch": "feat",
	}, tools.Context{Workspace: ws})
	if err == nil {
		t.Fatal("expected path-rejected error")
	}
}

func TestWorktree_Name(t *testing.T) {
	if (GitWorktreeTool{}).Name() != "git_worktree" {
		t.Fatal("name")
	}
}
