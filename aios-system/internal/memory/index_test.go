package memory

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// newTestIndex constructs an Index pointed at a
// fresh temp directory.
func newTestIndex(t *testing.T) *Index {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "index")
	idx, err := NewIndex(IndexConfig{Path: dir})
	if err != nil {
		t.Fatalf("NewIndex: %v", err)
	}
	return idx
}

// TestIndexAddIsIdempotent covers REQ-MEM-IDX-003.
// Adding the same artifact twice is a no-op
// (overwrites the entry, no duplicate).
func TestIndexAddIsIdempotent(t *testing.T) {
	idx := newTestIndex(t)
	a, _ := NewArtifact(KindTask, ScopeProject, "s1", []byte("x"))
	if err := idx.Add(a); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := idx.Add(a); err != nil {
		t.Fatalf("Add (second): %v", err)
	}
	if idx.Size() != 1 {
		t.Fatalf("Size = %d, want 1", idx.Size())
	}
}

// TestIndexGetMissing covers the ErrNotFound path.
func TestIndexGetMissing(t *testing.T) {
	idx := newTestIndex(t)
	if _, err := idx.Get(HashBytes([]byte("missing"))); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// TestIndexGetRejectsEmpty covers the empty-hash
// guard.
func TestIndexGetRejectsEmpty(t *testing.T) {
	idx := newTestIndex(t)
	if _, err := idx.Get(""); err == nil {
		t.Fatal("expected error for empty hash")
	}
}

// TestIndexPruneAndUnprune cover the prune cycle.
func TestIndexPruneAndUnprune(t *testing.T) {
	idx := newTestIndex(t)
	a, _ := NewArtifact(KindTask, ScopeProject, "s1", []byte("x"))
	_ = idx.Add(a)
	if err := idx.Prune(a.ContentHash); err != nil {
		t.Fatalf("Prune: %v", err)
	}
	if idx.PrunedCount() != 1 {
		t.Fatalf("PrunedCount = %d, want 1", idx.PrunedCount())
	}
	// Pruned entries are excluded from matches.
	q := Query{}
	matches, _ := q.RunIndexAll(idx.All())
	if len(matches) != 0 {
		t.Fatalf("expected 0 matches for pruned, got %d", len(matches))
	}
	if err := idx.Unprune(a.ContentHash); err != nil {
		t.Fatalf("Unprune: %v", err)
	}
	if idx.PrunedCount() != 0 {
		t.Fatalf("PrunedCount after unprune = %d, want 0", idx.PrunedCount())
	}
	matches, _ = q.RunIndexAll(idx.All())
	if len(matches) != 1 {
		t.Fatalf("expected 1 match after unprune, got %d", len(matches))
	}
}

// TestIndexSaveReload covers the atomic save
// through a rename. The on-disk file is rewritten
// on every Add.
func TestIndexSaveReload(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "index")
	idx, err := NewIndex(IndexConfig{Path: dir})
	if err != nil {
		t.Fatalf("NewIndex: %v", err)
	}
	a, _ := NewArtifact(KindTask, ScopeProject, "s1", []byte("x"))
	_ = idx.Add(a)
	// Open a fresh index on the same dir.
	idx2, err := NewIndex(IndexConfig{Path: dir})
	if err != nil {
		t.Fatalf("NewIndex (reload): %v", err)
	}
	if !idx2.Has(a.ContentHash) {
		t.Fatal("reloaded index missing the artifact")
	}
}

// TestIndexIDIsContentAddressed covers REQ-MEM-RET-005
// for the index. Two indexes with the same content
// have the same id.
func TestIndexIDIsContentAddressed(t *testing.T) {
	idx := newTestIndex(t)
	id1 := idx.ID()
	a, _ := NewArtifact(KindTask, ScopeProject, "s1", []byte("x"))
	_ = idx.Add(a)
	id2 := idx.ID()
	if id1 == id2 {
		t.Fatalf("id unchanged after add: %s", id1)
	}
	// Re-add same content (idempotent) — id must
	// be the same as id2.
	_ = idx.Add(a)
	id3 := idx.ID()
	if id3 != id2 {
		t.Fatalf("id changed after idempotent add: %s vs %s", id3, id2)
	}
}

// TestIndexCorruptFile covers the ErrIndexCorrupt
// path. The constructor returns the wrapped
// sentinel.
func TestIndexCorruptFile(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "index")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "index.json"), []byte("not json"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	_, err := NewIndex(IndexConfig{Path: dir})
	if !errors.Is(err, ErrIndexCorrupt) {
		t.Fatalf("expected ErrIndexCorrupt, got %v", err)
	}
}

// TestIndexEntryMatch covers the per-entry
// predicate the query uses.
func TestIndexEntryMatch(t *testing.T) {
	a, _ := NewArtifactWithMeta(KindDecision, ScopeProject, "s1", "t1", []string{"auth"}, []byte("Use HS256"))
	_ = a
	entry := IndexEntry{
		Kind:        KindDecision,
		Scope:       ScopeProject,
		SessionID:   "s1",
		TaskID:      "t1",
		Tags:        []string{"auth"},
		ContentHash: HashBytes([]byte("Use HS256")),
		Timestamp:   "2025-01-15T10:30:00Z",
	}
	q := Query{
		Kinds:    []string{"decision"},
		Scopes:   []string{"project"},
		Tags:     []string{"auth"},
		Operator: OpAND,
	}
	if !entry.Match(q) {
		t.Fatal("entry should match")
	}
	// Kinds mismatch.
	q2 := Query{Kinds: []string{"task"}, Operator: OpAND}
	if entry.Match(q2) {
		t.Fatal("entry should not match task kind")
	}
	// Scopes mismatch.
	q3 := Query{Scopes: []string{"session"}, Operator: OpAND}
	if entry.Match(q3) {
		t.Fatal("entry should not match session scope")
	}
	// Tags mismatch.
	q4 := Query{Tags: []string{"missing"}, Operator: OpAND}
	if entry.Match(q4) {
		t.Fatal("entry should not match missing tag")
	}
	// Pruned entry never matches.
	entry.Pruned = true
	if entry.Match(q) {
		t.Fatal("pruned entry should not match")
	}
	entry.Pruned = false
}

// TestIndexEntryMatchOR covers the OR operator
// branch.
func TestIndexEntryMatchOR(t *testing.T) {
	entry := IndexEntry{
		Kind:        KindDecision,
		Scope:       ScopeProject,
		SessionID:   "s1",
		ContentHash: HashBytes([]byte("x")),
		Timestamp:   "2025-01-15T10:30:00Z",
	}
	// No filters at all matches under OR.
	q := Query{Operator: OpOR}
	if !entry.Match(q) {
		t.Fatal("empty OR should match")
	}
	// One matching kind.
	q = Query{Operator: OpOR, Kinds: []string{"decision"}}
	if !entry.Match(q) {
		t.Fatal("matching kind should match under OR")
	}
	// One non-matching kind.
	q = Query{Operator: OpOR, Kinds: []string{"task"}}
	if entry.Match(q) {
		t.Fatal("non-matching kind should not match under OR")
	}
}

// TestIndexEntryMatchSince covers the since filter.
func TestIndexEntryMatchSince(t *testing.T) {
	entry := IndexEntry{
		Kind:        KindTask,
		Scope:       ScopeProject,
		SessionID:   "s1",
		ContentHash: HashBytes([]byte("x")),
		Timestamp:   "2025-01-15T10:30:00Z",
	}
	if !entry.Match(Query{Since: "2025-01-01T00:00:00Z", Operator: OpAND}) {
		t.Fatal("entry after since should match")
	}
	if entry.Match(Query{Since: "2026-01-01T00:00:00Z", Operator: OpAND}) {
		t.Fatal("entry before since should not match")
	}
	// Bad since returns false (parse failure).
	if entry.Match(Query{Since: "not-a-time", Operator: OpAND}) {
		t.Fatal("bad since should not match")
	}
	// Bad entry timestamp.
	bad := entry
	bad.Timestamp = "not-a-time"
	if bad.Match(Query{Since: "2025-01-01T00:00:00Z", Operator: OpAND}) {
		t.Fatal("bad timestamp should not match since")
	}
}

// TestIndexEntryMatchPattern covers the glob
// pattern filter (using content hash as input).
func TestIndexEntryMatchPattern(t *testing.T) {
	hash := HashBytes([]byte("hello world"))
	entry := IndexEntry{
		Kind:        KindFile,
		Scope:       ScopeProject,
		ContentHash: hash,
		Timestamp:   "2025-01-15T10:30:00Z",
	}
	// hash starts with a 64-char hex; use the
	// first 8 chars as the pattern.
	prefix := hash[:8]
	if !entry.Match(Query{Pattern: prefix + "*", Operator: OpAND}) {
		t.Fatal("prefix pattern should match")
	}
	if entry.Match(Query{Pattern: "deadbeef*", Operator: OpAND}) {
		t.Fatal("wrong prefix should not match")
	}
	// Bad pattern is permissive (returns false).
	if entry.Match(Query{Pattern: "[bad", Operator: OpAND}) {
		t.Fatal("bad pattern should not match")
	}
}

// TestIndexAllSorted covers the sort order of
// All(). The function returns entries sorted by
// content hash.
func TestIndexAllSorted(t *testing.T) {
	idx := newTestIndex(t)
	hashes := []string{}
	for i := 0; i < 5; i++ {
		a, _ := NewArtifact(KindTask, ScopeProject, "s", []byte{byte(i)})
		_ = idx.Add(a)
		hashes = append(hashes, a.ContentHash)
	}
	sort.Strings(hashes)
	all := idx.All()
	for i, e := range all {
		if e.ContentHash != hashes[i] {
			t.Fatalf("entry %d = %s, want %s", i, e.ContentHash, hashes[i])
		}
	}
}
