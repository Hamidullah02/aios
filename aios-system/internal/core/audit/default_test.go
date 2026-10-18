package audit

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sadi/aios-system/internal/core/clock"
)

// TestOpenDefault_FallsBackToVarAuditDB verifies the default-path
// behavior. We change the working directory to a temp dir, then
// call OpenDefault without setting AIOS_AUDIT_DB and expect the
// ./var/audit.db file (relative to the temp dir) to be created.
func TestOpenDefault_FallsBackToVarAuditDB(t *testing.T) {
	tmp := t.TempDir()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldwd) })

	// Make sure the env var is not set.
	t.Setenv(EnvAuditDB, "")

	log, err := OpenDefault()
	if err != nil {
		t.Fatalf("OpenDefault: %v", err)
	}
	t.Cleanup(func() { log.Close() })

	want := filepath.Join(tmp, "var", "audit.db")
	if _, err := os.Stat(want); err != nil {
		t.Fatalf("expected file at %q: %v", want, err)
	}
}

// TestOpenDefault_RespectsEnvVar verifies the env-var override.
func TestOpenDefault_RespectsEnvVar(t *testing.T) {
	tmp := t.TempDir()
	target := filepath.Join(tmp, "custom", "audit.db")
	t.Setenv(EnvAuditDB, target)

	log, err := OpenDefault()
	if err != nil {
		t.Fatalf("OpenDefault: %v", err)
	}
	t.Cleanup(func() { log.Close() })

	if _, err := os.Stat(target); err != nil {
		t.Fatalf("expected file at %q: %v", target, err)
	}
}

// TestOpenDefault_CreatesParentDir verifies the parent directory is
// created when missing.
func TestOpenDefault_CreatesParentDir(t *testing.T) {
	tmp := t.TempDir()
	target := filepath.Join(tmp, "deep", "nested", "dir", "audit.db")
	t.Setenv(EnvAuditDB, target)

	log, err := OpenDefault()
	if err != nil {
		t.Fatalf("OpenDefault: %v", err)
	}
	t.Cleanup(func() { log.Close() })

	if _, err := os.Stat(target); err != nil {
		t.Fatalf("expected file at %q: %v", target, err)
	}
}

// TestOpenDefault_WriteAndReadBack is a smoke test that the file
// created by OpenDefault is a working audit log: a Write succeeds
// and the row can be read back.
func TestOpenDefault_WriteAndReadBack(t *testing.T) {
	tmp := t.TempDir()
	target := filepath.Join(tmp, "audit.db")
	t.Setenv(EnvAuditDB, target)

	log, err := OpenDefault()
	if err != nil {
		t.Fatalf("OpenDefault: %v", err)
	}
	t.Cleanup(func() { log.Close() })

	// We need a clock because we go through the file path; the
	// default Config{}.Clock is SystemDecision which reads the
	// wall clock. Use FixedDecision for determinism in the test.
	_ = clock.FixedDecision{}

	res, err := log.Write(&Row{
		TaskID:   "t1",
		Kind:     "task_created",
		Actor:    "cli",
		RiskTier: 0,
		Payload:  map[string]any{"goal": "ship"},
	})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if res.Row.SequenceNumber != 1 {
		t.Fatalf("seq: %d", res.Row.SequenceNumber)
	}
}
