package tools

import (
	"fmt"
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
	var policyReason string
	if b.policy != nil {
		decision, _ := b.policy.Decide(policy.NewCall(name, args, ctx.Actor, ctx.TaskID))
		if decision != nil {
			policyReason = decision.Reason
		}
		switch decision.Action {
		case policy.ActionDeny:
			_ = b.writeAudit(tool, args, ctx, "deny", decision.Reason, "", 0)
			return Result{OK: false, Summary: "denied: " + decision.Reason}, fmt.Errorf("tools: bus: %s denied by policy: %s", name, decision.Reason)
		case policy.ActionAsk:
			_ = b.writeAudit(tool, args, ctx, "ask", decision.Reason, "", 0)
			return Result{OK: false, Summary: "ask: " + decision.Reason}, fmt.Errorf("tools: bus: %s requires approval: %s", name, decision.Reason)
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
