package schemas

import "encoding/json"

// ModelCapability is the typed record of what a model can do.
//
// Per REQ-CORE-SCH-008: validates model id, class, max tokens, supports.
// aios-models uses this to advertise models to aios-optimizer and
// aios-planner.
type ModelCapability struct {
	ModelID      string   `json:"model_id"`
	Class        string   `json:"class"` // "cheap", "mid", "strong"
	MaxTokens    int      `json:"max_tokens"`
	Supports     []string `json:"supports"` // e.g. "tools", "vision", "json_mode"
	CostPer1KUSD float64  `json:"cost_per_1k_usd"`
}

func (m *ModelCapability) Validate() error {
	return JoinValidationErrors(
		RequiredString("model_id", m.ModelID),
		OneOf("class", m.Class, "cheap", "mid", "strong"),
		InRange("max_tokens", m.MaxTokens, 1, 10_000_000),
		ListOfStrings("supports", m.Supports),
	)
}

func (m *ModelCapability) MarshalJSON() ([]byte, error) {
	type alias ModelCapability
	return MarshalCanonical((*alias)(m))
}

func (m *ModelCapability) UnmarshalJSON(data []byte) error {
	type alias ModelCapability
	var al alias
	if err := UnmarshalStrict(data, &al, []string{"model_id", "class", "max_tokens", "supports", "cost_per_1k_usd"}); err != nil {
		return err
	}
	*m = ModelCapability(al)
	return nil
}

var (
	_ Schema           = (*ModelCapability)(nil)
	_ json.Marshaler   = (*ModelCapability)(nil)
	_ json.Unmarshaler = (*ModelCapability)(nil)
)
