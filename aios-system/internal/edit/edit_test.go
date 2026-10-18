package edit

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sadi/aios-system/internal/tools"
)

func TestEdit_ReplaceUnique(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.txt", "hello world\n")
	res, err := EditTool{}.Run(map[string]any{
		"path":       "a.txt",
		"old_string": "hello",
		"new_string": "hi",
	}, tools.Context{Workspace: dir})
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK {
		t.Fatal("not OK")
	}
	if got := readFile(t, filepath.Join(dir, "a.txt")); got != "hi world\n" {
		t.Fatalf("got %q", got)
	}
	if got := res.Output["count"].(int); got != 1 {
		t.Fatalf("count = %d", got)
	}
}

func TestEdit_NotUnique(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.txt", "x x x\n")
	_, err := EditTool{}.Run(map[string]any{
		"path":       "a.txt",
		"old_string": "x",
		"new_string": "y",
	}, tools.Context{Workspace: dir})
	var e *EditNotUniqueError
	if !errors.As(err, &e) || e.Count != 3 {
		t.Fatalf("err = %v", err)
	}
}

func TestEdit_ReplaceAll(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.txt", "x x x\n")
	res, err := EditTool{}.Run(map[string]any{
		"path":        "a.txt",
		"old_string":  "x",
		"new_string":  "y",
		"replace_all": true,
	}, tools.Context{Workspace: dir})
	if err != nil {
		t.Fatal(err)
	}
	if got := res.Output["count"].(int); got != 3 {
		t.Fatalf("count = %d", got)
	}
	if got := readFile(t, filepath.Join(dir, "a.txt")); got != "y y y\n" {
		t.Fatalf("got %q", got)
	}
}

func TestEdit_NotFound(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.txt", "hello\n")
	_, err := EditTool{}.Run(map[string]any{
		"path":       "a.txt",
		"old_string": "missing",
		"new_string": "x",
	}, tools.Context{Workspace: dir})
	if !errors.Is(err, ErrEditNotFound) {
		t.Fatalf("err = %v", err)
	}
}

func TestEdit_EmptyOldString(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.txt", "hello\n")
	_, err := EditTool{}.Run(map[string]any{
		"path":       "a.txt",
		"old_string": "",
		"new_string": "x",
	}, tools.Context{Workspace: dir})
	if !errors.Is(err, ErrEditEmpty) {
		t.Fatalf("err = %v", err)
	}
}

func TestEdit_MissingPath(t *testing.T) {
	_, err := EditTool{}.Run(map[string]any{}, tools.Context{Workspace: t.TempDir()})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestEdit_PreservesContent(t *testing.T) {
	dir := t.TempDir()
	body := "alpha\nbeta\ngamma\n"
	writeFile(t, dir, "a.txt", body)
	_, err := EditTool{}.Run(map[string]any{
		"path":       "a.txt",
		"old_string": "beta",
		"new_string": "BETA",
	}, tools.Context{Workspace: dir})
	if err != nil {
		t.Fatal(err)
	}
	got := readFile(t, filepath.Join(dir, "a.txt"))
	if got != "alpha\nBETA\ngamma\n" {
		t.Fatalf("got %q", got)
	}
}

func TestEdit_DiffInOutput(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.txt", "x\ny\nz\n")
	res, err := EditTool{}.Run(map[string]any{
		"path":       "a.txt",
		"old_string": "y",
		"new_string": "Y",
	}, tools.Context{Workspace: dir})
	if err != nil {
		t.Fatal(err)
	}
	diff, _ := res.Output["diff"].(string)
	if !strings.Contains(diff, "-y") || !strings.Contains(diff, "+Y") {
		t.Fatalf("diff = %q", diff)
	}
}

func TestEdit_RejectsOutsideWorkspace(t *testing.T) {
	dir := t.TempDir()
	outside := t.TempDir()
	writeFile(t, outside, "a.txt", "x\n")
	_, err := EditTool{}.Run(map[string]any{
		"path":       outside,
		"old_string": "x",
		"new_string": "y",
	}, tools.Context{Workspace: dir})
	if err == nil {
		t.Fatal("expected path-rejected error")
	}
}

func TestEdit_FallbackWorkspace(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.txt", "hi\n")
	t.Setenv("AIOS_WORKSPACE", dir)
	res, err := EditTool{}.Run(map[string]any{
		"path":       "a.txt",
		"old_string": "hi",
		"new_string": "hello",
	}, tools.Context{})
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK {
		t.Fatal("not OK")
	}
}

func TestEdit_NoChange(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.txt", "hi\n")
	res, err := EditTool{}.Run(map[string]any{
		"path":       "a.txt",
		"old_string": "hi",
		"new_string": "hi",
	}, tools.Context{Workspace: dir})
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK {
		t.Fatal("not OK")
	}
	// No-change branch reports count=0 (no replacement
	// actually happened). File is unchanged.
	if got := readFile(t, filepath.Join(dir, "a.txt")); got != "hi\n" {
		t.Fatalf("file changed: %q", got)
	}
}

func TestEdit_ToolName(t *testing.T) {
	if (EditTool{}).Name() != "edit" {
		t.Fatal("name")
	}
	if (MultiEditTool{}).Name() != "multi_edit" {
		t.Fatal("multi name")
	}
	if (ApplyPatchTool{}).Name() != "apply_patch" {
		t.Fatal("apply_patch name")
	}
	if (NotebookEditTool{}).Name() != "notebook_edit" {
		t.Fatal("notebook name")
	}
}

func TestEdit_ReadFailure(t *testing.T) {
	dir := t.TempDir()
	_, err := EditTool{}.Run(map[string]any{
		"path":       "missing.txt",
		"old_string": "x",
		"new_string": "y",
	}, tools.Context{Workspace: dir})
	if err == nil {
		t.Fatal("expected read error")
	}
}
