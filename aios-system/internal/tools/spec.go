package tools

// Describe implements tools.Spec for ReadFileTool.
func (ReadFileTool) Describe() (string, string) {
	return "Read a file from the workspace. " +
			"Args: path (required, relative to the workspace or absolute inside it), " +
			"offset (optional int, 0-based line offset), limit (optional int, max lines).",
		`{"type":"object","required":["path"],"properties":{"path":{"type":"string"},` +
			`"offset":{"type":"integer"},"limit":{"type":"integer"}}}`
}

// Risk implements tools.RiskTier for ReadFileTool.
//
// Reading is tier 0: it changes nothing.
func (ReadFileTool) Risk() int { return 0 }

// Describe implements tools.Spec for WriteFileTool.
func (WriteFileTool) Describe() (string, string) {
	return "Write a file in the workspace atomically, replacing any existing content. " +
			"Args: path (required, inside the workspace), content (required string), " +
			"mode (optional int, default 0644).",
		`{"type":"object","required":["path","content"],"properties":{"path":{"type":"string"},` +
			`"content":{"type":"string"},"mode":{"type":"integer"}}}`
}

// Risk implements tools.RiskTier for WriteFileTool.
func (WriteFileTool) Risk() int { return 1 }

// Describe implements tools.Spec for SearchFilesTool.
func (SearchFilesTool) Describe() (string, string) {
	return "Find files in the workspace whose path matches a glob. " +
			"Args: pattern (required), include (optional glob the path must match), " +
			"exclude (optional glob the path must not match), max (optional int, default 1000).",
		`{"type":"object","required":["pattern"],"properties":{"pattern":{"type":"string"},` +
			`"include":{"type":"string"},"exclude":{"type":"string"},"max":{"type":"integer"}}}`
}

// Risk implements tools.RiskTier for SearchFilesTool.
func (SearchFilesTool) Risk() int { return 0 }

// Describe implements tools.Spec for RunShellTool.
func (RunShellTool) Describe() (string, string) {
	return "Run a shell command in the workspace. " +
			"Args: command (required, the argv as a list of strings), " +
			"timeout (optional int seconds, default 60), cwd (optional), env (optional map).",
		`{"type":"object","required":["command"],"properties":{"command":{"type":"array","items":{"type":"string"}},` +
			`"timeout":{"type":"integer"},"cwd":{"type":"string"},` +
			`"env":{"type":"object","additionalProperties":{"type":"string"}}}}`
}

// Risk implements tools.RiskTier for RunShellTool.
//
// Running a shell command is tier 2: it can do anything the
// process can do.
func (RunShellTool) Risk() int { return 2 }
