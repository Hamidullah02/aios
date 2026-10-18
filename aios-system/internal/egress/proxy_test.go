package egress

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/sadi/aios-system/internal/core/audit"
	"github.com/sadi/aios-system/internal/core/clock"
)

// newTestLog opens an in-memory audit log.
func newTestLog(t *testing.T) *audit.Log {
	t.Helper()
	log, err := audit.Open(audit.Config{
		Path:  ":memory:",
		Clock: clock.FixedDecision{T: time.Unix(1700000000, 0)},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { log.Close() })
	return log
}

// upstreamWithState is the upstream handler the proxy forwards to.
type upstreamWithState struct {
	headers http.Header
	body    string
	status  int
}

func newUpstream(t *testing.T) (*httptest.Server, *upstreamWithState) {
	t.Helper()
	state := &upstreamWithState{
		headers: http.Header{},
		status:  http.StatusOK,
		body:    "upstream-ok",
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for k, v := range r.Header {
			state.headers[k] = append(state.headers[k], v...)
		}
		w.Header().Set("Server", "upstream-1.0")
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(state.status)
		_, _ = io.WriteString(w, state.body)
	}))
	t.Cleanup(srv.Close)
	return srv, state
}

// upstreamHost extracts "host:port" from an httptest URL.
func upstreamHost(srv *httptest.Server) string {
	return strings.TrimPrefix(srv.URL, "http://")
}

// newProxyServer builds the proxy and starts it on a real
// loopback port. The proxy uses its own *http.Server, not an
// httptest.NewServer wrapper, because the wrapper binds a second
// port and on Windows TIME_WAIT between tests causes flaky
// "address in use" errors.
func newProxyServer(t *testing.T, a *AllowList, h *HeaderRules, log *audit.Log) (*Proxy, string) {
	t.Helper()
	cfg := Config{Allow: a, Audit: log, Listen: "127.0.0.1:0"}
	if h != nil {
		cfg.Headers = h
	} else {
		cfg.Headers = NewHeaderRules(nil, nil)
	}
	p, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	errCh := make(chan error, 1)
	go func() { errCh <- p.Serve() }()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()
		_ = p.Shutdown(ctx)
		<-errCh
	})
	deadline := time.Now().Add(2 * time.Second)
	for p.Addr() == "127.0.0.1:0" && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	addr := p.Addr()
	if addr == "" || addr == "127.0.0.1:0" {
		t.Fatalf("proxy did not bind: %q", addr)
	}
	return p, "http://" + addr
}

// forwardClient is a small helper that issues requests via the
// proxy. The proxy URL is wired into the Transport as an HTTP
// proxy, which causes the Go client to send the absolute target
// URL on the request line — exactly what a real sandbox does.
func forwardClient(proxyURL string) *http.Client {
	pu, err := url.Parse(proxyURL)
	if err != nil {
		panic(err)
	}
	return &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(pu)}}
}

// getVia sends a GET to targetURL through proxyURL. It uses
// http.Transport.Proxy so the request line is the absolute URL.
func getVia(t *testing.T, proxyURL, targetURL string) (*http.Response, error) {
	return forwardClient(proxyURL).Get(targetURL)
}

// doVia sends a custom request through the proxy.
func doVia(t *testing.T, proxyURL, method, targetURL string, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequest(method, targetURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return forwardClient(proxyURL).Do(req)
}

func TestProxy_ForwardsToAllowedHost(t *testing.T) {
	upstream, _ := newUpstream(t)
	target := "http://" + upstreamHost(upstream) + "/v1/chat"
	_, proxyURL := newProxyServer(t, NewAllowList([]string{upstreamHost(upstream)}, nil), nil, newTestLog(t))

	resp, err := getVia(t, proxyURL, target)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: %d", resp.StatusCode)
		t.Fatalf("status: %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "upstream-ok" {
		t.Fatalf("body: %q", body)
	}
}

func TestProxy_DeniesDisallowedHost(t *testing.T) {
	log := newTestLog(t)
	_, proxyURL := newProxyServer(t, NewAllowList([]string{"only-this.com"}, nil), nil, log)

	target := "http://evil.com/x"
	resp, err := getVia(t, proxyURL, target)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status: %d, want 403", resp.StatusCode)
	}

	rows, err := log.GetByKind("egress_request", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("audit rows: %d", len(rows))
	}
	var pld map[string]any
	if err := json.Unmarshal(rows[0].Payload, &pld); err != nil {
		t.Fatal(err)
	}
	if pld["status"].(float64) != 403 {
		t.Fatalf("audit status: %v", pld["status"])
	}
	if pld["outcome"] != "host_not_allowed" {
		t.Fatalf("audit outcome: %v", pld["outcome"])
	}
}

func TestProxy_AppliesHeaderRules(t *testing.T) {
	upstream, st := newUpstream(t)
	host := upstreamHost(upstream)
	rules := NewHeaderRules(
		[]HeaderRule{{Name: "X-Internal"}},
		[]HeaderRule{{Name: "Authorization", Value: "Bearer proxy"}, {Name: "X-Aios", Value: "1"}},
	)
	_, proxyURL := newProxyServer(t, NewAllowList([]string{host}, nil), rules, newTestLog(t))

	target := "http://" + host + "/x"
	resp, err := doVia(t, proxyURL, "GET", target, map[string]string{
		"X-Internal":    "leak",
		"Authorization": "Bearer original",
	})
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if got := st.headers.Get("Authorization"); got != "Bearer proxy" {
		t.Errorf("upstream Authorization: %q", got)
	}
	if got := st.headers.Get("X-Internal"); got != "" {
		t.Errorf("X-Internal should be removed, got %q", got)
	}
	if got := st.headers.Get("X-Aios"); got != "1" {
		t.Errorf("X-Aios should be added, got %q", got)
	}
}

func TestProxy_DenyListWins(t *testing.T) {
	upstream, _ := newUpstream(t)
	host := upstreamHost(upstream)
	_, proxyURL := newProxyServer(t, NewAllowList([]string{"*"}, []string{host}), nil, newTestLog(t))

	target := "http://" + host + "/x"
	resp, err := getVia(t, proxyURL, target)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status: %d, want 403", resp.StatusCode)
	}
}

func TestProxy_AuditRowCarriesURLAndDuration(t *testing.T) {
	upstream, _ := newUpstream(t)
	host := upstreamHost(upstream)
	log := newTestLog(t)
	_, proxyURL := newProxyServer(t, NewAllowList([]string{host}, nil), nil, log)

	target := "http://" + host + "/v1/foo"
	resp, err := getVia(t, proxyURL, target)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	rows, _ := log.GetByKind("egress_request", 10)
	if len(rows) != 1 {
		t.Fatalf("rows: %d", len(rows))
	}
	var pld map[string]any
	_ = json.Unmarshal(rows[0].Payload, &pld)
	if pld["url"] == "" || pld["url"] == nil {
		t.Fatal("url missing")
	}
	if pld["method"] != "GET" {
		t.Fatalf("method: %v", pld["method"])
	}
	if pld["duration_ms"] == nil {
		t.Fatal("duration_ms missing")
	}
}

func TestProxy_StripsHopByHopHeaders(t *testing.T) {
	upstream, st := newUpstream(t)
	host := upstreamHost(upstream)
	_, proxyURL := newProxyServer(t, NewAllowList([]string{host}, nil), nil, newTestLog(t))

	target := "http://" + host + "/x"
	resp, err := doVia(t, proxyURL, "GET", target, map[string]string{
		"Connection": "close",
		"Keep-Alive": "300",
	})
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if st.headers.Get("Connection") != "" {
		t.Error("Connection should not be forwarded")
	}
	if st.headers.Get("Keep-Alive") != "" {
		t.Error("Keep-Alive should not be forwarded")
	}
}

func TestProxy_UpstreamErrorMappedToBadGateway(t *testing.T) {
	_, proxyURL := newProxyServer(t, NewAllowList([]string{"127.0.0.1"}, nil), nil, newTestLog(t))

	// Port 1 is reserved; nothing listens. The proxy should
	// dial, fail, and return 502.
	c := forwardClient(proxyURL)
	c.Timeout = 5 * time.Second
	target := "http://127.0.0.1:1/x"
	resp, err := c.Get(target)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadGateway {
		t.Fatalf("status: %d, want 502", resp.StatusCode)
	}
}

func TestProxy_NilAllowListRejected(t *testing.T) {
	_, err := New(Config{Allow: nil, Listen: "127.0.0.1:0"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestProxy_HashChangesWithRules(t *testing.T) {
	upstream, _ := newUpstream(t)
	host := upstreamHost(upstream)
	a1 := NewAllowList([]string{host}, nil)
	a2 := NewAllowList([]string{host}, []string{"other.com"})
	p1, _ := New(Config{Allow: a1, Listen: "127.0.0.1:0"})
	p2, _ := New(Config{Allow: a2, Listen: "127.0.0.1:0"})
	if p1.Hash() == p2.Hash() {
		t.Fatal("different allow lists should hash differently")
	}
}

func TestProxy_ServeAndShutdown(t *testing.T) {
	a := NewAllowList([]string{"x"}, nil)
	p, err := New(Config{Allow: a, Listen: "127.0.0.1:0"})
	if err != nil {
		t.Fatal(err)
	}
	errCh := make(chan error, 1)
	go func() { errCh <- p.Serve() }()
	deadline := time.Now().Add(2 * time.Second)
	for p.Addr() == "127.0.0.1:0" && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	addr := p.Addr()
	if addr == "127.0.0.1:0" || addr == "" {
		t.Fatalf("addr: %q", addr)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	if err := p.Shutdown(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("server did not shut down")
	}
}

func TestProxy_DefaultListenIsZero(t *testing.T) {
	a := NewAllowList([]string{"x"}, nil)
	p, err := New(Config{Allow: a})
	if err != nil {
		t.Fatal(err)
	}
	if p.listen != "127.0.0.1:0" {
		t.Fatalf("listen: %q", p.listen)
	}
}

func TestURLString(t *testing.T) {
	if urlString(nil) != "" {
		t.Fatal("nil url should produce empty")
	}
}
