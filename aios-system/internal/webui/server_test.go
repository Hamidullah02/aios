package webui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

// newTestServer returns a server backed by a fresh hub.
func newTestServer(t *testing.T) (*Server, *Hub) {
	t.Helper()
	hub := NewHub()
	s, err := New(Config{Hub: hub, Addr: "127.0.0.1:0"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return s, hub
}

// do runs a request against the server's handler.
func do(t *testing.T, s *Server, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var r *http.Request
	if body == "" {
		r = httptest.NewRequest(method, path, nil)
	} else {
		r = httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, r)
	return w
}

// decode unmarshals a JSON response body into T.
func decode[T any](t *testing.T, w *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(w.Body.Bytes(), &v); err != nil {
		t.Fatalf("decode %q: %v", w.Body.String(), err)
	}
	return v
}

// TestSPAHandler_APIPathsNeverServeTheShell is the guard for
// the bug where every unmatched /api/... request was answered
// with index.html and a 200.
//
// That is the worst possible failure shape for an API: the
// caller sees success, the content type says HTML, and a
// client that only checks res.ok parses the shell as data.
// Both cases below used to return 200 + HTML.
func TestSPAHandler_APIPathsNeverServeTheShell(t *testing.T) {
	s, _ := newTestServer(t)

	t.Run("wrong method on a real endpoint", func(t *testing.T) {
		w := do(t, s, http.MethodPost, "/api/state", "")
		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("POST /api/state = %d, want 405", w.Code)
		}
		if ct := w.Header().Get("Content-Type"); strings.HasPrefix(ct, "text/html") {
			t.Errorf("content-type = %q, want JSON", ct)
		}
		if !strings.Contains(w.Body.String(), "error") {
			t.Errorf("body = %q, want a JSON error", w.Body.String())
		}
		if allow := w.Header().Get("Allow"); allow != "GET" {
			t.Errorf("Allow = %q, want \"GET\"", allow)
		}
	})

	t.Run("unknown endpoint", func(t *testing.T) {
		w := do(t, s, http.MethodGet, "/api/nope", "")
		if w.Code != http.StatusNotFound {
			t.Errorf("GET /api/nope = %d, want 404", w.Code)
		}
		if ct := w.Header().Get("Content-Type"); strings.HasPrefix(ct, "text/html") {
			t.Errorf("content-type = %q, want JSON", ct)
		}
	})

	t.Run("wrong method on the agent status route", func(t *testing.T) {
		w := do(t, s, http.MethodGet, "/api/agents/builder/status", "")
		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("GET agent status = %d, want 405", w.Code)
		}
	})

	t.Run("layout wrong method", func(t *testing.T) {
		w := do(t, s, http.MethodPost, "/api/layout", "{}")
		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("POST /api/layout = %d, want 405", w.Code)
		}
	})
}

// TestSPAHandler_ClientRoutesStillGetTheShell is the other
// half of the contract: tightening the API namespace must not
// break deep links. A refresh on a client-side route still
// has to load the app.
func TestSPAHandler_ClientRoutesStillGetTheShell(t *testing.T) {
	s, _ := newTestServer(t)
	for _, p := range []string{"/", "/index.html", "/agents/builder", "/some/deep/route"} {
		w := do(t, s, http.MethodGet, p, "")
		if w.Code != http.StatusOK {
			t.Errorf("GET %s = %d, want 200", p, w.Code)
			continue
		}
		if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
			t.Errorf("GET %s content-type = %q, want HTML", p, ct)
		}
		if !strings.Contains(w.Body.String(), "<!DOCTYPE html>") {
			t.Errorf("GET %s did not serve the shell", p)
		}
	}
}

// TestSPAHandler_StaticAssetsStillServe asserts the tightening
// did not swallow real files: app.js and app.css must come
// from disk, not from the HTML fallback.
func TestSPAHandler_StaticAssetsStillServe(t *testing.T) {
	s, _ := newTestServer(t)
	for p, want := range map[string]string{
		"/app.js":   "javascript",
		"/app.css":  "css",
	} {
		w := do(t, s, http.MethodGet, p, "")
		if w.Code != http.StatusOK {
			t.Errorf("GET %s = %d, want 200", p, w.Code)
			continue
		}
		if ct := w.Header().Get("Content-Type"); !strings.Contains(ct, want) {
			t.Errorf("GET %s content-type = %q, want %q", p, ct, want)
		}
		if strings.Contains(w.Body.String(), "<!DOCTYPE html>") {
			t.Errorf("GET %s served the shell instead of the asset", p)
		}
	}
}

// --- construction ---

func TestNew_RequiresHub(t *testing.T) {
	if _, err := New(Config{}); err == nil {
		t.Fatal("expected an error when the hub is nil")
	}
}

func TestNew_DefaultAddrIsLoopback(t *testing.T) {
	// The UI exposes the audit log, so binding a network
	// interface by default would be a security decision
	// made silently. This pins the loopback default.
	s, _ := newTestServer(t)
	if !strings.HasPrefix(s.Addr(), "127.0.0.1:") {
		t.Errorf("Addr = %q, want a loopback address", s.Addr())
	}
}

// --- embedded assets ---

func TestAssets_AreEmbedded(t *testing.T) {
	// The whole point of v0.41.0: the UI ships inside the
	// binary. A missing asset is a build-time failure, so
	// these three are the minimum the page cannot render
	// without.
	for _, name := range []string{"index.html", "app.css", "app.js"} {
		w := do(t, mustServer(t), "GET", "/"+name, "")
		if w.Code != http.StatusOK {
			t.Errorf("GET /%s = %d, want 200", name, w.Code)
		}
		if w.Body.Len() == 0 {
			t.Errorf("GET /%s returned an empty body", name)
		}
	}
}

func mustServer(t *testing.T) *Server {
	t.Helper()
	s, _ := newTestServer(t)
	return s
}

func TestIndex_ServesTheShell(t *testing.T) {
	w := do(t, mustServer(t), "GET", "/", "")
	if w.Code != http.StatusOK {
		t.Fatalf("GET / = %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Errorf("Content-Type = %q, want text/html", ct)
	}
	if !strings.Contains(w.Body.String(), "<html") {
		t.Error("GET / did not return an HTML document")
	}
}

// TestSPA_UnknownPathServesShell covers a browser refresh on
// a client-side route: the path is not a file, so the shell
// is served rather than a 404.
func TestSPA_UnknownPathServesShell(t *testing.T) {
	w := do(t, mustServer(t), "GET", "/dashboard/agents", "")
	if w.Code != http.StatusOK {
		t.Fatalf("deep link = %d, want 200 (the shell, not a 404)", w.Code)
	}
	if !strings.Contains(w.Body.String(), "<html") {
		t.Error("deep link did not return the shell")
	}
}

// --- state endpoint ---

func TestState_EmptyHub(t *testing.T) {
	w := do(t, mustServer(t), "GET", "/api/state", "")
	if w.Code != http.StatusOK {
		t.Fatalf("GET /api/state = %d", w.Code)
	}
	snap := decode[Snapshot](t, w)
	if len(snap.Agents) != 0 || len(snap.Messages) != 0 {
		t.Errorf("snapshot = %+v, want empty", snap)
	}
	// A default layout must be present even with no state, or
	// the client has nothing to render panels from.
	if len(snap.Layout.Panels) != len(AllPanels) {
		t.Errorf("layout has %d panels, want %d", len(snap.Layout.Panels), len(AllPanels))
	}
}

func TestState_ReflectsAgents(t *testing.T) {
	s, hub := newTestServer(t)
	hub.SetStatus("t1", StatusRunning, "")
	hub.SetStatus("t2", StatusFailed, "boom")

	w := do(t, s, "GET", "/api/state", "")
	snap := decode[Snapshot](t, w)
	if len(snap.Agents) != 2 {
		t.Fatalf("agents = %d, want 2", len(snap.Agents))
	}
	// Sorted by ID for a deterministic wire order.
	if snap.Agents[0].ID != "t1" || snap.Agents[1].ID != "t2" {
		t.Errorf("agents out of order: %s, %s", snap.Agents[0].ID, snap.Agents[1].ID)
	}
	if snap.Agents[1].Status != StatusFailed {
		t.Errorf("t2 status = %q, want failed", snap.Agents[1].Status)
	}
}

// --- agent status endpoint ---

func TestPostAgentStatus(t *testing.T) {
	s, hub := newTestServer(t)
	w := do(t, s, "POST", "/api/agents/t1/status", `{"status":"running","detail":"reading files"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("POST status = %d: %s", w.Code, w.Body.String())
	}
	a, ok := hub.Agent("t1")
	if !ok {
		t.Fatal("agent t1 was not created")
	}
	if a.Status != StatusRunning || a.Detail != "reading files" {
		t.Errorf("agent = %+v, want running/reading files", a)
	}
}

func TestPostAgentStatus_RejectsUnknownStatus(t *testing.T) {
	s, _ := newTestServer(t)
	w := do(t, s, "POST", "/api/agents/t1/status", `{"status":"exploded"}`)
	if w.Code != http.StatusBadRequest {
		t.Errorf("POST status = %d, want 400 for an unknown status", w.Code)
	}
}

func TestPostAgentStatus_RejectsBadJSON(t *testing.T) {
	s, _ := newTestServer(t)
	w := do(t, s, "POST", "/api/agents/t1/status", `{nope`)
	if w.Code != http.StatusBadRequest {
		t.Errorf("POST status = %d, want 400 for malformed JSON", w.Code)
	}
}

// --- messages ---

func TestPostMessage_Directed(t *testing.T) {
	s, hub := newTestServer(t)
	hub.SetStatus("t1", StatusIdle, "")
	hub.SetStatus("t2", StatusIdle, "")

	w := do(t, s, "POST", "/api/messages", `{"from":"t1","to":"t2","kind":"handoff","body":"take it"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("POST message = %d: %s", w.Code, w.Body.String())
	}
	m := decode[Message](t, w)
	if m.From != "t1" || m.To != "t2" || m.Body != "take it" {
		t.Errorf("message = %+v", m)
	}
	if m.ID == "" {
		t.Error("message was not assigned an id")
	}
}

func TestPostMessage_BroadcastNeedsNoRecipient(t *testing.T) {
	s, hub := newTestServer(t)
	w := do(t, s, "POST", "/api/messages", `{"from":"orchestrator","body":"stop"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("broadcast = %d: %s", w.Code, w.Body.String())
	}
	m := decode[Message](t, w)
	if m.To != "" {
		t.Errorf("To = %q, want empty for a broadcast", m.To)
	}
	_ = hub
}

func TestPostMessage_RejectsUnknownRecipient(t *testing.T) {
	// Silently dropping a message addressed to a task that
	// does not exist would make a typo look like a delivered
	// handoff.
	s, _ := newTestServer(t)
	w := do(t, s, "POST", "/api/messages", `{"from":"t1","to":"ghost","body":"hi"}`)
	if w.Code != http.StatusBadRequest {
		t.Errorf("POST message = %d, want 400 for an unknown recipient", w.Code)
	}
}

func TestPostMessage_RequiresFromAndBody(t *testing.T) {
	s, _ := newTestServer(t)
	if w := do(t, s, "POST", "/api/messages", `{"body":"hi"}`); w.Code != http.StatusBadRequest {
		t.Errorf("missing from = %d, want 400", w.Code)
	}
	hub := NewHub()
	hub.SetStatus("t1", StatusIdle, "")
	_ = s
	if got := NewHub().Messages(); len(got) != 0 {
		t.Error("a fresh hub must have no messages")
	}
}

func TestPostMessage_EmptyBodyRejected(t *testing.T) {
	s, hub := newTestServer(t)
	// Whitespace-only is as empty as empty: a stray space
	// from the compose box must not post a blank message.
	for _, body := range []string{
		`{"from":"t1","body":""}`,
		`{"from":"t1","body":"   "}`,
	} {
		w := do(t, s, "POST", "/api/messages", body)
		if w.Code == http.StatusOK {
			t.Errorf("POST %s was accepted, want a rejection", body)
		}
	}
	if len(hub.Messages()) != 0 {
		t.Errorf("messages = %d, want 0 after empty posts", len(hub.Messages()))
	}
}

// --- layout ---

func TestGetLayout_ReturnsDefault(t *testing.T) {
	s, _ := newTestServer(t)
	w := do(t, s, "GET", "/api/layout", "")
	if w.Code != http.StatusOK {
		t.Fatalf("GET layout = %d", w.Code)
	}
	l := decode[Layout](t, w)
	if len(l.Panels) != len(AllPanels) {
		t.Errorf("panels = %d, want %d", len(l.Panels), len(AllPanels))
	}
}

func TestPutLayout_Saves(t *testing.T) {
	s, hub := newTestServer(t)
	body := `{"panels":[{"id":"agents","visible":false,"order":0,"width":6}],` +
		`"theme":"light","compact":true}`
	w := do(t, s, "PUT", "/api/layout", body)
	if w.Code != http.StatusOK {
		t.Fatalf("PUT layout = %d: %s", w.Code, w.Body.String())
	}
	got := hub.Layout()
	if got.Theme != "light" || !got.Compact {
		t.Errorf("layout = %+v, want light/compact", got)
	}
	// The omitted panels must be added back, not dropped.
	if len(got.Panels) != len(AllPanels) {
		t.Errorf("panels = %d, want %d after normalisation", len(got.Panels), len(AllPanels))
	}
	var agents PanelState
	for _, p := range got.Panels {
		if p.ID == PanelAgents {
			agents = p
		}
	}
	if agents.Visible {
		t.Error("agents should be hidden after saving visible=false")
	}
	if agents.Width != 6 {
		t.Errorf("width = %d, want 6", agents.Width)
	}
}

func TestPutLayout_RejectsUnknownPanel(t *testing.T) {
	s, _ := newTestServer(t)
	w := do(t, s, "PUT", "/api/layout", `{"panels":[{"id":"nonsense","visible":true}]}`)
	if w.Code != http.StatusBadRequest {
		t.Errorf("PUT layout = %d, want 400 for an unknown panel", w.Code)
	}
}

// TestPutLayout_ClampsWidth covers the mis-drag case: a
// width outside the grid is clamped rather than rejected, so
// saving the rest of the layout still works.
func TestPutLayout_ClampsWidth(t *testing.T) {
	s, hub := newTestServer(t)
	w := do(t, s, "PUT", "/api/layout", `{"panels":[{"id":"agents","width":99,"visible":true}]}`)
	if w.Code != http.StatusOK {
		t.Fatalf("PUT layout = %d: %s", w.Code, w.Body.String())
	}
	for _, p := range hub.Layout().Panels {
		if p.Width > MaxWidth {
			t.Errorf("panel %s width = %d, want <= %d", p.ID, p.Width, MaxWidth)
		}
	}
}

func TestPutLayout_RejectsTooManyPanels(t *testing.T) {
	s, _ := newTestServer(t)
	var sb strings.Builder
	sb.WriteString(`{"panels":[`)
	for i := 0; i < MaxPanels+1; i++ {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(`{"id":"agents","visible":true}`)
	}
	sb.WriteString(`]}`)
	w := do(t, s, "PUT", "/api/layout", sb.String())
	if w.Code != http.StatusBadRequest {
		t.Errorf("PUT layout = %d, want 400 past the panel cap", w.Code)
	}
}

// --- hub behaviour exercised through the API ---

// TestMessages_HistoryIsBounded pins the retention cap so a
// long-running UI cannot grow without limit.
func TestMessages_HistoryIsBounded(t *testing.T) {
	hub := NewHub()
	for i := 0; i < DefaultMsgLimit+25; i++ {
		hub.Post(Message{From: "t1", Body: "m"})
	}
	got := hub.Messages()
	if len(got) != DefaultMsgLimit {
		t.Errorf("messages = %d, want %d", len(got), DefaultMsgLimit)
	}
}

func TestAgents_SortedByID(t *testing.T) {
	hub := NewHub()
	for _, id := range []string{"t3", "t1", "t2"} {
		hub.SetStatus(id, StatusIdle, "")
	}
	got := hub.Agents()
	want := []string{"t1", "t2", "t3"}
	for i, a := range got {
		if a.ID != want[i] {
			t.Errorf("agents[%d] = %s, want %s", i, a.ID, want[i])
		}
	}
}

func TestHub_NilSafe(t *testing.T) {
	// A nil hub must not panic: the CLI builds one
	// unconditionally, but a test may pass nil.
	var h *Hub
	h.SetStatus("t1", StatusRunning, "")
	h.Post(Message{From: "t1", Body: "x"})
	_ = h.Agents()
	_ = h.Messages()
	_ = h.Snapshot()
	_ = h.Layout()
}

// TestHub_ConcurrentUse is the race gate. The orchestrator
// writes from its task loop while HTTP handlers read, so the
// hub is exercised from many goroutines under -race.
func TestHub_ConcurrentUse(t *testing.T) {
	hub := NewHub()
	done := make(chan struct{})
	for w := 0; w < 8; w++ {
		go func(w int) {
			defer func() { done <- struct{}{} }()
			for i := 0; i < 50; i++ {
				id := "t" + strconv.Itoa(w)
				hub.SetStatus(id, StatusRunning, "x")
				hub.Post(Message{From: id, Body: "hello"})
				_ = hub.Agents()
				_ = hub.Snapshot()
				_ = hub.Layout()
			}
		}(w)
	}
	for w := 0; w < 8; w++ {
		<-done
	}
}

// --- SSE ---

// TestEvents_StreamsSnapshotAndDeltas verifies the stream
// opens with a snapshot and then carries updates. The
// request is cancelled by the client context so the handler
// returns.
func TestEvents_StreamsSnapshotAndDeltas(t *testing.T) {
	s, hub := newTestServer(t)
	r := httptest.NewRequest("GET", "/api/events", nil)
	ctx, cancel := contextWithCancel()
	r = r.WithContext(ctx)

	w := newFlushRecorder()
	done := make(chan struct{})
	go func() {
		s.Handler().ServeHTTP(w, r)
		close(done)
	}()

	// Give the handler time to write the snapshot before
	// mutating.
	waitFor(t, w, "event: snapshot")

	hub.SetStatus("t1", StatusRunning, "")
	waitFor(t, w, "event: agent")

	hub.Post(Message{From: "t1", Body: "hi"})
	waitFor(t, w, "event: message")

	cancel()
	<-done

	out := w.String()
	if !strings.Contains(out, "event: snapshot") {
		t.Error("stream did not open with a snapshot")
	}
	if !strings.Contains(out, `"id":"t1"`) {
		t.Error("stream did not carry the agent delta")
	}
	if !strings.Contains(out, "event: message") {
		t.Error("stream did not carry the message delta")
	}
}

// TestEvents_ContentType covers the handshake a browser
// EventSource requires.
//
// The handler runs until the client context is cancelled, so
// the request is served on its own goroutine and the cancel
// is fired from a test helper goroutine: calling cancel via
// defer AFTER ServeHTTP returns would deadlock, because
// ServeHTTP is precisely what waits for the cancel.
func TestEvents_ContentType(t *testing.T) {
	s, _ := newTestServer(t)
	r := httptest.NewRequest("GET", "/api/events", nil)
	ctx, cancel := contextWithCancel()
	r = r.WithContext(ctx)

	w := newFlushRecorder()
	// Cancel shortly after the handler has written its
	// headers, then wait for it to return.
	go func() {
		waitForQuiet(w, "event: snapshot", 2*time.Second)
		cancel()
	}()
	s.Handler().ServeHTTP(w, r)

	if ct := w.Header().Get("Content-Type"); ct != "text/event-stream" {
		t.Errorf("Content-Type = %q, want text/event-stream", ct)
	}
	if cc := w.Header().Get("Cache-Control"); cc != "no-cache" {
		t.Errorf("Cache-Control = %q, want no-cache", cc)
	}
}
