package memory

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestNowRFC3339IsParseable covers the timestamp
// helper.
func TestNowRFC3339IsParseable(t *testing.T) {
	s := nowRFC3339()
	if _, err := time.Parse(time.RFC3339Nano, s); err != nil {
		t.Fatalf("nowRFC3339 not parseable: %q (%v)", s, err)
	}
}

// TestParseTimestamp covers the timestamp parser.
func TestParseTimestamp(t *testing.T) {
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		ts := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC).Format(layout)
		_, ok := parseTimestamp(ts)
		if !ok {
			t.Fatalf("parseTimestamp(%q) = false, want true", ts)
		}
	}
	if _, ok := parseTimestamp("not-a-time"); ok {
		t.Fatal("parseTimestamp(bad) = true, want false")
	}
}

// TestMemoryErrorIsMatches covers the typed error
// Is method.
func TestMemoryErrorIsMatches(t *testing.T) {
	e1 := &MemoryError{Op: "x", Reason: "y"}
	e2 := &MemoryError{Op: "a", Reason: "b"}
	if !errors.Is(e1, e2) {
		t.Fatal("MemoryError should match itself")
	}
	var other error = errors.New("plain")
	if errors.Is(e1, other) {
		t.Fatal("MemoryError should not match a plain error")
	}
}

// TestErrBudgetTooSmallErrorIsMatches covers the
// typed Is method.
func TestErrBudgetTooSmallErrorIsMatches(t *testing.T) {
	e := &ErrBudgetTooSmallError{Budget: 0, Smallest: 10, Hash: "abc"}
	if !errors.Is(e, ErrBudgetTooSmall) {
		t.Fatal("ErrBudgetTooSmallError should match ErrBudgetTooSmall")
	}
}

// TestEstimateTokens covers the token heuristic.
func TestEstimateTokens(t *testing.T) {
	a, _ := NewArtifact(KindFile, ScopeProject, "s1", make([]byte, 400))
	if got := EstimateTokens(a); got != 100 {
		t.Fatalf("EstimateTokens = %d, want 100", got)
	}
}

// TestCompactedViewIsEmpty covers the IsEmpty
// helper.
func TestCompactedViewIsEmpty(t *testing.T) {
	if !(&CompactedView{}).IsEmpty() {
		t.Fatal("zero view should be empty")
	}
	v := &CompactedView{Artifacts: []Artifact{{Kind: KindTask}}}
	if v.IsEmpty() {
		t.Fatal("view with artifact should not be empty")
	}
}

// TestScopeViewPutScopeMismatch covers the scope
// validation in putLocked.
func TestScopeViewPutScopeMismatch(t *testing.T) {
	m := newTestMemory(t)
	a, _ := NewArtifact(KindTask, ScopeUser, "s1", []byte("x"))
	if err := m.Project().putLocked(a); err == nil {
		t.Fatal("expected scope mismatch error")
	}
}

// TestScopeViewSessionInMemory covers the
// in-memory session path.
func TestScopeViewSessionInMemory(t *testing.T) {
	m := newTestMemory(t)
	if m.Session().Store().Path() != "" {
		t.Fatalf("session store path = %q, want empty", m.Session().Store().Path())
	}
	if m.Session().Index().Path() != "" {
		t.Fatalf("session index path = %q, want empty", m.Session().Index().Path())
	}
	a, _ := NewArtifact(KindTask, ScopeSession, "s1", []byte("x"))
	hash, err := m.Remember(a)
	if err != nil {
		t.Fatalf("Remember: %v", err)
	}
	got, _ := m.Session().Store().Get(hash)
	if string(got) != "x" {
		t.Fatalf("got %q", got)
	}
}

// TestNewScopeViewRequiresPathForPersistentScopes
// covers the empty-path guard.
func TestNewScopeViewRequiresPathForPersistentScopes(t *testing.T) {
	if _, err := NewScopeView(ScopeViewConfig{Scope: ScopeProject, BasePath: ""}); err == nil {
		t.Fatal("expected error for empty path on project scope")
	}
}

// TestMemoryEncryptedUserScope covers the
// AES-GCM path through the user scope.
func TestMemoryEncryptedUserScope(t *testing.T) {
	dataDir := t.TempDir()
	home := t.TempDir()
	enc, err := NewAESGCMEncryptor(make([]byte, 32))
	if err != nil {
		t.Fatalf("NewAESGCMEncryptor: %v", err)
	}
	m, err := NewMemory(MemoryConfig{
		DataDir:       dataDir,
		Home:          home,
		UserEncryptor: enc,
	})
	if err != nil {
		t.Fatalf("NewMemory: %v", err)
	}
	a, _ := NewArtifact(KindDecision, ScopeUser, "s1", []byte("user-only"))
	hash, err := m.Remember(a)
	if err != nil {
		t.Fatalf("Remember: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(home, "memory", "raw", hash))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(raw) == "user-only" {
		t.Fatal("payload is plaintext on disk")
	}
	got, _ := m.Get(hash)
	if string(got.Content) != "user-only" {
		t.Fatalf("got %q", got.Content)
	}
}
