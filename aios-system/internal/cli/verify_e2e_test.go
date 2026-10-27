package cli

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Hamidullah02/aios/aios-system/internal/core/audit"
	"github.com/Hamidullah02/aios/aios-system/internal/policy"
	"github.com/Hamidullah02/aios/aios-system/internal/tools"
	"github.com/Hamidullah02/aios/aios-system/internal/verify"
)

// TestRunTests_EndToEnd exercises the verify-package
// RunTestsTool through the production tool bus. The
// test writes a tiny Go module with a passing test,
// dispatches run_tests with a `go test -json` command,
// and asserts:
//
//   - the call does not return ToolNotFoundError
//     (the bus has the tool registered)
//   - the audit log has a "run_tests" row
//
// This is the v0.35.0 catch-#2 smoke: the verify
// module is now wired into the bus, and a real
// `run_tests` task runs end-to-end.
func TestRunTests_EndToEnd(t *testing.T) {
	dir := t.TempDir()
	// Tiny Go module: a single test that always
	// passes. The module is required because
	// `go test -json` needs a go.mod in the dir.
	goMod := "module rtest\n\ngo 1.21\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "rtest_test.go"),
		[]byte("package rtest\n\nimport \"testing\"\n\nfunc TestPass(t *testing.T) {}\n"),
		0o644); err != nil {
		t.Fatal(err)
	}
	// Audit log.
	log, err := audit.Open(audit.Config{Path: filepath.Join(dir, "aios.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()
	// Policy that allows run_tests.
	gw := policy.New([]policy.Rule{
		{ID: "allow-tests", Match: policy.Match{Tool: "run_tests"}, Action: policy.ActionAllow},
	}, log)
	// Build the verify-tool map and put run_tests
	// into a bus. We use wireVerifyTools(nil, log) so
	// only run_tests is wired (no router for
	// self_check).
	verifyTools := wireVerifyTools(nil, log)
	bus := tools.New(tools.Config{
		Tools:  verifyTools,
		Policy: gw,
		Audit:  log,
	})
	// Dispatch. The run_tests tool accepts a
	// `command` (list) or `test_command` (single
	// string). The sandbox runner inherits the
	// caller's Cwd; we pass the temp dir as the
	// package selector so go test runs in the
	// module we just wrote.
	//
	// We do NOT assert OK=true here: `go test` may
	// exit 0 (tests pass) or non-zero (e.g. Windows
	// path with spaces trips the parser). What we
	// DO assert: the tool is registered (no
	// ToolNotFoundError) AND the audit log has a
	// `run_tests` row. The audit row is written on
	// both success and failure (the tool writes it
	// before returning), so a non-empty row proves
	// the dispatch reached the tool body.
	res, err := bus.Dispatch("run_tests", map[string]any{
		"test_command": "go test -json -count=1 " + dir,
		"runner":       "go-test",
	}, tools.Context{Actor: "t", TaskID: "t1"})
	if err != nil {
		// A non-nil error is fine; the contract is
		// that the tool ran (i.e. the error is NOT
		// a ToolNotFoundError).
		var tnf *tools.ToolNotFoundError
		if errors.As(err, &tnf) {
			t.Fatalf("run_tests not registered: %v", err)
		}
	}
	_ = res
	// The audit log should have a "run_tests" row.
	rows, err := log.GetByKind("run_tests", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) == 0 {
		t.Fatal("expected at least one run_tests audit row")
	}
}

// TestRunTests_NotFoundRunnerStillProvesRegistered
// proves the wiring works even when the runner
// cannot parse the output. The contract: NOT a
// tool-not-found error. Any other error is fine.
func TestRunTests_NotFoundRunnerStillProvesRegistered(t *testing.T) {
	dir := t.TempDir()
	log, err := audit.Open(audit.Config{Path: filepath.Join(dir, "aios.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()
	gw := policy.New([]policy.Rule{
		{ID: "allow-tests", Match: policy.Match{Tool: "run_tests"}, Action: policy.ActionAllow},
	}, log)
	verifyTools := wireVerifyTools(nil, log)
	bus := tools.New(tools.Config{
		Tools:  verifyTools,
		Policy: gw,
		Audit:  log,
	})
	// Use a command that produces output the runner
	// can't parse as go-test JSON (so we get a parse
	// error, NOT a not-found error). `go env GOOS`
	// runs without compiling; its output is plain
	// `KEY=val` lines, not the JSON stream go-test
	// emits.
	// `go env GOOS` runs quickly, exits 0, and emits
	// plain text (not go-test JSON). The parser
	// typically returns a zero-count TestResult with
	// no error, OR a parse error. Either outcome
	// proves the bus dispatched the call: the test
	// is "the tool is registered and reachable",
	// not "the tool errors on bad JSON".
	res2, err := bus.Dispatch("run_tests", map[string]any{
		"command": []any{"go", "env", "GOOS"},
		"runner":  "go-test",
	}, tools.Context{Actor: "t", TaskID: "t1"})
	_ = res2
	if err != nil {
		// A parse error is fine; the contract is
		// NOT a "not found" (which would mean the
		// tool is unregistered).
		if strings.Contains(err.Error(), "not found") {
			t.Errorf("run_tests not registered in the bus: %v", err)
		}
	}
	// If err is nil, the bus returned a result with
	// OK=true and zero counts — that also proves the
	// dispatch reached the tool body.
}

// TestSelfCheckTool_ParseResponse is a small
// regression: the verify package's SelfCheckTool
// parses the model's response into a
// VerificationResult. We do not exercise the model
// path (no API key) but we do exercise the parser
// via the public surface, to make sure the wiring
// did not break the parse contract.
func TestSelfCheckTool_ParseResponse(t *testing.T) {
	res, err := verify.ParseVerificationResponse(`{"status":"pass","reason":"ok","suggestions":["x"]}`)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != verify.StatusPass {
		t.Errorf("status: %s", res.Status)
	}
}
