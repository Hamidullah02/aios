package tools

import (
	"fmt"
	"os"
	"path/filepath"
)

// WriteFileTool writes a file (REQ-TOL-WR-001 to 004).
//
// The tool accepts:
//   - path (required): the file path, absolute or relative
//     to the workspace.
//   - content (required): the file content.
//   - mode (optional, default 0644): the file mode.
//
// Writes are atomic: the content goes to a temp file in the
// same directory, fsync'd, then renamed to the target
// (TRD §9.2). On error, the temp file is removed.
type WriteFileTool struct{}

func (WriteFileTool) Name() string { return "write_file" }

func (WriteFileTool) Run(args map[string]any, ctx Context) (Result, error) {
	path, _ := args["path"].(string)
	if path == "" {
		return Result{OK: false, Summary: "missing path"}, fmt.Errorf("tools: write_file: path is required")
	}
	content, _ := args["content"].(string)
	if content == "" {
		if _, ok := args["content"]; !ok {
			return Result{OK: false, Summary: "missing content"}, fmt.Errorf("tools: write_file: content is required")
		}
	}
	ws := ctx.Workspace
	if ws == "" {
		var err error
		ws, err = Workspace()
		if err != nil {
			return Result{OK: false, Summary: "no workspace"}, err
		}
	}
	resolved, err := ResolvePath(ws, path)
	if err != nil {
		return Result{OK: false, Summary: "path rejected"}, err
	}
	mode := os.FileMode(0o644)
	if m, ok := args["mode"].(int); ok && m > 0 {
		mode = os.FileMode(m)
	}

	// Atomic write: temp file in the same directory, then rename.
	dir := filepath.Dir(resolved)
	tmp, err := os.CreateTemp(dir, ".write-*")
	if err != nil {
		return Result{OK: false, Summary: "create temp"}, err
	}
	tmpName := tmp.Name()
	defer func() {
		// Best-effort cleanup if the rename didn't happen.
		_ = os.Remove(tmpName)
	}()
	if _, err := tmp.WriteString(content); err != nil {
		tmp.Close()
		return Result{OK: false, Summary: "write temp"}, err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return Result{OK: false, Summary: "sync temp"}, err
	}
	if err := tmp.Close(); err != nil {
		return Result{OK: false, Summary: "close temp"}, err
	}
	if err := os.Chmod(tmpName, mode); err != nil {
		return Result{OK: false, Summary: "chmod temp"}, err
	}
	if err := os.Rename(tmpName, resolved); err != nil {
		return Result{OK: false, Summary: "rename temp"}, err
	}
	return Result{
		OK:      true,
		Output:  map[string]any{"path": resolved, "bytes": len(content)},
		Summary: fmt.Sprintf("wrote %d bytes", len(content)),
	}, nil
}
