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
//
// It uses the same canonicalPath helper ResolvePath uses, so
// the workspace and the paths inside it are always in the same
// form. On Windows that matters: EvalSymlinks expands an 8.3
// short name into its long form, so a workspace of
// C:\Users\RUNNER~1\... and a path of C:\Users\runneradmin\...
// are the same directory and must compare equal.
func Workspace() (string, error) {
	root := os.Getenv(EnvWorkspace)
	if root == "" {
		wd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("tools: workspace: getwd: %w", err)
		}
		root = wd
	}
	abs, err := canonicalPath(root)
	if err != nil {
		return "", fmt.Errorf("tools: workspace: abs: %w", err)
	}
	return abs, nil
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
	// The workspace and the path are canonicalized the same way,
	// and they have to be.
	//
	// On Windows, EvalSymlinks expands an 8.3 short name into its
	// long form: C:\Users\RUNNER~1\AppData becomes
	// C:\Users\runneradmin\AppData. If one side is expanded and the
	// other is not, the prefix check fails and every path inside
	// the workspace is reported as being outside it.
	//
	// That is what happened on the Windows CI runner: the
	// workspace was ...\RUNNER~1\AppData\Local\Temp\... and the
	// path resolved to ...\runneradmin\AppData\Local\Temp\..., so
	// every file tool -- read, write, edit, apply_patch -- refused
	// to do its job.
	wsAbs, err := canonicalPath(workspace)
	if err != nil {
		return "", fmt.Errorf("tools: resolve: workspace abs: %w", err)
	}
	abs, err := canonicalPath(joined)
	if err != nil {
		return "", fmt.Errorf("tools: resolve: abs: %w", err)
	}
	if !isUnder(abs, wsAbs) {
		return "", &PathNotAllowedError{Path: path, Workspace: wsAbs}
	}
	return abs, nil
}

// canonicalPath returns the absolute, symlink-resolved form of
// path.
//
// The symlink resolution is best-effort: EvalSymlinks fails on a
// path that does not exist, and a path that does not exist is
// still perfectly safe to prefix-check, because it cannot be a
// symlink pointing out of the workspace.
//
// It is not that simple, though. EvalSymlinks fails on the WHOLE
// path when any part of it does not exist -- and the file a tool
// is about to create does not exist yet. So a path of
// C:\Users\RUNNER~1\AppData\Local\Temp\x\a.txt resolves to
// nothing, and the short name stays in it, while the workspace --
// which does exist -- is expanded to
// C:\Users\runneradmin\AppData\Local\Temp\x.
//
// The two no longer match, and every path inside the workspace is
// reported as being outside it. That is what the Windows CI runner
// found, and it broke every file tool at once.
//
// So the resolution is done on the longest existing ancestor, and
// the part that does not exist yet is re-attached to it. The short
// name is expanded, and a symlink that points out of the workspace
// is still followed and still caught.
func canonicalPath(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	abs = filepath.Clean(abs)

	// The whole path exists: resolve it in one go.
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		return resolved, nil
	}

	// Some part of it does not exist. Resolve the longest
	// existing ancestor and re-attach the rest.
	ancestor, rest := splitAtExistingAncestor(abs)
	if ancestor == "" {
		// Nothing exists, not even the root. The cleaned path
		// is the best answer available.
		return abs, nil
	}
	resolved, err := filepath.EvalSymlinks(ancestor)
	if err != nil {
		return abs, nil
	}
	if rest == "" {
		return resolved, nil
	}
	return filepath.Join(resolved, rest), nil
}

// splitAtExistingAncestor splits an absolute path into the
// longest ancestor that exists as a directory, and the remainder
// that does not exist yet.
//
// The remainder is returned in path order, so joining the two
// back together gives the original path.
func splitAtExistingAncestor(abs string) (ancestor, rest string) {
	rest = ""
	current := abs
	for {
		parent := filepath.Dir(current)
		if parent == current {
			// The root was reached without finding anything
			// that exists.
			return "", ""
		}
		rest = filepath.Join(filepath.Base(current), rest)
		current = parent
		if info, err := os.Stat(current); err == nil && info.IsDir() {
			return current, rest
		}
	}
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
