package sandbox

import "testing"

// TestDefaultNamespacePolicy_LeavesTheNetworkNamespaceOff asserts
// that the default policy does not use the network namespace.
//
// # Why
//
// The egress proxy is the product's network boundary, and it is
// bound to 127.0.0.1 in the parent's network namespace. A child
// in its own network namespace has a loopback interface in the
// DOWN state: no address, no route, and a connect to 127.0.0.1
// fails with EHOSTUNREACH ("network is unreachable").
//
// So the child could not reach the proxy. The allow list never
// ran, because the request never got as far as the proxy to be
// refused: the child simply had no network at all.
//
// The two are mutually exclusive as they stand. Making the network
// namespace work would need a veth pair between the two
// namespaces and a proxy bound to the parent's end of it, which
// is not implemented. Until it is, the honest configuration is the
// one where the boundary that exists is the boundary that applies.
//
// The user, mount and pid namespaces stay on: they do not conflict
// with the proxy.
func TestDefaultNamespacePolicy_LeavesTheNetworkNamespaceOff(t *testing.T) {
	p := DefaultNamespacePolicy()
	if p.Network {
		t.Error("the default policy must not use the network namespace: the child cannot reach the egress proxy")
	}
	if !p.User {
		t.Error("the user namespace should be on")
	}
	if !p.Mount {
		t.Error("the mount namespace should be on")
	}
	if !p.PID {
		t.Error("the pid namespace should be on")
	}
}
