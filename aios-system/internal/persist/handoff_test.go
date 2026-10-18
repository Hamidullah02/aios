package persist

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// newTestHandoff constructs a Handoff pointed at a
// fresh temp directory.
func newTestHandoff(t *testing.T) *Handoff {
	t.Helper()
	dir := t.TempDir()
	h, err := NewHandoff(HandoffConfig{Path: dir})
	if err != nil {
		t.Fatalf("NewHandoff: %v", err)
	}
	return h
}

// TestHandoffPutGetValue covers the value path. The
// value is JSON-serialized and stored.
func TestHandoffPutGetValue(t *testing.T) {
	h := newTestHandoff(t)
	if err := h.Put("task-1", "result", map[string]any{"answer": 42}, false); err != nil {
		t.Fatalf("Put: %v", err)
	}
	rec, err := h.Get("task-1", "result")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if rec.IsPath {
		t.Fatalf("expected IsPath=false")
	}
	m, ok := rec.Value.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", rec.Value)
	}
	// JSON numbers decode as float64.
	if v, ok := m["answer"].(float64); !ok || v != 42 {
		t.Fatalf("expected 42, got %+v", m["answer"])
	}
	if rec.StoredAt == "" {
		t.Fatalf("StoredAt should be set")
	}
}

// TestHandoffPutGetPath covers the file-path path. The
// handoff stores the path string with IsPath=true.
func TestHandoffPutGetPath(t *testing.T) {
	h := newTestHandoff(t)
	// Create the source file so the handoff test is
	// self-contained.
	src := filepath.Join(t.TempDir(), "src.txt")
	if err := os.WriteFile(src, []byte("hello"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := h.Put("task-1", "report", src, true); err != nil {
		t.Fatalf("Put: %v", err)
	}
	rec, err := h.Get("task-1", "report")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !rec.IsPath {
		t.Fatalf("expected IsPath=true")
	}
	if v, ok := rec.Value.(string); !ok || v != src {
		t.Fatalf("expected path %q, got %v", src, rec.Value)
	}
}

// TestHandoffGetNotFound covers the not-found branch on
// Get.
func TestHandoffGetNotFound(t *testing.T) {
	h := newTestHandoff(t)
	_, err := h.Get("task-1", "missing")
	if !errors.Is(err, &HandoffNotFoundError{}) {
		t.Fatalf("expected *HandoffNotFoundError, got %T", err)
	}
}

// TestHandoffGetEmptyArgs covers the empty-args branch.
func TestHandoffGetEmptyArgs(t *testing.T) {
	h := newTestHandoff(t)
	_, err := h.Get("", "k")
	if err == nil {
		t.Fatalf("expected error for empty task_id")
	}
	_, err = h.Get("t", "")
	if err == nil {
		t.Fatalf("expected error for empty key")
	}
}

// TestHandoffPutEmptyArgs covers the empty-args branch
// on Put.
func TestHandoffPutEmptyArgs(t *testing.T) {
	h := newTestHandoff(t)
	if err := h.Put("", "k", nil, false); err == nil {
		t.Fatalf("expected error for empty task_id")
	}
	if err := h.Put("t", "", nil, false); err == nil {
		t.Fatalf("expected error for empty key")
	}
}

// TestHandoffListKeys covers the list path. The keys are
// returned sorted, and non-json files are skipped.
func TestHandoffListKeys(t *testing.T) {
	h := newTestHandoff(t)
	for _, k := range []string{"c", "a", "b"} {
		if err := h.Put("task-1", k, "x", false); err != nil {
			t.Fatalf("Put %s: %v", k, err)
		}
	}
	// Add a non-json file to the directory.
	if err := os.WriteFile(filepath.Join(h.path, "task-1", "README"), []byte("#"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	keys, err := h.ListKeys("task-1")
	if err != nil {
		t.Fatalf("ListKeys: %v", err)
	}
	if len(keys) != 3 {
		t.Fatalf("expected 3 keys, got %v", keys)
	}
	if keys[0] != "a" || keys[1] != "b" || keys[2] != "c" {
		t.Fatalf("keys not sorted: %v", keys)
	}
}

// TestHandoffListKeysMissing covers the missing-task
// branch on ListKeys.
func TestHandoffListKeysMissing(t *testing.T) {
	h := newTestHandoff(t)
	keys, err := h.ListKeys("task-1")
	if err != nil {
		t.Fatalf("ListKeys: %v", err)
	}
	if len(keys) != 0 {
		t.Fatalf("expected empty, got %v", keys)
	}
}

// TestHandoffListKeysEmptyID covers the empty-id
// branch.
func TestHandoffListKeysEmptyID(t *testing.T) {
	h := newTestHandoff(t)
	_, err := h.ListKeys("")
	if err == nil {
		t.Fatalf("expected error")
	}
}

// TestHandoffDelete covers the delete path.
func TestHandoffDelete(t *testing.T) {
	h := newTestHandoff(t)
	if err := h.Put("task-1", "k", "v", false); err != nil {
		t.Fatalf("Put: %v", err)
	}
	if err := h.Delete("task-1", "k"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := h.Get("task-1", "k"); !errors.Is(err, &HandoffNotFoundError{}) {
		t.Fatalf("expected not found, got %v", err)
	}
}

// TestHandoffDeleteNotFound covers the not-found branch
// on Delete.
func TestHandoffDeleteNotFound(t *testing.T) {
	h := newTestHandoff(t)
	err := h.Delete("task-1", "k")
	if !errors.Is(err, &HandoffNotFoundError{}) {
		t.Fatalf("expected *HandoffNotFoundError, got %T", err)
	}
}

// TestHandoffDeleteEmptyArgs covers the empty-args
// branch on Delete.
func TestHandoffDeleteEmptyArgs(t *testing.T) {
	h := newTestHandoff(t)
	if err := h.Delete("", "k"); err == nil {
		t.Fatalf("expected error")
	}
	if err := h.Delete("t", ""); err == nil {
		t.Fatalf("expected error")
	}
}

// TestHandoffPutOverwrite covers the overwrite path.
// Putting the same key twice keeps the latest value.
func TestHandoffPutOverwrite(t *testing.T) {
	h := newTestHandoff(t)
	if err := h.Put("task-1", "k", "v1", false); err != nil {
		t.Fatalf("Put v1: %v", err)
	}
	if err := h.Put("task-1", "k", "v2", false); err != nil {
		t.Fatalf("Put v2: %v", err)
	}
	rec, err := h.Get("task-1", "k")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if v, ok := rec.Value.(string); !ok || v != "v2" {
		t.Fatalf("expected v2, got %v", rec.Value)
	}
}

// TestHandoffPath covers the path accessor.
func TestHandoffPath(t *testing.T) {
	h := newTestHandoff(t)
	if h.Path() == "" {
		t.Fatalf("expected non-empty path")
	}
}

// TestHandoffInitMissingDir covers the mkdir branch.
func TestHandoffInitMissingDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "handoffs")
	h, err := NewHandoff(HandoffConfig{Path: dir})
	if err != nil {
		t.Fatalf("NewHandoff: %v", err)
	}
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("expected dir to exist: %v", err)
	}
	if h.Path() != dir {
		t.Fatalf("unexpected path: %q", h.Path())
	}
}

// TestHandoffNotFoundErrorIs covers the errors.Is match
// for the typed error.
func TestHandoffNotFoundErrorIs(t *testing.T) {
	var err error = &HandoffNotFoundError{TaskID: "t", Key: "k"}
	if !errors.Is(err, &HandoffNotFoundError{}) {
		t.Fatalf("expected errors.Is to match")
	}
	if msg := err.Error(); !strings.Contains(msg, "t") || !strings.Contains(msg, "k") {
		t.Fatalf("expected ids in message: %q", msg)
	}
}

// TestHandoffErrorMessage covers the error format.
func TestHandoffErrorMessage(t *testing.T) {
	err := (&HandoffError{Op: "put", Reason: "boom"}).Error()
	if !strings.HasPrefix(err, "persist: handoff:") {
		t.Fatalf("unexpected prefix: %q", err)
	}
	if !strings.HasSuffix(err, ".") {
		t.Fatalf("missing trailing period: %q", err)
	}
}

// TestDefaultHandoffPath covers the default-path
// resolver.
func TestDefaultHandoffPath(t *testing.T) {
	t.Setenv("AIOS_DATA_DIR", "")
	got := defaultHandoffPath()
	if !strings.HasSuffix(got, "handoffs") {
		t.Fatalf("expected suffix 'handoffs', got %q", got)
	}
	t.Setenv("AIOS_DATA_DIR", "/custom/data")
	if got := defaultHandoffPath(); got != filepath.Join("/custom/data", "handoffs") {
		t.Fatalf("AIOS_DATA_DIR not honored: %q", got)
	}
}
