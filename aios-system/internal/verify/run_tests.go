package verify

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/Hamidullah02/aios/aios-system/internal/core/audit"
	"github.com/Hamidullah02/aios/aios-system/internal/sandbox"
	"github.com/Hamidullah02/aios/aios-system/internal/tools"
)

// Runner is the typed enum for the supported
// test runners (REQ-VRF-RNT-004). Each
// constant names a per-runner parser the
// RunTests tool dispatches to.
type Runner string

const (
	// RunnerGoTest is the `go test -json` parser.
	RunnerGoTest Runner = "go-test"
	// RunnerJest is the `jest --json` parser.
	RunnerJest Runner = "jest"
	// RunnerMocha is the `mocha --reporter json`
	// parser.
	RunnerMocha Runner = "mocha"
	// RunnerCargoTest is the `cargo test --message-format=json`
	// parser.
	RunnerCargoTest Runner = "cargo-test"
	// RunnerNpmTest is the `npm test -- --json`
	// parser (delegates to the underlying jest
	// or mocha output).
	RunnerNpmTest Runner = "npm-test"
)

// IsValid reports whether r is one of the
// known runners.
func (r Runner) IsValid() bool {
	switch r {
	case RunnerGoTest, RunnerJest, RunnerMocha, RunnerCargoTest, RunnerNpmTest:
		return true
	}
	return false
}

// AllRunners returns the known runners in
// deterministic order.
func AllRunners() []Runner {
	return []Runner{RunnerGoTest, RunnerJest, RunnerMocha, RunnerCargoTest, RunnerNpmTest}
}

// DetectRunner inspects the test command and
// returns the runner the tool will use. The
// function is the heuristic the run-tests
// tool uses when the caller does not pass an
// explicit runner. The detection is
// case-insensitive on the program name.
func DetectRunner(command []string) Runner {
	if len(command) == 0 {
		return ""
	}
	prog := strings.ToLower(command[0])
	base := prog
	if idx := strings.LastIndex(prog, "/"); idx >= 0 {
		base = prog[idx+1:]
	}
	if idx := strings.LastIndex(base, `\`); idx >= 0 {
		base = base[idx+1:]
	}
	if strings.HasSuffix(base, ".exe") {
		base = strings.TrimSuffix(base, ".exe")
	}
	switch base {
	case "go":
		return RunnerGoTest
	case "jest":
		return RunnerJest
	case "mocha":
		return RunnerMocha
	case "cargo":
		return RunnerCargoTest
	case "npm", "npx", "pnpm", "yarn":
		return RunnerNpmTest
	}
	return ""
}

// RunTestsTool is the tools.Tool that runs the
// test suite and parses the output
// (REQ-VRF-RNT-001 to 006).
//
// Args:
//
//	command      (list, required) — the test
//	                                command argv.
//	test_command (string, optional) —
//	                                single-string
//	                                command (split
//	                                on whitespace).
//	runner       (string, optional) —
//	                                explicit runner
//	                                override.
//	timeout      (int,    optional) —
//	                                wall-clock budget
//	                                in seconds. Zero
//	                                means
//	                                DefaultTimeout.
type RunTestsTool struct {
	// Sandbox is the subprocess runner. Required.
	Sandbox sandbox.Runner
	// DefaultRunner is the runner the tool
	// uses when the caller does not pass an
	// explicit runner and the detection
	// heuristic returns empty. Defaults to
	// RunnerGoTest.
	DefaultRunner Runner
	// Audit is the audit log. nil means
	// "skip the row" (REQ-VRF-RNT-006).
	Audit *audit.Log
	// Now is the clock. nil defaults to
	// time.Now. The field is the test seam
	// the duration measurement uses.
	Now func() time.Time
}

// Name returns the tool name.
func (RunTestsTool) Name() string { return "run_tests" }

// runTestsEnv is the per-tool environment the runner
// uses for the child process. It overlays CI=1 on top
// of the parent's environment (InheritEnv: true in
// RunOptions) and forces HOME / USERPROFILE to a
// writable directory.
//
// The HOME/USERPROFILE override is the v0.37.0 fix
// for the "go test hangs forever" symptom: a parent
// process (CI runner, system service) launched without
// a HOME makes `go test` fall back to a path it cannot
// write, and the run hangs for the full timeout instead
// of failing fast. The verify package owns this fix
// because the tool runs source the operator already
// trusts (the model runner picks a tool at plan time).
func runTestsEnv() map[string]string {
	env := map[string]string{
		"CI": "1",
	}
	// The os.TempDir fallback is the production
	// safe-default: it always exists and is
	// writable. The override makes `go test` use a
	// known-good cache dir under the OS temp area.
	cacheRoot := filepath.Join(os.TempDir(), "aios-verify-cache")
	_ = os.MkdirAll(cacheRoot, 0o755)
	if runtime.GOOS == "windows" {
		env["USERPROFILE"] = cacheRoot
		env["HOME"] = cacheRoot
		// Go on Windows looks up %LocalAppData% when
		// GOCACHE is unset. The build cache lives
		// there by default; without this variable,
		// `go test` errors with "build cache is
		// required, but could not be located"
		// and exits non-zero.
		env["LOCALAPPDATA"] = cacheRoot
		env["GOCACHE"] = filepath.Join(cacheRoot, "go-build")
	} else {
		env["HOME"] = cacheRoot
		env["GOCACHE"] = filepath.Join(cacheRoot, "go-build")
	}
	return env
}

// Run executes the test command.
func (t *RunTestsTool) Run(args map[string]any, ctx tools.Context) (tools.Result, error) {
	command, err := resolveCommand(args)
	if err != nil {
		return tools.Result{OK: false, Summary: "bad command"}, err
	}
	runner := Runner(strings.ToLower(stringArg(args, "runner")))
	if runner == "" {
		runner = DetectRunner(command)
	}
	if runner == "" {
		runner = t.DefaultRunner
	}
	// v0.37.0: the optional `cwd` argument sets the
	// working directory for the child process. The
	// doc set's REQ-VRF-RNT-002 does not name a cwd
	// argument, but without one the tool inherits the
	// aios process's directory, so a plan that says
	// `go test ./...` tests whatever module the
	// binary happened to be started in rather than the
	// project the plan targets. When `cwd` is empty
	// the previous behaviour (inherit) is preserved.
	cwd := stringArg(args, "cwd")
	if runner == "" {
		runner = RunnerGoTest
	}
	if !runner.IsValid() {
		return tools.Result{OK: false, Summary: "bad runner"},
			&TestRunnerError{Op: "run", Command: strings.Join(command, " "), Reason: "unknown runner: " + string(runner)}
	}
	if t.Sandbox == nil {
		return tools.Result{OK: false, Summary: "no sandbox"},
			&TestRunnerError{Op: "run", Command: strings.Join(command, " "), Reason: "sandbox is required"}
	}
	timeout := intArg(args, "timeout")
	if timeout <= 0 {
		timeout = sandbox.DefaultTimeout
	}
	now := t.Now
	if now == nil {
		now = time.Now
	}
	start := now()
	res, runErr := t.Sandbox.Run(sandbox.RunOptions{
		Command: command,
		Cwd:     cwd,
		// Test runners need PATH to find the toolchain
		// (`go`, `npm`, `cargo`) and a writable HOME /
		// cache dir for their module and build caches.
		// The default sandbox isolation
		// (REQ-SND-ISO-002) would clear the env and only
		// set CI=1, which silently breaks `go test` and
		// friends. We opt in to env inheritance here
		// because the tool runs source the operator
		// already trusts.
		//
		// On top of inheritance we force HOME and
		// USERPROFILE to a writable directory. A parent
		// process launched from a service manager (or a
		// test harness) frequently has no HOME at all,
		// and `go test` then tries to create its build
		// cache under a path it cannot write, which
		// hangs rather than failing fast.
		InheritEnv: true,
		Env:        runTestsEnv(),
		Timeout:    timeout,
	})
	duration := elapsedMs(start)
	// Even when the command failed (non-zero
	// exit, signal, timeout), the per-runner
	// parser may still recover the test counts
	// from the captured output. The pattern
	// is: parse what we have; surface the run
	// error on top of the parsed result.
	parsed, perr := ParseRunnerOutput(runner, res.Stdout, res.Stderr, strings.Join(command, " "))
	if perr != nil && runErr == nil {
		// The runner exited 0 but the output
		// was unparseable. Surface the parse
		// error.
		t.writeAudit(ctx, nil, command, perr)
		return tools.Result{OK: false, Summary: "parse failed"}, &TestRunnerError{Op: "parse", Command: strings.Join(command, " "), Reason: "parse", Err: perr}
	}
	parsed.DurationMs = duration
	parsed.Runner = string(runner)
	parsed.Command = strings.Join(command, " ")
	if verr := parsed.Validate(); verr != nil {
		t.writeAudit(ctx, nil, command, verr)
		return tools.Result{OK: false, Summary: "invalid result"}, &TestRunnerError{Op: "validate", Command: strings.Join(command, " "), Reason: verr.Error()}
	}
	// Failure surface: a runner error, a
	// non-zero exit code, or any parsed
	// failure is a failed run. The parsed
	// result is still returned so the caller
	// can inspect the failures.
	if runErr != nil {
		t.writeAudit(ctx, &parsed, command, runErr)
		return tools.Result{
			OK:      false,
			Output:  testResultOutput(parsed),
			Summary: fmt.Sprintf("tests failed: %d passed, %d failed", parsed.Passed, parsed.Failed),
		}, &TestRunnerError{Op: "run", Command: strings.Join(command, " "), Reason: "runner error", Err: runErr}
	}
	if res.ExitCode != 0 || parsed.Failed > 0 {
		failErr := &TestRunnerError{Op: "run", Command: strings.Join(command, " "), Reason: fmt.Sprintf("exit code %d", res.ExitCode)}
		t.writeAudit(ctx, &parsed, command, failErr)
		return tools.Result{
			OK:      false,
			Output:  testResultOutput(parsed),
			Summary: fmt.Sprintf("tests failed: %d passed, %d failed", parsed.Passed, parsed.Failed),
		}, failErr
	}
	t.writeAudit(ctx, &parsed, command, nil)
	return tools.Result{
		OK:      true,
		Output:  testResultOutput(parsed),
		Summary: fmt.Sprintf("%d passed, %d failed, %d skipped", parsed.Passed, parsed.Failed, parsed.Skipped),
	}, nil
}

// testResultOutput renders the parsed result
// as the tools.Result Output map.
func testResultOutput(r TestResult) map[string]any {
	return map[string]any{
		"passed":      r.Passed,
		"failed":      r.Failed,
		"skipped":     r.Skipped,
		"duration_ms": r.DurationMs,
		"failures":    r.Failures,
		"runner":      r.Runner,
		"command":     r.Command,
	}
}

// resolveCommand accepts the args as either
// `command: []any`, `command: "..."` (single string,
// split on whitespace), or `test_command: "..."`. The
// single-string forms are a convenience; callers that
// need arguments with embedded spaces should use the
// list form.
func resolveCommand(args map[string]any) ([]string, error) {
	if list, ok := args["command"].([]any); ok && len(list) > 0 {
		out := make([]string, 0, len(list))
		for _, raw := range list {
			s, ok := raw.(string)
			if !ok {
				return nil, &TestRunnerError{Op: "resolve", Reason: "command entries must be strings"}
			}
			out = append(out, s)
		}
		return out, nil
	}
	if raw, ok := args["command"].(string); ok && raw != "" {
		return strings.Fields(raw), nil
	}
	if single := stringArg(args, "test_command"); single != "" {
		return strings.Fields(single), nil
	}
	return nil, &TestRunnerError{Op: "resolve", Reason: "command (or test_command) is required"}
}

func stringArg(m map[string]any, k string) string {
	if m == nil {
		return ""
	}
	s, _ := m[k].(string)
	return s
}

func intArg(m map[string]any, k string) int {
	if m == nil {
		return 0
	}
	switch v := m[k].(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	}
	return 0
}

// writeAudit writes a "run_tests" audit row
// (REQ-VRF-RNT-006).
func (t *RunTestsTool) writeAudit(ctx tools.Context, r *TestResult, command []string, runErr error) {
	if t.Audit == nil {
		return
	}
	taskID := ctx.TaskID
	if taskID == "" {
		taskID = "run_tests"
	}
	actor := ctx.Actor
	if actor == "" {
		actor = "run_tests"
	}
	payload := map[string]any{
		"command": strings.Join(command, " "),
	}
	if r != nil {
		payload["passed"] = r.Passed
		payload["failed"] = r.Failed
		payload["skipped"] = r.Skipped
		payload["duration_ms"] = r.DurationMs
		payload["runner"] = r.Runner
		if len(r.Failures) > 0 {
			failures := make([]map[string]any, 0, len(r.Failures))
			for _, f := range r.Failures {
				failures = append(failures, map[string]any{
					"name":      f.Name,
					"message":   f.Message,
					"traceback": f.Traceback,
				})
			}
			payload["failures"] = failures
		}
	}
	if runErr != nil {
		payload["error"] = runErr.Error()
	}
	_, _ = t.Audit.Write(&audit.Row{
		TaskID:   taskID,
		Kind:     "run_tests",
		Actor:    actor,
		RiskTier: 0,
		Payload:  payload,
	})
}

// ParseRunnerOutput dispatches to the
// per-runner parser. The function is the
// public entry point the test suite and the
// tool use.
func ParseRunnerOutput(runner Runner, stdout, stderr, command string) (TestResult, error) {
	switch runner {
	case RunnerGoTest:
		return ParseGoTestJSON(stdout, command)
	case RunnerJest, RunnerNpmTest:
		return ParseJestJSON(stdout, command)
	case RunnerMocha:
		return ParseMochaJSON(stdout, command)
	case RunnerCargoTest:
		return ParseCargoJSON(stdout, command)
	}
	return TestResult{}, &TestRunnerError{Op: "parse", Command: command, Reason: "no parser for runner " + string(runner)}
}

// ---------------------------------------------------------------------------
// Per-runner parsers
// ---------------------------------------------------------------------------

// goTestEvent is the `go test -json` event
// shape the parser uses. The full spec is
// larger; the parser consumes only the fields
// the result schema needs.
type goTestEvent struct {
	Action  string  `json:"Action"`
	Package string  `json:"Package"`
	Test    string  `json:"Test"`
	Output  string  `json:"Output"`
	Elapsed float64 `json:"Elapsed"`
}

// ParseGoTestJSON parses `go test -json`
// output. The output is a stream of JSON
// objects, one per line. The parser
// aggregates per-package timing and per-test
// pass / fail / skip counts.
func ParseGoTestJSON(output, command string) (TestResult, error) {
	// `go test -json` events that are not
	// test events still carry useful data;
	// for example `Action: "pass"` with no
	// `Test` field is a per-package summary.
	// The parser counts each test event once.
	passed := 0
	failed := 0
	skipped := 0
	var failures []TestFailure
	var traceback strings.Builder
	var currentFailedTest string
	scanner := bufio.NewScanner(strings.NewReader(output))
	// Increase the scanner buffer for the
	// long `Output` lines `go test` emits.
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var ev goTestEvent
		if err := json.Unmarshal(line, &ev); err != nil {
			continue
		}
		switch ev.Action {
		case "pass":
			if ev.Test != "" {
				passed++
			}
		case "fail":
			if ev.Test != "" {
				failed++
				currentFailedTest = ev.Test
				if failures == nil {
					failures = []TestFailure{}
				}
				failures = append(failures, TestFailure{
					Name:      ev.Test,
					Message:   "test failed",
					Traceback: traceback.String(),
				})
				traceback.Reset()
			}
		case "skip":
			if ev.Test != "" {
				skipped++
			}
		case "output":
			// Capture failure traceback until
			// the next non-output event.
			if currentFailedTest != "" {
				traceback.WriteString(ev.Output)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return TestResult{}, &TestRunnerError{Op: "parse", Command: command, Reason: "go-test scan", Err: err}
	}
	_ = currentFailedTest
	return TestResult{
		Passed:   passed,
		Failed:   failed,
		Skipped:  skipped,
		Failures: failures,
	}, nil
}

// jestResult is the `jest --json` result
// shape the parser uses.
type jestResult struct {
	NumPassedTests  int `json:"numPassedTests"`
	NumFailedTests  int `json:"numFailedTests"`
	NumPendingTests int `json:"numPendingTests"`
	NumTotalTests   int `json:"numTotalTests"`
	TestResults     []struct {
		FullName    string `json:"fullName"`
		Status      string `json:"status"`
		Message     string `json:"message"`
		ConsoleLogs []struct {
			Message string `json:"message"`
		} `json:"consoleLogs"`
	} `json:"testResults"`
}

// ParseJestJSON parses `jest --json` output.
// The output is a single JSON object.
func ParseJestJSON(output, command string) (TestResult, error) {
	var r jestResult
	if err := json.Unmarshal([]byte(output), &r); err != nil {
		// Jest sometimes emits additional
		// status text before the JSON object.
		// Try to recover by scanning for the
		// first '{' and re-parsing.
		if idx := strings.Index(output, "{"); idx >= 0 {
			if err2 := json.Unmarshal([]byte(output[idx:]), &r); err2 != nil {
				return TestResult{}, &TestRunnerError{Op: "parse", Command: command, Reason: "jest unmarshal", Err: err}
			}
		} else {
			return TestResult{}, &TestRunnerError{Op: "parse", Command: command, Reason: "jest unmarshal", Err: err}
		}
	}
	failures := make([]TestFailure, 0, r.NumFailedTests)
	for _, t := range r.TestResults {
		if t.Status != "failed" {
			continue
		}
		var traceback strings.Builder
		for _, c := range t.ConsoleLogs {
			traceback.WriteString(c.Message)
			traceback.WriteString("\n")
		}
		failures = append(failures, TestFailure{
			Name:      t.FullName,
			Message:   t.Message,
			Traceback: traceback.String(),
		})
	}
	return TestResult{
		Passed:   r.NumPassedTests,
		Failed:   r.NumFailedTests,
		Skipped:  r.NumPendingTests,
		Failures: failures,
	}, nil
}

// mochaTest is the per-test entry in the
// mocha JSON reporter output.
type mochaTest struct {
	Title     string `json:"title"`
	FullTitle string `json:"fullTitle"`
	Status    string `json:"status"` // "passed" | "failed" | "pending"
	Err       struct {
		Message string `json:"message"`
		Stack   string `json:"stack"`
	} `json:"err"`
}

// mochaSuite is the `mocha --reporter json`
// outer object. The reporter emits a `stats`
// block at the end.
type mochaSuite struct {
	Stats struct {
		Passes   int `json:"passes"`
		Failures int `json:"failures"`
		Pending  int `json:"pending"`
	} `json:"stats"`
	Passes   []mochaTest `json:"passes"`
	Failures []mochaTest `json:"failures"`
	Pending  []mochaTest `json:"pending"`
}

// ParseMochaJSON parses `mocha --reporter
// json` output. The output is a single JSON
// object.
func ParseMochaJSON(output, command string) (TestResult, error) {
	var s mochaSuite
	if err := json.Unmarshal([]byte(output), &s); err != nil {
		return TestResult{}, &TestRunnerError{Op: "parse", Command: command, Reason: "mocha unmarshal", Err: err}
	}
	failures := make([]TestFailure, 0, len(s.Failures))
	for _, t := range s.Failures {
		name := t.FullTitle
		if name == "" {
			name = t.Title
		}
		failures = append(failures, TestFailure{
			Name:      name,
			Message:   t.Err.Message,
			Traceback: t.Err.Stack,
		})
	}
	return TestResult{
		Passed:   s.Stats.Passes,
		Failed:   s.Stats.Failures,
		Skipped:  s.Stats.Pending,
		Failures: failures,
	}, nil
}

// cargoEvent is the `cargo test
// --message-format=json` event shape. We
// only consume the fields the test result
// schema needs; the rest is ignored.
type cargoEvent struct {
	Reason string `json:"reason"` // "compiler-message" | "test-started" | "test-finished" | ...
	Type   string `json:"type"`   // "test" | "diagnostic" | ...
	Name   string `json:"name"`
	Event  struct {
		// "started" | "ok" | "failed" | "ignored"
		// for `type: "test"`.
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"event"`
}

// ParseCargoJSON parses `cargo test
// --message-format=json` output. The output
// is a stream of JSON objects, one per line.
func ParseCargoJSON(output, command string) (TestResult, error) {
	passed := 0
	failed := 0
	skipped := 0
	var failures []TestFailure
	pendingTraceback := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(output))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var ev cargoEvent
		if err := json.Unmarshal(line, &ev); err != nil {
			continue
		}
		if ev.Type != "test" {
			continue
		}
		switch ev.Event.Type {
		case "started":
			pendingTraceback[ev.Name] = ""
		case "ok":
			passed++
			delete(pendingTraceback, ev.Name)
		case "failed":
			failed++
			if failures == nil {
				failures = []TestFailure{}
			}
			failures = append(failures, TestFailure{
				Name:      ev.Name,
				Message:   ev.Event.Message,
				Traceback: pendingTraceback[ev.Name],
			})
			delete(pendingTraceback, ev.Name)
		case "ignored":
			skipped++
			delete(pendingTraceback, ev.Name)
		}
	}
	if err := scanner.Err(); err != nil {
		return TestResult{}, &TestRunnerError{Op: "parse", Command: command, Reason: "cargo scan", Err: err}
	}
	return TestResult{
		Passed:   passed,
		Failed:   failed,
		Skipped:  skipped,
		Failures: failures,
	}, nil
}

// bufferCloser is the small helper the
// tests use to make a bytes.Buffer satisfy
// the io.Closer the sandbox.Runner may
// expect. The helper is unused in production
// (the sandbox opens its own pipes) and is
// kept here for the test surface.
type bufferCloser struct {
	*bytes.Buffer
}

// Close is a no-op for the bufferCloser.
func (bufferCloser) Close() error { return nil }
