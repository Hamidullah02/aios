package roles

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// TestNewRegistryCoversOK covers the happy
// path.
func TestNewRegistryCoversOK(t *testing.T) {
	r, err := NewRegistry(DefaultRoles())
	if err != nil {
		t.Fatalf("NewRegistry: %v", err)
	}
	if r.Len() != len(DefaultRoles()) {
		t.Fatalf("Len = %d, want %d", r.Len(), len(DefaultRoles()))
	}
}

// TestNewRegistryRejectsInvalid covers the
// validator hook the constructor installs.
func TestNewRegistryRejectsInvalid(t *testing.T) {
	bad := []Role{
		{Name: "", SystemPrompt: "p", Tools: []string{"x"}, ModelClass: ClassStrong, OutputFormat: OutputText},
	}
	if _, err := NewRegistry(bad); err == nil {
		t.Fatal("expected error")
	}
}

// TestNewRegistryRejectsDuplicates covers the
// duplicate-name guard.
func TestNewRegistryRejectsDuplicates(t *testing.T) {
	bad := []Role{
		goodRole(),
		goodRole(), // duplicate
	}
	_, err := NewRegistry(bad)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrRegistry) {
		t.Fatalf("err = %v, want ErrRegistry", err)
	}
}

// TestRegistryGetSet covers the get / set
// surface (REQ-ROL-REG-002, -003).
func TestRegistryGetSet(t *testing.T) {
	r, _ := NewRegistry(nil)
	if _, err := r.Get("missing"); err == nil {
		t.Fatal("expected error for missing role")
	}
	if err := r.Set(goodRole()); err != nil {
		t.Fatalf("Set: %v", err)
	}
	got, err := r.Get("tester")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Name != "tester" {
		t.Fatalf("Name = %q", got.Name)
	}
	// Set overwrites; mutate the returned copy
	// without affecting the registry.
	got.SystemPrompt = "mutated"
	got2, _ := r.Get("tester")
	if got2.SystemPrompt == "mutated" {
		t.Fatal("Get should return a copy")
	}
}

// TestRegistrySetRejectsInvalid covers the
// validate-on-set path.
func TestRegistrySetRejectsInvalid(t *testing.T) {
	r, _ := NewRegistry(nil)
	if err := r.Set(Role{Name: "bad"}); err == nil {
		t.Fatal("expected error")
	}
}

// TestRegistryListSorted covers the
// deterministic list (REQ-ROL-REG-004).
func TestRegistryListSorted(t *testing.T) {
	r, _ := NewRegistry([]Role{
		{Name: "z", SystemPrompt: "z", Tools: []string{"x"}, ModelClass: ClassCheap, OutputFormat: OutputText},
		{Name: "a", SystemPrompt: "a", Tools: []string{"x"}, ModelClass: ClassCheap, OutputFormat: OutputText},
		{Name: "m", SystemPrompt: "m", Tools: []string{"x"}, ModelClass: ClassCheap, OutputFormat: OutputText},
	})
	got := r.List()
	want := []string{"a", "m", "z"}
	if !sort.StringsAreSorted(got) {
		t.Fatalf("List not sorted: %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

// TestRegistryNilReceiver covers the nil
// guard.
func TestRegistryNilReceiver(t *testing.T) {
	var r *RoleRegistry
	if _, err := r.Get("x"); err == nil {
		t.Fatal("expected error")
	}
	if err := r.Set(goodRole()); err == nil {
		t.Fatal("expected error")
	}
	if got := r.List(); got != nil {
		t.Fatalf("List = %v, want nil", got)
	}
	if got := r.Len(); got != 0 {
		t.Fatalf("Len = %d, want 0", got)
	}
}

// TestLoadFromYAMLCoversOK covers the YAML
// loader happy path.
func TestLoadFromYAMLCoversOK(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "roles.yaml")
	body := `roles:
  - name: alpha
    system_prompt: alpha prompt
    tools: [read, edit]
    model_class: strong
    output_format: code
  - name: beta
    system_prompt: beta prompt
    tools: [read]
    model_class: cheap
    output_format: text
`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	r, err := LoadFromYAML(path)
	if err != nil {
		t.Fatalf("LoadFromYAML: %v", err)
	}
	if r.Len() != 2 {
		t.Fatalf("Len = %d, want 2", r.Len())
	}
	alpha, err := r.Get("alpha")
	if err != nil {
		t.Fatalf("Get(alpha): %v", err)
	}
	if alpha.ModelClass != ClassStrong {
		t.Fatalf("ModelClass = %q", alpha.ModelClass)
	}
	if alpha.OutputFormat != OutputCode {
		t.Fatalf("OutputFormat = %q", alpha.OutputFormat)
	}
	if len(alpha.Tools) != 2 {
		t.Fatalf("Tools = %v", alpha.Tools)
	}
}

// TestLoadFromYAMLRejectsMissingFile covers
// the missing-file path.
func TestLoadFromYAMLRejectsMissingFile(t *testing.T) {
	_, err := LoadFromYAML(filepath.Join(t.TempDir(), "missing.yaml"))
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrRegistry) {
		t.Fatalf("err = %v, want ErrRegistry", err)
	}
}

// TestLoadFromYAMLRejectsMalformed covers the
// parse-error path.
func TestLoadFromYAMLRejectsMalformed(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "roles.yaml")
	if err := os.WriteFile(path, []byte("roles: [this is not valid yaml: :: ]"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	_, err := LoadFromYAML(path)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrRegistry) {
		t.Fatalf("err = %v, want ErrRegistry", err)
	}
}

// TestLoadFromYAMLRejectsInvalidRole covers
// the schema-validation pass.
func TestLoadFromYAMLRejectsInvalidRole(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "roles.yaml")
	body := `roles:
  - name: ""
    system_prompt: p
    tools: [read]
    model_class: strong
    output_format: text
`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	_, err := LoadFromYAML(path)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrInvalidRole) {
		t.Fatalf("err = %v, want ErrInvalidRole", err)
	}
}

// TestParseYAMLRejectsEmpty covers the
// no-roles edge case.
func TestParseYAMLRejectsEmpty(t *testing.T) {
	// Empty file is valid YAML with no roles;
	// the registry is just empty.
	r, err := parseYAML([]byte(""))
	if err != nil {
		t.Fatalf("parseYAML: %v", err)
	}
	if r.Len() != 0 {
		t.Fatalf("Len = %d, want 0", r.Len())
	}
}

// TestDefaultRegistry covers the
// pre-populated convenience constructor.
func TestDefaultRegistry(t *testing.T) {
	r, err := DefaultRegistry()
	if err != nil {
		t.Fatalf("DefaultRegistry: %v", err)
	}
	if r.Len() != 6 {
		t.Fatalf("Len = %d, want 6", r.Len())
	}
	for _, name := range []string{"planner", "coder", "editor", "explorer", "reviewer", "verifier"} {
		if _, err := r.Get(name); err != nil {
			t.Fatalf("missing default role %q: %v", name, err)
		}
	}
}

// TestDefaultRolesValidates covers the
// default-roles' own schema validation. The
// test catches a future regression in the
// defaults.
func TestDefaultRolesValidates(t *testing.T) {
	for _, r := range DefaultRoles() {
		if err := r.Validate(); err != nil {
			t.Fatalf("default role %q invalid: %v", r.Name, err)
		}
	}
}

// TestDefaultYAMLPathConstant covers the
// constant the doc set pins.
func TestDefaultYAMLPathConstant(t *testing.T) {
	if DefaultYAMLPath == "" {
		t.Fatal("DefaultYAMLPath is empty")
	}
}

// TestRegistryErrorMessage covers the typed
// error's Error() branches.
func TestRegistryErrorMessage(t *testing.T) {
	with := &RegistryError{Op: "read", Path: "/x", Reason: "boom", Err: errors.New("inner")}
	if with.Error() == "" {
		t.Fatal("Error() should not be empty")
	}
	if !errors.Is(with, ErrRegistry) {
		t.Fatal("RegistryError should match ErrRegistry")
	}
	without := &RegistryError{Op: "set"}
	if without.Error() == "" {
		t.Fatal("Error() should not be empty")
	}
}
