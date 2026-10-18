package policy

import "fmt"

// PolicyParseError is raised when the policy file is invalid.
//
// Per REQ-POL-ERR-001 the error includes the file path and
// the line number (when available from the YAML decoder).
type PolicyParseError struct {
	Path string
	Line int
	Err  error
}

func (e *PolicyParseError) Error() string {
	if e.Line > 0 {
		return fmt.Sprintf("policy: parse error in %s at line %d: %v",
			e.Path, e.Line, e.Err)
	}
	return fmt.Sprintf("policy: parse error in %s: %v", e.Path, e.Err)
}

func (e *PolicyParseError) Unwrap() error { return e.Err }

// RuleValidationError is raised when a rule is invalid.
//
// Per REQ-POL-ERR-002 the error includes the rule id and the
// failure reason.
type RuleValidationError struct {
	ID     string
	Reason string
}

func (e *RuleValidationError) Error() string {
	if e.ID == "" {
		return fmt.Sprintf("policy: rule validation: %s", e.Reason)
	}
	return fmt.Sprintf("policy: rule %q invalid: %s", e.ID, e.Reason)
}
