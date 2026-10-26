package cli

import (
	"bytes"
	"context"
	"net"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sadi/aios-system/internal/approvals"
	"github.com/sadi/aios-system/internal/core/schemas"
	"github.com/sadi/aios-system/internal/runtime"
	"github.com/sadi/aios-system/internal/webui"
)

func uiWFA(id string) *runtime.WaitingForApproval {
	return &runtime.WaitingForApproval{
		ApprovalID: id,
		Decision:   "ask",
		Payload: &schemas.ApprovalPayload{
			ApprovalID:  id,
			RequesterID: "t1",
			Action:      "shell",
			Target:      "rm -rf /",
			Reason:      "gated",
		},
	}
}

// TestUIApproval_DecideFromHub is the whole point of the
// feature: an operator in a browser unblocks a run.
//
// Before this, the UI reported `blocked / waiting_for_approval`
// and offered nothing. Now the decision reaches the run.
func TestUIApproval_DecideFromHub(t *testing.T) {
	hub := webui.NewHub()
	// A sync buffer, not a bytes.Buffer: the handler writes from
	// the run's goroutine while this test reads from its own.
	out := &syncBuffer{}
	h := NewUIApprovalHandler(hub, out, "http://127.0.0.1:9999/")

	type result struct {
		ok  bool
		err error
	}
	done := make(chan result, 1)
	go func() {
		ok, err := h.Approve(context.Background(), uiWFA("apr-1"))
		done <- result{ok, err}
	}()

	// Wait for the request to appear in the hub.
	var pending []webui.PendingApproval
	for i := 0; i < 100; i++ {
		pending = hub.PendingApprovals()
		if len(pending) == 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if len(pending) != 1 {
		t.Fatalf("approval never reached the hub (%d pending)", len(pending))
	}
	// The operator must see what they are authorizing.
	if pending[0].Target != "rm -rf /" || pending[0].Action != "shell" {
		t.Errorf("pending = %+v, want the action and target", pending[0])
	}
	// And the terminal must name the address actually in use.
	if !strings.Contains(out.String(), "9999") {
		t.Errorf("prompt should name the real address; got %q", out.String())
	}

	state, err := hub.DecideApproval("apr-1", true)
	if err != nil {
		t.Fatal(err)
	}
	if state != "granted" {
		t.Errorf("state = %q", state)
	}

	select {
	case r := <-done:
		if r.err != nil {
			t.Fatal(r.err)
		}
		if !r.ok {
			t.Error("run should see the approval as granted")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Approve never returned after the decision")
	}

	// A decided approval must not linger in the queue.
	if n := len(hub.PendingApprovals()); n != 0 {
		t.Errorf("pending after decision = %d, want 0", n)
	}
}

// TestUIApproval_DenyReachesRun covers the other branch.
func TestUIApproval_DenyReachesRun(t *testing.T) {
	hub := webui.NewHub()
	h := NewUIApprovalHandler(hub, &bytes.Buffer{}, "http://x/")
	done := make(chan bool, 1)
	go func() {
		ok, _ := h.Approve(context.Background(), uiWFA("apr-1"))
		done <- ok
	}()
	for i := 0; i < 100 && len(hub.PendingApprovals()) == 0; i++ {
		time.Sleep(10 * time.Millisecond)
	}
	if _, err := hub.DecideApproval("apr-1", false); err != nil {
		t.Fatal(err)
	}
	select {
	case ok := <-done:
		if ok {
			t.Error("a denial must not read as approved")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Approve never returned after the denial")
	}
}

// TestUIApproval_StaleDecisionIsRejected covers a browser that
// clicks after the run has moved on: a stale tab must get a
// clear error, not a silent no-op.
func TestUIApproval_StaleDecisionIsRejected(t *testing.T) {
	hub := webui.NewHub()
	h := NewUIApprovalHandler(hub, &bytes.Buffer{}, "http://x/")

	done := make(chan struct{})
	go func() {
		// Decide the approval first so Approve returns, then the
		// second attempt arrives with nobody waiting.
		for i := 0; i < 100 && len(hub.PendingApprovals()) == 0; i++ {
			time.Sleep(10 * time.Millisecond)
		}
		_, _ = hub.DecideApproval("apr-1", true)
		close(done)
	}()

	ok, err := h.Approve(context.Background(), uiWFA("apr-1"))
	<-done
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Error("the first decision should have granted it")
	}
	if _, err := h.Resolve("apr-1", true); err == nil {
		t.Error("a decision for a finished approval must be refused")
	}
}

// TestUIApproval_NilIsInert keeps the defensive contract.
func TestUIApproval_NilIsInert(t *testing.T) {
	var h *UIApprovalHandler
	if _, err := h.Resolve("x", true); err == nil {
		t.Error("nil handler must refuse")
	}
	if _, err := h.Approve(context.Background(), uiWFA("x")); err == nil {
		t.Error("nil handler must refuse Approve")
	}
}

// TestUIApproval_NoResolverBlocks pins the `aios web` shape: a
// hub with no resolver cannot decide.
func TestUIApproval_NoResolverBlocks(t *testing.T) {
	hub := webui.NewHub()
	if hub.CanResolveApprovals() {
		t.Error("a bare hub should not claim it can resolve")
	}
}

// TestDaemonResolver_Resolve covers the `aios web` + daemon
// path, including the wire verb.
//
// The CLI verb is `approve`/`deny`; the protocol verbs are
// `grant`/`deny`. Getting this wrong produces "unknown kind".
func TestDaemonResolver_Resolve(t *testing.T) {
	sock := startFakeDaemon(t)

	for _, c := range []struct {
		approve bool
		want    string
	}{{true, "granted"}, {false, "denied"}} {
		// Each case needs its own pending approval: a decision
		// is a one-way transition, so the id cannot be reused.
		id := daemonRequest(t, sock, "shell", "echo hi")
		res := NewDaemonApprovalResolver(sock, &bytes.Buffer{})
		state, err := res.Resolve(id, c.approve)
		if err != nil {
			t.Fatalf("approve=%v: %v", c.approve, err)
		}
		if state != c.want {
			t.Errorf("approve=%v: state = %q, want %q", c.approve, state, c.want)
		}
	}
}

// TestDaemonResolver_SyncReplacesQueue pins that a decision
// taken elsewhere clears the card.
//
// Merging instead of replacing would leave a resolved approval
// on screen with a live "Allow once" button, inviting a second
// click for something already decided.
func TestDaemonResolver_SyncReplacesQueue(t *testing.T) {
	sock := startFakeDaemon(t)
	hub := webui.NewHub()
	res := NewDaemonApprovalResolver(sock, &bytes.Buffer{})

	// Two approvals are pending on the daemon.
	id1 := daemonRequest(t, sock, "shell", "echo one")
	id2 := daemonRequest(t, sock, "shell", "echo two")

	if _, err := res.Sync(hub); err != nil {
		t.Fatal(err)
	}
	if n := len(hub.PendingApprovals()); n != 2 {
		t.Fatalf("pending = %d, want 2", n)
	}

	// An operator grants the first one at a terminal; the
	// daemon no longer reports it.
	daemonGrant(t, sock, id1)

	if _, err := res.Sync(hub); err != nil {
		t.Fatal(err)
	}
	got := hub.PendingApprovals()
	if len(got) != 1 || got[0].ID != id2 {
		t.Errorf("after a decision elsewhere, pending = %+v, want only %s", got, id2)
	}
}

// TestDaemonResolver_SyncReportsDaemonDown covers the case
// where no daemon is running: the UI must not pretend it can
// act.
func TestDaemonResolver_SyncReportsDaemonDown(t *testing.T) {
	res := NewDaemonApprovalResolver(filepath.Join(t.TempDir(), "missing.sock"), &bytes.Buffer{})
	if _, err := res.Sync(webui.NewHub()); err == nil {
		t.Error("a missing daemon must be reported, not swallowed")
	}
	if _, err := res.Resolve("apr-1", true); err == nil {
		t.Error("resolving without a daemon must be reported")
	}
}

// daemonRequest raises one pending approval over the socket.
func daemonRequest(t *testing.T, sock, action, target string) string {
	t.Helper()
	c, err := net.Dial("unix", sock)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	payload := schemas.ApprovalPayload{
		Action:      action,
		Target:      target,
		Reason:      "gated",
		RequesterID: "t1",
	}
	if err := encJSON(c, map[string]any{
		"kind": "request", "actor": "t1", "payload": payload,
	}); err != nil {
		t.Fatal(err)
	}
	var resp map[string]any
	if err := decJSON(c, &resp); err != nil {
		t.Fatal(err)
	}
	if ok, _ := resp["ok"].(bool); !ok {
		t.Fatalf("request refused: %v", resp)
	}
	id, _ := resp["id"].(string)
	if id == "" {
		t.Fatal("no id returned")
	}
	return id
}

// daemonGrant resolves an approval out of band.
func daemonGrant(t *testing.T, sock, id string) {
	t.Helper()
	c, err := net.Dial("unix", sock)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if err := encJSON(c, map[string]any{"kind": "grant", "id": id, "actor": "operator"}); err != nil {
		t.Fatal(err)
	}
	var resp map[string]any
	if err := decJSON(c, &resp); err != nil {
		t.Fatal(err)
	}
	if ok, _ := resp["ok"].(bool); !ok {
		t.Fatalf("grant refused: %v", resp)
	}
}

// startFakeDaemon serves the daemon protocol in-process,
// backed by an approvals service so list/grant/deny behave as
// they do in production.
func startFakeDaemon(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	sock := filepath.Join(dir, "d.sock")
	log := newTestAudit(t)
	svc := approvals.New(approvals.Config{Audit: log})
	d := NewDaemon(DaemonConfig{SocketPath: sock, PidPath: filepath.Join(dir, "d.pid"), Approvals: svc})
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	// The bind error is captured rather than discarded. A daemon
	// that fails to start used to look exactly like a daemon that
	// was slow to start: both ended with "socket never appeared",
	// and the second reading sent people looking at timeouts when
	// the real cause was a refused bind.
	started := make(chan error, 1)
	go func() { started <- d.Start(ctx) }()
	t.Cleanup(func() { _ = d.Stop() })

	deadline := time.Now().Add(10 * time.Second)
	for {
		if c, err := net.Dial("unix", sock); err == nil {
			_ = c.Close()
			return sock
		}
		// A bind failure is final; there is no point waiting.
		select {
		case err := <-started:
			if err != nil {
				t.Fatalf("daemon failed to start: %v", err)
			}
			// Start returned nil, so the daemon stopped on its
			// own. That is also final.
			t.Fatal("daemon exited before the socket became reachable")
		default:
		}
		if time.Now().After(deadline) {
			t.Fatalf("daemon socket never appeared at %s (socket path length %d)",
				sock, len(sock))
		}
		time.Sleep(10 * time.Millisecond)
	}
}
