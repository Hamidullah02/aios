// Package search is the code-search tool set.
//
// The package exports four tools: GlobTool, GrepTool,
// CodeSearchTool, DiagnosticsTool. Each tool implements the
// tools.Tool interface so the bus can dispatch it like any
// other tool. The package has no third-party dependencies
// and uses only the Go standard library plus aios-tools,
// aios-core, and aios-sandbox.
package search

import (
	"errors"
	"fmt"
)

// Sentinel errors. Per the aios-search SRS §9.
var (
	// ErrCheckerNotFound is returned when the checker binary
	// is not on the PATH (REQ-SRH-ERR-001).
	ErrCheckerNotFound = errors.New("search: checker not found")

	// ErrGrepTimeout is returned when the search exceeds the
	// timeout (REQ-SRH-ERR-002).
	ErrGrepTimeout = errors.New("search: grep timeout")

	// ErrUnsupportedLanguage is returned when the symbol
	// search has no grammar for the file (REQ-SRH-ERR-003).
	ErrUnsupportedLanguage = errors.New("search: unsupported language")
)

// CheckerNotFoundError is the typed error for a missing
// checker binary. The Name is the binary that was not
// found.
type CheckerNotFoundError struct {
	Name string
}

func (e *CheckerNotFoundError) Error() string {
	return fmt.Sprintf("search: checker %q not found on PATH", e.Name)
}

func (e *CheckerNotFoundError) Is(target error) bool { return target == ErrCheckerNotFound }

// GrepTimeoutError is the typed error for a search timeout.
type GrepTimeoutError struct {
	Pattern string
	Timeout string
}

func (e *GrepTimeoutError) Error() string {
	return fmt.Sprintf("search: grep %q timed out after %s", e.Pattern, e.Timeout)
}

func (e *GrepTimeoutError) Is(target error) bool { return target == ErrGrepTimeout }

// UnsupportedLanguageError is the typed error for symbol
// search on a file type without a grammar.
type UnsupportedLanguageError struct {
	Path string
}

func (e *UnsupportedLanguageError) Error() string {
	return fmt.Sprintf("search: no grammar for %q", e.Path)
}

func (e *UnsupportedLanguageError) Is(target error) bool { return target == ErrUnsupportedLanguage }
