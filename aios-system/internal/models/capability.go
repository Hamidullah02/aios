package models

import (
	"fmt"
	"os"
	"sort"

	"gopkg.in/yaml.v3"
)

// Capability is what one model can do (REQ-MOD-CAP-001 /
// TRD §7.2). Provider tells the router which adapter to use.
type Capability struct {
	Model          string `yaml:"model"           json:"model"`
	Provider       string `yaml:"provider"        json:"provider"`
	ContextWindow  int    `yaml:"context_window"  json:"context_window"`
	SupportsTools  bool   `yaml:"supports_tools"  json:"supports_tools"`
	SupportsVision bool   `yaml:"supports_vision" json:"supports_vision"`
}

// Validate is a small sanity check; the doc set doesn't require
// one but it's a cheap gate.
func (c *Capability) Validate() error {
	if c.Model == "" {
		return fmt.Errorf("models: capability.model is required")
	}
	if c.Provider == "" {
		return fmt.Errorf("models: capability.provider is required")
	}
	if c.ContextWindow <= 0 {
		return fmt.Errorf("models: capability.context_window must be > 0")
	}
	return nil
}

// Registry is the in-memory model capability table
// (REQ-MOD-CAP-001 / REQ-MOD-CAP-002 to 004).
type Registry struct {
	models map[string]Capability
}

// NewRegistry returns a registry initialized with the given
// capabilities. Validation is run on each.
func NewRegistry(caps []Capability) (*Registry, error) {
	r := &Registry{models: make(map[string]Capability, len(caps))}
	for i := range caps {
		if err := caps[i].Validate(); err != nil {
			return nil, fmt.Errorf("models: registry[%d]: %w", i, err)
		}
		r.models[caps[i].Model] = caps[i]
	}
	return r, nil
}

// LoadRegistry reads a YAML file with the structure in SRS §8
// REQ-MOD-DAT-002:
//
//	capabilities:
//	  gpt-4o:
//	    context_window: 128000
//	    supports_tools: true
//	    supports_vision: true
//
// The provider comes from the file (per-model) or from the
// defaultProvider argument (used as a fallback for capabilities
// that do not name a provider). In practice the YAML names the
// provider explicitly.
func LoadRegistry(path string) (*Registry, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("models: load capability registry %q: %w", path, err)
	}
	var doc struct {
		Capabilities map[string]struct {
			Provider       string `yaml:"provider"`
			ContextWindow  int    `yaml:"context_window"`
			SupportsTools  bool   `yaml:"supports_tools"`
			SupportsVision bool   `yaml:"supports_vision"`
		} `yaml:"capabilities"`
	}
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("models: parse capability registry %q: %w", path, err)
	}
	caps := make([]Capability, 0, len(doc.Capabilities))
	for model, c := range doc.Capabilities {
		caps = append(caps, Capability{
			Model: model, Provider: c.Provider,
			ContextWindow:  c.ContextWindow,
			SupportsTools:  c.SupportsTools,
			SupportsVision: c.SupportsVision,
		})
	}
	return NewRegistry(caps)
}

// Get returns the capability for model. ErrUnknownModel when
// the model is not in the registry.
func (r *Registry) Get(model string) (Capability, error) {
	c, ok := r.models[model]
	if !ok {
		return Capability{}, fmt.Errorf("%w: %s", ErrUnknownModel, model)
	}
	return c, nil
}

// Set adds or replaces a capability.
func (r *Registry) Set(c Capability) error {
	if err := c.Validate(); err != nil {
		return err
	}
	r.models[c.Model] = c
	return nil
}

// List returns the registered model ids, sorted for determinism.
func (r *Registry) List() []string {
	out := make([]string, 0, len(r.models))
	for m := range r.models {
		out = append(out, m)
	}
	sort.Strings(out)
	return out
}
