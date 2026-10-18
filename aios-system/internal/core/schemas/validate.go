package schemas

import (
	"fmt"
	"net/mail"
	"regexp"
	"strings"
)

// RequiredString returns a ValidationError if s is empty, otherwise nil.
func RequiredString(path, s string) error {
	if s == "" {
		return NewValidationError(path, "required")
	}
	return nil
}

// OptionalString returns a ValidationError if s is not empty but contains
// a NUL byte or other obvious garbage. Empty is OK.
func OptionalString(path, s string) error {
	if s == "" {
		return nil
	}
	if strings.ContainsRune(s, '\x00') {
		return NewValidationError(path, "contains NUL byte")
	}
	return nil
}

// NonEmptyList returns a ValidationError if the list is nil or empty.
func NonEmptyList[T any](path string, list []T) error {
	if len(list) == 0 {
		return NewValidationError(path, "required, at least one item")
	}
	return nil
}

// ListOfStrings validates a list of strings, all required.
func ListOfStrings(path string, list []string) error {
	if len(list) == 0 {
		return NewValidationError(path, "required, at least one item")
	}
	for i, s := range list {
		if err := RequiredString(fmt.Sprintf("%s.%d", path, i), s); err != nil {
			return err
		}
	}
	return nil
}

// Email returns a ValidationError if s is not a syntactically valid email.
// Empty string is OK (use RequiredString to require).
func Email(path, s string) error {
	if s == "" {
		return nil
	}
	if _, err := mail.ParseAddress(s); err != nil {
		return NewValidationError(path, "invalid email: "+err.Error())
	}
	return nil
}

// uuidV4Pattern matches a canonical lowercase UUIDv4 string.
var uuidV4Pattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

// UUIDv4 returns a ValidationError if s is not a canonical lowercase UUIDv4.
func UUIDv4(path, s string) error {
	if s == "" {
		return NewValidationError(path, "required")
	}
	if !uuidV4Pattern.MatchString(s) {
		return NewValidationError(path, "must be a canonical lowercase UUIDv4")
	}
	return nil
}

// InRange returns a ValidationError if n is outside [lo, hi] inclusive.
func InRange(path string, n, lo, hi int) error {
	if n < lo || n > hi {
		return NewValidationError(path, fmt.Sprintf("must be in [%d, %d]", lo, hi))
	}
	return nil
}

// OneOf returns a ValidationError if s is not in the allowed set.
func OneOf(path, s string, allowed ...string) error {
	for _, a := range allowed {
		if s == a {
			return nil
		}
	}
	return NewValidationError(path, fmt.Sprintf("must be one of %v", allowed))
}

// MinLen returns a ValidationError if s is shorter than n.
func MinLen(path, s string, n int) error {
	if len(s) < n {
		return NewValidationError(path, fmt.Sprintf("must be at least %d characters", n))
	}
	return nil
}
