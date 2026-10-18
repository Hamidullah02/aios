package fanout

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

// fakeBackend is the test double for the
// Worktree interface. The double records the
// create/remove calls and actually creates a
// directory on Create so the RunIn path can
// chdir into it.
type fakeBackend struct {
	created    []string
	removed    []string
	createFail error
	removeFail error
	count      int32
}

func (f *fakeBackend) Create(path, branch string) error {
	if c := atomic.AddInt32(&f.count, 1); c == 0 {
		return nil // wraparound, ignore
	}
	if f.createFail != nil {
		return f.createFail
	}
	if err := os.MkdirAll(path, 0o755); err != nil {
		return err
	}
	f.created = append(f.created, path)
	return nil
}

func (f *fakeBackend) Remove(path string) error {
	// Best-effort: ignore the remove error in
	// tests so the test can run without git.
	_ = os.RemoveAll(path)
	if f.removeFail != nil {
		return f.removeFail
	}
	f.removed = append(f.removed, path)
	return nil
}

func (f *fakeBackend) Exists(path string) (bool, error) {
	for _, p := range f.created {
		if p == path {
			return true, nil
		}
	}
	return defaultPathExists(path)
}

// TestWorktreeIsolationCreateRemove covers the
// create / remove cycle (REQ-FAN-WTI-002, -003).
func TestWorktreeIsolationCreateRemove(t *testing.T) {
	be := &fakeBackend{}
	w, err := NewWorktreeIsolation(WorktreeConfig{Backend: be, RootDir: t.TempDir()})
	if err != nil {
		t.Fatalf("NewWorktreeIsolation: %v", err)
	}
	path, err := w.Create("t1", "feat/x")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if !strings.HasSuffix(path, "t1") {
		t.Fatalf("path = %q, want suffix t1", path)
	}
	if len(be.created) != 1 {
		t.Fatalf("created = %d, want 1", len(be.created))
	}
	if err := w.Remove(path); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if len(be.removed) != 1 {
		t.Fatalf("removed = %d, want 1", len(be.removed))
	}
	if len(w.Active()) != 0 {
		t.Fatalf("active = %d, want 0", len(w.Active()))
	}
}

// TestWorktreeIsolationCreateEmptyID covers the
// empty-id guard.
func TestWorktreeIsolationCreateEmptyID(t *testing.T) {
	w, _ := NewWorktreeIsolation(WorktreeConfig{Backend: &fakeBackend{}, RootDir: t.TempDir()})
	if _, err := w.Create("", ""); err == nil {
		t.Fatal("expected error for empty id")
	}
}

// TestWorktreeIsolationRemoveEmptyPath covers
// the empty-path guard.
func TestWorktreeIsolationRemoveEmptyPath(t *testing.T) {
	w, _ := NewWorktreeIsolation(WorktreeConfig{Backend: &fakeBackend{}, RootDir: t.TempDir()})
	if err := w.Remove(""); err == nil {
		t.Fatal("expected error for empty path")
	}
}

// TestWorktreeIsolationRunIn covers the run-in
// semantics (REQ-FAN-WTI-004). The function
// changes the working directory, runs the
// callable, and restores the directory.
func TestWorktreeIsolationRunIn(t *testing.T) {
	be := &fakeBackend{}
	w, _ := NewWorktreeIsolation(WorktreeConfig{Backend: be, RootDir: t.TempDir()})
	path, _ := w.Create("t1", "")
	prev, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	defer os.Chdir(prev)
	runErr := w.RunIn(path, func() error {
		wd, _ := os.Getwd()
		// On Windows the path may have a drive
		// letter prefix; compare case-insensitively.
		if !strings.EqualFold(wd, path) {
			return errors.New("cwd not changed")
		}
		return nil
	})
	if runErr != nil {
		t.Fatalf("RunIn: %v", runErr)
	}
	cur, _ := os.Getwd()
	if !strings.EqualFold(cur, prev) {
		t.Fatalf("cwd not restored: %q vs %q", cur, prev)
	}
}

// TestWorktreeIsolationRunInNotFound covers the
// not-found path.
func TestWorktreeIsolationRunInNotFound(t *testing.T) {
	w, _ := NewWorktreeIsolation(WorktreeConfig{Backend: &fakeBackend{}, RootDir: t.TempDir()})
	err := w.RunIn("/nonexistent", func() error { return nil })
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestWorktreeIsolationRunInNilFn covers the
// nil-fn guard.
func TestWorktreeIsolationRunInNilFn(t *testing.T) {
	be := &fakeBackend{}
	w, _ := NewWorktreeIsolation(WorktreeConfig{Backend: be, RootDir: t.TempDir()})
	path, _ := w.Create("t1", "")
	if err := w.RunIn(path, nil); err == nil {
		t.Fatal("expected error for nil fn")
	}
}

// TestWorktreeIsolationCreateFails covers the
// create-failure path.
func TestWorktreeIsolationCreateFails(t *testing.T) {
	be := &fakeBackend{createFail: errors.New("boom")}
	w, _ := NewWorktreeIsolation(WorktreeConfig{Backend: be, RootDir: t.TempDir()})
	_, err := w.Create("t1", "")
	if err == nil {
		t.Fatal("expected error")
	}
	if !contains(err.Error(), "boom") {
		t.Fatalf("err = %v", err)
	}
}

// TestNewWorktreeIsolationDefaults covers the
// default config.
func TestNewWorktreeIsolationDefaults(t *testing.T) {
	w, err := NewWorktreeIsolation(WorktreeConfig{})
	if err != nil {
		t.Fatalf("NewWorktreeIsolation: %v", err)
	}
	if w.RootDir() == "" {
		t.Fatal("RootDir is empty")
	}
}

// TestWorktreeIsolationRemoveBackendFails
// covers the Remove error path.
func TestWorktreeIsolationRemoveBackendFails(t *testing.T) {
	be := &fakeBackend{removeFail: errors.New("remove boom")}
	w, _ := NewWorktreeIsolation(WorktreeConfig{Backend: be, RootDir: t.TempDir()})
	// Create first so the path is recorded.
	path, _ := w.Create("t1", "")
	if err := w.Remove(path); err == nil {
		t.Fatal("expected error")
	} else if !contains(err.Error(), "remove boom") {
		t.Fatalf("err = %v", err)
	}
}

// TestWorktreeIsolationRunInPanicRecovers
// covers the panic-recover path: the
// working directory should be restored even
// when the callable panics.
func TestWorktreeIsolationRunInPanicRecovers(t *testing.T) {
	be := &fakeBackend{}
	w, _ := NewWorktreeIsolation(WorktreeConfig{Backend: be, RootDir: t.TempDir()})
	path, _ := w.Create("t1", "")
	prev, _ := os.Getwd()
	defer os.Chdir(prev)
	err := w.RunIn(path, func() error {
		panic("boom")
	})
	if err == nil {
		t.Fatal("expected error from panic")
	}
	if !contains(err.Error(), "panic") {
		t.Fatalf("err = %v", err)
	}
	cur, _ := os.Getwd()
	if !strings.EqualFold(cur, prev) {
		t.Fatalf("cwd not restored: %q vs %q", cur, prev)
	}
}

// TestWorktreeIsolationActiveEmpty covers the
// empty active set path.
func TestWorktreeIsolationActiveEmpty(t *testing.T) {
	w, _ := NewWorktreeIsolation(WorktreeConfig{Backend: &fakeBackend{}, RootDir: t.TempDir()})
	if got := w.Active(); len(got) != 0 {
		t.Fatalf("Active = %d, want 0", len(got))
	}
}

// TestExecGitBackendCreateDryRun covers the
// ExecGitBackend's runner stub. The test does
// not shell out; it replaces the runner with a
// function that records the args and returns
// success.
func TestExecGitBackendCreateDryRun(t *testing.T) {
	be := NewExecGitBackend("")
	var gotName string
	var gotArgs []string
	be.Runner = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		gotName = name
		gotArgs = args
		return nil, nil
	}
	if err := be.Create("/tmp/wt", "feat/x"); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if gotName != "git" {
		t.Fatalf("name = %q, want git", gotName)
	}
	// Expect: worktree add /tmp/wt feat/x
	if len(gotArgs) < 4 || gotArgs[0] != "worktree" || gotArgs[1] != "add" {
		t.Fatalf("args = %v", gotArgs)
	}
	if gotArgs[2] != "/tmp/wt" || gotArgs[3] != "feat/x" {
		t.Fatalf("path/branch = %v", gotArgs)
	}
}

// TestExecGitBackendCreateDetached covers the
// empty-branch path (detached HEAD).
func TestExecGitBackendCreateDetached(t *testing.T) {
	be := NewExecGitBackend("")
	var gotArgs []string
	be.Runner = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		gotArgs = args
		return nil, nil
	}
	if err := be.Create("/tmp/wt", ""); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if gotArgs[2] != "--detach" {
		t.Fatalf("args = %v, want --detach", gotArgs)
	}
}

// TestExecGitBackendCreateError covers the
// runner-failure path.
func TestExecGitBackendCreateError(t *testing.T) {
	be := NewExecGitBackend("")
	be.Runner = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		return []byte("boom"), errors.New("exit 1")
	}
	err := be.Create("/tmp/wt", "")
	if err == nil {
		t.Fatal("expected error")
	}
	if !contains(err.Error(), "boom") {
		t.Fatalf("err = %v", err)
	}
}

// TestExecGitBackendRemove covers the remove
// path.
func TestExecGitBackendRemove(t *testing.T) {
	be := NewExecGitBackend("")
	be.Runner = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		return nil, nil
	}
	if err := be.Remove("/tmp/wt"); err != nil {
		t.Fatalf("Remove: %v", err)
	}
}

// TestExecGitBackendRemoveAlreadyGone covers
// the "worktree already removed" fallback.
func TestExecGitBackendRemoveAlreadyGone(t *testing.T) {
	be := NewExecGitBackend("")
	be.Runner = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		return []byte("fatal: /tmp/wt does not exist"), errors.New("exit 1")
	}
	if err := be.Remove("/tmp/wt"); err != nil {
		t.Fatalf("Remove (already gone): %v", err)
	}
}

// TestExecGitBackendExists covers the Exists
// helper.
func TestExecGitBackendExists(t *testing.T) {
	be := NewExecGitBackend("")
	dir := t.TempDir()
	ok, err := be.Exists(dir)
	if err != nil {
		t.Fatalf("Exists: %v", err)
	}
	if !ok {
		t.Fatal("expected exists for temp dir")
	}
	ok, _ = be.Exists(filepath.Join(dir, "missing"))
	if ok {
		t.Fatal("expected not exists")
	}
	ok, _ = be.Exists("")
	if ok {
		t.Fatal("empty path should not exist")
	}
}

// TestIsAlreadyGone covers the helper the
// Remove method uses to detect the "already
// gone" git error.
func TestIsAlreadyGone(t *testing.T) {
	cases := []struct {
		name string
		out  string
		err  error
		want bool
	}{
		{"does not exist", "fatal: /tmp/wt does not exist", errors.New("exit 1"), true},
		{"not a working tree", "fatal: /tmp/wt is not a working tree", errors.New("exit 1"), true},
		{"no such file", "No such file or directory", errors.New("exit 1"), true},
		{"other error", "some other error", errors.New("exit 1"), false},
		{"err only hint", "", errors.New("does not exist"), true},
		{"no err", "", nil, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := isAlreadyGone([]byte(c.out), c.err); got != c.want {
				t.Fatalf("isAlreadyGone(%q, %v) = %v, want %v", c.out, c.err, got, c.want)
			}
		})
	}
}

// TestContainsCoversAllBranches covers the
// contains helper's edge cases.
func TestContainsCoversAllBranches(t *testing.T) {
	if !contains("hello world", "") {
		t.Fatal("empty substr should always be contained")
	}
	if !contains("hello world", "world") {
		t.Fatal("world should be contained")
	}
	if contains("hi", "world") {
		t.Fatal("world should not be contained in 'hi'")
	}
}

// TestExecGitBackendCreateWithGitDir covers the
// GitDir branch.
func TestExecGitBackendCreateWithGitDir(t *testing.T) {
	be := NewExecGitBackend("/path/to/repo")
	var gotArgs []string
	be.Runner = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		gotArgs = args
		return nil, nil
	}
	if err := be.Create("/tmp/wt", "feat/x"); err != nil {
		t.Fatalf("Create: %v", err)
	}
	// Expect: -C /path/to/repo at the tail.
	found := false
	for i, a := range gotArgs {
		if a == "-C" && i+1 < len(gotArgs) && gotArgs[i+1] == "/path/to/repo" {
			found = true
		}
	}
	if !found {
		t.Fatalf("args = %v, want -C /path/to/repo", gotArgs)
	}
}

// TestExecGitBackendRemoveOtherFailure covers
// the Remove error path when the error is not
// the "already gone" case.
func TestExecGitBackendRemoveOtherFailure(t *testing.T) {
	be := NewExecGitBackend("")
	be.Runner = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		return []byte("some other failure"), errors.New("exit 2")
	}
	if err := be.Remove("/tmp/wt"); err == nil {
		t.Fatal("expected error")
	} else if !contains(err.Error(), "some other failure") {
		t.Fatalf("err = %v", err)
	}
}
