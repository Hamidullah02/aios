//go:build linux

package sandbox

import (
	"os/exec"
	"syscall"
)

// namespacePolicy is the Linux namespace configuration for a run.
//
// The namespaces are the security boundary, not the limits. A
// user namespace maps the child's uid 0 to an unprivileged uid on
// the host, so a child that finds a setuid binary or a kernel
// bug is confined to the identity the runner granted it rather
// than running as the operator.
//
// A network namespace gives the child its own empty network
// stack: no interfaces except loopback, no route to anywhere.
// That is the strongest of the four, and it is what makes the
// egress proxy the ONLY way out.
//
// A mount namespace gives the child its own view of the
// filesystem, so a mount the child makes does not affect the
// host.
//
// A pid namespace gives the child its own pid 1, so the kernel
// reaps its orphans and the runner's kill reaches a tree that
// has re-parented itself.
type namespacePolicy struct {
	// User enables the user namespace.
	User bool
	// Network enables the network namespace.
	Network bool
	// Mount enables the mount namespace.
	Mount bool
	// PID enables the pid namespace.
	PID bool
}

// defaultNamespacePolicy is the policy a Linux run uses.
//
// The user and network namespaces are on because they are the
// two that actually contain a child. The mount and pid
// namespaces are on because they are cheap once the user
// namespace is there, and they close the two escape routes that
// survive without them: a child that mounts something new, and a
// child whose orphans get re-parented and outlive the kill.
var defaultNamespacePolicy = defaultNamespacePolicyValue()

// defaultNamespacePolicyValue returns the internal policy that
// matches the exported one.
func defaultNamespacePolicyValue() namespacePolicy {
	return internalPolicy(DefaultNamespacePolicy())
}

// configureNamespaces puts the child in the configured
// namespaces.
//
// It is called before cmd.Start, so the namespaces are
// established at exec time.
//
// The uid and gid maps are the delicate part. The kernel
// requires that the process writing the map has CAP_SETUID in
// the parent namespace, and it requires that the mapping be
// written before the child execs. Go's exec package handles
// this: it writes the maps from the parent between fork and
// exec.
//
// The map is the identity of the operator's own uid onto the
// child's uid 0. That is what makes the child "root" INSIDE
// the namespace while being an ordinary user on the host. A
// child that tries to setuid(0) inside the namespace succeeds
// and stays unprivileged outside it.
//
// Setgroups is disabled because writing a gid map without the
// privilege to call setgroups is the classic EPERM: the kernel
// refuses the gid map outright. Disabling it means the child
// cannot change its supplementary groups, which is the correct
// trade for a sandbox.
func configureNamespaces(cmd *exec.Cmd, policy namespacePolicy) error {
	if !policy.User && !policy.Network && !policy.Mount && !policy.PID {
		return nil
	}
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}

	// The user namespace must come first in the flag set: the
	// other namespaces need the capabilities the user namespace
	// grants, and the kernel evaluates them in order.
	flags := uintptr(0)
	if policy.User {
		flags |= syscall.CLONE_NEWUSER
	}
	if policy.Network {
		flags |= syscall.CLONE_NEWNET
	}
	if policy.Mount {
		flags |= syscall.CLONE_NEWNS
	}
	if policy.PID {
		flags |= syscall.CLONE_NEWPID
	}
	cmd.SysProcAttr.Cloneflags = flags

	if policy.User {
		uid := syscall.Getuid()
		gid := syscall.Getgid()
		cmd.SysProcAttr.UidMappings = []syscall.SysProcIDMap{
			{ContainerID: 0, HostID: uid, Size: 1},
		}
		cmd.SysProcAttr.GidMappings = []syscall.SysProcIDMap{
			{ContainerID: 0, HostID: gid, Size: 1},
		}
		// Writing the gid map requires the privilege to call
		// setgroups, unless setgroups is disabled first.
		cmd.SysProcAttr.GidMappingsEnableSetgroups = false
	}
	return nil
}

// namespacesSupported reports whether the kernel allows the
// configured namespaces.
//
// It is a probe, not a guarantee: it forks a trivial child with
// the namespaces and checks whether it starts. A container that
// disallows CLONE_NEWUSER, a kernel built without user
// namespaces, and a hardened CI runner all fail this probe, and
// the right answer in all three is the same -- degrade to the
// weaker isolation rather than refuse to run.
//
// The probe is cheap: one exec of /bin/true.
func namespacesSupported(policy namespacePolicy) bool {
	cmd := exec.Command("/bin/true")
	if err := configureNamespaces(cmd, policy); err != nil {
		return false
	}
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}
