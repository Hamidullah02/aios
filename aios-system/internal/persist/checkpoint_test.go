package persist

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// newTestCheckpoint constructs a SessionCheckpoint
// pointed at a fresh temp directory. The function
// returns the checkpoint and a cleanup function.
func newTestCheckpoint(t *testing.T, compressAbove int) *SessionCheckpoint {
	t.Helper()
	dir := t.TempDir()
	fixed := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	c, err := NewSessionCheckpoint(SessionCheckpointConfig{
		Path:          dir,
		CompressAbove: compressAbove,
		Now:           func() time.Time { return fixed },
	})
	if err != nil {
		t.Fatalf("NewSessionCheckpoint: %v", err)
	}
	return c
}

// TestSessionCheckpointSaveLoad covers the round-trip
// save+load path.
func TestSessionCheckpointSaveLoad(t *testing.T) {
	c := newTestCheckpoint(t, 0)
	state := SessionState{
		SessionID: "sess-1",
		Status:    "running",
		StartedAt: "2025-01-01T12:00:00Z",
		Steps: []StepState{
			{StepID: "step-1", Tool: "read_file", Status: "done"},
		},
		Metadata: map[string]any{"key": "value"},
	}
	if err := c.Save(state); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := c.Load("sess-1")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.SessionID != "sess-1" {
		t.Fatalf("unexpected session id: %q", got.SessionID)
	}
	if got.Status != "running" {
		t.Fatalf("unexpected status: %q", got.Status)
	}
	if got.UpdatedAt == "" {
		t.Fatalf("UpdatedAt should be set")
	}
	if len(got.Steps) != 1 || got.Steps[0].StepID != "step-1" {
		t.Fatalf("steps not preserved: %+v", got.Steps)
	}
}

// TestSessionCheckpointSaveEmptyID covers the
// missing-id branch.
func TestSessionCheckpointSaveEmptyID(t *testing.T) {
	c := newTestCheckpoint(t, 0)
	err := c.Save(SessionState{})
	if err == nil {
		t.Fatalf("expected error")
	}
	var ce *CheckpointError
	if !errors.As(err, &ce) {
		t.Fatalf("expected *CheckpointError, got %T", err)
	}
}

// TestSessionCheckpointLoadNotFound covers the
// not-found branch on Load.
func TestSessionCheckpointLoadNotFound(t *testing.T) {
	c := newTestCheckpoint(t, 0)
	_, err := c.Load("missing")
	if !errors.Is(err, &SessionNotFoundError{}) {
		t.Fatalf("expected *SessionNotFoundError, got %T", err)
	}
}

// TestSessionCheckpointLoadEmptyID covers the empty-id
// branch on Load.
func TestSessionCheckpointLoadEmptyID(t *testing.T) {
	c := newTestCheckpoint(t, 0)
	_, err := c.Load("")
	if err == nil {
		t.Fatalf("expected error")
	}
}

// TestSessionCheckpointDelete covers the delete path.
// The session is removed and a subsequent Load raises
// SessionNotFoundError.
func TestSessionCheckpointDelete(t *testing.T) {
	c := newTestCheckpoint(t, 0)
	state := SessionState{SessionID: "sess-1", Status: "running"}
	if err := c.Save(state); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := c.Delete("sess-1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := c.Load("sess-1"); !errors.Is(err, &SessionNotFoundError{}) {
		t.Fatalf("expected not found after delete, got %v", err)
	}
}

// TestSessionCheckpointDeleteNotFound covers the
// not-found branch on Delete.
func TestSessionCheckpointDeleteNotFound(t *testing.T) {
	c := newTestCheckpoint(t, 0)
	err := c.Delete("missing")
	if !errors.Is(err, &SessionNotFoundError{}) {
		t.Fatalf("expected *SessionNotFoundError, got %T", err)
	}
}

// TestSessionCheckpointList covers the list path. The
// function returns the sessions sorted by UpdatedAt
// descending.
func TestSessionCheckpointList(t *testing.T) {
	c := newTestCheckpoint(t, 0)
	for _, id := range []string{"a", "b", "c"} {
		if err := c.Save(SessionState{SessionID: id, Status: "running"}); err != nil {
			t.Fatalf("Save %s: %v", id, err)
		}
	}
	list, err := c.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 3 {
		t.Fatalf("expected 3 sessions, got %d", len(list))
	}
	ids := make([]string, len(list))
	for i, s := range list {
		ids[i] = s.SessionID
	}
	// All UpdatedAt are equal in the test (fixed clock),
	// so the order between them is by Go's stable sort.
	// The important property: every id is present.
	for _, id := range []string{"a", "b", "c"} {
		found := false
		for _, got := range ids {
			if got == id {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("missing %q in %v", id, ids)
		}
	}
}

// TestSessionCheckpointCompressAboveThreshold covers the
// gzip path. When the JSON is above the threshold, the
// file is gzipped and Load transparently decompresses.
func TestSessionCheckpointCompressAboveThreshold(t *testing.T) {
	c := newTestCheckpoint(t, 1024) // 1 KB threshold
	big := strings.Repeat("a", 4096)
	state := SessionState{
		SessionID: "sess-big",
		Status:    "running",
		Metadata:  map[string]any{"data": big},
	}
	if err := c.Save(state); err != nil {
		t.Fatalf("Save: %v", err)
	}
	gzPath := c.pathForGzip("sess-big")
	if _, err := os.Stat(gzPath); err != nil {
		t.Fatalf("expected gzipped file at %s: %v", gzPath, err)
	}
	plainPath := c.pathForPlain("sess-big")
	if _, err := os.Stat(plainPath); err == nil {
		t.Fatalf("expected no plain file at %s", plainPath)
	}
	got, err := c.Load("sess-big")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.SessionID != "sess-big" {
		t.Fatalf("unexpected id: %q", got.SessionID)
	}
	if v, ok := got.Metadata["data"].(string); !ok || v != big {
		t.Fatalf("metadata not preserved")
	}
}

// TestSessionCheckpointCompressDefaultThreshold covers
// the default 1 MB threshold. The function uses a
// threshold of 0 to mean "default".
func TestSessionCheckpointCompressDefaultThreshold(t *testing.T) {
	c := newTestCheckpoint(t, 0)
	if c.compressAbove != DefaultCompressAbove {
		t.Fatalf("expected default threshold %d, got %d", DefaultCompressAbove, c.compressAbove)
	}
}

// TestSessionCheckpointDeleteBothForms covers the
// delete path when both the plain and gzipped files
// exist. The function removes both.
func TestSessionCheckpointDeleteBothForms(t *testing.T) {
	c := newTestCheckpoint(t, 1024)
	big := strings.Repeat("a", 4096)
	if err := c.Save(SessionState{SessionID: "sess", Metadata: map[string]any{"k": big}}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	// Now overwrite with a small payload so we get a
	// plain file too.
	if err := c.Save(SessionState{SessionID: "sess"}); err != nil {
		t.Fatalf("Save small: %v", err)
	}
	if err := c.Delete("sess"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := c.Load("sess"); !errors.Is(err, &SessionNotFoundError{}) {
		t.Fatalf("expected not found, got %v", err)
	}
}

// TestSessionCheckpointCompressFileIsValid covers the
// gzip file format. The file is a valid gzip stream.
func TestSessionCheckpointCompressFileIsValid(t *testing.T) {
	c := newTestCheckpoint(t, 1024)
	big := strings.Repeat("a", 4096)
	if err := c.Save(SessionState{SessionID: "sess", Metadata: map[string]any{"k": big}}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	raw, err := os.ReadFile(c.pathForGzip("sess"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	gr, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("gzip reader: %v", err)
	}
	defer gr.Close()
	decoded, err := io.ReadAll(gr)
	if err != nil {
		t.Fatalf("gzip read: %v", err)
	}
	var s SessionState
	if err := json.Unmarshal(decoded, &s); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if s.SessionID != "sess" {
		t.Fatalf("unexpected id: %q", s.SessionID)
	}
}

// TestSessionCheckpointListSkipsCorrupt covers the
// list-with-corrupt-file branch. A corrupt file is
// skipped, not fatal.
func TestSessionCheckpointListSkipsCorrupt(t *testing.T) {
	c := newTestCheckpoint(t, 0)
	if err := c.Save(SessionState{SessionID: "good", Status: "running"}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	// Drop a non-JSON file in the directory.
	if err := os.WriteFile(filepath.Join(c.path, "bad.json"), []byte("not json"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	list, err := c.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 1 || list[0].SessionID != "good" {
		t.Fatalf("expected only 'good', got %+v", list)
	}
}

// TestSessionCheckpointListSkipsUnknown covers the
// list-with-unknown-filename branch. A file whose name
// does not match the .json[.gz] pattern is skipped.
func TestSessionCheckpointListSkipsUnknown(t *testing.T) {
	c := newTestCheckpoint(t, 0)
	if err := c.Save(SessionState{SessionID: "good", Status: "running"}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := os.WriteFile(filepath.Join(c.path, "README.md"), []byte("#"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	list, err := c.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1, got %d", len(list))
	}
}

// TestSessionCheckpointPathCovers the Path accessor.
func TestSessionCheckpointPathCovers(t *testing.T) {
	c := newTestCheckpoint(t, 0)
	if c.Path() == "" {
		t.Fatalf("expected non-empty path")
	}
}

// TestSessionCheckpointUpdatedAt is set on every save.
func TestSessionCheckpointUpdatedAt(t *testing.T) {
	c := newTestCheckpoint(t, 0)
	if err := c.Save(SessionState{SessionID: "s", Status: "running"}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := c.Load("s")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.UpdatedAt == "" {
		t.Fatalf("UpdatedAt should be set")
	}
}

// TestSessionCheckpointInitMissingDir covers the
// mkdir branch. The function creates the directory if
// missing.
func TestSessionCheckpointInitMissingDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "sessions")
	c, err := NewSessionCheckpoint(SessionCheckpointConfig{Path: dir})
	if err != nil {
		t.Fatalf("NewSessionCheckpoint: %v", err)
	}
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("expected dir to exist: %v", err)
	}
	if c.Path() != dir {
		t.Fatalf("unexpected path: %q", c.Path())
	}
}

// TestSessionNotFoundErrorIs covers the errors.Is match
// for the typed error.
func TestSessionNotFoundErrorIs(t *testing.T) {
	var err error = &SessionNotFoundError{ID: "x"}
	if !errors.Is(err, &SessionNotFoundError{}) {
		t.Fatalf("expected errors.Is to match")
	}
	if msg := err.Error(); !strings.Contains(msg, "x") {
		t.Fatalf("expected id in message: %q", msg)
	}
}

// TestCheckpointErrorMessage covers the error format.
func TestCheckpointErrorMessage(t *testing.T) {
	err := (&CheckpointError{Op: "save", Reason: "boom"}).Error()
	if !strings.HasPrefix(err, "persist: checkpoint:") {
		t.Fatalf("unexpected prefix: %q", err)
	}
	if !strings.HasSuffix(err, ".") {
		t.Fatalf("missing trailing period: %q", err)
	}
}

// TestSessionIDFromFilename covers the filename parser.
func TestSessionIDFromFilename(t *testing.T) {
	cases := []struct {
		name string
		in   string
		id   string
		ok   bool
	}{
		{"plain", "sess-1.json", "sess-1", true},
		{"gz", "sess-1.json.gz", "sess-1", true},
		{"no ext", "sess-1", "", false},
		{"wrong ext", "sess-1.txt", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			id, ok := sessionIDFromFilename(tc.in)
			if id != tc.id || ok != tc.ok {
				t.Fatalf("got (%q, %v) want (%q, %v)", id, ok, tc.id, tc.ok)
			}
		})
	}
}

// TestDefaultSessionsPath covers the default-path
// resolver.
func TestDefaultSessionsPath(t *testing.T) {
	t.Setenv("AIOS_DATA_DIR", "")
	got := defaultSessionsPath()
	if !strings.HasSuffix(got, "sessions") {
		t.Fatalf("expected suffix 'sessions', got %q", got)
	}
	t.Setenv("AIOS_DATA_DIR", "/custom/data")
	if got := defaultSessionsPath(); got != filepath.Join("/custom/data", "sessions") {
		t.Fatalf("AIOS_DATA_DIR not honored: %q", got)
	}
}
