package cli

import (
	"context"
	"fmt"
	"io"

	"github.com/Hamidullah02/aios/aios-system/internal/webui"
)

// attachWebTaskSubmitter wires the browser's task submission
// path into a running UI server.
//
// It does three things, in an order that matters:
//
//  1. Installs the approval resolver, so a gated tool call
//     raised by a submitted task can be decided in the browser.
//  2. Installs the submitter, so a sentence in the browser can
//     become a plan and then a run.
//  3. Wires the submitter's own dependencies.
//
// The approval resolver comes first because the submitter's
// handler registers itself as the hub's resolver, and the
// composite needs that handler.
//
// A server without a planner gets no submitter. That is not a
// failure: the UI still serves, still shows whatever runs it can
// see, and says plainly that tasks cannot be started here. The
// alternative -- a submit button that returns "planner is not
// configured" after the operator has typed their request -- is
// worse.
func attachWebTaskSubmitter(hub *webui.Hub, deps *Deps, daemonRes *DaemonApprovalResolver, stderr io.Writer, url string) {
	if hub == nil {
		return
	}
	if deps == nil || deps.Planner == nil {
		// No planner: the server can watch but not start. If a
		// daemon is running, its approvals are still decidable
		// from here, so keep that path alive.
		if daemonRes != nil {
			hub.SetApprovalResolver(daemonRes)
		}
		return
	}

	// Approvals raised by a task this server runs are parked in
	// the UI and decided there. The handler registers itself as
	// the hub's resolver, which is what makes the browser's
	// Allow button work.
	local := NewUIApprovalHandler(hub, stderr, url)
	if daemonRes != nil {
		// A daemon is also running, so its approvals live in
		// another process. Both have to be reachable from one
		// button or one queue becomes a dead end. The composite
		// tries the local handler first and falls through to the
		// daemon.
		hub.SetApprovalResolver(&compositeResolver{local: local, daemon: daemonRes})
	} else {
		hub.SetApprovalResolver(local)
	}

	hub.SetSubmitter(&WebTaskSubmitter{
		Hub:      hub,
		Planner:  deps.Planner,
		Worker:   deps.Worker,
		Verify:   deps.Verifier,
		Resume:   deps.Resume,
		Budget:   deps.Budget,
		Grants:   deps.Grants,
		Audit:    deps.Audit,
		Approval: local,
		Out:      stderr,
		BaseCtx:  context.Background(),
	})
}

// compositeResolver routes an approval decision to whichever
// process owns the approval.
//
// The `web` server can be looking at two queues at once: the
// approvals its own tasks raised (parked in its own hub) and the
// approvals a run attached to the daemon raised (living in the
// daemon's process). One browser button has to reach both, or
// one of the queues becomes a dead end the operator cannot
// act on.
//
// The local handler is tried first because a locally-raised
// approval is the one this server can always resolve; the
// daemon is the fallback for approvals it does not own.
type compositeResolver struct {
	local  *UIApprovalHandler
	daemon *DaemonApprovalResolver
}

var _ webui.ApprovalResolver = (*compositeResolver)(nil)

// Resolve applies the decision to whichever owner holds it.
func (c *compositeResolver) Resolve(id string, approve bool) (string, error) {
	if c == nil {
		return "", webui.ErrNoResolver
	}
	if c.local != nil {
		state, err := c.local.Resolve(id, approve)
		if err == nil {
			return state, nil
		}
		// A local miss is not a failure: the approval may
		// belong to the daemon. Fall through and let the
		// daemon have its turn before reporting anything.
		if c.daemon == nil {
			return state, err
		}
	}
	if c.daemon != nil {
		state, err := c.daemon.Resolve(id, approve)
		if err != nil {
			// Name the approval. The daemon's own error is
			// about the socket ("socket path is empty", "the
			// daemon is not running"), and on its own it reads
			// like a server fault with no indication of what
			// the operator was trying to do.
			//
			// The ErrApprovalNotFound sentinel has to survive
			// the wrapping. The web server tells "this
			// approval is not here" (404, a stale tab) from
			// "this approval cannot be granted here" (409, a
			// configuration to change) by testing the error
			// against that sentinel. Wrapping the daemon's
			// socket error with the id alone dropped the
			// sentinel, so a second click on an already
			// decided approval came back 409 -- "try again"
			// for an approval that was never coming back.
			//
			// Both errors are kept: the daemon's own message
			// says what went wrong, and the sentinel says what
			// kind of failure it is.
			return "", fmt.Errorf("%w: approval %s: %v", webui.ErrApprovalNotFound, id, err)
		}
		return state, nil
	}
	return "", fmt.Errorf("%w: %s is not waiting on this server", webui.ErrApprovalNotFound, id)
}
