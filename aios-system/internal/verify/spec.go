package verify

// Describe implements tools.Spec for RunTestsTool.
func (RunTestsTool) Describe() (string, string) {
	return "Run the project's test suite and report the results. " +
			"Args: command (optional, the test command as a list of strings; " +
			"defaults to the runner's own command, e.g. go test), " +
			"runner (optional: go-test, npm-test, pytest, or cargo-test), " +
			"path (optional, the directory to run in), timeout (optional int seconds).",
		`{"type":"object","properties":{"command":{"type":"array","items":{"type":"string"}},` +
			`"runner":{"type":"string"},"path":{"type":"string"},"timeout":{"type":"integer"}}}`
}

// Risk implements tools.RiskTier for RunTestsTool.
//
// The tool runs a subprocess, so it is tier 2 even though the
// command it runs is normally read-only.
func (RunTestsTool) Risk() int { return 2 }

// Describe implements tools.Spec for SelfCheckTool.
func (SelfCheckTool) Describe() (string, string) {
	return "Ask a judge model whether a result satisfies a goal. " +
			"Args: result (required, the output to judge), goal (required, what it should achieve).",
		`{"type":"object","required":["result","goal"],"properties":{"result":{"type":"string"},` +
			`"goal":{"type":"string"}}}`
}

// Risk implements tools.RiskTier for SelfCheckTool.
//
// The tool calls a model but changes nothing in the workspace.
func (SelfCheckTool) Risk() int { return 0 }
