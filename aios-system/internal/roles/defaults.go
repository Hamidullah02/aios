package roles

// DefaultRoles returns the canonical default
// role set (REQ-ROL-DEF-001).
//
// The set covers the common cases the planner
// and the orchestrator reach for:
//
//   - planner  — strong / plan output.
//   - coder    — strong / code output.
//   - editor   — mid    / code output.
//   - explorer — cheap  / text output; read-only
//     tools.
//   - reviewer — strong / text output.
//   - verifier — strong / json output.
//
// The system prompts are minimal placeholders.
// The CLI is the layer that lets the operator
// edit the prompts; the package exports the
// defaults the loader's seed-data path uses
// when the operator runs `aios init`.
func DefaultRoles() []Role {
	return []Role{
		{
			Name:         "planner",
			SystemPrompt: "You are the planner. Decompose the task into a plan of dependent steps. Output JSON plan only.",
			Tools:        []string{"read", "search", "edit", "multi_edit", "write", "run_shell", "ask"},
			ModelClass:   ClassStrong,
			OutputFormat: OutputJSON,
			Temperature:  0,
		},
		{
			Name:         "coder",
			SystemPrompt: "You are the coder. Write and edit source code through the edit tool. Output code only.",
			Tools:        []string{"read", "search", "edit", "multi_edit", "write", "run_shell"},
			ModelClass:   ClassStrong,
			OutputFormat: OutputCode,
			Temperature:  0,
		},
		{
			Name:         "editor",
			SystemPrompt: "You are the editor. Apply small, targeted edits through edit / multi_edit. Output code only.",
			Tools:        []string{"read", "search", "edit", "multi_edit"},
			ModelClass:   ClassMid,
			OutputFormat: OutputCode,
			Temperature:  0,
		},
		{
			Name:         "explorer",
			SystemPrompt: "You are the explorer. Read and search the codebase. Read-only.",
			Tools:        []string{"read", "search", "glob", "grep"},
			ModelClass:   ClassCheap,
			OutputFormat: OutputText,
			Temperature:  0,
		},
		{
			Name:         "reviewer",
			SystemPrompt: "You are the reviewer. Read the diff and check correctness, safety, and style. Output text only.",
			Tools:        []string{"read", "search", "run_shell"},
			ModelClass:   ClassStrong,
			OutputFormat: OutputText,
			Temperature:  0,
		},
		{
			Name:         "verifier",
			SystemPrompt: "You are the verifier. Run the test suite and the lint, then return JSON {ok: bool, reason: string}.",
			Tools:        []string{"read", "run_shell"},
			ModelClass:   ClassStrong,
			OutputFormat: OutputJSON,
			Temperature:  0,
		},
	}
}

// DefaultRegistry returns a registry
// pre-populated with the default roles. The
// function is the convenience constructor the
// CLI uses when the operator has not yet
// created ./var/roles.yaml.
func DefaultRegistry() (*RoleRegistry, error) {
	return NewRegistry(DefaultRoles())
}
