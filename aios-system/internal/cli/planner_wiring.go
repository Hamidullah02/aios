package cli

import (
	"github.com/Hamidullah02/aios/aios-system/internal/planner"
	"github.com/Hamidullah02/aios/aios-system/internal/tools"
)

// plannerToolSpecs is the planner's view of the production
// tool set.
//
// The planner does not import the bus's Tool type; it consumes
// a flat {Name, Description, InputSchema, RiskTier} map. This
// function is the bridge: it reads the bus's live tool map and
// projects it into that shape.
//
// It used to be a hand-written table of five tools that
// "mirrored" the bus by hand. The mirroring did not hold: the
// bus carried twenty-five tools and the planner knew five, so
// twenty working tools — edit, apply_patch, grep, code_search,
// git_commit and the rest — could not be named in a plan. They
// were dead code however well they worked, and `aios tools`
// listed them as though they were reachable.
//
// Deriving the list from the bus removes the drift by
// construction. A tool added to the bus is offered to the
// planner automatically; a tool removed from the bus
// disappears from the planner's prompt. The two can no longer
// disagree, because there is only one list.
//
// The tool's own Describe method supplies the description and
// the JSON schema, so the text the model reads lives in the
// same file as the code that reads the arguments. A schema
// that drifts from its implementation produces a plan that
// fails at the tool call with no hint that the model had been
// misled.
//
// A tool that does not implement tools.Describer still appears
// in the map, with a placeholder description and an empty
// schema. It is better to tell the model that a tool's
// arguments are undocumented than to hide the tool: the model
// can then choose not to use it, whereas a hidden tool is one
// the operator believes exists and cannot reach.
func plannerToolSpecs(bus *tools.Bus) map[string]planner.ToolSpec {
	out := make(map[string]planner.ToolSpec)
	if bus == nil {
		return out
	}
	for name, spec := range bus.Specs() {
		out[name] = planner.ToolSpec{
			Name:        spec.Name,
			Description: spec.Description,
			InputSchema: spec.InputSchema,
			RiskTier:    spec.Risk,
		}
	}
	return out
}
