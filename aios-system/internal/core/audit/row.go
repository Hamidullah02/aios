// Package audit provides the hash-chained audit log.
//
// The package is the implementation of the aios-core audit store. It
// honors the global PRD §11.2 (three-hash design), §11.7 (verify),
// §11.12 (fail-closed), and §11.14 (idempotency).
//
// Public surface:
//
//   - Row: a single audit row. The caller fills in the fields and
//     passes the row to Log.Write.
//   - Log: a handle to a SQLite database file. Open with Open.
//
// Hash design (global PRD §11.2):
//
//   - content_hash = SHA256(redacted_payload || actor || kind || ts ||
//     approval_id || idempotency_key). Pure function of inputs.
//   - chain_hash   = SHA256(content_hash || prev_chain_hash ||
//     sequence_number). Adds the previous row's chain_hash and the
//     row's monotonic sequence number.
//   - redacted_hash = SHA256(redacted_payload). Stored so a verifier
//     who only has the redacted payload can confirm redaction was
//     applied without re-running the redactor.
//
// Storage: SQLite with WAL journaling. The driver is modernc.org/sqlite
// (pure Go, no CGO).
//
// Fail-closed (global PRD §11.12): if the audit log is locked, full, or
// unwritable, Write returns an error. The caller is responsible for
// treating the error as a run block; aios-core does not write a
// "could-not-audit" row, because that would be circular.
//
// Idempotency (global PRD §11.14): if the caller passes an empty
// idempotency_key, the writer defaults it to the content_hash. The
// (idempotency_key) pair is unique in the database. Re-writing the
// same row returns the existing row and a nil error.
package audit

import (
	"fmt"
	"time"

	"github.com/sadi/aios-system/internal/core/clock"
	"github.com/sadi/aios-system/internal/core/redact"
	"github.com/sadi/aios-system/internal/core/schemas"
)

// Row is a single audit row before hashing.
//
// The writer computes content_hash, chain_hash, and redacted_hash
// and persists the row. The caller never sets the hash fields.
type Row struct {
	// ID is a UUIDv4 assigned by the writer. The caller may pass a
	// pre-assigned ID for cross-module joins (e.g. tying an audit
	// row to an Envelope).
	ID string

	// SequenceNumber is assigned by the writer. The caller may pass
	// a value to override (used by replay); the writer will accept
	// it but only if the value is exactly the next-in-sequence.
	SequenceNumber int64

	// TaskID is the task this row belongs to. Required.
	TaskID string

	// StepID is the per-step identifier. May be empty for non-step
	// rows (e.g. task_created).
	StepID string

	// Kind is the row type. Examples: "task_created", "decision",
	// "tool_call", "approval_requested", "approval_granted",
	// "approval_denied", "policy_check", "task_resumed",
	// "task_cancelled", "error". Required.
	Kind string

	// Actor is the entity that performed the action. Examples:
	// "planner", "runtime", "operator:alice". Required.
	Actor string

	// RiskTier is the risk tier at the time of the row (0-3). 0 is
	// informational; 3 is the highest.
	RiskTier int

	// ApprovalID is the approval ID if this row is part of an
	// approval flow. Optional.
	ApprovalID string

	// IdempotencyKey is the caller's idempotency key. If empty, the
	// writer defaults to the content_hash of the row.
	IdempotencyKey string

	// Payload is the row's typed payload, marshaled to JSON before
	// hashing. May be any type that encoding/json can marshal.
	Payload any

	// RecordedAt is the row's decision-time timestamp. The writer
	// overrides this with the injected decision clock if the caller
	// leaves it zero.
	RecordedAt time.Time

	// payloadJSON is the canonical redacted JSON of Payload, set
	// during hash computation. It is what the writer persists in
	// the `payload` column.
	payloadJSON []byte

	// contentHash is set during Write, after redaction.
	contentHash []byte

	// chainHash is set during Write, after content hash.
	chainHash []byte
}

// validateForWrite checks the row is well-formed before hashing.
// Returns the first error encountered.
func (r *Row) validateForWrite() error {
	if r.TaskID == "" {
		return fmt.Errorf("audit row: task_id is required")
	}
	if r.Kind == "" {
		return fmt.Errorf("audit row: kind is required")
	}
	if r.Actor == "" {
		return fmt.Errorf("audit row: actor is required")
	}
	if r.RiskTier < 0 || r.RiskTier > 3 {
		return fmt.Errorf("audit row: risk_tier must be in [0, 3], got %d", r.RiskTier)
	}
	return nil
}

// marshalPayload canonicalizes the payload with redaction applied.
// The approval_id field is preserved verbatim (see redact.RedactValue).
//
// If the payload is a Schema, it is validated first. Validation
// failures are returned as errors.
func (r *Row) marshalPayload() error {
	if s, ok := r.Payload.(schemas.Schema); ok {
		if err := s.Validate(); err != nil {
			return fmt.Errorf("audit row payload: %w", err)
		}
	}
	// Marshal the payload as canonical JSON, then hand the
	// bytes to the redactor. The redactor performs its own
	// decode + recurse + re-encode; we store the result.
	raw, err := schemas.MarshalCanonical(r.Payload)
	if err != nil {
		return fmt.Errorf("audit row payload: marshal: %w", err)
	}
	final, err := redact.RedactJSONBytes(raw, "approval_id")
	if err != nil {
		return fmt.Errorf("audit row payload: redact: %w", err)
	}
	r.payloadJSON = final
	return nil
}

// timestamp returns the row's decision-time timestamp.
//
// If the caller left RecordedAt zero, the writer fills it from the
// injected decision clock. Tests pass a fixed clock.
func (r *Row) timestamp(decisionClock clock.Decision) time.Time {
	if !r.RecordedAt.IsZero() {
		return r.RecordedAt
	}
	if decisionClock != nil {
		return decisionClock.Now()
	}
	return clock.SystemDecision{}.Now()
}
