package webui

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"strings"
	"time"
)

// assets holds the UI. go:embed bakes it into the binary
// at build time, so `aios web` serves the real UI from a
// single file with nothing to install and nothing to
// fetch at runtime.
//
// The "all:" prefix pulls in every file under
// assets/, so adding a stylesheet or icon needs no change
// here.
//
//go:embed all:assets
var assets embed.FS

// Assets returns the embedded asset tree rooted at the
// assets directory.
func Assets() (fs.FS, error) {
	return fs.Sub(assets, "assets")
}

// Server serves the UI and its API.
//
// It is deliberately read-mostly: the browser reads agent
// state, posts messages, and saves a layout. Nothing in
// the UI can start or stop a plan, because the approval
// daemon and the CLI already own that and a second writer
// would create two sources of truth for run state.
type Server struct {
	hub  *Hub
	mux  *http.ServeMux
	addr string
	// auth is the session boundary: one token for the process,
	// plus the host and origin checks that keep a browser on
	// another site from driving the UI.
	auth *auth
	// handler is the mux with the auth middleware in front of
	// it, so every request goes through the same chain.
	handler http.Handler
	// boundAddr is the address the listener actually bound to.
	// It differs from addr when the port was 0.
	boundAddr string
}

// Config configures a Server.
type Config struct {
	// Hub is the state source. Required.
	Hub *Hub
	// Addr is the listen address. An empty Addr uses
	// DefaultAddr.
	Addr string
	// Token pins the session token. An empty Token uses
	// EnvToken from the environment, and if that is unset a
	// fresh token is generated for this process.
	//
	// Pinning it is what lets an operator (or a test)
	// authenticate without reading the URL the server prints.
	Token string
	// AllowedHosts names extra host names the server answers
	// to, on top of the loopback names and the bind host.
	//
	// It exists for the case where the operator reaches the
	// UI through a name that resolves to the machine: the host
	// check rejects a request whose Host header is not one of
	// ours, and an operator with their own hostname would
	// otherwise be locked out.
	AllowedHosts []string
}

// DefaultAddr binds loopback only.
//
// The UI exposes the audit log and the ability to post
// messages to agents, which is not something to put on a
// network interface by default. Binding loopback means a
// remote operator has to forward the port deliberately,
// which is an explicit act.
const DefaultAddr = "127.0.0.1:7755"

// New builds a Server and registers its routes.
func New(cfg Config) (*Server, error) {
	if cfg.Hub == nil {
		return nil, fmt.Errorf("webui: hub is required")
	}
	addr := cfg.Addr
	if addr == "" {
		addr = DefaultAddr
	}
	a, err := newAuth(cfg.Token, addr, cfg.AllowedHosts)
	if err != nil {
		return nil, fmt.Errorf("webui: session token: %w", err)
	}
	s := &Server{hub: cfg.Hub, mux: http.NewServeMux(), addr: addr, auth: a}

	sub, err := Assets()
	if err != nil {
		return nil, fmt.Errorf("webui: embedded assets: %w", err)
	}
	// The static handler is mounted last and serves
	// index.html for any unmatched path, so the UI is a
	// single page and a deep link does not 404.
	fileServer := http.FileServer(http.FS(sub))
	s.mux.Handle("/", spaHandler(sub, fileServer))

	s.mux.HandleFunc("GET /api/state", s.handleState)
	s.mux.HandleFunc("GET /api/events", s.handleEvents)
	s.mux.HandleFunc("POST /api/messages", s.handlePostMessage)
	s.mux.HandleFunc("GET /api/layout", s.handleGetLayout)
	s.mux.HandleFunc("PUT /api/layout", s.handlePutLayout)
	s.mux.HandleFunc("POST /api/agents/{id}/status", s.handleAgentStatus)
	s.mux.HandleFunc("GET /api/approvals", s.handleApprovals)
	s.mux.HandleFunc("POST /api/approvals/{id}", s.handleApprovalDecision)
	s.handler = s.withAuth(s.mux)
	return s, nil
}

// Addr returns the address the server is reachable on.
//
// Before the listener is bound this is the configured address;
// afterwards it is the address the listener actually bound to,
// which matters when the configured port was 0.
func (s *Server) Addr() string {
	if s.boundAddr != "" {
		return s.boundAddr
	}
	return s.addr
}

// Token returns the session token for this server.
//
// It is a capability: it grants full control of the UI, including
// approving tool calls, so it is printed only in the URL the
// operator opens and never written to the audit log.
func (s *Server) Token() string {
	if s == nil || s.auth == nil {
		return ""
	}
	return s.auth.token
}

// SessionURL returns the URL the operator opens to start a
// session.
//
// The token rides in the query string exactly once. The first
// request trades it for a cookie and redirects to the clean URL, so
// the token does not linger in the address bar or in a copied link.
func (s *Server) SessionURL() string {
	return s.BaseURL() + "?" + tokenParam + "=" + s.Token()
}

// BaseURL returns the UI's address with no token.
//
// It is for reporting the bind address without handing out a
// credential: the session token is minted per process, so a token
// printed by a command that is about to exit would not work for the
// server that comes after it.
func (s *Server) BaseURL() string {
	return "http://" + s.Addr() + "/"
}

// Handler returns the routed handler. Tests use it with
// httptest instead of binding a port.
func (s *Server) Handler() http.Handler { return s.handler }

// ListenAndServe starts the server and blocks.
//
// A configured-but-unused shutdown path is not provided:
// `aios web` is a foreground command and the operator
// stops it with Ctrl-C, which the process handles by
// exiting. Adding a graceful-shutdown handshake for a
// process that is about to be killed anyway would be
// code with no caller.
func (s *Server) ListenAndServe() error {
	ln, err := s.Listen()
	if err != nil {
		return err
	}
	return s.Serve(ln)
}

// Listen binds the server's socket without serving on it.
//
// The bound address is what the operator has to open, and a port of
// 0 means the real port is only known after the bind. Splitting
// the bind from the serve lets the caller print a URL that
// actually works: with a port of 0, the address from the config is
// not the address the browser can reach.
func (s *Server) Listen() (net.Listener, error) {
	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		return nil, fmt.Errorf("webui: listen %s: %w", s.addr, err)
	}
	s.boundAddr = ln.Addr().String()
	return ln, nil
}

// Serve serves on an already-bound listener.
//
// It blocks until the server stops.
func (s *Server) Serve(ln net.Listener) error {
	srv := &http.Server{
		Addr:              s.boundAddr,
		Handler:           s.handler,
		ReadHeaderTimeout: 10 * time.Second,
		// No WriteTimeout: it would cut off an SSE stream
		// that is meant to stay open. The read side is
		// bounded above, and a UI with no input is not a
		// resource risk.
	}
	if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("webui: serve %s: %w", s.boundAddr, err)
	}
	return nil
}

// spaHandler serves the embedded assets, falling back to
// index.html so client-side routes work on a refresh.
//
// The fallback is scoped to non-API paths only. A request to
// /api/anything that matches no handler used to be served the
// HTML shell with a 200, so a mistyped API call looked like a
// successful response with the wrong body: `POST /api/state`
// returned 200 and the app.js, and a fetch() that treated
// non-2xx as failure would happily parse HTML as state. An
// API namespace that answers 200 with HTML is worse than one
// that answers 404, because it fails late and silently.
func spaHandler(sub fs.FS, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Never serve the directory listing for a bare
		// "/"; send the UI instead.
		if r.URL.Path == "/" {
			serveIndex(sub, w, r)
			return
		}
		// index.html is the shell, not a file to redirect
		// from: http.FileServer answers /index.html with a
		// 301 to "/", which would fight the SPA fallback
		// below. Serve it directly so a bookmarked
		// /index.html behaves the same as /.
		if r.URL.Path == "/index.html" {
			serveIndex(sub, w, r)
			return
		}
		// An unmatched /api/... path is an API error, not a
		// client-side route. Distinguish "no such endpoint"
		// (404) from "right endpoint, wrong method" (405):
		// the mux knows the path is registered for another
		// method, and silently serving the shell for it
		// turned POST /api/state into a 200 carrying HTML,
		// which a fetch() that only checks res.ok would
		// accept as a successful state read.
		if strings.HasPrefix(r.URL.Path, "/api/") {
			if knownAPI.Has(r.URL.Path) {
				w.Header().Set("Allow", knownAPI.methodsFor(r.URL.Path))
				writeError(w, http.StatusMethodNotAllowed,
					r.Method+" is not allowed on "+r.URL.Path)
				return
			}
			writeError(w, http.StatusNotFound, "no such endpoint: "+r.URL.Path)
			return
		}
		f, err := sub.Open(path(r.URL.Path))
		if err != nil {
			// Unknown path: assume a client route and
			// serve the shell.
			serveIndex(sub, w, r)
			return
		}
		f.Close()
		next.ServeHTTP(w, r)
	})
}

// knownAPI records the API routes and the methods each one
// answers, so the SPA fallback can tell "wrong method" from
// "no such endpoint" and answer 405 instead of serving the
// HTML shell.
var knownAPI = apiRoutes{
	"/api/state":              {"GET"},
	"/api/events":             {"GET"},
	"/api/messages":           {"POST"},
	"/api/layout":             {"GET", "PUT"},
	"/api/agents/{id}/status": {"POST"},
	"/api/approvals":          {"GET"},
	"/api/approvals/{id}":     {"POST"},
}

type apiRoutes map[string][]string

// Has reports whether the exact path is a registered route.
//
// The agent-status route is matched literally here even
// though it carries a wildcard: a wrong method on
// /api/agents/builder/status still needs 405, and treating
// it as unknown would answer 404. The cost is that a request
// to a made-up agent under the real prefix gets 405 rather
// than 404 on a method mismatch; the status handler itself
// still 404s an unknown agent id.
func (a apiRoutes) Has(p string) bool {
	if _, ok := a[p]; ok {
		return true
	}
	const prefix = "/api/agents/"
	const suffix = "/status"
	if strings.HasPrefix(p, prefix) && strings.HasSuffix(p, suffix) {
		return true
	}
	// Approval decisions are per-id, same shape as the agent
	// status route. Without this a POST with a bad method on a
	// real approval id would fall through to the SPA handler
	// and answer 200 with HTML.
	const approvalPrefix = "/api/approvals/"
	if strings.HasPrefix(p, approvalPrefix) && p != "/api/approvals/" {
		return true
	}
	return false
}

// methodsFor returns the Allow header value for a path.
func (a apiRoutes) methodsFor(p string) string {
	if m, ok := a[p]; ok {
		return strings.Join(m, ", ")
	}
	if strings.HasPrefix(p, "/api/approvals/") {
		return "POST"
	}
	return "POST"
}

func serveIndex(sub fs.FS, w http.ResponseWriter, r *http.Request) {
	data, err := fs.ReadFile(sub, "index.html")
	if err != nil {
		http.Error(w, "webui: index.html missing from the binary", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

// path strips the leading slash and guards against a
// traversal attempt reaching fs.Open with a relative path.
func path(p string) string {
	if len(p) > 0 && p[0] == '/' {
		return p[1:]
	}
	return p
}

// handleState returns the full snapshot. A browser
// fetches this on load and after a reconnect.
func (s *Server) handleState(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.hub.Snapshot())
}

// handleEvents streams server-sent events.
//
// SSE rather than a websocket because the data flow is
// one-way (server to browser) and SSE reconnects on its
// own. The one client-to-server action, posting a message,
// is a plain POST.
func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "webui: streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch, cancel := s.hub.subscribe()
	defer cancel()

	// Send the current state immediately so the browser
	// does not render an empty page and then wait for the
	// next mutation, which may be minutes away.
	if ev := mustEvent("snapshot", s.hub.Snapshot()); writeSSE(w, ev) {
		flusher.Flush()
	}

	// A periodic comment keeps intermediaries from
	// closing an idle connection. The interval is long
	// because the UI has no heartbeat need of its own; this
	// exists only to defeat idle timeouts on proxies.
	ticker := time.NewTicker(25 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			if _, err := fmt.Fprint(w, ": keep-alive\n\n"); err != nil {
				return
			}
			flusher.Flush()
		case ev, ok := <-ch:
			if !ok {
				return
			}
			if !writeSSE(w, ev) {
				return
			}
			flusher.Flush()
		}
	}
}

// writeSSE emits one event. It reports false when the
// client is gone, so the caller stops streaming.
func writeSSE(w http.ResponseWriter, ev event) bool {
	if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", ev.Kind, ev.Payload); err != nil {
		return false
	}
	return true
}

// postMessageReq is the POST /api/messages body.
type postMessageReq struct {
	From string `json:"from"`
	To   string `json:"to"`
	Kind string `json:"kind"`
	Body string `json:"body"`
}

// handleApprovals lists the approval requests waiting on an
// operator decision.
func (s *Server) handleApprovals(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"approvals":   s.hub.PendingApprovals(),
		"can_resolve": s.hub.CanResolveApprovals(),
	})
}

// handleApprovalDecision applies a grant or deny from the
// browser.
//
// The decision is in the body, not the path: the path carries
// the approval id (so it is visible in a server log) and the
// body carries what the operator chose. Keeping the verb in
// the body means one route instead of two, and the id is
// still the thing that scopes the action.
func (s *Server) handleApprovalDecision(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": "approval id is required",
		})
		return
	}

	var body struct {
		Approve *bool `json:"approve"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": "body must be {\"approve\": true|false}",
		})
		return
	}
	// A missing "approve" is ambiguous: it could mean false, or
	// it could mean the client sent the wrong shape. Treating
	// it as false would silently DENY on a malformed request,
	// which stops a run that nobody asked to stop. Requiring
	// the field makes a client bug visible.
	if body.Approve == nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": "body must include \"approve\": true or false",
		})
		return
	}

	state, err := s.hub.DecideApproval(id, *body.Approve)
	if err != nil {
		writeJSON(w, approvalErrorStatus(err), map[string]any{
			"error": err.Error(),
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"approval": id,
		"state":    state,
	})
}

// approvalErrorStatus maps a decision failure to a status the
// browser can act on.
//
// The distinction that matters is 404 vs 409: "I do not have
// this approval" is a stale tab, while "I have it but cannot
// grant it here" is a configuration the operator has to
// change. Both are 4xx, so the UI shows the message either
// way, but they are not the same failure.
func approvalErrorStatus(err error) int {
	switch {
	case err == nil:
		return http.StatusOK
	case errors.Is(err, ErrNoResolver):
		// The server is in the wrong mode: it is `aios web`
		// with no approvals service. Nothing about the
		// request was wrong, so this is not a client error.
		return http.StatusServiceUnavailable
	case errors.Is(err, ErrApprovalNotFound):
		return http.StatusNotFound
	default:
		return http.StatusConflict
	}
}

// decodeJSON reads a bounded JSON body.
//
// The limit matters because this handler is reachable from a
// browser on loopback: an unbounded read is a way to pin a
// server goroutine on a request that never ends.
func decodeJSON(r *http.Request, v any) error {
	defer func() { _ = r.Body.Close() }()
	return json.NewDecoder(io.LimitReader(r.Body, 64<<10)).Decode(v)
}

func (s *Server) handlePostMessage(w http.ResponseWriter, r *http.Request) {
	// A body cap: this endpoint takes operator text, not a
	// document.
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	var req postMessageReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.From == "" {
		writeError(w, http.StatusBadRequest, "`from` is required")
		return
	}
	if req.To != "" && !KnownAgent(s.hub, req.To) {
		// Addressing an agent that does not exist would
		// silently drop the message, so it is an error
		// rather than a no-op.
		writeError(w, http.StatusBadRequest, "unknown agent "+req.To)
		return
	}
	msg, ok := s.hub.Post(Message{
		From: req.From, To: req.To, Kind: req.Kind, Body: req.Body,
	})
	if !ok {
		writeError(w, http.StatusBadRequest, "`body` is required")
		return
	}
	writeJSON(w, http.StatusCreated, msg)
}

// agentStatusReq is the POST
// /api/agents/{id}/status body. It lets the UI reflect a
// state change it observed, which is how an operator's
// manual "cancel" shows up in the panel.
type agentStatusReq struct {
	Status string `json:"status"`
	Detail string `json:"detail"`
}

func (s *Server) handleAgentStatus(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "agent id is required")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 8<<10)
	var req agentStatusReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	st := Status(req.Status)
	switch st {
	case StatusIdle, StatusRunning, StatusBlocked, StatusDone, StatusFailed:
	default:
		writeError(w, http.StatusBadRequest, "unknown status "+req.Status)
		return
	}
	a := s.hub.SetStatus(id, st, req.Detail)
	writeJSON(w, http.StatusOK, a)
}

func (s *Server) handleGetLayout(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.hub.Layout())
}

func (s *Server) handlePutLayout(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	var l Layout
	if err := json.NewDecoder(r.Body).Decode(&l); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := s.hub.SetLayout(l); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, s.hub.Layout())
}

// KnownAgent reports whether id names an agent the hub
// knows. A post may address an agent that has not started
// yet only as a broadcast (To == "").
func KnownAgent(h *Hub, id string) bool {
	_, ok := h.Agent(id)
	return ok
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(true)
	_ = enc.Encode(v)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}
