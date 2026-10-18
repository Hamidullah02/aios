package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// EnvWorkspace is the environment variable that overrides the
// workspace root. Per the aios-tools TRD §11. The default is
// the current working directory.
const EnvWorkspace = "AIOS_WORKSPACE"

// Workspace returns the absolute path of the workspace root.
// It honors $AIOS_WORKSPACE; otherwise it returns the
// current working directory.
//
// The path is resolved with filepath.Clean and
// filepath.EvalSymlinks (when the path exists); the result
// is a canonical absolute path suitable for prefix checks.
func Workspace() (string, error) {
	root := os.Getenv(EnvWorkspace)
	if root == "" {
		wd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("tools: workspace: getwd: %w", err)
		}
		root = wd
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("tools: workspace: abs: %w", err)
	}
	// Resolve symlinks when the path exists. EvalSymlinks
	// fails on non-existent paths; that's fine — the path
	// is still canonical for prefix checks.
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		abs = resolved
	}
	return filepath.Clean(abs), nil
}

// ResolvePath returns the absolute path of path joined with
// the workspace root, after path validation. The validation
// is the algorithm in the doc set (TRD §9.1): resolve to
// an absolute path, resolve the workspace to an absolute
// path, reject when the path is not under the workspace.
//
// ResolvePath returns *PathNotAllowedError when the path is
// outside the workspace. The error is wrapped around
// ErrPathNotAllowed so callers can use errors.Is.
func ResolvePath(workspace, path string) (string, error) {
	if workspace == "" {
		return "", fmt.Errorf("tools: resolve: workspace is empty")
	}
	if path == "" {
		return "", fmt.Errorf("tools: resolve: path is empty")
	}
	// Join with the workspace when the path is relative.
	joined := path
	if !filepath.IsAbs(path) {
		joined = filepath.Join(workspace, path)
	}
	abs, err := filepath.Abs(joined)
	if err != nil {
		return "", fmt.Errorf("tools: resolve: abs: %w", err)
	}
	// Resolve symlinks when the path exists. This catches
	// the "symlink points outside the workspace" attack.
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		abs = resolved
	}
	abs = filepath.Clean(abs)
	wsAbs, err := filepath.Abs(workspace)
	if err != nil {
		return "", fmt.Errorf("tools: resolve: workspace abs: %w", err)
	}
	if !isUnder(abs, wsAbs) {
		return "", &PathNotAllowedError{Path: path, Workspace: wsAbs}
	}
	return abs, nil
}

// isUnder reports whether path is under root. The check is a
// string-prefix comparison after path cleaning, which avoids
// the ".." traversal and the "/foo/bar2" / "/foo/bar"
// prefix-collision gotcha.
func isUnder(path, root string) bool {
	path = filepath.Clean(path)
	root = filepath.Clean(root)
	if path == root {
		return true
	}
	sep := string(filepath.Separator)
	return strings.HasPrefix(path, root+sep)
}
