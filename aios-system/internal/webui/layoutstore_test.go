package webui

import (
	"os"
	"path/filepath"
	"testing"
)

// TestLayoutStore_RoundTripsAcrossProcesses is the test that
// was missing entirely: an operator rearranges panels, the
// process exits, and a new one starts. Before this the layout
// lived only in the hub's memory, so the arrangement was gone.
//
// A fresh Hub reading the same store is the stand-in for the
// next process.
func TestLayoutStore_RoundTripsAcrossProcesses(t *testing.T) {
	path := filepath.Join(t.TempDir(), "layout.json")

	// Process one: the operator hides the audit panel and
	// switches to the compact theme.
	first := NewHub()
	if err := first.SetStore(NewLayoutStore(path)); err != nil {
		t.Fatalf("SetStore: %v", err)
	}
	saved := Layout{
		Panels: []PanelState{
			{ID: PanelAgents, Visible: true, Order: 0, Width: 12},
			{ID: PanelComms, Visible: true, Order: 1, Width: 6},
			{ID: PanelAudit, Visible: false, Order: 2, Width: 6},
		},
		Theme:   "dark",
		Compact: true,
	}
	if err := first.SetLayout(saved); err != nil {
		t.Fatalf("SetLayout: %v", err)
	}

	// Process two: a brand new hub over the same file.
	second := NewHub()
	if err := second.SetStore(NewLayoutStore(path)); err != nil {
		t.Fatalf("SetStore: %v", err)
	}
	got := second.Layout()
	if got.Theme != "dark" {
		t.Errorf("Theme = %q, want dark", got.Theme)
	}
	if !got.Compact {
		t.Error("Compact = false, want true")
	}
	for _, p := range got.Panels {
		if p.ID == PanelAudit && p.Visible {
			t.Error("the audit panel came back visible; the saved layout was lost")
		}
	}
}

// TestLayoutStore_MissingFileIsNotAnError covers first run.
func TestLayoutStore_MissingFileIsNotAnError(t *testing.T) {
	s := NewLayoutStore(filepath.Join(t.TempDir(), "nope.json"))
	l, found, err := s.Load()
	if err != nil {
		t.Fatalf("a missing layout file must not be an error: %v", err)
	}
	if found {
		t.Error("found = true for a file that does not exist")
	}
	if len(l.Panels) != len(AllPanels) {
		t.Errorf("got %d panels, want the %d defaults", len(l.Panels), len(AllPanels))
	}
}

// TestLayoutStore_NilStoreIsInert asserts a hub with no store
// keeps working in memory, which is what every test and a
// throwaway server rely on.
func TestLayoutStore_NilStoreIsInert(t *testing.T) {
	h := NewHub()
	if err := h.SetStore(nil); err != nil {
		t.Errorf("SetStore(nil): %v", err)
	}
	if err := h.SetLayout(DefaultLayout()); err != nil {
		t.Errorf("SetLayout with no store: %v", err)
	}
	if len(h.Layout().Panels) != len(AllPanels) {
		t.Error("the in-memory layout was not updated")
	}
}

// TestLayoutStore_NilPathWritesNothing covers the disabled
// case: a store with no path must not create files.
func TestLayoutStore_NilPathWritesNothing(t *testing.T) {
	dir := t.TempDir()
	s := NewLayoutStore("")
	if err := s.Save(DefaultLayout()); err != nil {
		t.Fatalf("Save with an empty path: %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("an inert store wrote %d file(s)", len(entries))
	}
}

// TestLayoutStore_CorruptFileFallsBack asserts a damaged file
// costs the layout, not the server.
func TestLayoutStore_CorruptFileFallsBack(t *testing.T) {
	path := filepath.Join(t.TempDir(), "layout.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := NewLayoutStore(path)
	l, found, err := s.Load()
	if err == nil {
		t.Error("a corrupt layout file should report an error")
	}
	if found {
		t.Error("found = true for a corrupt file")
	}
	if len(l.Panels) != len(AllPanels) {
		t.Errorf("the fallback layout is not usable: %+v", l)
	}

	// And the hub must still serve.
	h := NewHub()
	if err := h.SetStore(s); err == nil {
		t.Error("SetStore should report the corrupt file")
	}
	if len(h.Layout().Panels) != len(AllPanels) {
		t.Error("the hub did not fall back to a usable layout")
	}
}

// TestLayoutStore_UnknownPanelOnDiskIsRejected asserts a
// layout naming a panel this build lacks surfaces, rather
// than silently rendering without it.
func TestLayoutStore_UnknownPanelOnDiskIsRejected(t *testing.T) {
	path := filepath.Join(t.TempDir(), "layout.json")
	body := `{"panels":[{"id":"from-a-newer-build","visible":true,"order":0,"width":6}]}`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	_, found, err := NewLayoutStore(path).Load()
	if err == nil {
		t.Error("an unknown panel should be reported")
	}
	if found {
		t.Error("a layout that failed validation must not count as loaded")
	}
}

// TestLayoutStore_OldLayoutGainsNewPanels asserts forward
// compatibility: a file saved before a panel existed picks up
// the new panel instead of being discarded.
func TestLayoutStore_OldLayoutGainsNewPanels(t *testing.T) {
	path := filepath.Join(t.TempDir(), "layout.json")
	// Only the agents panel: an older build's file.
	body := `{"panels":[{"id":"agents","visible":true,"order":0,"width":12}],"theme":"light"}`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	l, found, err := NewLayoutStore(path).Load()
	if err != nil {
		t.Fatalf("an old but valid layout must load: %v", err)
	}
	if !found {
		t.Fatal("found = false for a readable layout")
	}
	if l.Theme != "light" {
		t.Errorf("Theme = %q, want light", l.Theme)
	}
	if len(l.Panels) != len(AllPanels) {
		t.Errorf("got %d panels, want all %d added back", len(l.Panels), len(AllPanels))
	}
}

// TestLayoutStore_SavedFileIsAlwaysValid asserts what goes
// on disk is normalisable, so the next Load cannot fail.
func TestLayoutStore_SavedFileIsAlwaysValid(t *testing.T) {
	path := filepath.Join(t.TempDir(), "layout.json")
	s := NewLayoutStore(path)
	// An out-of-range width and a negative order: clamped, not
	// rejected, so the rest of the layout still saves.
	if err := s.Save(Layout{
		Panels: []PanelState{{ID: PanelAgents, Visible: true, Order: -3, Width: 999}},
	}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	l, found, err := NewLayoutStore(path).Load()
	if err != nil {
		t.Fatalf("a layout this build wrote must load: %v", err)
	}
	if !found {
		t.Fatal("found = false after Save")
	}
	for _, p := range l.Panels {
		if p.Width < 1 || p.Width > MaxWidth {
			t.Errorf("panel %s has width %d, outside 1..%d", p.ID, p.Width, MaxWidth)
		}
		if p.Order < 0 {
			t.Errorf("panel %s has order %d", p.ID, p.Order)
		}
	}
}

// TestLayoutStore_SaveCreatesParentDir covers the first-run
// case where $AIOS_HOME does not exist yet.
func TestLayoutStore_SaveCreatesParentDir(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "deeper", "layout.json")
	if err := NewLayoutStore(path).Save(DefaultLayout()); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("layout not written: %v", err)
	}
}

// TestDefaultLayoutPath_HonoursEnv asserts the path follows
// the same convention as the rest of the system.
func TestDefaultLayoutPath_HonoursEnv(t *testing.T) {
	home := t.TempDir()
	t.Setenv("AIOS_HOME", home)
	want := filepath.Join(home, "layout.json")
	if got := DefaultLayoutPath(); got != want {
		t.Errorf("DefaultLayoutPath() = %s, want %s", got, want)
	}
}

// TestDefaultLayoutPath_FallsBackToUserProfile covers a
// machine with no HOME set, which is common on Windows.
func TestDefaultLayoutPath_FallsBackToUserProfile(t *testing.T) {
	t.Setenv("AIOS_HOME", "")
	t.Setenv("HOME", "")
	profile := t.TempDir()
	t.Setenv("USERPROFILE", profile)
	want := filepath.Join(profile, ".aios", "layout.json")
	if got := DefaultLayoutPath(); got != want {
		t.Errorf("DefaultLayoutPath() = %s, want %s", got, want)
	}
}

// TestHub_SetLayoutPersistsThroughStore is the direct
// end-to-end assertion that a UI write reaches disk.
func TestHub_SetLayoutPersistsThroughStore(t *testing.T) {
	path := filepath.Join(t.TempDir(), "layout.json")
	h := NewHub()
	if err := h.SetStore(NewLayoutStore(path)); err != nil {
		t.Fatalf("SetStore: %v", err)
	}
	if err := h.SetLayout(Layout{Theme: "light"}); err != nil {
		t.Fatalf("SetLayout: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("the layout was not written: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("wrote an empty layout file")
	}
	l, _, err := NewLayoutStore(path).Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if l.Theme != "light" {
		t.Errorf("persisted Theme = %q, want light", l.Theme)
	}
}

// TestHub_SetLayoutStillAppliesInMemoryOnStoreError asserts
// the in-memory update survives a persistence failure: the
// operator's drag took effect in the open tab even if the disk
// write failed.
func TestHub_SetLayoutStillAppliesInMemoryOnStoreError(t *testing.T) {
	// A path inside a file, so MkdirAll fails.
	dir := t.TempDir()
	blocker := filepath.Join(dir, "not-a-dir")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	h := NewHub()
	if err := h.SetStore(NewLayoutStore(filepath.Join(blocker, "layout.json"))); err != nil {
		t.Fatalf("SetStore: %v", err)
	}
	_ = h.SetLayout(Layout{Theme: "dark"})
	if h.Layout().Theme != "dark" {
		t.Error("the in-memory layout was discarded on a store failure")
	}
}
