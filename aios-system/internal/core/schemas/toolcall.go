package schemas

import "encoding/json"

// ToolCall is a request to a tool.
//
// Per REQ-CORE-SCH-009: validates tool name, args (as map), call_id.
type ToolCall struct {
	CallID   string         `json:"call_id"`
	Tool     string         `json:"tool"`
	Args     map[string]any `json:"args"`
	RiskTier int            `json:"risk_tier"`
}

// ToolResult is the response from a tool.
//
// Per REQ-CORE-SCH-010: validates call_id, ok, output (as map), error (if !ok).
type ToolResult struct {
	CallID     string         `json:"call_id"`
	OK         bool           `json:"ok"`
	Output     map[string]any `json:"output,omitempty"`
	Error      string         `json:"error,omitempty"`
	DurationMS int            `json:"duration_ms"`
}

func (c *ToolCall) Validate() error {
	return JoinValidationErrors(
		RequiredString("call_id", c.CallID),
		RequiredString("tool", c.Tool),
		InRange("risk_tier", c.RiskTier, 0, 3),
	)
}

func (r *ToolResult) Validate() error {
	if err := JoinValidationErrors(
		RequiredString("call_id", r.CallID),
		InRange("duration_ms", r.DurationMS, 0, 1_000_000_000),
	); err != nil {
		return err
	}
	if !r.OK && r.Error == "" {
		return NewValidationError("error", "required when ok=false")
	}
	if r.OK && r.Error != "" {
		return NewValidationError("error", "must be empty when ok=true")
	}
	return nil
}

func (c *ToolCall) MarshalJSON() ([]byte, error) {
	type alias ToolCall
	return MarshalCanonical((*alias)(c))
}

func (c *ToolCall) UnmarshalJSON(data []byte) error {
	type alias ToolCall
	var al alias
	if err := UnmarshalStrict(data, &al, []string{"call_id", "tool", "args", "risk_tier"}); err != nil {
		return err
	}
	*c = ToolCall(al)
	return nil
}

func (r *ToolResult) MarshalJSON() ([]byte, error) {
	type alias ToolResult
	return MarshalCanonical((*alias)(r))
}

func (r *ToolResult) UnmarshalJSON(data []byte) error {
	type alias ToolResult
	var al alias
	if err := UnmarshalStrict(data, &al, []string{"call_id", "ok", "output", "error", "duration_ms"}); err != nil {
		return err
	}
	*r = ToolResult(al)
	return nil
}

var (
	_ Schema           = (*ToolCall)(nil)
	_ Schema           = (*ToolResult)(nil)
	_ json.Marshaler   = (*ToolCall)(nil)
	_ json.Unmarshaler = (*ToolCall)(nil)
	_ json.Marshaler   = (*ToolResult)(nil)
	_ json.Unmarshaler = (*ToolResult)(nil)
)
