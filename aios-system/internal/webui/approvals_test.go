package webui

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// fakeResolver records decisions and reports a scripted
// result.
type fakeResolver struct {
	state string
	err   error
	seen  []string
	calls int
}

func (f *fakeResolver) Resolve(id string, approve bool) (string, error) {
	f.calls++
	f.seen = append(f.seen, id)
	if f.err != nil {
		return "", f.err
	}
	return f.state, nil
}

// newApprovalServer returns a server over a hub wired to res.
func newApprovalServer(t *testing.T, res ApprovalResolver) *Server {
	t.Helper()
	hub := NewHub()
	if res != nil {
		hub.SetApprovalResolver(res)
	}
	srv, err := New(Config{Hub: hub, Addr: "127.0.0.1:0", Token: testToken})
	if err != nil {
		t.Fatal(err)
	}
	return srv
}

func postDecision(t *testing.T, srv *Server, id, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := authRequest(http.MethodPost, "/api/approvals/"+id, body)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	return rec
}

// TestApprovals_ListCarriesWhatOperatorNeeds pins that the
// queue names the action and the target, not just an id.
//
// An operator asked to authorize "apr-3f9c" is being asked to
// consent to a string. The target is the payload of this
// whole panel.
func TestApprovals_ListCarriesWhatOperatorNeeds(t *testing.T) {
	srv := newApprovalServer(t, &fakeResolver{state: "granted"})
	srv.hub.AddApproval(PendingApproval{
		ID:     "apr-1",
		TaskID: "t1",
		Action: "shell",
		Target: "rm -rf /",
		Reason: "shell access is gated",
	})

	req := authRequest(http.MethodGet, "/api/approvals", "")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var got struct {
		Approvals  []PendingApproval `json:"approvals"`
		CanResolve bool              `json:"can_resolve"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Approvals) != 1 {
		t.Fatalf("approvals = %d, want 1", len(got.Approvals))
	}
	a := got.Approvals[0]
	if a.Target != "rm -rf /" {
		t.Errorf("target = %q", a.Target)
	}
	if a.Action != "shell" {
		t.Errorf("action = %q", a.Action)
	}
	if a.Reason == "" {
		t.Error("reason should be carried through")
	}
	if !got.CanResolve {
		t.Error("can_resolve should be true when a resolver is attached")
	}
}

// TestApprovals_GrantThenDeny covers the two decisions.
func TestApprovals_GrantThenDeny(t *testing.T) {
	for _, c := range []struct {
		body, want string
	}{
		{`{"approve":true}`, "granted"},
		{`{"approve":false}`, "denied"},
	} {
		res := &fakeResolver{state: c.want}
		srv := newApprovalServer(t, res)
		srv.hub.AddApproval(PendingApproval{ID: "apr-1", Action: "shell", Target: "x"})
		rec := postDecision(t, srv, "apr-1", c.body)
		if rec.Code != http.StatusOK {
			t.Fatalf("body %s: status = %d (%s)", c.body, rec.Code, rec.Body.String())
		}
		if len(res.seen) != 1 || res.seen[0] != "apr-1" {
			t.Errorf("resolver saw %v", res.seen)
		}
		// A decided approval must leave the queue, or the
		// card stays on screen inviting a second click.
		if n := len(srv.hub.PendingApprovals()); n != 0 {
			t.Errorf("pending after decision = %d, want 0", n)
		}
	}
}

// TestApprovals_MissingApproveFieldIsRejected pins the
// ambiguous-body rule.
//
// Treating a missing "approve" as false would silently DENY on
// a malformed request, stopping a run nobody asked to stop.
func TestApprovals_MissingApproveFieldIsRejected(t *testing.T) {
	srv := newApprovalServer(t, &fakeResolver{state: "granted"})
	srv.hub.AddApproval(PendingApproval{ID: "apr-1"})
	for _, body := range []string{`{}`, `{"approve":"yes"}`, `not json`} {
		rec := postDecision(t, srv, "apr-1", body)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("body %q: status = %d, want 400", body, rec.Code)
		}
	}
}

// TestApprovals_NoResolverIs503 pins the `aios web` case.
//
// A server that cannot apply a decision must say so with a
// status that means "this server cannot", not "your request
// was wrong" -- the operator's only mistake was starting the
// wrong command.
func TestApprovals_NoResolverIs503(t *testing.T) {
	srv := newApprovalServer(t, nil)
	rec := postDecision(t, srv, "apr-1", `{"approve":true}`)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "run --web") {
		t.Errorf("body should name the remedy; got %s", rec.Body.String())
	}
}

// TestApprovals_StaleTabIs404 covers clicking twice, or
// clicking after the run moved on.
func TestApprovals_StaleTabIs404(t *testing.T) {
	srv := newApprovalServer(t, &fakeResolver{err: ErrApprovalNotFound})
	rec := postDecision(t, srv, "gone", `{"approve":true}`)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

// TestApprovals_ResolverRefusalKeepsCard pins that a refused
// grant does not silently drop the request.
//
// The operator needs to see the refusal and decide again;
// making the card vanish would hide a real failure.
func TestApprovals_ResolverRefusalKeepsCard(t *testing.T) {
	res := &fakeResolver{err: errors.New("binding mismatch")}
	srv := newApprovalServer(t, res)
	srv.hub.AddApproval(PendingApproval{ID: "apr-1", Action: "shell", Target: "x"})

	rec := postDecision(t, srv, "apr-1", `{"approve":true}`)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "binding mismatch") {
		t.Errorf("body should carry the reason; got %s", rec.Body.String())
	}
	if len(srv.hub.PendingApprovals()) != 1 {
		t.Error("a refused decision must leave the card on screen")
	}
}

// TestApprovals_WrongMethodIs405 pins that an API path is never
// answered with the SPA's HTML.
//
// knownAPI.Has has to recognise /api/approvals/{id} as an API
// route, or a bad method falls through to the static handler
// and returns 200 with a page of HTML.
func TestApprovals_WrongMethodIs405(t *testing.T) {
	srv := newApprovalServer(t, &fakeResolver{state: "granted"})
	req := authRequest(http.MethodGet, "/api/approvals/apr-1", "")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); strings.Contains(ct, "text/html") {
		t.Errorf("an API path answered with HTML (Content-Type %q)", ct)
	}
}

// TestSnapshot_ApprovalsAlwaysPresent pins the JS contract: the
// key is never null.
func TestSnapshot_ApprovalsAlwaysPresent(t *testing.T) {
	snap := NewHub().Snapshot()
	b, err := json.Marshal(snap)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"approvals":[]`) {
		t.Errorf("snapshot should serialise an empty array, got %s", b)
	}
	if !strings.Contains(string(b), `"can_resolve_approvals":false`) {
		t.Errorf("snapshot should carry can_resolve_approvals, got %s", b)
	}
}

// TestLayout_ApprovalsPanelIsKnown pins that the new panel is
// a first-class panel, not an orphan template.
func TestLayout_ApprovalsPanelIsKnown(t *testing.T) {
	if !KnownPanel(PanelApprovals) {
		t.Fatal("approvals panel is not registered in AllPanels")
	}
	if DefaultLayout().Panels[0].ID != PanelApprovals {
		t.Error("approvals should lead the default layout")
	}
}

// TestLayout_AddedPanelDoesNotCollide is the regression for the
// order bug found end to end.
//
// A saved layout written before the approvals panel existed has
// agents at order 0. Appending approvals with its DEFAULT
// order (also 0) collides, and the alphabetical tie-break then
// puts "agents" first -- silently demoting the panel the
// operator most needs. The appended panel must land after the
// highest order already in use.
func TestLayout_AddedPanelDoesNotCollide(t *testing.T) {
	// A layout saved by an older build: no approvals panel,
	// dense orders.
	saved := Layout{Panels: []PanelState{
		{ID: PanelAgents, Visible: true, Order: 0, Width: 12},
		{ID: PanelComms, Visible: true, Order: 1, Width: 6},
		{ID: PanelAudit, Visible: true, Order: 2, Width: 6},
	}}
	norm, err := saved.Normalize()
	if err != nil {
		t.Fatal(err)
	}
	orders := map[int]PanelID{}
	for _, p := range norm.Panels {
		if prev, dup := orders[p.Order]; dup {
			t.Fatalf("order %d used by both %q and %q", p.Order, prev, p.ID)
		}
		orders[p.Order] = p.ID
	}
	// The new panel must not outrank what the operator placed.
	if norm.Panels[0].ID != PanelAgents {
		t.Errorf("first panel = %q, want agents (the operator's own order)", norm.Panels[0].ID)
	}
}

// TestApprovalRegistry_NilSafe keeps the defensive contract:
// several subcommands hold a hub with no approvals path.
func TestApprovalRegistry_NilSafe(t *testing.T) {
	h := NewHub()
	if h.CanResolveApprovals() {
		t.Error("a bare hub resolves nothing")
	}
	if _, err := h.DecideApproval("x", true); !errors.Is(err, ErrNoResolver) {
		t.Errorf("err = %v, want ErrNoResolver", err)
	}
	var r *approvalRegistry
	if len(r.Pending()) != 0 || r.CanResolve() || r.Has("x") {
		t.Error("nil registry must be inert")
	}
}

// TestHub_AddApprovalBroadcasts pins that an open browser sees
// a new request without polling.
func TestHub_AddApprovalBroadcasts(t *testing.T) {
	h := NewHub()
	ch, cancel := h.subscribe()
	defer cancel()

	h.AddApproval(PendingApproval{ID: "apr-1", Action: "shell", Target: "x"})
	select {
	case ev := <-ch:
		if ev.Kind != "approvals" {
			t.Fatalf("event kind = %q, want approvals", ev.Kind)
		}
	case <-time.After(time.Second):
		t.Fatal("no broadcast for a new approval")
	}
}

// TestHub_SnapshotCancelable covers the SSE context path with
// approvals present, which is the shape `run --web` uses.
func TestHub_SnapshotCancelable(t *testing.T) {
	h := NewHub()
	h.SetApprovalResolver(&fakeResolver{state: "granted"})
	h.AddApproval(PendingApproval{ID: "apr-1", Action: "shell"})

	srv, err := New(Config{Hub: h, Addr: "127.0.0.1:0", Token: testToken})
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL+"/api/events", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set(TokenHeader, testToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	buf := make([]byte, 4096)
	for {
		if ctx.Err() != nil {
			return
		}
		n, err := resp.Body.Read(buf)
		if err != nil {
			return
		}
		// The first frame must already carry the pending
		// approval: a browser that connects mid-wait has to
		// see it without waiting for a change.
		if strings.Contains(string(buf[:n]), "apr-1") {
			return
		}
	}
}
