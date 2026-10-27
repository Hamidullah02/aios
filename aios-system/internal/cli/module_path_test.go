package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestModulePath_MatchesTheRepository pins the Go module path
// against the repository it lives in.
//
// The module path was github.com/sadi/aios-system -- a repository
// that does not exist and never did. The README told users to run
//
//	go install github.com/sadi/aios-system/cmd/aios@latest
//
// and that could not work, no matter how public the repository was:
// the tooling resolved the path to a repository that was not there.
//
// The path now names the real repository and the subdirectory the
// module actually lives in. This test fails if either drifts.
func TestModulePath_MatchesTheRepository(t *testing.T) {
	const (
		wantModule   = "github.com/Hamidullah02/aios/aios-system"
		wantRepo     = "Hamidullah02/aios"
		wantSubdir   = "aios-system"
		mustNotAppea = "github.com/sadi/aios-system"
	)

	// go.mod is the authority on the module path.
	gomod, err := os.ReadFile(filepath.Join("..", "..", "go.mod"))
	if err != nil {
		t.Fatalf("read go.mod: %v", err)
	}
	moduleLine := ""
	for _, line := range strings.Split(string(gomod), "\n") {
		if strings.HasPrefix(line, "module ") {
			moduleLine = strings.TrimSpace(strings.TrimPrefix(line, "module "))
			break
		}
	}
	if moduleLine != wantModule {
		t.Errorf("go.mod module = %q, want %q", moduleLine, wantModule)
	}

	// The module path must carry the repository name and the
	// subdirectory, or the Go tooling cannot resolve it.
	if !strings.Contains(moduleLine, wantRepo) {
		t.Errorf("module path %q does not name the repository %q", moduleLine, wantRepo)
	}
	if !strings.HasSuffix(moduleLine, "/"+wantSubdir) {
		t.Errorf("module path %q does not end in the subdirectory %q", moduleLine, wantSubdir)
	}

	// No Go file may IMPORT the old, non-existent path. It would
	// still compile locally -- the import is resolved from the
	// module, not the network -- so nothing else would catch it.
	//
	// Only import statements are checked. The old path also
	// appears in this file and in first_run_test.go, in the
	// negative assertions that guard against it coming back; a
	// blanket text search would flag those as failures.
	root := filepath.Join("..", "..")
	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			switch info.Name() {
			case "var", "vendor", ".git":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, line := range strings.Split(string(body), "\n") {
			trimmed := strings.TrimSpace(line)
			if !strings.HasPrefix(trimmed, "import ") && !strings.HasPrefix(trimmed, `"`) {
				continue
			}
			if !strings.Contains(trimmed, mustNotAppea) {
				continue
			}
			// A quoted line that mentions the old path is only an
			// import if it is a quoted path on its own.
			if strings.HasPrefix(trimmed, "import ") {
				trimmed = strings.TrimSpace(strings.TrimPrefix(trimmed, "import "))
			}
			if !strings.HasPrefix(trimmed, `"`) {
				continue
			}
			rel, _ := filepath.Rel(root, path)
			t.Errorf("%s still imports the old module path %q", rel, mustNotAppea)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
}
