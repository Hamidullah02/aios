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
// This package is NOT a container, and it is not a chroot. It
// does not give the child a restricted filesystem view of its
// own, and it does not install a seccomp filter or an LSM
// policy. On Linux the mount namespace isolates the child's
// mounts from the host's, but the child can still read the whole
// filesystem it has permission to read.
//
// # What this package does provide
//
// The package provides a set of restrictions that hold:
//
//   - The environment is cleared by default. The child gets the
//     variables the caller listed, not the parent's environment.
//   - Resource limits bound CPU, memory and process count on
//     Linux, and the wall-clock timeout bounds the run
//     everywhere.
//   - The network policy rewrites the child's proxy variables and
//     removes any that were inherited, so a well-behaved HTTP
//     client can only reach the hosts the allow list names.
//   - The process tree dies with the run. The child is in its
//     own process group on POSIX, or its own Job Object on
//     Windows, and the timeout kills the whole tree. A command
//     that backgrounds work does not leave that work running.
//   - On Linux the child runs in its own user, mount and pid
//     namespaces. The user namespace maps the child's uid 0 onto
//     an unprivileged uid on the host, the mount namespace gives
//     the child its own filesystem view, and the pid namespace
//     makes the kernel reap the child's orphans.
//
// # What the namespaces do not buy
//
// On Linux, a child in a user namespace is unprivileged with
// respect to the host, but it is still root inside the namespace.
// It can read every file the operator can read, and the network
// policy is its only network boundary. For a hard guarantee, run
// the whole process inside a container.
//
// The network namespace is deliberately not used. The egress
// proxy is the network boundary, and it is bound to loopback in
// the parent's network namespace; a child in its own network
// namespace cannot reach it. See DefaultNamespacePolicy.
//
// The Windows path is stronger than the Linux path in one
// respect: the Job Object is a kernel-enforced boundary that a
// descendant cannot escape by changing its parent.
//
// The word "sandbox" in the package name is a description of
// intent, not a claim of equivalence with a container. Real
// isolation -- a chroot, a container, a VM -- would be layered
// on top of this package, not replaced by it.
package sandbox
