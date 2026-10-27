package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Hamidullah02/aios/aios-system/internal/version"
)

// TestVersionFlag_PrintsTheBuild pins that `aios --version`
// answers.
//
// The question it answers is the first one in every bug report:
// which build is this. Before this, the flag did not exist, so
// `aios --version` failed with "flag provided but not defined" and
// the answer had to be found some other way -- usually by asking
// the user to read the CHANGELOG and guess.
func TestVersionFlag_PrintsTheBuild(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Main([]string{"aios", "--version"}, &stdout, &stderr)

	if code != ExitOK {
		t.Fatalf("exit = %d, want 0; stderr = %s", code, stderr.String())
	}
	got := stdout.String()
	if !strings.Contains(got, "aios") {
		t.Errorf("stdout = %q, want it to name the tool", got)
	}
	if !strings.Contains(got, version.Version) {
		t.Errorf("stdout = %q, want it to carry the version %q", got, version.Version)
	}
	if stderr.Len() != 0 {
		t.Errorf("stderr = %q, want nothing on stderr", stderr.String())
	}
}

// TestVersionFlag_NeedsNoSubcommand pins that --version works on
// its own.
//
// A user who types `aios --version` has not been told to pick a
// subcommand first, and the flag exists precisely so they do not
// have to. If the parser demanded a subcommand, the command would
// be useless in the one situation it is needed: something else is
// broken and the user is checking which build they have.
func TestVersionFlag_NeedsNoSubcommand(t *testing.T) {
	args, err := Parse([]string{"aios", "--version"}, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if !args.ShowVersion {
		t.Error("ShowVersion was not set")
	}
	if args.Command != "" {
		t.Errorf("Command = %q, want empty -- --version needs no subcommand", args.Command)
	}
}

// TestVersionFlag_WorksWithASubcommand pins that the flag is also
// accepted alongside a subcommand.
//
// `aios --version fix "do the thing"` should print the version and
// stop, not run the thing. A user who types the flag by habit
// should never accidentally start work.
func TestVersionFlag_WorksWithASubcommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Main([]string{"aios", "--version", "fix", "do the thing"}, &stdout, &stderr)

	if code != ExitOK {
		t.Fatalf("exit = %d, want 0; stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), version.Version) {
		t.Errorf("stdout = %q, want the version", stdout.String())
	}
	if strings.Contains(stdout.String(), "do the thing") {
		t.Error("--version must not start the work it was given alongside")
	}
}

// TestVersionString_IncludesTheCommitWhenKnown pins the shape of
// the version line.
//
// Two builds from the same tag can differ, so the commit is what
// tells them apart. When there is no commit -- a plain `go build` --
// the line is still a complete answer.
func TestVersionString_IncludesTheCommitWhenKnown(t *testing.T) {
	oldVersion, oldCommit := version.Version, version.Commit
	t.Cleanup(func() {
		version.Version, version.Commit = oldVersion, oldCommit
	})

	version.Version = "v1.2.3"
	version.Commit = ""
	if got := version.String(); got != "aios v1.2.3" {
		t.Errorf("String() = %q, want %q", got, "aios v1.2.3")
	}

	version.Commit = "abc1234"
	got := version.String()
	if !strings.Contains(got, "v1.2.3") || !strings.Contains(got, "abc1234") {
		t.Errorf("String() = %q, want it to carry both the version and the commit", got)
	}
}
