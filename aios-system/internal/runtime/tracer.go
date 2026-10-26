package runtime

import (
	"github.com/sadi/aios-system/internal/core/audit"
	"github.com/sadi/aios-system/internal/core/schemas"
)

// Tracer records every decision the orchestrator makes
// (REQ-RT-DEC-001 / 002). It is a thin wrapper around the
// audit log: every Trace call writes a DecisionTrace audit
// row of kind `decision`.
type Tracer struct {
	log *audit.Log
}

// NewTracer returns a Tracer wired to the audit log.
func NewTracer(log *audit.Log) *Tracer {
	return &Tracer{log: log}
}

// Trace writes a decision audit row. The row carries the
// task id, the actor (e.g. "orchestrator"), the risk tier
// at the time of the decision, and a DecisionTrace payload
// (schemas.DecisionTrace).
func (t *Tracer) Trace(taskID, actor string, riskTier int, trace schemas.DecisionTrace) error {
	// The receiver itself can be nil: an orchestrator built
	// without an audit log has no tracer, and it still traces
	// decisions. A nil receiver means "no trace to write",
	// not a crash.
	if t == nil || t.log == nil {
		return nil
	}
	_, err := t.log.Write(&audit.Row{
		TaskID:   taskID,
		Kind:     "decision",
		Actor:    actor,
		RiskTier: riskTier,
		Payload:  &trace,
	})
	return err
}
