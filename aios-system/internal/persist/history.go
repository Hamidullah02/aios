package persist

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Hamidullah02/aios/aios-system/internal/core/audit"
	"github.com/Hamidullah02/aios/aios-system/internal/core/clock"
)

// HistoryEvent is the typed payload of one history
// event. The struct is the JSON the history writes to
// the audit log; List and Search return a slice of
// HistoryEvent.
//
// The field set is intentionally small. The event has
// the session id, the task id, the action, the actor,
// the timestamp, and an open metadata map for caller-
// specific data.
type HistoryEvent struct {
	SessionID string         `json:"session_id"`
	TaskID    string         `json:"task_id"`
	Action    string         `json:"action"`
	Actor     string         `json:"actor"`
	Timestamp string         `json:"timestamp"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

// SessionHistory is the session history component.
//
// The history writes events to the audit log. Each
// event is a row with kind="session_event" and the
// payload above. The history is append-only with a
// hash chain (the audit log's chain_hash).
//
// The history is safe for concurrent use by multiple
// goroutines because the audit log is.
type SessionHistory struct {
	log   *audit.Log
	now   func() time.Time
	actor string
}

// SessionHistoryConfig controls SessionHistory
// construction.
type SessionHistoryConfig struct {
	// Log is the audit log. Required.
	Log *audit.Log
	// Now is the decision clock. The clock is used for
	// the Timestamp field on Add. Defaults to time.Now
	// when nil.
	Now func() time.Time
	// Actor is the default actor for events. When the
	// caller leaves the HistoryEvent.Actor empty, the
	// history uses this value.
	Actor string
}

// NewSessionHistory constructs a SessionHistory. The
// function returns an error when the log is nil.
func NewSessionHistory(cfg SessionHistoryConfig) (*SessionHistory, error) {
	if cfg.Log == nil {
		return nil, &HistoryError{Op: "init", Reason: "audit log is required"}
	}
	now := cfg.Now
	if now == nil {
		now = time.Now
	}
	return &SessionHistory{
		log:   cfg.Log,
		now:   now,
		actor: cfg.Actor,
	}, nil
}

// historyKind is the audit-row kind used by the
// history. The constant is exported as HistoryKind.
const historyKind = "session_event"

// HistoryKind is the audit-row kind the history uses.
// The constant is exported so external code can query
// the audit log for history events.
const HistoryKind = historyKind

// Add persists a history event to the audit log.
//
// REQ-PRS-HST-002.
func (h *SessionHistory) Add(ev HistoryEvent) error {
	if ev.SessionID == "" {
		return &HistoryError{Op: "add", Reason: "session_id is required"}
	}
	if ev.TaskID == "" {
		return &HistoryError{Op: "add", Reason: "task_id is required"}
	}
	if ev.Action == "" {
		return &HistoryError{Op: "add", Reason: "action is required"}
	}
	if ev.Actor == "" {
		ev.Actor = h.actor
	}
	if ev.Timestamp == "" {
		ev.Timestamp = h.now().UTC().Format(time.RFC3339Nano)
	}
	row := &audit.Row{
		TaskID: ev.TaskID,
		Kind:   historyKind,
		Actor:  ev.Actor,
		// History events are informational. RiskTier
		// is 0 (per audit row risk_tier semantics).
		RiskTier: 0,
		Payload:  ev,
	}
	_, err := h.log.Write(row)
	if err != nil {
		return &HistoryError{Op: "add", Reason: err.Error()}
	}
	return nil
}

// List returns the events for a session, in time order
// (oldest first). The function filters the audit log by
// kind and session_id in SQL via json_extract, so the
// query scales with the (kind, sequence_number) index
// and does not require loading all rows into Go.
//
// REQ-PRS-HST-003.
func (h *SessionHistory) List(sessionID string) ([]HistoryEvent, error) {
	if sessionID == "" {
		return nil, &HistoryError{Op: "list", Reason: "session_id is required"}
	}
	rows, err := h.log.GetByKindPayloadField(historyKind, "session_id", sessionID, 10000)
	if err != nil {
		return nil, &HistoryError{Op: "list", Reason: err.Error()}
	}
	out := make([]HistoryEvent, 0, len(rows))
	for _, r := range rows {
		var ev HistoryEvent
		if err := json.Unmarshal(r.Payload, &ev); err != nil {
			// A corrupt row is skipped, not fatal.
			continue
		}
		out = append(out, ev)
	}
	return out, nil
}

// Search returns the events whose stringified payload
// contains query. The function reads the audit log by
// kind and applies a simple substring filter in Go. The
// read is bounded to a recent window because the search
// is a user-driven lookup, not a system-wide scan.
//
// REQ-PRS-HST-004.
func (h *SessionHistory) Search(query string) ([]HistoryEvent, error) {
	if query == "" {
		return nil, &HistoryError{Op: "search", Reason: "query is required"}
	}
	rows, err := h.log.GetByKind(historyKind, 1000)
	if err != nil {
		return nil, &HistoryError{Op: "search", Reason: err.Error()}
	}
	out := make([]HistoryEvent, 0, len(rows))
	for _, r := range rows {
		var ev HistoryEvent
		if err := json.Unmarshal(r.Payload, &ev); err != nil {
			continue
		}
		if h.eventMatches(ev, query) {
			out = append(out, ev)
		}
	}
	return out, nil
}

// eventMatches reports whether the query is a substring
// of any of the event's text fields. The match is
// case-insensitive.
func (h *SessionHistory) eventMatches(ev HistoryEvent, query string) bool {
	q := strings.ToLower(query)
	if strings.Contains(strings.ToLower(ev.SessionID), q) {
		return true
	}
	if strings.Contains(strings.ToLower(ev.TaskID), q) {
		return true
	}
	if strings.Contains(strings.ToLower(ev.Action), q) {
		return true
	}
	if strings.Contains(strings.ToLower(ev.Actor), q) {
		return true
	}
	if strings.Contains(strings.ToLower(ev.Timestamp), q) {
		return true
	}
	if ev.Metadata != nil {
		raw, err := json.Marshal(ev.Metadata)
		if err == nil && strings.Contains(strings.ToLower(string(raw)), q) {
			return true
		}
	}
	return false
}

// HistoryError is the typed error for the history
// component.
type HistoryError struct {
	Op     string
	Reason string
}

func (e *HistoryError) Error() string {
	return fmt.Sprintf("persist: history: %s: %s.", e.Op, e.Reason)
}

// _ keeps the clock import honest in case future
// versions of the history take a clock interface.
var _ clock.Decision = clock.SystemDecision{}
