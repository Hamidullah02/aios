package sandbox

import "fmt"

// NamespacePolicy is the set of Linux namespaces a sandboxed
// child runs in.
//
// The namespaces are the part of the sandbox that is a real
// security boundary. A resource limit says "this process may
// use this much CPU"; a namespace says "this process cannot see
// or touch anything outside this set". The limits bound the
// damage a child does to the machine. The namespaces decide
// what the child can reach at all.
//
// The four namespaces and what each one buys:
//
//   - User: the child's uid 0 maps to an unprivileged uid on
//     the host. A child that finds a setuid binary, a kernel
//     bug, or a confused-deputy path is confined to the
//     identity the operator granted rather than running as the
//     operator.
//   - Network: the child gets its own empty network stack.
//     No interface except loopback, no route to anywhere. This
//     is what makes the egress proxy the only way out, and it
//     is the strongest of the four.
//   - Mount: the child gets its own filesystem view. A mount
//     the child makes does not affect the host.
//   - PID: the child gets its own pid 1. The kernel reaps its
//     orphans, so a child whose grandchildren get re-parented
//     cannot outlive the kill.
//
// # Platform support
//
// The namespaces are a Linux feature. On macOS and Windows the
// policy is accepted and ignored; the equivalent containment
// there is the Job Object (Windows) and the process group
// (POSIX).
//
// # Degradation
//
// A kernel without user namespaces, a container that forbids
// CLONE_NEWUSER, and a hardened CI runner all reject the
// namespaces. The runner probes for support once and falls back
// to the weaker isolation rather than refusing to run. The
// fallback is reported, not hidden: see NamespaceSupport.
type NamespacePolicy struct {
	// User enables the user namespace.
	User bool
	// Network enables the network namespace.
	Network bool
	// Mount enables the mount namespace.
	Mount bool
	// PID enables the pid namespace.
	PID bool
}

// DefaultNamespacePolicy returns the policy a Linux run uses.
//
// It is the exported form of the package default: the user,
// network, mount and pid namespaces, all enabled.
func DefaultNamespacePolicy() NamespacePolicy {
	return NamespacePolicy{User: true, Network: true, Mount: true, PID: true}
}

// Validate checks the policy is well-formed.
//
// An entirely disabled policy is valid: it means "no namespace
// isolation", which is what a caller sets when it wants the old
// behaviour on purpose.
func (p *NamespacePolicy) Validate() error {
	if p == nil {
		return nil
	}
	return nil
}

// NamespaceError is raised when a namespace policy cannot be
// applied.
type NamespaceError struct {
	Field  string
	Reason string
}

func (e *NamespaceError) Error() string {
	return fmt.Sprintf("sandbox: namespace: %s: %s.", e.Field, e.Reason)
}

// Is reports whether target is also a *NamespaceError.
func (e *NamespaceError) Is(target error) bool {
	_, ok := target.(*NamespaceError)
	return ok
}

// internalPolicy converts an exported policy to the internal
// one.
//
// The two types exist because the internal one is platform
// specific: the Linux file names the namespaces it supports, and
// the exported one is the API callers use on every platform.
func internalPolicy(p NamespacePolicy) namespacePolicy {
	return namespacePolicy{
		User:    p.User,
		Network: p.Network,
		Mount:   p.Mount,
		PID:     p.PID,
	}
}

// NamespacesSupported reports whether the platform and the
// kernel allow the namespace isolation.
//
// It is a probe, not a compile-time answer: the namespaces are a
// runtime kernel feature, and the same binary runs on a machine
// that allows them and one that does not. A container that
// forbids CLONE_NEWUSER and a kernel built without user
// namespaces both make this false.
//
// The probe is cached, because it costs one exec of /bin/true and
// the answer does not change while the process runs.
func NamespacesSupported() bool {
	return namespacesSupported(internalPolicy(DefaultNamespacePolicy()))
}
