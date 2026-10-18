package policy

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Load reads the policy file at path and returns the merged
// list of rules. The include directive is resolved recursively
// with cycle protection (max depth 16).
//
// A policy file looks like:
//
//	rules:
//	  - id: rule-1
//	    match:
//	      tool: read_file
//	    action: allow
//	  - id: rule-2
//	    match:
//	      tool_glob: "write_*"
//	    action: ask
//	    approval:
//	      action: write
//	      resource: "{path}"
//	      reason: "Write to a project file"
//
// Including another file is done at the top level:
//
//	include: ./base.yaml
//	rules: ...
//
// The included file's rules are prepended to the including
// file's rules. The order matters because of first-match-wins.
func Load(path string) ([]Rule, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, &PolicyParseError{Path: path, Err: err}
	}
	visited := make(map[string]bool)
	rules, err := loadFile(abs, abs, visited, 0)
	if err != nil {
		return nil, err
	}
	if err := ValidateAll(rules); err != nil {
		return nil, err
	}
	return rules, nil
}

const maxIncludeDepth = 16

// loadFile loads one file and its includes. basePath is the
// original path passed by the caller; it is used as the
// reported source for PolicyParseError.
func loadFile(filePath, basePath string, visited map[string]bool, depth int) ([]Rule, error) {
	if depth > maxIncludeDepth {
		return nil, &PolicyParseError{
			Path: filePath,
			Err:  fmt.Errorf("include depth exceeds %d", maxIncludeDepth),
		}
	}
	if visited[filePath] {
		return nil, &PolicyParseError{
			Path: filePath,
			Err:  fmt.Errorf("include cycle: %s already loaded", filePath),
		}
	}
	visited[filePath] = true

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, &PolicyParseError{Path: filePath, Err: err}
	}

	// The on-disk shape is: include (optional, single string or list
	// of strings) plus rules (list of Rule).
	var raw struct {
		Include any    `yaml:"include"`
		Rules   []Rule `yaml:"rules"`
	}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, &PolicyParseError{Path: filePath, Err: err}
	}

	var merged []Rule
	if raw.Include != nil {
		includes, err := normalizeIncludeList(raw.Include)
		if err != nil {
			return nil, &PolicyParseError{Path: filePath, Err: err}
		}
		base := filepath.Dir(filePath)
		for _, inc := range includes {
			incAbs, err := NormalizePath(base, inc)
			if err != nil {
				return nil, &PolicyParseError{Path: filePath, Err: err}
			}
			included, err := loadFile(incAbs, basePath, visited, depth+1)
			if err != nil {
				return nil, err
			}
			merged = append(merged, included...)
		}
	}
	merged = append(merged, raw.Rules...)
	return merged, nil
}

// normalizeIncludeList accepts a string (one path) or a list
// of strings (multiple paths). Any other YAML shape is an error.
func normalizeIncludeList(v any) ([]string, error) {
	switch t := v.(type) {
	case string:
		return []string{t}, nil
	case []any:
		out := make([]string, 0, len(t))
		for _, e := range t {
			s, ok := e.(string)
			if !ok {
				return nil, fmt.Errorf("include list must contain only strings; got %T", e)
			}
			out = append(out, s)
		}
		return out, nil
	default:
		return nil, fmt.Errorf("include must be a string or a list of strings; got %T", v)
	}
}
