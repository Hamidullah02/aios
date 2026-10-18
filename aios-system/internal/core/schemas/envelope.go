package schemas

import "encoding/json"

// Envelope is the typed carrier between modules.
//
// Per REQ-CORE-SCH-006: Envelope wraps a value with explicit type and
// provenance. Cross-module calls move Envelopes, not raw values.
type Envelope struct {
	Type      string   `json:"type"`            // "task_plan", "decision_trace", ...
	Source    string   `json:"source"`          // module that produced it
	CreatedAt string   `json:"created_at"`      // RFC3339; the value is captured by the source's clock
	Payload   any      `json:"payload"`         // the typed value, marshaled to JSON
	Trace     []string `json:"trace,omitempty"` // provenance chain
}

func (e *Envelope) Validate() error {
	if err := JoinValidationErrors(
		RequiredString("type", e.Type),
		RequiredString("source", e.Source),
		RequiredString("created_at", e.CreatedAt),
	); err != nil {
		return err
	}
	for i, t := range e.Trace {
		if err := RequiredString("trace."+itoa(i), t); err != nil {
			return err
		}
	}
	return nil
}

func (e *Envelope) MarshalJSON() ([]byte, error) {
	type alias Envelope
	return MarshalCanonical((*alias)(e))
}

func (e *Envelope) UnmarshalJSON(data []byte) error {
	type alias Envelope
	var al alias
	if err := UnmarshalStrict(data, &al, []string{"type", "source", "created_at", "payload", "trace"}); err != nil {
		return err
	}
	*e = Envelope(al)
	return nil
}

var (
	_ Schema           = (*Envelope)(nil)
	_ json.Marshaler   = (*Envelope)(nil)
	_ json.Unmarshaler = (*Envelope)(nil)
)
