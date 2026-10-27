package git

import (
	"errors"
	"strings"
	"testing"

	"github.com/Hamidullah02/aios/aios-system/internal/sandbox"
	"github.com/Hamidullah02/aios/aios-system/internal/tools"
)

func TestCommit_MissingMessage(t *testing.T) {
	g := GitCommitTool{
		Runner: &fakeRunner{commands: map[string]sandbox.SandboxResult{"git": {ExitCode: 0}}},
	}
	_, err := g.Run(map[string]any{}, tools.Context{Workspace: t.TempDir()})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCommit_StagesFiles(t *testing.T) {
	rec := &recordingRunner{res: sandbox.SandboxResult{ExitCode: 0}}
	g := GitCommitTool{Runner: rec}
	_, _ = g.Run(map[string]any{
		"message": "msg",
		"files":   []any{"a.txt", "b.txt"},
	}, tools.Context{Workspace: t.TempDir()})
	opts, _ := rec.firstCall()
	argv := opts.Command
	if len(argv) < 3 || argv[0] != "git" || argv[1] != "add" {
		t.Fatalf("first call should be git add, got %v", argv)
	}
}

func TestCommit_EmptyStaged(t *testing.T) {
	// When `git diff --cached --quiet` exits 0, the commit
	// is rejected as empty.
	g := GitCommitTool{
		Runner: &fakeRunner{
			commands: map[string]sandbox.SandboxResult{
				"git": {ExitCode: 0}, // diff --cached --quiet → 0
			},
		},
	}
	_, err := g.Run(map[string]any{"message": "msg"}, tools.Context{Workspace: t.TempDir()})
	if !errors.Is(err, ErrEmptyCommit) {
		t.Fatalf("err = %v, want ErrEmptyCommit", err)
	}
}

func TestCommit_HappyPath(t *testing.T) {
	// First call (diff --cached) returns exit 1 (something
	// to commit). The commit and log calls return 0.
	call := 0
	runner := &scriptedRunner{
		fn: func(argv []string) sandbox.SandboxResult {
			call++
			if len(argv) >= 3 && argv[1] == "diff" {
				// diff --cached --quiet → 1 (changes present)
				return sandbox.SandboxResult{ExitCode: 1}
			}
			if len(argv) >= 2 && argv[1] == "commit" {
				return sandbox.SandboxResult{ExitCode: 0, Stdout: "[main abc123] msg\n"}
			}
			if len(argv) >= 2 && argv[1] == "log" {
				return sandbox.SandboxResult{
					ExitCode: 0,
					Stdout:   "abc123|Alice|2024-01-01T00:00:00Z|msg\x1e",
				}
			}
			return sandbox.SandboxResult{ExitCode: 0}
		},
	}
	g := GitCommitTool{Runner: runner}
	res, err := g.Run(map[string]any{"message": "msg"}, tools.Context{Workspace: t.TempDir()})
	assertOK(t, res, err)
	commit := res.Output["commit"].(GitCommit)
	if commit.Hash != "abc123" {
		t.Fatalf("hash = %q", commit.Hash)
	}
	if !strings.Contains(res.Summary, "abc123") {
		t.Fatalf("summary = %q", res.Summary)
	}
	if call < 3 {
		t.Fatalf("expected ≥3 runner calls, got %d", call)
	}
}

func TestCommit_AddFailure(t *testing.T) {
	// When `git add` fails, the tool should report it.
	g := GitCommitTool{
		Runner: &fakeRunner{err: errors.New("boom")},
	}
	_, err := g.Run(map[string]any{
		"message": "msg", "files": []any{"a.txt"},
	}, tools.Context{Workspace: t.TempDir()})
	if !errors.Is(err, ErrGitCommand) {
		t.Fatalf("err = %v", err)
	}
}

func TestCommit_CommitFailure(t *testing.T) {
	// diff returns 1 (something to commit), commit returns an error.
	runner := &scriptedRunner{
		fn: func(argv []string) sandbox.SandboxResult {
			if len(argv) >= 3 && argv[1] == "diff" {
				return sandbox.SandboxResult{ExitCode: 1}
			}
			if len(argv) >= 2 && argv[1] == "commit" {
				return sandbox.SandboxResult{ExitCode: 1, Stderr: "commit failed"}
			}
			return sandbox.SandboxResult{ExitCode: 0}
		},
	}
	g := GitCommitTool{Runner: runner}
	_, err := g.Run(map[string]any{"message": "msg"}, tools.Context{Workspace: t.TempDir()})
	if !errors.Is(err, ErrGitCommand) {
		t.Fatalf("err = %v", err)
	}
}

func TestCommit_Name(t *testing.T) {
	if (GitCommitTool{}).Name() != "git_commit" {
		t.Fatal("name")
	}
}

func TestCommit_Integration(t *testing.T) {
	if _, err := execLookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	dir := initRepo(t)
	writeFile(t, dir, "a.txt", "hi\n")
	g := GitCommitTool{}
	res, err := g.Run(map[string]any{
		"message": "add a", "files": []any{"a.txt"},
	}, tools.Context{Workspace: dir})
	assertOK(t, res, err)
	commit := res.Output["commit"].(GitCommit)
	if commit.Hash == "" {
		t.Fatal("hash is empty")
	}
}

// scriptedRunner returns SandboxResult based on a function.
// When the function returns ExitCode != 0, the runner also
// returns an error so the tool sees a failure.
type scriptedRunner struct {
	fn func(argv []string) sandbox.SandboxResult
}

func (s *scriptedRunner) Run(opts sandbox.RunOptions) (sandbox.SandboxResult, error) {
	r := s.fn(opts.Command)
	if r.ExitCode != 0 {
		return r, errors.New("non-zero exit")
	}
	return r, nil
}
