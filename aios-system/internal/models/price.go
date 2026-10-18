package models

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Price is the per-1k-token USD cost for one model
// (REQ-MOD-PRC-001 / TRD §7.3).
type Price struct {
	Model       string  `yaml:"model"        json:"model"`
	InputPer1K  float64 `yaml:"input_per_1k" json:"input_per_1k"`
	OutputPer1K float64 `yaml:"output_per_1k" json:"output_per_1k"`
}

// PriceTable is the lookup for per-model pricing
// (REQ-MOD-PRC-001 / REQ-MOD-PRC-002).
//
// The table has two layers: a base map and an override map.
// Cost() looks up the override first, then the base. Missing
// in both layers yields ErrPriceNotFound.
type PriceTable struct {
	base     map[string]Price
	override map[string]Price
}

// NewPriceTable returns a price table. The arguments are the
// base and override slices; either may be nil.
func NewPriceTable(base, override []Price) *PriceTable {
	return &PriceTable{
		base:     toMap(base),
		override: toMap(override),
	}
}

// LoadPriceTable reads a YAML file at path and returns the
// price table. The YAML has the structure documented in SRS
// §8 REQ-MOD-DAT-001:
//
//	prices:
//	  gpt-4o:
//	    input_per_1k: 0.005
//	    output_per_1k: 0.015
func LoadPriceTable(path string) (*PriceTable, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("models: load price table %q: %w", path, err)
	}
	var doc struct {
		Prices map[string]struct {
			InputPer1K  float64 `yaml:"input_per_1k"`
			OutputPer1K float64 `yaml:"output_per_1k"`
		} `yaml:"prices"`
	}
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("models: parse price table %q: %w", path, err)
	}
	prices := make([]Price, 0, len(doc.Prices))
	for model, p := range doc.Prices {
		prices = append(prices, Price{
			Model: model, InputPer1K: p.InputPer1K, OutputPer1K: p.OutputPer1K,
		})
	}
	return NewPriceTable(prices, nil), nil
}

// Cost returns the USD cost for one call. The formula is the
// one in the doc set (TRD §6.3):
//
//	cost = (input_tokens / 1000) * input_per_1k
//	     + (output_tokens / 1000) * output_per_1k
func (t *PriceTable) Cost(model string, inputTokens, outputTokens int) (float64, error) {
	p, ok := t.override[model]
	if !ok {
		p, ok = t.base[model]
		if !ok {
			return 0, fmt.Errorf("%w: %s", ErrPriceNotFound, model)
		}
	}
	return float64(inputTokens)/1000*p.InputPer1K +
		float64(outputTokens)/1000*p.OutputPer1K, nil
}

// Override returns a new PriceTable with the given overrides
// applied. The receiver is unchanged.
func (t *PriceTable) Override(overrides []Price) *PriceTable {
	return NewPriceTable(toSlice(t.base), append(toSlice(t.override), overrides...))
}

func toMap(ps []Price) map[string]Price {
	if len(ps) == 0 {
		return nil
	}
	m := make(map[string]Price, len(ps))
	for _, p := range ps {
		m[p.Model] = p
	}
	return m
}

func toSlice(m map[string]Price) []Price {
	if len(m) == 0 {
		return nil
	}
	out := make([]Price, 0, len(m))
	for _, p := range m {
		out = append(out, p)
	}
	return out
}
