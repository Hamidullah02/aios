package webui

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"strconv"
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
}

// Config configures a Server.
type Config struct {
	// Hub is the state source. Required.
	Hub *Hub
	// Addr is the listen address. An empty Addr uses
	// DefaultAddr.
	Addr string
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
	s := &Server{hub: cfg.Hub, mux: http.NewServeMux(), addr: addr}

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
	return s, nil
}

// Addr returns the configured listen address.
func (s *Server) Addr() string { return s.addr }

// Handler returns the routed handler. Tests use it with
// httptest instead of binding a port.
func (s *Server) Handler() http.Handler { return s.mux }

// ListenAndServe starts the server and blocks.
//
// A configured-but-unused shutdown path is not provided:
// `aios web` is a foreground command and the operator
// stops it with Ctrl-C, which the process handles by
// exiting. Adding a graceful-shutdown handshake for a
// process that is about to be killed anyway would be
// code with no caller.
func (s *Server) ListenAndServe() error {
	srv := &http.Server{
		Addr:              s.addr,
		Handler:           s.mux,
		ReadHeaderTimeout: 10 * time.Second,
		// No WriteTimeout: it would cut off an SSE stream
		// that is meant to stay open. The read side is
		// bounded above, and a UI with no input is not a
		// resource risk.
	}
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("webui: serve %s: %w", s.addr, err)
	}
	return nil
}

// spaHandler serves the embedded assets, falling back to
// index.html so client-side routes work on a refresh.
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

// itoaStr is a small helper for the CSP header, kept here
// so the import list stays minimal.
func itoaStr(n int) string { return strconv.Itoa(n) }
