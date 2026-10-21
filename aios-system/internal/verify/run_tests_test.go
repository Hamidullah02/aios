package verify

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/sadi/aios-system/internal/sandbox"
	"github.com/sadi/aios-system/internal/tools"
)

// fakeRunner is the sandbox.Runner the
// RunTests tests use. The fake records the
// command and the timeout and returns the
// captured output. The fake also captures
// the runner's exit code.
type fakeRunner struct {
	stdout   string
	stderr   string
	exitCode int
	err      error
	gotCmd   []string
	gotTo    int
}

func (f *fakeRunner) Run(opts sandbox.RunOptions) (sandbox.SandboxResult, error) {
	f.gotCmd = append([]string(nil), opts.Command...)
	f.gotTo = opts.Timeout
	return sandbox.SandboxResult{
		ExitCode:   f.exitCode,
		Stdout:     f.stdout,
		Stderr:     f.stderr,
		DurationMs: 10,
	}, f.err
}

func TestRunTestsTool_Name(t *testing.T) {
	if (RunTestsTool{}).Name() != "run_tests" {
		t.Errorf("name mismatch")
	}
}

func TestDetectRunner(t *testing.T) {
	tests := []struct {
		cmd  []string
		want Runner
	}{
		{[]string{"go", "test", "./..."}, RunnerGoTest},
		{[]string{"/usr/local/bin/go", "test"}, RunnerGoTest},
		{[]string{`C:\bin\go.exe`, "test"}, RunnerGoTest},
		{[]string{"jest"}, RunnerJest},
		{[]string{"npx", "jest"}, RunnerNpmTest},
		{[]string{"mocha"}, RunnerMocha},
		{[]string{"cargo", "test"}, RunnerCargoTest},
		{[]string{"npm", "test"}, RunnerNpmTest},
		{[]string{"pnpm", "test"}, RunnerNpmTest},
		{[]string{"yarn", "test"}, RunnerNpmTest},
		{[]string{"unknown"}, ""},
		{nil, ""},
	}
	for _, tt := range tests {
		if got := DetectRunner(tt.cmd); got != tt.want {
			t.Errorf("cmd=%v: got %q want %q", tt.cmd, got, tt.want)
		}
	}
}

func TestRunner_IsValid(t *testing.T) {
	for _, r := range AllRunners() {
		if !r.IsValid() {
			t.Errorf("expected %q valid", r)
		}
	}
	if Runner("nope").IsValid() {
		t.Errorf("expected unknown invalid")
	}
}

func TestRunTestsTool_Run_MissingCommand(t *testing.T) {
	tool := &RunTestsTool{Sandbox: &fakeRunner{}}
	if _, err := tool.Run(map[string]any{}, tools.Context{}); err == nil {
		t.Errorf("expected error")
	}
}

func TestRunTestsTool_Run_NilSandbox(t *testing.T) {
	tool := &RunTestsTool{}
	_, err := tool.Run(map[string]any{"command": []any{"go", "test"}}, tools.Context{})
	if err == nil {
		t.Errorf("expected error")
	}
}

func TestRunTestsTool_Run_UnknownRunner(t *testing.T) {
	tool := &RunTestsTool{Sandbox: &fakeRunner{}, DefaultRunner: "weird"}
	_, err := tool.Run(map[string]any{
		"command": []any{"go", "test"},
		"runner":  "weird",
	}, tools.Context{})
	if err == nil {
		t.Errorf("expected error")
	}
}

func TestRunTestsTool_Run_GoTest(t *testing.T) {
	fake := &fakeRunner{
		stdout: strings.Join([]string{
			`{"Action":"run","Package":"p","Test":"TestA"}`,
			`{"Action":"output","Package":"p","Test":"TestA","Output":"--- FAIL: TestA\n\texpected\n"}`,
			`{"Action":"fail","Package":"p","Test":"TestA","Elapsed":0.1}`,
			`{"Action":"pass","Package":"p","Test":"TestB","Elapsed":0.1}`,
			`{"Action":"skip","Package":"p","Test":"TestC","Elapsed":0}`,
		}, "\n"),
		exitCode: 1,
	}
	tool := &RunTestsTool{Sandbox: fake, DefaultRunner: RunnerGoTest}
	out, err := tool.Run(map[string]any{
		"command": []any{"go", "test", "-json", "./..."},
		"runner":  "go-test",
	}, tools.Context{})
	if err == nil {
		t.Fatalf("expected failure error")
	}
	if out.OK {
		t.Errorf("expected !OK")
	}
	if got := out.Output["passed"]; got != 1 {
		t.Errorf("passed: %v", got)
	}
	if got := out.Output["failed"]; got != 1 {
		t.Errorf("failed: %v", got)
	}
	if got := out.Output["skipped"]; got != 1 {
		t.Errorf("skipped: %v", got)
	}
}

func TestRunTestsTool_Run_GoTest_AllPass(t *testing.T) {
	fake := &fakeRunner{
		stdout: strings.Join([]string{
			`{"Action":"pass","Package":"p","Test":"TestB","Elapsed":0.1}`,
			`{"Action":"skip","Package":"p","Test":"TestC","Elapsed":0}`,
		}, "\n"),
		exitCode: 0,
	}
	tool := &RunTestsTool{Sandbox: fake, DefaultRunner: RunnerGoTest}
	out, err := tool.Run(map[string]any{
		"command": []any{"go", "test", "-json", "./..."},
		"runner":  "go-test",
	}, tools.Context{})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if !out.OK {
		t.Errorf("expected OK")
	}
}

func TestRunTestsTool_Run_GoTest_DefaultDetection(t *testing.T) {
	fake := &fakeRunner{
		stdout:   `{"Action":"pass","Package":"p","Test":"T"}`,
		exitCode: 0,
	}
	tool := &RunTestsTool{Sandbox: fake}
	_, err := tool.Run(map[string]any{
		"command": []any{"go", "test", "-json", "./..."},
	}, tools.Context{})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
}

func TestRunTestsTool_Run_SingleStringCommand(t *testing.T) {
	fake := &fakeRunner{stdout: `{"Action":"pass","Package":"p","Test":"T"}`}
	tool := &RunTestsTool{Sandbox: fake, DefaultRunner: RunnerGoTest}
	_, err := tool.Run(map[string]any{
		"test_command": "go test ./...",
	}, tools.Context{})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
}

func TestRunTestsTool_Run_NonZeroExit(t *testing.T) {
	fake := &fakeRunner{
		stdout:   `{"Action":"fail","Package":"p","Test":"T","Elapsed":0.1}`,
		exitCode: 1,
	}
	tool := &RunTestsTool{Sandbox: fake, DefaultRunner: RunnerGoTest}
	_, err := tool.Run(map[string]any{
		"command": []any{"go", "test", "-json", "./..."},
	}, tools.Context{})
	if err == nil {
		t.Errorf("expected error on non-zero exit")
	}
	var tre *TestRunnerError
	if !errors.As(err, &tre) {
		t.Errorf("expected TestRunnerError, got %v", err)
	}
}

func TestRunTestsTool_Run_SandboxError(t *testing.T) {
	fake := &fakeRunner{err: errors.New("sandbox boom")}
	tool := &RunTestsTool{Sandbox: fake, DefaultRunner: RunnerGoTest}
	_, err := tool.Run(map[string]any{"command": []any{"go", "test"}}, tools.Context{})
	if err == nil {
		t.Errorf("expected error")
	}
}

func TestRunTestsTool_Run_Jest(t *testing.T) {
	fake := &fakeRunner{
		stdout: `{
  "numPassedTests": 2,
  "numFailedTests": 1,
  "numPendingTests": 1,
  "numTotalTests": 4,
  "testResults": [
    {"fullName": "a", "status": "passed"},
    {"fullName": "b", "status": "passed"},
    {"fullName": "c", "status": "failed", "message": "boom", "consoleLogs": [{"message": "stack"}]},
    {"fullName": "d", "status": "pending"}
  ]
}`,
		exitCode: 1,
	}
	tool := &RunTestsTool{Sandbox: fake, DefaultRunner: RunnerJest}
	out, err := tool.Run(map[string]any{"command": []any{"jest", "--json"}}, tools.Context{})
	if err == nil {
		t.Fatalf("expected failure error")
	}
	if out.OK {
		t.Errorf("expected !OK")
	}
	if got := out.Output["passed"]; got != 2 {
		t.Errorf("passed: %v", got)
	}
	if got := out.Output["failed"]; got != 1 {
		t.Errorf("failed: %v", got)
	}
	if got := out.Output["skipped"]; got != 1 {
		t.Errorf("skipped: %v", got)
	}
}

func TestRunTestsTool_Run_Jest_NonZeroExit(t *testing.T) {
	fake := &fakeRunner{
		stdout:   `{"numPassedTests":1,"numFailedTests":1,"numPendingTests":0,"numTotalTests":2,"testResults":[{"fullName":"a","status":"passed"},{"fullName":"b","status":"failed","message":"x"}]}`,
		exitCode: 1,
	}
	tool := &RunTestsTool{Sandbox: fake, DefaultRunner: RunnerJest}
	out, err := tool.Run(map[string]any{"command": []any{"jest"}}, tools.Context{})
	if err == nil {
		t.Errorf("expected error on failure")
	}
	if out.Output == nil {
		t.Errorf("expected output even on failure")
	}
}

func TestRunTestsTool_Run_NpmTest(t *testing.T) {
	fake := &fakeRunner{stdout: `{"numPassedTests":3,"numFailedTests":0,"numPendingTests":0,"numTotalTests":3}`}
	tool := &RunTestsTool{Sandbox: fake, DefaultRunner: RunnerNpmTest}
	_, err := tool.Run(map[string]any{"command": []any{"npm", "test"}}, tools.Context{})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
}

func TestRunTestsTool_Run_Mocha(t *testing.T) {
	fake := &fakeRunner{
		stdout: `{
  "stats": {"passes": 2, "failures": 1, "pending": 0},
  "passes": [{"title": "a"}, {"title": "b"}],
  "failures": [{"title": "c", "fullTitle": "suite c", "err": {"message": "boom", "stack": "stack"}}]
}`,
		exitCode: 1,
	}
	tool := &RunTestsTool{Sandbox: fake, DefaultRunner: RunnerMocha}
	out, err := tool.Run(map[string]any{"command": []any{"mocha"}}, tools.Context{})
	if err == nil {
		t.Fatalf("expected failure error")
	}
	if got := out.Output["passed"]; got != 2 {
		t.Errorf("passed: %v", got)
	}
	if got := out.Output["failed"]; got != 1 {
		t.Errorf("failed: %v", got)
	}
}

func TestRunTestsTool_Run_Cargo(t *testing.T) {
	lines := []string{
		`{"reason":"test-started","type":"test","name":"t::a","event":{"type":"started"}}`,
		`{"reason":"test-started","type":"test","name":"t::a","event":{"type":"ok"}}`,
		`{"reason":"test-started","type":"test","name":"t::b","event":{"type":"started"}}`,
		`{"reason":"test-started","type":"test","name":"t::b","event":{"type":"failed","message":"x"}}`,
		`{"reason":"test-started","type":"test","name":"t::c","event":{"type":"ignored"}}`,
	}
	fake := &fakeRunner{stdout: strings.Join(lines, "\n"), exitCode: 1}
	tool := &RunTestsTool{Sandbox: fake, DefaultRunner: RunnerCargoTest}
	out, err := tool.Run(map[string]any{"command": []any{"cargo", "test"}}, tools.Context{})
	if err == nil {
		t.Fatalf("expected failure error")
	}
	if got := out.Output["passed"]; got != 1 {
		t.Errorf("passed: %v", got)
	}
	if got := out.Output["failed"]; got != 1 {
		t.Errorf("failed: %v", got)
	}
	if got := out.Output["skipped"]; got != 1 {
		t.Errorf("skipped: %v", got)
	}
}

func TestRunTestsTool_Run_Audit(t *testing.T) {
	tl := newVerifyTestLog(t)
	fake := &fakeRunner{stdout: `{"Action":"pass","Package":"p","Test":"T"}`}
	tool := &RunTestsTool{Sandbox: fake, DefaultRunner: RunnerGoTest, Audit: tl}
	_, _ = tool.Run(map[string]any{"command": []any{"go", "test"}}, tools.Context{TaskID: "task", Actor: "actor"})
	rows, _ := tl.GetByTask("task")
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0].Kind != "run_tests" {
		t.Errorf("kind: %s", rows[0].Kind)
	}
}

func TestRunTestsTool_Run_AuditFailure(t *testing.T) {
	tl := newVerifyTestLog(t)
	fake := &fakeRunner{
		stdout:   `{"Action":"fail","Package":"p","Test":"T"}`,
		exitCode: 1,
	}
	tool := &RunTestsTool{Sandbox: fake, DefaultRunner: RunnerGoTest, Audit: tl}
	_, _ = tool.Run(map[string]any{"command": []any{"go", "test"}}, tools.Context{TaskID: "task"})
	rows, _ := tl.GetByTask("task")
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if !bytes.Contains(rows[0].Payload, []byte(`"runner":"go-test"`)) {
		t.Errorf("expected runner=go-test in payload, got: %s", rows[0].Payload)
	}
}

func TestParseRunnerOutput_NoParser(t *testing.T) {
	if _, err := ParseRunnerOutput(Runner("bogus"), "x", "y", "c"); err == nil {
		t.Errorf("expected error")
	}
}

func TestParseGoTestJSON_Empty(t *testing.T) {
	r, err := ParseGoTestJSON("", "go test")
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if r.Passed != 0 || r.Failed != 0 {
		t.Errorf("expected zeros, got %+v", r)
	}
}

func TestParseGoTestJSON_InvalidLine(t *testing.T) {
	// Invalid JSON lines are skipped, not
	// returned as errors.
	r, err := ParseGoTestJSON("not json\n{\"Action\":\"pass\",\"Test\":\"T\"}", "go test")
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if r.Passed != 1 {
		t.Errorf("passed: %d", r.Passed)
	}
}

func TestParseJestJSON_TruncatedProse(t *testing.T) {
	// Jest sometimes emits a status line before
	// the JSON object; the parser should
	// recover.
	prose := "FAIL suite\n" + `{"numPassedTests":1,"numFailedTests":0,"numPendingTests":0,"numTotalTests":1}`
	r, err := ParseJestJSON(prose, "jest")
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if r.Passed != 1 {
		t.Errorf("passed: %d", r.Passed)
	}
}

func TestParseJestJSON_Invalid(t *testing.T) {
	if _, err := ParseJestJSON("not json at all", "jest"); err == nil {
		t.Errorf("expected error")
	}
}

func TestParseMochaJSON_Invalid(t *testing.T) {
	if _, err := ParseMochaJSON("{not-json", "mocha"); err == nil {
		t.Errorf("expected error")
	}
}

func TestParseCargoJSON_Empty(t *testing.T) {
	r, err := ParseCargoJSON("", "cargo test")
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if r.Passed != 0 {
		t.Errorf("expected zero passed")
	}
}

func TestParseCargoJSON_InvalidLine(t *testing.T) {
	r, err := ParseCargoJSON("not json\n{\"reason\":\"x\",\"type\":\"test\",\"name\":\"a\",\"event\":{\"type\":\"ok\"}}", "cargo test")
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if r.Passed != 1 {
		t.Errorf("passed: %d", r.Passed)
	}
}

func TestResolveCommand(t *testing.T) {
	c, err := resolveCommand(map[string]any{"command": []any{"go", "test"}})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if len(c) != 2 {
		t.Errorf("len: %d", len(c))
	}
	if _, err := resolveCommand(map[string]any{"command": []any{"go", 1}}); err == nil {
		t.Errorf("expected non-string error")
	}
	// v0.37.0: command may also be a single string
	// (the bus worker passes "command=value" form,
	// which becomes args["command"]="value").
	if c, err := resolveCommand(map[string]any{"command": "go test -json"}); err != nil {
		t.Fatalf("string command: %v", err)
	} else if len(c) != 3 || c[0] != "go" {
		t.Errorf("string command split: %v", c)
	}
	if _, err := resolveCommand(nil); err == nil {
		t.Errorf("expected missing error")
	}
}

func TestStringArgAndIntArg(t *testing.T) {
	if got := stringArg(map[string]any{"k": "v"}, "k"); got != "v" {
		t.Errorf("got %q", got)
	}
	if got := stringArg(map[string]any{"k": 1}, "k"); got != "" {
		t.Errorf("got %q", got)
	}
	if got := intArg(map[string]any{"k": 1}, "k"); got != 1 {
		t.Errorf("got %d", got)
	}
	if got := intArg(map[string]any{"k": int64(7)}, "k"); got != 7 {
		t.Errorf("got %d", got)
	}
	if got := intArg(map[string]any{"k": float64(3)}, "k"); got != 3 {
		t.Errorf("got %d", got)
	}
	if got := intArg(map[string]any{"k": "x"}, "k"); got != 0 {
		t.Errorf("got %d", got)
	}
}
