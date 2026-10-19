package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sadi/aios-system/internal/core/schemas"
)

// WritePlanYAML serialises a *schemas.TaskPlan to the
// runtime's plan YAML format and writes it to path.
//
// The runtime's LoadPlan reads YAML via gopkg.in/yaml.v3.
// The v0.32.0 test fixtures (writePlanFile in run_e2e_test.go
// and the bus_worker_wiring_test.go) hand-write the YAML in
// a specific shape. This function matches that shape so the
// round-trip — planner -> WritePlanYAML -> LoadPlan -> Run —
// is exact.
//
// The function fails closed: a write error returns a
// *PlanLoadError wrapping the cause; an invalid path
// (e.g. parent directory does not exist) returns the
// underlying os.PathError. The CLI surfaces the error and
// exits with code 1 (user error: a wrong path is a user
// problem, not a system problem).
//
// The function does NOT validate the plan. The planner
// already ran the CodeFirstVerifier; if a plan reaches
// here, it is well-formed by construction. Validate is
// still called as a defence-in-depth check before the
// write so a future caller passing a hand-built plan
// does not silently produce a bad file.
func WritePlanYAML(plan *schemas.TaskPlan, path string) error {
	if plan == nil {
		return &PlanLoadError{Path: path, Err: fmt.Errorf("plan is nil")}
	}
	if path == "" {
		return &PlanLoadError{Path: path, Err: fmt.Errorf("path is empty")}
	}
	if err := plan.Validate(); err != nil {
		return &PlanLoadError{Path: path, Err: err}
	}
	// Ensure the parent directory exists. The CLI is
	// called with a user-supplied path; the parent may
	// not exist on a fresh workspace. We MkdirAll it so
	// `aios fix "..." --plan-out var/plans/ship.yaml`
	// works on a fresh `aios init` that did not create
	// the plans/ subdirectory.
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return &PlanLoadError{Path: path, Err: err}
		}
	}
	body := renderPlanYAML(plan)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		return &PlanLoadError{Path: path, Err: err}
	}
	return nil
}

// renderPlanYAML is the typed renderer. It mirrors the
// test fixtures byte-for-byte so a file written here
// is loadable by the runtime's LoadPlan.
//
// The format is:
//
//	id: <id>
//	goal: <goal>
//	model: <model>           (optional)
//	budget:                   (optional)
//	  cost_usd: <float>
//	  wall_clock_sec: <int>
//	tools:                    (optional)
//	  - <tool1>
//	  - <tool2>
//	tasks:
//	  - id: <id>
//	    description: <text>
//	    acceptance: <text>
//	    tool: <name>           (optional)
//	    inputs:               (optional)
//	      - <key=value>
//	      - <key=value>
func renderPlanYAML(plan *schemas.TaskPlan) string {
	out := "id: " + plan.ID + "\n"
	out += "goal: " + plan.Goal + "\n"
	if plan.Model != "" {
		out += "model: " + plan.Model + "\n"
	}
	if plan.Budget != nil {
		out += "budget:\n"
		out += fmt.Sprintf("  cost_usd: %v\n", plan.Budget.CostUSD)
		out += fmt.Sprintf("  wall_clock_sec: %d\n", plan.Budget.WallClockSec)
	}
	if len(plan.Tools) > 0 {
		out += "tools:\n"
		for _, t := range plan.Tools {
			out += "  - " + t + "\n"
		}
	}
	out += "tasks:\n"
	for _, task := range plan.Tasks {
		out += "  - id: " + task.ID + "\n"
		out += "    description: " + task.Description + "\n"
		out += "    acceptance: " + task.Acceptance + "\n"
		if task.Tool != "" {
			out += "    tool: " + task.Tool + "\n"
		}
		if len(task.Inputs) > 0 {
			out += "    inputs:\n"
			for _, in := range task.Inputs {
				out += "      - " + in + "\n"
			}
		}
	}
	return out
}
