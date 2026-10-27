package webui

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
)

// LayoutStore persists an operator's panel arrangement.
//
// The hub holds the live layout in memory, which is the right
// source of truth while the UI is open. But an operator who
// rearranges panels, closes the server, and comes back later
// should find their arrangement intact: without this, every
// restart silently discarded the customisation.
//
// Scope is per operator, not per project: a panel layout is a
// preference about how someone likes to read a dashboard, not
// a property of the code being worked on. So it lives under
// $AIOS_HOME (default $HOME/.aios/) rather than in the
// project's ./var/, and switching projects does not rearrange
// someone's UI.
//
// A corrupt or unreadable file is not fatal. Losing a saved
// layout costs an operator one drag to restore; refusing to
// start the UI over it would be a worse trade.
type LayoutStore struct {
	path string
}

// NewLayoutStore returns a store backed by the file at path.
//
// A LayoutStore with an empty path is valid and inert: every
// operation succeeds and nothing is written. That is what a
// caller that does not want persistence uses.
func NewLayoutStore(path string) *LayoutStore {
	return &LayoutStore{path: path}
}

// DefaultLayoutPath returns the per-operator layout file
// path: $AIOS_HOME/layout.json, defaulting to
// $HOME/.aios/layout.json.
func DefaultLayoutPath() string {
	home := os.Getenv("AIOS_HOME")
	if home == "" {
		if h := os.Getenv("HOME"); h != "" {
			home = filepath.Join(h, ".aios")
		} else if h := os.Getenv("USERPROFILE"); h != "" {
			home = filepath.Join(h, ".aios")
		} else {
			home = "./.aios"
		}
	}
	return filepath.Join(home, "layout.json")
}

// Load returns the saved layout.
//
// The second return value reports whether a layout was
// actually loaded. A false first value with a nil error means
// "nothing saved yet", which is the normal first-run case and
// not a failure.
func (s *LayoutStore) Load() (Layout, bool, error) {
	if s == nil || s.path == "" {
		return DefaultLayout(), false, nil
	}
	data, err := os.ReadFile(s.path)
	if err != nil {
		if isNoLayoutFile(err) {
			return DefaultLayout(), false, nil
		}
		// Report the error but still hand back a usable
		// layout: the UI must render.
		return DefaultLayout(), false, err
	}
	var l Layout
	if err := json.Unmarshal(data, &l); err != nil {
		return DefaultLayout(), false, err
	}
	// Normalize what was on disk, because a layout saved by an
	// older build may name a panel that no longer exists or
	// predate a panel that now does.
	norm, err := l.Normalize()
	if err != nil {
		return DefaultLayout(), false, err
	}
	return norm, true, nil
}

// Save writes the layout, normalised first.
//
// Normalising on the way out means the file on disk is always
// valid: a later Load cannot fail on a shape this build would
// have rejected.
func (s *LayoutStore) Save(l Layout) error {
	if s == nil || s.path == "" {
		return nil
	}
	norm, err := l.Normalize()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(norm, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	// Write-then-rename so a crash mid-write cannot leave a
	// truncated layout that fails to parse on the next start.
	tmp, err := os.CreateTemp(dir, "layout-*.json")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, s.path); err != nil {
		os.Remove(tmpName)
		return err
	}
	return nil
}

// isNoLayoutFile reports whether err means "there is no layout
// file here" -- the ordinary first-run case -- rather than
// "something is wrong with the layout file".
//
// The check has to name more than os.IsNotExist, because the two
// platforms disagree about what a path whose parent is a regular
// file reports:
//
//   - Windows returns ERROR_PATH_NOT_FOUND, which os.IsNotExist
//     recognises.
//   - Linux returns ENOTDIR, which it does not.
//
// So a store pointed at a path that cannot exist was "nothing
// saved yet" on Windows and a hard error on Linux. The visible
// effect was that the same test passed on Windows and failed on
// Linux CI, and that a layout file in an unreachable place took
// the whole UI down with it on one platform and not the other.
//
// The rule is the one the user cares about: if the file is not
// there, there is no saved layout, and the server starts with the
// defaults. A file that IS there but cannot be read is still an
// error, and is still reported.
func isNoLayoutFile(err error) bool {
	if err == nil {
		return false
	}
	if os.IsNotExist(err) {
		return true
	}
	if errors.Is(err, fs.ErrNotExist) {
		return true
	}
	// ENOTDIR: a component of the path is not a directory, so
	// there is no file to read.
	if errors.Is(err, syscall.ENOTDIR) {
		return true
	}
	return false
}
