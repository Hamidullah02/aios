package sandbox

import (
	"fmt"
	"net/url"
	"os"
	"strings"
)

// NetworkPolicy describes the network the sandboxed child is
// allowed to reach.
//
// The policy exists because a resource limit is not a network
// limit. A child that can run `curl` can reach any host on any
// port, so "the command is gated by policy" and "the command can
// exfiltrate the repository" were the same statement. This type
// makes the second one expressible.
//
// Two modes:
//
//   - Proxy: the child is pointed at an HTTP proxy, and the proxy
//     decides what the child may reach. This is the enforceable
//     mode, and it is the only one that works for HTTPS.
//   - DenyAll: the child gets proxy variables pointing at a dead
//     address and NO_PROXY covering everything. Nothing leaves.
//
// What this type does NOT do is a firewall. It cannot stop a
// child that ignores the proxy variables, opens a raw socket, or
// speaks a protocol other than HTTP. The honest way to say so is
// in the type's contract: the enforcement depends on the child
// cooperating, and the only thing that makes it non-optional is
// the environment being empty of everything else.
//
// A child that can be trusted to ignore HTTP_PROXY can be
// trusted to ignore anything else too. That is why the enforcement
// mode removes the parent's proxy variables rather than adding to
// them: the child must not be able to fall back to a proxy it
// found in the environment.
type NetworkPolicy struct {
	// Mode is "proxy" or "deny". An empty Mode means no network
	// policy: the child gets an empty environment and whatever
	// the caller passed in Env. That is the historical behaviour,
	// kept so a caller that does not opt in is not silently
	// changed.
	Mode string
	// ProxyURL is the HTTP proxy the child must use, e.g.
	// "http://127.0.0.1:8080".
	//
	// It is used in both modes: in proxy mode it is the live
	// proxy, and in deny mode it is a black-hole address that
	// nothing listens on.
	ProxyURL string
	// NoProxy lists hosts the child may reach directly, bypassing
	// the proxy.
	//
	// It is empty in deny mode: nothing is reachable directly.
	// It exists for the legitimate case of a child that must
	// talk to a local service (a test database, a dev server)
	// that is not worth proxying.
	NoProxy []string
}

// Network modes.
const (
	// NetworkModeNone leaves the child's network untouched.
	NetworkModeNone = ""
	// NetworkModeProxy points the child at a proxy.
	NetworkModeProxy = "proxy"
	// NetworkModeDenyAll blocks the child's network.
	NetworkModeDenyAll = "deny"
)

// Validate checks the policy is well-formed.
//
// The rules are deliberately strict about the proxy URL: it must
// parse, and it must be an http or https URL. A socks5 URL would
// parse but would need a different client in the child, and a
// malformed URL would silently produce an environment that points
// nowhere -- which looks like "no network" rather than "a
// misconfiguration".
func (p *NetworkPolicy) Validate() error {
	if p == nil {
		return nil
	}
	switch p.Mode {
	case NetworkModeNone:
		return nil
	case NetworkModeDenyAll:
		// The deny mode does not need a proxy URL: it points
		// the child at a built-in black hole, so there is
		// nothing to validate.
		return nil
	case NetworkModeProxy:
	default:
		return &NetworkError{Field: "mode", Reason: "unknown mode " + p.Mode}
	}
	if strings.TrimSpace(p.ProxyURL) == "" {
		return &NetworkError{Field: "proxy_url", Reason: "is required in " + p.Mode + " mode"}
	}
	u, err := url.Parse(p.ProxyURL)
	if err != nil {
		return &NetworkError{Field: "proxy_url", Reason: err.Error()}
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return &NetworkError{Field: "proxy_url", Reason: "scheme must be http or https, got " + u.Scheme}
	}
	if u.Host == "" {
		return &NetworkError{Field: "proxy_url", Reason: "has no host"}
	}
	return nil
}

// NetworkError is raised when a network policy is not
// well-formed.
type NetworkError struct {
	Field  string
	Reason string
}

func (e *NetworkError) Error() string {
	return fmt.Sprintf("sandbox: network: %s: %s.", e.Field, e.Reason)
}

// Is reports whether target is also a *NetworkError.
func (e *NetworkError) Is(target error) bool {
	_, ok := target.(*NetworkError)
	return ok
}

// Apply returns the environment variables the child must run with,
// given the caller's environment.
//
// The returned map is the COMPLETE environment for the child: the
// caller's Env is overlaid with the proxy variables, and any
// proxy variable already present in the caller's Env is removed.
//
// That removal is the whole point. A child that can read
// HTTP_PROXY from the environment can choose to ignore it, and a
// second proxy variable pointing somewhere else is an escape
// hatch. So the policy rewrites the environment rather than
// adding to it.
//
// The child also gets NO_PROXY set to "*" in deny mode, which
// tells a client that uses the proxy variables to bypass the
// proxy for every host -- and since the proxy address is a
// black hole, bypassing it gets the child nowhere.
func (p *NetworkPolicy) Apply(env map[string]string) map[string]string {
	if p == nil || p.Mode == NetworkModeNone {
		return env
	}

	out := make(map[string]string, len(env)+4)
	for k, v := range env {
		// Drop every pre-existing proxy variable. The child's
		// network must be governed by this policy alone.
		if isProxyVar(k) {
			continue
		}
		out[k] = v
	}

	if p.Mode == NetworkModeDenyAll {
		// A black-hole proxy plus NO_PROXY="*" means a client
		// that honours the variables cannot reach anything:
		// it either tries the dead proxy or bypasses it into a
		// direct connection, and the direct connection is what
		// the sandbox's own isolation has to stop.
		out["HTTP_PROXY"] = blackHoleProxy
		out["HTTPS_PROXY"] = blackHoleProxy
		out["http_proxy"] = blackHoleProxy
		out["https_proxy"] = blackHoleProxy
		out["ALL_PROXY"] = blackHoleProxy
		out["all_proxy"] = blackHoleProxy
		out["NO_PROXY"] = "*"
		out["no_proxy"] = "*"
		return out
	}

	proxy := p.ProxyURL
	out["HTTP_PROXY"] = proxy
	out["HTTPS_PROXY"] = proxy
	out["http_proxy"] = proxy
	out["https_proxy"] = proxy
	// ALL_PROXY is the catch-all for clients that do not
	// distinguish http from https. Pointing it at the same proxy
	// is what makes a `curl` without a scheme, or a language
	// runtime with its own proxy logic, land in the same place.
	out["ALL_PROXY"] = proxy
	out["all_proxy"] = proxy
	if len(p.NoProxy) > 0 {
		noProxy := strings.Join(p.NoProxy, ",")
		out["NO_PROXY"] = noProxy
		out["no_proxy"] = noProxy
	} else {
		// An empty NO_PROXY means "no exceptions", which is
		// the safe default. "*" would mean the opposite.
		delete(out, "NO_PROXY")
		delete(out, "no_proxy")
	}
	return out
}

// blackHoleProxy is the address a denied child is pointed at.
//
// 127.0.0.1:1 is the conventional "nothing listens here" address:
// port 1 is reserved and never bound by a real service. A
// connection to it fails immediately with ECONNREFUSED rather than
// hanging, so a denied child fails fast instead of stalling until
// the timeout.
const blackHoleProxy = "http://127.0.0.1:1"

// isProxyVar reports whether a variable name is a proxy variable.
//
// The list is the set of variables that mainstream clients read:
// curl, wget, pip, npm, git, go, and most JVM and .NET runtimes
// all use one of these names.
func isProxyVar(name string) bool {
	switch strings.ToUpper(name) {
	case "HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY", "FTP_PROXY", "NO_PROXY":
		return true
	}
	return false
}

// EnvProxyURL returns the proxy URL from the environment, if any.
//
// It is the inverse of the policy's job: a caller that wants to
// know whether the process it is running in already has a proxy
// configured asks here.
func EnvProxyURL() string {
	for _, name := range []string{"HTTPS_PROXY", "https_proxy", "HTTP_PROXY", "http_proxy", "ALL_PROXY", "all_proxy"} {
		if v := strings.TrimSpace(os.Getenv(name)); v != "" {
			return v
		}
	}
	return ""
}
