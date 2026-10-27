package edit

import (
	"fmt"
	"os"
	"strings"

	"github.com/Hamidullah02/aios/aios-system/internal/tools"
)

// MultiEditTool applies a list of edits in memory
// (REQ-EDT-MED-001 to 005). The tool reads the file once,
// applies every edit to the in-memory buffer, and writes
// the final result. When an edit fails, the original
// content is written back so the file is never left half-
// edited (REQ-EDT-NFR-003).
//
// Args:
//
//	path    (string, required)
//	edits   ([]map[string]any, required) — each entry has
//	          old_string, new_string, optional replace_all.
type MultiEditTool struct{}

func (MultiEditTool) Name() string { return "multi_edit" }

func (MultiEditTool) Run(args map[string]any, ctx tools.Context) (tools.Result, error) {
	path, _ := args["path"].(string)
	if path == "" {
		return tools.Result{OK: false, Summary: "missing path"},
			fmt.Errorf("edit: path is required")
	}
	edits, _ := args["edits"].([]any)
	if len(edits) == 0 {
		return tools.Result{OK: false, Summary: "no edits"},
			fmt.Errorf("edit: edits list is required")
	}
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

	original, err := os.ReadFile(resolved)
	if err != nil {
		return tools.Result{OK: false, Summary: "read failed"}, err
	}
	info, _ := os.Stat(resolved)
	mode := os.FileMode(0o644)
	if info != nil {
		mode = info.Mode().Perm()
	}

	// Apply each edit to the buffer. On failure, write the
	// original back so the file is never half-edited.
	buffer := string(original)
	combined := strings.Builder{}
	for i, raw := range edits {
		e, _ := raw.(map[string]any)
		oldStr, _ := e["old_string"].(string)
		if oldStr == "" {
			_ = atomicWrite(resolved, original, mode)
			return tools.Result{OK: false, Summary: "empty old_string"},
				&MultiEditError{Path: resolved, Index: i, Reason: &EditEmptyError{Path: resolved}}
		}
		newStr, _ := e["new_string"].(string)
		replaceAll, _ := e["replace_all"].(bool)
		count := strings.Count(buffer, oldStr)
		if count == 0 {
			_ = atomicWrite(resolved, original, mode)
			return tools.Result{OK: false, Summary: "old_string not found"},
				&MultiEditError{Path: resolved, Index: i, Reason: &EditNotFoundError{Path: resolved}}
		}
		if count > 1 && !replaceAll {
			_ = atomicWrite(resolved, original, mode)
			return tools.Result{OK: false, Summary: "old_string not unique"},
				&MultiEditError{Path: resolved, Index: i, Reason: &EditNotUniqueError{Path: resolved, Count: count}}
		}
		next := strings.ReplaceAll(buffer, oldStr, newStr)
		diff := unifiedDiff(buffer, next)
		if combined.Len() > 0 {
			combined.WriteString("\n")
		}
		combined.WriteString(diff)
		buffer = next
	}
	if buffer == string(original) {
		return tools.Result{OK: true, Output: map[string]any{
			"path": resolved, "applied": len(edits), "diff": "",
		}, Summary: "no change"}, nil
	}
	if err := atomicWrite(resolved, []byte(buffer), mode); err != nil {
		// Restore on write failure too.
		_ = atomicWrite(resolved, original, mode)
		return tools.Result{OK: false, Summary: "write failed"}, err
	}
	return tools.Result{
		OK: true,
		Output: map[string]any{
			"path":     resolved,
			"applied":  len(edits),
			"diff":     combined.String(),
			"new_size": len(buffer),
		},
		Summary: fmt.Sprintf("applied %d edits to %s", len(edits), resolved),
	}, nil
}
