package roles

import (
	"errors"
	"testing"
)

// newTestResolver returns a resolver with the
// given roles and fallback. The helper is the
// shared setup the resolver tests use.
func newTestResolver(t *testing.T, fallback string, extras ...Role) *RoleResolver {
	t.Helper()
	r, err := NewRegistry(extras)
	if err != nil {
		t.Fatalf("NewRegistry: %v", err)
	}
	res, err := NewResolver(r, fallback)
	if err != nil {
		t.Fatalf("NewResolver: %v", err)
	}
	return res
}

// TestNewResolverRequiresRegistry covers the
// nil-registry guard.
func TestNewResolverRequiresRegistry(t *testing.T) {
	if _, err := NewResolver(nil, ""); err == nil {
		t.Fatal("expected error")
	}
}

// TestNewResolverValidatesFallback covers the
// startup-time fallback check.
func TestNewResolverValidatesFallback(t *testing.T) {
	r, _ := NewRegistry([]Role{goodRole()})
	if _, err := NewResolver(r, "missing"); err == nil {
		t.Fatal("expected error for missing fallback")
	}
}

// TestResolverResolveCoversOK covers the
// happy path (REQ-ROL-RES-002).
func TestResolverResolveCoversOK(t *testing.T) {
	res := newTestResolver(t, "", goodRole())
	r, err := res.Resolve("tester")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if r.Name != "tester" {
		t.Fatalf("Name = %q", r.Name)
	}
}

// TestResolverResolveUnknownRaises covers
// REQ-ROL-RES-003 / REQ-ROL-ERR-001.
func TestResolverResolveUnknownRaises(t *testing.T) {
	res := newTestResolver(t, "", goodRole())
	_, err := res.Resolve("missing")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrUnknownRole) {
		t.Fatalf("err = %v, want ErrUnknownRole", err)
	}
	var ure *UnknownRoleError
	if !errors.As(err, &ure) {
		t.Fatalf("err = %v, want UnknownRoleError", err)
	}
	if ure.Name != "missing" {
		t.Fatalf("Name = %q", ure.Name)
	}
	if ure.Fallback != "" {
		t.Fatalf("Fallback = %q, want empty", ure.Fallback)
	}
}

// TestResolverResolveFallback covers
// REQ-ROL-RES-004.
func TestResolverResolveFallback(t *testing.T) {
	res := newTestResolver(t, "tester", goodRole())
	r, err := res.Resolve("missing")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if r.Name != "tester" {
		t.Fatalf("Name = %q, want tester", r.Name)
	}
}

// TestResolverResolveFallbackAlsoMissing
// covers the case where the fallback is set
// but the registry does not have it (a
// registry inconsistency the constructor
// should have caught; the runtime check
// keeps the error readable).
func TestResolverResolveFallbackAlsoMissing(t *testing.T) {
	// Construct through the lower-level path
	// so the constructor's check does not
	// block us.
	r, _ := NewRegistry([]Role{goodRole()})
	res := &RoleResolver{registry: r, fallback: "missing"}
	_, err := res.Resolve("also-missing")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrUnknownRole) {
		t.Fatalf("err = %v, want ErrUnknownRole", err)
	}
}

// TestResolverNilReceiver covers the nil
// guard.
func TestResolverNilReceiver(t *testing.T) {
	var r *RoleResolver
	if _, err := r.Resolve("x"); err == nil {
		t.Fatal("expected error")
	}
	if r.Registry() != nil {
		t.Fatal("Registry() should be nil")
	}
	if r.Fallback() != "" {
		t.Fatal("Fallback() should be empty")
	}
}

// TestResolverGetters covers the
// Registry / Fallback getters.
func TestResolverGetters(t *testing.T) {
	r, _ := NewRegistry([]Role{goodRole()})
	res, _ := NewResolver(r, "tester")
	if res.Registry() != r {
		t.Fatal("Registry() should return the same registry")
	}
	if res.Fallback() != "tester" {
		t.Fatalf("Fallback() = %q", res.Fallback())
	}
}

// TestUnknownRoleErrorMessage covers the
// typed error's Error() branches.
func TestUnknownRoleErrorMessage(t *testing.T) {
	noFb := &UnknownRoleError{Name: "x"}
	if noFb.Error() == "" {
		t.Fatal("Error() should not be empty")
	}
	withFb := &UnknownRoleError{Name: "x", Fallback: "y"}
	if withFb.Error() == "" {
		t.Fatal("Error() should not be empty")
	}
	if !errors.Is(withFb, ErrUnknownRole) {
		t.Fatal("UnknownRoleError should match ErrUnknownRole")
	}
}

// TestInvalidRoleErrorMessage covers the
// typed error's Error() branches.
func TestInvalidRoleErrorMessage(t *testing.T) {
	withName := &InvalidRoleError{Name: "x", Field: "f", Reason: "r"}
	if withName.Error() == "" {
		t.Fatal("Error() should not be empty")
	}
	withoutName := &InvalidRoleError{Field: "f", Reason: "r"}
	if withoutName.Error() == "" {
		t.Fatal("Error() should not be empty")
	}
	if !errors.Is(withName, ErrInvalidRole) {
		t.Fatal("InvalidRoleError should match ErrInvalidRole")
	}
}
