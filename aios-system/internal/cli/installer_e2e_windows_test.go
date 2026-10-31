//go:build windows

package cli

import (
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestInstaller_RunsEndToEndAgainstALocalRelease is the test that
// would have caught the installer defect.
//
// Every other installer test reads the script's text and looks for
// a pattern. That kind of test passes as long as the file still
// contains the right words, even when the script cannot run --
// which is exactly what happened: the script read perfectly and
// aborted on every user's machine.
//
// This one runs it. A local HTTP server stands in for GitHub, the
// real install.ps1 downloads the real release asset through it,
// and the script runs exactly as it would for a user.
//
// The regression it pins: the installer's smoke check used to
// treat "the binary wrote to stderr" as "the binary did not run".
// Under $ErrorActionPreference = "Stop" -- which the installer
// sets -- Windows PowerShell turns any native command that writes
// to stderr into a terminating error, so the check threw on every
// run and the documented install command always failed.
func TestInstaller_RunsEndToEndAgainstALocalRelease(t *testing.T) {
	if _, err := exec.LookPath("powershell.exe"); err != nil {
		t.Skip("powershell.exe not available")
	}

	root := repoRoot(t)
	scriptPath := filepath.Join(root, "install.ps1")
	script, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatal(err)
	}

	// Build the binary the way the release builds it, so the asset
	// is the real one.
	binDir := t.TempDir()
	binary := filepath.Join(binDir, "aios.exe")
	build := exec.Command("go", "build", "-trimpath", "-o", binary, "./cmd/aios")
	build.Dir = moduleRoot(t)
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}

	// Package it as the release asset.
	assetName := "aios_windows_amd64.zip"
	assetPath := filepath.Join(binDir, assetName)
	if err := zipFile(assetPath, binary, "aios.exe"); err != nil {
		t.Fatal(err)
	}

	// Stand in for GitHub.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if filepath.Base(r.URL.Path) != assetName {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, assetPath)
	}))
	defer srv.Close()

	// Point a copy of the installer at the local server. Everything
	// else -- the download, the unpacking, the smoke check, the
	// PATH update, the reporting -- runs as written.
	local := strings.ReplaceAll(string(script), `$Repo = "Hamidullah02/aios"`, `$Repo = "local/test"`)
	local = strings.ReplaceAll(local,
		"https://github.com/$Repo/releases/download/$Version/$asset",
		srv.URL+"/$asset")
	if local == string(script) {
		t.Fatal("the installer no longer matches the shape this test patches; " +
			"the download URL in install.ps1 has changed")
	}

	workDir := t.TempDir()
	localScript := filepath.Join(workDir, "install-local.ps1")
	if err := os.WriteFile(localScript, []byte(local), 0o644); err != nil {
		t.Fatal(err)
	}

	installDir := filepath.Join(workDir, "installed")
	// -NoPathUpdate: the PATH update is a per-user registry write,
	// and a test must not change the machine it runs on. On the CI
	// runner it failed outright with "the process (1168) has
	// exited".
	cmd := exec.Command("powershell.exe", "-ExecutionPolicy", "Bypass", "-File", localScript,
		"-Version", "v0.0.0-test", "-InstallDir", installDir, "-NoPathUpdate")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("the installer failed: %v\n%s\nThe documented install command must succeed.", err, out)
	}

	// The binary must be there, and it must run.
	installed := filepath.Join(installDir, "aios.exe")
	if _, err := os.Stat(installed); err != nil {
		t.Fatalf("the installer did not put the binary in place: %v\n%s", err, out)
	}

	ver := exec.Command(installed, "--version")
	verOut, err := ver.CombinedOutput()
	if err != nil {
		t.Fatalf("the installed binary does not run: %v\n%s", err, verOut)
	}
	if !strings.Contains(string(verOut), "aios") {
		t.Errorf("the installed binary does not say what it is: %q", verOut)
	}

	// The installer must have reported the version to the operator.
	if !strings.Contains(string(out), "aios") {
		t.Errorf("the installer did not report the installed build:\n%s", out)
	}
}

// moduleRoot returns the module directory (the parent of internal/cli).
func moduleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
	}
	t.Fatalf("cannot find the module root above %s", dir)
	return ""
}
