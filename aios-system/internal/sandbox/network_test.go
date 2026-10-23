package sandbox

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNetworkPolicy_ProxyModeSetsProxyVars covers the proxy
// mode: the child must be pointed at the proxy, and the
// uppercase and lowercase variable names must both be set
// because different clients read different ones.
func TestNetworkPolicy_ProxyModeSetsProxyVars(t *testing.T) {
	p := &NetworkPolicy{Mode: NetworkModeProxy, ProxyURL: "http://127.0.0.1:9999"}
	env := p.Apply(map[string]string{"FOO": "bar"})

	if env["HTTP_PROXY"] != "http://127.0.0.1:9999" {
		t.Fatalf("HTTP_PROXY = %q", env["HTTP_PROXY"])
	}
	if env["http_proxy"] != "http://127.0.0.1:9999" {
		t.Fatalf("http_proxy = %q", env["http_proxy"])
	}
	if env["HTTPS_PROXY"] != "http://127.0.0.1:9999" {
		t.Fatalf("HTTPS_PROXY = %q", env["HTTPS_PROXY"])
	}
	if env["https_proxy"] != "http://127.0.0.1:9999" {
		t.Fatalf("https_proxy = %q", env["https_proxy"])
	}
	// ALL_PROXY catches the clients that do not distinguish
	// http from https.
	if env["ALL_PROXY"] != "http://127.0.0.1:9999" {
		t.Fatalf("ALL_PROXY = %q", env["ALL_PROXY"])
	}
	if env["FOO"] != "bar" {
		t.Fatalf("unrelated variable dropped: %v", env)
	}
}

// TestNetworkPolicy_ProxyModeRemovesInheritedProxy is the
// regression test for the escape hatch: a parent process that
// already has HTTPS_PROXY set must not be able to steer the
// child to a different proxy. The policy's proxy wins and the
// inherited one is gone.
func TestNetworkPolicy_ProxyModeRemovesInheritedProxy(t *testing.T) {
	p := &NetworkPolicy{Mode: NetworkModeProxy, ProxyURL: "http://127.0.0.1:9999"}
	env := p.Apply(map[string]string{
		"HTTPS_PROXY": "http://attacker.example:8080",
		"http_proxy":  "http://attacker.example:8080",
		"ALL_PROXY":   "http://attacker.example:8080",
		"NO_PROXY":    "attacker.example",
	})

	for _, name := range []string{"HTTPS_PROXY", "http_proxy", "ALL_PROXY", "NO_PROXY", "no_proxy"} {
		if v, ok := env[name]; ok && strings.Contains(v, "attacker") {
			t.Fatalf("%s survived: %q", name, v)
		}
	}
	if env["HTTPS_PROXY"] != "http://127.0.0.1:9999" {
		t.Fatalf("HTTPS_PROXY = %q", env["HTTPS_PROXY"])
	}
}

// TestNetworkPolicy_DenyAllBlocksEverything covers the deny
// mode. The child must get a dead proxy and NO_PROXY="*", and
// the inherited proxy must not survive.
func TestNetworkPolicy_DenyAllBlocksEverything(t *testing.T) {
	p := &NetworkPolicy{Mode: NetworkModeDenyAll, ProxyURL: "http://127.0.0.1:9999"}
	env := p.Apply(map[string]string{"HTTPS_PROXY": "http://attacker.example:8080"})

	if env["NO_PROXY"] != "*" {
		t.Fatalf("NO_PROXY = %q, want *", env["NO_PROXY"])
	}
	if env["no_proxy"] != "*" {
		t.Fatalf("no_proxy = %q, want *", env["no_proxy"])
	}
	if strings.Contains(env["HTTPS_PROXY"], "attacker") {
		t.Fatalf("inherited proxy survived: %q", env["HTTPS_PROXY"])
	}
	if env["HTTP_PROXY"] != blackHoleProxy {
		t.Fatalf("HTTP_PROXY = %q, want %q", env["HTTP_PROXY"], blackHoleProxy)
	}
}

// TestNetworkPolicy_NoneModeLeavesEnvAlone covers the no-policy
// case: the environment must come back unchanged, so a caller
// that never opted in is not silently changed.
func TestNetworkPolicy_NoneModeLeavesEnvAlone(t *testing.T) {
	in := map[string]string{"HTTP_PROXY": "http://keep.example:8080", "FOO": "bar"}
	p := &NetworkPolicy{Mode: NetworkModeNone}
	out := p.Apply(in)

	if out["HTTP_PROXY"] != "http://keep.example:8080" {
		t.Fatalf("HTTP_PROXY = %q", out["HTTP_PROXY"])
	}
	if out["FOO"] != "bar" {
		t.Fatalf("FOO = %q", out["FOO"])
	}
}

// TestNetworkPolicy_Validate covers the validation rules.
func TestNetworkPolicy_Validate(t *testing.T) {
	cases := []struct {
		name   string
		policy *NetworkPolicy
		wantOK bool
	}{
		{"nil", nil, true},
		{"none", &NetworkPolicy{Mode: NetworkModeNone}, true},
		{"proxy ok", &NetworkPolicy{Mode: NetworkModeProxy, ProxyURL: "http://127.0.0.1:1"}, true},
		{"deny ok", &NetworkPolicy{Mode: NetworkModeDenyAll, ProxyURL: "http://127.0.0.1:1"}, true},
		{"deny without url", &NetworkPolicy{Mode: NetworkModeDenyAll}, true},
		{"unknown mode", &NetworkPolicy{Mode: "wat", ProxyURL: "http://127.0.0.1:1"}, false},
		{"missing url", &NetworkPolicy{Mode: NetworkModeProxy}, false},
		{"bad scheme", &NetworkPolicy{Mode: NetworkModeProxy, ProxyURL: "socks5://127.0.0.1:1"}, false},
		{"no host", &NetworkPolicy{Mode: NetworkModeProxy, ProxyURL: "http://"}, false},
	}
	for _, tc := range cases {
		err := tc.policy.Validate()
		if tc.wantOK && err != nil {
			t.Errorf("%s: unexpected error: %v", tc.name, err)
		}
		if !tc.wantOK && err == nil {
			t.Errorf("%s: expected an error", tc.name)
		}
	}
}

// TestRunner_AppliesNetworkPolicyToChild is the end-to-end
// check: a child that prints its own environment must see the
// proxy variables the policy set.
func TestRunner_AppliesNetworkPolicyToChild(t *testing.T) {
	r := NewSubprocessRunnerWithNetwork(&NetworkPolicy{
		Mode:     NetworkModeProxy,
		ProxyURL: "http://127.0.0.1:9999",
	})
	opts := withHelperEnv(RunOptions{Timeout: 10})
	res, err := runSelf(t, r, opts, "TestHelperProxyEnv")
	if err != nil {
		t.Fatalf("unexpected error: %v (stderr=%q)", err, res.Stderr)
	}
	if !strings.Contains(res.Stdout, "HTTP_PROXY=http://127.0.0.1:9999") {
		t.Fatalf("child did not see the policy proxy: %q", res.Stdout)
	}
}

// TestHelperProxyEnv is the env helper for the network policy
// tests. It prints the proxy variables the child received, one
// per line, so the parent test can assert on them.
func TestHelperProxyEnv(t *testing.T) {
	if !helperMode() {
		return
	}
	for _, name := range []string{"HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY", "NO_PROXY", "AIOS_PARENT_MARKER"} {
		fmt.Printf("%s=%s\n", name, os.Getenv(name))
	}
}

// TestRunner_InheritEnvStillGetsPolicy is the regression test for
// the bug where the policy replaced the environment instead of
// overlaying it: a caller that inherits the parent environment
// (the run_tests tool does) must keep its inherited variables
// AND get the policy's proxy.
func TestRunner_InheritEnvStillGetsPolicy(t *testing.T) {
	t.Setenv("AIOS_PARENT_MARKER", "present")
	r := NewSubprocessRunnerWithNetwork(&NetworkPolicy{
		Mode:     NetworkModeProxy,
		ProxyURL: "http://127.0.0.1:9999",
	})
	opts := withHelperEnv(RunOptions{Timeout: 10, InheritEnv: true})
	res, err := runSelf(t, r, opts, "TestHelperProxyEnv")
	if err != nil {
		t.Fatalf("unexpected error: %v (stderr=%q)", err, res.Stderr)
	}
	if !strings.Contains(res.Stdout, "AIOS_PARENT_MARKER=present") {
		t.Fatalf("inherited variable lost: %q", res.Stdout)
	}
	if !strings.Contains(res.Stdout, "HTTP_PROXY=http://127.0.0.1:9999") {
		t.Fatalf("policy proxy missing: %q", res.Stdout)
	}
}

// TestRunner_RejectsInvalidNetworkPolicy covers the validation
// branch: an invalid policy must fail before the child starts.
func TestRunner_RejectsInvalidNetworkPolicy(t *testing.T) {
	r := NewSubprocessRunnerWithNetwork(&NetworkPolicy{Mode: "nonsense", ProxyURL: "http://127.0.0.1:1"})
	opts := withHelperEnv(RunOptions{Timeout: 10})
	_, err := runSelf(t, r, opts, "TestHelperEcho")
	if err == nil {
		t.Fatal("expected an error for an invalid network policy")
	}
}

// TestDefaultRunner_RoundTrip covers the process-wide default
// runner seam: SetDefaultRunner returns the previous value, and
// restoring it puts the process back where it was.
func TestDefaultRunner_RoundTrip(t *testing.T) {
	original := DefaultRunner()
	defer SetDefaultRunner(original)

	fake := &fakeRunner{}
	old := SetDefaultRunner(fake)
	if DefaultRunner() != fake {
		t.Fatal("default runner was not replaced")
	}
	if old != original {
		t.Fatal("SetDefaultRunner did not return the previous runner")
	}
	SetDefaultRunner(old)
	if DefaultRunner() != original {
		t.Fatal("default runner was not restored")
	}
}

// fakeRunner is a Runner that records the options it was given.
type fakeRunner struct {
	opts RunOptions
}

func (f *fakeRunner) Run(opts RunOptions) (SandboxResult, error) {
	f.opts = opts
	return SandboxResult{ExitCode: 0}, nil
}

// TestDefaultRunner_NilFallsBackToRealRunner covers the nil
// guard: a nil runner must not become a nil default that would
// panic at a call site.
func TestDefaultRunner_NilFallsBackToRealRunner(t *testing.T) {
	original := DefaultRunner()
	defer SetDefaultRunner(original)

	SetDefaultRunner(nil)
	if DefaultRunner() == nil {
		t.Fatal("default runner must never be nil")
	}
}

// TestEnvProxyURL covers the helper that reports a proxy from
// the environment.
func TestEnvProxyURL(t *testing.T) {
	t.Setenv("HTTP_PROXY", "")
	t.Setenv("http_proxy", "")
	t.Setenv("HTTPS_PROXY", "")
	t.Setenv("https_proxy", "")
	t.Setenv("ALL_PROXY", "")
	t.Setenv("all_proxy", "")
	if got := EnvProxyURL(); got != "" {
		t.Fatalf("expected no proxy, got %q", got)
	}

	t.Setenv("HTTPS_PROXY", "http://proxy.example:8080")
	if got := EnvProxyURL(); got != "http://proxy.example:8080" {
		t.Fatalf("EnvProxyURL = %q", got)
	}
}

// TestMergeEnv_OverrideWins covers the merge helper: the
// caller's explicit value wins over the inherited one.
func TestMergeEnv_OverrideWins(t *testing.T) {
	out := mergeEnv(
		map[string]string{"HOME": "/explicit"},
		[]string{"HOME=/inherited", "PATH=/bin"},
	)
	if out["HOME"] != "/explicit" {
		t.Fatalf("HOME = %q", out["HOME"])
	}
	if out["PATH"] != "/bin" {
		t.Fatalf("PATH = %q", out["PATH"])
	}
}

// TestMergeEnv_SkipsMalformedEntries covers the malformed-entry
// guard: an entry with no "=" is not a variable.
func TestMergeEnv_SkipsMalformedEntries(t *testing.T) {
	out := mergeEnv(nil, []string{"NOEQUALS", "=novalue", "GOOD=yes"})
	if len(out) != 1 {
		t.Fatalf("expected 1 entry, got %v", out)
	}
	if out["GOOD"] != "yes" {
		t.Fatalf("GOOD = %q", out["GOOD"])
	}
}

// TestNetworkPolicy_NoProxyExceptions covers the NoProxy field:
// when the caller lists exceptions they are set, and when it
// does not, NO_PROXY is absent rather than "*" (which would mean
// the opposite).
func TestNetworkPolicy_NoProxyExceptions(t *testing.T) {
	p := &NetworkPolicy{
		Mode:     NetworkModeProxy,
		ProxyURL: "http://127.0.0.1:9999",
		NoProxy:  []string{"localhost", "127.0.0.1"},
	}
	env := p.Apply(map[string]string{})
	if env["NO_PROXY"] != "localhost,127.0.0.1" {
		t.Fatalf("NO_PROXY = %q", env["NO_PROXY"])
	}
	if env["no_proxy"] != "localhost,127.0.0.1" {
		t.Fatalf("no_proxy = %q", env["no_proxy"])
	}

	p2 := &NetworkPolicy{Mode: NetworkModeProxy, ProxyURL: "http://127.0.0.1:9999"}
	env2 := p2.Apply(map[string]string{})
	if _, ok := env2["NO_PROXY"]; ok {
		t.Fatalf("NO_PROXY should be absent, got %q", env2["NO_PROXY"])
	}
}

// TestBlackHoleProxyIsUnreachable is a guard on the deny-mode
// constant: it must point at a port nothing listens on, so a
// denied child fails fast instead of hanging.
func TestBlackHoleProxyIsUnreachable(t *testing.T) {
	if !strings.Contains(blackHoleProxy, "127.0.0.1:1") {
		t.Fatalf("black hole proxy = %q", blackHoleProxy)
	}
	if _, err := os.Stat(filepath.Join(os.TempDir(), "never")); err == nil {
		t.Fatal("unreachable")
	}
}
