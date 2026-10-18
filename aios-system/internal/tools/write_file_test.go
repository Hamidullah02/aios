package tools

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteFileTool_Happy(t *testing.T) {
	dir := t.TempDir()
	w := WriteFileTool{}
	res, err := w.Run(map[string]any{
		"path":    "out.txt",
		"content": "hello",
	}, Context{Workspace: dir})
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK {
		t.Fatal("OK = false")
	}
	got, err := os.ReadFile(filepath.Join(dir, "out.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "hello" {
		t.Fatalf("written = %q", got)
	}
}

func TestWriteFileTool_Overwrites(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.txt")
	if err := os.WriteFile(path, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	w := WriteFileTool{}
	if _, err := w.Run(map[string]any{"path": "out.txt", "content": "new"}, Context{Workspace: dir}); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(path)
	if string(got) != "new" {
		t.Fatalf("after overwrite = %q", got)
	}
}

func TestWriteFileTool_Outside(t *testing.T) {
	dir := t.TempDir()
	other := t.TempDir()
	w := WriteFileTool{}
	_, err := w.Run(map[string]any{
		"path":    filepath.Join(other, "x"),
		"content": "x",
	}, Context{Workspace: dir})
	if !errors.Is(err, ErrPathNotAllowed) {
		t.Fatalf("err = %v, want ErrPathNotAllowed", err)
	}
}

func TestWriteFileTool_Atomic_NoLeftoverTemp(t *testing.T) {
	dir := t.TempDir()
	w := WriteFileTool{}
	if _, err := w.Run(map[string]any{"path": "out.txt", "content": "x"}, Context{Workspace: dir}); err != nil {
		t.Fatal(err)
	}
	// The temp file pattern is ".write-*". No leftovers.
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if len(e.Name()) >= 7 && e.Name()[:7] == ".write-" {
			t.Fatalf("leftover temp: %s", e.Name())
		}
	}
}
