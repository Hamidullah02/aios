package watch

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/sadi/aios-system/internal/core/audit"
	"github.com/sadi/aios-system/internal/core/clock"
	"github.com/sadi/aios-system/internal/tools"
)

// auditRows reads every audit row back from the log. We use
// this to assert the watcher wrote the right kind.
func auditRows(t *testing.T, log *audit.Log) []*audit.Row {
	t.Helper()
	persisted, err := log.GetByKind("file_watch", 0)
	if err != nil {
		t.Fatal(err)
	}
	out := make([]*audit.Row, len(persisted))
	for i := range persisted {
		r := persisted[i]
		out[i] = &audit.Row{
			TaskID: r.TaskID,
			Kind:   r.Kind,
			Actor:  r.Actor,
		}
	}
	return out
}

func openTestAudit(t *testing.T) *audit.Log {
	t.Helper()
	log, err := audit.Open(audit.Config{Path: ":memory:", Clock: clock.FixedDecision{}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { log.Close() })
	return log
}

func writeFile(t *testing.T, dir, name, body string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// TestFileWatcher_MissingPath covers the validation path.
func TestFileWatcher_MissingPath(t *testing.T) {
	w := &FileWatcherTool{}
	res, err := w.Run(map[string]any{"pattern": "*.go"}, tools.Context{Workspace: t.TempDir()})
	if err == nil {
		t.Fatal("expected error")
	}
	if res.OK {
		t.Fatal("expected !OK")
	}
	if !errIsWatch(err) {
		t.Fatalf("err = %v, want WatchError", err)
	}
}

// TestFileWatcher_MissingPattern covers the validation path.
func TestFileWatcher_MissingPattern(t *testing.T) {
	w := &FileWatcherTool{}
	_, err := w.Run(map[string]any{"path": "."}, tools.Context{Workspace: t.TempDir()})
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestFileWatcher_DetectsCreate covers REQ-WTC-FWT-003.
func TestFileWatcher_DetectsCreate(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.go", "package x\n")
	w := &FileWatcherTool{Interval: 5 * time.Millisecond, Timeout: 300 * time.Millisecond, Audit: openTestAudit(t)}
	// Schedule a create mid-run.
	time.AfterFunc(80*time.Millisecond, func() {
		writeFile(t, dir, "b.go", "package x\n")
	})
	res, err := w.Run(map[string]any{
		"path": ".", "pattern": "*.go",
	}, tools.Context{Workspace: dir, TaskID: "task-1"})
	if err != nil {
		t.Fatal(err)
	}
	events, _ := res.Output["events"].([]WatchEvent)
	if len(events) < 1 {
		t.Fatalf("expected at least 1 event, got %d (%+v)", len(events), events)
	}
	if events[0].EventType != EventCreated || events[0].Path != "b.go" {
		t.Fatalf("unexpected events: %+v", events)
	}
}

// TestFileWatcher_DetectsModify covers the modify path.
func TestFileWatcher_DetectsModify(t *testing.T) {
	dir := t.TempDir()
	p := writeFile(t, dir, "a.go", "v1")
	w := &FileWatcherTool{Interval: 5 * time.Millisecond, Timeout: 300 * time.Millisecond}
	time.AfterFunc(80*time.Millisecond, func() {
		// Write the file again to bump mtime (Windows
		// mtime resolution can be 1s; the new file
		// also gets a slightly later mtime).
		if err := os.WriteFile(p, []byte("v2"), 0o644); err != nil {
			t.Errorf("write: %v", err)
		}
	})
	res, err := w.Run(map[string]any{"path": ".", "pattern": "*.go"}, tools.Context{Workspace: dir})
	if err != nil {
		t.Fatal(err)
	}
	events, _ := res.Output["events"].([]WatchEvent)
	if len(events) < 1 || events[0].EventType != EventModified {
		t.Fatalf("events = %+v", events)
	}
}

// TestFileWatcher_DetectsDelete covers the delete path.
func TestFileWatcher_DetectsDelete(t *testing.T) {
	dir := t.TempDir()
	p := writeFile(t, dir, "a.go", "x")
	w := &FileWatcherTool{Interval: 5 * time.Millisecond, Timeout: 300 * time.Millisecond}
	time.AfterFunc(80*time.Millisecond, func() {
		if err := os.Remove(p); err != nil {
			t.Errorf("remove: %v", err)
		}
	})
	res, err := w.Run(map[string]any{"path": ".", "pattern": "*.go"}, tools.Context{Workspace: dir})
	if err != nil {
		t.Fatal(err)
	}
	events, _ := res.Output["events"].([]WatchEvent)
	if len(events) < 1 || events[0].EventType != EventDeleted {
		t.Fatalf("events = %+v", events)
	}
}

// TestFileWatcher_RespectsAiosignore covers REQ-WTC-FWT-005.
func TestFileWatcher_RespectsAiosignore(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".aiosignore"), []byte("skip.go\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	writeFile(t, dir, "keep.go", "x")
	writeFile(t, dir, "skip.go", "x")
	w := &FileWatcherTool{Interval: 5 * time.Millisecond, Timeout: 300 * time.Millisecond}
	// Add a "skip" file mid-run; the watcher must drop it.
	time.AfterFunc(80*time.Millisecond, func() {
		writeFile(t, dir, "skip.go", "y")
		writeFile(t, dir, "another.go", "z")
	})
	res, err := w.Run(map[string]any{"path": ".", "pattern": "*.go"}, tools.Context{Workspace: dir})
	if err != nil {
		t.Fatal(err)
	}
	events, _ := res.Output["events"].([]WatchEvent)
	sawAnother := false
	for _, e := range events {
		if filepath.Base(e.Path) == "skip.go" {
			t.Fatalf("skip.go should have been ignored: %+v", e)
		}
		if filepath.Base(e.Path) == "another.go" {
			sawAnother = true
		}
	}
	if !sawAnother {
		t.Fatalf("expected to see another.go in events: %+v", events)
	}
}

// TestFileWatcher_AuditRow covers REQ-WTC-FWT-006.
func TestFileWatcher_AuditRow(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.go", "x")
	log := openTestAudit(t)
	w := &FileWatcherTool{Interval: 5 * time.Millisecond, Timeout: 50 * time.Millisecond, Audit: log}
	if _, err := w.Run(map[string]any{"path": ".", "pattern": "*.go"},
		tools.Context{Workspace: dir, TaskID: "task-x", Actor: "alice"}); err != nil {
		t.Fatal(err)
	}
	rows := auditRows(t, log)
	if len(rows) != 1 {
		t.Fatalf("rows = %d", len(rows))
	}
	if rows[0].Kind != "file_watch" {
		t.Fatalf("kind = %q", rows[0].Kind)
	}
	if rows[0].TaskID != "task-x" || rows[0].Actor != "alice" {
		t.Fatalf("row = %+v", rows[0])
	}
}

// TestFileWatcher_PathNotDirectory covers the path-must-be-a-dir check.
func TestFileWatcher_PathNotDirectory(t *testing.T) {
	dir := t.TempDir()
	p := writeFile(t, dir, "afile", "x")
	w := &FileWatcherTool{}
	_, err := w.Run(map[string]any{"path": p, "pattern": "*"}, tools.Context{Workspace: dir})
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestFileWatcher_NoAuditWhenNil covers the optional-audit path.
func TestFileWatcher_NoAuditWhenNil(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.go", "x")
	w := &FileWatcherTool{Interval: 5 * time.Millisecond, Timeout: 50 * time.Millisecond}
	res, err := w.Run(map[string]any{"path": ".", "pattern": "*.go"}, tools.Context{Workspace: dir})
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK {
		t.Fatalf("res = %+v", res)
	}
}

// TestFileWatcher_EventsSorted covers the stable-ordering path.
func TestFileWatcher_EventsSorted(t *testing.T) {
	dir := t.TempDir()
	for _, n := range []string{"a.go", "b.go", "c.go"} {
		writeFile(t, dir, n, "x")
	}
	w := &FileWatcherTool{Interval: 5 * time.Millisecond, Timeout: 300 * time.Millisecond}
	time.AfterFunc(80*time.Millisecond, func() {
		writeFile(t, dir, "d.go", "x")
		writeFile(t, dir, "e.go", "x")
	})
	res, err := w.Run(map[string]any{"path": ".", "pattern": "*.go"}, tools.Context{Workspace: dir})
	if err != nil {
		t.Fatal(err)
	}
	events, _ := res.Output["events"].([]WatchEvent)
	if len(events) < 1 {
		t.Fatalf("expected events, got 0")
	}
	if !sort.SliceIsSorted(events, func(i, j int) bool { return events[i].Timestamp < events[j].Timestamp }) {
		t.Fatalf("events not sorted: %+v", events)
	}
}

// TestDiff_Unit covers the diff function directly so the
// create/modify/delete paths are tested without the polling
// loop.
func TestDiff_Unit(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	prev := snapshot{
		"a.go": {size: 1, mtime: now},
		"b.go": {size: 2, mtime: now},
	}
	cur := snapshot{
		"a.go": {size: 1, mtime: now}, // unchanged
		"b.go": {size: 3, mtime: now}, // modified
		"c.go": {size: 4, mtime: now}, // created
	}
	evs := diff(prev, cur, now)
	want := map[EventType][]string{
		EventCreated:  {"c.go"},
		EventModified: {"b.go"},
		EventDeleted:  {},
	}
	got := map[EventType][]string{}
	for _, e := range evs {
		got[e.EventType] = append(got[e.EventType], e.Path)
	}
	for k, v := range want {
		if len(v) == 0 {
			if _, ok := got[k]; ok {
				t.Fatalf("unexpected %v events: %v", k, got[k])
			}
			continue
		}
		if len(got[k]) != 1 || got[k][0] != v[0] {
			t.Fatalf("%v = %v, want %v", k, got[k], v)
		}
	}
	// Delete path: drop a key from cur.
	cur2 := snapshot{"a.go": prev["a.go"]}
	evs = diff(prev, cur2, now)
	if len(evs) != 1 || evs[0].EventType != EventDeleted {
		t.Fatalf("delete: %+v", evs)
	}
}

// TestWatchEvent_JSONSchema covers REQ-WTC-DAT-001: the JSON
// shape must have path, event_type, timestamp.
func TestWatchEvent_JSONSchema(t *testing.T) {
	e := WatchEvent{Path: "a.go", EventType: EventCreated, Timestamp: "2025-01-01T00:00:00Z"}
	data, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{`"path":"a.go"`, `"event_type":"created"`, `"timestamp":"2025-01-01T00:00:00Z"`} {
		if !contains(string(data), k) {
			t.Fatalf("missing key %s in %s", k, data)
		}
	}
}

// TestWatchError_IsCovers covers the errors.Is match.
func TestWatchError_IsCovers(t *testing.T) {
	e := &WatchError{Op: "x", Reason: "y"}
	if !errIsWatch(e) {
		t.Fatal("WatchError should match ErrWatch")
	}
	plain := error(&WatchError{Op: "x", Reason: "y", Err: errSomething{}})
	if !errIsWatch(plain) {
		t.Fatal("wrapped WatchError should still match ErrWatch")
	}
}

type errSomething struct{}

func (errSomething) Error() string { return "x" }

func errIsWatch(err error) bool {
	for err != nil {
		if we, ok := err.(*WatchError); ok {
			return we.Is(ErrWatch)
		}
		type unwrapper interface{ Unwrap() error }
		if u, ok := err.(unwrapper); ok {
			err = u.Unwrap()
			continue
		}
		break
	}
	return false
}

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

// silence unused import lints.
var _ = context.Background
