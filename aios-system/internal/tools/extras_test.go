package tools

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/sadi/aios-system/internal/core/audit"
	"github.com/sadi/aios-system/internal/sandbox"
)

func TestPathNotAllowedError_ErrorAndIs(t *testing.T) {
	e := &PathNotAllowedError{Path: "/etc/passwd", Workspace: "C:\\proj"}
	if !strings.Contains(e.Error(), "/etc/passwd") {
		t.Fatal("error should include path")
	}
	if !errors.Is(e, ErrPathNotAllowed) {
		t.Fatal("errors.Is should match sentinel")
	}
	if got := e.Unwrap(); got != ErrPathNotAllowed {
		t.Fatalf("unwrap = %v", got)
	}
}

func TestToolNotFoundError_ErrorAndIs(t *testing.T) {
	e := &ToolNotFoundError{Name: "x"}
	if !strings.Contains(e.Error(), "x") {
		t.Fatal("error should include name")
	}
	if !errors.Is(e, ErrToolNotFound) {
		t.Fatal("errors.Is should match sentinel")
	}
	if got := e.Unwrap(); got != ErrToolNotFound {
		t.Fatalf("unwrap = %v", got)
	}
}

func TestToolTimeoutError_ErrorAndIs(t *testing.T) {
	e := &ToolTimeoutError{Tool: "ok", Timeout: "30s"}
	if !errors.Is(e, ErrToolTimeout) {
		t.Fatal("errors.Is should match sentinel")
	}
	if got := e.Unwrap(); got != ErrToolTimeout {
		t.Fatalf("unwrap = %v", got)
	}
	if !strings.Contains(e.Error(), "ok") || !strings.Contains(e.Error(), "30s") {
		t.Fatalf("error = %q", e.Error())
	}
}

func TestWorkspace_NoEnv_ReturnsAbs(t *testing.T) {
	t.Setenv(EnvWorkspace, "")
	ws, err := Workspace()
	if err != nil {
		t.Fatal(err)
	}
	abs, _ := filepath.Abs(".")
	if ws != abs {
		t.Fatalf("got %q, want %q", ws, abs)
	}
}

func TestResolvePath_EmptyArgs(t *testing.T) {
	dir := t.TempDir()
	if _, err := ResolvePath(dir, ""); err == nil {
		t.Fatal("expected error for empty path")
	}
	if _, err := ResolvePath("", "x"); err == nil {
		t.Fatal("expected error for empty workspace")
	}
}

func TestResolvePath_AlreadyAbs_Inside(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "a.txt")
	got, err := ResolvePath(dir, p)
	if err != nil {
		t.Fatal(err)
	}
	if got != p {
		t.Fatalf("got %q, want %q", got, p)
	}
}

func TestIsUnder_RootEqual(t *testing.T) {
	if !isUnder("C:\\foo", "C:\\foo") {
		t.Fatal("root should equal root")
	}
}

func TestIsUnder_PrefixCollision(t *testing.T) {
	if isUnder("C:\\foobar", "C:\\foo") {
		t.Fatal("/foobar should NOT be under /foo (prefix collision)")
	}
}

func TestToInt_Variants(t *testing.T) {
	cases := map[string]int{
		"int":     5,
		"int64":   6,
		"float64": 7,
		"string":  8,
		"empty":   0,
		"unknown": 0,
	}
	for k, want := range cases {
		var v any
		switch k {
		case "int":
			v = int(5)
		case "int64":
			v = int64(6)
		case "float64":
			v = float64(7.0)
		case "string":
			v = "8"
		case "empty":
			v = ""
		case "unknown":
			v = []int{1, 2}
		}
		got, err := toInt(v)
		if err != nil && k == "string" {
			// string-only error path: we test the error case below
		}
		if k == "unknown" {
			if got != 0 {
				t.Fatalf("%s: got %d", k, got)
			}
			continue
		}
		if got != want {
			t.Fatalf("%s: got %d, want %d", k, got, want)
		}
	}
	if _, err := toInt("not-a-number"); err == nil {
		t.Fatal("expected error for bad string")
	}
}

func TestRiskFromArgs(t *testing.T) {
	cases := []struct {
		args map[string]any
		want int
	}{
		{nil, 1},
		{map[string]any{"risk_tier": 2}, 2},
		{map[string]any{"risk_tier": -1}, 0},
		{map[string]any{"risk_tier": 99}, 3},
		{map[string]any{"risk_tier": "bad"}, 1},
	}
	for _, c := range cases {
		got := riskFromArgs(c.args)
		if got != c.want {
			t.Fatalf("args=%v: got %d, want %d", c.args, got, c.want)
		}
	}
}

func TestReadFile_FallbackWorkspace(t *testing.T) {
	// Run with empty Context.Workspace and the env unset, but
	// the tool needs an existing file. We set the env to a
	// known dir to keep this deterministic.
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv(EnvWorkspace, dir)
	r := ReadFileTool{}
	res, err := r.Run(map[string]any{"path": "a.txt"}, Context{})
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK {
		t.Fatalf("OK = false")
	}
}

func TestReadFile_OffsetBeyondEnd(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "f.txt"), []byte("a\nb"), 0o644); err != nil {
		t.Fatal(err)
	}
	r := ReadFileTool{}
	res, err := r.Run(map[string]any{"path": "f.txt", "offset": 100}, Context{Workspace: dir})
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK {
		t.Fatal("OK = false")
	}
}

func TestWriteFile_EmptyContent(t *testing.T) {
	dir := t.TempDir()
	w := WriteFileTool{}
	res, err := w.Run(map[string]any{"path": "empty.txt", "content": ""}, Context{Workspace: dir})
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK {
		t.Fatal("OK = false")
	}
	got, _ := os.ReadFile(filepath.Join(dir, "empty.txt"))
	if string(got) != "" {
		t.Fatalf("got %q", got)
	}
}

func TestWriteFile_MissingContent(t *testing.T) {
	dir := t.TempDir()
	w := WriteFileTool{}
	_, err := w.Run(map[string]any{"path": "x.txt"}, Context{Workspace: dir})
	if err == nil {
		t.Fatal("expected error for missing content key")
	}
}

func TestWriteFile_MissingPath(t *testing.T) {
	w := WriteFileTool{}
	_, err := w.Run(map[string]any{"content": "x"}, Context{Workspace: t.TempDir()})
	if err == nil {
		t.Fatal("expected error for missing path")
	}
}

func TestWriteFile_FallbackWorkspace(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(EnvWorkspace, dir)
	w := WriteFileTool{}
	_, err := w.Run(map[string]any{"path": "a.txt", "content": "x"}, Context{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "a.txt")); err != nil {
		t.Fatal(err)
	}
}

func TestWriteFile_Mode(t *testing.T) {
	dir := t.TempDir()
	w := WriteFileTool{}
	if _, err := w.Run(map[string]any{
		"path":    "x.txt",
		"content": "x",
		"mode":    0o600,
	}, Context{Workspace: dir}); err != nil {
		t.Fatal(err)
	}
	// On Windows, os.Chmod only honors the read-only bit;
	// we still verify the file exists and the call path was
	// taken without error.
	info, err := os.Stat(filepath.Join(dir, "x.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if !info.Mode().IsRegular() {
		t.Fatalf("not a regular file: %v", info.Mode())
	}
}

func TestSearchFiles_IncludeMiss(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "a.go"), []byte("x"), 0o644)
	s := SearchFilesTool{}
	res, err := s.Run(map[string]any{
		"pattern": "*.go",
		"include": "nope/*.go",
	}, Context{Workspace: dir})
	if err != nil {
		t.Fatal(err)
	}
	if got := res.Output["matches"].([]string); len(got) != 0 {
		t.Fatalf("matches = %v", got)
	}
}

func TestSearchFiles_IncludeAndMax(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "cmd"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, n := range []string{"a.go", "b.go", "c.go"} {
		_ = os.WriteFile(filepath.Join(dir, "cmd", n), []byte("x"), 0o644)
	}
	_ = os.WriteFile(filepath.Join(dir, "skip.go"), []byte("x"), 0o644)
	s := SearchFilesTool{}
	res, err := s.Run(map[string]any{
		"pattern": "*.go",
		"include": "cmd/*.go",
		"max":     2,
	}, Context{Workspace: dir})
	if err != nil {
		t.Fatal(err)
	}
	if got := res.Output["matches"].([]string); len(got) != 2 {
		t.Fatalf("matches = %v", got)
	}
}

func TestSearchFiles_NoWorkspace(t *testing.T) {
	// The env var is set to a temp dir so Workspace() works
	// even when Context.Workspace is empty.
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "x.txt"), []byte("y"), 0o644)
	t.Setenv(EnvWorkspace, dir)
	s := SearchFilesTool{}
	res, err := s.Run(map[string]any{"pattern": "*.txt"}, Context{})
	if err != nil {
		t.Fatal(err)
	}
	if got := res.Output["matches"].([]string); len(got) != 1 {
		t.Fatalf("matches = %v", got)
	}
}

func TestSearchFiles_MissingPattern(t *testing.T) {
	s := SearchFilesTool{}
	_, err := s.Run(map[string]any{}, Context{Workspace: t.TempDir()})
	if err == nil {
		t.Fatal("expected error for missing pattern")
	}
}

func TestSearchFiles_MaxLimit(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < 5; i++ {
		_ = os.WriteFile(filepath.Join(dir, "f"+string(rune('a'+i))+".go"), []byte("x"), 0o644)
	}
	s := SearchFilesTool{}
	res, err := s.Run(map[string]any{
		"pattern": "*.go",
		"max":     2,
	}, Context{Workspace: dir})
	if err != nil {
		t.Fatal(err)
	}
	if got := res.Output["matches"].([]string); len(got) != 2 {
		t.Fatalf("matches = %d, want 2", len(got))
	}
}

func TestRunShell_NoRunner_UsesDefault(t *testing.T) {
	// A tool with no runner should still run; the default
	// runner is the real SubprocessRunner, so we use a
	// command that always exists.
	r := RunShellTool{}
	res, err := r.Run(map[string]any{"command": []string{"go", "version"}}, Context{})
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK {
		t.Fatalf("OK = false: %s", res.Summary)
	}
}

// TestRunShell_CommandAsString pins that a plan can express a
// command as a plain string.
//
// The plan format is a list of `key=value` strings, so
// `command=ls -la` is the natural way for a model to write a
// command, and it is the only way to write one that has not
// been JSON-encoded. The tool accepted only []string, so that
// plan failed with "command is required (argv)" -- an error
// that named neither the problem nor the way out of it.
func TestRunShell_CommandAsString(t *testing.T) {
	var got []string
	r := RunShellTool{Runner: &recordingRunner{onRun: func(opts sandbox.RunOptions) {
		got = opts.Command
	}}}
	res, err := r.Run(map[string]any{"command": "ls -la"}, Context{})
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK {
		t.Fatalf("OK = false: %s", res.Summary)
	}
	want := []string{"ls", "-la"}
	if len(got) != len(want) {
		t.Fatalf("command = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("command = %v, want %v", got, want)
		}
	}
}

// TestRunShell_CommandAsAnySlice pins that a command that came
// from a JSON decode works.
//
// A []any is what encoding/json produces for an array, so a
// plan whose inputs are decoded from JSON hands the tool a
// []any, not a []string.
func TestRunShell_CommandAsAnySlice(t *testing.T) {
	var got []string
	r := RunShellTool{Runner: &recordingRunner{onRun: func(opts sandbox.RunOptions) {
		got = opts.Command
	}}}
	res, err := r.Run(map[string]any{"command": []any{"ls", "-la"}}, Context{})
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK {
		t.Fatalf("OK = false: %s", res.Summary)
	}
	if len(got) != 2 || got[0] != "ls" || got[1] != "-la" {
		t.Fatalf("command = %v, want [ls -la]", got)
	}
}

// TestRunShell_CommandAsJSONArrayString pins that a plan can
// write a command as a JSON array in one string.
//
// The plan format is a list of `key=value` strings, so a plan
// writes `command=["ls", "-la"]` to pass a list. Splitting that
// on whitespace would tear it into `["ls",` and `-la"]` at the
// space after the comma, and the runner would try to execute a
// program called `["ls",`.
func TestRunShell_CommandAsJSONArrayString(t *testing.T) {
	var got []string
	r := RunShellTool{Runner: &recordingRunner{onRun: func(opts sandbox.RunOptions) {
		got = opts.Command
	}}}
	res, err := r.Run(map[string]any{"command": `["ls", "-la"]`}, Context{})
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK {
		t.Fatalf("OK = false: %s", res.Summary)
	}
	if len(got) != 2 || got[0] != "ls" || got[1] != "-la" {
		t.Fatalf("command = %v, want [ls -la]", got)
	}
}

// TestRunShell_NoCommandAtAll pins that a plan with no command
// at all is still an error.
func TestRunShell_NoCommandAtAll(t *testing.T) {
	r := RunShellTool{Runner: &recordingRunner{onRun: func(opts sandbox.RunOptions) {
		t.Error("the runner must not be called")
	}}}
	if _, err := r.Run(map[string]any{}, Context{}); err == nil {
		t.Fatal("expected an error for a missing command")
	}
}

func TestRunShell_EnvMapStringAny(t *testing.T) {
	// The "env" arg should accept map[string]any (YAML
	// unmarshals into that).
	called := false
	// Reuse the fakeRunner via the RunOptions.Env path
	// by directly setting up a runner that records the env.
	rec := &recordingRunner{onRun: func(opts sandbox.RunOptions) {
		called = true
		if opts.Env["X"] != "y" {
			t.Errorf("env X = %q, want y", opts.Env["X"])
		}
	}}
	r := RunShellTool{Runner: rec}
	_, err := r.Run(map[string]any{
		"command": []string{"echo"},
		"env":     map[string]any{"X": "y"},
	}, Context{})
	if err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("runner was not called")
	}
}

func TestRunShell_NonTimeoutError(t *testing.T) {
	r := RunShellTool{Runner: &fakeRunner{err: errors.New("other")}}
	_, err := r.Run(map[string]any{"command": []string{"x"}}, Context{})
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, ErrToolTimeout) {
		t.Fatal("err should not be ErrToolTimeout")
	}
}

func TestBus_AuditWriteFailure_ReturnsError(t *testing.T) {
	// The bus must report audit write failures. We point
	// the audit log at a closed DB so writes fail.
	log, err := audit.Open(audit.Config{Path: ":memory:"})
	if err != nil {
		t.Fatal(err)
	}
	_ = log.Close() // close so subsequent writes fail
	cnt := &atomic.Int32{}
	bus := New(Config{
		Tools: map[string]Tool{"ok": countingTool{count: cnt}},
		Audit: log,
	})
	_, err = bus.Dispatch("ok", nil, Context{Actor: "a", TaskID: "t1"})
	if err == nil {
		t.Fatal("expected audit-write error")
	}
}

func TestRunShell_EnvMapStringString(t *testing.T) {
	var got map[string]string
	rec := &recordingRunner{onRun: func(opts sandbox.RunOptions) {
		got = opts.Env
	}}
	r := RunShellTool{Runner: rec}
	_, err := r.Run(map[string]any{
		"command": []string{"echo"},
		"env":     map[string]string{"A": "b"},
	}, Context{})
	if err != nil {
		t.Fatal(err)
	}
	if got["A"] != "b" {
		t.Fatalf("env A = %q", got["A"])
	}
}

func TestReadFile_OutsideWithEnv(t *testing.T) {
	// Coverage: ResolvePath failure path inside Run.
	dir := t.TempDir()
	other := t.TempDir()
	t.Setenv(EnvWorkspace, dir)
	r := ReadFileTool{}
	_, err := r.Run(map[string]any{"path": filepath.Join(other, "x")}, Context{})
	if err == nil {
		t.Fatal("expected path-rejection error")
	}
}
