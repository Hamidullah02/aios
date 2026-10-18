// Package egress is the egress proxy for aios.
//
// The proxy is a deny-by-default HTTP forwarder. Every request that
// reaches the proxy is checked against the configured allow list. Hosts
// that are not in the allow list, or that match the deny list, are
// refused with a 403 and the decision is written to the audit log.
//
// The package exports three things:
//
//   - AllowList: the typed allow + deny rules.
//   - HeaderRules: the typed header rewriting rules (remove + add).
//   - Proxy: the HTTP server that ties the two together, plus the audit
//     log integration.
//
// The package has no third-party dependencies beyond aios-core and the
// Go standard library.
package egress

import "fmt"

// HostNotAllowedError is raised when a request targets a host that
// is not in the allow list or that is in the deny list.
//
// Per REQ-EGS-ERR-001 the error includes the host so the operator can
// see what was refused.
type HostNotAllowedError struct {
	Host string
}

func (e *HostNotAllowedError) Error() string {
	return fmt.Sprintf("egress: host %q is not in the allow list", e.Host)
}

// UpstreamError is raised when the upstream returns an error or the
// forwarder cannot reach it.
//
// Per REQ-EGS-ERR-002 the error wraps the underlying cause.
type UpstreamError struct {
	Host   string
	Method string
	Path   string
	Err    error
}

func (e *UpstreamError) Error() string {
	if e.Err == nil {
		return fmt.Sprintf("egress: upstream error for %s %s%s", e.Method, e.Host, e.Path)
	}
	return fmt.Sprintf("egress: upstream error for %s %s%s: %v", e.Method, e.Host, e.Path, e.Err)
}

func (e *UpstreamError) Unwrap() error { return e.Err }

// ParseError is raised when the egress YAML file is invalid. The error
// includes the file path and a description of the failure.
type ParseError struct {
	Path   string
	Reason string
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("egress: parse error in %s: %s", e.Path, e.Reason)
}
