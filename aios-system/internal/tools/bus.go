package tools

import (
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/sadi/aios-system/internal/core/audit"
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
}

// Config wires the bus. Tools is the tool map; Policy and
// Audit are optional. Without a policy gateway, the bus
// allows every call. Without an audit log, the bus skips
// the audit-row write.
type Config struct {
	Tools  map[string]Tool
	Policy *policy.Gateway
	Audit  *audit.Log
}

// New returns a Bus. An empty cfg.Tools is allowed; every
// dispatch then returns ErrToolNotFound. cfg.Policy and
// cfg.Audit are optional.
func New(cfg Config) *Bus {
	return &Bus{
		tools:  cfg.Tools,
		policy: cfg.Policy,
		log:    cfg.Audit,
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
			_ = b.writeAudit(tool, args, ctx, "ask", policyReason, "", 0)
			return Result{OK: false, Summary: "ask: " + policyReason}, fmt.Errorf("tools: bus: %s requires approval: %s", name, policyReason)
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
