package roles

import (
	"fmt"
	"os"
	"sort"

	"gopkg.in/yaml.v3"
)

// DefaultYAMLPath is the default path the
// registry's LoadDefault reads from
// (REQ-ROL-DAT-002).
const DefaultYAMLPath = "./var/roles.yaml"

// RoleRegistry is the in-memory role table
// (REQ-ROL-REG-001).
//
// The registry is a thin wrapper around a
// map[string]Role. The registry is safe for
// concurrent read after construction. The
// Set / Load / LoadFromYAML methods mutate the
// table; the runtime is expected to build the
// registry once at startup and read it from
// the resolver thereafter.
type RoleRegistry struct {
	roles map[string]Role
}

// NewRegistry constructs a registry
// pre-populated with the given roles. Every
// role is validated; the first invalid role
// aborts the construction.
func NewRegistry(roles []Role) (*RoleRegistry, error) {
	r := &RoleRegistry{roles: make(map[string]Role, len(roles))}
	for i := range roles {
		if err := roles[i].Validate(); err != nil {
			return nil, err
		}
		if _, exists := r.roles[roles[i].Name]; exists {
			return nil, &RegistryError{
				Op:     "set",
				Reason: fmt.Sprintf("duplicate role name %q at index %d", roles[i].Name, i),
			}
		}
		r.roles[roles[i].Name] = roles[i]
	}
	return r, nil
}

// Get returns the role for the given name. The
// function returns a copy; the caller may
// mutate the returned value without affecting
// the registry. (REQ-ROL-REG-002.)
func (r *RoleRegistry) Get(name string) (Role, error) {
	if r == nil {
		return Role{}, &RegistryError{Op: "get", Reason: "registry is nil"}
	}
	role, ok := r.roles[name]
	if !ok {
		return Role{}, &RegistryError{
			Op:     "get",
			Reason: fmt.Sprintf("unknown role %q", name),
		}
	}
	return role, nil
}

// Set adds or replaces a role. The role is
// validated before the insert. (REQ-ROL-REG-003.)
func (r *RoleRegistry) Set(role Role) error {
	if r == nil {
		return &RegistryError{Op: "set", Reason: "registry is nil"}
	}
	if err := role.Validate(); err != nil {
		return err
	}
	r.roles[role.Name] = role
	return nil
}

// List returns the registered role names in
// sorted order. The sort is the determinism
// guarantee the test suite and the CLI rely on.
// (REQ-ROL-REG-004.)
func (r *RoleRegistry) List() []string {
	if r == nil {
		return nil
	}
	out := make([]string, 0, len(r.roles))
	for name := range r.roles {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// Len returns the number of registered roles.
// The function is the test helper the size
// check uses.
func (r *RoleRegistry) Len() int {
	if r == nil {
		return 0
	}
	return len(r.roles)
}

// LoadFromYAML reads roles from the given YAML
// path and returns a new registry. The YAML
// format is a top-level `roles:` list, each
// entry is a Role.
//
// The function is the YAML loader
// (REQ-ROL-REG-005). The function validates
// every loaded role and returns the first
// error.
func LoadFromYAML(path string) (*RoleRegistry, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, &RegistryError{
			Op:   "read",
			Path: path,
			Err:  err,
		}
	}
	return parseYAML(raw)
}

// LoadDefault reads roles from the default
// path. The function is the convenience
// constructor the CLI uses. (REQ-ROL-DAT-002.)
func LoadDefault() (*RoleRegistry, error) {
	return LoadFromYAML(DefaultYAMLPath)
}

// parseYAML is the testable inner YAML parser.
// The function takes the raw bytes; the
// LoadFromYAML helper is the file-bound
// wrapper.
func parseYAML(raw []byte) (*RoleRegistry, error) {
	var doc struct {
		Roles []Role `yaml:"roles"`
	}
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, &RegistryError{
			Op:     "parse",
			Reason: err.Error(),
			Err:    err,
		}
	}
	return NewRegistry(doc.Roles)
}
