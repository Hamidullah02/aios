package tools

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestWorkspace_Default(t *testing.T) {
	ws, err := Workspace()
	if err != nil {
		t.Fatal(err)
	}
	// The workspace is canonicalized, so it is compared in
	// canonical form. On Windows that is what makes the
	// difference: EvalSymlinks expands an 8.3 short name into
	// its long form, so the raw path and the canonical one are
	// different strings for the same directory.
	want, err := canonicalPath(".")
	if err != nil {
		t.Fatal(err)
	}
	if ws != want {
		t.Fatalf("workspace = %q, want %q", ws, want)
	}
}

func TestWorkspace_FromEnv(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(EnvWorkspace, dir)
	ws, err := Workspace()
	if err != nil {
		t.Fatal(err)
	}
	want, err := canonicalPath(dir)
	if err != nil {
		t.Fatal(err)
	}
	if ws != want {
		t.Fatalf("workspace = %q, want %q", ws, want)
	}
}

// TestWorkspaceAndPathUseTheSameForm pins that the workspace and
// the paths inside it are canonicalized the same way.
//
// This is the bug the Windows CI runner found. On Windows,
// EvalSymlinks expands an 8.3 short name into its long form:
// C:\Users\RUNNER~1\AppData\Local\Temp becomes
// C:\Users\runneradmin\AppData\Local\Temp.
//
// The path was canonicalized and the workspace was not, so the two
// no longer matched and every path inside the workspace was
// reported as being outside it. Every file tool -- read, write,
// edit, apply_patch, notebook_edit -- refused to do its job.
func TestWorkspaceAndPathUseTheSameForm(t *testing.T) {
	dir := t.TempDir()
	want, err := canonicalPath(dir)
	if err != nil {
		t.Fatal(err)
	}
	// The workspace in its raw form.
	t.Setenv(EnvWorkspace, dir)
	ws, err := Workspace()
	if err != nil {
		t.Fatal(err)
	}
	if ws != want {
		t.Fatalf("workspace = %q, want %q", ws, want)
	}
	// And a path inside it resolves without being refused.
	got, err := ResolvePath(dir, "a.txt")
	if err != nil {
		t.Fatalf("ResolvePath: %v", err)
	}
	if !isUnder(got, ws) {
		t.Fatalf("resolved path %q is not under the workspace %q", got, ws)
	}
}

func TestResolvePath_Inside(t *testing.T) {
	dir := t.TempDir()
	got, err := ResolvePath(dir, "a/b.txt")
	if err != nil {
		t.Fatal(err)
	}
	// The result is canonicalized, so it is compared in canonical
	// form. On Windows the temp directory's path can carry an 8.3
	// short name that EvalSymlinks expands, and the raw join does
	// not.
	want, err := canonicalPath(filepath.Join(dir, "a", "b.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestResolvePath_Outside(t *testing.T) {
	dir := t.TempDir()
	_, err := ResolvePath(dir, "../escape.txt")
	if !errors.Is(err, ErrPathNotAllowed) {
		t.Fatalf("err = %v, want ErrPathNotAllowed", err)
	}
}

func TestResolvePath_AbsoluteOutside(t *testing.T) {
	dir := t.TempDir()
	other := t.TempDir()
	_, err := ResolvePath(dir, other+"/file.txt")
	if !errors.Is(err, ErrPathNotAllowed) {
		t.Fatalf("err = %v, want ErrPathNotAllowed", err)
	}
}

func TestResolvePath_SymlinkEscape(t *testing.T) {
	// A symlink inside the workspace that points outside
	// should be rejected.
	dir := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(dir, "escape")); err != nil {
		t.Skip("symlinks not supported on this platform")
	}
	_, err := ResolvePath(dir, "escape")
	if !errors.Is(err, ErrPathNotAllowed) {
		t.Fatalf("err = %v, want ErrPathNotAllowed", err)
	}
}
