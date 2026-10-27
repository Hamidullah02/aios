package search

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Hamidullah02/aios/aios-system/internal/tools"
)

// GlobTool finds files by name pattern (REQ-SRH-GLB-001 to 006).
//
// Args:
//
//	pattern  (string, required) — basename glob, e.g. "*.go".
//	include  (string, optional) — path glob the match must hit.
//	exclude  (string, optional) — path glob the match must miss.
//	max      (int,    optional) — cap on result count (default 1000).
type GlobTool struct{}

func (GlobTool) Name() string { return "glob" }

func (GlobTool) Run(args map[string]any, ctx tools.Context) (tools.Result, error) {
	pattern, _ := args["pattern"].(string)
	if pattern == "" {
		return tools.Result{OK: false, Summary: "missing pattern"},
			fmt.Errorf("search: glob: pattern is required")
	}
	include, _ := args["include"].(string)
	exclude, _ := args["exclude"].(string)
	max := 1000
	if v, ok := args["max"].(int); ok && v > 0 {
		max = v
	}
	ws, err := tools.Workspace()
	if ctx.Workspace != "" {
		ws = ctx.Workspace
	}
	if err != nil && ctx.Workspace == "" {
		return tools.Result{OK: false, Summary: "no workspace"}, err
	}
	wsAbs, err := filepath.Abs(ws)
	if err != nil {
		return tools.Result{OK: false, Summary: "workspace abs"}, err
	}
	ig := loadIgnore(wsAbs)

	var matches []string
	walkErr := filepath.WalkDir(wsAbs, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(wsAbs, path)
		rel = filepath.ToSlash(rel)
		if ig.match(rel) {
			return nil
		}
		if include != "" {
			ok, _ := filepath.Match(include, rel)
			if !ok {
				return nil
			}
		}
		if exclude != "" {
			ok, _ := filepath.Match(exclude, rel)
			if ok {
				return nil
			}
		}
		if ok, _ := filepath.Match(pattern, filepath.Base(path)); ok {
			matches = append(matches, rel)
			if len(matches) >= max {
				return filepath.SkipAll
			}
		}
		return nil
	})
	if walkErr != nil {
		return tools.Result{OK: false, Summary: "walk error"}, walkErr
	}
	sortStrings(matches)
	return tools.Result{
		OK:      true,
		Output:  map[string]any{"matches": matches, "count": len(matches)},
		Summary: fmt.Sprintf("found %d matches", len(matches)),
	}, nil
}
