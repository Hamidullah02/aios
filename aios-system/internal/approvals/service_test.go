package approvals

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Hamidullah02/aios/aios-system/internal/core/audit"
	"github.com/Hamidullah02/aios/aios-system/internal/core/clock"
	"github.com/Hamidullah02/aios/aios-system/internal/core/schemas"
)

// newTestService returns a service backed by an in-memory audit
// log with a fixed decision clock. The log is closed by t.Cleanup.
func newTestService(t *testing.T) *Service {
	t.Helper()
	log, err := audit.Open(audit.Config{
		Path:  ":memory:",
		Clock: clock.FixedDecision{T: time.Unix(1700000000, 0)},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { log.Close() })
	return New(Config{Audit: log})
}

// validPayload returns a minimal-but-valid ApprovalPayload.
func validPayload() schemas.ApprovalPayload {
	return schemas.ApprovalPayload{
		Action: "edit", Target: "/tmp/x", RiskTier: 3,
		Reason: "needed", RequesterID: "agent:planner",
	}
}

func TestRequest_RejectsInvalidPayload(t *testing.T) {
	s := newTestService(t)
	_, err := s.Request(schemas.ApprovalPayload{}, "alice", RequestOptions{})
	if !errors.Is(err, ErrInvalidPayload) {
		t.Fatalf("err = %v, want ErrInvalidPayload", err)
	}
}

func TestRequest_RejectsEmptyActor(t *testing.T) {
	s := newTestService(t)
	_, err := s.Request(validPayload(), "", RequestOptions{})
	if !errors.Is(err, ErrInvalidPayload) {
		t.Fatalf("err = %v, want ErrInvalidPayload", err)
	}
}

func TestRequest_ReturnsID(t *testing.T) {
	s := newTestService(t)
	id, err := s.Request(validPayload(), "alice", RequestOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if id == "" {
		t.Fatal("id is empty")
	}
	got, err := s.Get(id)
	if err != nil {
		t.Fatal(err)
	}
	if got.State != StatePending {
		t.Fatalf("state = %q, want pending", got.State)
	}
	if got.ID != id {
		t.Fatalf("id roundtrip: %q vs %q", got.ID, id)
	}
}

func TestGrant_TransitionsToGranted(t *testing.T) {
	s := newTestService(t)
	id, _ := s.Request(validPayload(), "alice", RequestOptions{TTL: -1})
	if err := s.Grant(id, "bob", Context{}); err != nil {
		t.Fatal(err)
	}
	got, _ := s.Get(id)
	if got.State != StateGranted {
		t.Fatalf("state = %q, want granted", got.State)
	}
}

func TestDeny_TransitionsToDenied(t *testing.T) {
	s := newTestService(t)
	id, _ := s.Request(validPayload(), "alice", RequestOptions{TTL: -1})
	if err := s.Deny(id, "bob"); err != nil {
		t.Fatal(err)
	}
	got, _ := s.Get(id)
	if got.State != StateDenied {
		t.Fatalf("state = %q, want denied", got.State)
	}
}

func TestGrant_BindingMismatch(t *testing.T) {
	s := newTestService(t)
	p := validPayload()
	p.BoundTo = &schemas.BoundTo{Tool: "edit", Resource: "/tmp/x"}
	id, _ := s.Request(p, "alice", RequestOptions{TTL: -1})
	// Tool matches, resource matches, but the call-site task
	// differs from the bound-to task. BoundTo has no Task field
	// set, so it's a wildcard; this grant should succeed.
	if err := s.Grant(id, "bob", Context{Task: "t1", Tool: "edit", Actor: "bob", Resource: "/tmp/x"}); err != nil {
		t.Fatalf("first grant: %v", err)
	}
	// A second approval with a bound-to that does not match.
	id2, _ := s.Request(validPayload(), "alice", RequestOptions{TTL: -1})
	// mutate the in-memory copy via the snapshot
	got, _ := s.Get(id2)
	got.Payload.BoundTo = &schemas.BoundTo{Tool: "delete", Resource: "/etc"}
	// We can't mutate the snapshot's Payload through Get (it's a
	// value); we use the request path instead. Re-request with a
	// payload that has a binding the call-site will not match.
	badPayload := validPayload()
	badPayload.BoundTo = &schemas.BoundTo{Tool: "delete", Resource: "/etc"}
	id3, _ := s.Request(badPayload, "alice", RequestOptions{TTL: -1})
	if err := s.Grant(id3, "bob", Context{Tool: "edit", Resource: "/tmp/x"}); !errors.Is(err, ErrBindingMismatch) {
		t.Fatalf("err = %v, want ErrBindingMismatch", err)
	}
}

func TestGrant_BindingMatches(t *testing.T) {
	s := newTestService(t)
	p := validPayload()
	p.BoundTo = &schemas.BoundTo{Tool: "edit", Resource: "/tmp/x", Task: "t1"}
	id, _ := s.Request(p, "alice", RequestOptions{TTL: -1})
	if err := s.Grant(id, "bob", Context{Task: "t1", Tool: "edit", Resource: "/tmp/x", Actor: "bob"}); err != nil {
		t.Fatalf("grant: %v", err)
	}
}

func TestGrant_UnknownID(t *testing.T) {
	s := newTestService(t)
	if err := s.Grant("nope", "bob", Context{}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestGrant_TwiceFails(t *testing.T) {
	s := newTestService(t)
	id, _ := s.Request(validPayload(), "alice", RequestOptions{TTL: -1})
	if err := s.Grant(id, "bob", Context{}); err != nil {
		t.Fatal(err)
	}
	if err := s.Grant(id, "bob", Context{}); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("err = %v, want ErrInvalidTransition", err)
	}
}

func TestDeny_AfterGrant(t *testing.T) {
	s := newTestService(t)
	id, _ := s.Request(validPayload(), "alice", RequestOptions{TTL: -1})
	_ = s.Grant(id, "bob", Context{})
	if err := s.Deny(id, "carol"); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("err = %v, want ErrInvalidTransition", err)
	}
}

func TestWait_BlocksUntilGrant(t *testing.T) {
	s := newTestService(t)
	id, _ := s.Request(validPayload(), "alice", RequestOptions{TTL: -1})
	go func() {
		time.Sleep(20 * time.Millisecond)
		_ = s.Grant(id, "bob", Context{})
	}()
	got, err := s.Wait(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if got.State != StateGranted {
		t.Fatalf("state = %q, want granted", got.State)
	}
}

func TestWait_ContextCancel(t *testing.T) {
	s := newTestService(t)
	id, _ := s.Request(validPayload(), "alice", RequestOptions{TTL: -1})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := s.Wait(ctx, id)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}

func TestWait_UnknownID(t *testing.T) {
	s := newTestService(t)
	_, err := s.Wait(context.Background(), "nope")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestExpire_TTLFires(t *testing.T) {
	s := newTestService(t)
	id, _ := s.Request(validPayload(), "alice", RequestOptions{TTL: 30 * time.Millisecond})
	got, err := s.Wait(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if got.State != StateExpired {
		t.Fatalf("state = %q, want expired", got.State)
	}
}

func TestExpire_NoOpAfterGrant(t *testing.T) {
	s := newTestService(t)
	id, _ := s.Request(validPayload(), "alice", RequestOptions{TTL: 30 * time.Millisecond})
	_ = s.Grant(id, "bob", Context{})
	// Wait long enough for the timer to fire; the in-memory state
	// must remain granted.
	time.Sleep(80 * time.Millisecond)
	got, _ := s.Get(id)
	if got.State != StateGranted {
		t.Fatalf("state = %q, want granted (TTL fired after grant)", got.State)
	}
}

func TestAuditRowsWritten(t *testing.T) {
	// End-to-end: open a log, request + grant, then read back
	// the rows by approval id and verify kinds.
	log, err := audit.Open(audit.Config{
		Path:  ":memory:",
		Clock: clock.FixedDecision{T: time.Unix(1700000000, 0)},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { log.Close() })
	s := New(Config{Audit: log})
	id, _ := s.Request(validPayload(), "alice", RequestOptions{TTL: -1})
	_ = s.Grant(id, "bob", Context{})

	rows, err := log.GetByApproval(id)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("audit rows = %d, want 2 (requested + granted)", len(rows))
	}
	if rows[0].Kind != "approval_requested" || rows[1].Kind != "approval_granted" {
		t.Fatalf("kinds = [%s, %s]", rows[0].Kind, rows[1].Kind)
	}
}
