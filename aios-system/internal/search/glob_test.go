package search

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Hamidullah02/aios/aios-system/internal/tools"
)

func TestGlob_BasicAndIncludeExclude(t *testing.T) {
	dir := t.TempDir()
	for _, n := range []string{"a.go", "b.go", "c.txt"} {
		_ = os.WriteFile(filepath.Join(dir, n), []byte("x"), 0o644)
	}
	if err := os.MkdirAll(filepath.Join(dir, "cmd"), 0o755); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(dir, "cmd", "x.go"), []byte("y"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "cmd", "skip.go"), []byte("y"), 0o644)

	g := GlobTool{}
	res, err := g.Run(map[string]any{
		"pattern": "*.go",
		"include": "cmd/*.go",
		"exclude": "cmd/skip*",
	}, tools.Context{Workspace: dir})
	if err != nil {
		t.Fatal(err)
	}
	hits := res.Output["matches"].([]string)
	if len(hits) != 1 || hits[0] != "cmd/x.go" {
		t.Fatalf("hits = %v", hits)
	}
}

func TestGlob_MissingPattern(t *testing.T) {
	g := GlobTool{}
	_, err := g.Run(map[string]any{}, tools.Context{Workspace: t.TempDir()})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGlob_RespectsAiosignore(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "keep.go"), []byte("x"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "skip.go"), []byte("x"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, ".aiosignore"), []byte("skip.go\n"), 0o644)
	res, err := GlobTool{}.Run(map[string]any{"pattern": "*.go"},
		tools.Context{Workspace: dir})
	if err != nil {
		t.Fatal(err)
	}
	hits := res.Output["matches"].([]string)
	if len(hits) != 1 || hits[0] != "keep.go" {
		t.Fatalf("hits = %v", hits)
	}
}

func TestGlob_Max(t *testing.T) {
	dir := t.TempDir()
	for _, n := range []string{"a.go", "b.go", "c.go", "d.go"} {
		_ = os.WriteFile(filepath.Join(dir, n), []byte("x"), 0o644)
	}
	res, err := GlobTool{}.Run(map[string]any{
		"pattern": "*.go", "max": 2,
	}, tools.Context{Workspace: dir})
	if err != nil {
		t.Fatal(err)
	}
	if got := res.Output["count"].(int); got != 2 {
		t.Fatalf("count = %d, want 2", got)
	}
}

func TestGlob_FallbackWorkspace(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("AIOS_WORKSPACE", dir)
	_ = os.WriteFile(filepath.Join(dir, "x.go"), []byte("x"), 0o644)
	res, err := GlobTool{}.Run(map[string]any{"pattern": "*.go"}, tools.Context{})
	if err != nil {
		t.Fatal(err)
	}
	if got := res.Output["count"].(int); got != 1 {
		t.Fatalf("count = %d, want 1", got)
	}
}

func TestToolNames(t *testing.T) {
	if (GlobTool{}).Name() != "glob" {
		t.Fatal("glob name")
	}
	if (GrepTool{}).Name() != "grep" {
		t.Fatal("grep name")
	}
	if (CodeSearchTool{}).Name() != "code_search" {
		t.Fatal("code_search name")
	}
	if (DiagnosticsTool{}).Name() != "diagnostics" {
		t.Fatal("diagnostics name")
	}
}
