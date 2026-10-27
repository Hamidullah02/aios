// Package policy is the policy gateway for aios.
//
// The package exports three things:
//
//   - Rule: a typed rule with match conditions and an action.
//   - Decision: the gateway's verdict for a tool call.
//   - Gateway: a stateless evaluator that walks the rules in
//     first-match-wins order and writes a policy_decision audit
//     row per call.
//
// A rule table is loaded from a YAML file via Load. The YAML
// supports an `include` directive so a base file can pull in
// additional files. Every rule has an `id` and a `match` block.
// Match supports: tool (exact), tool_glob, path_prefix, path_glob,
// host_glob, tier (int), actor (exact), task (exact). Actions
// are `allow`, `deny`, or `ask`. `ask` rules carry an
// `approval` template whose `{task}`, `{tool}`, `{path}` etc.
// placeholders are filled in from the tool call at decision time.
package policy

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"

	"github.com/Hamidullah02/aios/aios-system/internal/core/schemas"
)

// Action is the gateway's verdict for a tool call.
type Action string

const (
	ActionAllow Action = "allow"
	ActionDeny  Action = "deny"
	ActionAsk   Action = "ask"
)

// IsValid returns true if a is one of the three known actions.
func (a Action) IsValid() bool {
	switch a {
	case ActionAllow, ActionDeny, ActionAsk:
		return true
	}
	return false
}

// Match is the set of match conditions on a rule. A condition
// is a conjunction: all set conditions must match for the
// rule to fire. Empty Match matches everything.
type Match struct {
	Tool       string `yaml:"tool,omitempty"        json:"tool,omitempty"`
	ToolGlob   string `yaml:"tool_glob,omitempty"   json:"tool_glob,omitempty"`
	PathPrefix string `yaml:"path_prefix,omitempty" json:"path_prefix,omitempty"`
	PathGlob   string `yaml:"path_glob,omitempty"   json:"path_glob,omitempty"`
	HostGlob   string `yaml:"host_glob,omitempty"   json:"host_glob,omitempty"`
	Tier       *int   `yaml:"tier,omitempty"        json:"tier,omitempty"`
	Actor      string `yaml:"actor,omitempty"       json:"actor,omitempty"`
	Task       string `yaml:"task,omitempty"        json:"task,omitempty"`
}

// HasAny returns true if at least one condition is set.
func (m Match) HasAny() bool {
	return m.Tool != "" || m.ToolGlob != "" || m.PathPrefix != "" ||
		m.PathGlob != "" || m.HostGlob != "" || m.Tier != nil ||
		m.Actor != "" || m.Task != ""
}

// ApprovalTemplate is the template for the approval payload
// produced by an `ask` action. The placeholders are filled in
// at decision time by Substitute. See template.go for the
// supported placeholder names.
type ApprovalTemplate struct {
	Action   string            `yaml:"action"             json:"action"`
	Resource string            `yaml:"resource"           json:"resource"`
	Reason   string            `yaml:"reason"             json:"reason"`
	BoundTo  map[string]string `yaml:"bound_to,omitempty" json:"bound_to,omitempty"`
}

// Rule is one policy rule. The ID is required and must be
// unique within a rule table. Action must be one of the
// three known actions. Approval is required when Action is
// `ask`; it is ignored for `allow` and `deny`.
type Rule struct {
	ID       string            `yaml:"id"               json:"id"`
	Match    Match             `yaml:"match"            json:"match"`
	Action   Action            `yaml:"action"           json:"action"`
	Approval *ApprovalTemplate `yaml:"approval,omitempty" json:"approval,omitempty"`
	Reason   string            `yaml:"reason,omitempty" json:"reason,omitempty"`
}

// Validate returns nil when the rule is well-formed. The
// returned error is a *RuleValidationError and includes the
// rule's ID and the failure reason (REQ-POL-ERR-002).
func (r *Rule) Validate() error {
	if r == nil {
		return &RuleValidationError{ID: "", Reason: "rule is nil"}
	}
	if r.ID == "" {
		return &RuleValidationError{ID: r.ID, Reason: "id is required"}
	}
	if !r.Action.IsValid() {
		return &RuleValidationError{
			ID:     r.ID,
			Reason: fmt.Sprintf("action must be one of allow, deny, ask; got %q", r.Action),
		}
	}
	if r.Action == ActionAsk && r.Approval == nil {
		return &RuleValidationError{
			ID:     r.ID,
			Reason: "ask action requires an approval template",
		}
	}
	if r.Action != ActionAsk && r.Approval != nil {
		return &RuleValidationError{
			ID:     r.ID,
			Reason: "approval template is only allowed for ask action",
		}
	}
	if r.Action == ActionAsk && r.Approval != nil {
		if r.Approval.Action == "" {
			return &RuleValidationError{
				ID:     r.ID,
				Reason: "approval.action is required",
			}
		}
		if r.Approval.Resource == "" {
			return &RuleValidationError{
				ID:     r.ID,
				Reason: "approval.resource is required",
			}
		}
		if r.Approval.Reason == "" {
			return &RuleValidationError{
				ID:     r.ID,
				Reason: "approval.reason is required",
			}
		}
	}
	return nil
}

// ValidateAll validates every rule and returns the first error.
// It also checks that all IDs are unique.
func ValidateAll(rules []Rule) error {
	seen := make(map[string]bool, len(rules))
	for i := range rules {
		if err := rules[i].Validate(); err != nil {
			return err
		}
		if seen[rules[i].ID] {
			return &RuleValidationError{
				ID:     rules[i].ID,
				Reason: "duplicate rule id",
			}
		}
		seen[rules[i].ID] = true
	}
	return nil
}

// Decision is the gateway's verdict for a tool call.
type Decision struct {
	Action   Action                   `json:"action"`
	Reason   string                   `json:"reason"`
	RuleID   string                   `json:"rule_id"`
	Approval *schemas.ApprovalPayload `json:"approval,omitempty"`
}

// ToMap converts the Decision to a map for audit payload.
// The map is the form persisted to the audit row's `payload`
// column. We do not write the *schemas.ApprovalPayload type
// directly because its MarshalJSON is the canonical form, and
// the audit log already canonicalizes payloads.
func (d *Decision) ToMap() map[string]any {
	out := map[string]any{
		"action":  string(d.Action),
		"reason":  d.Reason,
		"rule_id": d.RuleID,
	}
	if d.Approval != nil {
		out["approval"] = d.Approval
	}
	return out
}

// Rules is an ordered list of rules with a content hash. The
// hash is computed over the canonical JSON of every rule, in
// order, so two rule tables with the same rules in the same
// order produce the same hash. This is the "rule table is
// content addressed" determinism guarantee from the PRD.
type Rules struct {
	List []Rule `json:"rules"`
	Hash string `json:"hash"`
}

// Hash computes SHA256 over the canonical JSON of every rule
// in the list and returns the hex.
//
// The order matters: two rule tables that differ only in
// order hash differently. This is intentional — first-match-wins
// makes the order part of the policy.
func Hash(rules []Rule) string {
	type doc struct {
		Rules []Rule `json:"rules"`
	}
	d := doc{Rules: rules}
	raw, err := schemas.MarshalCanonical(d)
	if err != nil {
		// MarshalCanonical on a simple struct should never fail.
		// Return the empty hash to keep the caller total.
		return ""
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

// File represents a parsed policy file (after the loader
// resolves `include` directives). The Rules field is the
// merged list. Source is the original path.
type File struct {
	Source string `json:"source"`
	Rules  []Rule `json:"rules"`
}

// NormalizePath resolves a path against a base directory and
// returns a clean absolute path. Used by the loader to
// canonicalize include paths.
func NormalizePath(base, p string) (string, error) {
	if base == "" {
		return filepath.Abs(p)
	}
	if filepath.IsAbs(p) {
		return filepath.Clean(p), nil
	}
	return filepath.Clean(filepath.Join(base, p)), nil
}
