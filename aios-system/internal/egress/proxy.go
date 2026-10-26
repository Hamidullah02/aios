package egress

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	"github.com/sadi/aios-system/internal/core/audit"
	"github.com/sadi/aios-system/internal/core/clock"
)

// Proxy is the egress proxy server.
//
// Per REQ-EGS-PRX-001..004 the proxy:
//
//   - accepts HTTP requests on the configured listen address;
//   - forwards the request to the upstream when the target host is
//     in the allow list;
//   - returns 403 when the host is not in the allow list (or matches
//     the deny list).
//
// The proxy also applies the header rules (REQ-EGS-HDR-001..002) and
// writes an `egress_request` audit row for every request with the URL,
// method, status, and duration (REQ-EGS-AUD-001).
type Proxy struct {
	allow   *AllowList
	headers *HeaderRules
	audit   *audit.Log
	clock   clock.Deadline

	// transport is the http.RoundTripper used to forward requests.
	// Tests can inject a custom transport; production uses the
	// default with sane timeouts.
	transport http.RoundTripper

	// listen is the configured listen address (host:port).
	listen string

	// mu guards server.
	//
	// The server is published by Listen and read by Addr and
	// Shutdown, which run on different goroutines: Listen is
	// called by the goroutine that starts serving, while Shutdown
	// is called from wherever the owner decides to stop it. Without
	// the lock, Shutdown can read a half-written *http.Server and
	// call Shutdown on it, and Addr can read a torn pointer.
	mu sync.RWMutex

	// server is the running http.Server. Nil until Serve is called.
	server *http.Server

	// requestCounter is a monotonic counter used as a fallback
	// request id when the client did not supply X-Aios-Request-Id.
	requestCounter uint64
}

// serverRef returns the running server under the lock.
//
// Every read of p.server goes through here. The returned pointer is
// stable once set, so the caller may use it after the lock is
// released: Listen replaces the pointer only by assigning a brand
// new server, and that happens once, before any reader exists.
func (p *Proxy) serverRef() *http.Server {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.server
}

// setServer publishes the server under the lock.
func (p *Proxy) setServer(s *http.Server) {
	p.mu.Lock()
	p.server = s
	p.mu.Unlock()
}

// serveConnect handles an HTTP CONNECT tunnel.
//
// The allow-list check happens on the CONNECT target, before a
// single byte of the tunneled stream is forwarded. That is the
// point of the method: once the tunnel is established the proxy
// cannot see inside it, so the decision has to be made on the
// authority the client asked for, and a client that asks for a
// host outside the list never gets a socket.
//
// The tunneled bytes are not inspected. The proxy enforces WHICH
// host the client may reach, not what the client says to it
// once it gets there. A host in the allow-list is a host the
// operator trusted; the traffic to it is the operator's problem
// to scope, and TLS is what keeps the proxy from being in that
// position at all.
func (p *Proxy) serveConnect(w http.ResponseWriter, r *http.Request, requestID string, start time.Time) {
	host := connectTarget(r)
	if host == "" {
		p.writeAuditRow(requestID, r.Method, "", http.StatusBadRequest, start, "bad_request", "malformed CONNECT target")
		http.Error(w, "bad request: malformed CONNECT target", http.StatusBadRequest)
		return
	}

	if !p.allow.Allows(host) {
		p.writeAuditRow(requestID, r.Method, host, http.StatusForbidden, start, "host_not_allowed", "")
		http.Error(w, "host not in allow list: "+host, http.StatusForbidden)
		return
	}

	// The client asked for a specific port. An authority of
	// "example.com" with no port means 443 for CONNECT, which
	// is the only scheme that uses this method.
	upstreamHost, port, err := net.SplitHostPort(host)
	if err != nil {
		upstreamHost = host
		port = "443"
	}

	// Dial the upstream directly. The transport's proxy setting
	// is deliberately nil so this cannot loop back into the
	// proxy.
	dialer := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
	upstream, err := dialer.DialContext(r.Context(), "tcp", net.JoinHostPort(upstreamHost, port))
	if err != nil {
		p.writeAuditRow(requestID, r.Method, host, http.StatusBadGateway, start, "upstream_error", err.Error())
		http.Error(w, "upstream error: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer upstream.Close()

	// A CONNECT needs a hijacked connection: the proxy stops
	// being an HTTP server for this socket and becomes a pipe.
	hj, ok := w.(http.Hijacker)
	if !ok {
		p.writeAuditRow(requestID, r.Method, host, http.StatusInternalServerError, start, "hijack_unsupported", "")
		http.Error(w, "proxy: connection hijacking unsupported", http.StatusInternalServerError)
		return
	}
	client, _, err := hj.Hijack()
	if err != nil {
		p.writeAuditRow(requestID, r.Method, host, http.StatusInternalServerError, start, "hijack_failed", err.Error())
		return
	}
	defer client.Close()

	// Answer 200 before the first byte of the tunnel is copied.
	// A client that receives anything else treats the tunnel as
	// failed and does not send the TLS handshake.
	if _, err := client.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n")); err != nil {
		p.writeAuditRow(requestID, r.Method, host, http.StatusBadGateway, start, "hijack_write_failed", err.Error())
		return
	}

	p.writeAuditRow(requestID, r.Method, host, http.StatusOK, start, "tunnel_established", "")

	// Copy in both directions. The first error to end either
	// direction tears the tunnel down, which is what a client
	// closing the connection looks like.
	errc := make(chan error, 2)
	go func() {
		_, err := io.Copy(upstream, client)
		errc <- err
	}()
	go func() {
		_, err := io.Copy(client, upstream)
		errc <- err
	}()
	<-errc
}

// connectTarget returns the host a CONNECT request asked for.
//
// RFC 7231 §4.3.6 defines the request target of a CONNECT as
// the authority form ("host:port"). Go's server puts that in
// r.Host, not r.RequestURI, so both are consulted: a hand-rolled
// client may send the authority on the request line.
func connectTarget(r *http.Request) string {
	if r.Host != "" {
		return strings.TrimSpace(r.Host)
	}
	return strings.TrimSpace(r.RequestURI)
}

// Config controls Proxy construction.
type Config struct {
	Allow     *AllowList
	Headers   *HeaderRules
	Audit     *audit.Log
	Listen    string
	Clock     clock.Deadline
	Transport http.RoundTripper
	// ForwardClientIP, if true, sets X-Forwarded-For on the
	// upstream request. Defaults to false: the proxy is on a
	// per-task sandbox and the client IP is localhost; the
	// upstream has no use for it.
	ForwardClientIP bool
}

// New constructs a Proxy. The allow list and header rules are
// required (a nil allow list denies everything).
func New(cfg Config) (*Proxy, error) {
	if cfg.Allow == nil {
		return nil, &ParseError{Reason: "allow list is required"}
	}
	if cfg.Listen == "" {
		cfg.Listen = "127.0.0.1:0" // tests can leave it zero; we resolve at Serve time
	}
	if cfg.Clock == nil {
		cfg.Clock = clock.SystemDeadline{}
	}
	if cfg.Transport == nil {
		cfg.Transport = &http.Transport{
			Proxy: nil, // do not honour the system's HTTP_PROXY
			DialContext: (&net.Dialer{
				Timeout:   10 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: 30 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			IdleConnTimeout:       90 * time.Second,
		}
	}
	return &Proxy{
		allow:     cfg.Allow,
		headers:   cfg.Headers,
		audit:     cfg.Audit,
		clock:     cfg.Clock,
		transport: cfg.Transport,
		listen:    cfg.Listen,
	}, nil
}

// Hash returns a content hash of the allow list and header rules.
// It is included in every audit row so a reviewer can confirm what
// policy was active at decision time.
func (p *Proxy) Hash() string {
	if p == nil {
		return ""
	}
	return p.allow.Hash() + ":" + p.headers.Hash()
}

// Handler returns the http.Handler that serves the proxy. Tests use
// this to exercise the proxy without binding a real port.
func (p *Proxy) Handler() http.Handler {
	return http.HandlerFunc(p.serve)
}

func (p *Proxy) serve(w http.ResponseWriter, r *http.Request) {
	start := p.clock.Now()
	requestID := p.requestID(r)

	// CONNECT is the tunnel method. It is handled separately
	// because its request target is an authority ("host:port"),
	// not an absolute URL, and because the body of the
	// conversation after the handshake is opaque TLS.
	//
	// Without this, the proxy could only police plain HTTP:
	// every HTTPS client would send CONNECT, the proxy would
	// refuse it as a malformed URL, and the allow-list would
	// have no effect on TLS traffic at all.
	if r.Method == http.MethodConnect {
		p.serveConnect(w, r, requestID, start)
		return
	}

	targetURL, terr := extractTargetURL(r)
	if terr != nil {
		p.writeAuditRow(requestID, r.Method, r.RequestURI, http.StatusBadRequest, start, "bad_request", terr.Error())
		http.Error(w, "bad request: "+terr.Error(), http.StatusBadRequest)
		return
	}
	host := targetURL.Host

	// Allow check.
	if !p.allow.Allows(host) {
		p.writeAuditRow(requestID, r.Method, targetURL.String(), 403, start, "host_not_allowed", "")
		http.Error(w, "host not in allow list: "+host, http.StatusForbidden)
		return
	}

	// Build the upstream request.
	body, err := readAndRestoreBody(r)
	if err != nil {
		p.writeAuditRow(requestID, r.Method, targetURL.String(), http.StatusBadGateway, start, "body_read_error", err.Error())
		http.Error(w, "failed to read request body", http.StatusBadGateway)
		return
	}
	upstream, err := http.NewRequestWithContext(r.Context(), r.Method, targetURL.String(), body)
	if err != nil {
		p.writeAuditRow(requestID, r.Method, targetURL.String(), http.StatusBadGateway, start, "request_build_error", err.Error())
		http.Error(w, "failed to build upstream request", http.StatusBadGateway)
		return
	}
	// Copy headers from the client.
	for k, v := range r.Header {
		// Skip hop-by-hop headers; net/http.Transport removes the
		// standard ones but we strip Connection-listed ones too.
		if isHopByHop(k) {
			continue
		}
		for _, val := range v {
			upstream.Header.Add(k, val)
		}
	}
	// Apply the proxy's own header rules on the request.
	p.headers.ApplyToRequest(upstream.Header)

	resp, err := p.transport.RoundTrip(upstream)
	if err != nil {
		p.writeAuditRow(requestID, r.Method, targetURL.String(), http.StatusBadGateway, start, "upstream_error", err.Error())
		http.Error(w, "upstream error: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Apply header rules to the response.
	respHeader := resp.Header.Clone()
	p.headers.ApplyToResponse(respHeader)
	for k, v := range respHeader {
		if isHopByHop(k) {
			continue
		}
		for _, val := range v {
			w.Header().Add(k, val)
		}
	}
	w.WriteHeader(resp.StatusCode)
	written, _ := io.Copy(w, resp.Body)

	p.writeAuditRow(requestID, r.Method, targetURL.String(), resp.StatusCode, start, "ok",
		strconv.FormatInt(written, 10))
}

// Serve binds the proxy to Config.Listen and starts serving. It
// blocks until the server exits. Use Shutdown to stop it from
// another goroutine.
func (p *Proxy) Serve() error {
	ln, err := p.Listen()
	if err != nil {
		return err
	}
	return p.serverRef().Serve(ln)
}

// ServeListener serves on an already-bound listener.
//
// It is the counterpart to Listen: the caller binds first, so it
// knows the address before the proxy starts accepting, and then
// serves on that same listener.
func (p *Proxy) ServeListener(ln net.Listener) error {
	srv := p.serverRef()
	if srv == nil {
		return fmt.Errorf("egress: serve: listener not bound")
	}
	return srv.Serve(ln)
}

// Listen binds the proxy's socket without serving on it.
//
// The split exists because the caller needs the bound address
// BEFORE the child can be pointed at it. The child is configured
// with the proxy URL in its environment, and that URL has to name
// the port the proxy actually bound to -- with a port of 0, the
// configured address is not the address the child can reach.
func (p *Proxy) Listen() (net.Listener, error) {
	ln, err := net.Listen("tcp", p.listen)
	if err != nil {
		return nil, fmt.Errorf("egress: listen %s: %w", p.listen, err)
	}
	p.setServer(&http.Server{
		Addr:              ln.Addr().String(),
		Handler:           p.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	})
	return ln, nil
}

// Shutdown gracefully stops the server. Safe to call from another
// goroutine; the caller's context controls the shutdown timeout.
func (p *Proxy) Shutdown(ctx context.Context) error {
	srv := p.serverRef()
	if srv == nil {
		return nil
	}
	return srv.Shutdown(ctx)
}

// Addr returns the bound listen address. Only valid after Serve has
// been called.
func (p *Proxy) Addr() string {
	srv := p.serverRef()
	if srv == nil {
		return p.listen
	}
	return srv.Addr
}

// URL returns the proxy URL a child should be pointed at.
//
// It is the address the proxy bound to, with the http scheme.
// A caller configures the child's environment with this, so the
// child and the proxy agree on the port even when the configured
// port was 0.
func (p *Proxy) URL() string {
	return "http://" + p.Addr()
}

func (p *Proxy) requestID(r *http.Request) string {
	if id := r.Header.Get("X-Aios-Request-Id"); id != "" {
		return id
	}
	n := atomic.AddUint64(&p.requestCounter, 1)
	return uuid.NewString() + "-" + strconv.FormatUint(n, 10)
}

// writeAuditRow writes the egress_request row. Failures are
// swallowed: the proxy is in the hot path and a denied audit
// write should not block the user. Per aios-core §11.12 the
// fail-closed contract is the caller's responsibility; here the
// caller is the proxy itself and the row is best-effort.
//
// If you want strict fail-closed semantics, wire the audit log in
// and check the returned error from a test; production deployments
// rely on the audit writer's own internal errors.
func (p *Proxy) writeAuditRow(requestID, method, urlStr string, status int, start time.Time, outcome, detail string) {
	if p.audit == nil {
		return
	}
	dur := p.clock.Now().Sub(start)
	row := &audit.Row{
		TaskID: requestID, // tasks are not the right grain for egress; use the request id
		Kind:   "egress_request",
		Actor:  "egress-proxy",
		Payload: map[string]any{
			"request_id":  requestID,
			"method":      method,
			"url":         urlStr,
			"status":      status,
			"duration_ms": dur.Milliseconds(),
			"outcome":     outcome,
			"detail":      detail,
			"policy_hash": p.Hash(),
		},
	}
	// We do not propagate the error: see the doc comment above.
	_, _ = p.audit.Write(row)
}

// isHopByHop reports whether a header is one of the standard
// connection-level headers that must not be forwarded. The list
// is from RFC 7230 §6.1.
func isHopByHop(name string) bool {
	switch http.CanonicalHeaderKey(name) {
	case "Connection",
		"Keep-Alive",
		"Proxy-Authenticate",
		"Proxy-Authorization",
		"Te",
		"Trailer",
		"Transfer-Encoding",
		"Upgrade":
		return true
	}
	return false
}

// readAndRestoreBody reads the request body into a buffer and
// returns a new io.ReadCloser that replays the buffer. The proxy
// needs to peek at the body to compute the request id; this helper
// also lets the upstream see the original bytes.
func readAndRestoreBody(r *http.Request) (io.ReadCloser, error) {
	if r.Body == nil {
		return http.NoBody, nil
	}
	data, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	_ = r.Body.Close()
	return io.NopCloser(bytes.NewReader(data)), nil
}

// urlString returns the full URL as a string, with empty fields
// left empty. Used for error reporting.
func urlString(u *url.URL) string {
	if u == nil {
		return ""
	}
	return u.String()
}

// extractTargetURL returns the absolute URL the client is
// targeting, in the form expected for the upstream request.
//
// Three request shapes arrive at the proxy, all per RFC 7230:
//
//  1. Forward proxy via curl / http.Client with Transport.Proxy set:
//
//     GET http://host/path?x=y HTTP/1.1
//     Host: <proxy-host>
//
//     In Go's net/http this becomes r.URL.Host = "<proxy-host>"
//     and r.URL.Path = "http://host/path?x=y" (the absolute form
//     stuffed into Path). We must look at r.URL.Path.
//
//  2. Direct call to the proxy URL (browsing to it as a web page,
//     or a sandbox whose DNS is misconfigured):
//
//     GET /path HTTP/1.1
//     Host: <proxy-host>
//
//     r.URL.Host is empty and r.URL.Path = "/path". Without an
//     absolute URL on the request line the target is ambiguous; we
//     refuse with 400.
//
//  3. Reverse-proxy / inline mode (out of scope for the doc set):
//     the proxy is set up in front of a single upstream and the
//     client uses a relative URL. We do not support this yet.
func extractTargetURL(r *http.Request) (*url.URL, error) {
	if r.URL == nil {
		return nil, &url.Error{Op: "extract", URL: r.RequestURI, Err: errEmptyURL}
	}
	// Case 1: absolute form on the request line. Go's http.Server
	// parses the absolute URL into r.URL (Host/Path/Scheme) and
	// also records the raw request-target in r.RequestURI.
	if r.RequestURI != "" && isAbsoluteRequestURI(r.RequestURI) {
		u, err := url.Parse(r.RequestURI)
		if err != nil {
			return nil, fmt.Errorf("absolute URL on request line: %w", err)
		}
		if u.Host == "" {
			return nil, errMalformedAbsolute
		}
		return u, nil
	}
	// Case 2: r.URL.Host populated but RequestURI is relative. This
	// happens when the client used Transport.Proxy with an http
	// (not https) target: the client sets the absolute URL on the
	// request line, but Go's transport may rewrite r.URL.Host to
	// the proxy on the server side. Fall back to r.URL.Host +
	// r.URL.Path + RawQuery.
	if r.URL.Host != "" {
		u := *r.URL
		if u.Scheme == "" {
			u.Scheme = "http"
		}
		return &u, nil
	}
	// Case 3: relative URL only. Without a target we cannot make
	// an allow-list decision; refuse.
	return nil, errRelativeURL
}

func isAbsoluteRequestURI(s string) bool {
	// Bare heuristic: starts with "http://" or "https://". A
	// more thorough check would scan for "://" and verify the
	// scheme is alphabetic; the proxy only handles those two
	// schemes, so this is enough.
	return len(s) >= 7 && (s[:7] == "http://" || (len(s) >= 8 && s[:8] == "https://"))
}

var (
	errRelativeURL       = errors.New("relative request URI; absolute URL required")
	errMalformedAbsolute = errors.New("malformed absolute URL on request line")
)
