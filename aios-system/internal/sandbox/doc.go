// Package sandbox runs untrusted commands in a restricted
// environment.
//
// The package provides three things:
//
//   - Runner: the runner interface that callers depend on.
//   - SubprocessRunner: the os/exec wrapper that implements Runner.
//   - ResourceLimits: the typed limits object the runner applies.
//   - NetworkPolicy: the network scope the runner applies to
//     every child it starts.
//
// The wrapper sets the working directory, sets the environment
// variables, applies the resource limits, and waits for the
// process with the timeout. On POSIX systems the runner enforces
// the CPU, memory, and process limits through syscall.Setrlimit.
// On Windows the runner falls back to a timeout because
// syscall.Setrlimit is not supported there.
//
// # What this package is not
//
// This package is NOT a security boundary in the sense that a
// container, a jail, or a VM is a security boundary. It does not
// create a namespace, a container, a seccomp filter, or a
// filesystem jail. A child it starts runs with the privileges of
// the parent process, and a child that wants to reach the disk
// outside its working directory can do so.
//
// What it does provide is a set of restrictions that hold for a
// child that cooperates with them:
//
//   - The environment is cleared by default. The child gets the
//     variables the caller listed, not the parent's environment.
//   - Resource limits bound CPU, memory and process count on
//     POSIX, and the wall-clock timeout bounds the run
//     everywhere.
//   - The network policy rewrites the child's proxy variables and
//     removes any that were inherited, so a well-behaved HTTP
//     client can only reach the hosts the allow list names.
//
// The word "sandbox" in the package name is aspirational. The
// honest description is a restricted subprocess runner. Real
// isolation -- Linux namespaces, bubblewrap, a container, or a
// restricted token on Windows -- would be layered on top of this
// package, not replaced by it.
package sandbox
