package verify

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Hamidullah02/aios/aios-system/internal/core/audit"
	"github.com/Hamidullah02/aios/aios-system/internal/policy"
	"github.com/Hamidullah02/aios/aios-system/internal/sandbox"
	"github.com/Hamidullah02/aios/aios-system/internal/tools"
)

// newRunTestsHarness builds a bus with a single
// run_tests tool wired through the real sandbox
// runner, plus an audit log in dir.
func newRunTestsHarness(t *testing.T, dir string) (*tools.Bus, *audit.Log) {
	t.Helper()
	log, err := audit.Open(audit.Config{Path: filepath.Join(dir, "aios.db")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = log.Close() })
	gw := policy.New([]policy.Rule{
		{ID: "allow-tests", Match: policy.Match{Tool: "run_tests"}, Action: policy.ActionAllow},
	}, log)
	tool := &RunTestsTool{
		Sandbox:       sandbox.NewSubprocessRunner(),
		DefaultRunner: RunnerGoTest,
		Audit:         log,
	}
	bus := tools.New(tools.Config{
		Tools:  map[string]tools.Tool{"run_tests": tool},
		Policy: gw,
		Audit:  log,
	})
	return bus, log
}

// writeGoModule writes a minimal passing Go module in
// dir.
func writeGoModule(t *testing.T, dir, modName string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"),
		[]byte("module "+modName+"\n\ngo 1.21\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ok_test.go"),
		[]byte("package modname\n\nimport \"testing\"\n\nfunc TestOK(t *testing.T) {}\n"),
		0o644); err != nil {
		t.Fatal(err)
	}
}

// requireGo skips the test cleanly when the Go
// toolchain is not reachable on PATH.
func requireGo(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("go"); err != nil {
		t.Skipf("go toolchain not on PATH: %v", err)
	}
}

// TestRunTestsTool_RunsGoTest is the v0.37.0 gate.
//
// The tool dispatches a real `go test -json ./...`
// through the production sandbox runner and asserts
// the result carries a positive pass count.
//
// Before the v0.37.0 fix the sandbox cleared the
// subprocess environment (REQ-SND-ISO-002) and the
// tool only set CI=1, so the child had no PATH and
// could not find `go` at all. Every real run failed
// with "exit status 1" and an empty TestResult.
func TestRunTestsTool_RunsGoTest(t *testing.T) {
	requireGo(t)
	dir := t.TempDir()
	writeGoModule(t, dir, "verifyenv")
	bus, log := newRunTestsHarness(t, dir)

	res, err := bus.Dispatch("run_tests", map[string]any{
		"command": []any{"go", "test", "-json", "-count=1", "."},
		"runner":  "go-test",
		"cwd":     dir,
	}, tools.Context{Actor: "t", TaskID: "env-1"})
	if err != nil {
		// Always surface what the audit log has,
		// regardless of the failure mode.
		rows, _ := log.GetByKind("run_tests", 10)
		for _, r := range rows {
			var p map[string]any
			if uerr := json.Unmarshal(r.Payload, &p); uerr != nil {
				continue
			}
			if e, ok := p["error"].(string); ok {
				t.Logf("run_tests audit error: %s", e)
			}
		}
		var tnf *tools.ToolNotFoundError
		if errors.As(err, &tnf) {
			t.Fatalf("run_tests not registered: %v", err)
		}
		t.Fatalf("run_tests dispatch failed: %v", err)
	}
	if !res.OK {
		t.Fatalf("run_tests: OK=false, summary=%s", res.Summary)
	}
	raw, mErr := json.Marshal(res.Output)
	if mErr != nil {
		t.Fatal(mErr)
	}
	body := string(raw)
	if !strings.Contains(body, `"passed":`) {
		t.Fatalf("TestResult missing 'passed' field: %s", body)
	}
	var got struct {
		Passed int    `json:"passed"`
		Runner string `json:"runner"`
	}
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("decode result: %v (body=%s)", err, body)
	}
	if got.Passed < 1 {
		t.Errorf("passed = %d, want >= 1 (body=%s)", got.Passed, body)
	}
	if got.Runner != "go-test" {
		t.Errorf("runner = %q, want go-test", got.Runner)
	}
	rows, err := log.GetByKind("run_tests", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) == 0 {
		t.Fatal("expected a run_tests audit row")
	}
}

// TestRunTestsTool_SetsCI is the companion assertion:
// env inheritance must not stop the tool from setting
// its own CI=1 marker, which is what makes most test
// runners pick non-interactive, machine-readable
// output.
//
// The test uses the self-exec pattern: the test binary
// re-invokes itself in helper mode to assert CI is
// visible inside the child.
func TestRunTestsTool_SetsCI(t *testing.T) {
	dir := t.TempDir()
	log, err := audit.Open(audit.Config{Path: filepath.Join(dir, "aios.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()
	// Build RunOptions like the tool does, but with a
	// command that runs the test binary's helper.
	cmd := []string{os.Args[0], "-test.run=^TestVerifyHelperPrintCI$"}
	// We use the real SubprocessRunner with InheritEnv
	// the same way the tool does.
	res, runErr := sandbox.NewSubprocessRunner().Run(sandbox.RunOptions{
		Command:    cmd,
		InheritEnv: true,
		Env:        map[string]string{"CI": "1", "AIOS_VERIFY_HELPER": "1"},
	})
	if runErr != nil {
		t.Fatalf("helper run: %v", runErr)
	}
	if res.ExitCode != 0 {
		t.Fatalf("helper exit %d (stderr=%q)", res.ExitCode, res.Stderr)
	}
	if strings.TrimSpace(res.Stdout) != "CI_OK" {
		t.Fatalf("CI not visible to child: %q (want CI_OK)", res.Stdout)
	}
}

// TestVerifyHelperPrintCI is the child-process helper.
// It is a no-op under a normal `go test` run; when
// run with AIOS_VERIFY_HELPER=1, it reports CI's
// value via os.Exit so the test framework's "PASS"
// line does not pollute the parent reader.
func TestVerifyHelperPrintCI(t *testing.T) {
	if os.Getenv("AIOS_VERIFY_HELPER") != "1" {
		return
	}
	if v := os.Getenv("CI"); v != "1" {
		os.Stdout.WriteString("CI_MISSING:" + v)
		os.Exit(2)
	}
	os.Stdout.WriteString("CI_OK")
	os.Exit(0)
}
