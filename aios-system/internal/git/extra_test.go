package git

import (
	"errors"
	"testing"

	"github.com/sadi/aios-system/internal/sandbox"
	"github.com/sadi/aios-system/internal/tools"
)

func TestBranch_Checkout_MissingName(t *testing.T) {
	g := GitBranchTool{Runner: &fakeRunner{}}
	_, err := g.Run(map[string]any{"action": "checkout"}, tools.Context{Workspace: t.TempDir()})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestBranch_Delete_MissingName(t *testing.T) {
	g := GitBranchTool{Runner: &fakeRunner{}}
	_, err := g.Run(map[string]any{"action": "delete"}, tools.Context{Workspace: t.TempDir()})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestBranch_Create_RunnerError(t *testing.T) {
	g := GitBranchTool{Runner: &fakeRunner{err: errors.New("boom")}}
	_, err := g.Run(map[string]any{"action": "create", "name": "x"},
		tools.Context{Workspace: t.TempDir()})
	if !errors.Is(err, ErrGitCommand) {
		t.Fatalf("err = %v", err)
	}
}

func TestBranch_Checkout_RunnerError(t *testing.T) {
	g := GitBranchTool{Runner: &fakeRunner{err: errors.New("boom")}}
	_, err := g.Run(map[string]any{"action": "checkout", "name": "x"},
		tools.Context{Workspace: t.TempDir()})
	if !errors.Is(err, ErrGitCommand) {
		t.Fatalf("err = %v", err)
	}
}

func TestBranch_Delete_RunnerError(t *testing.T) {
	g := GitBranchTool{Runner: &fakeRunner{err: errors.New("boom")}}
	_, err := g.Run(map[string]any{"action": "delete", "name": "x"},
		tools.Context{Workspace: t.TempDir()})
	if !errors.Is(err, ErrGitCommand) {
		t.Fatalf("err = %v", err)
	}
}

func TestWorktree_Add_RunnerError(t *testing.T) {
	g := GitWorktreeTool{Runner: &fakeRunner{err: errors.New("boom")}}
	_, err := g.Run(map[string]any{"action": "add", "path": "wt", "branch": "feat"},
		tools.Context{Workspace: t.TempDir()})
	if !errors.Is(err, ErrGitCommand) {
		t.Fatalf("err = %v", err)
	}
}

func TestWorktree_Remove_RunnerError(t *testing.T) {
	g := GitWorktreeTool{Runner: &fakeRunner{err: errors.New("boom")}}
	_, err := g.Run(map[string]any{"action": "remove", "path": "wt"},
		tools.Context{Workspace: t.TempDir()})
	if !errors.Is(err, ErrGitCommand) {
		t.Fatalf("err = %v", err)
	}
}

func TestWorktree_Remove_OutsideWorkspace(t *testing.T) {
	g := GitWorktreeTool{Runner: &fakeRunner{}}
	ws := t.TempDir()
	outside := t.TempDir()
	_, err := g.Run(map[string]any{
		"action": "remove", "path": outside,
	}, tools.Context{Workspace: ws})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCommit_LogReadFails(t *testing.T) {
	runner := &scriptedRunner{
		fn: func(argv []string) sandbox.SandboxResult {
			if len(argv) >= 3 && argv[1] == "diff" {
				return sandbox.SandboxResult{ExitCode: 1}
			}
			if len(argv) >= 2 && argv[1] == "commit" {
				return sandbox.SandboxResult{ExitCode: 0, Stdout: "ok\n"}
			}
			return sandbox.SandboxResult{ExitCode: 1, Stderr: "no log"}
		},
	}
	g := GitCommitTool{Runner: runner}
	res, err := g.Run(map[string]any{"message": "msg"}, tools.Context{Workspace: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK {
		t.Fatalf("not OK: %s", res.Summary)
	}
}

func TestDefaultRequireGit_Failure(t *testing.T) {
	prev := lookPath
	lookPath = func(string) (string, error) { return "", errors.New("missing") }
	t.Cleanup(func() { lookPath = prev })
	err := requireGitFunc()
	if !errors.Is(err, ErrGitCommand) {
		t.Fatalf("err = %v", err)
	}
}

func TestDefaultRequireGit_OK(t *testing.T) {
	prev := lookPath
	lookPath = func(string) (string, error) { return "/usr/bin/git", nil }
	t.Cleanup(func() { lookPath = prev })
	if err := requireGitFunc(); err != nil {
		t.Fatalf("err = %v", err)
	}
}
