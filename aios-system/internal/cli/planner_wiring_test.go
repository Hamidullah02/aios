package cli

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Hamidullah02/aios/aios-system/internal/tools"
)

// TestPlannerToolSpecs_CoverEveryBusTool is the gate that
// keeps the planner and the bus in step.
//
// The planner used to carry a hand-written table of five tools
// while the bus carried twenty-five. Twenty working tools
// could not be named in a plan, so they were dead code however
// well they worked, and `aios tools` listed them as though
// they were reachable.
//
// The planner's list is now derived from the bus, so this test
// cannot fail by drift alone. It pins the remaining
// requirement: every tool the bus can dispatch must actually
// be described to the model. A tool that reaches the bus but
// not the planner is still unreachable.
func TestPlannerToolSpecs_CoverEveryBusTool(t *testing.T) {
	bus := testBus(t)
	specs := plannerToolSpecs(bus)

	for _, name := range bus.Names() {
		spec, ok := specs[name]
		if !ok {
			t.Errorf("tool %q is in the bus but not in the planner's tool list", name)
			continue
		}
		if spec.Name != name {
			t.Errorf("tool %q: spec name = %q", name, spec.Name)
		}
		if strings.TrimSpace(spec.Description) == "" {
			t.Errorf("tool %q: empty description", name)
		}
		if strings.TrimSpace(spec.InputSchema) == "" {
			t.Errorf("tool %q: empty input schema", name)
		}
	}
}

// TestPlannerToolSpecs_EveryToolHasARealSchema checks that
// each tool describes itself properly rather than falling
// through to the placeholder.
//
// The placeholder exists so an undocumented tool is still
// visible. But a tool that uses it is a tool the model cannot
// call correctly, so every tool we ship should implement
// Describe. This test lists the offenders by name.
func TestPlannerToolSpecs_EveryToolHasARealSchema(t *testing.T) {
	bus := testBus(t)
	specs := plannerToolSpecs(bus)

	for name, spec := range specs {
		if !strings.Contains(spec.Description, "not documented") {
			continue
		}
		t.Errorf("tool %q does not implement tools.Describer; it is "+
			"dispatchable but the model cannot call it correctly. "+
			"Add a Describe method next to its Run method.", name)
	}
}

// TestPlannerToolSpecs_SchemasAreValidJSON checks that every
// schema is parseable JSON.
//
// The schema is handed to the model as-is. A malformed one
// makes the model guess, and a guess that happens to be wrong
// produces a plan that fails at the tool call with no hint
// that the tool's own documentation was broken.
func TestPlannerToolSpecs_SchemasAreValidJSON(t *testing.T) {
	bus := testBus(t)
	specs := plannerToolSpecs(bus)

	for name, spec := range specs {
		var decoded map[string]any
		if err := json.Unmarshal([]byte(spec.InputSchema), &decoded); err != nil {
			t.Errorf("tool %q: input schema is not valid JSON: %v", name, err)
			continue
		}
		if decoded["type"] != "object" {
			t.Errorf("tool %q: schema type = %v, want object", name, decoded["type"])
		}
	}
}

// TestPlannerToolSpecs_IncludeTheDocSetTools pins the tools
// that used to be the whole list.
//
// The point of the new list is that it is bigger than the old
// one. This test names the tools that were reachable before
// and asserts they still are, so a refactor cannot quietly
// drop one of them.
func TestPlannerToolSpecs_IncludeTheDocSetTools(t *testing.T) {
	specs := plannerToolSpecs(testBus(t))

	want := []string{
		"read_file", "search_files", "write_file", "run_shell", "ask_user",
		// The tools that used to exist but could not be
		// planned.
		"edit", "apply_patch", "grep", "code_search", "git_commit",
		"git_status", "glob", "run_tests",
	}
	for _, name := range want {
		if _, ok := specs[name]; !ok {
			t.Errorf("tool %q is missing from the planner's tool list", name)
		}
	}
}

// TestPlannerToolSpecs_RiskTiersAreInRange checks that the
// risk tier a tool declares is a tier the policy gateway
// understands.
//
// A tier outside 0..3 would make the policy rules that
// reference it unreachable.
func TestPlannerToolSpecs_RiskTiersAreInRange(t *testing.T) {
	specs := plannerToolSpecs(testBus(t))

	for name, spec := range specs {
		if spec.RiskTier < 0 || spec.RiskTier > 3 {
			t.Errorf("tool %q: risk tier = %d, want 0..3", name, spec.RiskTier)
		}
	}
}

// TestPlannerToolSpecs_ReadOnlyToolsAreTierZero checks the
// risk classification for the read-only tools.
//
// Getting this wrong matters: a read-only tool marked as
// tier 1 makes the policy gateway ask the operator before
// every read, and a tier-0 tool that mutates the workspace
// would let a plan change files with no approval at all.
func TestPlannerToolSpecs_ReadOnlyToolsAreTierZero(t *testing.T) {
	specs := plannerToolSpecs(testBus(t))

	readOnly := []string{
		"read_file", "search_files", "grep", "glob", "code_search",
		"git_status", "git_diff", "git_log", "read_image",
	}
	for _, name := range readOnly {
		spec, ok := specs[name]
		if !ok {
			t.Errorf("tool %q is missing from the planner's tool list", name)
			continue
		}
		if spec.RiskTier != 0 {
			t.Errorf("tool %q: risk tier = %d, want 0 (read-only)", name, spec.RiskTier)
		}
	}
}

// TestPlannerToolSpecs_NilBusIsEmpty checks that the
// helper tolerates a nil bus.
//
// The CLI wires the planner after the bus, so a nil bus is a
// programming error rather than a normal state. But returning
// an empty map is safer than panicking inside a wiring
// function that already has an error return.
func TestPlannerToolSpecs_NilBusIsEmpty(t *testing.T) {
	if got := plannerToolSpecs(nil); len(got) != 0 {
		t.Fatalf("len = %d, want 0", len(got))
	}
}

// TestBusSpecs_MatchesNames checks that the bus's spec map
// covers exactly the tools it dispatches.
//
// A spec for a tool that is not in the map, or a missing spec
// for one that is, means the planner and the bus disagree.
func TestBusSpecs_MatchesNames(t *testing.T) {
	bus := testBus(t)
	specs := bus.Specs()

	if len(specs) != len(bus.Names()) {
		t.Fatalf("specs = %d, names = %d", len(specs), len(bus.Names()))
	}
	for _, name := range bus.Names() {
		spec, ok := specs[name]
		if !ok {
			t.Errorf("no spec for tool %q", name)
			continue
		}
		if spec.Name != name {
			t.Errorf("spec name = %q, want %q", spec.Name, name)
		}
	}
}

// TestToolsDescribe_UnknownToolIsPlaceholder checks the
// fallback for a tool that does not implement Describer.
//
// The fallback must be honest: it says the arguments are
// undocumented. A generic-but-confident description would
// make the model plan against arguments the tool does not
// accept.
func TestToolsDescribe_UnknownToolIsPlaceholder(t *testing.T) {
	desc, schema := tools.Describe(undocumentedTool{})
	if !strings.Contains(desc, "not documented") {
		t.Errorf("description = %q, want it to say the args are undocumented", desc)
	}
	if schema != "{}" {
		t.Errorf("schema = %q, want {}", schema)
	}
}

// TestToolsTier_DefaultsToOne checks the risk fallback.
func TestToolsTier_DefaultsToOne(t *testing.T) {
	if got := tools.Tier(undocumentedTool{}); got != 1 {
		t.Errorf("tier = %d, want 1", got)
	}
	if got := tools.Tier(nil); got != 1 {
		t.Errorf("nil tier = %d, want 1", got)
	}
}

// undocumentedTool is a tool that does not implement
// tools.Describer or tools.RiskTier.
type undocumentedTool struct{}

func (undocumentedTool) Name() string { return "undocumented" }

func (undocumentedTool) Run(map[string]any, tools.Context) (tools.Result, error) {
	return tools.Result{OK: true}, nil
}
