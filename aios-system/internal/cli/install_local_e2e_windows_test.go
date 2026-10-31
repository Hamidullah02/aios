//go:build windows

package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestLocalInstaller_BuildsAndInstallsFromThisTree is the test that
// proves the local install path works.
//
// The pattern-based tests can only say that the script contains the
// right words. This one runs it: it builds the module, and the
// script installs the binary it built.
//
// The build is the slow part, so the script is given -SkipTests --
// the suite is run separately by CI, and running it twice in one
// test would double the time for no extra coverage.
func TestLocalInstaller_BuildsAndInstallsFromThisTree(t *testing.T) {
	if _, err := exec.LookPath("powershell.exe"); err != nil {
		t.Skip("powershell.exe not available")
	}

	root := repoRoot(t)
	script := filepath.Join(root, "install-local.ps1")
	if _, err := os.Stat(script); err != nil {
		t.Fatalf("install-local.ps1: %v", err)
	}

	workDir := t.TempDir()
	installDir := filepath.Join(workDir, "installed")

	cmd := exec.Command("powershell.exe", "-ExecutionPolicy", "Bypass", "-File", script,
		"-InstallDir", installDir, "-NoPathUpdate", "-SkipTests")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("the local installer failed: %v\n%s\n"+
			"Building from the source tree is the install path that works without a release.",
			err, out)
	}

	installed := filepath.Join(installDir, "aios.exe")
	if _, err := os.Stat(installed); err != nil {
		t.Fatalf("the local installer did not put the binary in place: %v\n%s", err, out)
	}

	// The installed binary must run.
	ver := exec.Command(installed, "--version")
	verOut, err := ver.CombinedOutput()
	if err != nil {
		t.Fatalf("the installed binary does not run: %v\n%s", err, verOut)
	}
	if !strings.Contains(string(verOut), "aios") {
		t.Errorf("the installed binary does not say what it is: %q", verOut)
	}

	// It must have been stamped with the commit it was built from,
	// so a bug report can name the build.
	commit := gitShortCommit(t, root)
	if commit != "" && !strings.Contains(string(verOut), commit) {
		t.Errorf("the installed binary does not carry the commit %q: %q", commit, verOut)
	}
}

// gitShortCommit returns the short commit of the tree, or "" when it
// is not a git checkout.
func gitShortCommit(t *testing.T, dir string) string {
	t.Helper()
	cmd := exec.Command("git", "rev-parse", "--short", "HEAD")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
