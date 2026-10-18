package memory

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// Index is the typed indexed layer.
//
// The index categorizes every artifact by kind, scope,
// session id, task id, timestamp, and tags. The index
// is keyed by content hash; the value is the index
// entry.
//
// Per REQ-MEM-IDX-005 the index is persisted as a JSON
// object. The default file is <path>/index.json. The
// file is rewritten on every Add call through a temp
// file in the same directory and a rename (atomic
// save).
type Index struct {
	mu      sync.RWMutex
	path    string
	entries map[string]IndexEntry
	dirty   bool
	now     func() time.Time
}

// IndexEntry is the typed index entry. The entry
// is what the index stores per artifact; the artifact
// itself is in the raw layer.
//
// The struct is a flattened view of the artifact: the
// fields the query language and the retrieval
// algorithm need at lookup time. The Content field is
// omitted because the raw layer owns it; the
// ContentHash is the join key.
type IndexEntry struct {
	Kind        ArtifactKind `json:"kind"`
	Scope       Scope        `json:"scope"`
	SessionID   string       `json:"session_id"`
	TaskID      string       `json:"task_id,omitempty"`
	Timestamp   string       `json:"timestamp"`
	Tags        []string     `json:"tags,omitempty"`
	ContentHash string       `json:"content_hash"`
	// Pruned marks the artifact as pruned. The raw
	// layer is unchanged; the retrieval and
	// compaction exclude pruned artifacts. Per
	// REQ-MEM-CLI-005 the developer can
	// `aios memory unprune` to restore.
	Pruned bool `json:"pruned,omitempty"`
}

// IndexConfig controls Index construction.
type IndexConfig struct {
	// Path is the directory under which the index
	// file is stored. The file is <path>/index.json.
	// Required.
	Path string
	// Now is the decision clock. Defaults to
	// time.Now when nil. The clock is reserved for
	// future use; the current index does not stamp
	// entries on Add (the artifact's Timestamp is
	// the canonical time).
	Now func() time.Time
}

// NewIndex constructs an Index and loads the on-disk
// file when one exists. A missing file is treated as
// an empty index, not as an error.
func NewIndex(cfg IndexConfig) (*Index, error) {
	if cfg.Path == "" {
		// In-memory index. The session scope uses
		// this mode; the data is gone on Close.
		idx := &Index{
			path:    "",
			entries: make(map[string]IndexEntry),
			now:     cfg.Now,
		}
		if idx.now == nil {
			idx.now = time.Now
		}
		return idx, nil
	}
	if err := os.MkdirAll(cfg.Path, 0o755); err != nil {
		return nil, &MemoryError{Op: "init", Reason: err.Error()}
	}
	idx := &Index{
		path:    cfg.Path,
		entries: make(map[string]IndexEntry),
		now:     cfg.Now,
	}
	if idx.now == nil {
		idx.now = time.Now
	}
	if err := idx.loadFromDisk(); err != nil {
		return nil, err
	}
	return idx, nil
}

// Add indexes an artifact. The method is idempotent:
// re-adding the same artifact overwrites the entry.
// The method does not touch the raw layer; the caller
// is expected to put the payload in the store first.
//
// Per REQ-MEM-IDX-003 the index is keyed by content
// hash. The add is followed by a Save so the on-disk
// file is always in sync.
func (i *Index) Add(a Artifact) error {
	if err := a.Validate(); err != nil {
		return err
	}
	entry := IndexEntry{
		Kind:        a.Kind,
		Scope:       a.Scope,
		SessionID:   a.SessionID,
		TaskID:      a.TaskID,
		Timestamp:   a.Timestamp,
		Tags:        append([]string(nil), a.Tags...),
		ContentHash: a.ContentHash,
	}
	i.mu.Lock()
	defer i.mu.Unlock()
	i.entries[a.ContentHash] = entry
	i.dirty = true
	return i.saveLocked()
}

// Get returns the index entry for a content hash.
// The method returns (IndexEntry{}, ErrNotFound)
// when the hash is not indexed.
func (i *Index) Get(hash string) (IndexEntry, error) {
	if hash == "" {
		return IndexEntry{}, &MemoryError{Op: "get", Reason: "hash is required"}
	}
	i.mu.RLock()
	defer i.mu.RUnlock()
	entry, ok := i.entries[hash]
	if !ok {
		return IndexEntry{}, ErrNotFound
	}
	return entry, nil
}

// Has reports whether the index contains the hash.
func (i *Index) Has(hash string) bool {
	if hash == "" {
		return false
	}
	i.mu.RLock()
	defer i.mu.RUnlock()
	_, ok := i.entries[hash]
	return ok
}

// Prune marks the artifact as pruned. The raw layer
// is unchanged; the next retrieval and compaction
// exclude the artifact. Per REQ-MEM-CLI-005.
func (i *Index) Prune(hash string) error {
	if hash == "" {
		return &MemoryError{Op: "prune", Reason: "hash is required"}
	}
	i.mu.Lock()
	defer i.mu.Unlock()
	entry, ok := i.entries[hash]
	if !ok {
		return ErrNotFound
	}
	if entry.Pruned {
		return nil
	}
	entry.Pruned = true
	i.entries[hash] = entry
	i.dirty = true
	return i.saveLocked()
}

// Unprune restores a pruned artifact.
func (i *Index) Unprune(hash string) error {
	if hash == "" {
		return &MemoryError{Op: "unprune", Reason: "hash is required"}
	}
	i.mu.Lock()
	defer i.mu.Unlock()
	entry, ok := i.entries[hash]
	if !ok {
		return ErrNotFound
	}
	if !entry.Pruned {
		return nil
	}
	entry.Pruned = false
	i.entries[hash] = entry
	i.dirty = true
	return i.saveLocked()
}

// Size returns the number of indexed entries,
// pruned entries included.
func (i *Index) Size() int {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return len(i.entries)
}

// PrunedCount returns the number of pruned entries.
func (i *Index) PrunedCount() int {
	i.mu.RLock()
	defer i.mu.RUnlock()
	n := 0
	for _, e := range i.entries {
		if e.Pruned {
			n++
		}
	}
	return n
}

// All returns a copy of every indexed entry, sorted
// by content hash. The function is the bulk export
// the retrieval algorithm uses.
func (i *Index) All() []IndexEntry {
	i.mu.RLock()
	defer i.mu.RUnlock()
	out := make([]IndexEntry, 0, len(i.entries))
	for _, e := range i.entries {
		out = append(out, e)
	}
	sort.Slice(out, func(a, b int) bool {
		return out[a].ContentHash < out[b].ContentHash
	})
	return out
}

// Match reports whether the entry satisfies the
// query. The function is the per-entry predicate
// the query language uses.
//
// The function returns true when the entry matches
// every active filter under the AND operator, and
// true when at least one active filter matches under
// the OR operator. Pruned entries never match; the
// retrieval and compaction exclude them.
func (e IndexEntry) Match(q Query) bool {
	if e.Pruned {
		return false
	}
	op := q.Operator
	if op == "" {
		op = OpAND
	}
	if op == OpAND {
		return e.matchKinds(q) && e.matchScopes(q) && e.matchTags(q) && e.matchPattern(q) && e.matchSince(q)
	}
	// OR: at least one active filter must match.
	anyMatched := false
	if len(q.Kinds) > 0 {
		if e.matchKinds(q) {
			anyMatched = true
		}
	}
	if !anyMatched && len(q.Scopes) > 0 && e.matchScopes(q) {
		anyMatched = true
	}
	if !anyMatched && len(q.Tags) > 0 && e.matchTags(q) {
		anyMatched = true
	}
	if !anyMatched && q.Pattern != "" && e.matchPattern(q) {
		anyMatched = true
	}
	if !anyMatched && q.Since != "" && e.matchSince(q) {
		anyMatched = true
	}
	// When no filters are set, OR returns true.
	if len(q.Kinds) == 0 && len(q.Scopes) == 0 && len(q.Tags) == 0 && q.Pattern == "" && q.Since == "" {
		return true
	}
	return anyMatched
}

// matchKinds reports whether the entry's kind is in
// the query's kinds list. An empty kinds list
// matches every kind.
func (e IndexEntry) matchKinds(q Query) bool {
	if len(q.Kinds) == 0 {
		return true
	}
	for _, k := range q.Kinds {
		if string(e.Kind) == k {
			return true
		}
	}
	return false
}

// matchScopes reports whether the entry's scope is
// in the query's scopes list. An empty scopes list
// matches every scope.
func (e IndexEntry) matchScopes(q Query) bool {
	if len(q.Scopes) == 0 {
		return true
	}
	for _, s := range q.Scopes {
		if string(e.Scope) == s {
			return true
		}
	}
	return false
}

// matchTags reports whether the entry shares at
// least one tag with the query's tags list. An
// empty tags list matches every entry.
func (e IndexEntry) matchTags(q Query) bool {
	if len(q.Tags) == 0 {
		return true
	}
	for _, qt := range q.Tags {
		for _, et := range e.Tags {
			if qt == et {
				return true
			}
		}
	}
	return false
}

// matchPattern reports whether the entry's content
// matches the query's glob pattern. The match uses
// the content hash as the input when the raw layer
// is unavailable (e.g. during a test or replay
// without the store).
//
// The function is the cheap path. Callers that have
// the artifact should call Artifact.ContentMatches.
func (e IndexEntry) matchPattern(q Query) bool {
	if q.Pattern == "" {
		return true
	}
	return matchGlob(q.Pattern, e.ContentHash)
}

// matchSince reports whether the entry's timestamp
// is at or after the query's since timestamp.
func (e IndexEntry) matchSince(q Query) bool {
	if q.Since == "" {
		return true
	}
	at, ok := parseTimestamp(e.Timestamp)
	if !ok {
		return false
	}
	bt, ok := parseTimestamp(q.Since)
	if !ok {
		return false
	}
	return !at.Before(bt)
}

// loadFromDisk reads the index file. A missing file
// is treated as an empty index. An unparseable file
// returns ErrIndexCorrupt.
func (i *Index) loadFromDisk() error {
	raw, err := os.ReadFile(i.filePath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return &MemoryError{Op: "load", Reason: err.Error()}
	}
	if len(raw) == 0 {
		return nil
	}
	entries := make(map[string]IndexEntry)
	if err := json.Unmarshal(raw, &entries); err != nil {
		return fmt.Errorf("%w: %v", ErrIndexCorrupt, err)
	}
	for hash, e := range entries {
		if hash == "" {
			return fmt.Errorf("%w: empty hash", ErrIndexCorrupt)
		}
		if !e.Kind.IsValid() {
			return fmt.Errorf("%w: invalid kind %q for %s", ErrIndexCorrupt, e.Kind, hash)
		}
		if !e.Scope.IsValid() {
			return fmt.Errorf("%w: invalid scope %q for %s", ErrIndexCorrupt, e.Scope, hash)
		}
	}
	i.entries = entries
	i.dirty = false
	return nil
}

// saveLocked writes the index to disk. The function
// is called with the index's mutex held.
//
// The save is atomic through a temp file in the
// same directory and a rename. The parent directory
// is created if missing.
func (i *Index) saveLocked() error {
	if !i.dirty {
		return nil
	}
	if i.path == "" {
		// In-memory index; nothing to persist.
		i.dirty = false
		return nil
	}
	raw, err := json.MarshalIndent(i.entries, "", "  ")
	if err != nil {
		return &MemoryError{Op: "save", Reason: err.Error()}
	}
	final := i.filePath()
	tmp, err := os.CreateTemp(i.path, "index.json.tmp-*")
	if err != nil {
		return &MemoryError{Op: "save", Reason: err.Error()}
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op if rename succeeded
	if _, err := tmp.Write(raw); err != nil {
		tmp.Close()
		return &MemoryError{Op: "save", Reason: err.Error()}
	}
	if err := tmp.Close(); err != nil {
		return &MemoryError{Op: "save", Reason: err.Error()}
	}
	if err := os.Rename(tmpName, final); err != nil {
		return &MemoryError{Op: "save", Reason: err.Error()}
	}
	i.dirty = false
	return nil
}

// filePath returns the index file path.
func (i *Index) filePath() string {
	return filepath.Join(i.path, "index.json")
}

// Path returns the index's directory. The function
// is the path getter for tests and the stats output.
func (i *Index) Path() string { return i.path }

// ID returns the content-addressed id of the current
// index. The id is the SHA-256 of the index's
// canonical JSON. The id changes when an entry is
// added or removed.
//
// Per REQ-MEM-DAT-003 the CompactedView's IndexID
// field is this id.
func (i *Index) ID() string {
	i.mu.RLock()
	defer i.mu.RUnlock()
	raw, err := json.Marshal(i.entries)
	if err != nil {
		return ""
	}
	return HashBytes(raw)
}

// matchGlob reports whether the pattern matches the
// input. The function is filepath.Match with a
// empty-pattern special case. Errors are returned as
// false.
func matchGlob(pattern, input string) bool {
	ok, err := filepath.Match(pattern, input)
	if err != nil {
		return false
	}
	return ok
}
