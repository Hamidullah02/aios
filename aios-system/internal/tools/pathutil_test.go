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
	abs, _ := filepath.Abs(".")
	if ws != abs {
		t.Fatalf("workspace = %q, want %q", ws, abs)
	}
}

func TestWorkspace_FromEnv(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(EnvWorkspace, dir)
	ws, err := Workspace()
	if err != nil {
		t.Fatal(err)
	}
	if ws != filepath.Clean(dir) {
		t.Fatalf("workspace = %q, want %q", ws, dir)
	}
}

func TestResolvePath_Inside(t *testing.T) {
	dir := t.TempDir()
	got, err := ResolvePath(dir, "a/b.txt")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(dir, "a", "b.txt")
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
