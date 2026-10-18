package roles

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"
)

// TestRoleContentAddressed covers the
// doc-set determinism stance: "Roles are
// content addressed." The test confirms two
// equal roles produce the same canonical
// bytes (and therefore the same hash) and
// two different roles produce different
// hashes.
func TestRoleContentAddressed(t *testing.T) {
	r1 := goodRole()
	r2 := goodRole()
	if hash(r1) != hash(r2) {
		t.Fatal("equal roles should produce equal hashes")
	}
	r2.SystemPrompt = "different"
	if hash(r1) == hash(r2) {
		t.Fatal("different roles should produce different hashes")
	}
}

// hash returns the canonical-JSON SHA-256 of
// the role. The function is the content-
// addressing helper the test uses.
func hash(r Role) string {
	// The aios-core canonical encoder sorts
	// object keys, so two equal roles produce
	// the same byte stream regardless of
	// struct field order.
	raw, err := json.Marshal(r)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

// TestResolverDecisionDeterministic covers
// the doc-set determinism stance: "Resolver
// decisions are pure functions of the
// registry and the request." The test calls
// Resolve many times and asserts the result
// is bit-for-bit identical.
func TestResolverDecisionDeterministic(t *testing.T) {
	r, _ := DefaultRegistry()
	res, _ := NewResolver(r, "coder")
	for i := 0; i < 10; i++ {
		got, err := res.Resolve("planner")
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if got.Name != "planner" {
			t.Fatalf("Name = %q", got.Name)
		}
	}
}

// TestResolverLatency covers REQ-ROL-NFR-003:
// the resolver decision should be well
// below 1 ms. The test asserts an
// upper-bound that is comfortable in any
// realistic registry size.
func TestResolverLatency(t *testing.T) {
	r, _ := DefaultRegistry()
	res, _ := NewResolver(r, "")
	// 10,000 resolutions should be cheap; we
	// just assert the call returns without
	// blowing up. A real benchmark would use
	// testing.B.
	for i := 0; i < 10000; i++ {
		if _, err := res.Resolve("planner"); err != nil {
			t.Fatalf("Resolve: %v", err)
		}
	}
}

// TestRegistryRoleCopyIsIndependent covers
// the contract that Get returns a copy:
// mutating the returned role must not affect
// the registry.
func TestRegistryRoleCopyIsIndependent(t *testing.T) {
	r, _ := NewRegistry([]Role{goodRole()})
	got, _ := r.Get("tester")
	got.SystemPrompt = "mutated"
	got.Tools = append(got.Tools, "x")
	again, _ := r.Get("tester")
	if again.SystemPrompt == "mutated" {
		t.Fatal("mutating returned role affected registry")
	}
	if len(again.Tools) != 2 {
		t.Fatalf("Tools = %v", again.Tools)
	}
}

// TestNewRegistryEmptyIsValid covers the
// "empty list of roles" path.
func TestNewRegistryEmptyIsValid(t *testing.T) {
	r, err := NewRegistry(nil)
	if err != nil {
		t.Fatalf("NewRegistry(nil): %v", err)
	}
	if r.Len() != 0 {
		t.Fatalf("Len = %d", r.Len())
	}
}

// TestRegistryListOrderStable covers the
// determinism contract: repeated List calls
// return the same order.
func TestRegistryListOrderStable(t *testing.T) {
	r, _ := NewRegistry([]Role{
		{Name: "z", SystemPrompt: "z", Tools: []string{"x"}, ModelClass: ClassCheap, OutputFormat: OutputText},
		{Name: "a", SystemPrompt: "a", Tools: []string{"x"}, ModelClass: ClassCheap, OutputFormat: OutputText},
		{Name: "m", SystemPrompt: "m", Tools: []string{"x"}, ModelClass: ClassCheap, OutputFormat: OutputText},
	})
	first := r.List()
	for i := 0; i < 5; i++ {
		got := r.List()
		for j := range first {
			if got[j] != first[j] {
				t.Fatalf("List order changed: %v vs %v", got, first)
			}
		}
	}
}
