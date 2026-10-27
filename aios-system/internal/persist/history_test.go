package persist

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Hamidullah02/aios/aios-system/internal/core/audit"
	"github.com/Hamidullah02/aios/aios-system/internal/core/clock"
)

// newTestHistory constructs a SessionHistory backed by
// an in-memory audit log. The function returns the
// history and a cleanup function the caller must defer.
func newTestHistory(t *testing.T) (*SessionHistory, func()) {
	t.Helper()
	fixed := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	dir := t.TempDir()
	log, err := audit.Open(audit.Config{
		Path:  filepath.Join(dir, "audit.db"),
		Clock: clock.FixedDecision{T: fixed},
	})
	if err != nil {
		t.Fatalf("audit.Open: %v", err)
	}
	h, err := NewSessionHistory(SessionHistoryConfig{
		Log:   log,
		Now:   func() time.Time { return fixed },
		Actor: "tester",
	})
	if err != nil {
		log.Close()
		t.Fatalf("NewSessionHistory: %v", err)
	}
	return h, func() { log.Close() }
}

// TestSessionHistoryAddList covers the round-trip add +
// list path. The events are returned in audit-log
// order.
func TestSessionHistoryAddList(t *testing.T) {
	h, cleanup := newTestHistory(t)
	defer cleanup()

	events := []HistoryEvent{
		{SessionID: "s1", TaskID: "t1", Action: "started"},
		{SessionID: "s1", TaskID: "t1", Action: "tool_call"},
		{SessionID: "s1", TaskID: "t2", Action: "completed"},
		{SessionID: "s2", TaskID: "t3", Action: "started"},
	}
	for _, ev := range events {
		if err := h.Add(ev); err != nil {
			t.Fatalf("Add: %v", err)
		}
	}
	got, err := h.List("s1")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 events for s1, got %d", len(got))
	}
	// The order is the audit-log order: by sequence
	// number. The events are in insertion order.
	wantActions := []string{"started", "tool_call", "completed"}
	for i, ev := range got {
		if ev.Action != wantActions[i] {
			t.Fatalf("event %d: expected %q, got %q", i, wantActions[i], ev.Action)
		}
	}
}

// TestSessionHistoryAddMissingFields covers the
// validation branch. The history rejects events with
// empty required fields.
func TestSessionHistoryAddMissingFields(t *testing.T) {
	h, cleanup := newTestHistory(t)
	defer cleanup()

	cases := []HistoryEvent{
		{TaskID: "t", Action: "a"},
		{SessionID: "s", Action: "a"},
		{SessionID: "s", TaskID: "t"},
	}
	for i, ev := range cases {
		if err := h.Add(ev); err == nil {
			t.Fatalf("case %d: expected error", i)
		}
	}
}

// TestSessionHistoryAddSetsActor covers the default-
// actor branch. When the caller leaves Actor empty, the
// history fills it from the constructor.
func TestSessionHistoryAddSetsActor(t *testing.T) {
	h, cleanup := newTestHistory(t)
	defer cleanup()

	if err := h.Add(HistoryEvent{SessionID: "s", TaskID: "t", Action: "a"}); err != nil {
		t.Fatalf("Add: %v", err)
	}
	got, err := h.List("s")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 event, got %d", len(got))
	}
	if got[0].Actor != "tester" {
		t.Fatalf("expected actor 'tester', got %q", got[0].Actor)
	}
}

// TestSessionHistoryAddSetsTimestamp covers the
// default-timestamp branch.
func TestSessionHistoryAddSetsTimestamp(t *testing.T) {
	h, cleanup := newTestHistory(t)
	defer cleanup()

	if err := h.Add(HistoryEvent{SessionID: "s", TaskID: "t", Action: "a"}); err != nil {
		t.Fatalf("Add: %v", err)
	}
	got, err := h.List("s")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if got[0].Timestamp == "" {
		t.Fatalf("Timestamp should be set")
	}
}

// TestSessionHistoryListEmpty covers the empty-session
// branch. The function returns an empty slice.
func TestSessionHistoryListEmpty(t *testing.T) {
	h, cleanup := newTestHistory(t)
	defer cleanup()

	got, err := h.List("missing")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty, got %+v", got)
	}
}

// TestSessionHistoryListEmptyID covers the empty-id
// branch.
func TestSessionHistoryListEmptyID(t *testing.T) {
	h, cleanup := newTestHistory(t)
	defer cleanup()

	_, err := h.List("")
	if err == nil {
		t.Fatalf("expected error")
	}
}

// TestSessionHistorySearch covers the search branch.
// The function returns events whose stringified payload
// matches the query.
func TestSessionHistorySearch(t *testing.T) {
	h, cleanup := newTestHistory(t)
	defer cleanup()

	if err := h.Add(HistoryEvent{SessionID: "s1", TaskID: "t1", Action: "started"}); err != nil {
		t.Fatalf("Add 1: %v", err)
	}
	if err := h.Add(HistoryEvent{SessionID: "s2", TaskID: "t2", Action: "completed"}); err != nil {
		t.Fatalf("Add 2: %v", err)
	}
	if err := h.Add(HistoryEvent{SessionID: "s3", TaskID: "t3", Action: "tool_call", Metadata: map[string]any{"tool": "read_file"}}); err != nil {
		t.Fatalf("Add 3: %v", err)
	}

	// Search by action.
	got, err := h.Search("tool_call")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(got) != 1 || got[0].Action != "tool_call" {
		t.Fatalf("expected 1 tool_call, got %+v", got)
	}
	// Search by metadata.
	got, err = h.Search("read_file")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 metadata match, got %+v", got)
	}
	// Search by session_id.
	got, err = h.Search("s2")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(got) != 1 || got[0].SessionID != "s2" {
		t.Fatalf("expected 1 s2 match, got %+v", got)
	}
	// Case-insensitive.
	got, err = h.Search("S1")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 case-insensitive match, got %+v", got)
	}
}

// TestSessionHistorySearchEmpty covers the empty-query
// branch.
func TestSessionHistorySearchEmpty(t *testing.T) {
	h, cleanup := newTestHistory(t)
	defer cleanup()

	_, err := h.Search("")
	if err == nil {
		t.Fatalf("expected error")
	}
}

// TestSessionHistorySearchNoMatch covers the no-match
// branch.
func TestSessionHistorySearchNoMatch(t *testing.T) {
	h, cleanup := newTestHistory(t)
	defer cleanup()

	if err := h.Add(HistoryEvent{SessionID: "s", TaskID: "t", Action: "a"}); err != nil {
		t.Fatalf("Add: %v", err)
	}
	got, err := h.Search("nope")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty, got %+v", got)
	}
}

// TestSessionHistoryNilLog covers the constructor's
// nil-log branch.
func TestSessionHistoryNilLog(t *testing.T) {
	_, err := NewSessionHistory(SessionHistoryConfig{})
	if err == nil {
		t.Fatalf("expected error for nil log")
	}
}

// TestHistoryErrorMessage covers the error format.
func TestHistoryErrorMessage(t *testing.T) {
	err := (&HistoryError{Op: "add", Reason: "boom"}).Error()
	if !strings.HasPrefix(err, "persist: history:") {
		t.Fatalf("unexpected prefix: %q", err)
	}
	if !strings.HasSuffix(err, ".") {
		t.Fatalf("missing trailing period: %q", err)
	}
}

// TestHistoryKindConstant covers the exported kind
// constant. The value is the audit-log kind the
// history uses.
func TestHistoryKindConstant(t *testing.T) {
	if HistoryKind != "session_event" {
		t.Fatalf("unexpected kind: %q", HistoryKind)
	}
}
