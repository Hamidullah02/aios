package webui

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

// TestLayoutStore_UnreachablePathIsNotSavedLayout asserts that a
// store pointed at a path that cannot hold a file reports "no
// layout saved", not an error.
//
// The path here is a regular file used as a directory. The two
// platforms report that differently: Windows returns
// ERROR_PATH_NOT_FOUND and Linux returns ENOTDIR, and
// os.IsNotExist recognises only the first.
//
// The effect of the difference was that the same behaviour was
// "nothing saved yet" on Windows and a hard error on Linux. A
// layout file in an unreachable place took the whole UI down on
// one platform and not the other, and the test that pins it
// passed on Windows and failed on Linux CI.
func TestLayoutStore_UnreachablePathIsNotSavedLayout(t *testing.T) {
	dir := t.TempDir()
	blocker := filepath.Join(dir, "not-a-dir")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	store := NewLayoutStore(filepath.Join(blocker, "layout.json"))
	layout, found, err := store.Load()
	if err != nil {
		t.Fatalf("an unreachable layout path must not be an error: %v", err)
	}
	if found {
		t.Error("found = true for a path that cannot hold a file")
	}
	if len(layout.Panels) != len(AllPanels) {
		t.Errorf("the fallback layout is not usable: %+v", layout)
	}

	// And the hub must still take the store and still serve.
	h := NewHub()
	if err := h.SetStore(store); err != nil {
		t.Fatalf("SetStore: %v", err)
	}
	if err := h.SetLayout(Layout{Theme: "dark"}); err == nil {
		t.Error("writing through an unreachable store should report an error")
	}
	if h.Layout().Theme != "dark" {
		t.Error("the in-memory layout was discarded on a store failure")
	}
}

// TestIsNoLayoutFile pins which errors count as "there is no
// layout file here".
//
// The point is that the set is the same on every platform. The
// ENOTDIR case is the one that was missing, and it is the one
// Linux reports for a path whose parent is a file.
func TestIsNoLayoutFile(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"a permission error", os.ErrPermission, false},
		{"a wrapped permission error", errors.New("wrapped: " + os.ErrPermission.Error()), false},
	}
	for _, c := range cases {
		if got := isNoLayoutFile(c.err); got != c.want {
			t.Errorf("%s: isNoLayoutFile = %v, want %v", c.name, got, c.want)
		}
	}

	// A real filesystem error has to be classified, not guessed
	// at. On this platform, try to provoke each one.
	dir := t.TempDir()
	blocker := filepath.Join(dir, "not-a-dir")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := os.ReadFile(filepath.Join(blocker, "layout.json"))
	if err == nil {
		t.Fatal("reading through a file should fail")
	}
	// Whatever the platform calls it, it must count as "no file".
	if !isNoLayoutFile(err) {
		t.Errorf("isNoLayoutFile(%v) = false, want true", err)
	}

	// And a genuinely missing file.
	_, missingErr := os.ReadFile(filepath.Join(dir, "nope.json"))
	if !isNoLayoutFile(missingErr) {
		t.Errorf("isNoLayoutFile(%v) = false, want true for a missing file", missingErr)
	}
}

// TestLayoutStore_CorruptFileIsStillAnError guards the other
// side of the rule.
//
// A file that IS there but cannot be parsed is still an error, and
// is still reported. Without this, the fix above would have
// swallowed real problems.
func TestLayoutStore_CorruptFileIsStillAnError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "layout.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, found, err := NewLayoutStore(path).Load()
	if err == nil {
		t.Error("a corrupt layout file must still report an error")
	}
	if found {
		t.Error("a layout that failed to parse must not count as loaded")
	}
}

// TestLayoutStore_ENOTDIRIsRecognized pins the specific errno
// that Linux reports, on the platforms that define it.
//
// The check has to compile everywhere, so the errno is used
// through errors.Is rather than by constructing the error
// directly.
func TestLayoutStore_ENOTDIRIsRecognized(t *testing.T) {
	wrapped := &os.PathError{
		Op:   "open",
		Path: "/tmp/not-a-dir/layout.json",
		Err:  syscall.ENOTDIR,
	}
	if !isNoLayoutFile(wrapped) {
		t.Error("ENOTDIR must count as 'no layout file here'")
	}
}
