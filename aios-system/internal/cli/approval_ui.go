package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/sadi/aios-system/internal/runtime"
	"github.com/sadi/aios-system/internal/webui"
)

// UIApprovalHandler parks an approval in the web UI and waits
// for an operator to decide there.
//
// It is the `--web` path. The alternative is the terminal
// prompt, which requires someone to be sitting at the machine
// the run is attached to. The UI is the better answer for a
// run on a background host, a dashboard opened on another
// screen, or a second administrator who is not the operator
// who started the process.
//
// Concurrency model. One handler serves the whole run, so it
// owns a map of approval id -> channel. Each Approve call
// publishes the request, waits on its own channel, and then
// deletes its entry. The hub is given a resolver bound to this
// handler, so the HTTP side never touches the run's own state:
// it hands a decision back through the channel and the run
// loop continues exactly as it would after a keystroke.
//
// Ordering is preserved. The orchestrator asks for one
// approval at a time, so a single blocking Approve per request
// is sufficient; a second concurrent request would be a
// different design and is not pretended at here.
//
// The output is written under a lock even so. The handler is
// shared, and the hub's resolver can be called from any HTTP
// goroutine, so two notes can reach the writer at once. An
// unsynchronised writer that receives two writes is a data
// race on the writer itself -- bytes.Buffer corrupts, and a
// terminal gets two half-lines. The lock also keeps one
// approval's five lines from interleaving with another's.
type UIApprovalHandler struct {
	// Hub receives the pending request and the decision.
	Hub *webui.Hub
	// Out receives a one-line note per decision, so an
	// operator watching the terminal is not left guessing
	// whether the browser click landed.
	Out io.Writer
	// URL is the address the operator should open. It is the
	// one the server actually bound to, not the default: a run
	// with --web-addr pointed elsewhere printed 7755 while
	// listening on 8761, so the hint sent the operator to a
	// port with nothing on it.
	URL string

	mu    sync.Mutex
	waits map[string]chan bool
	// outMu serialises writes to Out.
	outMu sync.Mutex
}

var (
	_ ApprovalHandler        = (*UIApprovalHandler)(nil)
	_ webui.ApprovalResolver = (*UIApprovalHandler)(nil)
)

// errUIApprovalTimeout is returned when no operator decides in
// time.
//
// It denies, for the same reason the terminal prompt denies on
// EOF: an approval nobody answered is not consent. The run
// stops with an error naming the timeout rather than hanging
// until someone notices the browser tab.
var errUIApprovalTimeout = errors.New("approval: no decision in the UI before the timeout")

// NewUIApprovalHandler returns a handler publishing into hub
// and registers hub as its decision sink.
//
// url is the address the operator should open; it is passed in
// rather than derived from a default so a run listening on a
// non-default port prints the port it is actually on.
func NewUIApprovalHandler(hub *webui.Hub, out io.Writer, url string) *UIApprovalHandler {
	h := &UIApprovalHandler{
		Hub:   hub,
		Out:   out,
		URL:   url,
		waits: make(map[string]chan bool),
	}
	// The hub calls back into this handler when the browser
	// posts a decision.
	hub.SetApprovalResolver(h)
	return h
}

// uiApprovalWait bounds how long a browser decision is awaited.
//
// It matches the terminal prompt's 60s so the two paths feel
// the same to an operator: if nobody acts within a minute, the
// run stops rather than sitting blocked indefinitely.
const uiApprovalWait = 60 * time.Second

// Approve implements ApprovalHandler.
func (h *UIApprovalHandler) Approve(ctx context.Context, wfa *runtime.WaitingForApproval) (bool, error) {
	if h == nil || wfa == nil {
		return false, errors.New("approval: nil request")
	}
	payload, _ := approvalPayload(wfa)

	// Register the waiter BEFORE publishing, so a decision that
	// arrives immediately cannot land on a channel nobody is
	// reading yet.
	ch := make(chan bool, 1)
	h.mu.Lock()
	h.waits[wfa.ApprovalID] = ch
	h.mu.Unlock()

	defer func() {
		h.mu.Lock()
		delete(h.waits, wfa.ApprovalID)
		h.mu.Unlock()
		h.Hub.RemoveApproval(wfa.ApprovalID)
	}()

	h.Hub.AddApproval(webui.PendingApproval{
		ID:       wfa.ApprovalID,
		TaskID:   payload.RequesterID,
		Action:   payload.Action,
		Target:   payload.Target,
		Reason:   payload.Reason,
		Actor:    payload.RequesterID,
		RiskTier: payload.RiskTier,
	})

	// The prompt is written as one block under the lock. A
	// lock per line would still interleave two approvals'
	// lines with each other; the point is that one approval's
	// prompt reads as one prompt.
	h.write(func(w io.Writer) {
		fmt.Fprintf(w, "\nAPPROVAL REQUIRED (decide in the web UI)\n")
		fmt.Fprintf(w, "  id     : %s\n", wfa.ApprovalID)
		if payload.Action != "" {
			fmt.Fprintf(w, "  action : %s\n", payload.Action)
		}
		if payload.Target != "" {
			fmt.Fprintf(w, "  target : %s\n", payload.Target)
		}
		if payload.Reason != "" {
			fmt.Fprintf(w, "  reason : %s\n", payload.Reason)
		}
		fmt.Fprintf(w, "  open   : %s\n", h.URL)
	})

	timer := time.NewTimer(uiApprovalWait)
	defer timer.Stop()

	select {
	case approved := <-ch:
		h.write(func(w io.Writer) {
			if approved {
				fmt.Fprintf(w, "approved in the UI: %s\n", wfa.ApprovalID)
			} else {
				fmt.Fprintf(w, "denied in the UI: %s\n", wfa.ApprovalID)
			}
		})
		return approved, nil
	case <-ctx.Done():
		return false, ctx.Err()
	case <-timer.C:
		return false, errUIApprovalTimeout
	}
}

// write runs fn against Out under the output lock.
//
// The lock is what makes the handler safe to share. The resolver
// is called from an HTTP goroutine and the run calls Approve from
// its own, so two notes can reach Out at the same moment. An
// unsynchronised write is a data race on the writer itself: a
// bytes.Buffer corrupts, and a terminal shows two half-lines.
func (h *UIApprovalHandler) write(fn func(io.Writer)) {
	if h == nil || h.Out == nil {
		return
	}
	h.outMu.Lock()
	defer h.outMu.Unlock()
	fn(h.Out)
}

// Resolve implements webui.ApprovalResolver: the browser's
// answer, handed back to the waiting run.
func (h *UIApprovalHandler) Resolve(id string, approve bool) (string, error) {
	if h == nil {
		return "", webui.ErrNoResolver
	}
	h.mu.Lock()
	ch, ok := h.waits[id]
	h.mu.Unlock()
	if !ok {
		// The run already moved on (decided elsewhere, timed
		// out, or finished). Say so plainly: this is a stale
		// browser tab, not a server fault.
		return "", fmt.Errorf("%w: %s is no longer waiting", webui.ErrApprovalNotFound, id)
	}
	// Buffered, so a late or duplicate click cannot block the
	// HTTP handler even if the run has stopped reading.
	select {
	case ch <- approve:
	default:
	}
	if approve {
		return "granted", nil
	}
	return "denied", nil
}
