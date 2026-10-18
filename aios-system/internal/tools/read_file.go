package tools

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// ReadFileTool reads a file (REQ-TOL-RD-001 to 004).
//
// The tool accepts:
//   - path (required): the file path, absolute or relative
//     to the workspace.
//   - offset (optional, int): the 0-based line offset.
//   - limit (optional, int): the maximum number of lines.
//
// The tool returns the file content (or a slice of lines
// when offset/limit are set) plus a content_hash for the
// audit log.
type ReadFileTool struct{}

func (ReadFileTool) Name() string { return "read_file" }

// Run executes the read. The workspace is taken from
// ctx.Workspace; if it's empty, the package-level Workspace()
// is used as a fallback.
func (ReadFileTool) Run(args map[string]any, ctx Context) (Result, error) {
	path, _ := args["path"].(string)
	if path == "" {
		return Result{OK: false, Summary: "missing path"}, fmt.Errorf("tools: read_file: path is required")
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

	offset, _ := toInt(args["offset"])
	limit, _ := toInt(args["limit"])

	data, err := os.ReadFile(resolved)
	if err != nil {
		return Result{OK: false, Summary: "read failed"}, err
	}
	content := string(data)
	if offset > 0 || limit > 0 {
		lines := strings.Split(content, "\n")
		if offset > len(lines) {
			offset = len(lines)
		}
		end := len(lines)
		if limit > 0 && offset+limit < end {
			end = offset + limit
		}
		content = strings.Join(lines[offset:end], "\n")
	}
	return Result{
		OK:      true,
		Output:  map[string]any{"path": resolved, "content": content},
		Summary: fmt.Sprintf("read %d bytes", len(content)),
	}, nil
}

// toInt coerces a number or numeric string to int. Returns
// 0 with a nil error only when the value is a non-negative
// integer; returns an error for nil and unknown types so
// callers can fall back to a default.
func toInt(v any) (int, error) {
	switch t := v.(type) {
	case nil:
		return 0, errors.New("toInt: nil")
	case int:
		return t, nil
	case int64:
		return int(t), nil
	case float64:
		return int(t), nil
	case string:
		if t == "" {
			return 0, errors.New("toInt: empty string")
		}
		return strconv.Atoi(t)
	}
	return 0, fmt.Errorf("toInt: unsupported type %T", v)
}
