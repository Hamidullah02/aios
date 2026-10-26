package audit

import (
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

// ErrDuplicate is returned by Write when a row with the same
// idempotency key already exists. The existing row is returned
// alongside via the second return value.
var ErrDuplicate = errors.New("audit: duplicate idempotency key")

// WriteResult is what Write returns.
//
// `Existing` is set when the row was already present (idempotent
// retry); it is nil for a fresh write.
type WriteResult struct {
	Row      *Row
	Existing *PersistedRow
}

// PersistedRow is the read-back shape of a row from the database.
type PersistedRow struct {
	SequenceNumber int64
	RowID          string
	TaskID         string
	StepID         string
	Kind           string
	Actor          string
	RiskTier       int
	ApprovalID     string
	IdempotencyKey string
	Payload        []byte
	RedactedHash   string
	ContentHash    string
	ChainHash      string
	TSNano         int64
}

// Write persists a row.
//
// Steps:
//
//  1. Validate the row (required fields, risk_tier range).
//  2. Canonicalize and redact the payload.
//  3. Fill in the timestamp from the decision clock if zero.
//  4. Compute content_hash, chain_hash, redacted_hash.
//  5. Insert atomically; on UNIQUE(idempotency_key) conflict, return
//     the existing row with ErrDuplicate.
//
// Fail-closed: if any step fails, the function returns the error
// and no row is written. The caller treats the error as a run block.
func (l *Log) Write(r *Row) (*WriteResult, error) {
	if r == nil {
		return nil, errors.New("audit.Log: nil row")
	}
	db, err := l.acquire()
	if err != nil {
		return nil, err
	}
	defer l.dbMu.RUnlock()
	l.mu.Lock()
	defer l.mu.Unlock()

	if err := r.validateForWrite(); err != nil {
		return nil, err
	}
	if err := r.marshalPayload(); err != nil {
		return nil, err
	}
	if r.RecordedAt.IsZero() {
		r.RecordedAt = l.clock.Now()
	}

	// Idempotency pre-check: if a row with this idempotency_key
	// already exists, return it without opening a transaction. This
	// avoids the case where the INSERT fails inside a transaction and
	// we have to roll back before the lookup, which can deadlock in
	// single-connection drivers like modernc.org/sqlite.
	if r.IdempotencyKey != "" {
		existing, lookupErr := lookupByIdempotencyKey(db, r.IdempotencyKey)
		if lookupErr == nil {
			return &WriteResult{Row: r, Existing: existing}, ErrDuplicate
		}
		if !errors.Is(lookupErr, sql.ErrNoRows) {
			return nil, fmt.Errorf("audit.Write: precheck lookup: %w", lookupErr)
		}
	}

	// Read the previous chain hash and the next sequence number
	// atomically. We use a transaction so the chain stays intact
	// even under concurrent writers.
	tx, err := db.Begin()
	if err != nil {
		return nil, fmt.Errorf("audit.Write: begin: %w", err)
	}
	defer tx.Rollback()

	var prevChainHex string
	if err := tx.QueryRow(`SELECT COALESCE((
		SELECT chain_hash FROM audit_rows ORDER BY sequence_number DESC LIMIT 1
	), '')`).Scan(&prevChainHex); err != nil {
		if err != sql.ErrNoRows {
			return nil, fmt.Errorf("audit.Write: read prev chain: %w", err)
		}
	}
	// prevChainHex is the hex of the previous row's chain hash, or
	// empty for the first row. Convert to 32 raw bytes for hashing.
	prevChain := make([]byte, 32)
	if prevChainHex != "" {
		decoded, err := hex.DecodeString(prevChainHex)
		if err != nil {
			return nil, fmt.Errorf("audit.Write: decode prev chain: %w", err)
		}
		if len(decoded) != 32 {
			return nil, fmt.Errorf("audit.Write: prev chain len: %d, want 32", len(decoded))
		}
		prevChain = decoded
	}

	// Compute hashes inside the transaction so we can use the row's
	// actual sequence number from AUTOINCREMENT.
	// We pre-compute a stub sequence number of 0 here; the real one
	// is assigned by SQLite and patched into the chain after insert.
	if err := r.hashAndSeal(prevChain, 0); err != nil {
		return nil, fmt.Errorf("audit.Write: hash: %w", err)
	}

	// Assign row_id if empty.
	rowID := r.ID
	if rowID == "" {
		rowID = uuid.NewString()
	}

	// Insert the row. If idempotency_key collides, return the existing.
	res, err := tx.Exec(`
INSERT INTO audit_rows
    (row_id, task_id, step_id, kind, actor, risk_tier, approval_id,
     idempotency_key, payload, redacted_hash, content_hash, chain_hash, ts_ns)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`,
		rowID, r.TaskID, r.StepID, r.Kind, r.Actor, r.RiskTier, r.ApprovalID,
		r.IdempotencyKey, r.payloadJSON,
		hexOf(redactedHash(r.payloadJSON)),
		hexOf(r.contentHash),
		hexOf(r.chainHash),
		r.RecordedAt.UnixNano(),
	)
	if err != nil {
		// UNIQUE(idempotency_key) conflict. Roll back the transaction
		// first (the defer will be a no-op), then look up the
		// existing row. Holding the transaction while doing another
		// query on the same DB will deadlock with single-connection
		// drivers like modernc.org/sqlite.
		_ = tx.Rollback()
		existing, lookupErr := lookupByIdempotencyKey(db, r.IdempotencyKey)
		if lookupErr == nil {
			return &WriteResult{Row: r, Existing: existing}, ErrDuplicate
		}
		return nil, fmt.Errorf("audit.Write: insert: %w, lookup: %v", err, lookupErr)
	}
	seq, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("audit.Write: LastInsertId: %w", err)
	}

	// Re-compute chain hash with the real sequence number and update
	// the row. This is a single-row update within the same transaction.
	newChain := chainHash(r.contentHash, prevChain, seq)
	if _, err := tx.Exec(`UPDATE audit_rows SET chain_hash = ? WHERE sequence_number = ?`,
		hexOf(newChain), seq); err != nil {
		return nil, fmt.Errorf("audit.Write: update chain: %w", err)
	}
	r.chainHash = newChain
	r.SequenceNumber = seq
	r.ID = rowID
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("audit.Write: commit: %w", err)
	}
	return &WriteResult{Row: r}, nil
}

// lookupByIdempotencyKey reads a single persisted row.
//
// The caller holds the handle's read lock, so the database is
// guaranteed to still be open for the duration of the query.
func lookupByIdempotencyKey(db *sql.DB, key string) (*PersistedRow, error) {
	row := &PersistedRow{}
	err := db.QueryRow(`
SELECT sequence_number, row_id, task_id, step_id, kind, actor, risk_tier,
       approval_id, idempotency_key, payload, redacted_hash, content_hash,
       chain_hash, ts_ns
FROM audit_rows
WHERE idempotency_key = ?
`, key).Scan(
		&row.SequenceNumber, &row.RowID, &row.TaskID, &row.StepID, &row.Kind,
		&row.Actor, &row.RiskTier, &row.ApprovalID, &row.IdempotencyKey,
		&row.Payload, &row.RedactedHash, &row.ContentHash, &row.ChainHash,
		&row.TSNano,
	)
	if err != nil {
		return nil, err
	}
	return row, nil
}
