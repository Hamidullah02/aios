package sandbox

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

// TestHelperPrintEnv is the env-inspection helper. It
// is a no-op when run as a normal test. When the test
// binary is invoked in helper mode (via selfExec +
// -test.run matching this test), it prints the value of
// the variables named in AIOS_SANDBOX_PROBE, one
// "NAME=VALUE" line per variable, then exits 0.
func TestHelperPrintEnv(t *testing.T) {
	if !helperMode() {
		return
	}
	for _, name := range strings.Split(os.Getenv("AIOS_SANDBOX_PROBE"), ",") {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		fmt.Printf("%s=%s\n", name, os.Getenv(name))
	}
}

// TestInheritEnv_DefaultIsolated asserts that the
// default (InheritEnv unset) keeps REQ-SND-ISO-002: a
// parent variable is NOT visible to the child.
//
// This is the regression guard for the run_shell tool,
// which must keep the isolated behaviour.
func TestInheritEnv_DefaultIsolated(t *testing.T) {
	t.Setenv("AIOS_SANDBOX_PARENT_PROBE", "parent-value")
	r := NewSubprocessRunner()
	opts := withHelperEnv(RunOptions{
		Command: []string{selfExec(t), "-test.run=^TestHelperPrintEnv$"},
	})
	opts.Env["AIOS_SANDBOX_PROBE"] = "AIOS_SANDBOX_PARENT_PROBE"
	res, err := r.Run(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v (stderr=%q)", err, res.Stderr)
	}
	if !strings.Contains(res.Stdout, "AIOS_SANDBOX_PARENT_PROBE=") {
		t.Fatalf("helper did not report the probe: %q", res.Stdout)
	}
	if strings.Contains(res.Stdout, "parent-value") {
		t.Fatalf("parent env leaked into the child: %q", res.Stdout)
	}
}

// TestInheritEnv_TrueSeesParentEnv asserts that
// InheritEnv=true makes the parent's variables visible
// to the child.
func TestInheritEnv_TrueSeesParentEnv(t *testing.T) {
	t.Setenv("AIOS_SANDBOX_PARENT_PROBE", "parent-value")
	r := NewSubprocessRunner()
	opts := withHelperEnv(RunOptions{
		Command:    []string{selfExec(t), "-test.run=^TestHelperPrintEnv$"},
		InheritEnv: true,
	})
	opts.Env["AIOS_SANDBOX_PROBE"] = "AIOS_SANDBOX_PARENT_PROBE"
	res, err := r.Run(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v (stderr=%q)", err, res.Stderr)
	}
	if !strings.Contains(res.Stdout, "AIOS_SANDBOX_PARENT_PROBE=parent-value") {
		t.Fatalf("parent env not inherited: %q", res.Stdout)
	}
}

// TestInheritEnv_OverlayWins asserts that when
// InheritEnv is true, an explicit Env entry overrides
// the parent's value for the same key. The child must
// see the overlay value, not the parent value.
func TestInheritEnv_OverlayWins(t *testing.T) {
	t.Setenv("AIOS_SANDBOX_OVERLAY_PROBE", "parent-value")
	r := NewSubprocessRunner()
	opts := withHelperEnv(RunOptions{
		Command:    []string{selfExec(t), "-test.run=^TestHelperPrintEnv$"},
		InheritEnv: true,
	})
	opts.Env["AIOS_SANDBOX_OVERLAY_PROBE"] = "overlay-value"
	opts.Env["AIOS_SANDBOX_PROBE"] = "AIOS_SANDBOX_OVERLAY_PROBE"
	res, err := r.Run(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v (stderr=%q)", err, res.Stderr)
	}
	if !strings.Contains(res.Stdout, "AIOS_SANDBOX_OVERLAY_PROBE=overlay-value") {
		t.Fatalf("overlay did not win: %q", res.Stdout)
	}
	if strings.Contains(res.Stdout, "=parent-value") {
		t.Fatalf("parent value still visible: %q", res.Stdout)
	}
}

// TestFormatEnv asserts the helper contract: nil /
// empty maps yield nil, non-empty maps yield
// "K=V" entries.
func TestFormatEnv(t *testing.T) {
	if got := formatEnv(nil); got != nil {
		t.Errorf("formatEnv(nil) = %v, want nil", got)
	}
	if got := formatEnv(map[string]string{}); got != nil {
		t.Errorf("formatEnv(empty) = %v, want nil", got)
	}
	got := formatEnv(map[string]string{"A": "1", "B": "2"})
	if len(got) != 2 {
		t.Fatalf("formatEnv: got %d entries, want 2", len(got))
	}
	joined := strings.Join(got, ",")
	if !strings.Contains(joined, "A=1") || !strings.Contains(joined, "B=2") {
		t.Errorf("formatEnv: got %q", joined)
	}
}
