package memory

import (
	"errors"
	"fmt"
)

// ErrNotFound is returned by Store.Get when the
// artifact is not in the store. The sentinel lets
// callers match with errors.Is regardless of the
// concrete value.
var ErrNotFound = errors.New("memory: artifact not found")

// ErrIndexCorrupt is returned by Index.Load when the
// index file fails to parse.
var ErrIndexCorrupt = errors.New("memory: index file is corrupt")

// ErrBudgetTooSmall is returned by Compaction.Compact
// when the budget is smaller than the smallest
// artifact.
var ErrBudgetTooSmall = errors.New("memory: budget is smaller than the smallest artifact")

// ErrPinSetCorrupt is returned by PinSet.Load when the
// pin set file fails to parse.
var ErrPinSetCorrupt = errors.New("memory: pin set file is corrupt")

// ErrInvalidArtifact is returned by NewArtifact and
// Store.Put when the artifact is missing required
// fields or has an invalid kind/scope.
var ErrInvalidArtifact = errors.New("memory: invalid artifact")

// ErrStoreCorrupt is returned by Store.Get when the
// on-disk file cannot be read (e.g. permission error).
var ErrStoreCorrupt = errors.New("memory: store file is corrupt")

// MemoryError is the typed error for the memory
// package. The error message includes the failing
// operation and the reason. Subtypes wrap a specific
// reason; the Is method lets callers detect a category
// of failure regardless of the concrete value.
type MemoryError struct {
	Op     string
	Reason string
}

func (e *MemoryError) Error() string {
	return fmt.Sprintf("memory: %s: %s.", e.Op, e.Reason)
}

// Is reports whether target matches this error's
// category. The method lets callers use errors.Is to
// detect a memory failure regardless of the concrete
// value.
func (e *MemoryError) Is(target error) bool {
	_, ok := target.(*MemoryError)
	return ok
}
