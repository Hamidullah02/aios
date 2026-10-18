package schemas

import "encoding/json"

// DecisionTrace is the per-step reasoning record.
//
// Per REQ-CORE-SCH-005: validates step_id, chosen action, alternatives,
// and the basis. DecisionTrace is what the writer (`internal/core/audit`)
// persists; this file only defines the shape.
type DecisionTrace struct {
	StepID        string         `json:"step_id"`
	Chosen        string         `json:"chosen"`
	Alternatives  []Alternative  `json:"alternatives"`
	Basis         string         `json:"basis"`
	PolicyRefs    []string       `json:"policy_refs,omitempty"`
	RiskTier      int            `json:"risk_tier"`
	RecordedInput map[string]any `json:"recorded_input,omitempty"`
}

// Alternative is one option the planner considered.
type Alternative struct {
	Action   string  `json:"action"`
	Score    float64 `json:"score"`
	Rejected string  `json:"rejected"`
}

func (d *DecisionTrace) Validate() error {
	return JoinValidationErrors(
		RequiredString("step_id", d.StepID),
		RequiredString("chosen", d.Chosen),
		RequiredString("basis", d.Basis),
		InRange("risk_tier", d.RiskTier, 0, 3),
		validateAlternatives(d.Alternatives),
	)
}

func validateAlternatives(alts []Alternative) error {
	for i, a := range alts {
		base := "alternatives." + itoa(i)
		if err := JoinValidationErrors(
			RequiredString(base+".action", a.Action),
			RequiredString(base+".rejected", a.Rejected),
		); err != nil {
			return err
		}
	}
	return nil
}

func (d *DecisionTrace) MarshalJSON() ([]byte, error) {
	type alias DecisionTrace
	return MarshalCanonical((*alias)(d))
}

func (d *DecisionTrace) UnmarshalJSON(data []byte) error {
	type alias DecisionTrace
	var al alias
	if err := UnmarshalStrict(data, &al, []string{"step_id", "chosen", "alternatives", "basis", "policy_refs", "risk_tier", "recorded_input"}); err != nil {
		return err
	}
	*d = DecisionTrace(al)
	return nil
}

func (a *Alternative) Validate() error {
	return JoinValidationErrors(
		RequiredString("action", a.Action),
		RequiredString("rejected", a.Rejected),
	)
}

func (a *Alternative) MarshalJSON() ([]byte, error) {
	type alias Alternative
	return MarshalCanonical((*alias)(a))
}

func (a *Alternative) UnmarshalJSON(data []byte) error {
	type alias Alternative
	var al alias
	if err := UnmarshalStrict(data, &al, []string{"action", "score", "rejected"}); err != nil {
		return err
	}
	*a = Alternative(al)
	return nil
}

var (
	_ Schema           = (*DecisionTrace)(nil)
	_ Schema           = (*Alternative)(nil)
	_ json.Marshaler   = (*DecisionTrace)(nil)
	_ json.Unmarshaler = (*DecisionTrace)(nil)
	_ json.Marshaler   = (*Alternative)(nil)
	_ json.Unmarshaler = (*Alternative)(nil)
)
