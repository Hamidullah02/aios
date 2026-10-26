package edit

// Describe implements tools.Spec for the edit package's tools.
//
// The descriptions and schemas live here, next to the Run
// methods that read the args, so they cannot drift apart. A
// planner that is told the wrong argument names produces a task
// that fails at the tool call with "old_string is required" and
// no hint that the model had been misled.
func (EditTool) Describe() (string, string) {
	return "Replace old_string with new_string in one file. " +
			"Args: path (required, inside the workspace), old_string (required), " +
			"new_string (required), replace_all (optional bool, default false).",
		`{"type":"object","required":["path","old_string","new_string"],` +
			`"properties":{"path":{"type":"string"},"old_string":{"type":"string"},` +
			`"new_string":{"type":"string"},"replace_all":{"type":"boolean"}}}`
}

// Risk implements tools.RiskTier for EditTool.
//
// Editing a file is tier 1: it changes the workspace but does
// not run anything.
func (EditTool) Risk() int { return 1 }

// Describe implements tools.Spec for MultiEditTool.
func (MultiEditTool) Describe() (string, string) {
	return "Apply a list of edits to one file in a single write. " +
			"Args: path (required, inside the workspace), edits (required list of " +
			"objects, each with old_string, new_string, and optional replace_all).",
		`{"type":"object","required":["path","edits"],"properties":{"path":{"type":"string"},` +
			`"edits":{"type":"array","items":{"type":"object","required":["old_string","new_string"],` +
			`"properties":{"old_string":{"type":"string"},"new_string":{"type":"string"},` +
			`"replace_all":{"type":"boolean"}}}}}}`
}

// Risk implements tools.RiskTier for MultiEditTool.
func (MultiEditTool) Risk() int { return 1 }

// Describe implements tools.Spec for ApplyPatchTool.
func (ApplyPatchTool) Describe() (string, string) {
	return "Apply a unified diff to one or more files in the workspace. " +
			"Args: patch (required, the unified diff text).",
		`{"type":"object","required":["patch"],"properties":{"patch":{"type":"string"}}}`
}

// Risk implements tools.RiskTier for ApplyPatchTool.
func (ApplyPatchTool) Risk() int { return 1 }

// Describe implements tools.Spec for WriteTool.
//
// The tool re-exports tools.WriteFileTool under the name
// "write"; the schema is the same as write_file's.
func (WriteTool) Describe() (string, string) {
	return "Write a file in the workspace atomically, replacing any existing content. " +
			"Args: path (required, inside the workspace), content (required string), " +
			"mode (optional int, default 0644).",
		`{"type":"object","required":["path","content"],"properties":{"path":{"type":"string"},` +
			`"content":{"type":"string"},"mode":{"type":"integer"}}}`
}

// Risk implements tools.RiskTier for WriteTool.
func (WriteTool) Risk() int { return 1 }

// Describe implements tools.Spec for NotebookEditTool.
func (NotebookEditTool) Describe() (string, string) {
	return "Update a cell of a Jupyter notebook. " +
			"Args: path (required, inside the workspace), cell_id (required), " +
			"new_source (optional), cell_type (optional: code or markdown), " +
			"cell_mode (optional: replace, insert, or delete; default replace).",
		`{"type":"object","required":["path","cell_id"],"properties":{"path":{"type":"string"},` +
			`"cell_id":{"type":"string"},"new_source":{"type":"string"},` +
			`"cell_type":{"type":"string"},"cell_mode":{"type":"string"}}}`
}

// Risk implements tools.RiskTier for NotebookEditTool.
func (NotebookEditTool) Risk() int { return 1 }
