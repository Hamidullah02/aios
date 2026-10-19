package schemas

import (
	"encoding/json"
	"fmt"
)

// TaskPlan is the top-level plan for a task.
//
// Per REQ-CORE-SCH-001: validates goal, tasks, tools, model, budget.
type TaskPlan struct {
	ID     string     `json:"id"`
	Goal   string     `json:"goal"`
	Tasks  []TaskSpec `json:"tasks"`
	Tools  []string   `json:"tools,omitempty"`
	Model  string     `json:"model,omitempty"`
	Budget *Budget    `json:"budget,omitempty"`
}

// TaskSpec is one task inside a TaskPlan.
type TaskSpec struct {
	ID          string   `json:"id"`
	Description string   `json:"description"`
	Tool        string   `json:"tool,omitempty"`
	Inputs      []string `json:"inputs,omitempty"`
	Acceptance  string   `json:"acceptance"`
}

// Budget is the per-task budget.
type Budget struct {
	CostUSD      float64 `json:"cost_usd" yaml:"cost_usd"`
	WallClockSec int     `json:"wall_clock_sec" yaml:"wall_clock_sec"`
}

// Validate implements Schema.
func (p *TaskPlan) Validate() error {
	return JoinValidationErrors(
		RequiredString("id", p.ID),
		RequiredString("goal", p.Goal),
		validateTaskSpecs(p.Tasks),
		validateOptionalStringList("tools", p.Tools),
		OptionalString("model", p.Model),
		validateBudget(p.Budget),
	)
}

func validateTaskSpecs(tasks []TaskSpec) error {
	if len(tasks) == 0 {
		return NewValidationError("tasks", "required, at least one task")
	}
	for i, t := range tasks {
		base := fmt.Sprintf("tasks.%d", i)
		if err := JoinValidationErrors(
			RequiredString(base+".id", t.ID),
			RequiredString(base+".description", t.Description),
			OptionalString(base+".tool", t.Tool),
			RequiredString(base+".acceptance", t.Acceptance),
		); err != nil {
			return err
		}
	}
	return nil
}

func validateOptionalStringList(path string, list []string) error {
	for i, s := range list {
		if err := OptionalString(fmt.Sprintf("%s.%d", path, i), s); err != nil {
			return err
		}
	}
	return nil
}

func validateBudget(b *Budget) error {
	if b == nil {
		return nil
	}
	if b.CostUSD < 0 {
		return NewValidationError("budget.cost_usd", "must be non-negative")
	}
	if b.WallClockSec < 0 {
		return NewValidationError("budget.wall_clock_sec", "must be non-negative")
	}
	return nil
}

// MarshalJSON encodes the TaskPlan in canonical form (sorted keys).
func (p *TaskPlan) MarshalJSON() ([]byte, error) {
	type alias TaskPlan
	return MarshalCanonical((*alias)(p))
}

// UnmarshalJSON parses a TaskPlan, rejecting unknown fields.
func (p *TaskPlan) UnmarshalJSON(data []byte) error {
	type alias TaskPlan
	var a alias
	if err := UnmarshalStrict(data, &a, []string{"id", "goal", "tasks", "tools", "model", "budget"}); err != nil {
		return err
	}
	*p = TaskPlan(a)
	return nil
}

// Ensure TaskSpec and Budget implement Schema for completeness.
var (
	_ Schema = (*TaskSpec)(nil)
	_ Schema = (*Budget)(nil)
)

func (t *TaskSpec) Validate() error {
	return JoinValidationErrors(
		RequiredString("id", t.ID),
		RequiredString("description", t.Description),
		OptionalString("tool", t.Tool),
		RequiredString("acceptance", t.Acceptance),
	)
}

func (b *Budget) Validate() error {
	return validateBudget(b)
}

func (t *TaskSpec) MarshalJSON() ([]byte, error) {
	type alias TaskSpec
	return MarshalCanonical((*alias)(t))
}

func (t *TaskSpec) UnmarshalJSON(data []byte) error {
	type alias TaskSpec
	var a alias
	if err := UnmarshalStrict(data, &a, []string{"id", "description", "tool", "inputs", "acceptance"}); err != nil {
		return err
	}
	*t = TaskSpec(a)
	return nil
}

func (b *Budget) MarshalJSON() ([]byte, error) {
	type alias Budget
	return MarshalCanonical((*alias)(b))
}

func (b *Budget) UnmarshalJSON(data []byte) error {
	type alias Budget
	var a alias
	if err := UnmarshalStrict(data, &a, []string{"cost_usd", "wall_clock_sec"}); err != nil {
		return err
	}
	*b = Budget(a)
	return nil
}

// Compile-time guard that TaskPlan also implements json.Marshaler and
// json.Unmarshaler via the methods above.
var _ json.Marshaler = (*TaskPlan)(nil)
var _ json.Unmarshaler = (*TaskPlan)(nil)
