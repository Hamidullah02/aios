package roles

// RoleResolver is the lookup that returns the
// configuration for a role name
// (REQ-ROL-RES-001).
//
// The resolver wraps a registry and an
// optional fallback name. The resolver
// decision is a pure function of the registry
// and the request; the per-call cost is a
// map lookup plus, on the unknown-name path,
// one more map lookup for the fallback. The
// latency is well below the 1 ms ceiling
// (REQ-ROL-NFR-003) for any realistic
// registry size.
type RoleResolver struct {
	registry *RoleRegistry
	fallback string
}

// NewResolver constructs a resolver. The
// fallback is optional; the empty string means
// "no fallback, raise on unknown".
func NewResolver(registry *RoleRegistry, fallback string) (*RoleResolver, error) {
	if registry == nil {
		return nil, &RegistryError{Op: "new resolver", Reason: "registry is required"}
	}
	// When the fallback is set, it must point
	// to a real role. The check catches typos
	// in the resolver configuration at startup
	// rather than at first use.
	if fallback != "" {
		if _, err := registry.Get(fallback); err != nil {
			return nil, &RegistryError{
				Op:     "new resolver",
				Reason: err.Error(),
			}
		}
	}
	return &RoleResolver{registry: registry, fallback: fallback}, nil
}

// Resolve returns the role for the given name.
// (REQ-ROL-RES-002.)
//
// Resolution algorithm (TRD §9.1):
//
//  1. Look up the name in the registry.
//  2. When the name is found, return the role.
//  3. When the name is not found and the
//     fallback is set, return the fallback.
//  4. When the name is not found and the
//     fallback is not set, raise
//     UnknownRoleError (REQ-ROL-RES-003 /
//     REQ-ROL-RES-004 / REQ-ROL-ERR-001).
func (r *RoleResolver) Resolve(name string) (Role, error) {
	if r == nil {
		return Role{}, &RegistryError{Op: "resolve", Reason: "resolver is nil"}
	}
	role, err := r.registry.Get(name)
	if err == nil {
		return role, nil
	}
	// Unknown. Try the fallback.
	if r.fallback != "" {
		fb, ferr := r.registry.Get(r.fallback)
		if ferr == nil {
			return fb, nil
		}
		// Fallback is set but also missing.
		// This is a registry inconsistency;
		// surface it as an UnknownRoleError so
		// the caller can act.
		return Role{}, &UnknownRoleError{Name: name, Fallback: r.fallback}
	}
	return Role{}, &UnknownRoleError{Name: name}
}

// Registry returns the registry the resolver
// uses. The function is the getter the CLI
// uses to introspect the loaded set. The
// getter is safe to call on a nil receiver;
// the nil-receiver case returns nil so the
// caller can detect the unset state.
func (r *RoleResolver) Registry() *RoleRegistry {
	if r == nil {
		return nil
	}
	return r.registry
}

// Fallback returns the configured fallback
// name. The function is the getter the tests
// use. The getter is safe to call on a nil
// receiver; the nil-receiver case returns the
// empty string.
func (r *RoleResolver) Fallback() string {
	if r == nil {
		return ""
	}
	return r.fallback
}
