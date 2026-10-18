// Package sandbox runs untrusted commands in a safe environment.
//
// The package provides three things:
//
//   - Runner: the runner interface that callers depend on.
//   - SubprocessRunner: the os/exec wrapper that implements Runner.
//   - ResourceLimits: the typed limits object the runner applies.
//
// The wrapper sets the working directory, sets the environment
// variables, applies the resource limits, and waits for the
// process with the timeout. On POSIX systems the runner enforces
// the CPU, memory, and process limits through syscall.Setrlimit.
// On Windows the runner falls back to a timeout because
// syscall.Setrlimit is not supported there.
package sandbox
