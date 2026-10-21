package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sadi/aios-system/internal/policy"
)

// TestPolicyPath_ReturnsPolicyFileWhenBaseMatches
// asserts rule 2: if --config already names a policy
// file, return it unchanged. This is the test-compat
// / operator-override path.
func TestPolicyPath_ReturnsPolicyFileWhenBaseMatches(t *testing.T) {
	dir := t.TempDir()
	pdir := filepath.Join(dir, "var")
	if err := os.MkdirAll(pdir, 0o755); err != nil {
		t.Fatal(err)
	}
	ppath := filepath.Join(pdir, "policy.yaml")
	if err := os.WriteFile(ppath, []byte("rules: []"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := policyPath(ppath)
	if err != nil {
		t.Fatal(err)
	}
	if got != ppath {
		t.Errorf("got %q, want %q", got, ppath)
	}
}

// TestPolicyPath_FindsSiblingForConfig asserts rule 3:
// if --config points at config.yaml (or anything else),
// look for a sibling policy.yaml and return its path.
func TestPolicyPath_FindsSiblingForConfig(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfg, []byte("defaults:\n  workspace: ./var"), 0o644); err != nil {
		t.Fatal(err)
	}
	ppath := filepath.Join(dir, "policy.yaml")
	if err := os.WriteFile(ppath, []byte("rules: []"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := policyPath(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if got != ppath {
		t.Errorf("got %q, want %q", got, ppath)
	}
}

// TestPolicyPath_FallsBackToConfigWhenNoSibling
// asserts rule 4: when --config points at a file
// with no sibling policy.yaml, fall back to
// --config itself. This preserves back-compat
// with callers (and tests) that put the rules in a
// single file.
func TestPolicyPath_FallsBackToConfigWhenNoSibling(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfg, []byte("defaults:\n  workspace: ./var"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := policyPath(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if got != cfg {
		t.Errorf("got %q, want fallback to %q", got, cfg)
	}
}

// TestPolicyPath_ErrorsOnEmptyConfig asserts the
// early-out: --config must be non-empty.
func TestPolicyPath_ErrorsOnEmptyConfig(t *testing.T) {
	_, err := policyPath("")
	if err == nil {
		t.Fatal("expected error: empty --config")
	}
}

// TestDefaultPolicyYAML_ParsesAndValidates asserts the
// v0.36.0 bug fix: the init template is now valid
// against policy.Load + Rule.Validate. The previous
// template wrote rules without `id` and was rejected
// by the validator.
func TestDefaultPolicyYAML_ParsesAndValidates(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "policy.yaml")
	if err := os.WriteFile(path, []byte(defaultPolicyYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	rules, err := policy.Load(path)
	if err != nil {
		t.Fatalf("default policy is invalid: %v", err)
	}
	if len(rules) == 0 {
		t.Fatal("default policy has no rules")
	}
	for i, r := range rules {
		if r.ID == "" {
			t.Errorf("rule[%d]: id is empty", i)
		}
		if err := r.Validate(); err != nil {
			t.Errorf("rule[%d] %q invalid: %v", i, r.ID, err)
		}
	}
}
