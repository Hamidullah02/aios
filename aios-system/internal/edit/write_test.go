package edit

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Hamidullah02/aios/aios-system/internal/tools"
)

func TestWriteTool_Run(t *testing.T) {
	dir := t.TempDir()
	w := WriteTool{}
	res, err := w.Run(map[string]any{
		"path":    "a.txt",
		"content": "hello\n",
	}, tools.Context{Workspace: dir})
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK {
		t.Fatal("not OK")
	}
	data, err := os.ReadFile(filepath.Join(dir, "a.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello\n" {
		t.Fatalf("got %q", data)
	}
}

func TestWriteTool_Name(t *testing.T) {
	if (WriteTool{}).Name() != "write" {
		t.Fatal("name mismatch")
	}
}
