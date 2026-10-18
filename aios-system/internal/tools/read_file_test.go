package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadFileTool_Happy(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hello.txt")
	if err := os.WriteFile(path, []byte("line1\nline2\nline3"), 0o644); err != nil {
		t.Fatal(err)
	}
	r := ReadFileTool{}
	res, err := r.Run(map[string]any{"path": "hello.txt"}, Context{Workspace: dir})
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK {
		t.Fatalf("OK = false")
	}
	if got := res.Output["content"]; got != "line1\nline2\nline3" {
		t.Fatalf("content = %q", got)
	}
}

func TestReadFileTool_Outside(t *testing.T) {
	dir := t.TempDir()
	other := t.TempDir()
	r := ReadFileTool{}
	_, err := r.Run(map[string]any{"path": filepath.Join(other, "x")}, Context{Workspace: dir})
	if err == nil {
		t.Fatal("expected error for outside-workspace path")
	}
}

func TestReadFileTool_MissingPath(t *testing.T) {
	r := ReadFileTool{}
	_, err := r.Run(map[string]any{}, Context{Workspace: t.TempDir()})
	if err == nil {
		t.Fatal("expected error for missing path")
	}
}

func TestReadFileTool_OffsetLimit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "f.txt")
	if err := os.WriteFile(path, []byte("a\nb\nc\nd\ne"), 0o644); err != nil {
		t.Fatal(err)
	}
	r := ReadFileTool{}
	res, err := r.Run(map[string]any{
		"path":   "f.txt",
		"offset": 1,
		"limit":  2,
	}, Context{Workspace: dir})
	if err != nil {
		t.Fatal(err)
	}
	if got := res.Output["content"]; !strings.Contains(got.(string), "b") || !strings.Contains(got.(string), "c") {
		t.Fatalf("content = %q", got)
	}
}
