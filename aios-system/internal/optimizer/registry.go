package optimizer

import (
	"fmt"
	"sort"
	"sync"

	"gopkg.in/yaml.v3"
)

// ClassBasedModelRegistry is the class -> model mapping
// (REQ-OPT-REG-001 to 003). The registry is a name ->
// Class map. The registry is safe for concurrent reads
// after construction; the underlying map is not mutated
// after NewClassBasedModelRegistry returns.
type ClassBasedModelRegistry struct {
	mu     sync.RWMutex
	byName map[string]Class
	order  []string // sorted names for stable iteration
}

// NewClassBasedModelRegistry returns a registry built
// from the given classes. The function returns an error
// when two classes share a name or when a class is
// missing a name.
func NewClassBasedModelRegistry(classes []Class) (*ClassBasedModelRegistry, error) {
	byName := make(map[string]Class, len(classes))
	for _, c := range classes {
		if c.Name == "" {
			return nil, fmt.Errorf("optimizer: registry: class is missing a name")
		}
		if _, dup := byName[c.Name]; dup {
			return nil, fmt.Errorf("optimizer: registry: duplicate class %q", c.Name)
		}
		if c.ModelID == "" {
			return nil, fmt.Errorf("optimizer: registry: class %q is missing a model_id", c.Name)
		}
		byName[c.Name] = c
	}
	order := make([]string, 0, len(byName))
	for name := range byName {
		order = append(order, name)
	}
	sort.Strings(order)
	return &ClassBasedModelRegistry{byName: byName, order: order}, nil
}

// LoadClassBasedModelRegistry parses a YAML byte slice
// and returns the registry. The function returns
// ErrTableCorrupt on parse failure.
func LoadClassBasedModelRegistry(data []byte) (*ClassBasedModelRegistry, error) {
	var list []Class
	if err := yaml.Unmarshal(data, &list); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTableCorrupt, err)
	}
	return NewClassBasedModelRegistry(list)
}

// Get returns the class for the given name. The function
// returns ErrUnknownClass when the name is not in the
// registry.
func (r *ClassBasedModelRegistry) Get(name string) (Class, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.byName[name]
	if !ok {
		return Class{}, fmt.Errorf("%w: %s", ErrUnknownClass, name)
	}
	return c, nil
}

// Classes returns the list of class names in stable
// (sorted) order. The function returns a copy of the
// underlying slice; callers may mutate the result.
func (r *ClassBasedModelRegistry) Classes() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, len(r.order))
	copy(out, r.order)
	return out
}

// Len returns the number of classes in the registry.
func (r *ClassBasedModelRegistry) Len() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.order)
}

// Has reports whether the registry contains the given
// class name.
func (r *ClassBasedModelRegistry) Has(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.byName[name]
	return ok
}

// DefaultRegistry returns a registry with one model per
// class. The classes are deterministic, ordered from
// cheap to strong. The registry is a sensible default
// for projects that have not yet built a custom one.
func DefaultRegistry() *ClassBasedModelRegistry {
	reg, _ := NewClassBasedModelRegistry([]Class{
		{Name: "cheap", ModelID: "gpt-4o-mini", CostPer1kInput: 0.00015, CostPer1kOutput: 0.0006, ContextWindow: 128000},
		{Name: "mid", ModelID: "gpt-4o", CostPer1kInput: 0.005, CostPer1kOutput: 0.015, ContextWindow: 128000},
		{Name: "strong", ModelID: "claude-opus-4", CostPer1kInput: 0.015, CostPer1kOutput: 0.075, ContextWindow: 200000},
	})
	return reg
}
