//go:build windows

package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestLocalInstaller_DoesNotStampAnotherRepository pins that the
// local installer takes the version and the commit from THIS
// project.
//
// The version and the commit are read with `git describe` and
// `git rev-parse`, which ask whatever repository the working
// directory is in. A copy of the source dropped into somebody else's
// git repository is exactly that: their repository.
//
// The result was a binary that announced itself as "aios v9.9.9
// (3bef6cd)" -- naming a version of aios that does not exist and a
// commit that has nothing to do with this code. A version string
// that lies about what it is is worse than one that says nothing,
// because it looks right: a bug report would quote it, and the
// commit it named would not exist in this repository.
//
// The stamp is only allowed when the module in the tree really is
// this project.
func TestLocalInstaller_DoesNotStampAnotherRepository(t *testing.T) {
	if _, err := exec.LookPath("powershell.exe"); err != nil {
		t.Skip("powershell.exe not available")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}

	root := repoRoot(t)
	workDir := t.TempDir()

	// A different repository, with its own tag, holding the source.
	other := filepath.Join(workDir, "theirproject")
	if err := os.MkdirAll(other, 0o755); err != nil {
		t.Fatal(err)
	}
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = other
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=someone", "GIT_AUTHOR_EMAIL=someone@example.com",
			"GIT_COMMITTER_NAME=someone", "GIT_COMMITTER_EMAIL=someone@example.com",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
		}
	}
	run("init", "-q", "-b", "main")
	if err := os.WriteFile(filepath.Join(other, "readme.txt"), []byte("their project\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", "-A")
	run("commit", "-q", "-m", "their first commit")
	run("tag", "v9.9.9")

	theirCommit := strings.TrimSpace(gitOutput(t, other, "rev-parse", "--short", "HEAD"))

	// Put the source inside their repository.
	copyTree(t, filepath.Join(root, "aios-system"), filepath.Join(other, "aios-system"))
	scriptBody, err := os.ReadFile(filepath.Join(root, "install-local.ps1"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(other, "install-local.ps1"), scriptBody, 0o644); err != nil {
		t.Fatal(err)
	}

	installDir := filepath.Join(workDir, "installed")
	cmd := exec.Command("powershell.exe", "-ExecutionPolicy", "Bypass", "-File",
		filepath.Join(other, "install-local.ps1"),
		"-InstallDir", installDir, "-NoPathUpdate", "-SkipTests")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("the local installer failed: %v\n%s", err, out)
	}

	installed := filepath.Join(installDir, "aios.exe")
	if _, err := os.Stat(installed); err != nil {
		t.Fatalf("the local installer did not put the binary in place: %v\n%s", err, out)
	}

	ver := exec.Command(installed, "--version")
	verOut, err := ver.CombinedOutput()
	if err != nil {
		t.Fatalf("the installed binary does not run: %v\n%s", err, verOut)
	}
	got := strings.TrimSpace(string(verOut))

	if strings.Contains(got, "v9.9.9") {
		t.Errorf("the binary claims the other repository's tag: %q", got)
	}
	if theirCommit != "" && strings.Contains(got, theirCommit) {
		t.Errorf("the binary claims the other repository's commit %q: %q", theirCommit, got)
	}
	// It must still be able to say what it is.
	if !strings.Contains(got, "aios") {
		t.Errorf("the binary does not say what it is: %q", got)
	}
}

// gitOutput runs git in dir and returns its trimmed output.
func gitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git %s: %v", strings.Join(args, " "), err)
	}
	return string(out)
}

// copyTree copies a directory tree, skipping the scratch directories.
func copyTree(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		// The scratch directories hold build output and journey
		// output; they are not part of the source.
		if info.IsDir() {
			switch info.Name() {
			case "var", ".git":
				return filepath.SkipDir
			}
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if strings.HasSuffix(info.Name(), ".exe") {
			return nil
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, body, 0o644)
	})
	if err != nil {
		t.Fatalf("copying %s: %v", src, err)
	}
}
