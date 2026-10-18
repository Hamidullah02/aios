package tools

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestSearchFilesTool_FindGo(t *testing.T) {
	dir := t.TempDir()
	for _, n := range []string{"a.go", "b.go", "c.txt"} {
		if err := os.WriteFile(filepath.Join(dir, n), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "sub", "d.go"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := SearchFilesTool{}
	res, err := s.Run(map[string]any{"pattern": "*.go"}, Context{Workspace: dir})
	if err != nil {
		t.Fatal(err)
	}
	got := res.Output["matches"].([]string)
	sort.Strings(got)
	want := []string{"a.go", "b.go", filepath.Join("sub", "d.go")}
	if !equalStrings(got, want) {
		t.Fatalf("matches = %v, want %v", got, want)
	}
}

func TestSearchFilesTool_Exclude(t *testing.T) {
	dir := t.TempDir()
	for _, n := range []string{"keep.go", "skip.go", "skip.bak"} {
		_ = os.WriteFile(filepath.Join(dir, n), []byte("x"), 0o644)
	}
	s := SearchFilesTool{}
	res, err := s.Run(map[string]any{
		"pattern": "*.go",
		"exclude": "skip*",
	}, Context{Workspace: dir})
	if err != nil {
		t.Fatal(err)
	}
	got := res.Output["matches"].([]string)
	if len(got) != 1 || got[0] != "keep.go" {
		t.Fatalf("matches = %v", got)
	}
}

func TestSearchFilesTool_Include(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "cmd"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "internal"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{
		filepath.Join("cmd", "a.go"),
		filepath.Join("internal", "b.go"),
	} {
		_ = os.WriteFile(filepath.Join(dir, p), []byte("x"), 0o644)
	}
	s := SearchFilesTool{}
	res, err := s.Run(map[string]any{
		"pattern": "*.go",
		"include": "cmd/*.go",
	}, Context{Workspace: dir})
	if err != nil {
		t.Fatal(err)
	}
	got := res.Output["matches"].([]string)
	if len(got) != 1 {
		t.Fatalf("matches = %v", got)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
