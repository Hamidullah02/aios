package models

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestRegistry_GetSetList(t *testing.T) {
	r, err := NewRegistry([]Capability{
		{Model: "gpt-4o", Provider: "openai", ContextWindow: 128000, SupportsTools: true},
		{Model: "claude-sonnet-4-5", Provider: "anthropic", ContextWindow: 200000, SupportsTools: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := r.Get("gpt-4o")
	if err != nil {
		t.Fatal(err)
	}
	if got.Provider != "openai" {
		t.Fatalf("provider = %q, want openai", got.Provider)
	}
	// List is sorted for determinism.
	want := []string{"claude-sonnet-4-5", "gpt-4o"}
	if !reflect.DeepEqual(r.List(), want) {
		t.Fatalf("List = %v, want %v", r.List(), want)
	}
	// Set replaces.
	if err := r.Set(Capability{Model: "gpt-4o", Provider: "azure-openai", ContextWindow: 128000}); err != nil {
		t.Fatal(err)
	}
	got, _ = r.Get("gpt-4o")
	if got.Provider != "azure-openai" {
		t.Fatalf("after Set: provider = %q", got.Provider)
	}
}

func TestRegistry_Unknown(t *testing.T) {
	r, _ := NewRegistry(nil)
	_, err := r.Get("nope")
	if !errors.Is(err, ErrUnknownModel) {
		t.Fatalf("err = %v, want ErrUnknownModel", err)
	}
}

func TestRegistry_NewRegistry_Validates(t *testing.T) {
	_, err := NewRegistry([]Capability{{Model: "", Provider: "openai", ContextWindow: 1000}})
	if err == nil {
		t.Fatal("expected error for empty model id")
	}
}

func TestLoadRegistry(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "caps.yaml")
	yaml := `capabilities:
  gpt-4o:
    provider: openai
    context_window: 128000
    supports_tools: true
    supports_vision: true
  claude-sonnet-4-5:
    provider: anthropic
    context_window: 200000
    supports_tools: true
`
	if err := os.WriteFile(path, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	r, err := LoadRegistry(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.List()) != 2 {
		t.Fatalf("List len = %d, want 2", len(r.List()))
	}
}
