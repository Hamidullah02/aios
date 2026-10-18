package git

import (
	"errors"
	"strings"
	"testing"

	"github.com/sadi/aios-system/internal/sandbox"
	"github.com/sadi/aios-system/internal/tools"
)

func TestLog_Run(t *testing.T) {
	out := "abc|Alice|2024-01-01T00:00:00Z|first\x1edef|Bob|2024-01-02T00:00:00Z|second\x1e"
	g := GitLogTool{
		Runner: &fakeRunner{commands: map[string]sandbox.SandboxResult{"git": {ExitCode: 0, Stdout: out}}},
	}
	res, err := g.Run(nil, tools.Context{Workspace: t.TempDir()})
	assertOK(t, res, err)
	commits, _ := res.Output["commits"].([]GitCommit)
	if len(commits) != 2 {
		t.Fatalf("commits = %d", len(commits))
	}
	if commits[0].Hash != "abc" {
		t.Fatalf("hash = %q", commits[0].Hash)
	}
}

func TestLog_DefaultMax(t *testing.T) {
	rec := &recordingRunner{res: sandbox.SandboxResult{ExitCode: 0}}
	g := GitLogTool{Runner: rec}
	_, _ = g.Run(nil, tools.Context{Workspace: t.TempDir()})
	opts, _ := rec.firstCall()
	argv := opts.Command
	found := false
	for i, a := range argv {
		if a == "-n" && i+1 < len(argv) && argv[i+1] == "10" {
			found = true
		}
	}
	if !found {
		t.Fatalf("argv missing -n 10: %v", argv)
	}
}

func TestLog_CustomMax(t *testing.T) {
	rec := &recordingRunner{res: sandbox.SandboxResult{ExitCode: 0}}
	g := GitLogTool{Runner: rec}
	_, _ = g.Run(map[string]any{"max": 3}, tools.Context{Workspace: t.TempDir()})
	opts, _ := rec.firstCall()
	argv := opts.Command
	found := false
	for i, a := range argv {
		if a == "-n" && i+1 < len(argv) && argv[i+1] == "3" {
			found = true
		}
	}
	if !found {
		t.Fatalf("argv missing -n 3: %v", argv)
	}
}

func TestLog_PathArg(t *testing.T) {
	rec := &recordingRunner{res: sandbox.SandboxResult{ExitCode: 0}}
	g := GitLogTool{Runner: rec}
	_, _ = g.Run(map[string]any{"path": "x.go"}, tools.Context{Workspace: t.TempDir()})
	opts, _ := rec.firstCall()
	argv := opts.Command
	if !strings.Contains(strings.Join(argv, " "), "-- x.go") {
		t.Fatalf("argv missing -- x.go: %v", argv)
	}
}

func TestLog_CommandFailure(t *testing.T) {
	g := GitLogTool{
		Runner: &fakeRunner{err: errors.New("boom")},
	}
	_, err := g.Run(nil, tools.Context{Workspace: t.TempDir()})
	if !errors.Is(err, ErrGitCommand) {
		t.Fatalf("err = %v", err)
	}
}

func TestLog_Name(t *testing.T) {
	if (GitLogTool{}).Name() != "git_log" {
		t.Fatal("name")
	}
}
