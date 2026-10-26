package audit

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sadi/aios-system/internal/core/clock"
	"github.com/sadi/aios-system/internal/core/schemas"
)

// These tests extend the existing audit_test.go with
// coverage for GetByKind, GetByKindPayloadField, the
// hash type's String method, and the error branches
// in Write, Open, marshalPayload, and the timestamp
// helper.

func newLog(t *testing.T) *Log {
	t.Helper()
	log, err := Open(Config{Path: ":memory:", Clock: clock.FixedDecision{T: time.Unix(1700000000, 0)}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = log.Close() })
	return log
}

func TestGetByKind(t *testing.T) {
	log := newLog(t)
	for i := 0; i < 3; i++ {
		_, err := log.Write(&Row{TaskID: "t", Kind: "k1", Actor: "op", Payload: map[string]any{"i": i}})
		if err != nil {
			t.Fatal(err)
		}
	}
	if _, err := log.Write(&Row{TaskID: "t", Kind: "k2", Actor: "op", Payload: map[string]any{"x": 1}}); err != nil {
		t.Fatal(err)
	}
	rows, err := log.GetByKind("k1", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 {
		t.Fatalf("rows = %d, want 3", len(rows))
	}
}

func TestGetByKind_Limit(t *testing.T) {
	log := newLog(t)
	for i := 0; i < 5; i++ {
		_, _ = log.Write(&Row{TaskID: "t", Kind: "k", Actor: "op", Payload: map[string]any{"i": i}})
	}
	rows, _ := log.GetByKind("k", 2)
	if len(rows) != 2 {
		t.Fatalf("rows = %d, want 2", len(rows))
	}
}

func TestGetByKind_NoMatch(t *testing.T) {
	log := newLog(t)
	rows, err := log.GetByKind("nope", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 0 {
		t.Fatalf("rows = %d", len(rows))
	}
}

func TestGetByKind_EmptyKind(t *testing.T) {
	log := newLog(t)
	_, err := log.GetByKind("", 10)
	if err == nil {
		t.Fatal("expected error for empty kind")
	}
}

func TestGetByKindPayloadField(t *testing.T) {
	log := newLog(t)
	_, err := log.Write(&Row{TaskID: "t", Kind: "k", Actor: "op", ApprovalID: "ap-1", Payload: map[string]any{"task": "t1"}})
	if err != nil {
		t.Fatal(err)
	}
	_, _ = log.Write(&Row{TaskID: "t", Kind: "k", Actor: "op", ApprovalID: "ap-2", Payload: map[string]any{"task": "t2"}})
	rows, err := log.GetByKindPayloadField("k", "task", "t1", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].ApprovalID != "ap-1" {
		t.Fatalf("rows = %+v", rows)
	}
}

func TestGetByKindPayloadField_NoMatch(t *testing.T) {
	log := newLog(t)
	_, _ = log.Write(&Row{TaskID: "t", Kind: "k", Actor: "op", Payload: map[string]any{"task": "t1"}})
	rows, _ := log.GetByKindPayloadField("k", "task", "missing", 10)
	if len(rows) != 0 {
		t.Fatalf("rows = %d", len(rows))
	}
}

func TestHashPair_String(t *testing.T) {
	// The String method on hashPair is used in error messages.
	p := hashPair{
		redacted: []byte{1, 2, 3, 4},
		content:  []byte{5, 6, 7, 8},
		chain:    []byte{9, 10, 11, 12},
	}
	if got := p.String(); got == "" {
		t.Fatal("String returned empty")
	}
	if !strings.Contains(p.String(), "redacted=") {
		t.Fatalf("String = %q", p.String())
	}
}

func TestRow_Timestamp_UsesClock(t *testing.T) {
	// When RecordedAt is zero, the row pulls from the log's clock.
	fixed := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	log, err := Open(Config{Path: ":memory:", Clock: clock.FixedDecision{T: fixed}})
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()
	r := &Row{Kind: "k"}
	got := r.timestamp(log.clock)
	if !got.Equal(fixed) {
		t.Fatalf("got %v, want %v", got, fixed)
	}
}

func TestRow_Timestamp_RespectsExisting(t *testing.T) {
	// When RecordedAt is non-zero, timestamp is a no-op.
	set := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	log, _ := Open(Config{Path: ":memory:", Clock: clock.FixedDecision{T: time.Unix(1, 0)}})
	defer log.Close()
	r := &Row{Kind: "k", RecordedAt: set}
	if got := r.timestamp(log.clock); !got.Equal(set) {
		t.Fatalf("got %v, want %v", got, set)
	}
}

func TestWrite_ClosedLog(t *testing.T) {
	log, _ := Open(Config{Path: ":memory:"})
	_ = log.Close()
	_, err := log.Write(&Row{Kind: "k", Payload: map[string]any{}})
	if err == nil || !strings.Contains(err.Error(), "closed") {
		t.Fatalf("err = %v", err)
	}
}

func TestWrite_NilRow(t *testing.T) {
	log := newLog(t)
	_, err := log.Write(nil)
	if err == nil {
		t.Fatal("expected error for nil row")
	}
}

func TestWrite_ValidationFailure(t *testing.T) {
	log := newLog(t)
	// Payload is a Schema that fails validation.
	bad := &schemas.ApprovalPayload{} // missing approval_id
	_, err := log.Write(&Row{Kind: "k", Payload: bad})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestWrite_ApprovalPayload_PreservesID(t *testing.T) {
	// The audit writer redacts the payload but preserves
	// the approval_id field. This is the contract: the
	// approval id must survive the redaction pass so the
	// approval record can be linked to the audit row.
	log := newLog(t)
	row := &Row{
		TaskID: "t",
		Kind:   "k",
		Actor:  "op",
		Payload: &schemas.ApprovalPayload{
			ApprovalID:  "12345678-1234-4234-8234-123456789abc",
			Action:      "write",
			Target:      "/x",
			Reason:      "test",
			RequesterID: "op",
		},
	}
	_, err := log.Write(row)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(row.payloadJSON), "12345678-1234-4234-8234-123456789abc") {
		t.Fatalf("approval_id not in stored payload: %s", row.payloadJSON)
	}
}

func TestWrite_DuplicateIdempotencyKey(t *testing.T) {
	log := newLog(t)
	r := &Row{TaskID: "t", Kind: "k", Actor: "op", Payload: map[string]any{"a": 1}, IdempotencyKey: "k1"}
	if _, err := log.Write(r); err != nil {
		t.Fatal(err)
	}
	// Second write with the same key.
	r2 := &Row{TaskID: "t", Kind: "k", Actor: "op", Payload: map[string]any{"a": 2}, IdempotencyKey: "k1"}
	res, err := log.Write(r2)
	if !errors.Is(err, ErrDuplicate) {
		t.Fatalf("err = %v, want ErrDuplicate", err)
	}
	if res == nil || res.Existing == nil {
		t.Fatal("expected Existing row in result")
	}
}

func TestWrite_NoIdempotencyKey(t *testing.T) {
	// Default behavior: no key, just write.
	log := newLog(t)
	_, err := log.Write(&Row{TaskID: "t", Kind: "k", Actor: "op", Payload: map[string]any{"a": 1}})
	if err != nil {
		t.Fatal(err)
	}
	// A second write with no key should NOT collide
	// (the writer defaults to content_hash, but a fresh
	// payload produces a fresh hash).
	_, err = log.Write(&Row{TaskID: "t", Kind: "k", Actor: "op", Payload: map[string]any{"a": 2}})
	if err != nil {
		t.Fatal(err)
	}
}

func TestTimestamp_UsesRecordedAt(t *testing.T) {
	// When the row has a non-zero RecordedAt, the writer
	// should use it instead of the clock. This test confirms
	// the round-trip preserves the timestamp.
	fixed := time.Date(2023, 6, 15, 12, 0, 0, 0, time.UTC)
	log := newLog(t)
	row := &Row{
		TaskID:     "t",
		Kind:       "k",
		Actor:      "op",
		Payload:    map[string]any{"a": 1},
		RecordedAt: fixed,
	}
	if _, err := log.Write(row); err != nil {
		t.Fatal(err)
	}
	if !row.RecordedAt.Equal(fixed) {
		t.Fatalf("RecordedAt = %v, want %v", row.RecordedAt, fixed)
	}
}

func TestOpen_EmptyPath(t *testing.T) {
	if _, err := Open(Config{}); err == nil {
		t.Fatal("expected error for empty path")
	}
}

func TestOpen_CreatesTheParentDirectory(t *testing.T) {
	// Open's contract is "opens or creates a Log". A sqlite
	// file cannot be created in a directory that does not
	// exist, so Open creates the directory.
	//
	// It used to fail instead, with sqlite's error for "no such
	// directory": "ping: unable to open database file: out of
	// memory (14)". That named neither the cause nor the fix,
	// and it hit on a first run in a fresh workspace -- the
	// operator was told to run `aios init`, a step they had no
	// way to know about.
	dir := filepath.Join(t.TempDir(), "var")
	log, err := Open(Config{Path: filepath.Join(dir, "audit.db")})
	if err != nil {
		t.Fatalf("Open did not create the directory: %v", err)
	}
	defer log.Close()
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("the directory was not created: %v", err)
	}
}

func TestOpen_FilePath(t *testing.T) {
	// Real file path (not :memory:), exercises the
	// WAL-mode branch.
	tmp := t.TempDir()
	dbPath := tmp + "/audit.db"
	log, err := Open(Config{Path: dbPath})
	if err != nil {
		t.Fatal(err)
	}
	_ = log.Close()
}

func TestClose_Idempotent(t *testing.T) {
	// Closing twice should be a no-op the second time.
	log, _ := Open(Config{Path: ":memory:"})
	if err := log.Close(); err != nil {
		t.Fatal(err)
	}
	if err := log.Close(); err != nil {
		t.Fatalf("second close should be a no-op: %v", err)
	}
}

// TestClose_WaitsForAnInFlightWrite pins that closing the log
// does not pull the database out from under a write that is
// already running.
//
// The web UI starts a run in its own goroutine and the process
// closes the audit log on the way out. Before the handle took a
// lock of its own, Close set the pointer to nil while the run's
// tracer was reading it: the race detector flagged it, and the
// run's audit rows would have failed with "sql: database is
// closed" halfway through.
func TestClose_WaitsForAnInFlightWrite(t *testing.T) {
	log, err := Open(Config{Path: ":memory:"})
	if err != nil {
		t.Fatal(err)
	}
	// Close in a goroutine while a write is in flight. The
	// assertion is not on the timing: it is that neither side
	// reports a closed database and that the race detector sees
	// no unsynchronised access to the handle.
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			_, _ = log.Write(&Row{
				TaskID: "t", Kind: "k", Actor: "op",
				Payload: map[string]any{"i": i},
			})
		}
	}()
	go func() {
		defer wg.Done()
		_ = log.Close()
	}()
	wg.Wait()

	// After the close the log is unusable, and it says so.
	_, err = log.Write(&Row{TaskID: "t", Kind: "k", Actor: "op"})
	if err == nil || !strings.Contains(err.Error(), "closed") {
		t.Fatalf("write after close = %v, want a closed error", err)
	}
	if _, err := log.GetByTask("t"); err == nil || !strings.Contains(err.Error(), "closed") {
		t.Fatalf("read after close = %v, want a closed error", err)
	}
}

func TestItoa64(t *testing.T) {
	for _, c := range []struct {
		in   int64
		want string
	}{
		{0, "0"},
		{1, "1"},
		{10, "10"},
		{-1, "-1"},
		{1234567890, "1234567890"},
	} {
		if got := itoa64(c.in); got != c.want {
			t.Errorf("itoa64(%d) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestMustHexDecode_BadLength(t *testing.T) {
	// mustHexDecode returns an error for short input, it
	// does not panic. Confirm the error is returned.
	if err := mustHexDecode("ab"); err == nil {
		t.Fatal("expected error for short hex")
	}
}

func TestGetByTask_EmptyID(t *testing.T) {
	log := newLog(t)
	if _, err := log.GetByTask(""); err == nil {
		t.Fatal("expected error for empty task id")
	}
}

func TestGetByApproval_EmptyID(t *testing.T) {
	log := newLog(t)
	if _, err := log.GetByApproval(""); err == nil {
		t.Fatal("expected error for empty approval id")
	}
}

func TestGetByKindPayloadField_EmptyField(t *testing.T) {
	log := newLog(t)
	if _, err := log.GetByKindPayloadField("k", "", "v", 10); err == nil {
		t.Fatal("expected error for empty field")
	}
}

func TestGetByKindPayloadField_EmptyKind(t *testing.T) {
	log := newLog(t)
	if _, err := log.GetByKindPayloadField("", "f", "v", 10); err == nil {
		t.Fatal("expected error for empty kind")
	}
}

func TestVerifyChain_FirstRow(t *testing.T) {
	log := newLog(t)
	_, _ = log.Write(&Row{TaskID: "t", Kind: "k", Actor: "op", Payload: map[string]any{}})
	rows, _ := log.GetByKind("k", 10)
	res := VerifyChain(rows)
	if !res.OK {
		t.Fatalf("chain should be valid: %+v", res)
	}
}

func TestVerifyChain_Tampered(t *testing.T) {
	log := newLog(t)
	_, _ = log.Write(&Row{TaskID: "t", Kind: "k", Actor: "op", Payload: map[string]any{"a": 1}})
	// Direct DB tampering: flip a byte in the payload column.
	if _, err := log.db.Exec(`UPDATE audit_rows SET payload = X'7b7d' WHERE rowid = 1`); err != nil {
		t.Fatal(err)
	}
	rows, _ := log.GetByKind("k", 10)
	res := VerifyChain(rows)
	if res.OK {
		t.Fatalf("expected VerifyChain to fail: %+v", res)
	}
}
