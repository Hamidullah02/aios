package webui

import (
	"errors"
	"strings"
	"sync"
	"time"
)

// PendingApproval is one approval request an operator can
// act on from the browser.
//
// It is a webui type rather than an approvals.Approval on
// purpose. The wire shape is the UI's contract, and it must
// carry exactly what a human needs to decide: the action, the
// target, and the reason. The CLI prompt was rebuilt for the
// same reason -- an operator cannot consent to an opaque id.
type PendingApproval struct {
	// ID is the approval id, the handle a decision is made
	// against.
	ID string `json:"id"`
	// TaskID is the task that raised it, so the panel can
	// tie the request to the agent card showing it blocked.
	TaskID string `json:"task_id,omitempty"`
	// Action is what is being authorized ("shell", "write").
	Action string `json:"action"`
	// Target is what is being acted on: the command, the
	// path. This is the field an operator reads first.
	Target string `json:"target"`
	// Reason is the policy's justification.
	Reason string `json:"reason,omitempty"`
	// Actor requested the approval.
	Actor string `json:"actor,omitempty"`
	// RiskTier is the tier from the policy decision, when set.
	RiskTier int `json:"risk_tier,omitempty"`
	// RequestedAt is when the request arrived.
	RequestedAt Time `json:"requested_at"`
}

// ApprovalResolver is the hook the server calls when an
// operator grants or denies from the browser.
//
// It returns the resulting state ("granted" / "denied"). An
// error is surfaced to the browser as-is so a refused grant
// (binding mismatch, already-decided) is visible rather than
// silently swallowed.
type ApprovalResolver interface {
	// Resolve applies the decision and reports the final state.
	Resolve(id string, approve bool) (string, error)
}

// ErrApprovalNotFound is returned when an operator acts on an
// approval the hub does not have.
//
// It is usually a stale tab: the run finished, or another
// operator decided first. The message says so, because
// "unknown approval" on its own reads like a server bug.
var ErrApprovalNotFound = errors.New("webui: no such pending approval")

// ErrNoResolver is returned when the hub has approvals to show
// but no way to resolve them.
//
// This is the `aios web` case: a separate process from the
// run, so it holds no approvals service. Showing an approve
// button that cannot work is worse than showing none, so the
// hub reports this and the UI points the operator at the
// terminal or the daemon instead.
var ErrNoResolver = errors.New("webui: this server cannot resolve approvals; use `aios run --web` or the approval daemon")

// approvalRegistry holds the pending approvals and routes
// decisions to whoever owns them.
//
// Ownership matters. `run --web` and the approval daemon each
// own their approvals in their own process, so the registry
// never decides anything itself: it records what is pending
// for display and forwards a decision to the owner. That
// keeps one source of truth for run state, which is the
// reason the server originally had no approval endpoints at
// all.
type approvalRegistry struct {
	mu       sync.RWMutex
	pending  map[string]PendingApproval
	resolver ApprovalResolver
}

// newApprovalRegistry returns an empty registry. A nil
// resolver means decisions cannot be applied here.
func newApprovalRegistry(resolver ApprovalResolver) *approvalRegistry {
	return &approvalRegistry{
		pending:  make(map[string]PendingApproval),
		resolver: resolver,
	}
}

// SetResolver attaches the owner of the approvals.
//
// It is set after NewHub rather than in the constructor
// because the run builds its approvals service and its hub in
// either order, and the hub is also created by `aios web`,
// which never has one.
func (r *approvalRegistry) SetResolver(res ApprovalResolver) {
	if r == nil {
		return
	}
	r.mu.Lock()
	r.resolver = res
	r.mu.Unlock()
}

// Add records a pending approval.
func (r *approvalRegistry) Add(a PendingApproval) {
	if r == nil || a.ID == "" {
		return
	}
	if a.RequestedAt.IsZero() {
		a.RequestedAt = NewTime(time.Now().UTC())
	}
	r.mu.Lock()
	r.pending[a.ID] = a
	r.mu.Unlock()
}

// Remove drops an approval from the pending list.
//
// Called once a decision has been made, whichever way: a
// granted approval still leaves the panel, because the run
// moves on to executing it and the operator no longer has
// anything to decide.
func (r *approvalRegistry) Remove(id string) {
	if r == nil || id == "" {
		return
	}
	r.mu.Lock()
	delete(r.pending, id)
	r.mu.Unlock()
}

// Pending returns every pending approval, oldest first, so
// the operator sees the one that has been waiting longest at
// the top.
func (r *approvalRegistry) Pending() []PendingApproval {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	out := make([]PendingApproval, 0, len(r.pending))
	for _, a := range r.pending {
		out = append(out, a)
	}
	r.mu.RUnlock()
	// Insertion sort outside the lock: the pending set is a
	// handful of items, and a comparator that closes over a
	// mutex is a deadlock waiting to happen.
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j].RequestedAt.Or().Before(out[j-1].RequestedAt.Or()); j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

// Has reports whether id is pending.
func (r *approvalRegistry) Has(id string) bool {
	if r == nil {
		return false
	}
	r.mu.RLock()
	_, ok := r.pending[id]
	r.mu.RUnlock()
	return ok
}

// CanResolve reports whether a decision can be applied here.
func (r *approvalRegistry) CanResolve() bool {
	if r == nil {
		return false
	}
	r.mu.RLock()
	res := r.resolver
	r.mu.RUnlock()
	return res != nil
}

// Decide applies an operator's decision.
//
// The approval leaves the pending list only after the resolver
// has accepted it. If the resolver refuses (the run already
// moved on, the binding no longer matches), the entry stays so
// the browser shows the refusal instead of an item that
// silently vanished.
func (r *approvalRegistry) Decide(id string, approve bool) (string, error) {
	if r == nil {
		return "", ErrNoResolver
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return "", errors.New("webui: approval id is required")
	}
	r.mu.RLock()
	res := r.resolver
	r.mu.RUnlock()
	if res == nil {
		return "", ErrNoResolver
	}
	state, err := res.Resolve(id, approve)
	if err != nil {
		return "", err
	}
	r.Remove(id)
	return state, nil
}
