package image

import (
	"errors"
	"fmt"
)

// Sentinel errors. REQ-IMG-ERR-001 to 003.
var (
	// ErrImageTooLarge is raised when the image is larger
	// than the maximum size.
	ErrImageTooLarge = errors.New("image: too large")
	// ErrUnsupportedFormat is raised when the image format
	// is not supported.
	ErrUnsupportedFormat = errors.New("image: unsupported format")
	// ErrValidationFailure is raised when the structured
	// output fails validation.
	ErrValidationFailure = errors.New("image: validation failure")
)

// ImageTooLargeError is the typed error for an oversized image.
type ImageTooLargeError struct {
	Path string
	Size int64
	Max  int64
}

func (e *ImageTooLargeError) Error() string {
	return fmt.Sprintf("image: %s is %d bytes, max %d", e.Path, e.Size, e.Max)
}

func (e *ImageTooLargeError) Is(target error) bool { return target == ErrImageTooLarge }

// UnsupportedFormatError is the typed error for an
// unsupported image format.
type UnsupportedFormatError struct {
	Path   string
	Format string
}

func (e *UnsupportedFormatError) Error() string {
	return fmt.Sprintf("image: %s has unsupported format %q", e.Path, e.Format)
}

func (e *UnsupportedFormatError) Is(target error) bool { return target == ErrUnsupportedFormat }

// ValidationFailure is the typed error for a structured
// output validation failure.
type ValidationFailure struct {
	Reason string
}

func (e *ValidationFailure) Error() string {
	return fmt.Sprintf("image: validation failure: %s", e.Reason)
}

func (e *ValidationFailure) Is(target error) bool { return target == ErrValidationFailure }
