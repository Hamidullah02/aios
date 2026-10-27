package runtime

import (
	"testing"
	"time"

	"github.com/Hamidullah02/aios/aios-system/internal/core/audit"
	"github.com/Hamidullah02/aios/aios-system/internal/core/clock"
	"github.com/Hamidullah02/aios/aios-system/internal/core/schemas"
)

func TestTracer_TraceWritesRow(t *testing.T) {
	log, err := audit.Open(audit.Config{
		Path:  ":memory:",
		Clock: clock.FixedDecision{T: time.Unix(1700000000, 0)},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { log.Close() })

	tr := NewTracer(log)
	if err := tr.Trace("t1", "orchestrator", 0, schemas.DecisionTrace{
		StepID: "st1", Chosen: "pass", Basis: "verifier=pass", RiskTier: 0,
	}); err != nil {
		t.Fatal(err)
	}
	rows, _ := log.GetByTask("t1")
	if len(rows) != 1 || rows[0].Kind != "decision" {
		t.Fatalf("rows = %+v", rows)
	}
}

func TestTracer_NilLogIsNoOp(t *testing.T) {
	tr := NewTracer(nil)
	if err := tr.Trace("t1", "orch", 0, schemas.DecisionTrace{}); err != nil {
		t.Fatalf("nil log should be no-op: %v", err)
	}
}
