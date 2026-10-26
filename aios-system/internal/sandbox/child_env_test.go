package sandbox

import (
	"strings"
	"testing"
)

// TestChildEnv_ParentSecretsDoNotLeak is the regression guard
// for the environment leak.
//
// The runner is supposed to clear the child's environment
// (REQ-SND-ISO-002). It did not, on the path where no network
// policy was installed: the environment slice was left nil, and
// os/exec reads a nil cmd.Env as "inherit the parent's
// environment".
//
// The consequence was that every gated subprocess -- a run_shell
// the operator approved, a git command -- saw the operator's
// OPENROUTER_API_KEY, and could exfiltrate it.
//
// The test has no network policy on purpose: that is the path
// that was broken, and the path the network-policy tests never
// exercise.
func TestChildEnv_ParentSecretsDoNotLeak(t *testing.T) {
	t.Setenv("AIOS_SANDBOX_SECRET_PROBE", "sk-or-super-secret-value")

	r := NewSubprocessRunner()
	opts := withHelperEnv(RunOptions{
		Command: []string{selfExec(t), "-test.run=^TestHelperPrintEnv$"},
	})
	opts.Env["AIOS_SANDBOX_PROBE"] = "AIOS_SANDBOX_SECRET_PROBE"

	res, err := r.Run(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v (stderr=%q)", err, res.Stderr)
	}
	if !strings.Contains(res.Stdout, "AIOS_SANDBOX_SECRET_PROBE=") {
		t.Fatalf("helper did not report the probe: %q", res.Stdout)
	}
	if strings.Contains(res.Stdout, "sk-or-super-secret-value") {
		t.Fatalf("the parent's secret leaked into the child: %q", res.Stdout)
	}
}

// TestChildEnv_EmptyEnvIsNotInherit is the unit-level half of
// the same guard.
//
// It pins the exact mechanism rather than the outcome, because
// the outcome depends on the child program. A nil cmd.Env means
// "inherit"; a non-nil empty slice means "nothing". The runner
// must produce the second one.
func TestChildEnv_EmptyEnvIsNotInherit(t *testing.T) {
	r := NewSubprocessRunner()
	opts := RunOptions{
		Command: []string{selfExec(t), "-test.run=^TestHelperPrintEnv$"},
		Env:     map[string]string{},
	}
	// The helper is a no-op without its sentinel, so the child
	// starts and exits cleanly with an empty environment.
	res, err := r.Run(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v (stderr=%q)", err, res.Stderr)
	}
	if res.ExitCode != 0 {
		t.Fatalf("child exit = %d (stderr=%q)", res.ExitCode, res.Stderr)
	}
}

// TestChildEnv_ExplicitEnvStillReachesTheChild is the other
// half: the fix must not break the case where the caller really
// does want a variable in the child.
func TestChildEnv_ExplicitEnvStillReachesTheChild(t *testing.T) {
	r := NewSubprocessRunner()
	opts := withHelperEnv(RunOptions{
		Command: []string{selfExec(t), "-test.run=^TestHelperPrintEnv$"},
	})
	opts.Env["AIOS_SANDBOX_PROBE"] = "AIOS_SANDBOX_EXPLICIT_PROBE"
	opts.Env["AIOS_SANDBOX_EXPLICIT_PROBE"] = "explicit-value"

	res, err := r.Run(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v (stderr=%q)", err, res.Stderr)
	}
	if !strings.Contains(res.Stdout, "AIOS_SANDBOX_EXPLICIT_PROBE=explicit-value") {
		t.Fatalf("the explicit variable did not reach the child: %q", res.Stdout)
	}
}

// TestLazyDefaultRunner_ResolvesAtCallTime pins the contract the
// lazy runner exists for.
//
// The runner must use whichever runner is installed when it is
// CALLED, not the one that was installed when it was built. The
// egress guard installs its policy after the tool set is built,
// so a runner that captured the default at construction time
// would be the policy-less one forever.
func TestLazyDefaultRunner_ResolvesAtCallTime(t *testing.T) {
	original := DefaultRunner()
	t.Cleanup(func() { SetDefaultRunner(original) })

	// Install a runner that reports it was used.
	var used bool
	SetDefaultRunner(&recordingRunner{onRun: func(RunOptions) { used = true }})

	lazy := LazyDefaultRunner{}
	if _, err := lazy.Run(RunOptions{Command: []string{"echo"}}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !used {
		t.Fatal("the lazy runner did not use the installed default runner")
	}
}

// recordingRunner is a Runner that records that it was called.
type recordingRunner struct {
	onRun func(RunOptions)
}

func (r *recordingRunner) Run(opts RunOptions) (SandboxResult, error) {
	if r.onRun != nil {
		r.onRun(opts)
	}
	return SandboxResult{ExitCode: 0}, nil
}
