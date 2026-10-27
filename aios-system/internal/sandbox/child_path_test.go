package sandbox

import (
	"os/exec"
	"runtime"
	"strings"
	"testing"
)

// TestChildEnv_CanFindProgramByName is the regression guard for
// the "empty environment" bug.
//
// The sandbox cleared the child's environment to stop the parent's
// API keys leaking into a gated subprocess, and it went one step too
// far: it also removed the PATH. A child with no PATH cannot start
// any program it did not name by absolute path.
//
// On Windows the failure is quiet and confusing, because cmd.exe
// and PowerShell are built into their own shells: `cmd /c echo
// hello` works, while `cmd /c git --version` fails with "'git' is
// not recognized as an internal or external command". The plan the
// model wrote was correct and the product could not run it.
//
// The test asks the real question an operator hits: can a
// sandboxed child run a program by name? It runs the platform's own
// shell, because every machine has one and every shell resolves
// programs through PATH.
func TestChildEnv_CanFindProgramByName(t *testing.T) {
	if _, err := exec.LookPath("cmd"); err != nil {
		t.Skip("cmd is not available")
	}

	r := NewSubprocessRunner()
	opts := withHelperEnv(RunOptions{
		Command: []string{"cmd", "/c", "where", "cmd"},
		Timeout: 15,
	})
	res, err := r.Run(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v (stderr=%q)", err, res.Stderr)
	}
	if res.ExitCode != 0 {
		t.Fatalf("the child could not resolve a program by name: exit %d (stderr=%q)", res.ExitCode, res.Stderr)
	}
	if !strings.Contains(res.Stdout, "cmd") {
		t.Fatalf("the child did not report where it found the program: %q", res.Stdout)
	}
}

// TestChildEnv_PathIsPresent pins the mechanism behind the test
// above: the child gets a PATH.
//
// The outcome test above depends on the machine having a shell and
// a program to find. This one does not: it checks the variable
// itself, so a regression is reported as a missing PATH rather than
// as a shell-specific error.
func TestChildEnv_PathIsPresent(t *testing.T) {
	r := NewSubprocessRunner()
	opts := withHelperEnv(RunOptions{
		Command: []string{selfExec(t), "-test.run=^TestHelperPrintEnv$"},
	})
	opts.Env["AIOS_SANDBOX_PROBE"] = "PATH"

	res, err := r.Run(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v (stderr=%q)", err, res.Stderr)
	}
	if !strings.Contains(res.Stdout, "PATH=") {
		t.Fatalf("the child did not report a PATH: %q", res.Stdout)
	}
}

// TestChildEnv_CallerPathWins pins the precedence: a caller that
// sets its own PATH gets that one.
//
// The base PATH is a default, not an override. A caller that runs a
// program from a directory of its own choosing must be able to say
// so.
func TestChildEnv_CallerPathWins(t *testing.T) {
	r := NewSubprocessRunner()
	opts := withHelperEnv(RunOptions{
		Command: []string{selfExec(t), "-test.run=^TestHelperPrintEnv$"},
	})
	opts.Env["AIOS_SANDBOX_PROBE"] = "PATH"
	opts.Env["PATH"] = "/caller/chosen/path"

	res, err := r.Run(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v (stderr=%q)", err, res.Stderr)
	}
	if !strings.Contains(res.Stdout, "PATH=/caller/chosen/path") {
		t.Fatalf("the caller's PATH did not win: %q", res.Stdout)
	}
}

// TestChildEnv_SecretsStillDoNotLeak is the other half of the
// contract, and the reason the PATH was missing in the first place.
//
// Giving the child a PATH must not give it the operator's
// credentials. The test sets a secret in the parent's environment
// and asks the child to report it.
func TestChildEnv_SecretsStillDoNotLeak(t *testing.T) {
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
	if strings.Contains(res.Stdout, "sk-or-super-secret-value") {
		t.Fatalf("the parent's secret leaked into the child: %q", res.Stdout)
	}
}

// TestChildBaseEnv_OnlyLocations pins what the base environment
// may contain.
//
// Every variable in it names a location -- a directory to search, a
// directory to write to, a directory to read from. None of them is
// a credential, and the list is checked by name so a future change
// that adds an API key is caught here rather than in a breach.
func TestChildBaseEnv_OnlyLocations(t *testing.T) {
	allowed := map[string]bool{
		EnvPath:        true,
		EnvSystemRoot:  true,
		EnvComSpec:     true,
		EnvPathext:     true,
		EnvTemp:        true,
		EnvTmp:         true,
		EnvTmpDir:      true,
		EnvHome:        true,
		EnvUserProfile: true,
	}
	for name := range childBaseEnv() {
		if !allowed[name] {
			t.Errorf("the base child environment carries %q, which is not a location", name)
		}
	}
}

// TestChildBaseEnv_NoProxyVariables pins that the base environment
// carries no proxy settings.
//
// The network policy is what decides how a child reaches the
// network, and it is applied after this function. A proxy variable
// arriving from here would be a way around the policy, so the base
// environment must not carry one.
func TestChildBaseEnv_NoProxyVariables(t *testing.T) {
	for name := range childBaseEnv() {
		if isProxyVar(name) {
			t.Errorf("the base child environment carries the proxy variable %q", name)
		}
	}
}

// TestChildBaseEnv_PosixHasNoWindowsVariables pins that the
// Windows-only variables do not leak onto POSIX.
//
// The child on Linux is a different program with different
// requirements. Giving it SystemRoot and ComSpec would be noise,
// and a variable that names a Windows path is a small lie.
func TestChildBaseEnv_PosixHasNoWindowsVariables(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("this is the POSIX case")
	}
	env := childBaseEnv()
	for _, name := range []string{EnvSystemRoot, EnvComSpec, EnvPathext, EnvUserProfile} {
		if _, ok := env[name]; ok {
			t.Errorf("the POSIX child environment carries the Windows variable %q", name)
		}
	}
}
