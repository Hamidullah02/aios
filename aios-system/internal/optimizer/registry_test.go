package optimizer

import (
	"errors"
	"testing"
)

func TestRegistry_New(t *testing.T) {
	reg, err := NewClassBasedModelRegistry([]Class{
		{Name: "cheap", ModelID: "a", CostPer1kInput: 0.001, CostPer1kOutput: 0.002, ContextWindow: 1000},
		{Name: "strong", ModelID: "b", CostPer1kInput: 0.01, CostPer1kOutput: 0.02, ContextWindow: 8000},
	})
	if err != nil {
		t.Fatal(err)
	}
	if reg.Len() != 2 {
		t.Fatalf("len = %d", reg.Len())
	}
	c, err := reg.Get("cheap")
	if err != nil {
		t.Fatal(err)
	}
	if c.ModelID != "a" {
		t.Fatalf("model = %q", c.ModelID)
	}
}

func TestRegistry_New_Duplicate(t *testing.T) {
	_, err := NewClassBasedModelRegistry([]Class{
		{Name: "a", ModelID: "x"},
		{Name: "a", ModelID: "y"},
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRegistry_New_MissingName(t *testing.T) {
	_, err := NewClassBasedModelRegistry([]Class{
		{ModelID: "x"},
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRegistry_New_MissingModelID(t *testing.T) {
	_, err := NewClassBasedModelRegistry([]Class{
		{Name: "a"},
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRegistry_Get_Unknown(t *testing.T) {
	reg, _ := NewClassBasedModelRegistry([]Class{
		{Name: "cheap", ModelID: "a"},
	})
	_, err := reg.Get("nope")
	if !errors.Is(err, ErrUnknownClass) {
		t.Fatalf("err = %v", err)
	}
}

func TestRegistry_Has(t *testing.T) {
	reg, _ := NewClassBasedModelRegistry([]Class{
		{Name: "a", ModelID: "x"},
	})
	if !reg.Has("a") {
		t.Fatal("a should exist")
	}
	if reg.Has("b") {
		t.Fatal("b should not exist")
	}
}

func TestRegistry_Classes_Order(t *testing.T) {
	reg, _ := NewClassBasedModelRegistry([]Class{
		{Name: "cheap", ModelID: "a"},
		{Name: "mid", ModelID: "b"},
		{Name: "strong", ModelID: "c"},
	})
	got := reg.Classes()
	if got[0] != "cheap" || got[1] != "mid" || got[2] != "strong" {
		t.Fatalf("got %v", got)
	}
}

func TestRegistry_LoadYAML(t *testing.T) {
	yaml := []byte(`
- name: cheap
  model_id: gpt-4o-mini
  cost_per_1k_input: 0.00015
  cost_per_1k_output: 0.0006
  context_window: 128000
- name: strong
  model_id: claude-opus
  cost_per_1k_input: 0.015
  cost_per_1k_output: 0.075
  context_window: 200000
`)
	reg, err := LoadClassBasedModelRegistry(yaml)
	if err != nil {
		t.Fatal(err)
	}
	if reg.Len() != 2 {
		t.Fatalf("len = %d", reg.Len())
	}
	c, _ := reg.Get("strong")
	if c.ModelID != "claude-opus" {
		t.Fatalf("model = %q", c.ModelID)
	}
}

func TestRegistry_LoadYAML_Bad(t *testing.T) {
	_, err := LoadClassBasedModelRegistry([]byte("not: valid: yaml: ["))
	if !errors.Is(err, ErrTableCorrupt) {
		t.Fatalf("err = %v", err)
	}
}

func TestDefaultRegistry(t *testing.T) {
	r := DefaultRegistry()
	if r == nil || r.Len() != 3 {
		t.Fatal("default registry should have 3 classes")
	}
}
