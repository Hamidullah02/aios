package edit

import (
	"fmt"
	"os"
	"path/filepath"
)

// atomicWrite writes data to a temp file in the same
// directory and renames it over dst. The temp file is
// removed on any failure. This is the atomic-write path
// shared by every edit tool (REQ-EDT-NFR-003).
func atomicWrite(dst string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(dst)
	tmp, err := os.CreateTemp(dir, ".edit-*")
	if err != nil {
		return fmt.Errorf("edit: create temp: %w", err)
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("edit: write temp: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("edit: sync temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("edit: close temp: %w", err)
	}
	if err := os.Chmod(tmpName, mode); err != nil {
		return fmt.Errorf("edit: chmod: %w", err)
	}
	if err := os.Rename(tmpName, dst); err != nil {
		return fmt.Errorf("edit: rename: %w", err)
	}
	return nil
}
