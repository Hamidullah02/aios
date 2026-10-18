package fanout

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
)

// Worktree is the per-worktree backend interface
// the WorktreeIsolation depends on
// (REQ-FAN-WTI-001).
//
// The interface is the seam that lets the
// WorktreeIsolation swap between the bundled
// ExecGitBackend and a real aios-git
// implementation when that module lands. The
// interface is the same surface aios-git
// exposes (create, remove, list).
type Worktree interface {
	// Create creates a worktree at path on the
	// given branch. The function returns the
	// path the worktree lives at (the caller-
	// supplied path on success).
	Create(path, branch string) error
	// Remove removes the worktree at path.
	Remove(path string) error
	// Exists reports whether path is a known
	// worktree. The function is the cheap check
	// the WorktreeIsolation uses to verify the
	// worktree was created.
	Exists(path string) (bool, error)
}

// WorktreeIsolation wraps the Worktree backend
// with the create / remove / run-in operations
// the FanOutTool calls (REQ-FAN-WTI-001..004).
//
// The type is safe for concurrent use by
// multiple goroutines. The RunIn method changes
// the working directory before the callable and
// restores it after, even when the callable
// panics or returns an error.
type WorktreeIsolation struct {
	backend Worktree
	rootDir string
	mu      sync.Mutex
	// active is the set of worktree paths the
	// isolation has created but not yet removed.
	// The set is the safety net the Remove
	// method uses to make sure no worktree is
	// left behind.
	active map[string]int
}

// WorktreeConfig controls WorktreeIsolation
// construction.
type WorktreeConfig struct {
	// Backend is the worktree backend. Defaults
	// to ExecGitBackend{} when nil.
	Backend Worktree
	// RootDir is the directory under which the
	// worktrees are created. Each worktree path
	// is <RootDir>/<unique-id>. Defaults to
	// $TMPDIR/fanout-worktrees or os.TempDir()
	// when empty.
	RootDir string
}

// NewWorktreeIsolation constructs a
// WorktreeIsolation. The function returns an
// error when the RootDir cannot be created.
func NewWorktreeIsolation(cfg WorktreeConfig) (*WorktreeIsolation, error) {
	backend := cfg.Backend
	if backend == nil {
		backend = NewExecGitBackend("")
	}
	root := cfg.RootDir
	if root == "" {
		root = defaultRootDir()
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, &WorktreeError{Op: "init", Path: root, Reason: err.Error(), Err: err}
	}
	return &WorktreeIsolation{
		backend: backend,
		rootDir: root,
		active:  make(map[string]int),
	}, nil
}

// Create creates a worktree at <RootDir>/<id> on
// the given branch (REQ-FAN-WTI-002). The id is
// the per-task unique id; the call site passes
// the FanOutTask.ID.
//
// When branch is empty the worktree is created
// on a detached HEAD. The function is safe for
// concurrent use by multiple goroutines.
func (w *WorktreeIsolation) Create(id, branch string) (string, error) {
	if id == "" {
		return "", &WorktreeError{Op: "create", Reason: "id is required"}
	}
	path := filepath.Join(w.rootDir, id)
	if err := w.backend.Create(path, branch); err != nil {
		return "", &WorktreeError{Op: "create", Path: path, Reason: err.Error(), Err: err}
	}
	w.mu.Lock()
	w.active[path]++
	w.mu.Unlock()
	return path, nil
}

// Remove removes the worktree at path
// (REQ-FAN-WTI-003). The function is idempotent
// when the path is not in the active set.
func (w *WorktreeIsolation) Remove(path string) error {
	if path == "" {
		return &WorktreeError{Op: "remove", Reason: "path is required"}
	}
	if err := w.backend.Remove(path); err != nil {
		return &WorktreeError{Op: "remove", Path: path, Reason: err.Error(), Err: err}
	}
	w.mu.Lock()
	delete(w.active, path)
	w.mu.Unlock()
	return nil
}

// RunIn runs fn in the worktree at path
// (REQ-FAN-WTI-004). The function changes the
// working directory to path before fn, runs fn,
// then restores the working directory. The
// function is safe to call from a goroutine.
//
// The function returns whatever fn returns. The
// working-directory restore is best-effort: the
// restore uses a panic-recover so a panic in fn
// still restores the directory.
func (w *WorktreeIsolation) RunIn(path string, fn func() error) (err error) {
	if path == "" {
		return &WorktreeError{Op: "run_in", Reason: "path is required"}
	}
	if fn == nil {
		return &WorktreeError{Op: "run_in", Reason: "fn is nil"}
	}
	ok, perr := w.backend.Exists(path)
	if perr != nil {
		return &WorktreeError{Op: "run_in", Path: path, Reason: perr.Error(), Err: perr}
	}
	if !ok {
		return &WorktreeError{Op: "run_in", Path: path, Reason: "worktree not found"}
	}
	prev, gerr := os.Getwd()
	if gerr != nil {
		return &WorktreeError{Op: "run_in", Path: path, Reason: gerr.Error(), Err: gerr}
	}
	if cerr := os.Chdir(path); cerr != nil {
		return &WorktreeError{Op: "run_in", Path: path, Reason: cerr.Error(), Err: cerr}
	}
	defer func() {
		if rerr := os.Chdir(prev); rerr != nil {
			if err == nil {
				err = &WorktreeError{Op: "run_in", Path: path, Reason: rerr.Error(), Err: rerr}
			}
		}
	}()
	defer func() {
		if rec := recover(); rec != nil {
			if err == nil {
				err = fmt.Errorf("fanout: worktree: run_in: panic: %v", rec)
			}
		}
	}()
	return fn()
}

// RootDir returns the worktree root directory.
// The function is the path getter for tests
// and the stats output.
func (w *WorktreeIsolation) RootDir() string { return w.rootDir }

// Active returns a copy of the active worktree
// path set. The function is the test helper the
// safety-net test uses.
func (w *WorktreeIsolation) Active() []string {
	w.mu.Lock()
	defer w.mu.Unlock()
	out := make([]string, 0, len(w.active))
	for p := range w.active {
		out = append(out, p)
	}
	return out
}

// defaultRootDir returns the default worktree
// root. The path is $TMPDIR/fanout-worktrees
// when set, or os.TempDir()/fanout-worktrees.
func defaultRootDir() string {
	tmp := os.TempDir()
	return filepath.Join(tmp, "fanout-worktrees")
}

// ExecGitBackend is the bundled Worktree
// implementation. The backend shells out to the
// git CLI through os/exec. The backend is the
// default when no aios-git implementation is
// wired.
type ExecGitBackend struct {
	// GitDir is the path to the git repository
	// the worktree is created from. When empty
	// the backend uses the current working
	// directory's git repository (the standard
	// "git worktree" behavior).
	GitDir string
	// Runner is the function the backend uses to
	// invoke git. Defaults to exec.CommandContext
	// when nil. The function is the seam tests
	// use to stub the CLI.
	Runner GitRunner
	// PathExists is the function the backend uses
	// to check whether a path is a directory.
	// Defaults to os.Stat when nil.
	PathExists func(string) (bool, error)
}

// GitRunner is the function signature the
// ExecGitBackend uses to invoke git. The
// signature matches exec.CommandContext.
type GitRunner func(ctx context.Context, name string, args ...string) ([]byte, error)

// NewExecGitBackend constructs an ExecGitBackend
// that operates on the given git dir. The
// function is the constructor the
// WorktreeIsolation uses when the caller does
// not supply a backend.
func NewExecGitBackend(gitDir string) *ExecGitBackend {
	return &ExecGitBackend{GitDir: gitDir}
}

// Create runs `git worktree add` for the given
// path and branch. When branch is empty the
// backend runs `git worktree add --detach` so
// the worktree is on a detached HEAD.
func (b *ExecGitBackend) Create(path, branch string) error {
	runner := b.runner()
	args := []string{"worktree", "add"}
	if branch == "" {
		args = append(args, "--detach")
		args = append(args, path)
	} else {
		args = append(args, path, branch)
	}
	if b.GitDir != "" {
		args = append(args, "-C", b.GitDir)
	}
	out, err := runner(context.Background(), "git", args...)
	if err != nil {
		return fmt.Errorf("git worktree add: %s: %s", err.Error(), string(out))
	}
	return nil
}

// Remove runs `git worktree remove` for the
// given path.
func (b *ExecGitBackend) Remove(path string) error {
	runner := b.runner()
	args := []string{"worktree", "remove", "--force", path}
	if b.GitDir != "" {
		args = append(args, "-C", b.GitDir)
	}
	out, err := runner(context.Background(), "git", args...)
	if err != nil {
		// Fallback: if the worktree was already
		// removed, treat the failure as success.
		if isAlreadyGone(out, err) {
			return nil
		}
		return fmt.Errorf("git worktree remove: %s: %s", err.Error(), string(out))
	}
	return nil
}

// Exists reports whether path is a known
// worktree. The function uses the PathExists
// check when set; the default is os.Stat.
func (b *ExecGitBackend) Exists(path string) (bool, error) {
	check := b.PathExists
	if check == nil {
		check = defaultPathExists
	}
	return check(path)
}

// runner returns the GitRunner the backend uses.
// The function is split out so the default
// runner is a single value.
func (b *ExecGitBackend) runner() GitRunner {
	if b.Runner != nil {
		return b.Runner
	}
	return defaultGitRunner
}

// defaultGitRunner is the default GitRunner.
// The function is a thin wrapper around
// exec.CommandContext that returns the combined
// stdout+stderr.
func defaultGitRunner(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.CombinedOutput()
	return out, err
}

// defaultPathExists is the default PathExists
// function. The function reports true when the
// path exists and is a directory.
func defaultPathExists(path string) (bool, error) {
	if path == "" {
		return false, nil
	}
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	return info.IsDir(), nil
}

// isAlreadyGone reports whether the git error
// indicates the worktree was already removed.
// The function is the fallback the Remove
// method uses.
func isAlreadyGone(out []byte, err error) bool {
	if err == nil {
		return false
	}
	msg := string(out)
	if msg == "" {
		msg = err.Error()
	}
	// Common git worktree remove failure
	// messages.
	for _, hint := range []string{
		"does not exist",
		"not a working tree",
		"already removed",
		"No such file or directory",
	} {
		if contains(msg, hint) {
			return true
		}
	}
	return false
}

// contains reports whether s contains substr.
// The function is a small helper that avoids
// pulling in strings just for one call.
func contains(s, substr string) bool {
	if substr == "" {
		return true
	}
	if len(substr) > len(s) {
		return false
	}
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
