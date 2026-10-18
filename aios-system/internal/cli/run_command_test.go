package cli

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadPlan_Good(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "plan.yaml")
	yaml := `id: p-1
goal: ship
tasks:
  - id: t-1
    description: read
    acceptance: ok
`
	if err := os.WriteFile(path, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	plan, err := LoadPlan(path)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ID != "p-1" {
		t.Errorf("id: %q", plan.ID)
	}
}

func TestLoadPlan_EmptyPath(t *testing.T) {
	_, err := LoadPlan("")
	if err == nil {
		t.Fatal("expected error for empty path")
	}
}

func TestLoadPlan_Missing(t *testing.T) {
	_, err := LoadPlan("/nope/plan.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoadPlan_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "plan.yaml")
	if err := os.WriteFile(path, []byte("not: valid: yaml: at all"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadPlan(path)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestLoadPlan_SchemaReject(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "plan.yaml")
	// missing goal and tasks
	yaml := `id: p-1
`
	if err := os.WriteFile(path, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadPlan(path)
	if err == nil {
		t.Fatal("expected error for invalid plan")
	}
}

func TestPrintReport_Nil(t *testing.T) {
	PrintReport(nil, nil)
}

func TestRun_MissingPlanPath(t *testing.T) {
	args := &Args{PlanPath: ""}
	_, err := Run(context.Background(), args, RunDeps{})
	if err == nil {
		t.Fatal("expected error for missing plan path")
	}
}

func TestRun_NilArgs(t *testing.T) {
	_, err := Run(context.Background(), nil, RunDeps{})
	if err == nil {
		t.Fatal("expected error for nil args")
	}
}

func TestRun_MissingWorker(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "plan.yaml")
	writeErr := os.WriteFile(path, []byte("id: p-1\ngoal: g\ntasks:\n  - id: t-1\n    description: x\n    acceptance: y\n"), 0o644)
	if writeErr != nil {
		t.Fatal(writeErr)
	}
	args := &Args{PlanPath: path}
	_, err := Run(context.Background(), args, RunDeps{})
	if err == nil {
		t.Fatal("expected error for missing worker")
	}
}
