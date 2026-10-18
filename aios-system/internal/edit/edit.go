package edit

import (
	"fmt"
	"os"
	"strings"

	"github.com/sadi/aios-system/internal/tools"
)

// EditTool replaces old_string with new_string in one file
// (REQ-EDT-EDT-001 to 009).
//
// Args:
//
//	path         (string, required)
//	old_string   (string, required)
//	new_string   (string, required)
//	replace_all  (bool, optional, default false)
type EditTool struct{}

func (EditTool) Name() string { return "edit" }

func (EditTool) Run(args map[string]any, ctx tools.Context) (tools.Result, error) {
	path, _ := args["path"].(string)
	if path == "" {
		return tools.Result{OK: false, Summary: "missing path"},
			fmt.Errorf("edit: path is required")
	}
	oldStr, _ := args["old_string"].(string)
	if oldStr == "" {
		return tools.Result{OK: false, Summary: "empty old_string"},
			&EditEmptyError{Path: path}
	}
	newStr, _ := args["new_string"].(string)
	replaceAll, _ := args["replace_all"].(bool)
	ws, err := tools.Workspace()
	if ctx.Workspace != "" {
		ws = ctx.Workspace
	}
	if err != nil && ctx.Workspace == "" {
		return tools.Result{OK: false, Summary: "no workspace"}, err
	}
	resolved, err := tools.ResolvePath(ws, path)
	if err != nil {
		return tools.Result{OK: false, Summary: "path rejected"}, err
	}

	data, err := os.ReadFile(resolved)
	if err != nil {
		return tools.Result{OK: false, Summary: "read failed"}, err
	}
	before := string(data)
	count := strings.Count(before, oldStr)
	if count == 0 {
		return tools.Result{OK: false, Summary: "old_string not found"},
			&EditNotFoundError{Path: resolved}
	}
	if count > 1 && !replaceAll {
		return tools.Result{OK: false, Summary: "old_string not unique"},
			&EditNotUniqueError{Path: resolved, Count: count}
	}
	after := strings.ReplaceAll(before, oldStr, newStr)
	if after == before {
		return tools.Result{OK: true, Output: map[string]any{
			"path": resolved, "count": 0, "diff": "",
		}, Summary: "no change"}, nil
	}
	info, _ := os.Stat(resolved)
	mode := os.FileMode(0o644)
	if info != nil {
		mode = info.Mode().Perm()
	}
	if err := atomicWrite(resolved, []byte(after), mode); err != nil {
		return tools.Result{OK: false, Summary: "write failed"}, err
	}
	diff := unifiedDiff(before, after)
	return tools.Result{
		OK: true,
		Output: map[string]any{
			"path":     resolved,
			"count":    count,
			"diff":     diff,
			"new_size": len(after),
		},
		Summary: fmt.Sprintf("replaced %d match(es) in %s", count, resolved),
	}, nil
}
