//go:build !linux

package sandbox

import (
	"os/exec"
)

// namespacePolicy is the POSIX-without-Linux stub.
//
// The namespaces are a Linux feature. macOS has the sandbox
// API, but it is a different mechanism with a different shape,
// and pretending it is the same thing would be dishonest.
type namespacePolicy struct {
	User    bool
	Network bool
	Mount   bool
	PID     bool
}

// configureNamespaces is a no-op off Linux.
func configureNamespaces(cmd *exec.Cmd, policy namespacePolicy) error {
	return nil
}

// namespacesSupported reports false off Linux: there is no
// namespace isolation to support.
func namespacesSupported(policy namespacePolicy) bool {
	return false
}
