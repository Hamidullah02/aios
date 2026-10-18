package schemas

import "encoding/json"

// BoundTo is the binding for a tier-2/tier-3 approval.
//
// Per the aios-approvals SRS §5.4 REQ-APR-BND-001, an approval is
// granted only when the binding matches the call-site context
// (task, tool, actor, resource). The four fields form a
// conjunction: all of them must match. An empty field is a
// wildcard; the call-site value is accepted as long as the
// other (non-empty) fields match.
//
// The type is referenced from ApprovalPayload.BoundTo. The
// aios-approvals service compares two BoundTo values for
// equality; see internal/approvals/binding.go.
type BoundTo struct {
	Task     string `json:"task,omitempty"`
	Tool     string `json:"tool,omitempty"`
	Actor    string `json:"actor,omitempty"`
	Resource string `json:"resource,omitempty"`
}

// Match returns true when the call-site context (task, tool,
// actor, resource) satisfies every non-empty field of the
// receiver. An empty field is a wildcard. A zero-value
// BoundTo matches everything.
func (b *BoundTo) Match(task, tool, actor, resource string) bool {
	if b == nil {
		return true
	}
	if b.Task != "" && b.Task != task {
		return false
	}
	if b.Tool != "" && b.Tool != tool {
		return false
	}
	if b.Actor != "" && b.Actor != actor {
		return false
	}
	if b.Resource != "" && b.Resource != resource {
		return false
	}
	return true
}

// Validate returns nil when the struct is well-formed. All
// fields are optional; the only rejection is malformed JSON
// (handled by UnmarshalJSON).
func (b *BoundTo) Validate() error { return nil }

func (b *BoundTo) MarshalJSON() ([]byte, error) {
	type alias BoundTo
	return MarshalCanonical((*alias)(b))
}

func (b *BoundTo) UnmarshalJSON(data []byte) error {
	type alias BoundTo
	var a alias
	if err := UnmarshalStrict(data, &a, []string{"task", "tool", "actor", "resource"}); err != nil {
		return err
	}
	*b = BoundTo(a)
	return nil
}

var (
	_ Schema           = (*BoundTo)(nil)
	_ json.Marshaler   = (*BoundTo)(nil)
	_ json.Unmarshaler = (*BoundTo)(nil)
)
