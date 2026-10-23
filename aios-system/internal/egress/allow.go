package egress

import (
	"path/filepath"
	"strings"
)

// AllowList is the typed allow + deny list of host patterns.
//
// Per REQ-EGS-ALW-001..003 the list supports exact match and glob
// match. The list is loaded from a YAML file by LoadFile.
//
// Semantics:
//
//   - A request is allowed if the host matches at least one pattern in
//     `Allow` AND does not match any pattern in `Deny`.
//   - `Deny` wins over `Allow` (deny takes precedence; see the §12
//     risk register "The proxy has a bypass" mitigation: "The proxy
//     supports the deny list").
//   - Patterns are matched case-insensitively. Host names are not
//     case-sensitive in DNS; a tool calling `API.Example.com` and the
//     allow list `api.example.com` should agree.
//   - Patterns and the request host are both normalised at compare
//     time: the port is stripped (since a single host may be reached
//     on multiple ports; the allow list matches the hostname) and the
//     string is lower-cased.
//   - An empty allow list denies every host (deny by default).
type AllowList struct {
	allow []hostRule
	deny  []hostRule
}

// AllowCount returns the number of allow patterns in the list.
//
// It exists so an operator-facing summary can say how many hosts
// are reachable without exposing the patterns themselves.
func (a *AllowList) AllowCount() int {
	if a == nil {
		return 0
	}
	return len(a.allow)
}

// hostRule is a single pattern. The original string is kept for the
// hash and the error path; `lower` is the normalised comparison key.
type hostRule struct {
	original string
	lower    string
	isGlob   bool
}

// NewAllowList constructs an allow list from the given patterns.
// The slices are copied; mutating the caller's slices afterwards
// does not affect the list.
func NewAllowList(allow, deny []string) *AllowList {
	return &AllowList{
		allow: compileHostRules(allow),
		deny:  compileHostRules(deny),
	}
}

// compileHostRules normalises each pattern: trims whitespace, strips
// the port, lower-cases, and flags glob patterns.
func compileHostRules(in []string) []hostRule {
	if len(in) == 0 {
		return nil
	}
	out := make([]hostRule, 0, len(in))
	for _, p := range in {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		lower := normaliseHost(p)
		out = append(out, hostRule{
			original: p,
			lower:    lower,
			isGlob:   lower != "" && strings.ContainsAny(lower, "*?"),
		})
	}
	return out
}

// Allows reports whether host is permitted.
func (a *AllowList) Allows(host string) bool {
	if a == nil {
		return false
	}
	host = normaliseHost(host)
	if host == "" {
		return false
	}
	if matchesAny(a.deny, host) {
		return false
	}
	// An empty allow list denies everything (deny by default).
	if len(a.allow) == 0 {
		return false
	}
	return matchesAny(a.allow, host)
}

// matchesAny reports whether host matches any rule in rules. The
// caller is expected to have already normalised host.
func matchesAny(rules []hostRule, host string) bool {
	for _, r := range rules {
		if r.isGlob {
			ok, err := filepath.Match(r.lower, host)
			if err == nil && ok {
				return true
			}
			continue
		}
		if r.lower == host {
			return true
		}
	}
	return false
}

// normaliseHost strips a port and lower-cases. An empty input or
// whitespace-only input is returned as "".
func normaliseHost(h string) string {
	h = strings.TrimSpace(h)
	if h == "" {
		return ""
	}
	// If the host is an IPv6 literal it is wrapped in [...]. Strip
	// the brackets and the optional port. net.SplitHostPort does
	// the right thing for the bracketed form.
	if strings.HasPrefix(h, "[") {
		if host, _, err := splitHostPort(h); err == nil {
			return strings.ToLower(host)
		}
		return strings.ToLower(strings.Trim(h, "[]"))
	}
	if host, _, err := splitHostPort(h); err == nil {
		return strings.ToLower(host)
	}
	return strings.ToLower(h)
}

// Hash returns a content hash of the allow + deny rules in canonical
// order. Used to make the policy content-addressed (per the PRD
// determinism stance: "Policy is content addressed").
func (a *AllowList) Hash() string {
	if a == nil {
		return ""
	}
	allow := make([]string, len(a.allow))
	for i, r := range a.allow {
		allow[i] = r.original
	}
	deny := make([]string, len(a.deny))
	for i, r := range a.deny {
		deny[i] = r.original
	}
	return hashListPair(allow, deny)
}
