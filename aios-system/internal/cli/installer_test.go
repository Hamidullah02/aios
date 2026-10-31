package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestInstaller_SmokeCheckReadsTheExitCode pins that the
// installer's smoke check asks the binary for its exit code
// instead of trying to interpret its output.
//
// This is the defect that made the documented install command fail
// for everybody.
//
// The installer ran `& $installed --help` inside a try/catch and
// treated any exception as "the binary did not run". Windows
// PowerShell raises a terminating error for a native command that
// writes to stderr when $ErrorActionPreference is "Stop" -- and
// the installer sets it to "Stop" so that a failed download stops
// the script. `aios --help` wrote to stderr, so the check threw
// every single time, and the installer aborted with "the
// installed binary did not run" for a binary that was working
// perfectly.
//
// The second half of the bug is quieter: when PowerShell converts
// that stderr into a terminating error it also sets $LASTEXITCODE
// to -1, not the real exit code. So a check that only looked at
// $LASTEXITCODE in the catch block could not tell a working
// binary from a broken one either.
//
// The check has to read the exit code from a mechanism that reports
// it truthfully, and it must not be fooled by output on stderr.
func TestInstaller_SmokeCheckReadsTheExitCode(t *testing.T) {
	root := repoRoot(t)
	body, err := os.ReadFile(filepath.Join(root, "install.ps1"))
	if err != nil {
		t.Fatal(err)
	}
	ps := string(body)

	// The old check.
	if strings.Contains(ps, "& $installed --help | Out-Null") {
		t.Error("the installer still checks by running --help and discarding the output; " +
			"a native command that writes to stderr raises a terminating error under " +
			"$ErrorActionPreference = \"Stop\", so this check fails for a working binary")
	}

	// The check must read an exit code.
	if !strings.Contains(ps, ".ExitCode") {
		t.Error("the installer does not read an exit code; it must judge the binary by its exit status")
	}

	// And it must not judge the binary by whether stderr was quiet.
	if strings.Contains(ps, "$ErrorActionPreference = \"Stop\"") &&
		strings.Contains(ps, "& $installed --help | Out-Null") {
		t.Error("the installer combines $ErrorActionPreference = \"Stop\" with an output-based check")
	}
}

// TestInstaller_ReportsTheInstalledVersion pins that the installer
// tells the operator which build it actually installed.
//
// An operator who files a bug report needs the version of the
// binary in front of them, not the one they meant to run. The
// installer used to ask for the version with `& $installed
// --version 2>$null`, which has the same stderr problem as the
// smoke check it replaced: the version line was being read through
// a call that PowerShell could turn into a terminating error.
func TestInstaller_ReportsTheInstalledVersion(t *testing.T) {
	root := repoRoot(t)
	body, err := os.ReadFile(filepath.Join(root, "install.ps1"))
	if err != nil {
		t.Fatal(err)
	}
	ps := string(body)

	if !strings.Contains(ps, "--version") {
		t.Error("the installer does not ask the installed binary for its version")
	}
	if !strings.Contains(ps, "$installedVersion") {
		t.Error("the installer does not report a version to the operator")
	}
}
