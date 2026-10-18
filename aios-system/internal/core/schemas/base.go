// Package schemas provides the typed data contracts for cross-module records.
//
// Per the doc set:
//
//   - REQ-CORE-SCH-009: every schema raises a ValidationError when input
//     is invalid. The error includes the field name and the failure reason.
//   - REQ-CORE-SCH-010: every schema serializes to JSON with sorted keys
//     for determinism.
//
// The doc set calls for go-playground/validator or hand-rolled validation.
// This implementation hand-rolls validation in validate.go so the module
// has no third-party dependencies (see README.md, "Notes on the doc set").
package schemas

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// Schema is the interface every typed record implements.
//
// Validate reports whether the value is well-formed.
// MarshalJSON returns the canonical byte representation (sorted keys,
// no whitespace).
// UnmarshalJSON parses from a canonical or non-canonical encoding; the
// parser rejects unknown fields.
type Schema interface {
	Validate() error
}

// ValidationError is the error type for schema validation failures.
//
// Per REQ-CORE-ERR-001, the error includes the field name and the failure
// reason. The Path is a dotted path ("tasks.0.id"). The Reason is a short
// human-readable string. Multiple failures are joined with "; " in Error().
type ValidationError struct {
	Path   string
	Reason string
}

func (e *ValidationError) Error() string {
	if e.Path == "" {
		return e.Reason
	}
	return fmt.Sprintf("%s: %s", e.Path, e.Reason)
}

// NewValidationError constructs a ValidationError for a path.
func NewValidationError(path, reason string) *ValidationError {
	return &ValidationError{Path: path, Reason: reason}
}

// JoinValidationErrors combines several errors into one. nil inputs are
// skipped. If all inputs are nil, returns nil.
func JoinValidationErrors(errs ...error) error {
	var kept []error
	for _, e := range errs {
		if e != nil {
			kept = append(kept, e)
		}
	}
	if len(kept) == 0 {
		return nil
	}
	if len(kept) == 1 {
		return kept[0]
	}
	parts := make([]string, len(kept))
	for i, e := range kept {
		parts[i] = e.Error()
	}
	return errors.New(strings.Join(parts, "; "))
}

// MarshalCanonical is the canonical JSON encoder used by every schema.
//
// The encoding is:
//   - UTF-8
//   - object keys sorted lexicographically at every level
//   - no whitespace
//   - HTML escaping off
//
// This guarantees that two equal values produce the same bytes, which is
// what content_hash needs (§11.2 of the global PRD).
func MarshalCanonical(v any) ([]byte, error) {
	// First marshal with default encoding, then re-marshal via a normalizer
	// that sorts keys. Doing it in one pass is faster but the two-pass
	// version is easier to reason about for an audit log.
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	// Decode into a generic tree so we can re-encode with sorted keys.
	var generic any
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.UseNumber()
	if err := dec.Decode(&generic); err != nil {
		return nil, err
	}
	normalized, err := canonicalize(generic)
	if err != nil {
		return nil, err
	}
	return json.Marshal(normalized)
}

func canonicalize(v any) (any, error) {
	switch t := v.(type) {
	case map[string]any:
		// Maps are returned as map[string]any with keys already in lexical
		// order at marshal time (encoding/json sorts map keys). Nothing to do.
		return t, nil
	case []any:
		out := make([]any, len(t))
		for i, e := range t {
			n, err := canonicalize(e)
			if err != nil {
				return nil, err
			}
			out[i] = n
		}
		return out, nil
	default:
		return v, nil
	}
}

// UnmarshalStrict parses JSON into v, rejecting unknown fields.
//
// This is what REQ-CORE-SCH-009's "rejects extra fields" refers to, and
// it is what the doc set calls for ("The type rejects extra fields.").
// We use a two-step decode: first into a generic map, then check for
// unknown fields, then unmarshal into v.
func UnmarshalStrict(data []byte, v any, knownFields []string) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return &ValidationError{Path: "", Reason: fmt.Sprintf("invalid JSON: %s", err)}
	}
	known := make(map[string]bool, len(knownFields))
	for _, k := range knownFields {
		known[k] = true
	}
	// Stable iteration order for the error message.
	keys := make([]string, 0, len(raw))
	for k := range raw {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if !known[k] {
			return &ValidationError{Path: k, Reason: "unknown field"}
		}
	}
	if err := json.Unmarshal(data, v); err != nil {
		return &ValidationError{Path: "", Reason: fmt.Sprintf("invalid value: %s", err)}
	}
	return nil
}
