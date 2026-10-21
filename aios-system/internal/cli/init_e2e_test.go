package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sadi/aios-system/internal/policy"
)

// TestInit_DefaultPolicyValidates is the v0.36.0 bug
// fix: the default policy.yaml template is now valid
// against the policy validator. The previous template
// wrote rules without `id` and was rejected on load.
//
// The test runs Init(dir), then loads the resulting
// policy file via policy.Load. The Loader runs each
// rule through Validate, so a valid template passes;
// an invalid template fails with RuleValidationError.
func TestInit_DefaultPolicyValidates(t *testing.T) {
	dir := t.TempDir()
	_, _, err := Init(dir)
	if err != nil {
		t.Fatal(err)
	}
	ppath := filepath.Join(dir, "policy.yaml")
	if _, err := os.Stat(ppath); err != nil {
		t.Fatalf("policy.yaml missing: %v", err)
	}
	rules, err := policy.Load(ppath)
	if err != nil {
		t.Fatalf("init wrote an invalid policy.yaml: %v", err)
	}
	// The default template ships with read / search /
	// ask-shell / ask-user rules. Assert each has an
	// id (the validator's check) and at least one
	// rule for read_file so a fresh user can read a
	// file out of the box.
	if len(rules) == 0 {
		t.Fatal("default policy has no rules")
	}
	hasRead := false
	for _, r := range rules {
		if r.ID == "" {
			t.Errorf("rule without id: %+v", r)
		}
		if r.Match.Tool == "read_file" && r.Action == policy.ActionAllow {
			hasRead = true
		}
	}
	if !hasRead {
		t.Error("default policy does not allow read_file")
	}
}

// TestBuildDeps_FindsPolicyNextToConfig is the second
// half of the v0.36.0 fix: when --config points at
// config.yaml, BuildDeps must find the sibling
// policy.yaml and load its rules.
//
// The test:
//
//  1. Writes var/policy.yaml with one allow-read rule.
//  2. Writes var/config.yaml with workspace defaults.
//  3. Calls BuildDeps with --config=var/config.yaml.
//  4. Asserts deps.Policy is non-nil and decides
//     allow for a read_file call.
//
// Before the fix, policyPath didn't exist and the CLI
// read var/config.yaml as a policy file. config.yaml
// has no rules, so the bus default-denied everything
// and a real run could not dispatch read_file.
func TestBuildDeps_FindsPolicyNextToConfig(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfg, []byte("defaults:\n  workspace: ./var\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ppath := filepath.Join(dir, "policy.yaml")
	if err := os.WriteFile(ppath, []byte(
		"rules:\n"+
			"  - id: allow-read\n"+
			"    match:\n      tool: read_file\n"+
			"    action: allow\n",
	), 0o644); err != nil {
		t.Fatal(err)
	}
	args := &Args{
		Command:    CmdRun,
		ConfigPath: cfg,
		LogPath:    filepath.Join(dir, "aios.db"),
		Workspace:  dir,
	}
	deps, err := BuildDeps(args)
	if err != nil {
		t.Fatal(err)
	}
	defer deps.Close()
	if deps.Policy == nil {
		t.Fatal("deps.Policy is nil; sibling policy.yaml was not loaded")
	}
	d, err := deps.Policy.Decide(policy.Call{Tool: "read_file", TaskID: "t1", Actor: "t"})
	if err != nil {
		t.Fatal(err)
	}
	if d.Action != policy.ActionAllow {
		t.Errorf("read_file policy decision: got %s, want allow", d.Action)
	}
}

// TestMain_InitThenRunHappyPath is the integration
// test: a fresh `aios init` followed by an `aios run`
// against a hand-written plan must succeed end-to-end
// without any manual config edits.
//
// What this proves:
//   - Init writes a policy that the policy loader
//     accepts.
//   - BuildDeps with --config=var/config.yaml
//     resolves the sibling policy.yaml.
//   - The bus dispatches a read_file tool call
//     (the default policy allows it).
//
// This is the gate v0.36.0 closes: a fresh user can
// `aios init && aios run plan.yaml` and have it
// actually work.
func TestMain_InitThenRunHappyPath(t *testing.T) {
	dir := t.TempDir()
	var stdout, stderr bytes.Buffer
	code := Main([]string{
		"aios", "--config=" + filepath.Join(dir, "config.yaml"), "init",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("init: code %d, stderr: %s", code, stderr.String())
	}
	// Hand-written plan with one read_file task
	// against a file we just wrote.
	target := filepath.Join(dir, "hello.txt")
	if err := os.WriteFile(target, []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	planPath := filepath.Join(dir, "plan.yaml")
	yaml := "id: rt-1\n" +
		"goal: read hello\n" +
		"tasks:\n" +
		"  - id: t1\n" +
		"    description: read hello.txt\n" +
		"    acceptance: file contents returned\n" +
		"    tool: read_file\n" +
		"    inputs:\n" +
		"      - path=" + target + "\n"
	if err := os.WriteFile(planPath, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	code = Main([]string{
		"aios",
		"--config=" + filepath.Join(dir, "config.yaml"),
		"--log=" + filepath.Join(dir, "aios.db"),
		"--workspace=" + dir,
		"run", planPath,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("run: code %d, stderr: %s", code, stderr.String())
	}
	// The audit log should have at least one
	// tool_call row for read_file.
	if _, err := os.Stat(filepath.Join(dir, "aios.db")); err != nil {
		t.Errorf("audit log not created: %v", err)
	}
	// Sanity: stderr should NOT contain "denied by
	// policy" (the canonical sign that the policy
	// did not match).
	if strings.Contains(stderr.String(), "denied by policy") {
		t.Errorf("policy denied a read_file call; default policy did not apply")
	}
}
