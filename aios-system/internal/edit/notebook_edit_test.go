package edit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Hamidullah02/aios/aios-system/internal/tools"
)

func sampleNotebook(t *testing.T, dir string) string {
	t.Helper()
	nb := notebook{Cells: []notebookCell{
		{ID: "a", CellType: "code", Source: []string{"x = 1\n"}},
		{ID: "b", CellType: "markdown", Source: []string{"# Hi\n"}},
	}}
	data, err := json.MarshalIndent(nb, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, "nb.ipynb")
	if err := os.WriteFile(p, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestNotebookEdit_Replace(t *testing.T) {
	dir := t.TempDir()
	p := sampleNotebook(t, dir)
	_, err := NotebookEditTool{}.Run(map[string]any{
		"path":       "nb.ipynb",
		"cell_id":    "a",
		"new_source": "x = 2\n",
		"cell_type":  "code",
		"cell_mode":  "replace",
	}, tools.Context{Workspace: dir})
	if err != nil {
		t.Fatal(err)
	}
	nb := decodeNB(t, p)
	if nb.Cells[0].Source[0] != "x = 2\n" {
		t.Fatalf("cell = %+v", nb.Cells[0])
	}
}

func TestNotebookEdit_Insert(t *testing.T) {
	dir := t.TempDir()
	p := sampleNotebook(t, dir)
	_, err := NotebookEditTool{}.Run(map[string]any{
		"path":       "nb.ipynb",
		"cell_id":    "c",
		"new_source": "y = 3\n",
		"cell_type":  "code",
		"cell_mode":  "insert",
	}, tools.Context{Workspace: dir})
	if err != nil {
		t.Fatal(err)
	}
	nb := decodeNB(t, p)
	if len(nb.Cells) != 3 {
		t.Fatalf("cells = %d", len(nb.Cells))
	}
	if nb.Cells[2].ID != "c" {
		t.Fatalf("last cell id = %q", nb.Cells[2].ID)
	}
}

func TestNotebookEdit_Delete(t *testing.T) {
	dir := t.TempDir()
	p := sampleNotebook(t, dir)
	_, err := NotebookEditTool{}.Run(map[string]any{
		"path":      "nb.ipynb",
		"cell_id":   "a",
		"cell_mode": "delete",
	}, tools.Context{Workspace: dir})
	if err != nil {
		t.Fatal(err)
	}
	nb := decodeNB(t, p)
	if len(nb.Cells) != 1 {
		t.Fatalf("cells = %d", len(nb.Cells))
	}
	if nb.Cells[0].ID != "b" {
		t.Fatalf("remaining cell = %q", nb.Cells[0].ID)
	}
}

func TestNotebookEdit_CellNotFound(t *testing.T) {
	dir := t.TempDir()
	sampleNotebook(t, dir)
	_, err := NotebookEditTool{}.Run(map[string]any{
		"path":       "nb.ipynb",
		"cell_id":    "missing",
		"new_source": "x",
		"cell_type":  "code",
		"cell_mode":  "replace",
	}, tools.Context{Workspace: dir})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestNotebookEdit_BadCellType(t *testing.T) {
	dir := t.TempDir()
	sampleNotebook(t, dir)
	_, err := NotebookEditTool{}.Run(map[string]any{
		"path":       "nb.ipynb",
		"cell_id":    "a",
		"new_source": "x",
		"cell_type":  "raw",
		"cell_mode":  "replace",
	}, tools.Context{Workspace: dir})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestNotebookEdit_BadMode(t *testing.T) {
	dir := t.TempDir()
	sampleNotebook(t, dir)
	_, err := NotebookEditTool{}.Run(map[string]any{
		"path":      "nb.ipynb",
		"cell_id":   "a",
		"cell_mode": "merge",
	}, tools.Context{Workspace: dir})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestNotebookEdit_MissingArgs(t *testing.T) {
	dir := t.TempDir()
	_, err := NotebookEditTool{}.Run(map[string]any{}, tools.Context{Workspace: dir})
	if err == nil {
		t.Fatal("expected path error")
	}
	_, err = NotebookEditTool{}.Run(map[string]any{"path": "x"},
		tools.Context{Workspace: dir})
	if err == nil {
		t.Fatal("expected cell_id error")
	}
}

func TestNotebookEdit_DeleteNotFound(t *testing.T) {
	dir := t.TempDir()
	sampleNotebook(t, dir)
	_, err := NotebookEditTool{}.Run(map[string]any{
		"path":      "nb.ipynb",
		"cell_id":   "missing",
		"cell_mode": "delete",
	}, tools.Context{Workspace: dir})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestNotebookEdit_FallbackWorkspace(t *testing.T) {
	dir := t.TempDir()
	sampleNotebook(t, dir)
	t.Setenv("AIOS_WORKSPACE", dir)
	_, err := NotebookEditTool{}.Run(map[string]any{
		"path":      "nb.ipynb",
		"cell_id":   "a",
		"cell_mode": "delete",
	}, tools.Context{})
	if err != nil {
		t.Fatal(err)
	}
}

func TestNotebookEdit_RejectsOutsideWorkspace(t *testing.T) {
	dir := t.TempDir()
	outside := t.TempDir()
	sampleNotebook(t, outside)
	_, err := NotebookEditTool{}.Run(map[string]any{
		"path":      outside,
		"cell_id":   "a",
		"cell_mode": "delete",
	}, tools.Context{Workspace: dir})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestNotebookEdit_ReadFailure(t *testing.T) {
	dir := t.TempDir()
	_, err := NotebookEditTool{}.Run(map[string]any{
		"path":      "missing.ipynb",
		"cell_id":   "a",
		"cell_mode": "delete",
	}, tools.Context{Workspace: dir})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestNotebookEdit_BadJSON(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "nb.ipynb", "not json")
	_, err := NotebookEditTool{}.Run(map[string]any{
		"path":      "nb.ipynb",
		"cell_id":   "a",
		"cell_mode": "delete",
	}, tools.Context{Workspace: dir})
	if err == nil {
		t.Fatal("expected error")
	}
}

func decodeNB(t *testing.T, path string) notebook {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var nb notebook
	if err := json.Unmarshal(data, &nb); err != nil {
		t.Fatal(err)
	}
	return nb
}
