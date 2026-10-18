package egress

import (
	"net/http"
	"strings"
)

// HeaderRule is a single header-rewriting rule.
//
// The shape is shared by the `remove` and `add` lists in the YAML
// file. For remove, Value is ignored. For add, Value is the value
// to set on every request.
//
// Header names are matched case-insensitively (Go's http.Header is
// canonicalised so the round-trip preserves case for the canonical
// key while matching user-supplied rules).
type HeaderRule struct {
	Name  string
	Value string
}

// HeaderRules is the typed remove + add rule set.
//
// Per REQ-EGS-HDR-001 the rules support remove and add. The proxy
// applies remove first, then add, so a rule can replace a header
// by removing it and adding a new one with the same name.
//
// Per REQ-EGS-HDR-002 the proxy also supports the rewriting of
// `Authorization` and `User-Agent`. Those are just two common
// special cases of the general remove + add mechanism; the proxy
// does not have a separate code path for them.
type HeaderRules struct {
	Remove []HeaderRule
	Add    []HeaderRule
}

// NewHeaderRules constructs a HeaderRules from the two rule slices.
// The slices are copied; mutating the caller's slices afterwards
// does not affect the rules.
func NewHeaderRules(remove, add []HeaderRule) *HeaderRules {
	return &HeaderRules{
		Remove: copyRules(remove),
		Add:    copyRules(add),
	}
}

func copyRules(in []HeaderRule) []HeaderRule {
	if in == nil {
		return nil
	}
	out := make([]HeaderRule, len(in))
	copy(out, in)
	return out
}

// ApplyToRequest mutates the request's headers in place: removes
// every rule in Remove, then adds every rule in Add. The same rule
// can both remove and add (the order above makes the add win).
func (h *HeaderRules) ApplyToRequest(hdr http.Header) {
	if h == nil {
		return
	}
	for _, r := range h.Remove {
		hdr.Del(r.Name)
	}
	for _, r := range h.Add {
		hdr.Set(r.Name, r.Value)
	}
}

// ApplyToResponse is the same as ApplyToRequest but acts on a
// response header set. It exists so callers do not have to know
// that the operation is the same in Go.
func (h *HeaderRules) ApplyToResponse(hdr http.Header) {
	h.ApplyToRequest(hdr)
}

// Hash returns a content hash of the rule set, used to make the
// header policy content-addressed.
func (h *HeaderRules) Hash() string {
	if h == nil {
		return ""
	}
	return hashHeaderRules(h.Remove, h.Add)
}

// validateHeaderName returns the canonical form of a header name
// after a light sanity check. An empty name is rejected.
//
// The check is not the full RFC 7230 token validation; it catches
// the practical cases (empty, whitespace, leading/trailing space).
// Go's http package will reject malformed names at Set/Del time.
func validateHeaderName(name string) (string, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "", &ParseError{Reason: "header name is empty"}
	}
	if trimmed != name {
		return "", &ParseError{Reason: "header name has surrounding whitespace: " + name}
	}
	return trimmed, nil
}
