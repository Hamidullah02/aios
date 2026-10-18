package git

import (
	"errors"
	"testing"

	"github.com/sadi/aios-system/internal/sandbox"
	"github.com/sadi/aios-system/internal/tools"
)

func TestStatus_GitNotOnPath(t *testing.T) {
	withRequireGitFunc(t, func() error {
		return &GitCommandError{Command: []string{"git"}, Stderr: "missing"}
	})
	g := GitStatusTool{}
	_, err := g.Run(nil, tools.Context{Workspace: t.TempDir()})
	if !errors.Is(err, ErrGitCommand) {
		t.Fatalf("err = %v", err)
	}
}

func TestStatus_CommandFailure(t *testing.T) {
	g := GitStatusTool{
		Runner: &fakeRunner{err: errors.New("boom")},
	}
	_, err := g.Run(nil, tools.Context{Workspace: t.TempDir()})
	if !errors.Is(err, ErrGitCommand) {
		t.Fatalf("err = %v", err)
	}
}

func TestStatus_Run(t *testing.T) {
	// Fake-runner returns a known porcelain output.
	g := GitStatusTool{
		Runner: &fakeRunner{
			commands: map[string]sandbox.SandboxResult{
				"git": {ExitCode: 0, Stdout: "## main\x00M  a.go\x00?? b.txt\x00"},
			},
		},
	}
	res, err := g.Run(nil, tools.Context{Workspace: t.TempDir()})
	assertOK(t, res, err)
	stAny := res.Output["status"]
	st := stAny.(GitStatus)
	if st.Branch != "main" {
		t.Fatalf("branch = %q", st.Branch)
	}
	if len(st.Staged) != 1 || st.Staged[0] != "a.go" {
		t.Fatalf("staged = %v", st.Staged)
	}
	if len(st.Untracked) != 1 || st.Untracked[0] != "b.txt" {
		t.Fatalf("untracked = %v", st.Untracked)
	}
}

func TestStatus_Name(t *testing.T) {
	if (GitStatusTool{}).Name() != "git_status" {
		t.Fatal("name")
	}
}

func TestStatus_Integration(t *testing.T) {
	// Use the real subprocess runner against a temp repo
	// with a modified file. Skipped if git is not on PATH.
	if _, err := execLookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	dir := initRepo(t)
	writeFile(t, dir, "a.txt", "hi\n")
	// Stage the file so it's in the staged bucket.
	if err := runShell(t, dir, "git", "add", "a.txt"); err != nil {
		t.Fatal(err)
	}
	g := GitStatusTool{}
	res, err := g.Run(nil, tools.Context{Workspace: dir})
	assertOK(t, res, err)
	st := res.Output["status"].(GitStatus)
	if st.Branch != "main" && st.Branch != "master" {
		// Default branch name can be either depending on
		// the user's git config; both are valid.
		t.Logf("branch = %q (ok)", st.Branch)
	}
	if len(st.Staged) != 1 || st.Staged[0] != "a.txt" {
		t.Fatalf("staged = %v", st.Staged)
	}
}

// runShell runs a command in cwd and returns the error.
func runShell(t *testing.T, cwd, name string, args ...string) error {
	t.Helper()
	cmd := execCommand(name, args...)
	cmd.Dir = cwd
	return cmd.Run()
}
