package audit

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/sadi/aios-system/internal/core/clock"

	_ "modernc.org/sqlite" // pure-Go SQLite driver
)

// Log is a handle to the audit database.
//
// The Log is safe for concurrent use by multiple goroutines, but a
// single Write call is not. Each Write takes the writer's mutex.
//
// The database handle has its own lock. Close releases the
// database while another goroutine may be halfway through a
// query, so the pointer itself -- not just the database behind it
// -- has to be protected: reading `l.db` and writing `l.db` are
// two operations on the same word, and the race detector is right
// to complain about them.
type Log struct {
	// mu serialises writes. The chain hash of a row is computed
	// from the previous row's hash, so two writers must not
	// interleave.
	mu sync.Mutex
	// dbMu guards the db pointer. An operation holds it for
	// reading for the whole operation, so Close waits for
	// in-flight work rather than pulling the database out from
	// under it.
	dbMu sync.RWMutex
	db   *sql.DB

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
		// Create the directory the database lives in.
		//
		// The function's contract is "opens or creates a Log",
		// and a sqlite file cannot be created in a directory
		// that does not exist. Without this, a first run in a
		// fresh workspace failed with "audit.Open: ping:
		// unable to open database file: out of memory (14)" --
		// which is sqlite's error for "no such directory", and
		// it names neither the cause nor the fix. The operator
		// is told to run `aios init`, which is a step they had
		// no way to know about.
		if dir := filepath.Dir(abs); dir != "" {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return nil, fmt.Errorf("audit.Open: create directory %s: %w", dir, err)
			}
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

// errClosed is the error every operation on a closed Log returns.
var errClosed = errors.New("audit.Log: closed")

// Close releases the database. After Close the Log is unusable.
//
// Close takes the handle's write lock, so it waits for an
// operation that is already in flight to finish before the
// database goes away. Without that wait, a task still running
// under a server that is shutting down would find its audit
// rows failing with "sql: database is closed" halfway through,
// and the pointer itself would be read and written at the same
// time.
func (l *Log) Close() error {
	l.dbMu.Lock()
	defer l.dbMu.Unlock()
	if l.db == nil {
		return nil
	}
	err := l.db.Close()
	l.db = nil
	return err
}

// acquire takes the database handle for one operation.
//
// On success the caller owns the handle's read lock and MUST
// release it with `defer l.dbMu.RUnlock()`. On failure the lock
// has already been released and the caller must not unlock it
// again.
//
// The read lock spans the whole operation on purpose: it is what
// makes "the database is still there" and "the database is still
// usable" the same statement.
func (l *Log) acquire() (*sql.DB, error) {
	l.dbMu.RLock()
	if l.db == nil {
		l.dbMu.RUnlock()
		return nil, errClosed
	}
	return l.db, nil
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
