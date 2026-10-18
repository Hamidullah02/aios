package audit

import (
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"sync"

	"github.com/sadi/aios-system/internal/core/clock"

	_ "modernc.org/sqlite" // pure-Go SQLite driver
)

// Log is a handle to the audit database.
//
// The Log is safe for concurrent use by multiple goroutines, but a
// single Write call is not. Each Write takes the writer's mutex.
type Log struct {
	mu sync.Mutex
	db *sql.DB

	// clock is the decision clock used for row timestamps when the
	// caller leaves RecordedAt zero.
	clock clock.Decision
}

// Config controls Open behavior.
type Config struct {
	// Path is the SQLite database file path. Use ":memory:" for an
	// in-memory database (used in tests).
	Path string

	// Clock is the decision clock. If nil, a SystemDecision is used.
	Clock clock.Decision
}

// Open opens or creates a Log at cfg.Path.
//
// The database is created with WAL journaling, NORMAL synchronous
// mode, and foreign keys enabled. The schema is applied on first
// open. The returned Log must be closed with Close.
func Open(cfg Config) (*Log, error) {
	if cfg.Path == "" {
		return nil, errors.New("audit.Open: path is required")
	}
	c := cfg.Clock
	if c == nil {
		c = clock.SystemDecision{}
	}

	// _journal_mode=WAL & _synchronous=NORMAL are applied via
	// connection parameters. The modernc.org/sqlite driver
	// accepts DSN parameters after the file path.
	dsn := cfg.Path
	if cfg.Path != ":memory:" {
		// Make the path absolute so the user can't be surprised
		// by working-directory changes.
		abs, err := filepath.Abs(cfg.Path)
		if err != nil {
			return nil, fmt.Errorf("audit.Open: abs path: %w", err)
		}
		dsn = abs + "?_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=foreign_keys(1)"
	} else {
		dsn = "file::memory:?cache=shared&_pragma=foreign_keys(1)"
	}

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("audit.Open: sql.Open: %w", err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("audit.Open: ping: %w", err)
	}
	if err := applySchema(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("audit.Open: schema: %w", err)
	}
	return &Log{db: db, clock: c}, nil
}

// Close releases the database. After Close the Log is unusable.
func (l *Log) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.db == nil {
		return nil
	}
	err := l.db.Close()
	l.db = nil
	return err
}

// applySchema creates the audit_rows table if it doesn't exist.
func applySchema(db *sql.DB) error {
	_, err := db.Exec(`
CREATE TABLE IF NOT EXISTS audit_rows (
    sequence_number    INTEGER PRIMARY KEY AUTOINCREMENT,
    row_id             TEXT    NOT NULL,
    task_id            TEXT    NOT NULL,
    step_id            TEXT    NOT NULL DEFAULT '',
    kind               TEXT    NOT NULL,
    actor              TEXT    NOT NULL,
    risk_tier          INTEGER NOT NULL,
    approval_id        TEXT    NOT NULL DEFAULT '',
    idempotency_key    TEXT    NOT NULL,
    payload            BLOB    NOT NULL,
    redacted_hash      TEXT    NOT NULL,
    content_hash       TEXT    NOT NULL,
    chain_hash         TEXT    NOT NULL,
    ts_ns              INTEGER NOT NULL,
    UNIQUE(idempotency_key)
);
CREATE INDEX IF NOT EXISTS idx_audit_rows_task_id ON audit_rows(task_id, sequence_number);
CREATE INDEX IF NOT EXISTS idx_audit_rows_approval_id ON audit_rows(approval_id) WHERE approval_id != '';
CREATE INDEX IF NOT EXISTS idx_audit_rows_kind ON audit_rows(kind, sequence_number);
`)
	return err
}
