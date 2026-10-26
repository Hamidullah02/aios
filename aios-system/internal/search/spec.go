package search

// Describe implements tools.Spec for GrepTool.
func (GrepTool) Describe() (string, string) {
	return "Search file contents in the workspace with a regular expression. Read-only. " +
			"Args: pattern (required, regex), path (optional, file or directory to search), " +
			"glob (optional, restrict to files matching this basename glob), " +
			"output_mode (optional: content, files_with_matches, or count; default content), " +
			"show_line_numbers (optional bool), max_results (optional int, default 1000), " +
			"timeout (optional duration string, default 30s).",
		`{"type":"object","required":["pattern"],"properties":{"pattern":{"type":"string"},` +
			`"path":{"type":"string"},"glob":{"type":"string"},"output_mode":{"type":"string"},` +
			`"show_line_numbers":{"type":"boolean"},"max_results":{"type":"integer"},` +
			`"timeout":{"type":"string"}}}`
}

// Risk implements tools.RiskTier for GrepTool.
func (GrepTool) Risk() int { return 0 }

// Describe implements tools.Spec for GlobTool.
func (GlobTool) Describe() (string, string) {
	return "Find files in the workspace by basename glob. Read-only. " +
			"Args: pattern (required, e.g. \"*.go\"), include (optional path glob the match must hit), " +
			"exclude (optional path glob the match must miss), max (optional int, default 1000).",
		`{"type":"object","required":["pattern"],"properties":{"pattern":{"type":"string"},` +
			`"include":{"type":"string"},"exclude":{"type":"string"},"max":{"type":"integer"}}}`
}

// Risk implements tools.RiskTier for GlobTool.
func (GlobTool) Risk() int { return 0 }

// Describe implements tools.Spec for CodeSearchTool.
func (CodeSearchTool) Describe() (string, string) {
	return "Search the workspace for a code symbol (function, type, or method). Read-only. " +
			"Args: symbol (required), kind (optional: func, type, or method), " +
			"lang (optional, restrict to one language), path (optional, search root), " +
			"max (optional int, default 200).",
		`{"type":"object","required":["symbol"],"properties":{"symbol":{"type":"string"},` +
			`"kind":{"type":"string"},"lang":{"type":"string"},"path":{"type":"string"},` +
			`"max":{"type":"integer"}}}`
}

// Risk implements tools.RiskTier for CodeSearchTool.
func (CodeSearchTool) Risk() int { return 0 }

// Describe implements tools.Spec for DiagnosticsTool.
func (DiagnosticsTool) Describe() (string, string) {
	return "Run a language checker (go vet, gofmt, tsc, ruff, eslint) over the workspace. Read-only. " +
			"Args: checker (optional, default \"go vet\"), path (optional, defaults to the workspace), " +
			"extra_args (optional list of strings passed to the checker).",
		`{"type":"object","properties":{"checker":{"type":"string"},"path":{"type":"string"},` +
			`"extra_args":{"type":"array","items":{"type":"string"}}}}`
}

// Risk implements tools.RiskTier for DiagnosticsTool.
//
// The checker runs a subprocess, so it is tier 2 even though it
// is meant to be read-only.
func (DiagnosticsTool) Risk() int { return 2 }
