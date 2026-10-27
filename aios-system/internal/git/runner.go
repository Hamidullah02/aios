package git

import (
	"os/exec"

	"github.com/Hamidullah02/aios/aios-system/internal/sandbox"
	"github.com/Hamidullah02/aios/aios-system/internal/tools"
)

// runner is the interface every tool uses to run git. The
// interface is the sandbox.Runner so the tools can be
// tested with a fake runner.
type runner = sandbox.Runner

// newRunner returns the default subprocess runner.
//
// It is the process-wide default rather than a fresh runner so a
// policy installed at startup (the network policy, for one)
// applies to git as well. A git command that fetches from a
// remote is a network operation like any other.
func newRunner() runner { return sandbox.DefaultRunner() }

// lookPath is the binary lookup. The default is exec.LookPath;
// tests can override it on the tool struct.
var lookPath = exec.LookPath

// workspaceFunc returns the workspace for a tool call. The
// package-level default reads ctx.Workspace first, then
// tools.Workspace() as a fallback. Tests can override the
// default to force the fallback error path.
var workspaceFunc = func(ctxWS string) (string, error) {
	if ctxWS != "" {
		return ctxWS, nil
	}
	return tools.Workspace()
}

// gitWorkspace is the helper used by every tool. It is a
// one-line indirection so the workspace error path is
// exercisable in tests.
func gitWorkspace(ctxWS string) (string, error) { return workspaceFunc(ctxWS) }

// requireGitFunc returns a friendly error when git is not
// on PATH. The package-level default uses exec.LookPath;
// tests can override it.
var requireGitFunc = func() error {
	if _, err := lookPath("git"); err != nil {
		return &GitCommandError{Command: []string{"git"}, Stderr: "git not found on PATH", Exit: -1}
	}
	return nil
}
