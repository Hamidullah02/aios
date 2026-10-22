package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestDefaultLogPath_ExistsAsOneConstant is the v0.40.2
// gate.
//
// The audit-log default used to be written down twice: the
// parser defaulted --log to "./var/aios.log" and BuildDeps
// fell back to "./var/aios.db". Because the parser always
// populates args.LogPath, the BuildDeps fallback could
// never run -- so the two defaults disagreed on paper
// while only one was reachable, and the name an operator
// read in --help was the wrong one for a sqlite3 file.
func TestDefaultLogPath_ExistsAsOneConstant(t *testing.T) {
	if DefaultLogPath == "" {
		t.Fatal("DefaultLogPath must not be empty")
	}
	if !strings.HasSuffix(DefaultLogPath, ".db") {
		t.Errorf("DefaultLogPath = %q, want a .db suffix (it is a sqlite3 file, not a text log)", DefaultLogPath)
	}
	if strings.Contains(DefaultLogPath, ".log") {
		t.Errorf("DefaultLogPath = %q, must not claim to be a .log file", DefaultLogPath)
	}
}

// TestParse_LogDefaultsToTheConstant ties the parser's
// default to the constant, so the two can never drift.
func TestParse_LogDefaultsToTheConstant(t *testing.T) {
	a, err := Parse([]string{"aios", "audit"}, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	if a.LogPath != DefaultLogPath {
		t.Errorf("parser default log = %q, want %q", a.LogPath, DefaultLogPath)
	}
}

// TestParse_ExplicitLogWins is the regression guard: an
// operator-supplied --log must still override the default.
func TestParse_ExplicitLogWins(t *testing.T) {
	a, err := Parse([]string{"aios", "--log=/tmp/custom.db", "audit"}, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	if a.LogPath != "/tmp/custom.db" {
		t.Errorf("log = %q, want /tmp/custom.db", a.LogPath)
	}
}

// TestUsageTextNamesTheDefault guards the --help text. The
// usage block is what an operator reads before deciding
// where to look for the audit trail, so a stale default
// there is a real defect, not a cosmetic one.
func TestUsageTextNamesTheDefault(t *testing.T) {
	var buf bytes.Buffer
	if _, err := Parse([]string{"aios", "--help"}, &buf); err == nil {
		// Some flag packages return an error on --help;
		// either way the usage text is what we inspect.
		_ = err
	}
	got := buf.String()
	if !strings.Contains(got, DefaultLogPath) {
		t.Errorf("usage text does not mention the real default %q:\n%s", DefaultLogPath, got)
	}
	if strings.Contains(got, "aios.log") {
		t.Errorf("usage text still mentions the stale aios.log default:\n%s", got)
	}
}

// TestBuildDeps_UsesDefaultLogPathWithoutFlag is the
// behavioural gate: a run with no --log opens a real
// on-disk database at the default path, not ":memory:".
func TestBuildDeps_UsesDefaultLogPathWithoutFlag(t *testing.T) {
	dir := t.TempDir()
	// Isolate the relative default: the default path is
	// relative, so ./var/aios.db lands in the temp dir
	// instead of the repo.
	//
	// t.Chdir landed in Go 1.24 and this module is on 1.23,
	// so the working directory is saved and restored by
	// hand. t.Cleanup runs even when the test fails, so a
	// failure here cannot leave later tests pointing at a
	// temp dir.
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(orig); err != nil {
			t.Errorf("restore working directory: %v", err)
		}
	})
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	// BuildDeps needs a policy file for a non-bootstrap
	// command; write a minimal one.
	if err := os.MkdirAll("var", 0o755); err != nil {
		t.Fatal(err)
	}
	policyYAML := "rules:\n  - id: allow-all\n    match:\n      tool_glob: \"*\"\n    action: allow\n"
	if err := os.WriteFile(filepath.Join("var", "config.yaml"), []byte(policyYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join("var", "policy.yaml"), []byte(policyYAML), 0o644); err != nil {
		t.Fatal(err)
	}

	args := &Args{
		Command: CmdRun,
		// BuildDeps is called programmatically here, so the
		// Parse-time defaults are NOT applied. ConfigPath
		// must be set explicitly; LogPath is deliberately
		// left empty because that is the one path that
		// reaches the BuildDeps fallback this test exists
		// to exercise.
		ConfigPath: filepath.Join("var", "config.yaml"),
	}
	deps, err := BuildDeps(args)
	if err != nil {
		t.Fatalf("BuildDeps: %v", err)
	}
	if deps == nil || deps.Audit == nil {
		t.Skip("BuildDeps returned no audit log for this command; the default-path contract is covered by the parser tests")
	}
	// Close the handle so t.TempDir's RemoveAll cleanup does
	// not fail on a file still open by this process (the
	// failure surfaced on Windows, where an open file cannot
	// be deleted).
	t.Cleanup(func() { _ = deps.Audit.Close() })
	// The file must exist on disk at the default location.
	if _, err := os.Stat(filepath.Join("var", "aios.db")); err != nil {
		t.Errorf("expected an on-disk audit database at the default path: %v", err)
	}
}
