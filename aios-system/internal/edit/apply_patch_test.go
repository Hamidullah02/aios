package edit

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Hamidullah02/aios/aios-system/internal/tools"
)

func TestParsePatch_SingleFile(t *testing.T) {
	patch := `--- a/a.txt
+++ b/a.txt
@@ -1,3 +1,3 @@
 one
-two
+two!
 three
`
	files, err := parsePatch(patch)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatalf("files = %d", len(files))
	}
	if files[0].OldPath != "a/a.txt" {
		t.Fatalf("OldPath = %q", files[0].OldPath)
	}
	if len(files[0].Hunks) != 1 {
		t.Fatalf("hunks = %d", len(files[0].Hunks))
	}
}

func TestParsePatch_MultiFile(t *testing.T) {
	patch := `--- a/a.txt
+++ b/a.txt
@@ -1 +1 @@
-old
+new
--- a/b.txt
+++ b/b.txt
@@ -1 +1 @@
-1
+2
`
	files, err := parsePatch(patch)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Fatalf("files = %d", len(files))
	}
}

func TestParsePatch_RejectsMalformed(t *testing.T) {
	_, err := parsePatch("not a patch\n")
	if !errors.Is(err, ErrPatchParse) {
		t.Fatalf("err = %v", err)
	}
}

func TestParsePatch_RejectsBadHunk(t *testing.T) {
	_, err := parsePatch("--- a/x\n+++ b/x\n@@ bad header\n")
	if !errors.Is(err, ErrPatchParse) {
		t.Fatalf("err = %v", err)
	}
}

func TestParsePatch_PreservesContent(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.txt", "one\ntwo\nthree\n")
	patch := `--- a/a.txt
+++ b/a.txt
@@ -2,1 +2,1 @@
-two
+TWO
`
	res, err := ApplyPatchTool{}.Run(map[string]any{
		"patch": patch,
	}, tools.Context{Workspace: dir})
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK {
		t.Fatal("not OK")
	}
	if got := readFile(t, filepath.Join(dir, "a.txt")); got != "one\nTWO\nthree\n" {
		t.Fatalf("got %q", got)
	}
}

func TestApplyPatch_ContextMismatch(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.txt", "one\nWRONG\nthree\n")
	patch := `--- a/a.txt
+++ b/a.txt
@@ -2,1 +2,1 @@
-two
+TWO
`
	_, err := ApplyPatchTool{}.Run(map[string]any{
		"patch": patch,
	}, tools.Context{Workspace: dir})
	if !errors.Is(err, ErrPatchContext) {
		t.Fatalf("err = %v", err)
	}
	// File must be unchanged.
	if got := readFile(t, filepath.Join(dir, "a.txt")); !strings.Contains(got, "WRONG") {
		t.Fatalf("file changed: %q", got)
	}
}

func TestApplyPatch_MultiFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.txt", "a1\n")
	writeFile(t, dir, "b.txt", "b1\n")
	patch := `--- a/a.txt
+++ b/a.txt
@@ -1 +1 @@
-a1
+A1
--- a/b.txt
+++ b/b.txt
@@ -1 +1 @@
-b1
+B1
`
	res, err := ApplyPatchTool{}.Run(map[string]any{
		"patch": patch,
	}, tools.Context{Workspace: dir})
	if err != nil {
		t.Fatal(err)
	}
	if got := res.Output["count"].(int); got != 2 {
		t.Fatalf("count = %d", got)
	}
	if got := readFile(t, filepath.Join(dir, "a.txt")); got != "A1\n" {
		t.Fatalf("a.txt = %q", got)
	}
	if got := readFile(t, filepath.Join(dir, "b.txt")); got != "B1\n" {
		t.Fatalf("b.txt = %q", got)
	}
}

func TestApplyPatch_MissingPatch(t *testing.T) {
	_, err := ApplyPatchTool{}.Run(map[string]any{}, tools.Context{Workspace: t.TempDir()})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestApplyPatch_AddLine(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.txt", "a\nb\n")
	patch := `--- a/a.txt
+++ b/a.txt
@@ -1,2 +1,3 @@
 a
+new
 b
`
	_, err := ApplyPatchTool{}.Run(map[string]any{
		"patch": patch,
	}, tools.Context{Workspace: dir})
	if err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, filepath.Join(dir, "a.txt")); got != "a\nnew\nb\n" {
		t.Fatalf("got %q", got)
	}
}

func TestApplyPatch_DeleteLine(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.txt", "a\nb\nc\n")
	patch := `--- a/a.txt
+++ b/a.txt
@@ -1,3 +1,2 @@
 a
-b
 c
`
	_, err := ApplyPatchTool{}.Run(map[string]any{
		"patch": patch,
	}, tools.Context{Workspace: dir})
	if err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, filepath.Join(dir, "a.txt")); got != "a\nc\n" {
		t.Fatalf("got %q", got)
	}
}

func TestApplyPatch_FallbackWorkspace(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.txt", "x\n")
	t.Setenv("AIOS_WORKSPACE", dir)
	patch := `--- a/a.txt
+++ b/a.txt
@@ -1 +1 @@
-x
+y
`
	_, err := ApplyPatchTool{}.Run(map[string]any{"patch": patch}, tools.Context{})
	if err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, filepath.Join(dir, "a.txt")); got != "y\n" {
		t.Fatalf("got %q", got)
	}
}

func TestParseRange(t *testing.T) {
	r, err := parseRange("-1,3")
	if err != nil {
		t.Fatal(err)
	}
	if r.start != 1 || r.length != 3 {
		t.Fatalf("range = %+v", r)
	}
	r, err = parseRange("+5")
	if err != nil {
		t.Fatal(err)
	}
	if r.start != 5 || r.length != 1 {
		t.Fatalf("range = %+v", r)
	}
}

func TestApplyPatch_NoChange(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.txt", "a\nb\nc\n")
	patch := `--- a/a.txt
+++ b/a.txt
@@ -1,3 +1,3 @@
 a
-b
+b
 c
`
	_, err := ApplyPatchTool{}.Run(map[string]any{"patch": patch},
		tools.Context{Workspace: dir})
	if err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, filepath.Join(dir, "a.txt")); got != "a\nb\nc\n" {
		t.Fatalf("got %q", got)
	}
}

func TestApplyPatch_BadPathOutsideWorkspace(t *testing.T) {
	dir := t.TempDir()
	patch := `--- a/a.txt
+++ b/a.txt
@@ -1 +1 @@
-x
+y
`
	_, err := ApplyPatchTool{}.Run(map[string]any{"patch": patch},
		tools.Context{Workspace: dir})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestApplyPatch_RejectsPlusBeforeMinus(t *testing.T) {
	_, err := parsePatch("+++ b/x\n--- a/x\n@@ -1 +1 @@\n")
	if !errors.Is(err, ErrPatchParse) {
		t.Fatalf("err = %v", err)
	}
}

func TestApplyPatch_OutOfOrderHunk(t *testing.T) {
	// Two hunks but the second one points to a line before
	// the first — must reject with a context mismatch.
	dir := t.TempDir()
	writeFile(t, dir, "a.txt", "a\nb\nc\nd\n")
	patch := `--- a/a.txt
+++ b/a.txt
@@ -3,1 +3,1 @@
-c
+C
@@ -1,1 +1,1 @@
-a
+A
`
	_, err := ApplyPatchTool{}.Run(map[string]any{"patch": patch},
		tools.Context{Workspace: dir})
	if !errors.Is(err, ErrPatchContext) {
		t.Fatalf("err = %v", err)
	}
}

func TestApplyPatch_ReadFailure(t *testing.T) {
	dir := t.TempDir()
	patch := `--- a/missing.txt
+++ b/missing.txt
@@ -1 +1 @@
-x
+y
`
	_, err := ApplyPatchTool{}.Run(map[string]any{"patch": patch},
		tools.Context{Workspace: dir})
	if err == nil {
		t.Fatal("expected read error")
	}
}

func TestParseHunkHeader_Malformed(t *testing.T) {
	_, err := parseHunkHeader("@@ not a header @@")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseHunkHeader_BadRange(t *testing.T) {
	_, err := parseHunkHeader("@@ -abc +1 @@")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLineAt_OutOfBounds(t *testing.T) {
	if lineAt([]string{"a"}, 5) != "" {
		t.Fatal("expected empty")
	}
	if lineAt([]string{"a"}, -1) != "" {
		t.Fatal("expected empty")
	}
}
