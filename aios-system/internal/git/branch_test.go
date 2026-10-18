package git

import (
	"errors"
	"testing"

	"github.com/sadi/aios-system/internal/sandbox"
	"github.com/sadi/aios-system/internal/tools"
)

func TestBranch_List(t *testing.T) {
	g := GitBranchTool{
		Runner: &fakeRunner{
			commands: map[string]sandbox.SandboxResult{
				"git": {ExitCode: 0, Stdout: "main\nfeature\n"},
			},
		},
	}
	res, err := g.Run(map[string]any{"action": "list"}, tools.Context{Workspace: t.TempDir()})
	assertOK(t, res, err)
	branches, _ := res.Output["branches"].([]string)
	if len(branches) != 2 || branches[0] != "main" || branches[1] != "feature" {
		t.Fatalf("branches = %v", branches)
	}
}

func TestBranch_ListDefault(t *testing.T) {
	// Empty action defaults to "list".
	rec := &recordingRunner{res: sandbox.SandboxResult{ExitCode: 0, Stdout: "main\n"}}
	g := GitBranchTool{Runner: rec}
	_, _ = g.Run(map[string]any{}, tools.Context{Workspace: t.TempDir()})
	opts, _ := rec.firstCall()
	argv := opts.Command
	if len(argv) < 2 || argv[1] != "branch" {
		t.Fatalf("argv = %v", argv)
	}
}

func TestBranch_Create(t *testing.T) {
	rec := &recordingRunner{res: sandbox.SandboxResult{ExitCode: 0, Stdout: "main\n"}}
	g := GitBranchTool{Runner: rec}
	res, err := g.Run(map[string]any{"action": "create", "name": "feat"}, tools.Context{Workspace: t.TempDir()})
	assertOK(t, res, err)
	opts, _ := rec.firstCall()
	argv := opts.Command
	if len(argv) < 3 || argv[1] != "branch" || argv[2] != "feat" {
		t.Fatalf("argv = %v", argv)
	}
}

func TestBranch_Create_MissingName(t *testing.T) {
	g := GitBranchTool{Runner: &fakeRunner{}}
	_, err := g.Run(map[string]any{"action": "create"}, tools.Context{Workspace: t.TempDir()})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestBranch_Checkout(t *testing.T) {
	rec := &recordingRunner{res: sandbox.SandboxResult{ExitCode: 0, Stdout: "feat\n"}}
	g := GitBranchTool{Runner: rec}
	res, err := g.Run(map[string]any{"action": "checkout", "name": "feat"},
		tools.Context{Workspace: t.TempDir()})
	assertOK(t, res, err)
	opts, _ := rec.firstCall()
	argv := opts.Command
	if len(argv) < 3 || argv[1] != "checkout" || argv[2] != "feat" {
		t.Fatalf("argv = %v", argv)
	}
}

func TestBranch_Delete(t *testing.T) {
	rec := &recordingRunner{res: sandbox.SandboxResult{ExitCode: 0, Stdout: "main\n"}}
	g := GitBranchTool{Runner: rec}
	res, err := g.Run(map[string]any{"action": "delete", "name": "feat"},
		tools.Context{Workspace: t.TempDir()})
	assertOK(t, res, err)
	opts, _ := rec.firstCall()
	argv := opts.Command
	if len(argv) < 4 || argv[1] != "branch" || argv[2] != "-d" {
		t.Fatalf("argv = %v", argv)
	}
}

func TestBranch_UnknownAction(t *testing.T) {
	g := GitBranchTool{Runner: &fakeRunner{}}
	_, err := g.Run(map[string]any{"action": "merge"}, tools.Context{Workspace: t.TempDir()})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestBranch_CommandFailure(t *testing.T) {
	g := GitBranchTool{
		Runner: &fakeRunner{err: errors.New("boom")},
	}
	_, err := g.Run(map[string]any{"action": "list"}, tools.Context{Workspace: t.TempDir()})
	if !errors.Is(err, ErrGitCommand) {
		t.Fatalf("err = %v", err)
	}
}

func TestBranch_Name(t *testing.T) {
	if (GitBranchTool{}).Name() != "git_branch" {
		t.Fatal("name")
	}
}

func TestBranch_Integration(t *testing.T) {
	if _, err := execLookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	dir := initRepo(t)
	g := GitBranchTool{}
	res, err := g.Run(map[string]any{"action": "create", "name": "feat1"},
		tools.Context{Workspace: dir})
	assertOK(t, res, err)
	// The new branch should appear in list.
	res, err = g.Run(map[string]any{"action": "list"}, tools.Context{Workspace: dir})
	assertOK(t, res, err)
	branches, _ := res.Output["branches"].([]string)
	found := false
	for _, b := range branches {
		if b == "feat1" {
			found = true
		}
	}
	if !found {
		t.Fatalf("feat1 not in %v", branches)
	}
}
