package persist

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// SessionState is the saved state of one session.
//
// The struct is a tagged union: the Status field names
// the lifecycle state, the StartedAt field is the
// creation time, the Steps field is the per-step state,
// and the Metadata field is an open map for caller-
// specific data (e.g. last tool call, current task).
//
// The checkpoint is content addressed through the
// ContentHash method; the hash is computed over the
// canonical JSON of the struct.
type SessionState struct {
	SessionID string         `json:"session_id"`
	Status    string         `json:"status"`
	StartedAt string         `json:"started_at"`
	UpdatedAt string         `json:"updated_at"`
	Steps     []StepState    `json:"steps,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

// StepState is the per-step state within a session.
//
// The struct is the minimal "what was happening at this
// step" record. The fields are intentionally free-form:
// the planner and the runtime each add their own
// metadata.
type StepState struct {
	StepID    string         `json:"step_id"`
	Tool      string         `json:"tool,omitempty"`
	Status    string         `json:"status,omitempty"`
	StartedAt string         `json:"started_at,omitempty"`
	EndedAt   string         `json:"ended_at,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

// SessionCheckpoint is the session state checkpoint.
//
// The checkpoint stores each session as a JSON file
// under <path>/<session_id>.json. When the JSON is
// above the compress threshold (default 1 MB), the file
// is gzipped on disk; the suffix becomes .json.gz.
//
// The checkpoint is safe for concurrent use by multiple
// goroutines.
type SessionCheckpoint struct {
	mu            sync.RWMutex
	path          string
	compressAbove int
	now           func() time.Time
}

// SessionCheckpointConfig controls SessionCheckpoint
// construction.
type SessionCheckpointConfig struct {
	// Path is the directory under which the session
	// files are stored. Defaults to
	// $AIOS_DATA_DIR/sessions/ when empty.
	Path string
	// CompressAbove is the byte size above which the
	// checkpoint is gzipped on disk. Zero means the
	// default 1 MB.
	CompressAbove int
	// Now is the decision clock. The clock is used for
	// the UpdatedAt field on save. Defaults to time.Now
	// when nil.
	Now func() time.Time
}

// DefaultCompressAbove is the default byte-size threshold
// above which a checkpoint is gzipped on disk.
const DefaultCompressAbove = 1 << 20

// NewSessionCheckpoint constructs a SessionCheckpoint.
// The function creates the directory if missing. A
// non-existing directory is created; an existing
// directory is left as-is.
func NewSessionCheckpoint(cfg SessionCheckpointConfig) (*SessionCheckpoint, error) {
	p := cfg.Path
	if p == "" {
		p = defaultSessionsPath()
	}
	if err := os.MkdirAll(p, 0o755); err != nil {
		return nil, &CheckpointError{Op: "init", Reason: err.Error()}
	}
	ca := cfg.CompressAbove
	if ca == 0 {
		ca = DefaultCompressAbove
	}
	now := cfg.Now
	if now == nil {
		now = time.Now
	}
	return &SessionCheckpoint{
		path:          p,
		compressAbove: ca,
		now:           now,
	}, nil
}

// Save persists the session state. The function updates
// the UpdatedAt field, marshals the state to JSON, and
// writes the file atomically. The compression decision
// is made after the marshal: the file is gzipped when
// the JSON is above the threshold.
//
// REQ-PRS-CHK-002.
func (c *SessionCheckpoint) Save(state SessionState) error {
	if state.SessionID == "" {
		return &CheckpointError{Op: "save", Reason: "session_id is required"}
	}
	state.UpdatedAt = c.now().UTC().Format(time.RFC3339Nano)
	raw, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return &CheckpointError{Op: "save", Reason: fmt.Sprintf("marshal: %v", err)}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(raw) > c.compressAbove {
		return c.writeGzipped(state.SessionID, raw)
	}
	return c.writePlain(state.SessionID, raw)
}

// Load reads a session state from disk. The function
// transparently decompresses gzip-compressed files.
//
// REQ-PRS-CHK-003.
func (c *SessionCheckpoint) Load(sessionID string) (SessionState, error) {
	if sessionID == "" {
		return SessionState{}, &CheckpointError{Op: "load", Reason: "session_id is required"}
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	plain, err := c.readIfExists(c.pathForPlain(sessionID))
	if err != nil {
		return SessionState{}, err
	}
	if plain == nil {
		raw, gerr := c.readGzipped(c.pathForGzip(sessionID))
		if gerr != nil {
			return SessionState{}, gerr
		}
		if raw == nil {
			return SessionState{}, &SessionNotFoundError{ID: sessionID}
		}
		plain = raw
	}
	var state SessionState
	if err := json.Unmarshal(plain, &state); err != nil {
		return SessionState{}, &CheckpointError{Op: "load", Reason: fmt.Sprintf("decode: %v", err)}
	}
	return state, nil
}

// List returns the metadata for every session. The
// function reads each file's small header (session id,
// status, started_at, updated_at) and returns the
// ordered list. The list is sorted by UpdatedAt
// descending (most recent first).
//
// REQ-PRS-CHK-004.
func (c *SessionCheckpoint) List() ([]SessionState, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entries, err := os.ReadDir(c.path)
	if err != nil {
		return nil, &CheckpointError{Op: "list", Reason: err.Error()}
	}
	out := make([]SessionState, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		sessionID, ok := sessionIDFromFilename(name)
		if !ok {
			continue
		}
		state, err := c.loadFromPath(filepath.Join(c.path, name), sessionID)
		if err != nil {
			// A corrupt file is skipped, not fatal. The
			// CLI shows the sessions that loaded.
			continue
		}
		out = append(out, state)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UpdatedAt > out[j].UpdatedAt })
	return out, nil
}

// Delete removes a session from disk. The function
// accepts both the plain and the gzipped filename. A
// missing session is reported as
// *SessionNotFoundError.
//
// REQ-PRS-CHK-005.
func (c *SessionCheckpoint) Delete(sessionID string) error {
	if sessionID == "" {
		return &CheckpointError{Op: "delete", Reason: "session_id is required"}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	plain := c.pathForPlain(sessionID)
	gzip := c.pathForGzip(sessionID)
	plainErr := os.Remove(plain)
	gzipErr := os.Remove(gzip)
	if plainErr != nil && !errors.Is(plainErr, os.ErrNotExist) {
		return &CheckpointError{Op: "delete", Reason: plainErr.Error()}
	}
	if gzipErr != nil && !errors.Is(gzipErr, os.ErrNotExist) {
		return &CheckpointError{Op: "delete", Reason: gzipErr.Error()}
	}
	if plainErr != nil && gzipErr != nil {
		return &SessionNotFoundError{ID: sessionID}
	}
	return nil
}

// Path returns the directory under which the session
// files are stored.
func (c *SessionCheckpoint) Path() string { return c.path }

// writePlain writes the JSON to <id>.json.
func (c *SessionCheckpoint) writePlain(id string, raw []byte) error {
	return c.writeAtomic(c.pathForPlain(id), raw)
}

// writeGzipped writes the JSON to <id>.json.gz.
func (c *SessionCheckpoint) writeGzipped(id string, raw []byte) error {
	var buf bytes.Buffer
	gw, err := gzip.NewWriterLevel(&buf, gzip.DefaultCompression)
	if err != nil {
		return &CheckpointError{Op: "save", Reason: fmt.Sprintf("gzip: %v", err)}
	}
	if _, err := gw.Write(raw); err != nil {
		return &CheckpointError{Op: "save", Reason: fmt.Sprintf("gzip write: %v", err)}
	}
	if err := gw.Close(); err != nil {
		return &CheckpointError{Op: "save", Reason: fmt.Sprintf("gzip close: %v", err)}
	}
	return c.writeAtomic(c.pathForGzip(id), buf.Bytes())
}

// readGzipped reads and decompresses <id>.json.gz.
func (c *SessionCheckpoint) readGzipped(path string) ([]byte, error) {
	raw, err := c.readIfExists(path)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, nil
	}
	gr, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		return nil, &CheckpointError{Op: "load", Reason: fmt.Sprintf("gzip: %v", err)}
	}
	defer gr.Close()
	decoded, err := io.ReadAll(gr)
	if err != nil {
		return nil, &CheckpointError{Op: "load", Reason: fmt.Sprintf("gzip read: %v", err)}
	}
	return decoded, nil
}

// readIfExists returns the file's bytes, or (nil, nil)
// when the file does not exist.
func (c *SessionCheckpoint) readIfExists(path string) ([]byte, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, &CheckpointError{Op: "load", Reason: err.Error()}
	}
	return raw, nil
}

// loadFromPath reads the file at path and parses it as
// a SessionState. The function is used by List.
func (c *SessionCheckpoint) loadFromPath(path, sessionID string) (SessionState, error) {
	var raw []byte
	if filepath.Ext(path) == ".gz" {
		decoded, err := c.readGzipped(path)
		if err != nil {
			return SessionState{}, err
		}
		raw = decoded
	} else {
		r, err := c.readIfExists(path)
		if err != nil {
			return SessionState{}, err
		}
		raw = r
	}
	if raw == nil {
		return SessionState{}, &SessionNotFoundError{ID: sessionID}
	}
	var state SessionState
	if err := json.Unmarshal(raw, &state); err != nil {
		return SessionState{}, &CheckpointError{Op: "list", Reason: fmt.Sprintf("decode: %v", err)}
	}
	return state, nil
}

// writeAtomic writes data to path via a temp file in
// the same directory. The temp file is renamed to
// path on success.
func (c *SessionCheckpoint) writeAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".tmp-*")
	if err != nil {
		return &CheckpointError{Op: "save", Reason: fmt.Sprintf("temp: %v", err)}
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op if rename succeeded
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return &CheckpointError{Op: "save", Reason: fmt.Sprintf("write: %v", err)}
	}
	if err := tmp.Close(); err != nil {
		return &CheckpointError{Op: "save", Reason: fmt.Sprintf("close: %v", err)}
	}
	if err := os.Rename(tmpName, path); err != nil {
		return &CheckpointError{Op: "save", Reason: fmt.Sprintf("rename: %v", err)}
	}
	return nil
}

// pathForPlain returns <path>/<id>.json.
func (c *SessionCheckpoint) pathForPlain(id string) string {
	return filepath.Join(c.path, id+".json")
}

// pathForGzip returns <path>/<id>.json.gz.
func (c *SessionCheckpoint) pathForGzip(id string) string {
	return filepath.Join(c.path, id+".json.gz")
}

// sessionIDFromFilename extracts the session id from a
// session filename. The function accepts both the plain
// and gzipped forms.
func sessionIDFromFilename(name string) (string, bool) {
	if filepath.Ext(name) == ".gz" {
		name = strings.TrimSuffix(name, ".gz")
	}
	if filepath.Ext(name) != ".json" {
		return "", false
	}
	return strings.TrimSuffix(name, ".json"), true
}

// CheckpointError is the typed error for the checkpoint
// component.
type CheckpointError struct {
	Op     string
	Reason string
}

func (e *CheckpointError) Error() string {
	return fmt.Sprintf("persist: checkpoint: %s: %s.", e.Op, e.Reason)
}

// SessionNotFoundError is raised by Load and Delete when
// the session id is not in the checkpoint
// (REQ-PRS-ERR-002).
type SessionNotFoundError struct {
	ID string
}

func (e *SessionNotFoundError) Error() string {
	return fmt.Sprintf("persist: checkpoint: session %q not found.", e.ID)
}

// Is reports whether target is also a
// SessionNotFoundError. The method lets callers use
// errors.Is to detect a missing session regardless of
// the concrete value.
func (e *SessionNotFoundError) Is(target error) bool {
	_, ok := target.(*SessionNotFoundError)
	return ok
}

// defaultSessionsPath returns the default path. The path
// is $AIOS_DATA_DIR/sessions/ when set, or
// ./var/sessions/ otherwise.
func defaultSessionsPath() string {
	if dir := os.Getenv("AIOS_DATA_DIR"); dir != "" {
		return filepath.Join(dir, "sessions")
	}
	return filepath.Join("var", "sessions")
}
