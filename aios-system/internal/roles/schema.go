package roles

import (
	"fmt"
	"strings"

	"github.com/sadi/aios-system/internal/core/schemas"
)

// ModelClass is the model capability level the
// role targets (REQ-ROL-DAT-001). The class
// matches the aios-core ModelCapability class
// enum ("cheap" | "mid" | "strong").
type ModelClass string

const (
	// ClassCheap targets the cheap model class.
	ClassCheap ModelClass = "cheap"
	// ClassMid targets the mid model class.
	ClassMid ModelClass = "mid"
	// ClassStrong targets the strong model class.
	ClassStrong ModelClass = "strong"
)

// IsValid reports whether c is one of the three
// known classes.
func (c ModelClass) IsValid() bool {
	switch c {
	case ClassCheap, ClassMid, ClassStrong:
		return true
	}
	return false
}

// AllClasses returns the three known classes in
// deterministic order. The function is the
// helper the YAML loader and the tests use.
func AllClasses() []ModelClass {
	return []ModelClass{ClassCheap, ClassMid, ClassStrong}
}

// OutputFormat is the typed enum for the role
// output format (REQ-ROL-CFG-002).
type OutputFormat string

const (
	// OutputText is the free-text output format.
	OutputText OutputFormat = "text"
	// OutputJSON is the structured JSON output
	// format.
	OutputJSON OutputFormat = "json"
	// OutputCode is the code output format.
	OutputCode OutputFormat = "code"
)

// IsValid reports whether f is one of the three
// known formats.
func (f OutputFormat) IsValid() bool {
	switch f {
	case OutputText, OutputJSON, OutputCode:
		return true
	}
	return false
}

// AllFormats returns the three known formats in
// deterministic order.
func AllFormats() []OutputFormat {
	return []OutputFormat{OutputText, OutputJSON, OutputCode}
}

// Role is the typed object that holds the agent
// configuration (REQ-ROL-DAT-001).
//
// The role is the unit the registry stores and
// the resolver returns. The role is content-
// addressed through the aios-core canonical
// JSON encoder; the registry's set + get
// operations are pure functions of the role
// fields.
type Role struct {
	// Name is the role name. The field is the
	// registry's lookup key.
	Name string `json:"name" yaml:"name"`
	// SystemPrompt is the system prompt the
	// model sees. The field is the canonical
	// "behave like X" prompt.
	SystemPrompt string `json:"system_prompt" yaml:"system_prompt"`
	// Tools is the list of tool names the role
	// is allowed to call.
	Tools []string `json:"tools" yaml:"tools"`
	// ModelClass is the model class the role
	// targets.
	ModelClass ModelClass `json:"model_class" yaml:"model_class"`
	// OutputFormat is the model's output
	// format.
	OutputFormat OutputFormat `json:"output_format" yaml:"output_format"`
	// Temperature is the model's temperature.
	// The default is 0 (deterministic). The
	// field is optional in the YAML and JSON.
	Temperature float64 `json:"temperature,omitempty" yaml:"temperature,omitempty"`
	// MaxTokens is the model's max-tokens cap.
	// The field is optional; 0 means "model
	// default".
	MaxTokens int `json:"max_tokens,omitempty" yaml:"max_tokens,omitempty"`
	// StopSequences is the list of stop
	// sequences the model should stop on.
	// The field is optional.
	StopSequences []string `json:"stop_sequences,omitempty" yaml:"stop_sequences,omitempty"`
}

// Validate checks the role is well-formed. The
// function is the gate the registry calls when
// a role is added or loaded from YAML
// (REQ-ROL-CFG-001 to 005; REQ-ROL-ERR-002).
func (r *Role) Validate() error {
	if r == nil {
		return &InvalidRoleError{Field: "role", Reason: "is nil"}
	}
	// Name: required, non-empty.
	if err := schemas.RequiredString("name", r.Name); err != nil {
		return &InvalidRoleError{Field: "name", Reason: err.Error()}
	}
	// SystemPrompt: required, non-empty.
	if err := schemas.RequiredString("system_prompt", r.SystemPrompt); err != nil {
		return &InvalidRoleError{Name: r.Name, Field: "system_prompt", Reason: err.Error()}
	}
	// Tools: required, must be a list of
	// non-empty strings.
	if len(r.Tools) == 0 {
		return &InvalidRoleError{Name: r.Name, Field: "tools", Reason: "must be non-empty"}
	}
	for i, t := range r.Tools {
		if strings.TrimSpace(t) == "" {
			return &InvalidRoleError{
				Name:   r.Name,
				Field:  fmt.Sprintf("tools[%d]", i),
				Reason: "must be a non-empty string",
			}
		}
	}
	// ModelClass: one of cheap/mid/strong.
	if !r.ModelClass.IsValid() {
		return &InvalidRoleError{
			Name:   r.Name,
			Field:  "model_class",
			Reason: fmt.Sprintf("must be one of %v", allClassNames()),
		}
	}
	// OutputFormat: one of text/json/code.
	if !r.OutputFormat.IsValid() {
		return &InvalidRoleError{
			Name:   r.Name,
			Field:  "output_format",
			Reason: fmt.Sprintf("must be one of %v", allFormatNames()),
		}
	}
	// Temperature: optional, but must be in
	// [0, 2] when set.
	if r.Temperature < 0 || r.Temperature > 2 {
		return &InvalidRoleError{
			Name:   r.Name,
			Field:  "temperature",
			Reason: "must be in [0, 2]",
		}
	}
	// MaxTokens: optional, but must be
	// non-negative.
	if r.MaxTokens < 0 {
		return &InvalidRoleError{
			Name:   r.Name,
			Field:  "max_tokens",
			Reason: "must be non-negative",
		}
	}
	// StopSequences: optional. Each entry, when
	// present, must be a non-empty string.
	for i, s := range r.StopSequences {
		if s == "" {
			return &InvalidRoleError{
				Name:   r.Name,
				Field:  fmt.Sprintf("stop_sequences[%d]", i),
				Reason: "must be a non-empty string",
			}
		}
	}
	return nil
}

// allClassNames is the helper the
// validation-error message uses. The helper
// produces the deterministic string-list form
// the error message renders.
func allClassNames() []string {
	cs := AllClasses()
	out := make([]string, len(cs))
	for i, c := range cs {
		out[i] = string(c)
	}
	return out
}

// allFormatNames is the helper the
// validation-error message uses.
func allFormatNames() []string {
	fs := AllFormats()
	out := make([]string, len(fs))
	for i, f := range fs {
		out[i] = string(f)
	}
	return out
}
