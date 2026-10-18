// Package git is the git tool set.
//
// The package exports six tools: GitStatusTool, GitDiffTool,
// GitLogTool, GitCommitTool, GitBranchTool, GitWorktreeTool.
// Each tool implements the tools.Tool interface. The tools
// shell out to the `git` CLI through the sandbox runner from
// aios-sandbox; they do not link a git library.
package git

import (
	"errors"
	"fmt"
)

// Sentinel errors. Per the aios-git SRS §9.
var (
	// ErrNotARepository is raised when the workspace is not
	// a git repo (REQ-GIT-ERR-001).
	ErrNotARepository = errors.New("git: not a repository")

	// ErrGitCommand is raised when the git command fails
	// (REQ-GIT-ERR-002).
	ErrGitCommand = errors.New("git: command failed")

	// ErrEmptyCommit is raised when the commit tool finds
	// no files to commit (REQ-GIT-CMT-005).
	ErrEmptyCommit = errors.New("git: empty commit")
)

// NotARepositoryError is the typed error for non-repos.
type NotARepositoryError struct {
	Path string
}

func (e *NotARepositoryError) Error() string {
	return fmt.Sprintf("git: %q is not a repository", e.Path)
}

func (e *NotARepositoryError) Is(target error) bool { return target == ErrNotARepository }

// GitCommandError is the typed error for failed git commands.
// Command is the full argv; Stderr is the captured stderr.
type GitCommandError struct {
	Command []string
	Stderr  string
	Exit    int
}

func (e *GitCommandError) Error() string {
	return fmt.Sprintf("git: %v: exit %d: %s", e.Command, e.Exit, e.Stderr)
}

func (e *GitCommandError) Is(target error) bool { return target == ErrGitCommand }

// EmptyCommitError is the typed error for an empty commit.
type EmptyCommitError struct {
	Reason string
}

func (e *EmptyCommitError) Error() string {
	return fmt.Sprintf("git: empty commit: %s", e.Reason)
}

func (e *EmptyCommitError) Is(target error) bool { return target == ErrEmptyCommit }
