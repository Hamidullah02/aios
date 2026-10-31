package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestLocalInstaller_ExistsAndIsWiredToTheModule pins that the
// local install path is there and points at the real module.
//
// There are two ways to install aios: download a prebuilt binary
// from a release, or build from the source tree. The second one is
// the only one that works on a machine with no release to download
// and no network to download it from -- a copy of the source, which
// is how the product actually gets used.
func TestLocalInstaller_ExistsAndIsWiredToTheModule(t *testing.T) {
	root := repoRoot(t)

	body, err := os.ReadFile(filepath.Join(root, "install-local.ps1"))
	if err != nil {
		t.Fatalf("install-local.ps1: %v\n"+
			"the local install path is the only one that works without a release", err)
	}
	ps := string(body)

	// It must build the module that is actually there.
	if !strings.Contains(ps, "aios-system") {
		t.Error("install-local.ps1 does not mention the aios-system module directory")
	}
	if !strings.Contains(ps, "go.mod") {
		t.Error("install-local.ps1 does not check that it found the module")
	}
	if !strings.Contains(ps, "./cmd/aios") {
		t.Error("install-local.ps1 does not build ./cmd/aios")
	}

	// It must not depend on the network. The whole point of it is
	// that a machine with the source needs neither GitHub nor a
	// release.
	for _, forbidden := range []string{
		"raw.githubusercontent.com",
		"api.github.com",
		"releases/download",
	} {
		if strings.Contains(ps, forbidden) {
			t.Errorf("install-local.ps1 refers to %q; the local install path must not need the network", forbidden)
		}
	}
}

// TestLocalInstaller_ChecksTheBinaryByItsExitCode pins that the
// local installer judges the binary the way the release installer
// does.
//
// This is the defect that broke the documented install command for
// every user: the check treated "the binary wrote to stderr" as "the
// binary did not run". Windows PowerShell raises a terminating
// error for a native command that writes to stderr under
// $ErrorActionPreference = "Stop", and it sets $LASTEXITCODE to -1
// rather than the real exit code, so neither the exception nor
// $LASTEXITCODE could tell a working binary from a broken one.
//
// The local installer has to read the exit code the way the release
// one now does, or it repeats the same failure for anyone who uses
// it.
func TestLocalInstaller_ChecksTheBinaryByItsExitCode(t *testing.T) {
	root := repoRoot(t)
	body, err := os.ReadFile(filepath.Join(root, "install-local.ps1"))
	if err != nil {
		t.Fatal(err)
	}
	ps := string(body)

	if strings.Contains(ps, "& $build --help | Out-Null") ||
		strings.Contains(ps, "& $installed --help | Out-Null") {
		t.Error("install-local.ps1 checks the binary by discarding its output; " +
			"a native command that writes to stderr raises a terminating error under " +
			"$ErrorActionPreference = \"Stop\", so that check fails for a working binary")
	}
	if !strings.Contains(ps, ".ExitCode") {
		t.Error("install-local.ps1 does not read an exit code; it must judge the binary by its exit status")
	}
}

// TestLocalInstaller_StampsTheVersion pins that the locally built
// binary can say what it is.
//
// A binary that cannot name its own build makes every bug report
// start with "which build is this?". The local build stamps the
// version and the commit, so it can answer.
func TestLocalInstaller_StampsTheVersion(t *testing.T) {
	root := repoRoot(t)
	body, err := os.ReadFile(filepath.Join(root, "install-local.ps1"))
	if err != nil {
		t.Fatal(err)
	}
	ps := string(body)

	if !strings.Contains(ps, "internal/version.Version") {
		t.Error("install-local.ps1 does not stamp the version into the binary")
	}
	if !strings.Contains(ps, "internal/version.Commit") {
		t.Error("install-local.ps1 does not stamp the commit into the binary")
	}
}

// TestLocalInstaller_CanSkipThePathUpdate pins that the PATH write
// can be turned off.
//
// Adding the install directory to the user PATH is a per-user
// registry write: it changes the environment of every process the
// user starts afterwards, on that machine, for good. That is right
// for an installer and wrong for a test.
func TestLocalInstaller_CanSkipThePathUpdate(t *testing.T) {
	root := repoRoot(t)
	body, err := os.ReadFile(filepath.Join(root, "install-local.ps1"))
	if err != nil {
		t.Fatal(err)
	}
	ps := string(body)

	if !strings.Contains(ps, "[switch]$NoPathUpdate") {
		t.Error("install-local.ps1 has no -NoPathUpdate switch")
	}
	if !strings.Contains(ps, "if ($NoPathUpdate)") {
		t.Error("install-local.ps1 ignores the -NoPathUpdate switch")
	}
}
