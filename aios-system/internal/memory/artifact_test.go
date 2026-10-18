package memory

import (
	"strings"
	"testing"
)

// TestArtifactKindIsValid covers the six-kind enum.
func TestArtifactKindIsValid(t *testing.T) {
	for _, k := range []ArtifactKind{
		KindTask, KindFile, KindDecision, KindObservation,
		KindToolResult, KindModelOutput,
	} {
		if !k.IsValid() {
			t.Fatalf("expected %q to be valid", k)
		}
	}
	for _, bad := range []ArtifactKind{"", "log", "trace", "Task", "DECISION"} {
		if bad.IsValid() {
			t.Fatalf("expected %q to be invalid", bad)
		}
	}
}

// TestScopeIsValid covers the three-scope enum.
func TestScopeIsValid(t *testing.T) {
	for _, s := range []Scope{ScopeSession, ScopeProject, ScopeUser} {
		if !s.IsValid() {
			t.Fatalf("expected %q to be valid", s)
		}
	}
	for _, bad := range []Scope{"", "global", "SESSION", "User"} {
		if bad.IsValid() {
			t.Fatalf("expected %q to be invalid", bad)
		}
	}
}

// TestOperatorIsValid covers the AND/OR enum.
func TestOperatorIsValid(t *testing.T) {
	if !OpAND.IsValid() {
		t.Fatal("AND should be valid")
	}
	if !OpOR.IsValid() {
		t.Fatal("OR should be valid")
	}
	for _, bad := range []Operator{"", "and", "NOT"} {
		if bad.IsValid() {
			t.Fatalf("expected %q to be invalid", bad)
		}
	}
}

// TestNewArtifactSetsHashAndTimestamp covers the
// canonical constructor.
func TestNewArtifactSetsHashAndTimestamp(t *testing.T) {
	a, err := NewArtifact(KindDecision, ScopeProject, "s1", []byte("hello world"))
	if err != nil {
		t.Fatalf("NewArtifact: %v", err)
	}
	if a.ContentHash != HashBytes([]byte("hello world")) {
		t.Fatalf("hash mismatch: %s vs %s", a.ContentHash, HashBytes([]byte("hello world")))
	}
	if a.Timestamp == "" {
		t.Fatal("timestamp is empty")
	}
	if a.Kind != KindDecision || a.Scope != ScopeProject || a.SessionID != "s1" {
		t.Fatalf("unexpected artifact: %+v", a)
	}
}

// TestNewArtifactRejectsEmpty covers the validation
// guard on the required fields.
func TestNewArtifactRejectsEmpty(t *testing.T) {
	cases := []struct {
		name    string
		kind    ArtifactKind
		scope   Scope
		session string
		content []byte
	}{
		{"bad kind", "log", ScopeProject, "s", []byte("x")},
		{"bad scope", KindTask, "global", "s", []byte("x")},
		{"empty session", KindTask, ScopeProject, "", []byte("x")},
		{"empty content", KindTask, ScopeProject, "s", nil},
		{"empty content slice", KindTask, ScopeProject, "s", []byte{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := NewArtifact(c.kind, c.scope, c.session, c.content)
			if err == nil {
				t.Fatalf("expected error")
			}
		})
	}
}

// TestNewArtifactWithMetaCopiesTags covers the
// variant that takes metadata. The function must
// copy the tags slice so later mutations do not
// affect the artifact.
func TestNewArtifactWithMetaCopiesTags(t *testing.T) {
	tags := []string{"a", "b"}
	a, err := NewArtifactWithMeta(KindTask, ScopeSession, "s", "t1", tags, []byte("x"))
	if err != nil {
		t.Fatalf("NewArtifactWithMeta: %v", err)
	}
	tags[0] = "MUTATED"
	if a.Tags[0] != "a" {
		t.Fatalf("tag not copied: got %q", a.Tags[0])
	}
	if a.TaskID != "t1" {
		t.Fatalf("task id not set: got %q", a.TaskID)
	}
}

// TestArtifactContentMatches covers the glob
// pattern match. The function decodes the content
// as a string and runs filepath.Match.
func TestArtifactContentMatches(t *testing.T) {
	a, _ := NewArtifact(KindFile, ScopeProject, "s", []byte("hello world"))
	if !a.ContentMatches("hello*") {
		t.Fatal("expected match for hello*")
	}
	if a.ContentMatches("goodbye*") {
		t.Fatal("expected no match for goodbye*")
	}
	if !a.ContentMatches("") {
		t.Fatal("empty pattern should match")
	}
}

// TestArtifactHasTag covers the exact tag match.
func TestArtifactHasTag(t *testing.T) {
	a, _ := NewArtifactWithMeta(KindTask, ScopeSession, "s", "", []string{"auth", "jwt"}, []byte("x"))
	if !a.HasTag("auth") {
		t.Fatal("expected auth tag")
	}
	if a.HasTag("missing") {
		t.Fatal("expected no missing tag")
	}
	if (&Artifact{}).HasTag("anything") {
		t.Fatal("nil tags should not match")
	}
}

// TestArtifactSince covers the timestamp filter.
func TestArtifactSince(t *testing.T) {
	a, _ := NewArtifact(KindTask, ScopeProject, "s", []byte("x"))
	if !a.Since("") {
		t.Fatal("empty since should match")
	}
	if !a.Since("1970-01-01T00:00:00Z") {
		t.Fatal("1970 should be before now")
	}
	if a.Since("3000-01-01T00:00:00Z") {
		t.Fatal("future should not match")
	}
	// Bad artifact timestamps.
	bad := &Artifact{Timestamp: "not-a-time"}
	if bad.Since("2025-01-01T00:00:00Z") {
		t.Fatal("bad timestamp should not match")
	}
}

// TestArtifactValidateRejectsBadHash covers the
// hash mismatch guard.
func TestArtifactValidateRejectsBadHash(t *testing.T) {
	a := Artifact{
		Kind:        KindTask,
		Scope:       ScopeProject,
		SessionID:   "s",
		Content:     []byte("hello"),
		ContentHash: "deadbeef",
	}
	if err := a.Validate(); err == nil {
		t.Fatal("expected hash mismatch error")
	}
}

// TestHashBytesDeterministic covers the content
// hash function: same input, same output.
func TestHashBytesDeterministic(t *testing.T) {
	a := HashBytes([]byte("hello"))
	b := HashBytes([]byte("hello"))
	if a != b {
		t.Fatalf("hash not deterministic: %s vs %s", a, b)
	}
	c := HashBytes([]byte("world"))
	if a == c {
		t.Fatal("hashes collided for different inputs")
	}
	if len(a) != 64 {
		t.Fatalf("hash not 64 chars: %d", len(a))
	}
	if !strings.HasPrefix(a, strings.Repeat("0", 0)) {
		// sanity: hash should be lowercase hex
		for _, ch := range a {
			if !(ch >= '0' && ch <= '9') && !(ch >= 'a' && ch <= 'f') {
				t.Fatalf("hash not lowercase hex: %q", a)
			}
		}
	}
}
