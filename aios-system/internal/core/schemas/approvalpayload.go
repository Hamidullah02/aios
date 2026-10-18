package schemas

import "encoding/json"

// ApprovalPayload is the body of a pending-approval request.
//
// Per REQ-CORE-SCH-007: validates action, target, risk_tier, reason.
// aios-approvals consumes and produces this; aios-core only defines
// the type and validates it.
type ApprovalPayload struct {
	ApprovalID string `json:"approval_id"`
	Action     string `json:"action"`
	Target     string `json:"target"`
	RiskTier   int    `json:"risk_tier"`
	Reason     string `json:"reason"`
	Scope      string `json:"scope,omitempty"`
	// RequesterID is the agent or operator that requested approval.
	RequesterID string `json:"requester_id"`
	// BoundTo is the binding for the approval. When non-nil,
	// aios-approvals checks every non-empty field against the
	// call-site context (task, tool, actor, resource) before
	// honoring a grant. See aios-approvals SRS §5.4
	// REQ-APR-BND-001 and the BoundTo type.
	BoundTo *BoundTo `json:"bound_to,omitempty"`
}

func (a *ApprovalPayload) Validate() error {
	return JoinValidationErrors(
		UUIDv4("approval_id", a.ApprovalID),
		RequiredString("action", a.Action),
		RequiredString("target", a.Target),
		InRange("risk_tier", a.RiskTier, 0, 3),
		RequiredString("reason", a.Reason),
		OptionalString("scope", a.Scope),
		RequiredString("requester_id", a.RequesterID),
	)
}

func (a *ApprovalPayload) MarshalJSON() ([]byte, error) {
	type alias ApprovalPayload
	return MarshalCanonical((*alias)(a))
}

func (a *ApprovalPayload) UnmarshalJSON(data []byte) error {
	type alias ApprovalPayload
	var al alias
	if err := UnmarshalStrict(data, &al, []string{"approval_id", "action", "target", "risk_tier", "reason", "scope", "requester_id", "bound_to"}); err != nil {
		return err
	}
	*a = ApprovalPayload(al)
	return nil
}

var (
	_ Schema           = (*ApprovalPayload)(nil)
	_ json.Marshaler   = (*ApprovalPayload)(nil)
	_ json.Unmarshaler = (*ApprovalPayload)(nil)
)
