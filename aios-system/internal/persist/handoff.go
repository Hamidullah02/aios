package persist

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// Handoff is the structured handoff.
//
// The handoff stores an artifact under
// <path>/<task_id>/<key>.json. The handoff supports
// two kinds of payloads:
//
//   - file: the value is a path. The handoff stores the
//     path. When the next task reads the handoff, the
//     handoff returns the path (and the existence flag).
//   - value: the value is a JSON-serializable object.
//     The handoff stores the value as JSON.
//
// The two kinds are distinguished by the IsPath field on
// Put. The Get function reads the marker and returns the
// raw value; the caller decides how to interpret a path
// payload.
//
// The handoff is safe for concurrent use by multiple
// goroutines.
type Handoff struct {
	mu   sync.RWMutex
	path string
}

// HandoffConfig controls Handoff construction.
type HandoffConfig struct {
	// Path is the directory under which the handoffs
	// are stored. Defaults to
	// $AIOS_DATA_DIR/handoffs/ when empty.
	Path string
}

// NewHandoff constructs a Handoff. The function creates
// the directory if missing.
func NewHandoff(cfg HandoffConfig) (*Handoff, error) {
	p := cfg.Path
	if p == "" {
		p = defaultHandoffPath()
	}
	if err := os.MkdirAll(p, 0o755); err != nil {
		return nil, &HandoffError{Op: "init", Reason: err.Error()}
	}
	return &Handoff{path: p}, nil
}

// HandoffRecord is the on-disk form of a handoff. The
// struct is the JSON the handoff writes to disk; the
// caller sees the Value through Get.
//
// IsPath is true when the value is a file path. The
// caller is expected to interpret the value as a path
// and read the file.
type HandoffRecord struct {
	TaskID   string `json:"task_id"`
	Key      string `json:"key"`
	Value    any    `json:"value"`
	IsPath   bool   `json:"is_path,omitempty"`
	StoredAt string `json:"stored_at"`
}

// Put stores a handoff. The function serializes the
// value to JSON and writes the file atomically. The
// record's IsPath field is honored on read; the caller
// is expected to interpret the value appropriately.
//
// REQ-PRS-HND-002.
func (h *Handoff) Put(taskID, key string, value any, isPath bool) error {
	if taskID == "" {
		return &HandoffError{Op: "put", Reason: "task_id is required"}
	}
	if key == "" {
		return &HandoffError{Op: "put", Reason: "key is required"}
	}
	rec := HandoffRecord{
		TaskID:   taskID,
		Key:      key,
		Value:    value,
		IsPath:   isPath,
		StoredAt: nowRFC3339(),
	}
	raw, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return &HandoffError{Op: "put", Reason: fmt.Sprintf("marshal: %v", err)}
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if err := os.MkdirAll(filepath.Join(h.path, taskID), 0o755); err != nil {
		return &HandoffError{Op: "put", Reason: err.Error()}
	}
	path := filepath.Join(h.path, taskID, key+".json")
	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".tmp-*")
	if err != nil {
		return &HandoffError{Op: "put", Reason: fmt.Sprintf("temp: %v", err)}
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op if rename succeeded
	if _, err := tmp.Write(raw); err != nil {
		tmp.Close()
		return &HandoffError{Op: "put", Reason: fmt.Sprintf("write: %v", err)}
	}
	if err := tmp.Close(); err != nil {
		return &HandoffError{Op: "put", Reason: fmt.Sprintf("close: %v", err)}
	}
	if err := os.Rename(tmpName, path); err != nil {
		return &HandoffError{Op: "put", Reason: fmt.Sprintf("rename: %v", err)}
	}
	return nil
}

// Get reads a handoff. The function returns the
// HandoffRecord so the caller can check IsPath and
// interpret the value accordingly.
//
// REQ-PRS-HND-003.
func (h *Handoff) Get(taskID, key string) (HandoffRecord, error) {
	if taskID == "" {
		return HandoffRecord{}, &HandoffError{Op: "get", Reason: "task_id is required"}
	}
	if key == "" {
		return HandoffRecord{}, &HandoffError{Op: "get", Reason: "key is required"}
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	path := filepath.Join(h.path, taskID, key+".json")
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return HandoffRecord{}, &HandoffNotFoundError{TaskID: taskID, Key: key}
		}
		return HandoffRecord{}, &HandoffError{Op: "get", Reason: err.Error()}
	}
	var rec HandoffRecord
	if err := json.Unmarshal(raw, &rec); err != nil {
		return HandoffRecord{}, &HandoffError{Op: "get", Reason: fmt.Sprintf("decode: %v", err)}
	}
	return rec, nil
}

// ListKeys returns the keys for a task. The list is
// sorted alphabetically. A missing task directory is
// reported as an empty list.
func (h *Handoff) ListKeys(taskID string) ([]string, error) {
	if taskID == "" {
		return nil, &HandoffError{Op: "list", Reason: "task_id is required"}
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	dir := filepath.Join(h.path, taskID)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, &HandoffError{Op: "list", Reason: err.Error()}
	}
	keys := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if filepath.Ext(name) != ".json" {
			continue
		}
		keys = append(keys, strings.TrimSuffix(name, ".json"))
	}
	sort.Strings(keys)
	return keys, nil
}

// Delete removes a handoff. A missing handoff is
// reported as *HandoffNotFoundError.
func (h *Handoff) Delete(taskID, key string) error {
	if taskID == "" {
		return &HandoffError{Op: "delete", Reason: "task_id is required"}
	}
	if key == "" {
		return &HandoffError{Op: "delete", Reason: "key is required"}
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	path := filepath.Join(h.path, taskID, key+".json")
	if err := os.Remove(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &HandoffNotFoundError{TaskID: taskID, Key: key}
		}
		return &HandoffError{Op: "delete", Reason: err.Error()}
	}
	return nil
}

// Path returns the directory under which the handoffs
// are stored.
func (h *Handoff) Path() string { return h.path }

// HandoffError is the typed error for the handoff
// component.
type HandoffError struct {
	Op     string
	Reason string
}

func (e *HandoffError) Error() string {
	return fmt.Sprintf("persist: handoff: %s: %s.", e.Op, e.Reason)
}

// HandoffNotFoundError is raised by Get and Delete when
// the key is not in the handoff (REQ-PRS-ERR-003).
type HandoffNotFoundError struct {
	TaskID string
	Key    string
}

func (e *HandoffNotFoundError) Error() string {
	return fmt.Sprintf("persist: handoff: %s/%s not found.", e.TaskID, e.Key)
}

// Is reports whether target is also a
// HandoffNotFoundError. The method lets callers use
// errors.Is to detect a missing handoff regardless of
// the concrete value.
func (e *HandoffNotFoundError) Is(target error) bool {
	_, ok := target.(*HandoffNotFoundError)
	return ok
}

// defaultHandoffPath returns the default path. The path
// is $AIOS_DATA_DIR/handoffs/ when set, or
// ./var/handoffs/ otherwise.
func defaultHandoffPath() string {
	if dir := os.Getenv("AIOS_DATA_DIR"); dir != "" {
		return filepath.Join(dir, "handoffs")
	}
	return filepath.Join("var", "handoffs")
}
