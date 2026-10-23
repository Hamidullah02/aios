package webui

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
)

// EnvToken is the environment variable an operator can set to pin
// the UI's session token.
//
// A pinned token lets a headless operator (or a test) authenticate
// without scraping the URL the server prints. The value is a
// secret: it grants full control of the UI, including approving
// tool calls, so it belongs in the environment of the process and
// nowhere else.
const EnvToken = "AIOS_WEB_TOKEN"

// SessionCookieName is the cookie carrying the session token.
//
// The cookie is what makes the UI work: an EventSource cannot set
// request headers, so the SSE stream has to authenticate with
// something the browser attaches on its own.
const SessionCookieName = "aios_session"

// TokenHeader is the header a non-browser client uses instead of the
// cookie.
//
// A CLI or a script authenticates with this; a browser uses the
// cookie. Both carry the same token, so a client can choose.
const TokenHeader = "X-Aios-Token"

// tokenParam is the query parameter that carries the token on the
// one request that mints the cookie.
//
// The token appears in a URL exactly once, when the operator opens
// the link the server printed. The client script strips it from the
// address bar afterwards so it does not linger in a copied link.
const tokenParam = "token"

// tokenBytes is how much randomness a session token carries.
//
// 32 bytes is 256 bits: the token is a capability, so guessing one
// must be computationally hopeless rather than merely unlikely.
const tokenBytes = 32

// NewSessionToken returns a fresh random session token.
//
// The token is URL-safe (base64 raw, no padding) because it travels
// in a query string.
func NewSessionToken() (string, error) {
	b := make([]byte, tokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// errNoToken is returned when a session token was required and
// none could be produced.
var errNoToken = errors.New("webui: no session token")

// auth holds the server's session token and the host names it
// answers to.
//
// It is the whole of the UI's authentication story, and it is
// deliberately small: one token for the process, checked on every
// request, plus the host and origin checks that stop a browser
// elsewhere on the machine from driving it.
type auth struct {
	// token is the session secret. Empty means "no token", which
	// is never a usable configuration: New fails rather than
	// serving an unauthenticated UI.
	token string
	// hosts is the set of host names the server answers to. An
	// empty map with anyHost set means "any host", which is the
	// wildcard-bind case.
	hosts map[string]bool
	// anyHost relaxes the host check for a wildcard bind. The
	// token and origin checks still apply, so a wildcard bind
	// weakens DNS-rebinding protection but not the session
	// boundary.
	anyHost bool
}

// newAuth builds the auth state for a server.
//
// addr is the configured listen address; allowedHosts names
// additional host names the operator wants to answer to.
func newAuth(token, addr string, allowedHosts []string) (*auth, error) {
	if token == "" {
		token = strings.TrimSpace(os.Getenv(EnvToken))
	}
	if token == "" {
		generated, err := NewSessionToken()
		if err != nil {
			return nil, err
		}
		token = generated
	}
	hosts := map[string]bool{
		"localhost": true,
		"127.0.0.1": true,
		"::1":       true,
	}
	anyHost := false
	bindHost := hostOnly(addr)
	switch bindHost {
	case "":
		// No configured address: the default is loopback, so the
		// loopback names above are the whole list.
	case "0.0.0.0", "::":
		// A wildcard bind answers on every interface, so the
		// host names it answers to are not knowable in advance.
		// The session token and the origin check remain in force.
		anyHost = true
	default:
		hosts[bindHost] = true
	}
	for _, h := range allowedHosts {
		h = strings.ToLower(strings.TrimSpace(h))
		if h == "" {
			continue
		}
		hosts[h] = true
	}
	return &auth{token: token, hosts: hosts, anyHost: anyHost}, nil
}

// hostOnly returns the host part of a host:port pair, lower-cased
// and without IPv6 brackets.
//
// An address with no port is returned as-is, because "localhost"
// and "127.0.0.1" are both valid host names on their own.
func hostOnly(addr string) string {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return ""
	}
	// The bracketed IPv6 form splits cleanly by hand;
	// net.SplitHostPort rejects "::1:7755" because it cannot
	// tell the address from the port without brackets.
	if strings.HasPrefix(addr, "[") {
		if i := strings.Index(addr, "]"); i > 0 {
			return strings.ToLower(addr[1:i])
		}
	}
	if h, _, err := net.SplitHostPort(addr); err == nil {
		return strings.ToLower(strings.Trim(h, "[]"))
	}
	return strings.ToLower(strings.Trim(addr, "[]"))
}

// validToken reports whether candidate is the session token.
//
// The comparison is constant-time so a caller cannot learn the
// token by timing how long the comparison took.
func (a *auth) validToken(candidate string) bool {
	if a == nil || a.token == "" || candidate == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a.token), []byte(candidate)) == 1
}

// authenticated reports whether the request carries the session
// token, in the cookie or in the header.
func (a *auth) authenticated(r *http.Request) bool {
	if a == nil || r == nil {
		return false
	}
	if c, err := r.Cookie(SessionCookieName); err == nil && a.validToken(c.Value) {
		return true
	}
	return a.validToken(r.Header.Get(TokenHeader))
}

// hostAllowed reports whether the request's Host header names a
// host this server answers to.
//
// This is the DNS-rebinding defence. A rebound attack resolves the
// attacker's name to 127.0.0.1, so the browser connects to this
// server but sends "Host: attacker.example". Rejecting any host
// that is not ours stops the request before it reaches a handler.
func (a *auth) hostAllowed(r *http.Request) bool {
	if a == nil || r == nil {
		return false
	}
	if a.anyHost {
		return true
	}
	host := hostOnly(r.Host)
	if host == "" {
		return false
	}
	return a.hosts[host]
}

// originAllowed reports whether a state-changing request came from
// this server's own origin.
//
// The comparison is against the request's own Host rather than a
// list: a legitimate browser request carries the origin the page
// was served from, which is this server, while a request driven by
// a page on another site carries that site's origin instead. A
// request with no Origin header is not a browser cross-origin
// request; it still needs the session token.
func (a *auth) originAllowed(r *http.Request) bool {
	if a == nil || r == nil {
		return false
	}
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		return true
	}
	// "null" is what a sandboxed frame or a local file sends. It
	// is not this server, so it is refused.
	if strings.EqualFold(origin, "null") {
		return false
	}
	oHost, oPort, ok := splitOrigin(origin)
	if !ok {
		return false
	}
	hHost, hPort := splitHostHeader(r.Host)
	if hHost == "" {
		return false
	}
	if !strings.EqualFold(oHost, hHost) {
		return false
	}
	return oPort == hPort
}

// splitOrigin returns the host and port of an origin header value.
//
// A missing port is filled in from the scheme's default, so
// "http://example.com" is port 80 and "https://example.com" is
// port 443. The boolean is false when the value is not an origin
// this server understands.
func splitOrigin(origin string) (string, int, bool) {
	rest := origin
	scheme := ""
	if i := strings.Index(rest, "://"); i >= 0 {
		scheme = strings.ToLower(rest[:i])
		rest = rest[i+3:]
	}
	if j := strings.IndexAny(rest, "/?#"); j >= 0 {
		rest = rest[:j]
	}
	if rest == "" {
		return "", 0, false
	}
	host, portStr, err := net.SplitHostPort(rest)
	if err != nil {
		host = strings.Trim(rest, "[]")
		var port int
		switch scheme {
		case "http":
			port = 80
		case "https":
			port = 443
		default:
			return "", 0, false
		}
		return strings.ToLower(host), port, true
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return "", 0, false
	}
	return strings.ToLower(strings.Trim(host, "[]")), port, true
}

// splitHostHeader returns the host and port of a Host header.
//
// The server speaks plain HTTP, so a Host header with no port is
// port 80. That matches the Origin a browser sends for the same
// request, which is what the comparison needs.
func splitHostHeader(hostHeader string) (string, int) {
	hostHeader = strings.TrimSpace(hostHeader)
	if hostHeader == "" {
		return "", 0
	}
	host, portStr, err := net.SplitHostPort(hostHeader)
	if err != nil {
		return strings.ToLower(strings.Trim(hostHeader, "[]")), 80
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return strings.ToLower(strings.Trim(host, "[]")), 80
	}
	return strings.ToLower(strings.Trim(host, "[]")), port
}

// isMutating reports whether a method changes state.
//
// A GET is safe to serve without an origin check: a cross-origin
// page cannot read the response, because the server sends no CORS
// headers, and nosniff stops the browser from running a JSON
// response as script.
func isMutating(method string) bool {
	switch strings.ToUpper(method) {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	}
	return false
}
