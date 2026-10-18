package policy

import (
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"

	"github.com/sadi/aios-system/internal/core/audit"
	"github.com/sadi/aios-system/internal/core/clock"
	"github.com/sadi/aios-system/internal/core/schemas"
)

// Gateway is the policy gateway. It is safe for concurrent
// use by multiple goroutines because it is stateless after
// construction: the rule table is immutable and the audit log
// is the only shared mutable state, and the audit log takes
// its own lock.
type Gateway struct {
	rules []Rule
	hash  string
	audit *audit.Log
}

// New constructs a Gateway with the given rules and audit log.
// The rules slice is copied; mutating the caller's slice
// afterwards does not affect the gateway. The audit log may
// be nil, in which case Decide does not write an audit row
// (useful in tests that do not need persistence).
func New(rules []Rule, log *audit.Log) *Gateway {
	cp := make([]Rule, len(rules))
	copy(cp, rules)
	return &Gateway{rules: cp, hash: Hash(cp), audit: log}
}

// Rules returns a copy of the gateway's rule table. Mutating
// the returned slice does not affect the gateway.
func (g *Gateway) Rules() []Rule {
	cp := make([]Rule, len(g.rules))
	copy(cp, g.rules)
	return cp
}

// Hash returns the content hash of the rule table.
func (g *Gateway) Hash() string { return g.hash }

// Decide evaluates the tool call against the rule table and
// returns the decision. Per REQ-POL-FUN-004..006 the gateway
// uses first-match-wins and returns the default deny when no
// rule matches.
//
// For an `ask` action the gateway also writes a
// policy_decision audit row carrying the approval payload
// (REQ-POL-FUN-009, REQ-POL-FUN-010).
func (g *Gateway) Decide(c Call) (*Decision, error) {
	for i := range g.rules {
		if !g.rules[i].Match.Matches(c) {
			continue
		}
		d, err := g.decisionFor(&g.rules[i], c)
		if err != nil {
			return nil, err
		}
		if g.audit != nil {
			if err := g.writeAudit(c, d); err != nil {
				return nil, fmt.Errorf("policy: audit: %w", err)
			}
		}
		return d, nil
	}
	// Default deny (REQ-POL-FUN-006).
	d := &Decision{
		Action: ActionDeny,
		Reason: "no rule matched; default deny",
		RuleID: "",
	}
	if g.audit != nil {
		if err := g.writeAudit(c, d); err != nil {
			return nil, fmt.Errorf("policy: audit: %w", err)
		}
	}
	return d, nil
}

func (g *Gateway) decisionFor(r *Rule, c Call) (*Decision, error) {
	d := &Decision{
		Action: r.Action,
		Reason: r.Reason,
		RuleID: r.ID,
	}
	if r.Action == ActionAsk {
		// Substitute template values from the call.
		action, err := Substitute(r.Approval.Action, c)
		if err != nil {
			return nil, fmt.Errorf("policy: rule %q: approval.action: %w", r.ID, err)
		}
		resource, err := Substitute(r.Approval.Resource, c)
		if err != nil {
			return nil, fmt.Errorf("policy: rule %q: approval.resource: %w", r.ID, err)
		}
		reason, err := Substitute(r.Approval.Reason, c)
		if err != nil {
			return nil, fmt.Errorf("policy: rule %q: approval.reason: %w", r.ID, err)
		}
		bound, err := SubstituteMap(r.Approval.BoundTo, c)
		if err != nil {
			return nil, fmt.Errorf("policy: rule %q: approval.bound_to: %w", r.ID, err)
		}
		// approval_id is a fresh UUIDv4. The audit row records it
		// so aios-approvals can join.
		approvalID := uuid.NewString()
		d.Approval = &schemas.ApprovalPayload{
			ApprovalID:  approvalID,
			Action:      action,
			Target:      resource,
			RiskTier:    c.RiskTier,
			Reason:      reason,
			Scope:       flattenBoundTo(bound),
			RequesterID: c.Actor,
		}
	}
	return d, nil
}

// flattenBoundTo turns the bound_to map into the Scope string
// expected by schemas.ApprovalPayload. Format: "k=v;k=v".
//
// Keys are sorted so the resulting string is deterministic; the
// scope is part of the row's payload and therefore part of the
// audit content_hash.
func flattenBoundTo(bound map[string]string) string {
	if len(bound) == 0 {
		return ""
	}
	keys := make([]string, 0, len(bound))
	for k := range bound {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+bound[k])
	}
	return strings.Join(parts, ";")
}

// writeAudit persists the decision as a policy_decision audit
// row. The row's actor is the caller. The risk_tier is the
// call's tier. The approval_id is set on `ask` decisions.
func (g *Gateway) writeAudit(c Call, d *Decision) error {
	row := &audit.Row{
		TaskID:   c.TaskID,
		Kind:     "policy_decision",
		Actor:    c.Actor,
		RiskTier: c.RiskTier,
		Payload:  d.ToMap(),
	}
	if d.Approval != nil {
		row.ApprovalID = d.Approval.ApprovalID
	}
	_, err := g.audit.Write(row)
	return err
}

// compileTimeClockAssertion keeps the clock package import
// honest: if the clock.Decision interface is ever removed,
// this line will fail to compile, alerting us that callers of
// the gateway need to update their clock wiring.
var _ clock.Decision = clock.SystemDecision{}
