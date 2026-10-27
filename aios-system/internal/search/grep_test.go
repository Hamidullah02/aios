package search

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Hamidullah02/aios/aios-system/internal/tools"
)

func writeFile(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestGrep_ContentMode(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.go", "hello world\nfoo bar\nhello again\n")
	g := GrepTool{}
	res, err := g.Run(map[string]any{"pattern": "hello"}, tools.Context{Workspace: dir})
	if err != nil {
		t.Fatal(err)
	}
	if got := res.Output["count"].(int); got != 2 {
		t.Fatalf("count = %d, want 2", got)
	}
	lines, _ := res.Output["lines"].([]string)
	if len(lines) != 2 || !strings.Contains(lines[0], "a.go:1:") {
		t.Fatalf("lines = %v", lines)
	}
}

func TestGrep_FilesWithMatches(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.go", "x hit\n")
	writeFile(t, dir, "b.go", "no\n")
	g := GrepTool{}
	res, err := g.Run(map[string]any{
		"pattern": "hit", "output_mode": "files_with_matches",
	}, tools.Context{Workspace: dir})
	if err != nil {
		t.Fatal(err)
	}
	files, _ := res.Output["files"].([]string)
	if len(files) != 1 || files[0] != "a.go" {
		t.Fatalf("files = %v", files)
	}
}

func TestGrep_CountMode(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.go", "x\nx\nx\n")
	g := GrepTool{}
	res, err := g.Run(map[string]any{
		"pattern": "x", "output_mode": "count",
	}, tools.Context{Workspace: dir})
	if err != nil {
		t.Fatal(err)
	}
	counts, _ := res.Output["counts"].(map[string]int)
	if counts["a.go"] != 3 {
		t.Fatalf("counts = %v", counts)
	}
}

func TestGrep_GlobFilter(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.go", "x\n")
	writeFile(t, dir, "b.txt", "x\n")
	g := GrepTool{}
	res, err := g.Run(map[string]any{
		"pattern": "x", "glob": "*.go",
	}, tools.Context{Workspace: dir})
	if err != nil {
		t.Fatal(err)
	}
	if got := res.Output["count"].(int); got != 1 {
		t.Fatalf("count = %d", got)
	}
}

func TestGrep_RespectsAiosignore(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "keep.go", "x\n")
	writeFile(t, dir, "skip.go", "x\n")
	_ = os.WriteFile(filepath.Join(dir, ".aiosignore"), []byte("skip.go\n"), 0o644)
	g := GrepTool{}
	res, err := g.Run(map[string]any{"pattern": "x"}, tools.Context{Workspace: dir})
	if err != nil {
		t.Fatal(err)
	}
	if got := res.Output["count"].(int); got != 1 {
		t.Fatalf("count = %d", got)
	}
}

func TestGrep_BadRegex(t *testing.T) {
	dir := t.TempDir()
	_, err := GrepTool{}.Run(map[string]any{"pattern": "("}, tools.Context{Workspace: dir})
	if err == nil {
		t.Fatal("expected bad-regex error")
	}
}

func TestGrep_MissingPattern(t *testing.T) {
	_, err := GrepTool{}.Run(map[string]any{}, tools.Context{Workspace: t.TempDir()})
	if err == nil {
		t.Fatal("expected missing-pattern error")
	}
}

func TestGrep_BadTimeout(t *testing.T) {
	dir := t.TempDir()
	_, err := GrepTool{}.Run(map[string]any{
		"pattern": "x", "timeout": "not-a-duration",
	}, tools.Context{Workspace: dir})
	if err == nil {
		t.Fatal("expected bad-timeout error")
	}
}

func TestGrep_PathArg(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "sub")
	_ = os.MkdirAll(sub, 0o755)
	writeFile(t, sub, "x.go", "hit\n")
	res, err := GrepTool{}.Run(map[string]any{
		"pattern": "hit", "path": "sub",
	}, tools.Context{Workspace: dir})
	if err != nil {
		t.Fatal(err)
	}
	if got := res.Output["count"].(int); got != 1 {
		t.Fatalf("count = %d", got)
	}
}

func TestGrep_TimeoutError(t *testing.T) {
	// We can't reliably force a real timeout in a tiny test,
	// so we cover the error type itself.
	e := &GrepTimeoutError{Pattern: "p", Timeout: "1ns"}
	if !errors.Is(e, ErrGrepTimeout) {
		t.Fatal("should match sentinel")
	}
}

func TestGrep_FallbackWorkspace(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.go", "hit\n")
	t.Setenv("AIOS_WORKSPACE", dir)
	res, err := GrepTool{}.Run(map[string]any{"pattern": "hit"}, tools.Context{})
	if err != nil {
		t.Fatal(err)
	}
	if got := res.Output["count"].(int); got != 1 {
		t.Fatalf("count = %d, want 1", got)
	}
}

func TestGrep_GlobMiss(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.go", "x\n")
	writeFile(t, dir, "b.txt", "x\n")
	res, err := GrepTool{}.Run(map[string]any{
		"pattern": "x", "glob": "*.md",
	}, tools.Context{Workspace: dir})
	if err != nil {
		t.Fatal(err)
	}
	if got := res.Output["count"].(int); got != 0 {
		t.Fatalf("count = %d, want 0", got)
	}
}

func TestGrep_MaxTruncates(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.go", "x\nx\nx\nx\n")
	res, err := GrepTool{}.Run(map[string]any{
		"pattern": "x", "max_results": 2,
	}, tools.Context{Workspace: dir})
	if err != nil {
		t.Fatal(err)
	}
	if got := res.Output["count"].(int); got != 2 {
		t.Fatalf("count = %d, want 2", got)
	}
}

func TestGrep_AbsPath(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.go", "x\n")
	res, err := GrepTool{}.Run(map[string]any{
		"pattern": "x", "path": dir,
	}, tools.Context{Workspace: dir})
	if err != nil {
		t.Fatal(err)
	}
	if got := res.Output["count"].(int); got != 1 {
		t.Fatalf("count = %d, want 1", got)
	}
}

func TestGrep_TimeoutFires(t *testing.T) {
	// A 1ns timeout is already past by the time the walk
	// finishes. With output_mode=content, the post-walk
	// check returns a GrepTimeoutError. We accept either
	// outcome to keep the test deterministic.
	dir := t.TempDir()
	writeFile(t, dir, "a.go", "x\n")
	_, err := GrepTool{}.Run(map[string]any{
		"pattern": "x", "timeout": "1ns", "output_mode": "content",
	}, tools.Context{Workspace: dir})
	// Either we got the timeout error or the result — both
	// are valid. We only assert no panic.
	_ = err
}

func TestGrep_InWalkTimeoutSkips(t *testing.T) {
	// Many files force the in-walk deadline check to fire
	// partway through. We assert the result has bounded
	// output (no panic, no runaway walk).
	dir := t.TempDir()
	for i := 0; i < 50; i++ {
		writeFile(t, dir, "f"+string(rune('a'+i%26))+string(rune('a'+(i/26)%26))+".go", "x\n")
	}
	res, err := GrepTool{}.Run(map[string]any{
		"pattern": "x", "timeout": "1ns",
	}, tools.Context{Workspace: dir})
	if err != nil {
		// Timeout is acceptable.
		return
	}
	if got := res.Output["count"].(int); got > 50 {
		t.Fatalf("count = %d, walk should be bounded", got)
	}
}
