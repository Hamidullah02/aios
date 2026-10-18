// Package edit is the code-edit tool set.
//
// The package exports five tools: EditTool, MultiEditTool,
// ApplyPatchTool, NotebookEditTool, WriteTool. Each tool
// implements the tools.Tool interface so the bus can dispatch
// it like any other tool. The package has no third-party
// dependencies and uses only the Go standard library plus
// aios-tools.
package edit

import (
	"errors"
	"fmt"
)

// Sentinel errors. Per the aios-edit SRS §9.
var (
	// ErrEditNotUnique is returned when old_string matches
	// more than one location and replace_all is false
	// (REQ-EDT-ERR-001).
	ErrEditNotUnique = errors.New("edit: not unique")

	// ErrEditNotFound is returned when old_string does not
	// match (REQ-EDT-ERR-002).
	ErrEditNotFound = errors.New("edit: not found")

	// ErrEditEmpty is returned when old_string is empty.
	ErrEditEmpty = errors.New("edit: empty old_string")

	// ErrPatchParse is returned when the patch is malformed
	// (REQ-EDT-ERR-003).
	ErrPatchParse = errors.New("edit: patch parse")

	// ErrPatchContext is returned when the hunk context does
	// not match the file (REQ-EDT-ERR-004).
	ErrPatchContext = errors.New("edit: patch context mismatch")
)

// EditNotUniqueError is the typed error for non-unique
// matches. The Count is the number of times old_string was
// found.
type EditNotUniqueError struct {
	Path  string
	Count int
}

func (e *EditNotUniqueError) Error() string {
	return fmt.Sprintf("edit: %q: %d matches (set replace_all to replace every match)", e.Path, e.Count)
}

func (e *EditNotUniqueError) Is(target error) bool { return target == ErrEditNotUnique }

// EditNotFoundError is the typed error for missing matches.
type EditNotFoundError struct {
	Path string
}

func (e *EditNotFoundError) Error() string {
	return fmt.Sprintf("edit: %q: old_string not found", e.Path)
}

func (e *EditNotFoundError) Is(target error) bool { return target == ErrEditNotFound }

// EditEmptyError is the typed error for empty old_string.
type EditEmptyError struct{ Path string }

func (e *EditEmptyError) Error() string {
	return fmt.Sprintf("edit: %q: old_string is empty", e.Path)
}

func (e *EditEmptyError) Is(target error) bool { return target == ErrEditEmpty }

// MultiEditError is the typed error for MultiEdit failures.
type MultiEditError struct {
	Path   string
	Index  int
	Reason error
}

func (e *MultiEditError) Error() string {
	return fmt.Sprintf("edit: multi_edit %q: edit %d failed: %v", e.Path, e.Index, e.Reason)
}

func (e *MultiEditError) Unwrap() error { return e.Reason }

// PatchParseError is the typed error for malformed patches.
type PatchParseError struct {
	Patch string
	Line  int
	Msg   string
}

func (e *PatchParseError) Error() string {
	if e.Line > 0 {
		return fmt.Sprintf("edit: patch: line %d: %s", e.Line, e.Msg)
	}
	return fmt.Sprintf("edit: patch: %s", e.Msg)
}

func (e *PatchParseError) Is(target error) bool { return target == ErrPatchParse }

// PatchContextMismatch is the typed error for hunk context
// failures.
type PatchContextMismatch struct {
	Path   string
	Hunk   int
	Line   int
	Expect string
	Got    string
}

func (e *PatchContextMismatch) Error() string {
	return fmt.Sprintf("edit: patch %q hunk %d line %d: context mismatch (got %q, want %q)", e.Path, e.Hunk, e.Line, e.Got, e.Expect)
}

func (e *PatchContextMismatch) Is(target error) bool { return target == ErrPatchContext }
