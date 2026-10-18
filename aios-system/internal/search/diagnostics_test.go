package search

import (
	"errors"
	"testing"

	"github.com/sadi/aios-system/internal/sandbox"
	"github.com/sadi/aios-system/internal/tools"
)

type fakeRunner struct {
	res sandbox.SandboxResult
	err error
}

func (f *fakeRunner) Run(opts sandbox.RunOptions) (sandbox.SandboxResult, error) {
	return f.res, f.err
}

// recordingRunner records the last Run call's argv so tests
// can assert on command construction.
type recordingRunner struct {
	res      sandbox.SandboxResult
	cmdArgv  []string
	lastOpts sandbox.RunOptions
}

func (r *recordingRunner) Run(opts sandbox.RunOptions) (sandbox.SandboxResult, error) {
	r.lastOpts = opts
	r.cmdArgv = append([]string(nil), opts.Command...)
	return r.res, nil
}

// alwaysFoundLookPath is a test override that says every
// binary exists. We use it to bypass the PATH check.
func alwaysFoundLookPath(name string) (string, error) { return name, nil }

func newDiag(checker string, res sandbox.SandboxResult) DiagnosticsTool {
	return DiagnosticsTool{Runner: &fakeRunner{res: res}, LookPath: alwaysFoundLookPath}
}

func TestDiagnostics_GoVetParse(t *testing.T) {
	d := newDiag("go vet", sandbox.SandboxResult{
		ExitCode: 1,
		Stderr:   "a.go:10:5: suspicious printf verb %s",
	})
	res, err := d.Run(map[string]any{"checker": "go vet"}, tools.Context{Workspace: "."})
	if err != nil {
		t.Fatal(err)
	}
	diags, _ := res.Output["diagnostics"].([]Diagnostic)
	if len(diags) != 1 {
		t.Fatalf("count = %d, want 1", len(diags))
	}
	if diags[0].Line != 10 || diags[0].Column != 5 {
		t.Fatalf("loc = %d:%d", diags[0].Line, diags[0].Column)
	}
}

func TestDiagnostics_GofmtParse(t *testing.T) {
	d := newDiag("gofmt", sandbox.SandboxResult{ExitCode: 0, Stdout: "a.go\nb.go\n"})
	res, err := d.Run(map[string]any{"checker": "gofmt"}, tools.Context{Workspace: "."})
	if err != nil {
		t.Fatal(err)
	}
	diags, _ := res.Output["diagnostics"].([]Diagnostic)
	if len(diags) != 2 || diags[0].Code != "gofmt" {
		t.Fatalf("diags = %+v", diags)
	}
}

func TestDiagnostics_PylintParse(t *testing.T) {
	jsonOut := `[{"path":"x.py","line":3,"column":7,"type":"error","symbol":"E1","message":"bad"}]`
	d := newDiag("pylint", sandbox.SandboxResult{ExitCode: 4, Stdout: jsonOut})
	res, _ := d.Run(map[string]any{"checker": "pylint"}, tools.Context{Workspace: "."})
	diags, _ := res.Output["diagnostics"].([]Diagnostic)
	if len(diags) != 1 || diags[0].Severity != "error" || diags[0].Code != "E1" {
		t.Fatalf("diags = %+v", diags)
	}
}

func TestDiagnostics_ESLintParse(t *testing.T) {
	jsonOut := `{"results":[{"filePath":"x.js","messages":[{"line":1,"column":1,"severity":2,"ruleId":"no-unused","message":"x"}]}]}`
	d := newDiag("eslint", sandbox.SandboxResult{ExitCode: 1, Stdout: jsonOut})
	res, _ := d.Run(map[string]any{"checker": "eslint"}, tools.Context{Workspace: "."})
	diags, _ := res.Output["diagnostics"].([]Diagnostic)
	if len(diags) != 1 || diags[0].Severity != "error" {
		t.Fatalf("diags = %+v", diags)
	}
}

func TestDiagnostics_CheckerNotFound(t *testing.T) {
	d := DiagnosticsTool{LookPath: func(string) (string, error) {
		return "", errors.New("missing")
	}}
	_, err := d.Run(map[string]any{"checker": "pylint"},
		tools.Context{Workspace: "."})
	if !errors.Is(err, ErrCheckerNotFound) {
		t.Fatalf("err = %v, want ErrCheckerNotFound", err)
	}
}

func TestDiagnostics_UnknownChecker(t *testing.T) {
	d := DiagnosticsTool{Runner: &fakeRunner{}}
	_, err := d.Run(map[string]any{"checker": "bogus-checker"},
		tools.Context{Workspace: "."})
	if err == nil {
		t.Fatal("expected unknown-checker error")
	}
}

func TestDiagnostics_RunnerError(t *testing.T) {
	d := DiagnosticsTool{Runner: &fakeRunner{err: errors.New("boom")}}
	_, err := d.Run(map[string]any{"checker": "go vet"}, tools.Context{Workspace: "."})
	if err == nil {
		t.Fatal("expected runner error")
	}
}

func TestDiagnostics_GoVetNoOutput(t *testing.T) {
	d := newDiag("go vet", sandbox.SandboxResult{ExitCode: 0, Stderr: ""})
	res, err := d.Run(map[string]any{"checker": "go vet"}, tools.Context{Workspace: "."})
	if err != nil {
		t.Fatal(err)
	}
	diags, _ := res.Output["diagnostics"].([]Diagnostic)
	if len(diags) != 0 {
		t.Fatalf("diags = %+v", diags)
	}
}

func TestDiagnostics_GofmtEmpty(t *testing.T) {
	d := newDiag("gofmt", sandbox.SandboxResult{ExitCode: 0, Stdout: ""})
	res, err := d.Run(map[string]any{"checker": "gofmt"}, tools.Context{Workspace: "."})
	if err != nil {
		t.Fatal(err)
	}
	diags, _ := res.Output["diagnostics"].([]Diagnostic)
	if len(diags) != 0 {
		t.Fatalf("diags = %+v", diags)
	}
}

func TestDiagnostics_PylintBadJSON(t *testing.T) {
	d := newDiag("pylint", sandbox.SandboxResult{ExitCode: 0, Stdout: "not json"})
	res, _ := d.Run(map[string]any{"checker": "pylint"}, tools.Context{Workspace: "."})
	diags, _ := res.Output["diagnostics"].([]Diagnostic)
	if len(diags) != 0 {
		t.Fatalf("diags = %+v", diags)
	}
}

func TestDiagnostics_ESLintBadJSON(t *testing.T) {
	d := newDiag("eslint", sandbox.SandboxResult{ExitCode: 0, Stdout: "not json"})
	res, _ := d.Run(map[string]any{"checker": "eslint"}, tools.Context{Workspace: "."})
	diags, _ := res.Output["diagnostics"].([]Diagnostic)
	if len(diags) != 0 {
		t.Fatalf("diags = %+v", diags)
	}
}

func TestDiagnostics_ESLintWarningSeverity(t *testing.T) {
	jsonOut := `{"results":[{"filePath":"x.js","messages":[{"line":2,"column":3,"severity":1,"ruleId":"r","message":"m"}]}]}`
	d := newDiag("eslint", sandbox.SandboxResult{ExitCode: 0, Stdout: jsonOut})
	res, _ := d.Run(map[string]any{"checker": "eslint"}, tools.Context{Workspace: "."})
	diags, _ := res.Output["diagnostics"].([]Diagnostic)
	if len(diags) != 1 || diags[0].Severity != "warning" {
		t.Fatalf("diags = %+v", diags)
	}
}

func TestParseGoVet_NoMatch(t *testing.T) {
	if got := parseGoVet("random line\n", ""); len(got) != 0 {
		t.Fatalf("got %+v", got)
	}
}

func TestParseAtoi_NonDigit(t *testing.T) {
	// atoi returns 0 on the first non-digit (lenient) and 0
	// on the empty string.
	if got := atoi("12x"); got != 0 {
		t.Fatalf("atoi = %d, want 0", got)
	}
	if got := atoi(""); got != 0 {
		t.Fatalf("atoi empty = %d", got)
	}
}

func TestRelPath_EmptyBase(t *testing.T) {
	if got := relPath("/foo/bar", ""); got != "/foo/bar" {
		t.Fatalf("relPath = %q", got)
	}
}

func TestRelPath_RelFallback(t *testing.T) {
	// Rel returns an error for an absolute base that doesn't
	// contain the target — the function should fall back to
	// the original path.
	got := relPath("/foo", "C:\\bar")
	if got != "/foo" {
		t.Fatalf("relPath = %q, want /foo", got)
	}
}

func TestDiagnostics_DefaultChecker(t *testing.T) {
	// No checker key — defaults to "go vet".
	d := newDiag("go vet", sandbox.SandboxResult{ExitCode: 0})
	res, err := d.Run(map[string]any{}, tools.Context{Workspace: "."})
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK {
		t.Fatalf("OK = false: %s", res.Summary)
	}
}

func TestDiagnostics_PathOverride(t *testing.T) {
	d := newDiag("go vet", sandbox.SandboxResult{ExitCode: 0})
	rec := &recordingRunner{}
	rec.res = sandbox.SandboxResult{ExitCode: 0}
	d.Runner = rec
	_, _ = d.Run(map[string]any{
		"checker": "go vet", "path": "subdir",
	}, tools.Context{Workspace: "."})
	// argv should be [go vet subdir]
	if len(rec.cmdArgv) < 3 || rec.cmdArgv[2] != "subdir" {
		t.Fatalf("argv = %v", rec.cmdArgv)
	}
}

func TestDiagnostics_ExtraArgs(t *testing.T) {
	d := newDiag("go vet", sandbox.SandboxResult{ExitCode: 0})
	rec := &recordingRunner{}
	rec.res = sandbox.SandboxResult{ExitCode: 0}
	d.Runner = rec
	_, _ = d.Run(map[string]any{
		"checker": "go vet", "extra_args": []string{"-all"},
	}, tools.Context{Workspace: "."})
	// Expect "-all" appended to argv.
	found := false
	for _, a := range rec.cmdArgv {
		if a == "-all" {
			found = true
		}
	}
	if !found {
		t.Fatalf("argv missing -all: %v", rec.cmdArgv)
	}
}

func TestDiagnostics_FallbackWorkspace(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("AIOS_WORKSPACE", dir)
	d := newDiag("go vet", sandbox.SandboxResult{ExitCode: 0})
	res, err := d.Run(map[string]any{"checker": "go vet"}, tools.Context{})
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK {
		t.Fatalf("OK = false")
	}
}
