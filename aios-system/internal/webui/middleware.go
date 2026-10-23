package webui

import (
	"encoding/json"
	"net/http"
)

// contentSecurityPolicy is the CSP the UI serves.
//
// The UI has no inline script, no inline style, and no external
// resource, so the policy can be strict: everything must come from
// this origin, and scripts and styles may not be inlined at all.
// That is what turns an injected string into text instead of
// script.
//
// The 'unsafe-inline' qualifier is absent from script-src and
// style-src on purpose. The stylesheet uses CSS custom properties
// and color-mix, not inline style attributes, so nothing needs it.
const contentSecurityPolicy = "default-src 'none'; " +
	"script-src 'self'; " +
	"style-src 'self'; " +
	"img-src 'self' data:; " +
	"font-src 'self'; " +
	"connect-src 'self'; " +
	"form-action 'none'; " +
	"frame-ancestors 'none'; " +
	"base-uri 'none'"

// securityHeaders are the response headers every request carries.
//
// X-Content-Type-Options stops a browser from running a JSON
// response as script; X-Frame-Options stops the UI being framed by
// a page on another site; Referrer-Policy keeps the session URL
// (which carries the token) out of any Referer header the browser
// would otherwise send to a third party.
func securityHeaders() map[string]string {
	return map[string]string{
		"Content-Security-Policy":      contentSecurityPolicy,
		"X-Content-Type-Options":       "nosniff",
		"X-Frame-Options":              "DENY",
		"Referrer-Policy":              "no-referrer",
		"Cross-Origin-Opener-Policy":   "same-origin",
		"Cross-Origin-Resource-Policy": "same-origin",
	}
}

// withAuth wraps next in the UI's authentication and request
// hardening middleware.
//
// The chain, in order:
//
//  1. Security headers on every response.
//  2. The Host check, which rejects a request whose Host header is
//     not one this server answers to. This is the DNS-rebinding
//     defence and it runs before anything else looks at the body.
//  3. The session token check, except on the one request that
//     carries the token in the query string and trades it for a
//     cookie.
//  4. The Origin check on state-changing requests.
//
// The order matters: a rejected request must not reach a handler,
// and the token check must not be the thing that decides whether a
// hostile page can make the server do work.
func (s *Server) withAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for k, v := range securityHeaders() {
			w.Header().Set(k, v)
		}

		if !s.auth.hostAllowed(r) {
			writeAuthError(w, http.StatusBadRequest, "bad host")
			return
		}

		// The token in the query string is the handshake: it is
		// exchanged for a cookie and the browser is redirected to
		// the clean URL. The token is never accepted from the
		// query string on any other path, so a link that leaks
		// does not stay usable.
		//
		// Only a GET can mint: a redirect cannot carry a POST
		// body, so accepting a token on a state-changing request
		// would silently drop the request the operator intended.
		if tok := r.URL.Query().Get(tokenParam); tok != "" {
			if r.Method != http.MethodGet {
				writeAuthError(w, http.StatusUnauthorized, "session token must not be sent in the query string of a state-changing request")
				return
			}
			s.mintSession(w, r, tok)
			return
		}

		if !s.auth.authenticated(r) {
			writeAuthError(w, http.StatusUnauthorized, "missing or invalid session token")
			return
		}

		if isMutating(r.Method) && !s.auth.originAllowed(r) {
			writeAuthError(w, http.StatusForbidden, "cross-origin request refused")
			return
		}

		next.ServeHTTP(w, r)
	})
}

// mintSession trades a token for a session cookie and redirects to
// the clean URL.
//
// A bad token is refused outright: silently redirecting to a page
// that cannot authenticate would leave the operator staring at a
// blank UI with no explanation.
func (s *Server) mintSession(w http.ResponseWriter, r *http.Request, tok string) {
	if !s.auth.validToken(tok) {
		writeAuthError(w, http.StatusUnauthorized, "invalid session token")
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:  SessionCookieName,
		Value: tok,
		Path:  "/",
		// HttpOnly keeps the token out of page scripts: a
		// cross-site script must not be able to read the
		// session out of the cookie jar.
		HttpOnly: true,
		// Lax rather than Strict. The cookie has to survive a
		// top-level navigation that did not start on this
		// site -- an operator clicking the link out of a
		// terminal, a chat window, or a bookmark -- and Strict
		// would refuse it there.
		//
		// Lax does not weaken the cross-site story: a
		// cross-site fetch or form POST does not carry a Lax
		// cookie at all, and the Origin check refuses any
		// state-changing request that does not come from this
		// server. The Origin check, not the cookie attribute,
		// is what stops a hostile page from driving the UI.
		SameSite: http.SameSiteLaxMode,
	})
	// Redirect to the same path without the token, so the
	// address bar stops carrying the secret.
	target := r.URL.Path
	if target == "" {
		target = "/"
	}
	if q := r.URL.Query(); len(q) > 0 {
		q.Del(tokenParam)
		if enc := q.Encode(); enc != "" {
			target += "?" + enc
		}
	}
	http.Redirect(w, r, target, http.StatusFound)
}

// writeAuthError writes a JSON error for a refused request.
//
// The body says what happened without saying how: a caller learns
// that the request was refused, not whether the token was close.
func writeAuthError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
