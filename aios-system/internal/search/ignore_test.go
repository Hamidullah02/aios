package search

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadIgnore_Missing(t *testing.T) {
	p := loadIgnore(t.TempDir())
	if p.match("anything") {
		t.Fatal("empty patterns should not match")
	}
}

func TestLoadIgnore_CommentsAndBlank(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".aiosignore"),
		[]byte("# a comment\n\n*.bak\n# another\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	p := loadIgnore(dir)
	if !p.match("a.bak") {
		t.Fatal(".bak should match")
	}
	if p.match("a.go") {
		t.Fatal(".go should not match")
	}
}

func TestIgnoreMatch_DirectoryPattern(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, ".aiosignore"),
		[]byte("node_modules/\n"), 0o644)
	p := loadIgnore(dir)
	if !p.match("node_modules/foo.js") {
		t.Fatal("directory pattern should match nested file")
	}
	if p.match("src/foo.js") {
		t.Fatal("directory pattern should not match unrelated path")
	}
}
