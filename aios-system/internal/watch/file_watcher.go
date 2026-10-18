package watch

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/sadi/aios-system/internal/core/audit"
	"github.com/sadi/aios-system/internal/search"
	"github.com/sadi/aios-system/internal/tools"
)

// EventType is the kind of file-system event the watcher emits
// (REQ-WTC-DAT-001).
type EventType string

const (
	EventCreated  EventType = "created"
	EventModified EventType = "modified"
	EventDeleted  EventType = "deleted"
	EventMoved    EventType = "moved"
)

// WatchEvent is the typed record for a file-system event. The
// schema matches REQ-WTC-DAT-001.
type WatchEvent struct {
	Path      string    `json:"path"`
	EventType EventType `json:"event_type"`
	Timestamp string    `json:"timestamp"`
}

// FileWatcherTool watches a directory for files matching a
// pattern (REQ-WTC-FWT-001 to 006).
//
// The watcher is a polling loop: every Interval it snapshots
// the directory, diffs against the previous snapshot, and
// appends the changes to the result. Polling keeps the
// watcher deterministic in (filesystem, clock) — the same
// as a fsnotify-based watcher but with no third-party
// dependency.
//
// The watcher respects the .aiosignore patterns loaded by
// the aios-search package (REQ-WTC-FWT-005).
//
// The tool writes one "file_watch" audit row per call
// (REQ-WTC-FWT-006) when an audit log is configured.
type FileWatcherTool struct {
	// Interval is the polling interval. Zero or negative
	// means 1 second.
	Interval time.Duration
	// Timeout caps the watcher's lifetime. Zero means
	// the caller is responsible for cancelling the
	// context.
	Timeout time.Duration
	// Audit is the audit log. nil means "skip the row".
	Audit *audit.Log
	// Now is the wall clock. nil defaults to time.Now.
	Now func() time.Time
}

func (FileWatcherTool) Name() string { return "file_watcher" }

// snapshot maps a slash-separated relative path to (size,
// mtime). The diff is a pure function of two snapshots.
type snapshot map[string]snapEntry

type snapEntry struct {
	size  int64
	mtime time.Time
}

// Run executes the watch (REQ-WTC-FWT-002 to 004).
//
// Args:
//
//	path     (string, required) — directory to watch.
//	pattern  (string, required) — basename glob the path
//	                            must match, e.g. "*.go".
//	max      (int,    optional) — cap on the events list
//	                            (default 1000).
//
// The watcher runs until the timeout fires (if set). It
// returns the events it saw during the run, in timestamp
// order.
func (t *FileWatcherTool) Run(args map[string]any, ctx tools.Context) (tools.Result, error) {
	start := t.now().UTC()
	path, _ := args["path"].(string)
	if path == "" {
		return tools.Result{OK: false, Summary: "missing path"},
			&WatchError{Op: "run", Reason: "path is required"}
	}
	pattern, _ := args["pattern"].(string)
	if pattern == "" {
		return tools.Result{OK: false, Summary: "missing pattern"},
			&WatchError{Op: "run", Reason: "pattern is required"}
	}
	max := 1000
	if v, ok := args["max"].(int); ok && v > 0 {
		max = v
	}

	// Resolve path against the workspace root.
	ws := ctx.Workspace
	if ws == "" {
		var err error
		ws, err = tools.Workspace()
		if err != nil {
			return tools.Result{OK: false, Summary: "no workspace"}, err
		}
	}
	root, err := filepath.Abs(filepath.Join(ws, path))
	if err != nil {
		return tools.Result{OK: false, Summary: "abs failed"}, &WatchError{Op: "run", Path: path, Reason: "abs", Err: err}
	}
	info, err := os.Stat(root)
	if err != nil {
		return tools.Result{OK: false, Summary: "stat failed"}, &WatchError{Op: "run", Path: root, Reason: "stat", Err: err}
	}
	if !info.IsDir() {
		return tools.Result{OK: false, Summary: "not a directory"}, &WatchError{Op: "run", Path: root, Reason: "not a directory"}
	}

	ig := search.LoadIgnore(root)
	runCtx := context.Background()
	if t.Timeout > 0 {
		var cancel context.CancelFunc
		runCtx, cancel = context.WithTimeout(runCtx, t.Timeout)
		defer cancel()
	}

	events, err := t.poll(runCtx, root, pattern, max, ig)
	if err != nil {
		return tools.Result{OK: false, Summary: "watch failed"}, &WatchError{Op: "run", Path: root, Reason: "poll", Err: err}
	}
	sort.Slice(events, func(i, j int) bool { return events[i].Timestamp < events[j].Timestamp })

	if t.Audit != nil {
		t.writeAudit(ctx, root, pattern, start, len(events))
	}

	return tools.Result{
		OK: true,
		Output: map[string]any{
			"path":   root,
			"events": events,
			"count":  len(events),
		},
		Summary: fmt.Sprintf("watched %s, %d events", root, len(events)),
	}, nil
}

// poll runs the polling loop. Each tick builds a fresh
// snapshot, diffs it against the previous one, and emits the
// events.
func (t *FileWatcherTool) poll(ctx context.Context, root, pattern string, max int, ig search.IgnorePatterns) ([]WatchEvent, error) {
	tick := time.NewTicker(t.interval())
	defer tick.Stop()

	prev := scan(root, pattern, ig)
	var events []WatchEvent
	for {
		select {
		case <-ctx.Done():
			return events, nil
		case <-tick.C:
		}
		cur := scan(root, pattern, ig)
		for _, ev := range diff(prev, cur, t.now().UTC()) {
			events = append(events, ev)
			if len(events) >= max {
				return events, nil
			}
		}
		prev = cur
	}
}

// scan builds a snapshot of root. The snapshot is keyed by
// the slash-separated relative path; only entries that match
// pattern and pass the ignore filter are recorded.
func scan(root, pattern string, ig search.IgnorePatterns) snapshot {
	out := snapshot{}
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(root, p)
		rel = filepath.ToSlash(rel)
		if ig.Match(rel) {
			return nil
		}
		if ok, _ := filepath.Match(pattern, filepath.Base(rel)); !ok {
			return nil
		}
		fi, err := d.Info()
		if err != nil {
			return nil
		}
		out[rel] = snapEntry{size: fi.Size(), mtime: fi.ModTime().UTC()}
		return nil
	})
	return out
}

// diff returns the creates, modifies, and deletes between two
// snapshots. A new key is a create; a changed size or mtime
// is a modify; a missing key is a delete.
func diff(prev, cur snapshot, now time.Time) []WatchEvent {
	ts := now.Format(time.RFC3339Nano)
	var out []WatchEvent
	for path, c := range cur {
		p, ok := prev[path]
		switch {
		case !ok:
			out = append(out, WatchEvent{Path: path, EventType: EventCreated, Timestamp: ts})
		case p.size != c.size || !p.mtime.Equal(c.mtime):
			out = append(out, WatchEvent{Path: path, EventType: EventModified, Timestamp: ts})
		}
	}
	for path := range prev {
		if _, ok := cur[path]; !ok {
			out = append(out, WatchEvent{Path: path, EventType: EventDeleted, Timestamp: ts})
		}
	}
	return out
}

// interval returns the configured interval, or 1 second.
func (t *FileWatcherTool) interval() time.Duration {
	if t.Interval <= 0 {
		return time.Second
	}
	return t.Interval
}

// now returns the configured clock or time.Now.
func (t *FileWatcherTool) now() time.Time {
	if t.Now != nil {
		return t.Now()
	}
	return time.Now()
}

// writeAudit writes a "file_watch" audit row
// (REQ-WTC-FWT-006). Errors are reported but do not fail the
// call: the events are already in hand.
func (t *FileWatcherTool) writeAudit(ctx tools.Context, path, pattern string, start time.Time, count int) {
	taskID := ctx.TaskID
	if taskID == "" {
		taskID = "watcher"
	}
	actor := ctx.Actor
	if actor == "" {
		actor = "watcher"
	}
	dur := t.now().UTC().Sub(start)
	_, _ = t.Audit.Write(&audit.Row{
		TaskID:   taskID,
		Kind:     "file_watch",
		Actor:    actor,
		RiskTier: 0,
		Payload: map[string]any{
			"path":        path,
			"pattern":     pattern,
			"event_count": count,
			"duration_ms": dur.Milliseconds(),
		},
	})
}
