package cli

import (
	"sync"

	"github.com/sadi/aios-system/internal/approvals"
	"github.com/sadi/aios-system/internal/core/schemas"
	"github.com/sadi/aios-system/internal/policy"
	"github.com/sadi/aios-system/internal/runtime"
	"github.com/sadi/aios-system/internal/tools"
)

// grantStore remembers the approvals an operator gave during
// THIS run so the bus can let the retried call through.
//
// It exists because of how the retry loop works. The
// orchestrator cannot un-ask a question, so an `ask`
// decision is answered by re-running the same task: the
// worker dispatches the identical call, the gateway returns
// the identical decision, and the bus needs to recognise
// that this exact call was already approved rather than
// asking the operator the same thing twice.
//
// Keying
// ------
// Grants are keyed by the call's identity -- task, action,
// and target -- NOT by the approval id. The policy gateway
// mints a fresh UUID for every decision (gateway.go:
// approvalID := uuid.NewString()), so an id-keyed grant
// would never match the retry and the operator would be
// asked about the same command once per attempt. The
// task/action/target triple is what the operator actually
// answered a question about, and it is identical across the
// retry, so it is what a grant must be checked against.
//
// Three properties matter for safety:
//
//   - It is scoped to one run and never persisted. A grant
//     means "this human, this call, this run". It does not
//     become a standing permission and does not survive the
//     process. That is deliberately narrower than "remember
//     my answer": the policy author chose `ask` precisely
//     because this action should not be rubber-stamped by
//     history.
//
//   - The target is part of the key. A grant for one command
//     cannot authorise a different one.
//
//   - The task is part of the key, so approving a gated call
//     in one task never silently authorises the same
//     command in a later task.
//
// The approvals.Service (audit-backed, TTL'd, daemon-facing)
// remains the authority for anything outliving a single run.
// This is the in-memory shadow of one decision inside one
// run loop.
type grantStore struct {
	mu     sync.RWMutex
	grants map[string]bool
	// svc, when non-nil, records the operator's answer in the
	// audit log so an approval given at a terminal is as
	// traceable as one given over the socket.
	svc *approvals.Service
}

var _ tools.ApprovalGrantChecker = (*grantStore)(nil)

// newGrantStore returns an empty store bound to svc.
func newGrantStore(svc *approvals.Service) *grantStore {
	return &grantStore{grants: make(map[string]bool), svc: svc}
}

// grantKey is the identity of a gated call. Every field a
// human was shown has to be in it: change what is being run
// and the prior answer no longer applies.
func grantKey(task, action, target string) string {
	return task + "\x00" + action + "\x00" + target
}

// keyForPayload builds the key from an approval payload. The
// requester field carries the task id on this path (the bus
// worker fills RequesterID), and the payload carries the
// action and target the operator was shown.
func keyForPayload(p *schemas.ApprovalPayload) string {
	if p == nil {
		return ""
	}
	return grantKey(p.RequesterID, p.Action, p.Target)
}

// keyForDecision builds the key from the policy decision the
// bus is currently evaluating.
func keyForDecision(d *policy.Decision) string {
	if d == nil || d.Approval == nil {
		return ""
	}
	return keyForPayload(d.Approval)
}

// record marks a key as granted.
func (g *grantStore) record(key string) {
	if g == nil || key == "" {
		return
	}
	g.mu.Lock()
	g.grants[key] = true
	g.mu.Unlock()
}

// IsApproved implements tools.ApprovalGrantChecker.
func (g *grantStore) IsApproved(d *policy.Decision) bool {
	key := keyForDecision(d)
	if g == nil || key == "" {
		return false
	}
	g.mu.RLock()
	ok := g.grants[key]
	g.mu.RUnlock()
	return ok
}

// recordServiceGrant mirrors the operator's answer into the
// approvals service for the audit trail.
//
// Best-effort by design: the run is already authorized by the
// operator answering the prompt, and failing the task
// because an audit row could not be written would turn a
// successful, approved run into an error over the one thing
// the operator cannot act on.
//
// The service issues its own approval ids, so the payload is
// requested through the service rather than re-used.
func (g *grantStore) recordServiceGrant(p *schemas.ApprovalPayload) {
	if g == nil || g.svc == nil || p == nil {
		return
	}
	id, err := g.svc.Request(*p, p.RequesterID, approvals.RequestOptions{})
	if err != nil {
		// The payload may already carry an id the service did
		// not issue (the gateway minted it). That is expected
		// on this path and is not worth surfacing.
		return
	}
	_ = g.svc.Grant(id, "operator", approvals.Context{
		Task:     p.RequesterID,
		Actor:    p.RequesterID,
		Resource: p.Target,
	})
}

// approvalPayload extracts the policy payload from an
// approval request, accepting both the pointer and the value
// form.
//
// The type is `any` because runtime.WaitingForApproval carries
// it untyped, and two producers put different things in it:
// the bus worker stores the *pointer* the policy gateway
// built (need.Approval), while a test or a daemon reply may
// hand over a value. Asserting one shape silently yields the
// zero struct for the other, which is what produced an
// approval loop with every field blank: the grant was keyed
// on task/action/target, all of them "", so it never matched
// the call being retried.
func approvalPayload(wfa *runtime.WaitingForApproval) (schemas.ApprovalPayload, bool) {
	if wfa == nil || wfa.Payload == nil {
		return schemas.ApprovalPayload{}, false
	}
	switch p := wfa.Payload.(type) {
	case schemas.ApprovalPayload:
		return p, true
	case *schemas.ApprovalPayload:
		if p == nil {
			return schemas.ApprovalPayload{}, false
		}
		return *p, true
	default:
		return schemas.ApprovalPayload{}, false
	}
}

// Grant records the operator's approval of wfa for this run.
// Called once per human "yes".
func (g *grantStore) Grant(wfa *runtime.WaitingForApproval) {
	if g == nil || wfa == nil {
		return
	}
	payload, ok := approvalPayload(wfa)
	if !ok {
		return
	}
	g.record(keyForPayload(&payload))
	g.recordServiceGrant(&payload)
}

// has reports whether this exact call was granted. Used by
// tests.
func (g *grantStore) has(task, action, target string) bool {
	if g == nil {
		return false
	}
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.grants[grantKey(task, action, target)]
}