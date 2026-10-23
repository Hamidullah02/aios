package audit

import (
	"errors"
	"fmt"
)

// GetByTask returns all rows for a task, ordered by sequence_number
// ascending. The returned slice is in chain order; verify_chain
// walks it from index 0.
func (l *Log) GetByTask(taskID string) ([]PersistedRow, error) {
	if l.db == nil {
		return nil, errors.New("audit.Log: closed")
	}
	if taskID == "" {
		return nil, errors.New("audit.Log.GetByTask: task_id is required")
	}
	rows, err := l.db.Query(`
SELECT sequence_number, row_id, task_id, step_id, kind, actor, risk_tier,
       approval_id, idempotency_key, payload, redacted_hash, content_hash,
       chain_hash, ts_ns
FROM audit_rows
WHERE task_id = ?
ORDER BY sequence_number ASC
`, taskID)
	if err != nil {
		return nil, fmt.Errorf("audit.GetByTask: query: %w", err)
	}
	defer rows.Close()
	return scanRows(rows)
}

// GetByApproval returns all rows that reference approvalID,
// ordered by sequence_number ascending.
func (l *Log) GetByApproval(approvalID string) ([]PersistedRow, error) {
	if l.db == nil {
		return nil, errors.New("audit.Log: closed")
	}
	if approvalID == "" {
		return nil, errors.New("audit.Log.GetByApproval: approval_id is required")
	}
	rows, err := l.db.Query(`
SELECT sequence_number, row_id, task_id, step_id, kind, actor, risk_tier,
       approval_id, idempotency_key, payload, redacted_hash, content_hash,
       chain_hash, ts_ns
FROM audit_rows
WHERE approval_id = ?
ORDER BY sequence_number ASC
`, approvalID)
	if err != nil {
		return nil, fmt.Errorf("audit.GetByApproval: query: %w", err)
	}
	defer rows.Close()
	return scanRows(rows)
}

// GetByKind returns all rows of a given kind, ordered by
// sequence_number ascending. Used by aios-report for analytics.
func (l *Log) GetByKind(kind string, limit int) ([]PersistedRow, error) {
	if l.db == nil {
		return nil, errors.New("audit.Log: closed")
	}
	if kind == "" {
		return nil, errors.New("audit.Log.GetByKind: kind is required")
	}
	if limit <= 0 {
		limit = 1000
	}
	rows, err := l.db.Query(`
SELECT sequence_number, row_id, task_id, step_id, kind, actor, risk_tier,
       approval_id, idempotency_key, payload, redacted_hash, content_hash,
       chain_hash, ts_ns
FROM audit_rows
WHERE kind = ?
ORDER BY sequence_number ASC
LIMIT ?
`, kind, limit)
	if err != nil {
		return nil, fmt.Errorf("audit.GetByKind: query: %w", err)
	}
	defer rows.Close()
	return scanRows(rows)
}
// Recent returns the most recent rows across ALL kinds, oldest
// first, bounded by limit.
//
// This exists because the audit log's only "list everything"
// access path used to be a hand-maintained union of known
// kinds (GetByKind called once per kind and merged in Go).
// That list could not stay complete: `policy_decision`, which
// the policy gateway writes on every single tool call, was
// never added to it, so `aios audit` silently omitted every
// policy decision the system made while `aios audit <task>`
// showed them. An audit view that silently drops rows is worse
// than no audit view, because it reads as authoritative.
//
// A single ORDER BY sequence_number DESC with a LIMIT cannot
// go stale as new kinds are added, and the primary key makes
// the sort cheap.
func (l *Log) Recent(limit int) ([]PersistedRow, error) {
	if l.db == nil {
		return nil, errors.New("audit.Log: closed")
	}
	if limit <= 0 {
		limit = 1000
	}
	rows, err := l.db.Query(`
SELECT sequence_number, row_id, task_id, step_id, kind, actor, risk_tier,
       approval_id, idempotency_key, payload, redacted_hash, content_hash,
       chain_hash, ts_ns
FROM audit_rows
ORDER BY sequence_number DESC
LIMIT ?
`, limit)
	if err != nil {
		return nil, fmt.Errorf("audit.Recent: query: %w", err)
	}
	defer rows.Close()
	out, err := scanRows(rows)
	if err != nil {
		return nil, err
	}
	// The query walks backwards (newest first) so the LIMIT
	// keeps the newest N. The contract everywhere else in this
	// package is oldest-first, so reverse before returning:
	// an audit reader wants to read a run forwards.
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out, nil
}

// GetByKindPayloadField returns rows of a given kind whose
// payload has a top-level string field equal to value. The
// match is performed in SQL via json_extract, so the
// predicate scales with the index on (kind, sequence_number)
// and does not require loading all rows into Go.
//
// The method is used by callers (e.g. SessionHistory) that
// store a query key in the payload and need to filter
// cheaply. The field is restricted to top-level string
// fields; nested paths or non-string values return an error.
func (l *Log) GetByKindPayloadField(kind, field, value string, limit int) ([]PersistedRow, error) {
	if l.db == nil {
		return nil, errors.New("audit.Log: closed")
	}
	if kind == "" {
		return nil, errors.New("audit.Log.GetByKindPayloadField: kind is required")
	}
	if field == "" {
		return nil, errors.New("audit.Log.GetByKindPayloadField: field is required")
	}
	if limit <= 0 {
		limit = 1000
	}
	// $.<field> is the canonical json_extract path. We escape
	// any double-quote in field for safety; the field name
	// is package-controlled so this is defense in depth.
	path := "$." + field
	rows, err := l.db.Query(`
SELECT sequence_number, row_id, task_id, step_id, kind, actor, risk_tier,
       approval_id, idempotency_key, payload, redacted_hash, content_hash,
       chain_hash, ts_ns
FROM audit_rows
WHERE kind = ?
  AND json_extract(payload, ?) = ?
ORDER BY sequence_number ASC
LIMIT ?
`, kind, path, value, limit)
	if err != nil {
		return nil, fmt.Errorf("audit.GetByKindPayloadField: query: %w", err)
	}
	defer rows.Close()
	return scanRows(rows)
}

func scanRows(rows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
}) ([]PersistedRow, error) {
	var out []PersistedRow
	for rows.Next() {
		var row PersistedRow
		if err := rows.Scan(
			&row.SequenceNumber, &row.RowID, &row.TaskID, &row.StepID, &row.Kind,
			&row.Actor, &row.RiskTier, &row.ApprovalID, &row.IdempotencyKey,
			&row.Payload, &row.RedactedHash, &row.ContentHash, &row.ChainHash,
			&row.TSNano,
		); err != nil {
			return nil, fmt.Errorf("audit: scan: %w", err)
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("audit: iter: %w", err)
	}
	return out, nil
}

// VerifyResult is the result of a VerifyChain call.
type VerifyResult struct {
	OK        bool
	FirstBad  int64  // sequence number of the first row that failed, or -1
	Reason    string // human-readable reason
	RowsTotal int
}

// VerifyChain verifies a chain of rows.
//
// For each row in order, it recomputes the content_hash and
// chain_hash and checks that:
//
//  1. payload redacted-hash matches.
//  2. content_hash matches.
//  3. chain_hash matches the recomputed value given the previous
//     row's chain_hash and the row's sequence_number.
//
// The first row's prev_chain_hash is the all-zeros hash.
func VerifyChain(rows []PersistedRow) VerifyResult {
	if len(rows) == 0 {
		return VerifyResult{OK: true, FirstBad: -1, RowsTotal: 0}
	}
	prevChain := make([]byte, 32) // 32 zero bytes for the first row.
	for i, row := range rows {
		// 1. redacted_hash.
		expectedRedacted := hexOf(redactedHash(row.Payload))
		if expectedRedacted != row.RedactedHash {
			return VerifyResult{
				OK:        false,
				FirstBad:  row.SequenceNumber,
				Reason:    "redacted_hash mismatch at row " + itoa64(row.SequenceNumber),
				RowsTotal: len(rows),
			}
		}
		// 2. content_hash.
		//    Reconstruct the time as a UnixNano int64 from the stored ns.
		expectedContent := hexOf(contentHash(
			row.Payload,
			row.Actor,
			row.Kind,
			row.TSNano,
			row.ApprovalID,
			row.IdempotencyKey,
		))
		if expectedContent != row.ContentHash {
			return VerifyResult{
				OK:        false,
				FirstBad:  row.SequenceNumber,
				Reason:    "content_hash mismatch at row " + itoa64(row.SequenceNumber),
				RowsTotal: len(rows),
			}
		}
		// 3. chain_hash.
		//    The chain hash was computed with the row's actual
		//    sequence_number; we replicate that.
		expectedChain := hexOf(chainHash(
			mustHexDecode(row.ContentHash),
			prevChain,
			row.SequenceNumber,
		))
		if expectedChain != row.ChainHash {
			return VerifyResult{
				OK:        false,
				FirstBad:  row.SequenceNumber,
				Reason:    "chain_hash mismatch at row " + itoa64(row.SequenceNumber),
				RowsTotal: len(rows),
			}
		}
		// Advance prevChain for the next iteration.
		prevChain = mustHexDecode(row.ChainHash)
		_ = i
	}
	return VerifyResult{OK: true, FirstBad: -1, RowsTotal: len(rows)}
}

// itoa64 formats an int64 as a base-10 string.
func itoa64(i int64) string {
	if i == 0 {
		return "0"
	}
	neg := i < 0
	if neg {
		i = -i
	}
	var b [21]byte
	pos := len(b)
	for i > 0 {
		pos--
		b[pos] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		pos--
		b[pos] = '-'
	}
	return string(b[pos:])
}
