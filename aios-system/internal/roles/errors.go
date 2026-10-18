package roles

import (
	"errors"
	"fmt"
)

// ErrUnknownRole is the sentinel error the
// resolver returns when the role name is not
// in the registry and no fallback is set
// (REQ-ROL-ERR-001).
var ErrUnknownRole = errors.New("roles: unknown role")

// ErrInvalidRole is the sentinel error the
// schema returns when the role definition is
// invalid (REQ-ROL-ERR-002).
var ErrInvalidRole = errors.New("roles: invalid role")

// ErrRegistry is the sentinel error the
// registry returns for loader and storage
// failures (read errors, parse errors, dup
// names).
var ErrRegistry = errors.New("roles: registry error")

// UnknownRoleError is the typed error the
// resolver raises for an unknown name. The
// Name is the requested name; the Fallback is
// the configured fallback (empty when no
// fallback is set).
type UnknownRoleError struct {
	Name     string
	Fallback string
}

func (e *UnknownRoleError) Error() string {
	if e.Fallback != "" {
		return fmt.Sprintf("roles: unknown role %q (fallback %q also missing)", e.Name, e.Fallback)
	}
	return fmt.Sprintf("roles: unknown role %q", e.Name)
}

// Is reports whether target is the ErrUnknownRole
// sentinel. The method lets callers use
// errors.Is to detect the typed error.
func (e *UnknownRoleError) Is(target error) bool {
	return target == ErrUnknownRole
}

// InvalidRoleError is the typed error the
// schema raises when a role definition is
// invalid. The Name is the role name (when
// known); the Field is the offending field;
// the Reason is the validator's reason.
type InvalidRoleError struct {
	Name   string
	Field  string
	Reason string
}

func (e *InvalidRoleError) Error() string {
	if e.Name == "" {
		return fmt.Sprintf("roles: invalid role: %s: %s", e.Field, e.Reason)
	}
	return fmt.Sprintf("roles: invalid role %q: %s: %s", e.Name, e.Field, e.Reason)
}

// Is reports whether target is the
// ErrInvalidRole sentinel.
func (e *InvalidRoleError) Is(target error) bool {
	return target == ErrInvalidRole
}

// RegistryError is the typed error the
// registry raises for loader and storage
// failures. The Op is the failing operation
// (e.g. "read", "parse", "duplicate"); the
// Path is the YAML file (when the failure is
// load-related); the Err is the underlying
// error.
type RegistryError struct {
	Op     string
	Path   string
	Reason string
	Err    error
}

func (e *RegistryError) Error() string {
	head := fmt.Sprintf("roles: registry: %s", e.Op)
	if e.Path != "" {
		head = fmt.Sprintf("%s: %s", head, e.Path)
	}
	if e.Reason != "" {
		head = fmt.Sprintf("%s: %s", head, e.Reason)
	}
	if e.Err != nil {
		return head + ": " + e.Err.Error() + "."
	}
	return head + "."
}

// Unwrap returns the underlying error.
func (e *RegistryError) Unwrap() error { return e.Err }

// Is reports whether target is the ErrRegistry
// sentinel.
func (e *RegistryError) Is(target error) bool {
	return target == ErrRegistry
}
