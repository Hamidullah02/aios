//go:build windows

package tools

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

// getShortPathName returns the 8.3 short form of a path.
//
// This is what the GitHub Windows runner hands the test: the
// runner's temp directory is C:\Users\RUNNER~1\AppData\Local\Temp
// and the long form is C:\Users\runneradmin\AppData\Local\Temp.
func getShortPathName(path string) (string, error) {
	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return "", err
	}
	buf := make([]uint16, 260)
	n, err := syscall.GetShortPathName(p, &buf[0], uint32(len(buf)))
	if err != nil {
		return "", err
	}
	if n == 0 {
		return "", syscall.EINVAL
	}
	return syscall.UTF16ToString(buf[:n]), nil
}

// TestResolvePath_ShortNameWorkspace reproduces the Windows CI
// failure.
//
// The runner's temp directory carries an 8.3 short name. The path
// inside it resolved to the long form, the workspace stayed in the
// short form, the prefix check failed, and every file tool
// reported the path as being outside the workspace.
//
// The check is skipped when the filesystem has no short names --
// which is the case on a developer machine, and which is exactly
// why this bug survived locally and only appeared on CI.
func TestResolvePath_ShortNameWorkspace(t *testing.T) {
	dir := t.TempDir()

	short, err := getShortPathName(dir)
	if err != nil {
		t.Skipf("no 8.3 short name for %q: %v", dir, err)
	}
	if short == "" || !strings.Contains(short, "~") {
		t.Skipf("the filesystem has no 8.3 short names: %q", short)
	}
	t.Logf("long  %q", dir)
	t.Logf("short %q", short)

	// The workspace in its short form, the path resolved from it.
	// Before the fix this was refused.
	got, err := ResolvePath(short, "a.txt")
	if err != nil {
		t.Fatalf("ResolvePath with a short-name workspace: %v", err)
	}
	want, err := canonicalPath(filepath.Join(dir, "a.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}

	// And the workspace itself must come back in the same form as
	// the resolved path.
	t.Setenv(EnvWorkspace, short)
	ws, err := Workspace()
	if err != nil {
		t.Fatal(err)
	}
	wsWant, err := canonicalPath(dir)
	if err != nil {
		t.Fatal(err)
	}
	if ws != wsWant {
		t.Fatalf("workspace = %q, want %q", ws, wsWant)
	}
}

// TestResolvePath_SymlinkEscapeStillRejected asserts that the
// short-name fix did not weaken the containment.
//
// The fix resolves the longest existing ancestor and re-attaches
// the part that does not exist yet. A symlink that points out of
// the workspace exists, so it is still followed -- and the target
// it points at is outside the workspace, so it is still refused.
func TestResolvePath_SymlinkEscapeStillRejected(t *testing.T) {
	dir := t.TempDir()
	outside := t.TempDir()

	// Creating a symlink on Windows needs the Developer Mode
	// privilege or an elevated shell. When it is not available
	// the test skips rather than pretending to have checked.
	if err := os.Symlink(outside, filepath.Join(dir, "escape")); err != nil {
		t.Skipf("cannot create a symlink here: %v", err)
	}

	// The symlink itself.
	if _, err := ResolvePath(dir, "escape"); !errors.Is(err, ErrPathNotAllowed) {
		t.Fatalf("the symlink itself: err = %v, want ErrPathNotAllowed", err)
	}
	// A file under it, which does not exist yet.
	if _, err := ResolvePath(dir, filepath.Join("escape", "new.txt")); !errors.Is(err, ErrPathNotAllowed) {
		t.Fatalf("a file under the symlink: err = %v, want ErrPathNotAllowed", err)
	}
}
