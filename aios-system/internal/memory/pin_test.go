package memory

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func newTestPinSet(t *testing.T) *PinSet {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "pins")
	ps, err := NewPinSet(PinSetConfig{Path: dir})
	if err != nil {
		t.Fatalf("NewPinSet: %v", err)
	}
	return ps
}

// TestPinSetAddRemove covers the basic add/remove
// cycle (REQ-MEM-PIN-001).
func TestPinSetAddRemove(t *testing.T) {
	ps := newTestPinSet(t)
	h := HashBytes([]byte("x"))
	if err := ps.Add(h, "important", "alice"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if !ps.Has(h) {
		t.Fatal("Has should be true after Add")
	}
	if err := ps.Remove(h); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if ps.Has(h) {
		t.Fatal("Has should be false after Remove")
	}
	// Remove of missing hash is a no-op.
	if err := ps.Remove(h); err != nil {
		t.Fatalf("Remove (missing): %v", err)
	}
}

// TestPinSetAddRejectsBadHash covers the
// content-hash guard.
func TestPinSetAddRejectsBadHash(t *testing.T) {
	ps := newTestPinSet(t)
	for _, bad := range []string{"", "short", "UPPERCASE" + HashBytes([]byte("x"))[7:], "../escape"} {
		if err := ps.Add(bad, "", ""); err == nil {
			t.Fatalf("expected error for %q", bad)
		}
	}
}

// TestPinSetAddRejectsEmpty covers the empty-hash
// guard.
func TestPinSetAddRejectsEmpty(t *testing.T) {
	ps := newTestPinSet(t)
	if err := ps.Add("", "", ""); err == nil {
		t.Fatal("expected error for empty hash")
	}
}

// TestPinSetIDIsContentAddressed covers
// REQ-MEM-PIN-002. The pin set id is the SHA-256
// of the pin set's content. Re-adding the same
// hash does not change the id; adding a new hash
// does.
func TestPinSetIDIsContentAddressed(t *testing.T) {
	ps := newTestPinSet(t)
	id0 := ps.ID()
	h := HashBytes([]byte("x"))
	_ = ps.Add(h, "r", "a")
	id1 := ps.ID()
	if id0 == id1 {
		t.Fatal("id unchanged after Add")
	}
	// Add same hash again — id unchanged
	// because the timestamp is preserved.
	_ = ps.Add(h, "r", "a")
	id2 := ps.ID()
	if id1 != id2 {
		t.Fatalf("id changed after idempotent add: %s vs %s", id1, id2)
	}
	// Save and reload: id is stable across
	// save/reload.
	dir := ps.Path()
	ps3, err := NewPinSet(PinSetConfig{Path: dir})
	if err != nil {
		t.Fatalf("NewPinSet: %v", err)
	}
	if ps3.ID() != id2 {
		t.Fatalf("id not stable across reload: %s vs %s", ps3.ID(), id2)
	}
}

// TestPinSetSaveReload covers the on-disk save.
func TestPinSetSaveReload(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "pins")
	ps, _ := NewPinSet(PinSetConfig{Path: dir})
	h := HashBytes([]byte("x"))
	_ = ps.Add(h, "r", "a")
	// Reload from disk.
	ps2, err := NewPinSet(PinSetConfig{Path: dir})
	if err != nil {
		t.Fatalf("NewPinSet: %v", err)
	}
	if !ps2.Has(h) {
		t.Fatal("reloaded pin set missing the hash")
	}
}

// TestPinSetCorruptFile covers the ErrPinSetCorrupt
// path.
func TestPinSetCorruptFile(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "pins")
	_ = os.MkdirAll(dir, 0o755)
	_ = os.WriteFile(filepath.Join(dir, "pins.json"), []byte("not json"), 0o644)
	_, err := NewPinSet(PinSetConfig{Path: dir})
	if !errors.Is(err, ErrPinSetCorrupt) {
		t.Fatalf("expected ErrPinSetCorrupt, got %v", err)
	}
}

// TestPinSetListSorted covers the sorted listing.
func TestPinSetListSorted(t *testing.T) {
	ps := newTestPinSet(t)
	hashes := []string{}
	for i := 0; i < 5; i++ {
		h := HashBytes([]byte{byte(i)})
		_ = ps.Add(h, "", "")
		hashes = append(hashes, h)
	}
	sort.Strings(hashes)
	got := ps.Hashes()
	for i, h := range got {
		if h != hashes[i] {
			t.Fatalf("hashes[%d] = %s, want %s", i, h, hashes[i])
		}
	}
	list := ps.List()
	if len(list) != 5 {
		t.Fatalf("List len = %d, want 5", len(list))
	}
}

// TestPinSetEntry covers the per-hash entry
// lookup.
func TestPinSetEntry(t *testing.T) {
	ps := newTestPinSet(t)
	h := HashBytes([]byte("x"))
	_ = ps.Add(h, "important decision", "alice")
	entry, ok := ps.Entry(h)
	if !ok {
		t.Fatal("Entry should return true")
	}
	if entry.Reason != "important decision" {
		t.Fatalf("reason = %q", entry.Reason)
	}
	if entry.Actor != "alice" {
		t.Fatalf("actor = %q", entry.Actor)
	}
	if entry.Timestamp == "" {
		t.Fatal("timestamp should be set")
	}
	// Missing.
	if _, ok := ps.Entry(""); ok {
		t.Fatal("Entry('') should return false")
	}
	if _, ok := ps.Entry(HashBytes([]byte("missing"))); ok {
		t.Fatal("Entry(missing) should return false")
	}
}
