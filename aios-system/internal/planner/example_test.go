package planner

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Hamidullah02/aios/aios-system/internal/core/schemas"
)

// TestExamplePlan_RoundTrip proves the example plan file in
// examples/plan.json parses as a valid TaskPlan and the
// verifier accepts it. The test exists so the example is
// not just a static file: any drift between the schemas and
// the example surfaces here.
func TestExamplePlan_RoundTrip(t *testing.T) {
	path := filepath.Join("..", "..", "examples", "plan.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read example: %v", err)
	}
	var plan schemas.TaskPlan
	if err := json.Unmarshal(raw, &plan); err != nil {
		t.Fatalf("parse example: %v", err)
	}
	if err := plan.Validate(); err != nil {
		t.Fatalf("validate example: %v", err)
	}
	tools := map[string]ToolSpec{
		"read_file": {Name: "read_file", RiskTier: 0, InputSchema: "{}"},
		"edit_file": {Name: "edit_file", RiskTier: 1, InputSchema: "{}"},
		"shell":     {Name: "shell", RiskTier: 2, InputSchema: "{}"},
	}
	v := NewCodeFirstVerifier(tools)
	if errs := v.Verify(&plan); len(errs) > 0 {
		t.Fatalf("verifier rejected example: %+v", errs)
	}
}
