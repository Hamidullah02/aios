package memory

import (
	"path/filepath"
	"sort"
	"testing"
)

// newTestMemory constructs a Memory pointed at
// fresh temp directories. The function returns the
// memory and a cleanup the caller must defer.
func newTestMemory(t *testing.T) *Memory {
	t.Helper()
	dataDir := t.TempDir()
	home := t.TempDir()
	m, err := NewMemory(MemoryConfig{
		DataDir: dataDir,
		Home:    home,
	})
	if err != nil {
		t.Fatalf("NewMemory: %v", err)
	}
	return m
}

// mustArtifact constructs an artifact and fails the
// test on error. The helper is the typed constructor
// for test bodies that do not want to plumb errors.
func mustArtifact(t *testing.T, kind ArtifactKind, scope Scope, session string, payload []byte) Artifact {
	t.Helper()
	a, err := NewArtifact(kind, scope, session, payload)
	if err != nil {
		t.Fatalf("NewArtifact: %v", err)
	}
	return a
}

// seedMemory populates the memory with N artifacts
// of the given kind and returns the artifacts in
// insertion order.
func seedMemory(t *testing.T, m *Memory, kind ArtifactKind, n int) []Artifact {
	t.Helper()
	out := make([]Artifact, n)
	for i := 0; i < n; i++ {
		payload := []byte{byte(i)}
		a, err := NewArtifact(kind, ScopeProject, "s1", payload)
		if err != nil {
			t.Fatalf("NewArtifact: %v", err)
		}
		if _, err := m.Remember(a); err != nil {
			t.Fatalf("Remember: %v", err)
		}
		out[i] = a
	}
	return out
}

// TestMemoryRememberAndGet covers the round-trip
// through the store and the index.
func TestMemoryRememberAndGet(t *testing.T) {
	m := newTestMemory(t)
	a, _ := NewArtifact(KindDecision, ScopeProject, "s1", []byte("Use HS256"))
	hash, err := m.Remember(a)
	if err != nil {
		t.Fatalf("Remember: %v", err)
	}
	if hash != a.ContentHash {
		t.Fatalf("hash mismatch: %s vs %s", hash, a.ContentHash)
	}
	got, err := m.Get(hash)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if string(got.Content) != "Use HS256" {
		t.Fatalf("content = %q", got.Content)
	}
	if got.Kind != KindDecision {
		t.Fatalf("kind = %q", got.Kind)
	}
}

// TestMemoryHas covers the existence check.
func TestMemoryHas(t *testing.T) {
	m := newTestMemory(t)
	a, _ := NewArtifact(KindTask, ScopeProject, "s1", []byte("x"))
	_, _ = m.Remember(a)
	if !m.Has(a.ContentHash) {
		t.Fatal("Has should be true")
	}
	if m.Has(HashBytes([]byte("missing"))) {
		t.Fatal("Has(missing) should be false")
	}
	if m.Has("") {
		t.Fatal("Has('') should be false")
	}
}

// TestMemoryQueryAcrossScopes covers the merged
// query path: artifacts from session + project +
// user are joined.
func TestMemoryQueryAcrossScopes(t *testing.T) {
	m := newTestMemory(t)
	// Session artifact.
	s1, _ := NewArtifact(KindTask, ScopeSession, "s1", []byte("session"))
	_, _ = m.Remember(s1)
	// Project artifact.
	p1, _ := NewArtifact(KindDecision, ScopeProject, "s1", []byte("project"))
	_, _ = m.Remember(p1)
	// User artifact.
	u1, _ := NewArtifact(KindObservation, ScopeUser, "s1", []byte("user"))
	_, _ = m.Remember(u1)
	// Query everything.
	got, err := m.Query(Query{Operator: OpAND})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("got %d, want 3", len(got))
	}
	// Query by kind.
	got, _ = m.Query(Query{Kinds: []string{"decision"}, Operator: OpAND})
	if len(got) != 1 || got[0].Kind != KindDecision {
		t.Fatalf("got %v, want one decision", got)
	}
}

// TestMemoryResetSession covers the session reset.
func TestMemoryResetSession(t *testing.T) {
	m := newTestMemory(t)
	s1, _ := NewArtifact(KindTask, ScopeSession, "s1", []byte("x"))
	_, _ = m.Remember(s1)
	if m.Session().Index().Size() != 1 {
		t.Fatalf("session size = %d, want 1", m.Session().Index().Size())
	}
	if err := m.ResetSession(); err != nil {
		t.Fatalf("ResetSession: %v", err)
	}
	if m.Session().Index().Size() != 0 {
		t.Fatal("session should be empty after reset")
	}
}

// TestMemoryPruneAndUnprune cover the prune cycle
// through the index.
func TestMemoryPruneAndUnprune(t *testing.T) {
	m := newTestMemory(t)
	a, _ := NewArtifact(KindTask, ScopeProject, "s1", []byte("x"))
	_, _ = m.Remember(a)
	if err := m.Project().Index().Prune(a.ContentHash); err != nil {
		t.Fatalf("Prune: %v", err)
	}
	got, _ := m.Query(Query{Operator: OpAND})
	if len(got) != 0 {
		t.Fatalf("expected 0 after prune, got %d", len(got))
	}
	if err := m.Project().Index().Unprune(a.ContentHash); err != nil {
		t.Fatalf("Unprune: %v", err)
	}
	got, _ = m.Query(Query{Operator: OpAND})
	if len(got) != 1 {
		t.Fatalf("expected 1 after unprune, got %d", len(got))
	}
}

// storeResolver returns an EntryResolver that
// joins the index with the given store. The
// resolver is the production-style join tests
// use when they need the artifact's payload for
// the budget calculation.
func storeResolver(store *Store) EntryResolver {
	return func(e IndexEntry) (Artifact, bool) {
		payload, err := store.Get(e.ContentHash)
		if err != nil {
			return Artifact{}, false
		}
		return entryToArtifact(e, payload), true
	}
}

// TestRetrievalPinnedAlwaysIncluded covers
// REQ-MEM-RET-003 priority 1: pinned artifacts are
// always in the view.
func TestRetrievalPinnedAlwaysIncluded(t *testing.T) {
	m := newTestMemory(t)
	artifacts := seedMemory(t, m, KindTask, 5)
	// Pin the first one.
	_ = m.PinSet().Add(artifacts[0].ContentHash, "important", "test")
	r := NewRetrieval(m.Project().Index(), m.PinSet(), 10000, storeResolver(m.Project().Store()))
	view, err := r.Retrieve(Query{Operator: OpAND}, "")
	if err != nil {
		t.Fatalf("Retrieve: %v", err)
	}
	if len(view.Artifacts) != 5 {
		t.Fatalf("got %d, want 5", len(view.Artifacts))
	}
	if view.Artifacts[0].ContentHash != artifacts[0].ContentHash {
		t.Fatalf("pinned artifact not first")
	}
}

// TestRetrievalCurrentTaskFirst covers
// REQ-MEM-RET-003 priority 2: current-task artifacts
// are always in the view and rank above siblings.
func TestRetrievalCurrentTaskFirst(t *testing.T) {
	m := newTestMemory(t)
	// Three artifacts: t1, t2, t3.
	for i, taskID := range []string{"t2", "t1", "t3"} {
		a, _ := NewArtifactWithMeta(KindTask, ScopeProject, "s1", taskID, nil, []byte{byte(i)})
		_, _ = m.Remember(a)
	}
	r := NewRetrieval(m.Project().Index(), m.PinSet(), 10000, storeResolver(m.Project().Store()))
	view, err := r.Retrieve(Query{Operator: OpAND}, "t1")
	if err != nil {
		t.Fatalf("Retrieve: %v", err)
	}
	if len(view.Artifacts) != 3 {
		t.Fatalf("got %d, want 3", len(view.Artifacts))
	}
	if view.Artifacts[0].TaskID != "t1" {
		t.Fatalf("current task not first: %s", view.Artifacts[0].TaskID)
	}
}

// TestRetrievalBudgetEnforced covers REQ-MEM-RET-004.
func TestRetrievalBudgetEnforced(t *testing.T) {
	m := newTestMemory(t)
	_ = seedMemory(t, m, KindFile, 4) // each 1 byte → 0 tokens
	// Add a 100-byte artifact.
	big, _ := NewArtifact(KindFile, ScopeProject, "s1", make([]byte, 100))
	hash, err := m.Remember(big)
	if err != nil {
		t.Fatalf("Remember: %v", err)
	}
	r := NewRetrieval(m.Project().Index(), m.PinSet(), 10, storeResolver(m.Project().Store()))
	view, err := r.Retrieve(Query{Operator: OpAND}, "")
	if err != nil {
		t.Fatalf("Retrieve: %v", err)
	}
	for _, a := range view.Artifacts {
		if a.ContentHash == hash {
			t.Fatal("big artifact should be dropped")
		}
	}
}

// TestRetrievalPinnedSurvivesBudget covers
// REQ-MEM-PIN-004: a pinned artifact cannot be
// dropped even when the budget is tight.
func TestRetrievalPinnedSurvivesBudget(t *testing.T) {
	m := newTestMemory(t)
	artifacts := seedMemory(t, m, KindFile, 3)
	_ = m.PinSet().Add(artifacts[2].ContentHash, "important", "test")
	// Budget of 1 token: only artifacts with 0
	// tokens can fit. The pinned 3rd artifact is
	// 1 byte (0 tokens) and fits.
	r := NewRetrieval(m.Project().Index(), m.PinSet(), 1, storeResolver(m.Project().Store()))
	view, err := r.Retrieve(Query{Operator: OpAND}, "")
	if err != nil {
		t.Fatalf("Retrieve: %v", err)
	}
	found := false
	for _, a := range view.Artifacts {
		if a.ContentHash == artifacts[2].ContentHash {
			found = true
		}
	}
	if !found {
		t.Fatal("pinned artifact not in view")
	}
}

// TestRetrievalPure covers REQ-MEM-RET-005.
func TestRetrievalPure(t *testing.T) {
	m := newTestMemory(t)
	_ = seedMemory(t, m, KindTask, 3)
	r := NewRetrieval(m.Project().Index(), m.PinSet(), 10000, storeResolver(m.Project().Store()))
	view1, _ := r.Retrieve(Query{Operator: OpAND}, "")
	view2, _ := r.Retrieve(Query{Operator: OpAND}, "")
	if view1.PinSetID != view2.PinSetID {
		t.Fatalf("PinSetID differs: %s vs %s", view1.PinSetID, view2.PinSetID)
	}
	if view1.IndexID != view2.IndexID {
		t.Fatalf("IndexID differs: %s vs %s", view1.IndexID, view2.IndexID)
	}
	if len(view1.Artifacts) != len(view2.Artifacts) {
		t.Fatalf("artifact count differs: %d vs %d", len(view1.Artifacts), len(view2.Artifacts))
	}
	hashes1 := make([]string, len(view1.Artifacts))
	hashes2 := make([]string, len(view2.Artifacts))
	for i, a := range view1.Artifacts {
		hashes1[i] = a.ContentHash
	}
	for i, a := range view2.Artifacts {
		hashes2[i] = a.ContentHash
	}
	sort.Strings(hashes1)
	sort.Strings(hashes2)
	for i := range hashes1 {
		if hashes1[i] != hashes2[i] {
			t.Fatalf("hash %d differs: %s vs %s", i, hashes1[i], hashes2[i])
		}
	}
}

// TestCompactionBudget covers REQ-MEM-COM-003.
func TestCompactionBudget(t *testing.T) {
	m := newTestMemory(t)
	for i := 0; i < 5; i++ {
		payload := make([]byte, 200)
		payload[0] = byte(i) // distinct content
		_, err := m.Remember(mustArtifact(t, KindFile, ScopeProject, "s1", payload))
		if err != nil {
			t.Fatalf("Remember: %v", err)
		}
	}
	c := NewCompaction(m.Project().Index(), m.PinSet(), storeResolver(m.Project().Store()))
	view, err := c.Compact(Query{Operator: OpAND}, "", 100)
	if err != nil {
		t.Fatalf("Compact: %v", err)
	}
	if view.TotalTokens > 100 {
		t.Fatalf("TotalTokens = %d, want <= 100", view.TotalTokens)
	}
}

// TestCompactionObservable covers REQ-MEM-COM-005.
// The view's Dropped list is populated when the
// budget is tight.
func TestCompactionObservable(t *testing.T) {
	m := newTestMemory(t)
	// 3 large artifacts (400 bytes = 100 tokens each).
	for i := 0; i < 3; i++ {
		payload := make([]byte, 400)
		payload[0] = byte(i) // ensure distinct content
		_, err := m.Remember(mustArtifact(t, KindFile, ScopeProject, "s1", payload))
		if err != nil {
			t.Fatalf("Remember: %v", err)
		}
	}
	c := NewCompaction(m.Project().Index(), m.PinSet(), storeResolver(m.Project().Store()))
	view, err := c.Compact(Query{Operator: OpAND}, "", 50)
	if err != nil {
		t.Fatalf("Compact: %v", err)
	}
	if !view.Truncated {
		t.Fatalf("view should be truncated (Kept=%d, Dropped=%d)", len(view.Artifacts), len(view.Dropped))
	}
	if len(view.Dropped) != 3 {
		t.Fatalf("Dropped = %d, want 3", len(view.Dropped))
	}
}

// TestCompactionForTask covers REQ-MEM-COM-004.
func TestCompactionForTask(t *testing.T) {
	m := newTestMemory(t)
	for i := 0; i < 3; i++ {
		a, _ := NewArtifactWithMeta(KindTask, ScopeProject, "s1", "t1", nil, []byte{byte(i)})
		_, _ = m.Remember(a)
	}
	for i := 0; i < 2; i++ {
		a, _ := NewArtifactWithMeta(KindTask, ScopeProject, "s1", "t2", nil, []byte{byte(i + 10)})
		_, _ = m.Remember(a)
	}
	pinned, _ := NewArtifact(KindDecision, ScopeUser, "s1", []byte("pinned"))
	_, _ = m.Remember(pinned)
	_ = m.PinSet().Add(pinned.ContentHash, "pinned", "test")
	c := NewCompaction(m.Project().Index(), m.PinSet(), storeResolver(m.Project().Store()))
	view, err := c.CompactForTask("t1", 0)
	if err != nil {
		t.Fatalf("CompactForTask: %v", err)
	}
	if len(view.Artifacts) != 3 {
		t.Fatalf("got %d, want 3", len(view.Artifacts))
	}
	if !view.Truncated {
		t.Fatalf("view should be truncated (t2 dropped)")
	}
	if len(view.Dropped) != 2 {
		t.Fatalf("Dropped = %d, want 2", len(view.Dropped))
	}
}

// TestCompactionPure covers REQ-MEM-COM-006.
func TestCompactionPure(t *testing.T) {
	m := newTestMemory(t)
	_ = seedMemory(t, m, KindTask, 3)
	c := NewCompaction(m.Project().Index(), m.PinSet(), storeResolver(m.Project().Store()))
	v1, _ := c.Compact(Query{Operator: OpAND}, "", 1000)
	v2, _ := c.Compact(Query{Operator: OpAND}, "", 1000)
	if v1.PinSetID != v2.PinSetID {
		t.Fatalf("PinSetID differs: %s vs %s", v1.PinSetID, v2.PinSetID)
	}
	if v1.IndexID != v2.IndexID {
		t.Fatalf("IndexID differs: %s vs %s", v1.IndexID, v2.IndexID)
	}
	if len(v1.Artifacts) != len(v2.Artifacts) {
		t.Fatalf("artifact count differs")
	}
}

// TestMemoryDefaultPaths covers the env-var
// fallback.
func TestMemoryDefaultPaths(t *testing.T) {
	dataDir := t.TempDir()
	home := t.TempDir()
	t.Setenv("AIOS_DATA_DIR", dataDir)
	t.Setenv("AIOS_HOME", home)
	m, err := NewMemory(MemoryConfig{})
	if err != nil {
		t.Fatalf("NewMemory: %v", err)
	}
	// Compare with forward slashes to be
	// cross-platform.
	want := filepath.ToSlash(filepath.Join(dataDir, "memory", "raw"))
	got := filepath.ToSlash(m.Project().Store().Path())
	if got != want {
		t.Fatalf("project store path = %s, want %s", got, want)
	}
}
