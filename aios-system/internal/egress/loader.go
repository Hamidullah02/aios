// Package egress loader: read and validate a YAML egress policy.
//
// The on-disk shape (per REQ-EGS-DAT-001) is:
//
//	allow:
//	  - api.openai.com
//	  - "*.example.com"
//	deny:
//	  - "*.evil.com"
//	headers:
//	  remove:
//	    - X-Internal-Token
//	  add:
//	    X-Aios-Request: "true"
//
// The file is optional; absence is treated as an empty allow list
// (every request denied). The default path is "./var/egress.yaml";
// callers can pass any other path to Load.
package egress

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// fileShape is the on-disk representation. Keeping it private lets
// the public API hide the YAML tags.
type fileShape struct {
	Allow   []string         `yaml:"allow,omitempty"`
	Deny    []string         `yaml:"deny,omitempty"`
	Headers headersFileShape `yaml:"headers,omitempty"`
}

type headersFileShape struct {
	Remove []string          `yaml:"remove,omitempty"`
	Add    map[string]string `yaml:"add,omitempty"`
}

// LoadFile loads the egress policy file at path and returns the
// typed AllowList and HeaderRules. The file does not have to exist;
// a missing file produces empty (deny-by-default) rules and a nil
// error, because a deployment may run without an egress policy in
// tests.
//
// An invalid file (parse error, duplicate header name, empty header
// name) returns a *ParseError wrapping the cause.
func LoadFile(path string) (*AllowList, *HeaderRules, error) {
	if path == "" {
		return nil, nil, &ParseError{Reason: "path is empty"}
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, nil, &ParseError{Path: path, Reason: err.Error()}
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		if os.IsNotExist(err) {
			// Missing file is not an error: deny by default.
			return NewAllowList(nil, nil), NewHeaderRules(nil, nil), nil
		}
		return nil, nil, &ParseError{Path: abs, Reason: err.Error()}
	}
	return Parse(data, abs)
}

// Parse parses the YAML bytes and returns the typed structures.
// The path argument is used only for error reporting and may be
// the empty string when the input did not come from a file.
func Parse(data []byte, path string) (*AllowList, *HeaderRules, error) {
	var raw fileShape
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, nil, &ParseError{Path: path, Reason: err.Error()}
	}
	allow := NewAllowList(raw.Allow, raw.Deny)
	rules, err := buildHeaderRules(raw.Headers, path)
	if err != nil {
		return nil, nil, err
	}
	return allow, rules, nil
}

// buildHeaderRules converts the file-shape headers into the typed
// HeaderRules. It validates header names and rejects duplicates.
func buildHeaderRules(in headersFileShape, path string) (*HeaderRules, error) {
	remove := make([]HeaderRule, 0, len(in.Remove))
	for _, name := range in.Remove {
		canonical, err := validateHeaderName(name)
		if err != nil {
			return nil, &ParseError{Path: path, Reason: err.Error()}
		}
		remove = append(remove, HeaderRule{Name: canonical})
	}
	add := make([]HeaderRule, 0, len(in.Add))
	for name, value := range in.Add {
		canonical, err := validateHeaderName(name)
		if err != nil {
			return nil, &ParseError{Path: path, Reason: err.Error()}
		}
		add = append(add, HeaderRule{Name: canonical, Value: value})
	}
	return NewHeaderRules(remove, add), nil
}
