package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sadi/aios-system/internal/tools"
)

// TestRunSearch_Glob exercises the search glob
// subcommand from the CLI. The test creates a temp
// dir with two files and asserts both names appear in
// the output.
func TestRunSearch_Glob(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"alpha.txt", "beta.md"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("AIOS_WORKSPACE", dir)
	var stdout, stderr bytes.Buffer
	args := &Args{
		Command:    CmdSearch,
		Positional: []string{"glob"},
		Pattern:    "*.txt",
		Path:       dir,
	}
	code := runSearch(context.Background(), args, nil, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("code = %d, stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "alpha.txt") {
		t.Errorf("missing alpha.txt in output: %s", stdout.String())
	}
	if strings.Contains(stdout.String(), "beta.md") {
		t.Errorf("beta.md should not match *.txt: %s", stdout.String())
	}
}

// TestRunSearch_UnknownSubCommand ensures an unknown
// sub-action produces a user error.
func TestRunSearch_UnknownSubCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	args := &Args{
		Command:    CmdSearch,
		Positional: []string{"facet"},
	}
	code := runSearch(context.Background(), args, nil, &stdout, &stderr)
	if code != ExitUserError {
		t.Errorf("code = %d, want user error", code)
	}
}

// TestRunSearch_NoArgs ensures a missing positional
// argument produces a user error.
func TestRunSearch_NoArgs(t *testing.T) {
	var stdout, stderr bytes.Buffer
	args := &Args{Command: CmdSearch}
	code := runSearch(context.Background(), args, nil, &stdout, &stderr)
	if code != ExitUserError {
		t.Errorf("code = %d, want user error", code)
	}
}

// TestRunPlan_Validates exercises the plan validate
// subcommand on a real plan YAML. The encoder in
// planner.EncodePlan is the same one the agent uses.
func TestRunPlan_Validates(t *testing.T) {
	dir := t.TempDir()
	planPath := filepath.Join(dir, "plan.yaml")
	yaml := `id: test-1
goal: smoke
tasks:
  - id: t1
    description: hello
    acceptance: ok
    tool: read_file
`
	if err := os.WriteFile(planPath, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	args := &Args{Command: CmdPlan, PlanPath: planPath}
	code := runPlan(context.Background(), args, nil, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("code = %d, stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"id":"test-1"`) {
		t.Errorf("encoded plan missing id: %s", stdout.String())
	}
}

// TestRunPlan_InvalidYAML ensures a bad plan produces
// a user error.
func TestRunPlan_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	planPath := filepath.Join(dir, "plan.yaml")
	if err := os.WriteFile(planPath, []byte("not: valid: yaml: : :"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	args := &Args{Command: CmdPlan, PlanPath: planPath}
	code := runPlan(context.Background(), args, nil, &stdout, &stderr)
	if code != ExitUserError {
		t.Errorf("code = %d, want user error", code)
	}
}

// TestRunModel_Missing ensures a missing price file
// produces a system error (not a panic).
func TestRunModel_Missing(t *testing.T) {
	var stdout, stderr bytes.Buffer
	args := &Args{
		Command:   CmdModel,
		PricePath: filepath.Join(t.TempDir(), "missing.yaml"),
	}
	code := runModel(context.Background(), args, nil, &stdout, &stderr)
	if code == ExitOK {
		t.Errorf("code = OK, want system error")
	}
}

// TestRunSession_EmptyDir ensures a non-existent
// session dir produces a system error.
func TestRunSession_Missing(t *testing.T) {
	var stdout, stderr bytes.Buffer
	args := &Args{
		Command:      CmdSession,
		SessionsPath: filepath.Join(t.TempDir(), "no", "such", "dir"),
	}
	code := runSession(context.Background(), args, nil, &stdout, &stderr)
	if code == ExitOK {
		t.Errorf("code = OK, want system error")
	}
}

// TestResolveWorkspace_AbsPath ensures absolute paths
// are honoured by the helper.
func TestResolveWorkspace_AbsPath(t *testing.T) {
	got, err := resolveWorkspace(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if got == "" {
		t.Error("empty workspace")
	}
}

// TestResolveWorkspace_RelPath ensures a relative path
// is resolved to absolute.
func TestResolveWorkspace_RelPath(t *testing.T) {
	got, err := resolveWorkspace(".")
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(got) {
		t.Errorf("expected absolute path, got %q", got)
	}
}

// TestPrintToolResult_OkOutput exercises the small
// result writer. The helper is the single point that
// knows how to print a tool's output map.
func TestPrintToolResult_OkOutput(t *testing.T) {
	var buf bytes.Buffer
	printToolResult(&buf, tools.Result{OK: true, Output: map[string]any{"x": "y"}}, nil)
	if !strings.Contains(buf.String(), `"x":"y"`) && !strings.Contains(buf.String(), "y") {
		t.Errorf("output missing y: %s", buf.String())
	}
}

// TestPrintToolResult_NotOk ensures a failed result
// prints the summary.
func TestPrintToolResult_NotOk(t *testing.T) {
	var buf bytes.Buffer
	printToolResult(&buf, tools.Result{OK: false, Summary: "nope"}, nil)
	if !strings.Contains(buf.String(), "ok=false") {
		t.Errorf("missing ok=false: %s", buf.String())
	}
}

// TestReadPriceYAMLBytes is a focused unit test for
// the YAML reader.
func TestReadPriceYAMLBytes(t *testing.T) {
	prices, err := readPriceYAMLBytes([]byte(`
prices:
  - model: gpt-4
    input_per_1k: 0.03
    output_per_1k: 0.06
  - model: claude-3
    input_per_1k: 0.015
    output_per_1k: 0.075
`))
	if err != nil {
		t.Fatal(err)
	}
	if len(prices) != 2 {
		t.Fatalf("expected 2 prices, got %d", len(prices))
	}
	if prices[0].Model != "gpt-4" {
		t.Errorf("model = %q", prices[0].Model)
	}
	if prices[1].InputPer1K != 0.015 {
		t.Errorf("input = %v", prices[1].InputPer1K)
	}
}

// TestReadPriceYAMLBytes_BadInput ensures parse errors
// surface.
func TestReadPriceYAMLBytes_BadInput(t *testing.T) {
	_, err := readPriceYAMLBytes([]byte("not: valid: yaml: : :"))
	if err == nil {
		t.Fatal("expected error")
	}
}
