package schemas

import "encoding/json"

// ScheduleSpec is the specification for a schedule.
//
// Per REQ-CORE-SCH-004: validates trigger, workflow, limits.
type ScheduleSpec struct {
	ID       string          `json:"id"`
	Trigger  ScheduleTrigger `json:"trigger"`
	Workflow WorkflowSpec    `json:"workflow"`
	Limits   ScheduleLimits  `json:"limits"`
}

// ScheduleTrigger describes when the schedule fires.
type ScheduleTrigger struct {
	Kind string `json:"kind"` // "cron" or "once"
	Cron string `json:"cron,omitempty"`
	At   string `json:"at,omitempty"` // RFC3339 timestamp, used when kind="once"
}

// ScheduleLimits are the per-run ceilings for a scheduled run.
type ScheduleLimits struct {
	MaxRuns    int `json:"max_runs,omitempty"`
	MaxWallSec int `json:"max_wall_sec,omitempty"`
	MaxCostUSD int `json:"max_cost_usd,omitempty"`
}

func (s *ScheduleSpec) Validate() error {
	return JoinValidationErrors(
		RequiredString("id", s.ID),
		s.Trigger.Validate(),
		s.Limits.Validate(),
	)
}

func (t *ScheduleTrigger) Validate() error {
	if err := OneOf("trigger.kind", t.Kind, "cron", "once"); err != nil {
		return err
	}
	switch t.Kind {
	case "cron":
		if t.Cron == "" {
			return NewValidationError("trigger.cron", "required when kind=cron")
		}
	case "once":
		if t.At == "" {
			return NewValidationError("trigger.at", "required when kind=once")
		}
	}
	return nil
}

func (l *ScheduleLimits) Validate() error {
	if l.MaxRuns < 0 {
		return NewValidationError("limits.max_runs", "must be non-negative")
	}
	if l.MaxWallSec < 0 {
		return NewValidationError("limits.max_wall_sec", "must be non-negative")
	}
	if l.MaxCostUSD < 0 {
		return NewValidationError("limits.max_cost_usd", "must be non-negative")
	}
	return nil
}

func (s *ScheduleSpec) MarshalJSON() ([]byte, error) {
	type alias ScheduleSpec
	return MarshalCanonical((*alias)(s))
}

func (s *ScheduleSpec) UnmarshalJSON(data []byte) error {
	type alias ScheduleSpec
	var al alias
	if err := UnmarshalStrict(data, &al, []string{"id", "trigger", "workflow", "limits"}); err != nil {
		return err
	}
	*s = ScheduleSpec(al)
	return nil
}

func (t *ScheduleTrigger) MarshalJSON() ([]byte, error) {
	type alias ScheduleTrigger
	return MarshalCanonical((*alias)(t))
}

func (t *ScheduleTrigger) UnmarshalJSON(data []byte) error {
	type alias ScheduleTrigger
	var al alias
	if err := UnmarshalStrict(data, &al, []string{"kind", "cron", "at"}); err != nil {
		return err
	}
	*t = ScheduleTrigger(al)
	return nil
}

func (l *ScheduleLimits) MarshalJSON() ([]byte, error) {
	type alias ScheduleLimits
	return MarshalCanonical((*alias)(l))
}

func (l *ScheduleLimits) UnmarshalJSON(data []byte) error {
	type alias ScheduleLimits
	var al alias
	if err := UnmarshalStrict(data, &al, []string{"max_runs", "max_wall_sec", "max_cost_usd"}); err != nil {
		return err
	}
	*l = ScheduleLimits(al)
	return nil
}

var (
	_ Schema           = (*ScheduleSpec)(nil)
	_ Schema           = (*ScheduleTrigger)(nil)
	_ Schema           = (*ScheduleLimits)(nil)
	_ json.Marshaler   = (*ScheduleSpec)(nil)
	_ json.Unmarshaler = (*ScheduleSpec)(nil)
	_ json.Marshaler   = (*ScheduleTrigger)(nil)
	_ json.Unmarshaler = (*ScheduleTrigger)(nil)
	_ json.Marshaler   = (*ScheduleLimits)(nil)
	_ json.Unmarshaler = (*ScheduleLimits)(nil)
)
