package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SearchFilesTool searches files by name pattern
// (REQ-TOL-SR-001 to 003).
//
// The tool accepts:
//   - pattern (required): a glob for the file name (e.g.
//     "*.go" or "test_*.py"). The match is against the
//     base name, not the full path.
//   - include (optional): a glob the path must match. Useful
//     for scoping ("cmd/**/*.go" matches cmd/foo/bar.go).
//   - exclude (optional): a glob the path must NOT match.
//   - max (optional, default 1000): the maximum number of
//     results.
//
// The walk is bounded by the workspace: paths outside the
// workspace are skipped.
type SearchFilesTool struct{}

func (SearchFilesTool) Name() string { return "search_files" }

func (SearchFilesTool) Run(args map[string]any, ctx Context) (Result, error) {
	pattern, _ := args["pattern"].(string)
	if pattern == "" {
		return Result{OK: false, Summary: "missing pattern"}, fmt.Errorf("tools: search_files: pattern is required")
	}
	include, _ := args["include"].(string)
	exclude, _ := args["exclude"].(string)
	max := 1000
	if m, ok := toInt(args["max"]); ok == nil && m > 0 {
		max = m
	}
	ws := ctx.Workspace
	if ws == "" {
		var err error
		ws, err = Workspace()
		if err != nil {
			return Result{OK: false, Summary: "no workspace"}, err
		}
	}
	wsAbs, err := filepath.Abs(ws)
	if err != nil {
		return Result{OK: false, Summary: "workspace abs"}, err
	}

	var matches []string
	walkErr := filepath.WalkDir(wsAbs, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // skip unreadable entries
		}
		if d.IsDir() {
			return nil
		}
		// Path is the absolute path inside (or equal to) wsAbs.
		if !isUnder(path, wsAbs) {
			return nil
		}
		rel, _ := filepath.Rel(wsAbs, path)
		if include != "" {
			ok, _ := filepath.Match(include, filepath.ToSlash(rel))
			if !ok {
				return nil
			}
		}
		if exclude != "" {
			ok, _ := filepath.Match(exclude, filepath.ToSlash(rel))
			if ok {
				return nil
			}
		}
		base := filepath.Base(path)
		ok, _ := filepath.Match(pattern, base)
		if !ok {
			return nil
		}
		matches = append(matches, rel)
		if len(matches) >= max {
			return filepath.SkipAll
		}
		return nil
	})
	if walkErr != nil {
		return Result{OK: false, Summary: "walk error"}, walkErr
	}
	// Sort for determinism.
	sortStrings(matches)
	return Result{
		OK:      true,
		Output:  map[string]any{"matches": matches, "count": len(matches)},
		Summary: fmt.Sprintf("found %d matches", len(matches)),
	}, nil
}

// sortStrings is a tiny inline sort so we don't import
// "sort" just for one call.
func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		j := i
		for j > 0 && s[j-1] > s[j] {
			s[j-1], s[j] = s[j], s[j-1]
			j--
		}
	}
}

// keep the strings import live in case callers want to
// extend the tool.
var _ = strings.HasPrefix
