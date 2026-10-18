package edit

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/sadi/aios-system/internal/tools"
)

// NotebookEditTool updates a Jupyter notebook cell
// (REQ-EDT-NTE-001 to 006). The tool reads the notebook,
// finds the cell, applies the change, and writes the
// notebook back atomically.
//
// Args:
//
//	path      (string, required)
//	cell_id   (string, required) — the cell's id field
//	new_source (string, optional) — the replacement source
//	cell_type (string, optional) — "code" or "markdown"
//	cell_mode (string, optional, default "replace")
//	           — "replace", "insert", or "delete"
type NotebookEditTool struct{}

func (NotebookEditTool) Name() string { return "notebook_edit" }

func (NotebookEditTool) Run(args map[string]any, ctx tools.Context) (tools.Result, error) {
	path, _ := args["path"].(string)
	if path == "" {
		return tools.Result{OK: false, Summary: "missing path"},
			fmt.Errorf("edit: path is required")
	}
	cellID, _ := args["cell_id"].(string)
	if cellID == "" {
		return tools.Result{OK: false, Summary: "missing cell_id"},
			fmt.Errorf("edit: cell_id is required")
	}
	newSource, _ := args["new_source"].(string)
	cellType, _ := args["cell_type"].(string)
	mode, _ := args["cell_mode"].(string)
	if mode == "" {
		mode = "replace"
	}
	if mode == "replace" || mode == "insert" {
		if cellType != "code" && cellType != "markdown" {
			return tools.Result{OK: false, Summary: "bad cell_type"},
				fmt.Errorf("edit: cell_type must be code or markdown")
		}
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

	data, err := os.ReadFile(resolved)
	if err != nil {
		return tools.Result{OK: false, Summary: "read failed"}, err
	}
	var nb notebook
	if err := json.Unmarshal(data, &nb); err != nil {
		return tools.Result{OK: false, Summary: "notebook json"}, err
	}

	switch mode {
	case "replace":
		idx := -1
		for i := range nb.Cells {
			if nb.Cells[i].ID == cellID {
				idx = i
				break
			}
		}
		if idx < 0 {
			return tools.Result{OK: false, Summary: "cell not found"},
				fmt.Errorf("edit: cell %q not found", cellID)
		}
		nb.Cells[idx].CellType = cellType
		nb.Cells[idx].Source = []string{newSource}
	case "insert":
		nb.Cells = append(nb.Cells, notebookCell{
			ID:       cellID,
			CellType: cellType,
			Source:   []string{newSource},
		})
	case "delete":
		out := nb.Cells[:0]
		removed := false
		for _, c := range nb.Cells {
			if c.ID == cellID {
				removed = true
				continue
			}
			out = append(out, c)
		}
		if !removed {
			return tools.Result{OK: false, Summary: "cell not found"},
				fmt.Errorf("edit: cell %q not found", cellID)
		}
		nb.Cells = out
	default:
		return tools.Result{OK: false, Summary: "bad mode"},
			fmt.Errorf("edit: cell_mode must be replace, insert, or delete")
	}

	out, err := json.MarshalIndent(nb, "", "  ")
	if err != nil {
		return tools.Result{OK: false, Summary: "marshal"}, err
	}
	info, _ := os.Stat(resolved)
	mode2 := os.FileMode(0o644)
	if info != nil {
		mode2 = info.Mode().Perm()
	}
	if err := atomicWrite(resolved, out, mode2); err != nil {
		return tools.Result{OK: false, Summary: "write failed"}, err
	}
	return tools.Result{
		OK:      true,
		Output:  map[string]any{"path": resolved, "cell_id": cellID, "mode": mode},
		Summary: fmt.Sprintf("notebook %s: cell %q %s", mode, cellID, resolved),
	}, nil
}

// notebook is the minimal subset of nbformat v4 we need.
type notebook struct {
	Cells []notebookCell `json:"cells"`
}

type notebookCell struct {
	ID       string   `json:"id"`
	CellType string   `json:"cell_type"`
	Source   []string `json:"source"`
}
