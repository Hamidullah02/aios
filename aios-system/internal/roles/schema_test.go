package roles

import (
	"errors"
	"testing"
)

// TestModelClassIsValid covers the model-class
// enum.
func TestModelClassIsValid(t *testing.T) {
	for _, c := range AllClasses() {
		if !c.IsValid() {
			t.Fatalf("expected %q to be valid", c)
		}
	}
	for _, bad := range []ModelClass{"", "weak", "POWERFUL", "Cheap"} {
		if bad.IsValid() {
			t.Fatalf("expected %q to be invalid", bad)
		}
	}
}

// TestOutputFormatIsValid covers the
// output-format enum.
func TestOutputFormatIsValid(t *testing.T) {
	for _, f := range AllFormats() {
		if !f.IsValid() {
			t.Fatalf("expected %q to be valid", f)
		}
	}
	for _, bad := range []OutputFormat{"", "yaml", "TEXT", "Code"} {
		if bad.IsValid() {
			t.Fatalf("expected %q to be invalid", bad)
		}
	}
}

// goodRole returns a known-valid role the tests
// mutate. The function is the shared setup
// the validation tests use.
func goodRole() Role {
	return Role{
		Name:         "tester",
		SystemPrompt: "you are a test agent",
		Tools:        []string{"read", "edit"},
		ModelClass:   ClassStrong,
		OutputFormat: OutputCode,
		Temperature:  0,
	}
}

// TestRoleValidateCoversOK covers the happy
// path.
func TestRoleValidateCoversOK(t *testing.T) {
	r := goodRole()
	if err := r.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
}

// TestRoleValidateCoversAllBranches covers
// every branch of the validator. Each case is
// the negation of one field.
func TestRoleValidateCoversAllBranches(t *testing.T) {
	cases := []struct {
		name  string
		mut   func(*Role)
		field string
	}{
		{"empty name", func(r *Role) { r.Name = "" }, "name"},
		{"empty system_prompt", func(r *Role) { r.SystemPrompt = "" }, "system_prompt"},
		{"empty tools", func(r *Role) { r.Tools = nil }, "tools"},
		{"empty tool entry", func(r *Role) { r.Tools = []string{"read", ""} }, "tools[1]"},
		{"whitespace tool entry", func(r *Role) { r.Tools = []string{"   "} }, "tools[0]"},
		{"bad model_class", func(r *Role) { r.ModelClass = "ultra" }, "model_class"},
		{"bad output_format", func(r *Role) { r.OutputFormat = "yaml" }, "output_format"},
		{"negative temperature", func(r *Role) { r.Temperature = -0.1 }, "temperature"},
		{"high temperature", func(r *Role) { r.Temperature = 3 }, "temperature"},
		{"negative max_tokens", func(r *Role) { r.MaxTokens = -1 }, "max_tokens"},
		{"empty stop_sequence", func(r *Role) { r.StopSequences = []string{""} }, "stop_sequences[0]"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := goodRole()
			c.mut(&r)
			err := r.Validate()
			if err == nil {
				t.Fatal("expected error")
			}
			var ire *InvalidRoleError
			if !errors.As(err, &ire) {
				t.Fatalf("err = %v, want InvalidRoleError", err)
			}
			if ire.Field != c.field {
				t.Fatalf("Field = %q, want %q", ire.Field, c.field)
			}
		})
	}
}

// TestRoleValidateNilReceiver covers the
// nil-receiver guard.
func TestRoleValidateNilReceiver(t *testing.T) {
	var r *Role
	if err := r.Validate(); err == nil {
		t.Fatal("expected error")
	}
}

// TestAllClassAndFormatNamesDeterminism covers
// the helper functions the validator's error
// messages use.
func TestAllClassAndFormatNamesDeterminism(t *testing.T) {
	if got := allClassNames(); len(got) != 3 {
		t.Fatalf("allClassNames = %v", got)
	}
	if got := allFormatNames(); len(got) != 3 {
		t.Fatalf("allFormatNames = %v", got)
	}
}
