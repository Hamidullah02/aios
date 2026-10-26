package tools

import (
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/sadi/aios-system/internal/core/audit"
	"github.com/sadi/aios-system/internal/core/schemas"
	"github.com/sadi/aios-system/internal/policy"
)

// Bus dispatches tool calls (REQ-TOL-BUS-001 to 005).
//
// The bus is a small wrapper: it looks up the tool by name,
// calls the policy gateway, runs the tool, and writes the
// audit row. The bus is the only place in aios-tools that
// touches the policy gateway or the audit log; the tools
// themselves are policy-free and audit-free.
type Bus struct {
	tools  map[string]Tool
	policy *policy.Gateway
	log    *audit.Log
	// grants, when set, answers "has a human already approved
	// THIS call?". The bus consults it before rejecting an
	// `ask` decision.
	//
	// This was the missing half of the approval feature. The
	// bus rejected every `ask` unconditionally, so a human
	// answering "yes" changed nothing: the orchestrator
	// retried the task, the bus returned the same `ask`, and
	// the operator was asked about the identical call again.
	// A gate you cannot open is not a gate, it is a wall --
	// and the retry loop turned it into a double prompt that
	// ended in a spurious denial.
	grants ApprovalGrantChecker
}

// ApprovalGrantChecker reports whether an operator has
// already approved the call described by the policy
// decision.
//
// The decision (not the raw args) is passed because the
// approval id in it is the canonical handle the approvals
// service issued; the args do not carry it.
type ApprovalGrantChecker interface {
	// IsApproved reports whether the approval described by d
	// has been granted for this specific call.
	IsApproved(d *policy.Decision) bool
}

// Config wires the bus. Tools is the tool map; Policy and
// Audit are optional. Without a policy gateway, the bus
// allows every call. Without an audit log, the bus skips
// the audit-row write.
type Config struct {
	Tools  map[string]Tool
	Policy *policy.Gateway
	Audit  *audit.Log
	// Grants is optional. When nil, an `ask` decision is
	// rejected exactly as before, so existing callers and
	// tests keep their current behaviour.
	Grants ApprovalGrantChecker
}

// New returns a Bus. An empty cfg.Tools is allowed; every
// dispatch then returns ErrToolNotFound. cfg.Policy and
// cfg.Audit are optional.
func New(cfg Config) *Bus {
	return &Bus{
		tools:  cfg.Tools,
		policy: cfg.Policy,
		log:    cfg.Audit,
		grants: cfg.Grants,
	}
}

// Register adds a tool to the bus. The bus is normally
// immutable after construction; this method exists for
// the deferred-wiring case where a dependency is built
// after the bus (e.g. verify.SelfCheckTool needs a
// model router that is built only by the `fix`
// subcommand). The method is a single-line helper; it
// is intentionally not part of the public doc-set
// surface.
func (b *Bus) Register(name string, tool Tool) {
	if b.tools == nil {
		b.tools = make(map[string]Tool)
	}
	b.tools[name] = tool
}

// Names returns the sorted names of every registered
// tool.
//
// The method exists so the CLI can report the live tool
// set (`aios tools`) and so a wiring regression is
// testable: before v0.38.0 the edit / git / search /
// image tools were implemented but never registered,
// which no test could catch because nothing could ask
// the bus what it had.
//
// Names returns a fresh slice; the caller may sort or
// mutate it without affecting the bus.
func (b *Bus) Names() []string {
	out := make([]string, 0, len(b.tools))
	for name := range b.tools {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// Spec is one tool's self-description.
//
// It is the same shape the planner consumes, but it lives
// here rather than in the planner package so the bus can
// report it without importing the planner.
type Spec struct {
	// Name is the tool's canonical name.
	Name string
	// Description is the human-readable summary the model
	// reads when it chooses a tool.
	Description string
	// InputSchema is the raw JSON Schema for the tool's
	// arguments.
	InputSchema string
	// Risk is the tool's risk tier, 0 (read-only) to 3.
	Risk int
}

// Specs returns every registered tool's self-description,
// keyed by the tool's canonical name.
//
// The bus is the one place that knows which tools exist, so
// the planner's view of the tool set is derived from here
// rather than maintained by hand. Before this method the
// planner had a hand-written table of five tools while the
// bus carried twenty-five, so twenty of them could not be
// named in a plan however well they worked.
//
// A tool that does not implement tools.Spec still appears,
// with a placeholder description and an empty schema. It is
// better to tell the model that a tool's arguments are
// undocumented than to hide the tool: the model can then
// choose not to use it, whereas a hidden tool is a tool the
// operator believes exists and cannot reach.
func (b *Bus) Specs() map[string]Spec {
	out := make(map[string]Spec, len(b.tools))
	for name, tool := range b.tools {
		description, schema := Describe(tool)
		out[name] = Spec{
			Name:        name,
			Description: description,
			InputSchema: schema,
			Risk:        Tier(tool),
		}
	}
	return out
}

// Dispatch runs the tool call. The bus returns a Result and
// an error. A successful call has OK=true and a nil error;
// a policy-deny has OK=false and ErrPolicyDeny wrapped in
// the error; a tool error has OK=false and the tool's
// error; a missing tool has OK=false and ErrToolNotFound.
func (b *Bus) Dispatch(name string, args map[string]any, ctx Context) (Result, error) {
	tool, ok := b.tools[name]
	if !ok {
		return Result{OK: false, Summary: "tool not found"}, &ToolNotFoundError{Name: name}
	}

	// Policy check. The bus uses a "deny" outcome to short-
	// circuit; "ask" surfaces to the operator (the doc set
	// expects the worker to handle approvals; the bus
	// rejects `ask` because it cannot wait synchronously).
	//
	// v0.38.0: the gateway is default-deny
	// (REQ-POL-FUN-004) and always returns a decision on
	// success, but Decide also returns (nil, err) when a
	// rule's approval template is malformed. The old code
	// read decision.Action unconditionally, so that error
	// path panicked with a nil dereference instead of
	// surfacing the error. A nil decision now falls
	// through to the tool, matching how a nil policy is
	// treated.
	// The reason the gateway gave for an allow. Empty when
	// there is no policy or no decision; the audit row
	// carries it either way.
	var policyReason string
	if b.policy != nil {
		decision, policyErr := b.policy.Decide(policy.NewCall(name, args, ctx.Actor, ctx.TaskID))
		if policyErr != nil {
			_ = b.writeAudit(tool, args, ctx, "deny", policyErr.Error(), "", 0)
			return Result{OK: false, Summary: "policy error"}, fmt.Errorf("tools: bus: policy: %w", policyErr)
		}
		if decision != nil {
			policyReason = decision.Reason
		}
		switch {
		case decision == nil:
			// No decision: fall through to the tool.
		case decision.Action == policy.ActionDeny:
			_ = b.writeAudit(tool, args, ctx, "deny", policyReason, "", 0)
			return Result{OK: false, Summary: "denied: " + policyReason}, fmt.Errorf("tools: bus: %s denied by policy: %s", name, policyReason)
		case decision.Action == policy.ActionAsk:
			// An `ask` means "a human must decide". If
			// one already has -- for this exact call --
			// the gate is open and the call proceeds.
			//
			// Without this check the approval feature could
			// not work end to end: the operator says yes,
			// the task retries, and the bus returns the
			// same `ask` for the same arguments. The
			// operator is prompted twice for one action
			// and the run then fails on a denial they
			// never gave.
			//
			// The check is deliberately keyed to the
			// decision's approval id, not to the tool
			// name, so a grant for one command never
			// authorises a different one.
			if b.grants != nil && b.grants.IsApproved(decision) {
				policyReason = decision.Reason + " (approved)"
				break
			}
			_ = b.writeAudit(tool, args, ctx, "ask", policyReason, "", 0)
			// The approval payload rides along on the
			// error so the operator is shown what they
			// are approving. It used to be discarded,
			// which left the prompt with nothing to
			// display but an id.
			return Result{OK: false, Summary: "ask: " + policyReason},
				&ApprovalRequiredError{
					Tool:       name,
					Reason:     policyReason,
					ApprovalID: approvalIDOf(decision),
					Approval:   decision.Approval,
				}
		}
	}

	// Run the tool.
	start := time.Now()
	result, err := tool.Run(args, ctx)
	dur := time.Since(start)

	// Audit row (success or failure). The audit failure
	// is reported as an error wrapping the tool's error,
	// because the row is required for every call.
	auditErr := b.writeAudit(tool, args, ctx, "allow", policyReason, result.Summary, dur)

	if err != nil {
		if auditErr != nil {
			return result, fmt.Errorf("%w (audit: %v)", err, auditErr)
		}
		return result, err
	}
	if auditErr != nil {
		return result, fmt.Errorf("tools: bus: audit write: %w", auditErr)
	}
	return result, nil
}

// writeAudit writes a tool_call row. The Decision column
// carries the policy verdict (allow/deny/ask). The Result
// column carries the tool's summary.
func (b *Bus) writeAudit(tool Tool, args map[string]any, ctx Context, decision, reason, summary string, dur time.Duration) error {
	if b.log == nil {
		return nil
	}
	row := &audit.Row{
		TaskID:   ctx.TaskID,
		Kind:     "tool_call",
		Actor:    ctx.Actor,
		RiskTier: riskFromArgs(args),
		Payload: map[string]any{
			"call_id":     uuid.NewString(),
			"tool":        tool.Name(),
			"args":        args,
			"decision":    decision,
			"reason":      reason,
			"summary":     summary,
			"duration_ms": dur.Milliseconds(),
		},
	}
	_, err := b.log.Write(row)
	return err
}

// ApprovalRequiredError is returned when policy says `ask`
// and no operator has approved the call yet.
//
// It is a typed error, not a formatted string, because the
// caller needs the payload: the approval id and the
// action/target an operator is being asked to authorize. The
// old `fmt.Errorf("...requires approval: %s")` string forced
// the worker to recover those details with
// strings.Contains and then throw them away, which is how the
// prompt ended up showing an operator nothing about the
// command it wanted them to run.
type ApprovalRequiredError struct {
	Tool       string
	Reason     string
	ApprovalID string
	Approval   *schemas.ApprovalPayload
}

func (e *ApprovalRequiredError) Error() string {
	return fmt.Sprintf("tools: bus: %s requires approval: %s", e.Tool, e.Reason)
}

// approvalIDOf returns the canonical approval id from a
// decision, or "" when the rule carried no approval.
func approvalIDOf(d *policy.Decision) string {
	if d == nil || d.Approval == nil {
		return ""
	}
	return d.Approval.ApprovalID
}

// riskFromArgs returns the risk tier named in args (1 by
// default). The worker fills this in.
func riskFromArgs(args map[string]any) int {
	if v, ok := toInt(args["risk_tier"]); ok == nil {
		if v < 0 {
			return 0
		}
		if v > 3 {
			return 3
		}
		return v
	}
	return 1
}
