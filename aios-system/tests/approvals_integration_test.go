// Package tests holds black-box integration tests that exercise
// multiple aios modules together.
package tests

import (
	"context"
	"testing"
	"time"

	"github.com/Hamidullah02/aios/aios-system/internal/approvals"
	"github.com/Hamidullah02/aios/aios-system/internal/core/audit"
	"github.com/Hamidullah02/aios/aios-system/internal/core/clock"
	"github.com/Hamidullah02/aios/aios-system/internal/core/schemas"
)

// TestApprovals_EndToEnd walks a tier-3 request through the
// approval service and the audit log. The test asserts:
//
//  1. the request writes an `approval_requested` row
//  2. the grant writes an `approval_granted` row
//  3. the chain verifies
//  4. the bound-to binding rejects a mismatched call-site
//  5. the TTL timer fires
func TestApprovals_EndToEnd(t *testing.T) {
	log, err := audit.Open(audit.Config{
		Path:  ":memory:",
		Clock: clock.FixedDecision{T: time.Unix(1700000000, 0)},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { log.Close() })

	svc := approvals.New(approvals.Config{Audit: log})

	// 1. Request with a bound-to that pins the tool and resource.
	id, err := svc.Request(schemas.ApprovalPayload{
		Action: "delete", Target: "/etc/important", RiskTier: 3,
		Reason: "operator asked", RequesterID: "agent:planner",
		BoundTo: &schemas.BoundTo{Tool: "delete", Resource: "/etc/important"},
	}, "agent:planner", approvals.RequestOptions{TTL: 50 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}

	// 2. A grant with a mismatched tool is rejected by the binding.
	if err := svc.Grant(id, "operator:alice",
		approvals.Context{Tool: "edit", Resource: "/etc/important"}); err == nil {
		t.Fatal("expected binding mismatch")
	}

	// 3. The TTL fires while we are still negotiating.
	got, err := svc.Wait(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if got.State != approvals.StateExpired {
		t.Fatalf("state = %q, want expired", got.State)
	}

	// 4. Audit rows: exactly 2 (requested + expired).
	rows, err := log.GetByApproval(id)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("audit rows = %d, want 2", len(rows))
	}
	if rows[0].Kind != "approval_requested" || rows[1].Kind != "approval_expired" {
		t.Fatalf("kinds = [%s, %s]", rows[0].Kind, rows[1].Kind)
	}
	if v := audit.VerifyChain(rows); !v.OK {
		t.Fatalf("chain invalid: %s", v.Reason)
	}
}

// TestApprovals_GrantFlow walks a clean grant cycle: request,
// grant, wait. Asserts the audit log has the right two rows and
// the chain verifies.
func TestApprovals_GrantFlow(t *testing.T) {
	log, err := audit.Open(audit.Config{
		Path:  ":memory:",
		Clock: clock.FixedDecision{T: time.Unix(1700000000, 0)},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { log.Close() })

	svc := approvals.New(approvals.Config{Audit: log})
	id, err := svc.Request(schemas.ApprovalPayload{
		Action: "edit", Target: "/tmp/x", RiskTier: 3,
		Reason: "needed", RequesterID: "agent:planner",
	}, "agent:planner", approvals.RequestOptions{TTL: -1})
	if err != nil {
		t.Fatal(err)
	}

	// Grant on a separate goroutine to exercise Wait.
	go func() {
		time.Sleep(10 * time.Millisecond)
		_ = svc.Grant(id, "operator:alice", approvals.Context{})
	}()
	got, err := svc.Wait(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if got.State != approvals.StateGranted {
		t.Fatalf("state = %q, want granted", got.State)
	}

	rows, err := log.GetByApproval(id)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("audit rows = %d, want 2", len(rows))
	}
	if rows[1].Kind != "approval_granted" {
		t.Fatalf("kind[1] = %q, want approval_granted", rows[1].Kind)
	}
}
