package git

import (
	"errors"
	"testing"

	"github.com/sadi/aios-system/internal/sandbox"
	"github.com/sadi/aios-system/internal/tools"
)

func TestDiff_Run(t *testing.T) {
	g := GitDiffTool{
		Runner: &recordingRunner{
			res: sandbox.SandboxResult{ExitCode: 0, Stdout: "diff --git a/x b/x\n+x\n"},
		},
	}
	res, err := g.Run(map[string]any{"path": "x.txt"}, tools.Context{Workspace: t.TempDir()})
	assertOK(t, res, err)
	if got := res.Output["bytes"].(int); got == 0 {
		t.Fatal("expected non-zero bytes")
	}
}

func TestDiff_StagedFlag(t *testing.T) {
	rec := &recordingRunner{res: sandbox.SandboxResult{ExitCode: 0}}
	g := GitDiffTool{Runner: rec}
	_, _ = g.Run(map[string]any{"staged": true}, tools.Context{Workspace: t.TempDir()})
	opts, _ := rec.firstCall()
	argv := opts.Command
	found := false
	for _, a := range argv {
		if a == "--staged" {
			found = true
		}
	}
	if !found {
		t.Fatalf("argv missing --staged: %v", argv)
	}
}

func TestDiff_CommitArg(t *testing.T) {
	rec := &recordingRunner{res: sandbox.SandboxResult{ExitCode: 0}}
	g := GitDiffTool{Runner: rec}
	_, _ = g.Run(map[string]any{"commit": "HEAD~1"}, tools.Context{Workspace: t.TempDir()})
	opts, _ := rec.firstCall()
	argv := opts.Command
	found := false
	for _, a := range argv {
		if a == "HEAD~1" {
			found = true
		}
	}
	if !found {
		t.Fatalf("argv missing HEAD~1: %v", argv)
	}
}

func TestDiff_PathArg(t *testing.T) {
	rec := &recordingRunner{res: sandbox.SandboxResult{ExitCode: 0}}
	g := GitDiffTool{Runner: rec}
	_, _ = g.Run(map[string]any{"path": "foo.go"}, tools.Context{Workspace: t.TempDir()})
	opts, _ := rec.firstCall()
	argv := opts.Command
	found := false
	for i, a := range argv {
		if a == "--" && i+1 < len(argv) && argv[i+1] == "foo.go" {
			found = true
		}
	}
	if !found {
		t.Fatalf("argv missing -- foo.go: %v", argv)
	}
}

func TestDiff_GitMissing(t *testing.T) {
	withRequireGitFunc(t, func() error {
		return &GitCommandError{Command: []string{"git"}, Stderr: "missing"}
	})
	g := GitDiffTool{}
	_, err := g.Run(nil, tools.Context{Workspace: t.TempDir()})
	if !errors.Is(err, ErrGitCommand) {
		t.Fatalf("err = %v", err)
	}
}

func TestDiff_CommandFailure(t *testing.T) {
	g := GitDiffTool{
		Runner: &fakeRunner{err: errors.New("boom")},
	}
	_, err := g.Run(nil, tools.Context{Workspace: t.TempDir()})
	if !errors.Is(err, ErrGitCommand) {
		t.Fatalf("err = %v", err)
	}
}

func TestDiff_Name(t *testing.T) {
	if (GitDiffTool{}).Name() != "git_diff" {
		t.Fatal("name")
	}
}
