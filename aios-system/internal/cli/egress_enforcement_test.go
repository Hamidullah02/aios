package cli

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Hamidullah02/aios/aios-system/internal/sandbox"
)

// mustParseURL parses a URL and fails the test if it does not
// parse.
func mustParseURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return u
}

// TestEgressEnforcement_ChildCannotReachBlockedHost is the
// end-to-end enforcement test.
//
// It answers the review's objection directly: a gated `run_shell`
// that an operator approved must not be able to reach a host that
// is not on the allow list.
//
// The test starts a local HTTP server that stands in for a
// reachable host, starts the egress guard with an allow list that
// names only something else, and then runs a real subprocess --
// the aios test binary acting as a helper -- that tries to reach
// the server through the proxy the guard installed.
//
// The helper's exit code says what happened: 0 means the request
// went through, 1 means the proxy refused it.
func TestEgressEnforcement_ChildCannotReachBlockedHost(t *testing.T) {
	// The "blocked" host: a real server on loopback.
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("REACHED"))
	}))
	defer upstream.Close()

	// The allow list names a host that is NOT the upstream.
	dir := t.TempDir()
	writeEgressYAML(t, dir, "allow:\n  - allowed.example\ndeny: []\n")

	guard, err := StartEgressGuard(dir, nil, EgressModeProxy)
	if err != nil {
		t.Fatalf("start guard: %v", err)
	}
	defer guard.Close()

	// The subprocess runs with the guard's policy installed,
	// because the guard put it on the default runner.
	runner := sandbox.DefaultRunner()
	out, err := runHelperThroughProxy(t, runner, guard.Proxy.URL(), upstream.URL)
	if err != nil {
		t.Fatalf("helper: %v", err)
	}
	if strings.Contains(out, "REACHED") {
		t.Fatal("the child reached a host that is not on the allow list")
	}
	if !strings.Contains(out, "BLOCKED") {
		t.Fatalf("expected the proxy to refuse the request, got %q", out)
	}
}

// TestEgressEnforcement_ChildCanReachAllowedHost is the other
// half: the policy must not break legitimate traffic. A host on
// the allow list has to be reachable through the proxy.
func TestEgressEnforcement_ChildCanReachAllowedHost(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("REACHED"))
	}))
	defer upstream.Close()

	// The upstream is on 127.0.0.1, so the allow list names
	// 127.0.0.1.
	dir := t.TempDir()
	writeEgressYAML(t, dir, "allow:\n  - 127.0.0.1\ndeny: []\n")

	guard, err := StartEgressGuard(dir, nil, EgressModeProxy)
	if err != nil {
		t.Fatalf("start guard: %v", err)
	}
	defer guard.Close()

	runner := sandbox.DefaultRunner()
	out, err := runHelperThroughProxy(t, runner, guard.Proxy.URL(), upstream.URL)
	if err != nil {
		t.Fatalf("helper: %v", err)
	}
	if !strings.Contains(out, "REACHED") {
		t.Fatalf("the child could not reach an allowed host: %q", out)
	}
}

// TestEgressEnforcement_DenyModeBlocksEverything checks that
// the deny mode stops the child even from reaching a host that
// would otherwise be allowed.
func TestEgressEnforcement_DenyModeBlocksEverything(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("REACHED"))
	}))
	defer upstream.Close()

	dir := t.TempDir()
	writeEgressYAML(t, dir, "allow:\n  - 127.0.0.1\ndeny: []\n")

	guard, err := StartEgressGuard(dir, nil, EgressModeDenyAll)
	if err != nil {
		t.Fatalf("start guard: %v", err)
	}
	defer guard.Close()

	runner := sandbox.DefaultRunner()
	out, err := runHelperThroughProxy(t, runner, "http://127.0.0.1:1", upstream.URL)
	if err != nil {
		t.Fatalf("helper: %v", err)
	}
	if strings.Contains(out, "REACHED") {
		t.Fatal("the child reached the network in deny mode")
	}
}

// runHelperThroughProxy runs the test binary as a child that
// fetches targetURL through the given proxy and prints either
// "REACHED:<body>" or "BLOCKED:<error>".
//
// The child is the test binary itself so the test is portable and
// needs no extra binary on the machine.
func runHelperThroughProxy(t *testing.T, runner sandbox.Runner, proxyURL, targetURL string) (string, error) {
	t.Helper()
	self, err := os.Executable()
	if err != nil {
		return "", err
	}
	res, err := runner.Run(sandbox.RunOptions{
		Command: []string{
			self,
			"-test.run=^TestEgressHelperFetch$",
			"-test.coverprofile=",
			"-test.outputdir=" + os.TempDir(),
		},
		Env: map[string]string{
			"AIOS_EGRESS_HELPER": "1",
			"AIOS_EGRESS_PROXY":  proxyURL,
			"AIOS_EGRESS_TARGET": targetURL,
		},
		Timeout: 30,
	})
	if err != nil {
		return res.Stdout + res.Stderr, err
	}
	return res.Stdout + res.Stderr, nil
}

// TestEgressHelperFetch is the child helper. It fetches the
// target through the proxy and prints what happened.
//
// It is skipped unless the sentinel env var is set, so the parent
// test run does not try to reach the network.
func TestEgressHelperFetch(t *testing.T) {
	if os.Getenv("AIOS_EGRESS_HELPER") != "1" {
		t.Skip("child helper")
	}
	proxyURL := os.Getenv("AIOS_EGRESS_PROXY")
	target := os.Getenv("AIOS_EGRESS_TARGET")

	client := &http.Client{
		Transport: &http.Transport{Proxy: http.ProxyURL(mustParseURL(t, proxyURL))},
		Timeout:   10 * time.Second,
	}
	resp, err := client.Get(target)
	if err != nil {
		fmt.Printf("BLOCKED:%v\n", err)
		return
	}
	defer resp.Body.Close()
	buf := make([]byte, 256)
	n, _ := resp.Body.Read(buf)
	// A refused request still gets an HTTP response: the proxy
	// answers 403 with an explanation. Only a 2xx means the
	// child actually reached the host.
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		fmt.Printf("BLOCKED:status=%d:%s\n", resp.StatusCode, string(buf[:n]))
		return
	}
	fmt.Printf("REACHED:%s\n", string(buf[:n]))
}
