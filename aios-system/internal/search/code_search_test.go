package search

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Hamidullah02/aios/aios-system/internal/tools"
)

func TestCodeSearch_GoFunction(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.go", `package x
func Hello() {}
func other() {}
`)
	res, err := CodeSearchTool{}.Run(map[string]any{
		"symbol": "Hello", "kind": "function",
	}, tools.Context{Workspace: dir})
	if err != nil {
		t.Fatal(err)
	}
	if got := res.Output["count"].(int); got != 1 {
		t.Fatalf("count = %d", got)
	}
}

func TestCodeSearch_GoMethod(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.go", `package x
type S struct{}
func (S) Foo() {}
func Bar() {}
`)
	res, err := CodeSearchTool{}.Run(map[string]any{
		"symbol": "Foo", "kind": "method",
	}, tools.Context{Workspace: dir})
	if err != nil {
		t.Fatal(err)
	}
	if got := res.Output["count"].(int); got != 1 {
		t.Fatalf("count = %d", got)
	}
}

func TestCodeSearch_PythonClass(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "m.py", `def helper():
    pass
class Widget:
    pass
`)
	res, err := CodeSearchTool{}.Run(map[string]any{
		"symbol": "Widget", "kind": "class", "lang": "python",
	}, tools.Context{Workspace: dir})
	if err != nil {
		t.Fatal(err)
	}
	if got := res.Output["count"].(int); got != 1 {
		t.Fatalf("count = %d", got)
	}
}

func TestCodeSearch_UnknownLang(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.go", "func Foo() {}\n")
	_, err := CodeSearchTool{}.Run(map[string]any{
		"symbol": "Foo", "lang": "kotlin",
	}, tools.Context{Workspace: dir})
	if err == nil {
		t.Fatal("expected unsupported-language error")
	}
	if !errors.Is(err, ErrUnsupportedLanguage) {
		t.Fatalf("err = %v, want ErrUnsupportedLanguage", err)
	}
}

func TestCodeSearch_MissingSymbol(t *testing.T) {
	_, err := CodeSearchTool{}.Run(map[string]any{}, tools.Context{Workspace: t.TempDir()})
	if err == nil {
		t.Fatal("expected missing-symbol error")
	}
}

func TestCodeSearch_RespectsAiosignore(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.go", "func Show() {}\n")
	writeFile(t, dir, "b.go", "func Show() {}\n")
	_ = os.WriteFile(filepath.Join(dir, ".aiosignore"), []byte("b.go\n"), 0o644)
	res, err := CodeSearchTool{}.Run(map[string]any{
		"symbol": "Show", "kind": "function",
	}, tools.Context{Workspace: dir})
	if err != nil {
		t.Fatal(err)
	}
	if got := res.Output["count"].(int); got != 1 {
		t.Fatalf("count = %d", got)
	}
}

func TestCodeSearch_AnyKind(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.go", "var Greeting = 1\n")
	res, err := CodeSearchTool{}.Run(map[string]any{
		"symbol": "Greeting",
	}, tools.Context{Workspace: dir})
	if err != nil {
		t.Fatal(err)
	}
	if got := res.Output["count"].(int); got != 1 {
		t.Fatalf("count = %d", got)
	}
}

func TestCodeSearch_PathArg(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "pkg")
	_ = os.MkdirAll(sub, 0o755)
	writeFile(t, sub, "a.go", "func Hi() {}\n")
	res, err := CodeSearchTool{}.Run(map[string]any{
		"symbol": "Hi", "kind": "function", "path": "pkg",
	}, tools.Context{Workspace: dir})
	if err != nil {
		t.Fatal(err)
	}
	if got := res.Output["count"].(int); got != 1 {
		t.Fatalf("count = %d", got)
	}
}

func TestCodeSearch_FallbackWorkspace(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.go", "func Hi() {}\n")
	t.Setenv("AIOS_WORKSPACE", dir)
	res, err := CodeSearchTool{}.Run(map[string]any{
		"symbol": "Hi", "kind": "function",
	}, tools.Context{})
	if err != nil {
		t.Fatal(err)
	}
	if got := res.Output["count"].(int); got != 1 {
		t.Fatalf("count = %d", got)
	}
}

func TestCodeSearch_CppFunction(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.cpp", "void hello(int x) {}\nclass Widget;\n")
	res, err := CodeSearchTool{}.Run(map[string]any{
		"symbol": "hello", "kind": "function", "lang": "cpp",
	}, tools.Context{Workspace: dir})
	if err != nil {
		t.Fatal(err)
	}
	if got := res.Output["count"].(int); got < 1 {
		t.Fatalf("count = %d", got)
	}
}

func TestCodeSearch_JavaClass(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "A.java", "public class Widget { }\n")
	res, err := CodeSearchTool{}.Run(map[string]any{
		"symbol": "Widget", "kind": "class", "lang": "java",
	}, tools.Context{Workspace: dir})
	if err != nil {
		t.Fatal(err)
	}
	if got := res.Output["count"].(int); got != 1 {
		t.Fatalf("count = %d", got)
	}
}

func TestCodeSearch_NoMatchTail(t *testing.T) {
	// The candidate line has the decl keyword but the symbol
	// name does not appear — exercises the "no match" tail
	// path in scanSymbols.
	dir := t.TempDir()
	writeFile(t, dir, "a.go", "func Other() {}\n")
	res, err := CodeSearchTool{}.Run(map[string]any{
		"symbol": "Missing", "kind": "function",
	}, tools.Context{Workspace: dir})
	if err != nil {
		t.Fatal(err)
	}
	if got := res.Output["count"].(int); got != 0 {
		t.Fatalf("count = %d, want 0", got)
	}
}

func TestCodeSearch_TruncateAtMax(t *testing.T) {
	// Two matches, max=1 → walk exits early at >=max.
	// The post-loop truncate (len(hits) > max) is hard to
	// reach because WalkDir exits at >=max. We accept the
	// SkipAll path as the equivalent gate.
	dir := t.TempDir()
	writeFile(t, dir, "a.go", "func Hi() {}\n")
	writeFile(t, dir, "b.go", "func Hi() {}\n")
	res, err := CodeSearchTool{}.Run(map[string]any{
		"symbol": "Hi", "kind": "function", "max": 1,
	}, tools.Context{Workspace: dir})
	if err != nil {
		t.Fatal(err)
	}
	if got := res.Output["count"].(int); got != 1 {
		t.Fatalf("count = %d, want 1", got)
	}
}

func TestCodeSearch_UnknownExtSkipped(t *testing.T) {
	// File extension not in any grammar list is skipped
	// silently. Cover the "langFor returns ''" path.
	dir := t.TempDir()
	writeFile(t, dir, "data.xyz", "func Hi() {}\n")
	writeFile(t, dir, "a.go", "func Hi() {}\n")
	res, err := CodeSearchTool{}.Run(map[string]any{
		"symbol": "Hi", "kind": "function",
	}, tools.Context{Workspace: dir})
	if err != nil {
		t.Fatal(err)
	}
	if got := res.Output["count"].(int); got != 1 {
		t.Fatalf("count = %d, want 1", got)
	}
}

func TestCodeSearch_AbsPath(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.go", "func Hi() {}\n")
	res, err := CodeSearchTool{}.Run(map[string]any{
		"symbol": "Hi", "kind": "function", "path": dir,
	}, tools.Context{Workspace: dir})
	if err != nil {
		t.Fatal(err)
	}
	if got := res.Output["count"].(int); got != 1 {
		t.Fatalf("count = %d, want 1", got)
	}
}

func TestCodeSearch_LangFilterMiss(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.go", "func Hi() {}\n")
	res, err := CodeSearchTool{}.Run(map[string]any{
		"symbol": "Hi", "kind": "function", "lang": "python",
	}, tools.Context{Workspace: dir})
	if err != nil {
		t.Fatal(err)
	}
	if got := res.Output["count"].(int); got != 0 {
		t.Fatalf("count = %d, want 0", got)
	}
}
