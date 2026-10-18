package memory

import (
	"testing"
)

// TestQueryValidate covers the validation rules.
func TestQueryValidate(t *testing.T) {
	q := Query{Kinds: []string{"task"}, Scopes: []string{"project"}, Operator: OpAND}
	if err := q.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	q.Operator = "NOPE"
	if err := q.Validate(); err == nil {
		t.Fatal("expected error for bad operator")
	}
	q.Operator = OpAND
	q.Kinds = []string{"log"}
	if err := q.Validate(); err == nil {
		t.Fatal("expected error for bad kind")
	}
	q.Kinds = []string{"task"}
	q.Scopes = []string{"global"}
	if err := q.Validate(); err == nil {
		t.Fatal("expected error for bad scope")
	}
}

// TestQueryNewConstructors cover the typed
// constructors.
func TestQueryNewConstructors(t *testing.T) {
	q := NewQuery([]string{"task"}, []string{"project"}, []string{"auth"})
	if q.Operator != OpAND {
		t.Fatalf("operator = %q, want AND", q.Operator)
	}
	if len(q.Kinds) != 1 || q.Kinds[0] != "task" {
		t.Fatalf("kinds = %v", q.Kinds)
	}
	q2 := NewQueryWithPattern("hello*")
	if q2.Pattern != "hello*" {
		t.Fatalf("pattern = %q", q2.Pattern)
	}
	if q2.Operator != OpAND {
		t.Fatalf("operator = %q", q2.Operator)
	}
}

// TestQueryRunIndexKinds covers the kind filter
// (REQ-MEM-QUE-001 kind).
func TestQueryRunIndexKinds(t *testing.T) {
	entries := []IndexEntry{
		{Kind: KindTask, ContentHash: "a", Timestamp: "2025-01-01T00:00:00Z"},
		{Kind: KindFile, ContentHash: "b", Timestamp: "2025-01-02T00:00:00Z"},
		{Kind: KindDecision, ContentHash: "c", Timestamp: "2025-01-03T00:00:00Z"},
	}
	q := Query{Kinds: []string{"task"}, Operator: OpAND}
	got, err := q.RunIndex(entries)
	if err != nil {
		t.Fatalf("RunIndex: %v", err)
	}
	if len(got) != 1 || got[0].Kind != KindTask {
		t.Fatalf("got %+v", got)
	}
}

// TestQueryRunIndexScopes covers the scope filter.
func TestQueryRunIndexScopes(t *testing.T) {
	entries := []IndexEntry{
		{Kind: KindTask, Scope: ScopeSession, ContentHash: "a", Timestamp: "2025-01-01T00:00:00Z"},
		{Kind: KindTask, Scope: ScopeProject, ContentHash: "b", Timestamp: "2025-01-02T00:00:00Z"},
		{Kind: KindTask, Scope: ScopeUser, ContentHash: "c", Timestamp: "2025-01-03T00:00:00Z"},
	}
	q := Query{Scopes: []string{"project"}, Operator: OpAND}
	got, err := q.RunIndex(entries)
	if err != nil {
		t.Fatalf("RunIndex: %v", err)
	}
	if len(got) != 1 || got[0].Scope != ScopeProject {
		t.Fatalf("got %+v", got)
	}
}

// TestQueryRunIndexTags covers the tag filter.
func TestQueryRunIndexTags(t *testing.T) {
	entries := []IndexEntry{
		{Kind: KindTask, Tags: []string{"auth"}, ContentHash: "a", Timestamp: "2025-01-01T00:00:00Z"},
		{Kind: KindTask, Tags: []string{"db"}, ContentHash: "b", Timestamp: "2025-01-02T00:00:00Z"},
		{Kind: KindTask, Tags: []string{"auth", "jwt"}, ContentHash: "c", Timestamp: "2025-01-03T00:00:00Z"},
	}
	q := Query{Tags: []string{"auth"}, Operator: OpAND}
	got, err := q.RunIndex(entries)
	if err != nil {
		t.Fatalf("RunIndex: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d, want 2", len(got))
	}
}

// TestQueryRunIndexPattern covers the pattern
// filter against the content hash.
func TestQueryRunIndexPattern(t *testing.T) {
	entries := []IndexEntry{
		{Kind: KindFile, ContentHash: "abcdef1234567890", Timestamp: "2025-01-01T00:00:00Z"},
		{Kind: KindFile, ContentHash: "deadbeef00000000", Timestamp: "2025-01-02T00:00:00Z"},
	}
	q := Query{Pattern: "abcdef*", Operator: OpAND}
	got, _ := q.RunIndex(entries)
	if len(got) != 1 || got[0].ContentHash != "abcdef1234567890" {
		t.Fatalf("got %+v", got)
	}
}

// TestQueryRunIndexSince covers the since filter.
func TestQueryRunIndexSince(t *testing.T) {
	entries := []IndexEntry{
		{Kind: KindTask, ContentHash: "a", Timestamp: "2024-01-01T00:00:00Z"},
		{Kind: KindTask, ContentHash: "b", Timestamp: "2025-06-01T00:00:00Z"},
		{Kind: KindTask, ContentHash: "c", Timestamp: "2026-01-01T00:00:00Z"},
	}
	q := Query{Since: "2025-01-01T00:00:00Z", Operator: OpAND}
	got, _ := q.RunIndex(entries)
	if len(got) != 2 {
		t.Fatalf("got %d, want 2", len(got))
	}
}

// TestQueryRunIndexLimit covers the limit filter
// (REQ-MEM-QUE-001 limit).
func TestQueryRunIndexLimit(t *testing.T) {
	entries := []IndexEntry{}
	for i := 0; i < 10; i++ {
		entries = append(entries, IndexEntry{
			Kind:        KindTask,
			ContentHash: string(rune('a' + i)),
			Timestamp:   "2025-01-01T00:00:00Z",
		})
	}
	q := Query{Limit: 3, Operator: OpAND}
	got, _ := q.RunIndex(entries)
	if len(got) != 3 {
		t.Fatalf("got %d, want 3", len(got))
	}
}

// TestQueryRunIndexSortedDesc covers the timestamp
// descending sort (REQ-MEM-QUE-002).
func TestQueryRunIndexSortedDesc(t *testing.T) {
	entries := []IndexEntry{
		{Kind: KindTask, ContentHash: "a", Timestamp: "2025-01-01T00:00:00Z"},
		{Kind: KindTask, ContentHash: "b", Timestamp: "2025-06-01T00:00:00Z"},
		{Kind: KindTask, ContentHash: "c", Timestamp: "2025-03-01T00:00:00Z"},
	}
	q := Query{Operator: OpAND}
	got, _ := q.RunIndex(entries)
	if len(got) != 3 {
		t.Fatalf("got %d", len(got))
	}
	if got[0].ContentHash != "b" || got[1].ContentHash != "c" || got[2].ContentHash != "a" {
		t.Fatalf("sort order wrong: %v", got)
	}
}

// TestQueryRunIndexANDCombinesFilters covers the
// AND combination.
func TestQueryRunIndexANDCombinesFilters(t *testing.T) {
	entries := []IndexEntry{
		{Kind: KindTask, Scope: ScopeProject, Tags: []string{"auth"}, ContentHash: "a", Timestamp: "2025-01-01T00:00:00Z"},
		{Kind: KindTask, Scope: ScopeProject, Tags: []string{"db"}, ContentHash: "b", Timestamp: "2025-01-01T00:00:00Z"},
		{Kind: KindFile, Scope: ScopeProject, Tags: []string{"auth"}, ContentHash: "c", Timestamp: "2025-01-01T00:00:00Z"},
		{Kind: KindTask, Scope: ScopeSession, Tags: []string{"auth"}, ContentHash: "d", Timestamp: "2025-01-01T00:00:00Z"},
	}
	q := Query{
		Kinds:    []string{"task"},
		Scopes:   []string{"project"},
		Tags:     []string{"auth"},
		Operator: OpAND,
	}
	got, _ := q.RunIndex(entries)
	if len(got) != 1 || got[0].ContentHash != "a" {
		t.Fatalf("AND got %v, want [a]", got)
	}
}

// TestQueryRunIndexORCombinesFilters covers the
// OR combination.
func TestQueryRunIndexORCombinesFilters(t *testing.T) {
	entries := []IndexEntry{
		{Kind: KindTask, Scope: ScopeProject, Tags: []string{"auth"}, ContentHash: "a", Timestamp: "2025-01-01T00:00:00Z"},
		{Kind: KindTask, Scope: ScopeProject, Tags: []string{"db"}, ContentHash: "b", Timestamp: "2025-01-01T00:00:00Z"},
		{Kind: KindFile, Scope: ScopeProject, Tags: []string{"auth"}, ContentHash: "c", Timestamp: "2025-01-01T00:00:00Z"},
		{Kind: KindObservation, Scope: ScopeSession, Tags: []string{"none"}, ContentHash: "d", Timestamp: "2025-01-01T00:00:00Z"},
	}
	q := Query{
		Kinds:    []string{"task"},
		Tags:     []string{"auth"},
		Operator: OpOR,
	}
	got, _ := q.RunIndex(entries)
	// a (task+auth), b (task), c (auth). d matches
	// neither kind nor tag.
	if len(got) != 3 {
		t.Fatalf("OR got %d, want 3: %+v", len(got), got)
	}
}

// TestQueryEmpty covers the empty-query path.
func TestQueryEmpty(t *testing.T) {
	q := Query{}
	if !q.IsEmpty() {
		t.Fatal("Query{} should be empty")
	}
	q.Kinds = []string{"task"}
	if q.IsEmpty() {
		t.Fatal("non-empty query should not be empty")
	}
}

// TestQueryNewSessionQuery covers the session
// constructor.
func TestQueryNewSessionQuery(t *testing.T) {
	q := NewSessionQuery("s1")
	if q.Operator != OpAND {
		t.Fatalf("operator = %q", q.Operator)
	}
	if len(q.Tags) != 1 || q.Tags[0] != "session:s1" {
		t.Fatalf("tags = %v", q.Tags)
	}
}
