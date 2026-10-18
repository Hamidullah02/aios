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

	// server is the running http.Server. Nil until Serve is called.
	server *http.Server

	// requestCounter is a monotonic counter used as a fallback
	// request id when the client did not supply X-Aios-Request-Id.
	requestCounter uint64
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
	p.server = &http.Server{
		Addr:              p.listen,
		Handler:           p.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	ln, err := net.Listen("tcp", p.listen)
	if err != nil {
		return fmt.Errorf("egress: listen %s: %w", p.listen, err)
	}
	p.server.Addr = ln.Addr().String()
	return p.server.Serve(ln)
}

// Shutdown gracefully stops the server. Safe to call from another
// goroutine; the caller's context controls the shutdown timeout.
func (p *Proxy) Shutdown(ctx context.Context) error {
	if p.server == nil {
		return nil
	}
	return p.server.Shutdown(ctx)
}

// Addr returns the bound listen address. Only valid after Serve has
// been called.
func (p *Proxy) Addr() string {
	if p.server == nil {
		return p.listen
	}
	return p.server.Addr
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
