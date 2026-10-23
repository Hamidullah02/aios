package webui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// testToken is the session token used by the package's tests.
//
// It is a constant rather than a random value so a failure message
// can name the token that was expected, and so the whole suite
// exercises the same "the operator knows the token" path.
const testToken = "test-session-token"

// newAuthTestServer returns a server with a pinned token and a
// hub, for the tests that care about the auth boundary.
func newAuthTestServer(t *testing.T) (*Server, *Hub) {
	t.Helper()
	hub := NewHub()
	s, err := New(Config{Hub: hub, Addr: "127.0.0.1:0", Token: testToken})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return s, hub
}

// authRequest builds a request that passes the auth boundary.
//
// The Host is set explicitly because httptest.NewRequest defaults to
// "example.com", which is not a host this server answers to -- and
// the host check is one of the things under test.
func authRequest(method, path, body string) *http.Request {
	var r *http.Request
	if body == "" {
		r = httptest.NewRequest(method, path, nil)
	} else {
		r = httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
	}
	r.Host = "127.0.0.1:7755"
	r.Header.Set(TokenHeader, testToken)
	return r
}

// doAuth runs an authenticated request against the server.
func doAuth(t *testing.T, s *Server, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, authRequest(method, path, body))
	return w
}

// --- the boundary itself ---

// TestAuth_UnauthenticatedRequestsAreRefused is the gate for the
// whole feature.
//
// Before this, every endpoint on the UI was open to anything that
// could reach the port: a page on another site, another process on
// the machine, a rebound DNS name. The UI can approve tool calls,
// so an unauthenticated request is not a nuisance -- it is a
// capability handed to a stranger.
func TestAuth_UnauthenticatedRequestsAreRefused(t *testing.T) {
	s, _ := newAuthTestServer(t)

	cases := []struct {
		method, path, body string
	}{
		{http.MethodGet, "/api/state", ""},
		{http.MethodGet, "/api/approvals", ""},
		{http.MethodPost, "/api/messages", `{"from":"t1","body":"x"}`},
		{http.MethodPut, "/api/layout", `{}`},
		{http.MethodPost, "/api/agents/t1/status", `{"status":"running"}`},
		{http.MethodPost, "/api/approvals/apr-1", `{"approve":true}`},
		{http.MethodGet, "/", ""},
		{http.MethodGet, "/app.js", ""},
	}
	for _, c := range cases {
		r := httptest.NewRequest(c.method, c.path, strings.NewReader(c.body))
		r.Host = "127.0.0.1:7755"
		w := httptest.NewRecorder()
		s.Handler().ServeHTTP(w, r)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("%s %s without a token = %d, want 401", c.method, c.path, w.Code)
		}
		// A refused request must be JSON, never the SPA shell:
		// an HTML body here would let a caller mistake a refusal
		// for a successful load of the page.
		if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
			t.Errorf("%s %s refused with content-type %q, want JSON", c.method, c.path, ct)
		}
	}
}

// TestAuth_TokenHeaderAuthenticates pins the non-browser path: a
// script authenticates with a header.
func TestAuth_TokenHeaderAuthenticates(t *testing.T) {
	s, _ := newAuthTestServer(t)
	w := doAuth(t, s, http.MethodGet, "/api/state", "")
	if w.Code != http.StatusOK {
		t.Fatalf("GET /api/state with a valid token = %d, want 200", w.Code)
	}
}

// TestAuth_CookieAuthenticates pins the browser path: the EventSource
// cannot set a header, so the cookie is what the SSE stream uses.
func TestAuth_CookieAuthenticates(t *testing.T) {
	s, _ := newAuthTestServer(t)
	r := httptest.NewRequest(http.MethodGet, "/api/state", nil)
	r.Host = "127.0.0.1:7755"
	r.AddCookie(&http.Cookie{Name: SessionCookieName, Value: testToken})
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /api/state with a valid cookie = %d, want 200", w.Code)
	}
}

// TestAuth_WrongTokenIsRefused pins that a near-miss is refused.
func TestAuth_WrongTokenIsRefused(t *testing.T) {
	s, _ := newAuthTestServer(t)
	for _, tok := range []string{"", "wrong", testToken + "x", " " + testToken} {
		r := httptest.NewRequest(http.MethodGet, "/api/state", nil)
		r.Host = "127.0.0.1:7755"
		if tok != "" {
			r.Header.Set(TokenHeader, tok)
		}
		w := httptest.NewRecorder()
		s.Handler().ServeHTTP(w, r)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("token %q = %d, want 401", tok, w.Code)
		}
	}
}

// --- the handshake ---

// TestAuth_QueryTokenMintsCookieAndRedirects covers the one place
// the token is allowed in a URL.
//
// The operator opens the link the server printed; the server trades
// the token for a cookie and redirects to the clean URL. Without
// the redirect the token would sit in the address bar forever, and
// a copied link would stay a working credential.
func TestAuth_QueryTokenMintsCookieAndRedirects(t *testing.T) {
	s, _ := newAuthTestServer(t)
	r := httptest.NewRequest(http.MethodGet, "/?"+tokenParam+"="+testToken, nil)
	r.Host = "127.0.0.1:7755"
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, r)

	if w.Code != http.StatusFound {
		t.Fatalf("GET /?token=... = %d, want 302", w.Code)
	}
	loc := w.Header().Get("Location")
	if strings.Contains(loc, tokenParam) {
		t.Errorf("redirect kept the token in the URL: %q", loc)
	}
	var found bool
	for _, c := range w.Result().Cookies() {
		if c.Name == SessionCookieName {
			found = true
			if c.Value != testToken {
				t.Errorf("cookie value = %q, want the session token", c.Value)
			}
			// HttpOnly keeps the token out of page scripts.
			if !c.HttpOnly {
				t.Error("the session cookie must be HttpOnly")
			}
			// Lax, not Strict: the cookie has to survive a
			// top-level navigation that did not start on this
			// site -- the operator clicking the link out of a
			// terminal. The cross-site story does not depend on
			// this attribute: the Origin check refuses any
			// state-changing request that did not come from this
			// server, and Lax cookies are not sent on a
			// cross-site fetch at all.
			if c.SameSite != http.SameSiteLaxMode {
				t.Errorf("SameSite = %v, want Lax", c.SameSite)
			}
		}
	}
	if !found {
		t.Fatal("no session cookie was set")
	}
}

// TestAuth_BadQueryTokenIsRefused pins that the handshake does not
// become a way in.
func TestAuth_BadQueryTokenIsRefused(t *testing.T) {
	s, _ := newAuthTestServer(t)
	r := httptest.NewRequest(http.MethodGet, "/?"+tokenParam+"=nope", nil)
	r.Host = "127.0.0.1:7755"
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("GET /?token=nope = %d, want 401", w.Code)
	}
	if len(w.Result().Cookies()) != 0 {
		t.Error("a bad token must not mint a cookie")
	}
}

// TestAuth_QueryTokenOnlyWorksOnGET pins that the handshake cannot
// be used to smuggle a state-changing request: a redirect cannot
// carry a body, so accepting the token on a POST would silently drop
// what the operator asked for.
func TestAuth_QueryTokenOnlyWorksOnGET(t *testing.T) {
	s, _ := newAuthTestServer(t)
	r := httptest.NewRequest(http.MethodPost, "/api/messages?"+tokenParam+"="+testToken,
		strings.NewReader(`{"from":"t1","body":"x"}`))
	r.Host = "127.0.0.1:7755"
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("POST with a query token = %d, want 401", w.Code)
	}
	if len(s.hub.Messages()) != 0 {
		t.Error("the message must not have been posted")
	}
}

// --- DNS rebinding ---

// TestAuth_ForeignHostIsRefused is the DNS-rebinding defence.
//
// A rebound name resolves to 127.0.0.1, so the browser connects to
// this server but sends "Host: attacker.example". Refusing a host
// the server does not answer to stops the request before it reaches
// a handler.
func TestAuth_ForeignHostIsRefused(t *testing.T) {
	s, _ := newAuthTestServer(t)
	for _, host := range []string{
		"attacker.example",
		"localhost.evil.com",
		"aios.attacker.test:7755",
	} {
		r := httptest.NewRequest(http.MethodGet, "/api/state", nil)
		r.Host = host
		r.Header.Set(TokenHeader, testToken)
		w := httptest.NewRecorder()
		s.Handler().ServeHTTP(w, r)
		if w.Code != http.StatusBadRequest {
			t.Errorf("Host %q = %d, want 400", host, w.Code)
		}
	}
}

// TestAuth_KnownHostsAreAccepted pins the other side: the loopback
// names and the bind host must work, or the UI is unreachable.
func TestAuth_KnownHostsAreAccepted(t *testing.T) {
	s, _ := newAuthTestServer(t)
	for _, host := range []string{"127.0.0.1:7755", "localhost:7755", "[::1]:7755"} {
		r := httptest.NewRequest(http.MethodGet, "/api/state", nil)
		r.Host = host
		r.Header.Set(TokenHeader, testToken)
		w := httptest.NewRecorder()
		s.Handler().ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Errorf("Host %q = %d, want 200", host, w.Code)
		}
	}
}

// TestAuth_AllowedHostsExtendsTheList pins the escape hatch for an
// operator who reaches the UI through their own hostname.
func TestAuth_AllowedHostsExtendsTheList(t *testing.T) {
	hub := NewHub()
	s, err := New(Config{Hub: hub, Addr: "127.0.0.1:0", Token: testToken,
		AllowedHosts: []string{"aios.internal"}})
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(http.MethodGet, "/api/state", nil)
	r.Host = "aios.internal:7755"
	r.Header.Set(TokenHeader, testToken)
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("Host aios.internal = %d, want 200", w.Code)
	}
}

// --- cross-origin ---

// TestAuth_CrossOriginMutationIsRefused pins that a page on another
// site cannot drive the UI even with a valid token.
//
// The token alone is not enough: a browser that has the cookie will
// attach it to a request the page makes. The Origin check is what
// says "this request came from us".
func TestAuth_CrossOriginMutationIsRefused(t *testing.T) {
	s, hub := newAuthTestServer(t)
	r := httptest.NewRequest(http.MethodPost, "/api/messages",
		strings.NewReader(`{"from":"t1","body":"x"}`))
	r.Host = "127.0.0.1:7755"
	r.Header.Set(TokenHeader, testToken)
	r.Header.Set("Origin", "http://evil.example")
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, r)
	if w.Code != http.StatusForbidden {
		t.Fatalf("cross-origin POST = %d, want 403", w.Code)
	}
	if len(hub.Messages()) != 0 {
		t.Error("the message must not have been posted")
	}
}

// TestAuth_SameOriginMutationIsAllowed pins the other side: the
// UI's own fetches carry its origin and must work.
func TestAuth_SameOriginMutationIsAllowed(t *testing.T) {
	s, hub := newAuthTestServer(t)
	r := httptest.NewRequest(http.MethodPost, "/api/messages",
		strings.NewReader(`{"from":"t1","body":"x"}`))
	r.Host = "127.0.0.1:7755"
	r.Header.Set(TokenHeader, testToken)
	r.Header.Set("Origin", "http://127.0.0.1:7755")
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("same-origin POST = %d, want 201 (%s)", w.Code, w.Body.String())
	}
	if len(hub.Messages()) != 1 {
		t.Error("the message was not posted")
	}
}

// TestAuth_NullOriginIsRefused pins that a sandboxed frame or a
// local file is not us.
func TestAuth_NullOriginIsRefused(t *testing.T) {
	s, _ := newAuthTestServer(t)
	r := httptest.NewRequest(http.MethodPost, "/api/messages",
		strings.NewReader(`{"from":"t1","body":"x"}`))
	r.Host = "127.0.0.1:7755"
	r.Header.Set(TokenHeader, testToken)
	r.Header.Set("Origin", "null")
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, r)
	if w.Code != http.StatusForbidden {
		t.Fatalf("null-origin POST = %d, want 403", w.Code)
	}
}

// TestAuth_SameOriginReadIsAllowed pins that the origin check only
// gates mutations. A GET from another origin is harmless: the
// response is not readable cross-origin without CORS headers.
func TestAuth_SameOriginReadIsAllowed(t *testing.T) {
	s, _ := newAuthTestServer(t)
	r := httptest.NewRequest(http.MethodGet, "/api/state", nil)
	r.Host = "127.0.0.1:7755"
	r.Header.Set(TokenHeader, testToken)
	r.Header.Set("Origin", "http://evil.example")
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("cross-origin GET = %d, want 200", w.Code)
	}
}

// --- headers ---

// TestAuth_SecurityHeadersArePresent pins the hardening headers.
//
// The CSP is the one that matters most: the UI has no inline
// script, so a strict policy turns an injected string into text
// instead of script.
func TestAuth_SecurityHeadersArePresent(t *testing.T) {
	s, _ := newAuthTestServer(t)
	w := doAuth(t, s, http.MethodGet, "/api/state", "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	csp := w.Header().Get("Content-Security-Policy")
	if csp == "" {
		t.Fatal("no Content-Security-Policy header")
	}
	for _, want := range []string{
		"default-src 'none'",
		"script-src 'self'",
		"frame-ancestors 'none'",
	} {
		if !strings.Contains(csp, want) {
			t.Errorf("CSP %q is missing %q", csp, want)
		}
	}
	if strings.Contains(csp, "unsafe-inline") {
		t.Errorf("CSP %q allows inline script or style", csp)
	}
	if got := w.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q", got)
	}
	if got := w.Header().Get("X-Frame-Options"); got != "DENY" {
		t.Errorf("X-Frame-Options = %q", got)
	}
	if got := w.Header().Get("Referrer-Policy"); got != "no-referrer" {
		t.Errorf("Referrer-Policy = %q, want no-referrer", got)
	}
}

// --- configuration ---

// TestAuth_TokenIsGeneratedWhenUnset pins that a server always has
// a token. There is no "no auth" configuration: an open UI would
// be a silently reintroduced hole.
func TestAuth_TokenIsGeneratedWhenUnset(t *testing.T) {
	t.Setenv(EnvToken, "")
	s, err := New(Config{Hub: NewHub(), Addr: "127.0.0.1:0"})
	if err != nil {
		t.Fatal(err)
	}
	if s.Token() == "" {
		t.Fatal("no session token was generated")
	}
	if len(s.Token()) < 32 {
		t.Errorf("session token is %d chars, want at least 32", len(s.Token()))
	}
}

// TestAuth_TokenComesFromTheEnvironment pins that an operator can
// pin the token, which is what a headless setup needs.
func TestAuth_TokenComesFromTheEnvironment(t *testing.T) {
	t.Setenv(EnvToken, "pinned-token")
	s, err := New(Config{Hub: NewHub(), Addr: "127.0.0.1:0"})
	if err != nil {
		t.Fatal(err)
	}
	if s.Token() != "pinned-token" {
		t.Errorf("Token = %q, want the pinned value", s.Token())
	}
}

// TestAuth_ExplicitTokenBeatsTheEnvironment pins the precedence:
// the flag wins over the environment, so a one-off run can use a
// different token without changing the operator's setup.
func TestAuth_ExplicitTokenBeatsTheEnvironment(t *testing.T) {
	t.Setenv(EnvToken, "env-token")
	s, err := New(Config{Hub: NewHub(), Addr: "127.0.0.1:0", Token: "explicit"})
	if err != nil {
		t.Fatal(err)
	}
	if s.Token() != "explicit" {
		t.Errorf("Token = %q, want explicit", s.Token())
	}
}

// TestAuth_TokensAreUnique pins that two servers do not share a
// token: each process gets its own.
func TestAuth_TokensAreUnique(t *testing.T) {
	t.Setenv(EnvToken, "")
	a, err := New(Config{Hub: NewHub(), Addr: "127.0.0.1:0"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := New(Config{Hub: NewHub(), Addr: "127.0.0.1:0"})
	if err != nil {
		t.Fatal(err)
	}
	if a.Token() == b.Token() {
		t.Error("two servers were given the same token")
	}
}

// TestAuth_SessionURLCarriesTheToken pins that the URL the operator
// opens is the one that works.
func TestAuth_SessionURLCarriesTheToken(t *testing.T) {
	s, _ := newAuthTestServer(t)
	url := s.SessionURL()
	if !strings.Contains(url, testToken) {
		t.Errorf("SessionURL %q does not carry the token", url)
	}
	if !strings.HasPrefix(url, "http://") {
		t.Errorf("SessionURL %q is not an http URL", url)
	}
}

// --- the helpers themselves ---

// TestHostOnly pins the host extraction, including the IPv6 form
// that net.SplitHostPort rejects without brackets.
func TestHostOnly(t *testing.T) {
	cases := map[string]string{
		"127.0.0.1:7755": "127.0.0.1",
		"localhost:7755": "localhost",
		"0.0.0.0:7755":   "0.0.0.0",
		"[::1]:7755":     "::1",
		"::1":            "::1",
		"Example.COM:80": "example.com",
		"":               "",
	}
	for in, want := range cases {
		if got := hostOnly(in); got != want {
			t.Errorf("hostOnly(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestSplitOrigin pins the origin parsing.
func TestSplitOrigin(t *testing.T) {
	cases := []struct {
		in     string
		host   string
		port   int
		wantOK bool
	}{
		{"http://127.0.0.1:7755", "127.0.0.1", 7755, true},
		{"http://127.0.0.1", "127.0.0.1", 80, true},
		{"https://aios.example", "aios.example", 443, true},
		{"null", "", 0, false},
		{"", "", 0, false},
		{"http://", "", 0, false},
	}
	for _, c := range cases {
		host, port, ok := splitOrigin(c.in)
		if ok != c.wantOK {
			t.Errorf("splitOrigin(%q) ok = %v, want %v", c.in, ok, c.wantOK)
			continue
		}
		if !c.wantOK {
			continue
		}
		if host != c.host || port != c.port {
			t.Errorf("splitOrigin(%q) = %q,%d, want %q,%d", c.in, host, port, c.host, c.port)
		}
	}
}
