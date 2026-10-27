package cli

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Hamidullah02/aios/aios-system/internal/egress"
	"github.com/Hamidullah02/aios/aios-system/internal/sandbox"
)

// writeEgressYAML writes an egress policy file into a
// workspace directory.
func writeEgressYAML(t *testing.T, dir, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "egress.yaml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestStartEgressGuard_ProxyModeInstallsPolicy is the core
// wiring test: the guard must start a proxy, install a policy on
// the default runner, and the policy must point at the proxy that
// is actually running.
func TestStartEgressGuard_ProxyModeInstallsPolicy(t *testing.T) {
	dir := t.TempDir()
	writeEgressYAML(t, dir, "allow:\n  - 127.0.0.1\ndeny: []\n")

	guard, err := StartEgressGuard(dir, nil, EgressModeProxy)
	if err != nil {
		t.Fatalf("start guard: %v", err)
	}
	defer guard.Close()

	if guard.Proxy == nil {
		t.Fatal("guard has no proxy")
	}
	if guard.Policy == nil {
		t.Fatal("guard has no policy")
	}
	if guard.Policy.Mode != sandbox.NetworkModeProxy {
		t.Fatalf("policy mode = %q", guard.Policy.Mode)
	}
	// The policy must name the port the proxy actually bound
	// to, not the configured one. A mismatch here is the bug
	// that would make every child fail to connect.
	if !strings.Contains(guard.Policy.ProxyURL, guard.Proxy.Addr()) {
		t.Fatalf("policy proxy %q does not match bound addr %q", guard.Policy.ProxyURL, guard.Proxy.Addr())
	}
}

// TestStartEgressGuard_InstallsDefaultRunner is the enforcement
// test: after the guard starts, the process-wide default runner
// must carry the policy, so every tool that falls back to the
// default is covered.
func TestStartEgressGuard_InstallsDefaultRunner(t *testing.T) {
	dir := t.TempDir()
	writeEgressYAML(t, dir, "allow:\n  - 127.0.0.1\ndeny: []\n")

	before := sandbox.DefaultRunner()
	guard, err := StartEgressGuard(dir, nil, EgressModeProxy)
	if err != nil {
		t.Fatalf("start guard: %v", err)
	}
	defer guard.Close()

	after, ok := sandbox.DefaultRunner().(*sandbox.SubprocessRunner)
	if !ok {
		t.Fatalf("default runner is %T, want *sandbox.SubprocessRunner", sandbox.DefaultRunner())
	}
	if after.Network == nil {
		t.Fatal("default runner has no network policy")
	}
	if after.Network.Mode != sandbox.NetworkModeProxy {
		t.Fatalf("default runner policy mode = %q", after.Network.Mode)
	}
	if before == nil {
		t.Fatal("no previous runner")
	}
}

// TestStartEgressGuard_CloseRestoresDefaultRunner is the
// teardown test: closing the guard must put the process back the
// way it found it.
func TestStartEgressGuard_CloseRestoresDefaultRunner(t *testing.T) {
	dir := t.TempDir()
	writeEgressYAML(t, dir, "allow:\n  - 127.0.0.1\ndeny: []\n")

	original := sandbox.DefaultRunner()
	guard, err := StartEgressGuard(dir, nil, EgressModeProxy)
	if err != nil {
		t.Fatalf("start guard: %v", err)
	}
	if sandbox.DefaultRunner() == original {
		t.Fatal("default runner was not replaced")
	}
	if err := guard.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	if sandbox.DefaultRunner() != original {
		t.Fatal("default runner was not restored")
	}
}

// TestStartEgressGuard_DenyModeInstallsDenyPolicy covers the deny
// mode: no proxy is started and the policy denies everything.
func TestStartEgressGuard_DenyModeInstallsDenyPolicy(t *testing.T) {
	dir := t.TempDir()

	original := sandbox.DefaultRunner()
	guard, err := StartEgressGuard(dir, nil, EgressModeDenyAll)
	if err != nil {
		t.Fatalf("start guard: %v", err)
	}
	defer guard.Close()

	if guard.Proxy != nil {
		t.Fatal("deny mode should not start a proxy")
	}
	runner, ok := sandbox.DefaultRunner().(*sandbox.SubprocessRunner)
	if !ok {
		t.Fatalf("default runner is %T", sandbox.DefaultRunner())
	}
	if runner.Network == nil || runner.Network.Mode != sandbox.NetworkModeDenyAll {
		t.Fatalf("default runner policy = %+v", runner.Network)
	}
	_ = original
}

// TestStartEgressGuard_OffModeInstallsNothing covers the off
// mode: no proxy, no policy, and the default runner is left
// alone.
func TestStartEgressGuard_OffModeInstallsNothing(t *testing.T) {
	original := sandbox.DefaultRunner()
	guard, err := StartEgressGuard(t.TempDir(), nil, EgressModeOff)
	if err != nil {
		t.Fatalf("start guard: %v", err)
	}
	defer guard.Close()

	if guard.Proxy != nil {
		t.Fatal("off mode should not start a proxy")
	}
	if sandbox.DefaultRunner() != original {
		t.Fatal("off mode changed the default runner")
	}
	if !strings.Contains(guard.Describe(), "off") {
		t.Fatalf("describe = %q", guard.Describe())
	}
}

// TestStartEgressGuard_UnknownModeFails covers the validation
// branch.
func TestStartEgressGuard_UnknownModeFails(t *testing.T) {
	_, err := StartEgressGuard(t.TempDir(), nil, "bogus")
	if err == nil {
		t.Fatal("expected an error for an unknown mode")
	}
}

// TestStartEgressGuard_MissingEgressFileFailsClosed is the
// fail-closed test: a workspace with no egress.yaml gets a
// proxy with an empty allow list, which denies everything. The
// guard must still start.
func TestStartEgressGuard_MissingEgressFileFailsClosed(t *testing.T) {
	dir := t.TempDir()
	guard, err := StartEgressGuard(dir, nil, EgressModeProxy)
	if err != nil {
		t.Fatalf("start guard: %v", err)
	}
	defer guard.Close()

	if guard.Allow == nil {
		t.Fatal("guard has no allow list")
	}
	if guard.Allow.AllowCount() != 0 {
		t.Fatalf("expected an empty allow list, got %d rules", guard.Allow.AllowCount())
	}
	if guard.Allow.Allows("example.com") {
		t.Fatal("an empty allow list must deny every host")
	}
}

// TestEgressGuard_ProxyEnforcesAllowList is the end-to-end
// enforcement test: a request through the guard's proxy to a
// host that is not on the allow list must be refused.
func TestEgressGuard_ProxyEnforcesAllowList(t *testing.T) {
	dir := t.TempDir()
	writeEgressYAML(t, dir, "allow:\n  - allowed.example\ndeny: []\n")

	guard, err := StartEgressGuard(dir, nil, EgressModeProxy)
	if err != nil {
		t.Fatalf("start guard: %v", err)
	}
	defer guard.Close()

	client := &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(httpProxyURL(t, guard.Proxy.URL()))}}
	resp, err := client.Get("http://blocked.example/")
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", resp.StatusCode)
	}
}

// TestEgressGuard_DescribeMentionsTheScope checks the operator
// facing summary.
func TestEgressGuard_DescribeMentionsTheScope(t *testing.T) {
	dir := t.TempDir()
	writeEgressYAML(t, dir, "allow:\n  - allowed.example\ndeny: []\n")

	guard, err := StartEgressGuard(dir, nil, EgressModeProxy)
	if err != nil {
		t.Fatalf("start guard: %v", err)
	}
	defer guard.Close()

	got := guard.Describe()
	if !strings.Contains(got, "proxy") {
		t.Fatalf("describe = %q", got)
	}
	if !strings.Contains(got, "1 host rule") {
		t.Fatalf("describe = %q", got)
	}
}

// TestEgressGuard_CloseIsIdempotent covers the double-close
// guard, which a deferred close plus an explicit close would hit.
func TestEgressGuard_CloseIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	writeEgressYAML(t, dir, "allow: []\ndeny: []\n")

	guard, err := StartEgressGuard(dir, nil, EgressModeProxy)
	if err != nil {
		t.Fatalf("start guard: %v", err)
	}
	if err := guard.Close(); err != nil {
		t.Fatalf("first close: %v", err)
	}
	if err := guard.Close(); err != nil {
		t.Fatalf("second close: %v", err)
	}
}

// TestEgressGuard_NilCloseIsSafe covers the nil guard.
func TestEgressGuard_NilCloseIsSafe(t *testing.T) {
	var guard *EgressGuard
	if err := guard.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	if !strings.Contains(guard.Describe(), "off") {
		t.Fatal("describe should report off for a nil guard")
	}
}

// TestEgressGuard_ProxyShutdownStopsListening checks that the
// guard's teardown actually closes the socket.
func TestEgressGuard_ProxyShutdownStopsListening(t *testing.T) {
	dir := t.TempDir()
	writeEgressYAML(t, dir, "allow: []\ndeny: []\n")

	guard, err := StartEgressGuard(dir, nil, EgressModeProxy)
	if err != nil {
		t.Fatalf("start guard: %v", err)
	}
	addr := guard.Proxy.Addr()
	if err := guard.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	// The socket must be gone. A connection attempt has to
	// fail rather than succeed.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+addr+"/", nil)
	if err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(httpProxyURL(t, guard.Proxy.URL()))}}
	resp, err := client.Do(req)
	if err == nil {
		resp.Body.Close()
		t.Fatal("the proxy is still serving after Close")
	}
}

// httpProxyURL parses a proxy URL for a test transport.
func httpProxyURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return u
}

// TestWorkspaceForArgs covers the workspace resolution the guard
// depends on.
func TestWorkspaceForArgs(t *testing.T) {
	explicit := &Args{Workspace: "C:/ws", ConfigPath: "C:/other/config.yaml"}
	got := workspaceForArgs(explicit)
	if !strings.Contains(got, "ws") {
		t.Fatalf("workspace = %q", got)
	}

	fromConfig := &Args{ConfigPath: filepath.Join(t.TempDir(), "config.yaml")}
	got2 := workspaceForArgs(fromConfig)
	if got2 == "" {
		t.Fatal("workspace is empty")
	}

	empty := &Args{}
	if workspaceForArgs(empty) == "" {
		t.Fatal("workspace is empty for a bare Args")
	}
	if workspaceForArgs(nil) == "" {
		t.Fatal("workspace is empty for a nil Args")
	}
}

// TestParser_EgressFlagHasADefault checks the flag default. The
// default has to be the enforcing mode: a run that does not ask
// for enforcement should still be enforced.
func TestParser_EgressFlagHasADefault(t *testing.T) {
	args, err := Parse([]string{"aios", "run", "--plan=x.yaml"}, io.Discard)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if args.EgressMode != EgressModeProxy {
		t.Fatalf("default egress mode = %q, want %q", args.EgressMode, EgressModeProxy)
	}
}

// TestParser_EgressFlagIsOverridable checks that the operator can
// turn enforcement off deliberately.
func TestParser_EgressFlagIsOverridable(t *testing.T) {
	args, err := Parse([]string{"aios", "--egress=off", "run", "--plan=x.yaml"}, io.Discard)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if args.EgressMode != EgressModeOff {
		t.Fatalf("egress mode = %q, want %q", args.EgressMode, EgressModeOff)
	}
}

// TestAllowCount covers the helper the operator summary uses.
func TestAllowCount(t *testing.T) {
	a := egress.NewAllowList([]string{"a.example", "b.example"}, nil)
	if a.AllowCount() != 2 {
		t.Fatalf("allow count = %d, want 2", a.AllowCount())
	}
	var nilList *egress.AllowList
	if nilList.AllowCount() != 0 {
		t.Fatalf("nil allow count = %d, want 0", nilList.AllowCount())
	}
}
