package schemas

import "encoding/json"

// WorkflowSpec is the specification for a workflow (a graph of steps).
//
// Per REQ-CORE-SCH-003: validates steps, transitions, inputs.
type WorkflowSpec struct {
	ID          string               `json:"id"`
	Name        string               `json:"name"`
	Steps       []WorkflowStep       `json:"steps"`
	Initial     string               `json:"initial,omitempty"`
	Transitions []WorkflowTransition `json:"transitions"`
	Inputs      map[string]any       `json:"inputs,omitempty"`
}

type WorkflowStep struct {
	ID   string `json:"id"`
	Kind string `json:"kind"` // e.g. "tool", "approve", "branch"
	Tool string `json:"tool,omitempty"`
	Spec string `json:"spec,omitempty"`
}

type WorkflowTransition struct {
	From string `json:"from"`
	To   string `json:"to"`
	When string `json:"when,omitempty"` // optional event/guard name
}

func (w *WorkflowSpec) Validate() error {
	return JoinValidationErrors(
		RequiredString("id", w.ID),
		RequiredString("name", w.Name),
		validateWorkflowSteps(w.Steps),
		validateWorkflowTransitions(w.Transitions, w.Steps),
	)
}

func validateWorkflowSteps(steps []WorkflowStep) error {
	if len(steps) == 0 {
		return NewValidationError("steps", "required, at least one step")
	}
	seen := make(map[string]bool, len(steps))
	for i, s := range steps {
		base := "steps." + itoa(i)
		if seen[s.ID] {
			return NewValidationError(base+".id", "duplicate step id: "+s.ID)
		}
		seen[s.ID] = true
		if err := JoinValidationErrors(
			RequiredString(base+".id", s.ID),
			RequiredString(base+".kind", s.Kind),
			OptionalString(base+".tool", s.Tool),
			OptionalString(base+".spec", s.Spec),
			OneOf(base+".kind", s.Kind, "tool", "approve", "branch", "end"),
		); err != nil {
			return err
		}
	}
	return nil
}

func validateWorkflowTransitions(ts []WorkflowTransition, steps []WorkflowStep) error {
	known := make(map[string]bool, len(steps))
	for _, s := range steps {
		known[s.ID] = true
	}
	for i, t := range ts {
		base := "transitions." + itoa(i)
		if err := JoinValidationErrors(
			RequiredString(base+".from", t.From),
			RequiredString(base+".to", t.To),
			OptionalString(base+".when", t.When),
		); err != nil {
			return err
		}
		if !known[t.From] {
			return NewValidationError(base+".from", "unknown step id: "+t.From)
		}
		if !known[t.To] {
			return NewValidationError(base+".to", "unknown step id: "+t.To)
		}
	}
	return nil
}

func (w *WorkflowSpec) MarshalJSON() ([]byte, error) {
	type alias WorkflowSpec
	return MarshalCanonical((*alias)(w))
}

func (w *WorkflowSpec) UnmarshalJSON(data []byte) error {
	type alias WorkflowSpec
	var al alias
	if err := UnmarshalStrict(data, &al, []string{"id", "name", "steps", "initial", "transitions", "inputs"}); err != nil {
		return err
	}
	*w = WorkflowSpec(al)
	return nil
}

func (s *WorkflowStep) Validate() error {
	return JoinValidationErrors(
		RequiredString("id", s.ID),
		RequiredString("kind", s.Kind),
		OptionalString("tool", s.Tool),
		OptionalString("spec", s.Spec),
		OneOf("kind", s.Kind, "tool", "approve", "branch", "end"),
	)
}

func (s *WorkflowStep) MarshalJSON() ([]byte, error) {
	type alias WorkflowStep
	return MarshalCanonical((*alias)(s))
}

func (s *WorkflowStep) UnmarshalJSON(data []byte) error {
	type alias WorkflowStep
	var al alias
	if err := UnmarshalStrict(data, &al, []string{"id", "kind", "tool", "spec"}); err != nil {
		return err
	}
	*s = WorkflowStep(al)
	return nil
}

func (t *WorkflowTransition) Validate() error {
	return JoinValidationErrors(
		RequiredString("from", t.From),
		RequiredString("to", t.To),
		OptionalString("when", t.When),
	)
}

func (t *WorkflowTransition) MarshalJSON() ([]byte, error) {
	type alias WorkflowTransition
	return MarshalCanonical((*alias)(t))
}

func (t *WorkflowTransition) UnmarshalJSON(data []byte) error {
	type alias WorkflowTransition
	var al alias
	if err := UnmarshalStrict(data, &al, []string{"from", "to", "when"}); err != nil {
		return err
	}
	*t = WorkflowTransition(al)
	return nil
}

// itoa is a small helper to avoid pulling in strconv for one place.
func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	neg := false
	if i < 0 {
		neg = true
		i = -i
	}
	var b [20]byte
	pos := len(b)
	for i > 0 {
		pos--
		b[pos] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		pos--
		b[pos] = '-'
	}
	return string(b[pos:])
}

var (
	_ Schema           = (*WorkflowSpec)(nil)
	_ Schema           = (*WorkflowStep)(nil)
	_ Schema           = (*WorkflowTransition)(nil)
	_ json.Marshaler   = (*WorkflowSpec)(nil)
	_ json.Unmarshaler = (*WorkflowSpec)(nil)
	_ json.Marshaler   = (*WorkflowStep)(nil)
	_ json.Unmarshaler = (*WorkflowStep)(nil)
	_ json.Marshaler   = (*WorkflowTransition)(nil)
	_ json.Unmarshaler = (*WorkflowTransition)(nil)
)
