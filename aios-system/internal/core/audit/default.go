package audit

import (
	"fmt"
	"os"
	"path/filepath"
)

// EnvAuditDB is the environment variable that overrides the default
// audit log path. When set, its value is used verbatim. When unset
// or empty, OpenDefault falls back to DefaultPath.
//
// Per the aios-core TRD §11 and SRS §8 REQ-CORE-DAT-001.
const EnvAuditDB = "AIOS_AUDIT_DB"

// DefaultPath is the path used when AIOS_AUDIT_DB is not set.
//
// It is relative to the process's current working directory at the
// time OpenDefault is called, NOT at the time the binary was
// started. Callers that want a stable path should set the env var.
const DefaultPath = "var/audit.db"

// OpenDefault opens the audit log at the path named by the
// AIOS_AUDIT_DB environment variable, falling back to DefaultPath
// when the env var is unset or empty.
//
// The function creates the parent directory of the resolved path if
// it does not exist (mode 0o755). It does NOT create the file;
// Log.Write creates the file on first insert.
//
// Use Open directly if you want a non-filesystem-backed Log
// (e.g. ":memory:") or a different config (e.g. a fixed decision
// clock for tests).
func OpenDefault() (*Log, error) {
	path := os.Getenv(EnvAuditDB)
	if path == "" {
		path = DefaultPath
	}
	// Create the parent directory if needed. For ":memory:" this
	// is a no-op because filepath.Dir returns ".".
	if path != ":memory:" {
		dir := filepath.Dir(path)
		if dir != "" && dir != "." {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return nil, fmt.Errorf("audit.OpenDefault: mkdir %q: %w", dir, err)
			}
		}
	}
	log, err := Open(Config{Path: path})
	if err != nil {
		return nil, fmt.Errorf("audit.OpenDefault: %w", err)
	}
	return log, nil
}
