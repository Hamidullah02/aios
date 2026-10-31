//go:build windows

package cli

import (
	"archive/zip"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestInstaller_AcceptsABinaryThatWritesToStderr is the test that
// reproduces the published install failure exactly.
//
// The installer that shipped in v0.57.0 ran `& $installed --help`
// inside a try/catch and treated anything thrown as "the binary did
// not run". Windows PowerShell turns a native command that writes
// to stderr into a terminating error when $ErrorActionPreference is
// "Stop" -- and the installer sets it to "Stop" -- so the check
// threw, the installer called Die, and the documented install
// command failed with "the installed binary did not run" for every
// user, no matter that the binary was working.
//
// `aios --help` wrote its usage to stderr, so the real binary
// always triggered it. The failure was reproduced against the
// published installer: the same binary that the published
// installer rejected is accepted by the fixed one.
//
// This test installs a binary that writes to stderr and exits 0 --
// the shape that broke the release -- and requires the installer
// to succeed. A binary that says something on the way out is
// normal; it is not a failure, and the installer must judge it by
// its exit code.
func TestInstaller_AcceptsABinaryThatWritesToStderr(t *testing.T) {
	if _, err := exec.LookPath("powershell.exe"); err != nil {
		t.Skip("powershell.exe not available")
	}

	root := repoRoot(t)
	script, err := os.ReadFile(filepath.Join(root, "install.ps1"))
	if err != nil {
		t.Fatal(err)
	}

	workDir := t.TempDir()
	assetName := "aios_windows_amd64.zip"

	// Build the binary that breaks the old installer: it writes to
	// stderr and exits 0.
	stderrBinary := buildStderrWritingBinary(t, workDir)
	assetPath := filepath.Join(workDir, assetName)
	if err := zipFile(assetPath, stderrBinary, "aios.exe"); err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if filepath.Base(r.URL.Path) != assetName {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, assetPath)
	}))
	defer srv.Close()

	local := strings.ReplaceAll(string(script), `$Repo = "Hamidullah02/aios"`, `$Repo = "local/test"`)
	local = strings.ReplaceAll(local,
		"https://github.com/$Repo/releases/download/$Version/$asset",
		srv.URL+"/$asset")
	if local == string(script) {
		t.Fatal("the installer no longer matches the shape this test patches; " +
			"the download URL in install.ps1 has changed")
	}

	localScript := filepath.Join(workDir, "install-local.ps1")
	if err := os.WriteFile(localScript, []byte(local), 0o644); err != nil {
		t.Fatal(err)
	}

	installDir := filepath.Join(workDir, "installed")
	cmd := exec.Command("powershell.exe", "-ExecutionPolicy", "Bypass", "-File", localScript,
		"-Version", "v0.0.0-test", "-InstallDir", installDir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("the installer rejected a working binary that wrote to stderr: %v\n%s\n"+
			"A binary that says something on the way out is not a broken binary; "+
			"the check must read the exit code.", err, out)
	}

	installed := filepath.Join(installDir, "aios.exe")
	if _, err := os.Stat(installed); err != nil {
		t.Fatalf("the installer did not put the binary in place: %v\n%s", err, out)
	}
}

// buildStderrWritingBinary compiles a binary that writes to stderr
// and exits 0 -- the shape that broke the published installer.
func buildStderrWritingBinary(t *testing.T, dir string) string {
	t.Helper()
	src := `package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "aios - the aios CLI")
	fmt.Fprintln(os.Stderr, "Usage:")
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Fprintln(os.Stdout, "aios v0.0.0-test")
	}
	os.Exit(0)
}
`
	srcDir := filepath.Join(dir, "src")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "main.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	mod := filepath.Join(srcDir, "go.mod")
	if err := os.WriteFile(mod, []byte("module stderrbinary\n\ngo 1.23\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out := filepath.Join(dir, "stderr-binary.exe")
	build := exec.Command("go", "build", "-o", out, ".")
	build.Dir = srcDir
	if b, err := build.CombinedOutput(); err != nil {
		t.Fatalf("building the test binary: %v\n%s", err, b)
	}
	return out
}

// zipFile writes a zip containing one file under the given name.
func zipFile(dest, source, entryName string) error {
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	w := zip.NewWriter(out)
	entry, err := w.Create(entryName)
	if err != nil {
		return err
	}
	src, err := os.Open(source)
	if err != nil {
		return err
	}
	defer src.Close()
	if _, err := io.Copy(entry, src); err != nil {
		return err
	}
	return w.Close()
}
