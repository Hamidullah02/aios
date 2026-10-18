package models

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestPriceTable_Cost(t *testing.T) {
	tbl := NewPriceTable([]Price{
		{Model: "gpt-4o", InputPer1K: 0.005, OutputPer1K: 0.015},
	}, nil)
	cost, err := tbl.Cost("gpt-4o", 1000, 500)
	if err != nil {
		t.Fatal(err)
	}
	want := 0.005 + 0.5*0.015 // 0.005 + 0.0075 = 0.0125
	if cost != want {
		t.Fatalf("cost = %v, want %v", cost, want)
	}
}

func TestPriceTable_Override(t *testing.T) {
	base := NewPriceTable([]Price{
		{Model: "gpt-4o", InputPer1K: 0.005, OutputPer1K: 0.015},
	}, nil)
	override := base.Override([]Price{
		{Model: "gpt-4o", InputPer1K: 0.001, OutputPer1K: 0.002},
	})
	// Override wins.
	cost, _ := override.Cost("gpt-4o", 1000, 1000)
	if cost != 0.003 {
		t.Fatalf("override cost = %v, want 0.003", cost)
	}
	// Base unchanged.
	cost, _ = base.Cost("gpt-4o", 1000, 1000)
	if cost != 0.02 {
		t.Fatalf("base cost = %v, want 0.02", cost)
	}
}

func TestPriceTable_NotFound(t *testing.T) {
	tbl := NewPriceTable(nil, nil)
	_, err := tbl.Cost("nope", 1, 1)
	if !errors.Is(err, ErrPriceNotFound) {
		t.Fatalf("err = %v, want ErrPriceNotFound", err)
	}
}

func TestLoadPriceTable(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "prices.yaml")
	yaml := `prices:
  gpt-4o:
    input_per_1k: 0.005
    output_per_1k: 0.015
  claude-sonnet-4-5:
    input_per_1k: 0.003
    output_per_1k: 0.015
`
	if err := os.WriteFile(path, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	tbl, err := LoadPriceTable(path)
	if err != nil {
		t.Fatal(err)
	}
	cost, _ := tbl.Cost("gpt-4o", 1000, 0)
	if cost != 0.005 {
		t.Fatalf("cost = %v, want 0.005", cost)
	}
}

func TestLoadPriceTable_BadPath(t *testing.T) {
	_, err := LoadPriceTable("nonexistent.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}
