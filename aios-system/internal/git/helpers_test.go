package git

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/sadi/aios-system/internal/sandbox"
	"github.com/sadi/aios-system/internal/tools"
)

// fakeRunner is a deterministic stand-in for sandbox.Runner.
// commands maps the first argv element to the desired
// SandboxResult; otherwise the run returns the default
// error.
type fakeRunner struct {
	commands map[string]sandbox.SandboxResult
	err      error
}

func (f *fakeRunner) Run(opts sandbox.RunOptions) (sandbox.SandboxResult, error) {
	if f.err != nil {
		return sandbox.SandboxResult{}, f.err
	}
	if len(opts.Command) == 0 {
		return sandbox.SandboxResult{}, errors.New("no command")
	}
	if r, ok := f.commands[opts.Command[0]]; ok {
		return r, nil
	}
	return sandbox.SandboxResult{ExitCode: 1, Stderr: "unknown: " + opts.Command[0]}, nil
}

// recordingRunner records every Run call.
type recordingRunner struct {
	calls []sandbox.RunOptions
	res   sandbox.SandboxResult
}

func (r *recordingRunner) Run(opts sandbox.RunOptions) (sandbox.SandboxResult, error) {
	r.calls = append(r.calls, opts)
	return r.res, nil
}

// lastCall returns the most recent call. Returns false when
// the runner has not been called.
func (r *recordingRunner) lastCall() (sandbox.RunOptions, bool) {
	if len(r.calls) == 0 {
		return sandbox.RunOptions{}, false
	}
	return r.calls[len(r.calls)-1], true
}

// firstCall returns the first call. Returns false when
// the runner has not been called.
func (r *recordingRunner) firstCall() (sandbox.RunOptions, bool) {
	if len(r.calls) == 0 {
		return sandbox.RunOptions{}, false
	}
	return r.calls[0], true
}

// gitLookPath says "git is always on PATH". Used by tests
// that want to override the package-level requireGitFunc.
func gitLookPath(name string) (string, error) {
	if name == "git" {
		return "/usr/bin/git", nil
	}
	return "", errors.New("missing: " + name)
}

// missingGitLookPath says "git is not on PATH".
func missingGitLookPath(name string) (string, error) {
	return "", errors.New("not found: " + name)
}

// initRepo creates a temp git repo with one initial empty
// commit. The repo is configured with a fixed user name and
// email so commits succeed.
func initRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	cmds := [][]string{
		{"git", "init", "-q"},
		{"git", "config", "user.email", "ai@test.local"},
		{"git", "config", "user.name", "ai"},
		{"git", "config", "commit.gpgsign", "false"},
		{"git", "commit", "--allow-empty", "-m", "init", "-q"},
	}
	for _, c := range cmds {
		cmd := exec.Command(c[0], c[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v: %v\n%s", c, err, out)
		}
	}
	return dir
}

// writeFile writes body to dir/name.
func writeFile(t *testing.T, dir, name, body string) {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertOK(t *testing.T, res tools.Result, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK {
		t.Fatalf("not OK: %s", res.Summary)
	}
}

// execCommand builds an os/exec.Cmd. We wrap it so callers
// don't need to import os/exec at every test file.
func execCommand(name string, args ...string) *exec.Cmd {
	return exec.Command(name, args...)
}

// execLookPath is the os/exec.LookPath wrapper used by
// integration tests that want to skip when git is missing.
func execLookPath(name string) (string, error) {
	return exec.LookPath(name)
}
