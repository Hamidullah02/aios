package egress

import (
	"bufio"
	"context"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

// newEchoServer starts a plain TCP server that echoes back
// whatever it receives, prefixed with "echo:".
//
// It is not an HTTP server on purpose. A CONNECT tunnel carries
// opaque bytes, so the test has to prove the proxy pipes bytes
// through rather than reinterpreting them as HTTP.
func newEchoServer(t *testing.T) net.Listener {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				buf := make([]byte, 4096)
				for {
					n, err := c.Read(buf)
					if n > 0 {
						if _, werr := c.Write(append([]byte("echo:"), buf[:n]...)); werr != nil {
							return
						}
					}
					if err != nil {
						return
					}
				}
			}(conn)
		}
	}()
	return ln
}

// connectThrough opens a CONNECT tunnel to target through the
// proxy and returns the hijacked connection.
//
// It writes the CONNECT request by hand: Go's http.Client does
// not expose a way to open a CONNECT tunnel, and a real sandbox
// client (curl, pip, git) does exactly this.
func connectThrough(t *testing.T, proxyURL, target string) (net.Conn, *http.Response, error) {
	t.Helper()
	proxyAddr := strings.TrimPrefix(proxyURL, "http://")
	conn, err := net.DialTimeout("tcp", proxyAddr, 2*time.Second)
	if err != nil {
		return nil, nil, err
	}
	req := "CONNECT " + target + " HTTP/1.1\r\nHost: " + target + "\r\n\r\n"
	if _, err := conn.Write([]byte(req)); err != nil {
		conn.Close()
		return nil, nil, err
	}
	br := bufio.NewReader(conn)
	resp, err := http.ReadResponse(br, nil)
	if err != nil {
		conn.Close()
		return nil, nil, err
	}
	if resp.StatusCode != http.StatusOK {
		conn.Close()
		return nil, resp, nil
	}
	return conn, resp, nil
}

// TestProxy_CONNECTAllowedHostTunnels is the core test for the
// CONNECT path: a host on the allow list must get a working
// tunnel, and the bytes must survive intact.
func TestProxy_CONNECTAllowedHostTunnels(t *testing.T) {
	echo := newEchoServer(t)
	_, proxyURL := newProxyServer(t, NewAllowList([]string{"127.0.0.1"}, nil), nil, newTestLog(t))

	conn, resp, err := connectThrough(t, proxyURL, echo.Addr().String())
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	defer conn.Close()

	if _, err := conn.Write([]byte("ping")); err != nil {
		t.Fatalf("write: %v", err)
	}
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	buf := make([]byte, 64)
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if got := string(buf[:n]); got != "echo:ping" {
		t.Fatalf("tunnel payload = %q, want %q", got, "echo:ping")
	}
}

// TestProxy_CONNECTDeniedHostIsRefused is the enforcement test:
// a host that is NOT on the allow list must be refused with 403
// and no tunnel.
func TestProxy_CONNECTDeniedHostIsRefused(t *testing.T) {
	_, proxyURL := newProxyServer(t, NewAllowList([]string{"allowed.example"}, nil), nil, newTestLog(t))

	conn, resp, err := connectThrough(t, proxyURL, "blocked.example:443")
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	if conn != nil {
		conn.Close()
	}
	if resp == nil {
		t.Fatal("expected a response")
	}
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", resp.StatusCode)
	}
}

// TestProxy_CONNECTDenyListWins checks that a host on the deny
// list is refused even when it is also on the allow list.
func TestProxy_CONNECTDenyListWins(t *testing.T) {
	_, proxyURL := newProxyServer(t,
		NewAllowList([]string{"127.0.0.1"}, []string{"127.0.0.1"}),
		nil, newTestLog(t))

	conn, resp, err := connectThrough(t, proxyURL, "127.0.0.1:443")
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	if conn != nil {
		conn.Close()
	}
	if resp == nil || resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %v, want 403", resp)
	}
}

// TestProxy_CONNECTEmptyAllowListDenies is the fail-closed
// check: an empty allow list must refuse every CONNECT.
func TestProxy_CONNECTEmptyAllowListDenies(t *testing.T) {
	_, proxyURL := newProxyServer(t, NewAllowList(nil, nil), nil, newTestLog(t))

	conn, resp, err := connectThrough(t, proxyURL, "anything.example:443")
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	if conn != nil {
		conn.Close()
	}
	if resp == nil || resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %v, want 403", resp)
	}
}

// TestProxy_CONNECTGlobPatternMatches checks that a glob in the
// allow list matches a CONNECT target.
//
// The echo server listens on 127.0.0.1 with an ephemeral port,
// so the glob has to cover the port too. The allow list strips
// the port before matching, so a pattern of "127.0.0.*" covers
// every loopback address.
func TestProxy_CONNECTGlobPatternMatches(t *testing.T) {
	echo := newEchoServer(t)
	_, proxyURL := newProxyServer(t, NewAllowList([]string{"127.0.0.*"}, nil), nil, newTestLog(t))

	conn, resp, err := connectThrough(t, proxyURL, echo.Addr().String())
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	if resp == nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %v, want 200", resp)
	}
	if conn == nil {
		t.Fatal("expected a tunnel")
	}
	defer conn.Close()

	if _, err := conn.Write([]byte("ping")); err != nil {
		t.Fatalf("write: %v", err)
	}
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	buf := make([]byte, 64)
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if got := string(buf[:n]); got != "echo:ping" {
		t.Fatalf("tunnel payload = %q, want %q", got, "echo:ping")
	}
}

// TestProxy_CONNECTUpstreamUnreachableIsBadGateway checks the
// error path: an allowed host that cannot be dialled gets a 502
// rather than a hung connection.
func TestProxy_CONNECTUpstreamUnreachableIsBadGateway(t *testing.T) {
	// Port 1 is reserved; nothing listens there.
	_, proxyURL := newProxyServer(t, NewAllowList([]string{"127.0.0.1"}, nil), nil, newTestLog(t))

	conn, resp, err := connectThrough(t, proxyURL, "127.0.0.1:1")
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	if conn != nil {
		conn.Close()
	}
	if resp == nil || resp.StatusCode != http.StatusBadGateway {
		t.Fatalf("status = %v, want 502", resp)
	}
}

// TestProxy_URLReflectsBoundPort checks the helper the CLI uses
// to point a child at the proxy: the URL must name the port the
// proxy actually bound to, not the configured one.
func TestProxy_URLReflectsBoundPort(t *testing.T) {
	p, proxyURL := newProxyServer(t, NewAllowList([]string{"127.0.0.1"}, nil), nil, newTestLog(t))
	if p.URL() != proxyURL {
		t.Fatalf("URL() = %q, want %q", p.URL(), proxyURL)
	}
}

// TestProxy_ListenBindsBeforeServe covers the split between
// binding and serving: the caller needs the bound address before
// the proxy starts accepting.
func TestProxy_ListenBindsBeforeServe(t *testing.T) {
	p, err := New(Config{
		Allow:   NewAllowList([]string{"127.0.0.1"}, nil),
		Headers: NewHeaderRules(nil, nil),
		Audit:   newTestLog(t),
		Listen:  "127.0.0.1:0",
	})
	if err != nil {
		t.Fatal(err)
	}
	ln, err := p.Listen()
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()
	if p.Addr() == "127.0.0.1:0" {
		t.Fatalf("addr = %q, want a bound port", p.Addr())
	}
}

// TestProxy_CONNECTThroughRealHTTPClient checks the path a real
// sandbox client takes: an http.Client configured with the proxy
// as its Proxy setting, asked for an https URL. Go's client
// issues CONNECT for https targets.
func TestProxy_CONNECTThroughRealHTTPClient(t *testing.T) {
	// A TLS test server. The proxy must tunnel to it and the
	// client must complete the handshake through the tunnel.
	upstream := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "secure-ok")
	}))
	defer upstream.Close()

	host := strings.TrimPrefix(upstream.URL, "https://")
	_, proxyURL := newProxyServer(t, NewAllowList([]string{"127.0.0.1"}, nil), nil, newTestLog(t))

	pu, err := url.Parse(proxyURL)
	if err != nil {
		t.Fatal(err)
	}
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(pu),
			// The upstream uses a self-signed certificate, so
			// the TLS verification is skipped. This test is
			// about the tunnel, not the certificate.
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	resp, err := client.Get(upstream.URL)
	if err != nil {
		t.Fatalf("get through tunnel: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(body) != "secure-ok" {
		t.Fatalf("body = %q", body)
	}
	_ = host
}

// TestProxy_ShutdownStopsTheServer is the teardown check.
func TestProxy_ShutdownStopsTheServer(t *testing.T) {
	p, err := New(Config{
		Allow:   NewAllowList([]string{"127.0.0.1"}, nil),
		Headers: NewHeaderRules(nil, nil),
		Audit:   newTestLog(t),
		Listen:  "127.0.0.1:0",
	})
	if err != nil {
		t.Fatal(err)
	}
	errCh := make(chan error, 1)
	go func() { errCh <- p.Serve() }()
	deadline := time.Now().Add(2 * time.Second)
	for p.Addr() == "127.0.0.1:0" && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := p.Shutdown(ctx); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
	select {
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			t.Fatalf("serve returned %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("server did not stop")
	}
}
