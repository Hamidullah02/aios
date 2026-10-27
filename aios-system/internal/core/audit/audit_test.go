package audit

import (
	"errors"
	"testing"
	"time"

	"github.com/Hamidullah02/aios/aios-system/internal/core/clock"
)

// newTestLog returns an in-memory Log with a fixed decision clock.
func newTestLog(t *testing.T) *Log {
	t.Helper()
	log, err := Open(Config{
		Path:  ":memory:",
		Clock: clock.FixedDecision{T: time.Unix(1700000000, 0)},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { log.Close() })
	return log
}

func TestWriteAndRead(t *testing.T) {
	log := newTestLog(t)
	res, err := log.Write(&Row{
		TaskID:   "t1",
		Kind:     "task_created",
		Actor:    "cli",
		RiskTier: 0,
		Payload:  map[string]any{"goal": "ship aios"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Row.SequenceNumber != 1 {
		t.Fatalf("seq: %d", res.Row.SequenceNumber)
	}
	if res.Row.ID == "" {
		t.Fatal("row_id should be assigned")
	}
	if len(res.Row.chainHash) != 32 {
		t.Fatalf("chain hash len: %d", len(res.Row.chainHash))
	}

	rows, err := log.GetByTask("t1")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows: %d", len(rows))
	}
	if rows[0].Kind != "task_created" {
		t.Fatalf("kind: %s", rows[0].Kind)
	}
}

func TestWriteIdempotent(t *testing.T) {
	log := newTestLog(t)
	row := &Row{
		TaskID:         "t1",
		Kind:           "task_created",
		Actor:          "cli",
		RiskTier:       0,
		IdempotencyKey: "fixed-key",
		Payload:        map[string]any{"goal": "ship"},
	}
	first, err := log.Write(row)
	if err != nil {
		t.Fatal(err)
	}
	second, err := log.Write(row)
	if !errors.Is(err, ErrDuplicate) {
		t.Fatalf("expected ErrDuplicate, got %v", err)
	}
	if second.Existing == nil {
		t.Fatal("existing should be populated")
	}
	if second.Existing.SequenceNumber != first.Row.SequenceNumber {
		t.Fatalf("sequence mismatch: %d vs %d",
			second.Existing.SequenceNumber, first.Row.SequenceNumber)
	}
}

func TestIdempotencyKeyDefaultsToContentHash(t *testing.T) {
	log := newTestLog(t)
	row := &Row{
		TaskID:   "t1",
		Kind:     "task_created",
		Actor:    "cli",
		RiskTier: 0,
		Payload:  map[string]any{"goal": "ship"},
	}
	res, err := log.Write(row)
	if err != nil {
		t.Fatal(err)
	}
	if res.Row.IdempotencyKey == "" {
		t.Fatal("idempotency_key should be set")
	}
	if len(res.Row.IdempotencyKey) != 64 {
		// SHA256 hex.
		t.Fatalf("idempotency_key len: %d (want 64)", len(res.Row.IdempotencyKey))
	}
}

func TestChainIsValid(t *testing.T) {
	log := newTestLog(t)
	for i := 0; i < 5; i++ {
		_, err := log.Write(&Row{
			TaskID:   "t1",
			Kind:     "decision",
			Actor:    "planner",
			RiskTier: 1,
			Payload:  map[string]any{"step": i},
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	rows, err := log.GetByTask("t1")
	if err != nil {
		t.Fatal(err)
	}
	v := VerifyChain(rows)
	if !v.OK {
		t.Fatalf("chain invalid: %s", v.Reason)
	}
}

func TestChainFailsIfPayloadMutated(t *testing.T) {
	log := newTestLog(t)
	_, err := log.Write(&Row{
		TaskID:   "t1",
		Kind:     "decision",
		Actor:    "planner",
		RiskTier: 1,
		Payload:  map[string]any{"step": "a"},
	})
	if err != nil {
		t.Fatal(err)
	}
	// Tamper with the payload in the DB.
	if _, err := log.db.Exec(`UPDATE audit_rows SET payload = ? WHERE sequence_number = 1`,
		[]byte(`{"step":"TAMPERED"}`)); err != nil {
		t.Fatal(err)
	}
	rows, err := log.GetByTask("t1")
	if err != nil {
		t.Fatal(err)
	}
	v := VerifyChain(rows)
	if v.OK {
		t.Fatal("expected verification to fail after tamper")
	}
	if v.FirstBad != 1 {
		t.Fatalf("first bad: %d", v.FirstBad)
	}
}

func TestRedactionStored(t *testing.T) {
	log := newTestLog(t)
	_, err := log.Write(&Row{
		TaskID:   "t1",
		Kind:     "tool_call",
		Actor:    "runtime",
		RiskTier: 2,
		Payload: map[string]any{
			"args": map[string]any{
				"creds": "api_key=abcdefghijklmnop1234",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	rows, err := log.GetByTask("t1")
	if err != nil {
		t.Fatal(err)
	}
	// The stored payload should not contain the secret.
	if got := string(rows[0].Payload); contains(got, "abcdefghijklmnop1234") {
		t.Fatalf("secret leaked into stored payload: %s", got)
	}
	if !contains(string(rows[0].Payload), "REDACTED:api_key:") {
		t.Fatalf("expected placeholder, got: %s", string(rows[0].Payload))
	}
}

func TestGetByApproval(t *testing.T) {
	log := newTestLog(t)
	_, err := log.Write(&Row{
		TaskID:     "t1",
		Kind:       "approval_requested",
		Actor:      "planner",
		RiskTier:   3,
		ApprovalID: "550e8400-e29b-41d4-a716-446655440000",
		Payload:    map[string]any{"action": "delete"},
	})
	if err != nil {
		t.Fatal(err)
	}
	rows, err := log.GetByApproval("550e8400-e29b-41d4-a716-446655440000")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows: %d", len(rows))
	}
}

func TestGetByTask_Empty(t *testing.T) {
	log := newTestLog(t)
	rows, err := log.GetByTask("nope")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 0 {
		t.Fatalf("rows: %d", len(rows))
	}
}

func TestWriteValidationErrors(t *testing.T) {
	log := newTestLog(t)
	cases := []struct {
		name string
		row  *Row
	}{
		{"empty task", &Row{Kind: "x", Actor: "a", RiskTier: 0}},
		{"empty kind", &Row{TaskID: "t", Actor: "a", RiskTier: 0}},
		{"empty actor", &Row{TaskID: "t", Kind: "x", RiskTier: 0}},
		{"bad tier", &Row{TaskID: "t", Kind: "x", Actor: "a", RiskTier: 7}},
	}
	for _, c := range cases {
		_, err := log.Write(c.row)
		if err == nil {
			t.Fatalf("%s: expected error", c.name)
		}
	}
}

func TestCloseRejectsWrites(t *testing.T) {
	log, err := Open(Config{Path: ":memory:"})
	if err != nil {
		t.Fatal(err)
	}
	if err := log.Close(); err != nil {
		t.Fatal(err)
	}
	_, err = log.Write(&Row{TaskID: "t", Kind: "x", Actor: "a", RiskTier: 0})
	if err == nil {
		t.Fatal("expected error after close")
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
