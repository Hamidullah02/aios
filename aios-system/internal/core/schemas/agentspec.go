package schemas

import "encoding/json"

// AgentSpec is the specification for one agent.
//
// Per REQ-CORE-SCH-002: validates role, tools, model, system prompt.
type AgentSpec struct {
	ID           string   `json:"id"`
	Role         string   `json:"role"`
	Tools        []string `json:"tools"`
	Model        string   `json:"model"`
	SystemPrompt string   `json:"system_prompt,omitempty"`
}

func (a *AgentSpec) Validate() error {
	return JoinValidationErrors(
		RequiredString("id", a.ID),
		RequiredString("role", a.Role),
		ListOfStrings("tools", a.Tools),
		RequiredString("model", a.Model),
		OptionalString("system_prompt", a.SystemPrompt),
	)
}

func (a *AgentSpec) MarshalJSON() ([]byte, error) {
	type alias AgentSpec
	return MarshalCanonical((*alias)(a))
}

func (a *AgentSpec) UnmarshalJSON(data []byte) error {
	type alias AgentSpec
	var al alias
	if err := UnmarshalStrict(data, &al, []string{"id", "role", "tools", "model", "system_prompt"}); err != nil {
		return err
	}
	*a = AgentSpec(al)
	return nil
}

var (
	_ Schema           = (*AgentSpec)(nil)
	_ json.Marshaler   = (*AgentSpec)(nil)
	_ json.Unmarshaler = (*AgentSpec)(nil)
)
