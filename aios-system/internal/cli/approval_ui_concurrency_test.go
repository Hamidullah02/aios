package cli

import (
	"bytes"
	"context"
	"sync"
	"testing"
	"time"

	"github.com/Hamidullah02/aios/aios-system/internal/webui"
)

// TestUIApproval_ConcurrentDecisionsAndReads stresses the one
// place this feature adds a genuinely new concurrency edge: the
// HTTP handlers read and write the hub while the run's Approve
// call is blocked on a channel and the run loop is otherwise
// live.
//
// `go test -race` is the tool for this. It needs cgo and a C
// compiler; on Windows the toolchain here finds one under
// msys64 (ucrt64\bin\gcc.exe), so the race detector does run.
// The test is still worth keeping as a plain test: it asserts
// the queue drains and the waiters clear, which the race
// detector does not check.
func TestUIApproval_ConcurrentDecisionsAndReads(t *testing.T) {
	hub := webui.NewHub()
	// A sync buffer, not a bytes.Buffer: eight approvals write to
	// the same writer at once, and a bytes.Buffer corrupts under
	// concurrent writes even when the handler's own lock is in
	// place.
	out := &syncBuffer{}
	h := NewUIApprovalHandler(hub, out, "http://x/")

	const approvals = 8
	var wg sync.WaitGroup

	// One run-side waiter per approval, all in flight at once.
	for i := 0; i < approvals; i++ {
		id := "apr-" + string(rune('a'+i))
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_, _ = h.Approve(ctx, uiWFA(id))
		}()
	}

	// Readers hammering the hub the way the HTTP layer does.
	stop := make(chan struct{})
	var readers sync.WaitGroup
	for i := 0; i < 4; i++ {
		readers.Add(1)
		go func() {
			defer readers.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				_ = hub.PendingApprovals()
				_ = hub.Snapshot()
				_ = hub.CanResolveApprovals()
			}
		}()
	}

	// Wait for everything to be visible, then decide them all
	// concurrently.
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if len(hub.PendingApprovals()) == approvals {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if n := len(hub.PendingApprovals()); n != approvals {
		close(stop)
		readers.Wait()
		wg.Wait()
		t.Fatalf("pending = %d, want %d", n, approvals)
	}

	for i := 0; i < approvals; i++ {
		id := "apr-" + string(rune('a'+i))
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = hub.DecideApproval(id, true)
		}()
	}

	wg.Wait()
	close(stop)
	readers.Wait()

	if n := len(hub.PendingApprovals()); n != 0 {
		t.Errorf("pending after every decision = %d, want 0", n)
	}
	// The run side must have no waiters left either.
	h.mu.Lock()
	left := len(h.waits)
	h.mu.Unlock()
	if left != 0 {
		t.Errorf("run-side waiters left = %d, want 0", left)
	}
}

// TestUIApproval_DecideAfterTimeoutIsRefused pins the case that
// motivated the buffered channel: a click that lands after the
// run gave up must not panic or be treated as a decision.
func TestUIApproval_DecideAfterTimeoutIsRefused(t *testing.T) {
	hub := webui.NewHub()
	h := NewUIApprovalHandler(hub, &bytes.Buffer{}, "http://x/")

	// Cancel the context so Approve returns at once, as the
	// timeout path does.
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		_, _ = h.Approve(ctx, uiWFA("apr-1"))
		close(done)
	}()
	for i := 0; i < 200 && len(hub.PendingApprovals()) == 0; i++ {
		time.Sleep(5 * time.Millisecond)
	}
	cancel()
	<-done

	// The late click must be refused, not panic on a send to a
	// channel nobody reads.
	if _, err := h.Resolve("apr-1", true); err == nil {
		t.Error("a click after the run moved on must be refused")
	}
	if n := len(hub.PendingApprovals()); n != 0 {
		t.Errorf("pending after the run gave up = %d, want 0", n)
	}
}
