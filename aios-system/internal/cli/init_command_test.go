package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sadi/aios-system/internal/policy"
)

func TestInit_FreshWorkspace(t *testing.T) {
	dir := t.TempDir()
	created, existed, err := Init(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(created) != 4 {
		t.Errorf("created: %d", len(created))
	}
	if len(existed) != 0 {
		t.Errorf("existed: %d", len(existed))
	}
	for _, name := range []string{"policy.yaml", "egress.yaml", "prices.yaml", "config.yaml"} {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("missing %s: %v", name, err)
		}
	}
}

func TestInit_ReRunIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	_, _, err := Init(dir)
	if err != nil {
		t.Fatal(err)
	}
	created, existed, err := Init(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(created) != 0 {
		t.Errorf("created: %d (should be 0 on re-run)", len(created))
	}
	if len(existed) != 4 {
		t.Errorf("existed: %d", len(existed))
	}
}

func TestInit_EmptyWorkspace(t *testing.T) {
	_, _, err := Init("")
	if err != nil {
		t.Fatal(err)
	}
	// Cleanup: the default workspace landed somewhere on
	// disk; we accept the side-effect for this test.
	_ = os.RemoveAll("./var")
}

func TestPrintInitReport(t *testing.T) {
	var buf bytes.Buffer
	PrintInitReport(&buf, []string{"/a/b/c"}, []string{"/a/b/d"})
	s := buf.String()
	if !strings.Contains(s, "Created:") {
		t.Errorf("missing Created: %s", s)
	}
	if !strings.Contains(s, "Already present:") {
		t.Errorf("missing Already present: %s", s)
	}
	if !strings.Contains(s, "/a/b/c") {
		t.Errorf("missing created path: %s", s)
	}
}

func TestPrintInitReport_BothEmpty(t *testing.T) {
	var buf bytes.Buffer
	PrintInitReport(&buf, nil, nil)
	if !strings.Contains(buf.String(), "Nothing to do") {
		t.Errorf("got %q", buf.String())
	}
}

func TestJoinLines(t *testing.T) {
	if got := joinLines("a", "b", "c"); got != "a\nb\nc" {
		t.Errorf("got %q", got)
	}
}

// TestInit_ScaffoldedPolicyLoads is the guard for the bug where
// `aios init` wrote a policy the policy loader rejected.
//
// The scaffold shipped `resource: "{command}"`, but the
// template engine only accepts the reserved names
// (task/tool/path/host/actor/tier) and `arg.<name>` for tool
// arguments. The loader rejected the file, so the default
// policy -- the one every new user starts from -- could not be
// used, and every run_shell call died with a config error
// before it ever reached the approval prompt the rule exists
// to raise.
//
// The other init tests assert the files EXIST. That is not
// the same as the files being USABLE, and the difference is
// exactly where this bug lived.
func TestInit_ScaffoldedPolicyLoads(t *testing.T) {
	dir := t.TempDir()
	if _, _, err := Init(dir); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "policy.yaml")
	rules := mustLoadPolicy(t, path)
	if len(rules) == 0 {
		t.Fatal("scaffolded policy has no rules")
	}
}

// TestInit_ScaffoldedPolicyShellRuleWorks drives the gate the
// scaffold actually ships: a run_shell call must produce an
// `ask` decision with a fully substituted approval payload,
// not a template error.
func TestInit_ScaffoldedPolicyShellRuleWorks(t *testing.T) {
	dir := t.TempDir()
	if _, _, err := Init(dir); err != nil {
		t.Fatal(err)
	}
	gw := policy.New(mustLoadPolicy(t, filepath.Join(dir, "policy.yaml")), nil)
		d, err := gw.Decide(policy.Call{
		TaskID: "t1",
		Tool:   "run_shell",
		Actor:  "bus-worker",
		Args:   map[string]any{"command": "echo,hello"},
	})
	if err != nil {
		t.Fatalf("decide: %v", err)
	}
	if d.Action != policy.ActionAsk {
		t.Fatalf("action = %q, want %q", d.Action, policy.ActionAsk)
	}
	if d.Approval == nil {
		t.Fatal("ask decision has no approval payload")
	}
	if !strings.Contains(d.Approval.Target, "echo") {
		t.Errorf("target = %q, want the substituted command", d.Approval.Target)
	}
}

func mustLoadPolicy(t *testing.T, path string) []policy.Rule {
	t.Helper()
	rules, err := policy.Load(path)
	if err != nil {
		t.Fatalf("load %s: %v", path, err)
	}
	return rules
}