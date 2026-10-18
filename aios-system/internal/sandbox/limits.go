package sandbox

import (
	"errors"
	"fmt"
)

// ResourceLimits is the typed object that holds the resource
// limits for one sandbox run.
//
// All four core fields are optional. The time limit is
// honored through the timeout argument. The CPU, memory,
// and process limits are honored through syscall.Setrlimit
// on POSIX systems. On Windows the runner falls back to a
// timeout.
//
// Per the doc set the time limit is the only field that is
// always required by the runner, so the other three are
// zero-valued when the caller does not set them. The zero
// value of a limit is "no limit", which matches the POSIX
// RLIM_INFINITY semantics for a syscall.Rlimit with the Cur
// field set to the platform infinity.
//
// The UID and GID fields are an extension that is honored
// on POSIX through syscall.Setuid and syscall.Setgid. The
// fields are ignored on Windows. A non-negative UID is
// honored as the target user; a negative UID is treated
// as "do not drop privileges". REQ-SND-ISO-003.
type ResourceLimits struct {
	// CPUSeconds is the CPU time limit in seconds.
	CPUSeconds int
	// MemoryBytes is the address-space limit in bytes.
	MemoryBytes int64
	// TimeSeconds is the wall-clock limit in seconds.
	TimeSeconds int
	// Processes is the process count limit. The runner
	// applies this on POSIX via RLIMIT_NPROC; the field is
	// ignored on Windows.
	Processes int
	// UID is the target user. A negative value means "do
	// not drop privileges". The field is honored on POSIX
	// only.
	UID int
	// GID is the target group. A negative value means "do
	// not drop privileges". The field is honored on POSIX
	// only.
	GID int
}

// Validate checks the limits are well-formed. A limit is
// well-formed when every set value is positive. The time
// limit is the only field whose absence the runner treats
// as a configuration error: the runner needs a wall-clock
// budget to wait on the process. The other three fields
// are optional; a zero value means "no limit".
func (l ResourceLimits) Validate() error {
	if l.TimeSeconds <= 0 {
		return &LimitsError{Field: "time_seconds", Reason: "must be positive"}
	}
	if l.CPUSeconds < 0 {
		return &LimitsError{Field: "cpu_seconds", Reason: "must be non-negative"}
	}
	if l.MemoryBytes < 0 {
		return &LimitsError{Field: "memory_bytes", Reason: "must be non-negative"}
	}
	if l.Processes < 0 {
		return &LimitsError{Field: "processes", Reason: "must be non-negative"}
	}
	return nil
}

// LimitsError is raised by ResourceLimits.Validate when one
// of the fields is outside the accepted range.
//
// Per the project coding standard, errors use a specific
// type per package. The error message starts with a
// lowercase letter and ends with a period.
type LimitsError struct {
	Field  string
	Reason string
}

func (e *LimitsError) Error() string {
	return fmt.Sprintf("sandbox: limits: %s: %s.", e.Field, e.Reason)
}

// ErrLimitsNotSupported is returned by Limits.Apply on
// platforms where the rlimit API is not available. The
// caller is expected to fall back to a timeout.
var ErrLimitsNotSupported = errors.New("sandbox: limits: platform does not support rlimit")
