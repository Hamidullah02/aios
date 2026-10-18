// Package approvals is the approval service for aios.
//
// The service owns the lifecycle of a single approval:
//
//	pending -> granted   (operator says y)
//	pending -> denied    (operator says n)
//	pending -> expired   (TTL fires before either)
//
// A grant is honored only when the payload's BoundTo matches the
// call-site context. The service writes one audit row per state
// transition. The audit row is content-addressed, so a re-grant
// of the same approval is a no-op (idempotency) and a re-grant
// against a tampered payload is rejected (binding).
//
// The service is in-memory. Persistence is the audit log; the
// service's only durable state is what the audit log holds. A
// restart loses the in-flight waiters; the in-flight approvals
// can be re-loaded from the audit log by replaying the
// `approval_requested` rows and the matching
// `approval_granted` / `approval_denied` / `approval_expired`
// rows in sequence.
package approvals

import (
	"encoding/json"
	"fmt"

	"github.com/sadi/aios-system/internal/core/schemas"
)

// State is the lifecycle state of one approval.
type State string

const (
	StatePending State = "pending"
	StateGranted State = "granted"
	StateDenied  State = "denied"
	StateExpired State = "expired"
)

// IsValid reports whether s is one of the four known states.
func (s State) IsValid() bool {
	switch s {
	case StatePending, StateGranted, StateDenied, StateExpired:
		return true
	}
	return false
}

// IsTerminal reports whether the state is final. Granted, denied,
// and expired are terminal; pending is the only non-terminal state.
func (s State) IsTerminal() bool {
	return s != StatePending
}

// Approval is the in-memory record of one approval request.
//
// The struct is the source of truth for the in-flight state. The
// audit log is the source of truth for the historical record.
type Approval struct {
	ID        string                  `json:"id"`
	State     State                   `json:"state"`
	Payload   schemas.ApprovalPayload `json:"payload"`
	Actor     string                  `json:"actor"`
	CreatedAt string                  `json:"created_at"`
	UpdatedAt string                  `json:"updated_at"`
	TTL       int                     `json:"ttl,omitempty"`
}

// Validate checks the approval is well-formed. It is the same
// validation that schemas.ApprovalPayload.Validate runs, plus a
// check on the ID and the actor.
func (a *Approval) Validate() error {
	if a == nil {
		return fmt.Errorf("approvals: nil approval")
	}
	if a.ID == "" {
		return fmt.Errorf("approvals: id is required")
	}
	if a.Actor == "" {
		return fmt.Errorf("approvals: actor is required")
	}
	if !a.State.IsValid() {
		return fmt.Errorf("approvals: invalid state %q", a.State)
	}
	return a.Payload.Validate()
}

// MarshalJSON serializes the approval in canonical form.
func (a *Approval) MarshalJSON() ([]byte, error) {
	type alias Approval
	return json.Marshal((*alias)(a))
}
