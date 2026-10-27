package edit

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/Hamidullah02/aios/aios-system/internal/tools"
)

func TestMultiEdit_AppliesInOrder(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.txt", "one\ntwo\nthree\n")
	res, err := MultiEditTool{}.Run(map[string]any{
		"path": "a.txt",
		"edits": []any{
			map[string]any{"old_string": "one", "new_string": "1"},
			map[string]any{"old_string": "three", "new_string": "3"},
		},
	}, tools.Context{Workspace: dir})
	if err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, filepath.Join(dir, "a.txt")); got != "1\ntwo\n3\n" {
		t.Fatalf("got %q", got)
	}
	if got := res.Output["applied"].(int); got != 2 {
		t.Fatalf("applied = %d", got)
	}
}

func TestMultiEdit_RollsBackOnFailure(t *testing.T) {
	dir := t.TempDir()
	original := "alpha\nbeta\ngamma\n"
	writeFile(t, dir, "a.txt", original)
	_, err := MultiEditTool{}.Run(map[string]any{
		"path": "a.txt",
		"edits": []any{
			map[string]any{"old_string": "alpha", "new_string": "ALPHA"},
			map[string]any{"old_string": "missing", "new_string": "X"},
		},
	}, tools.Context{Workspace: dir})
	if !errors.Is(err, ErrEditNotFound) {
		t.Fatalf("err = %v, want ErrEditNotFound", err)
	}
	if got := readFile(t, filepath.Join(dir, "a.txt")); got != original {
		t.Fatalf("file not rolled back: %q", got)
	}
}

func TestMultiEdit_RollsBackOnNotUnique(t *testing.T) {
	dir := t.TempDir()
	original := "x\nx\ny\n"
	writeFile(t, dir, "a.txt", original)
	_, err := MultiEditTool{}.Run(map[string]any{
		"path": "a.txt",
		"edits": []any{
			map[string]any{"old_string": "x", "new_string": "Y"},
		},
	}, tools.Context{Workspace: dir})
	if !errors.Is(err, ErrEditNotUnique) {
		t.Fatalf("err = %v", err)
	}
	if got := readFile(t, filepath.Join(dir, "a.txt")); got != original {
		t.Fatalf("not rolled back: %q", got)
	}
}

func TestMultiEdit_RollsBackOnEmptyOldString(t *testing.T) {
	dir := t.TempDir()
	original := "x\n"
	writeFile(t, dir, "a.txt", original)
	_, err := MultiEditTool{}.Run(map[string]any{
		"path": "a.txt",
		"edits": []any{
			map[string]any{"old_string": "", "new_string": "y"},
		},
	}, tools.Context{Workspace: dir})
	if !errors.Is(err, ErrEditEmpty) {
		t.Fatalf("err = %v", err)
	}
	if got := readFile(t, filepath.Join(dir, "a.txt")); got != original {
		t.Fatalf("not rolled back: %q", got)
	}
}

func TestMultiEdit_NoChange(t *testing.T) {
	dir := t.TempDir()
	original := "x\ny\n"
	writeFile(t, dir, "a.txt", original)
	_, err := MultiEditTool{}.Run(map[string]any{
		"path": "a.txt",
		"edits": []any{
			map[string]any{"old_string": "x", "new_string": "x"},
		},
	}, tools.Context{Workspace: dir})
	if err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, filepath.Join(dir, "a.txt")); got != original {
		t.Fatalf("file changed: %q", got)
	}
}

func TestMultiEdit_MissingPath(t *testing.T) {
	_, err := MultiEditTool{}.Run(map[string]any{
		"edits": []any{},
	}, tools.Context{Workspace: t.TempDir()})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestMultiEdit_NoEdits(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.txt", "x\n")
	_, err := MultiEditTool{}.Run(map[string]any{
		"path":  "a.txt",
		"edits": []any{},
	}, tools.Context{Workspace: dir})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestMultiEdit_FallbackWorkspace(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.txt", "x\ny\n")
	t.Setenv("AIOS_WORKSPACE", dir)
	_, err := MultiEditTool{}.Run(map[string]any{
		"path": "a.txt",
		"edits": []any{
			map[string]any{"old_string": "x", "new_string": "X"},
		},
	}, tools.Context{})
	if err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, filepath.Join(dir, "a.txt")); got != "X\ny\n" {
		t.Fatalf("got %q", got)
	}
}

func TestMultiEdit_RejectsOutsideWorkspace(t *testing.T) {
	dir := t.TempDir()
	outside := t.TempDir()
	writeFile(t, outside, "a.txt", "x\n")
	_, err := MultiEditTool{}.Run(map[string]any{
		"path": outside,
		"edits": []any{
			map[string]any{"old_string": "x", "new_string": "y"},
		},
	}, tools.Context{Workspace: dir})
	if err == nil {
		t.Fatal("expected error")
	}
}
