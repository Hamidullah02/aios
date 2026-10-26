package memorytool

// kindEnum is the JSON Schema enum for the artifact kinds.
//
// The enum is in the schema, not just in the prose, because the
// schema is what the model reads when it picks an argument
// value. Without it the model invents one: a request to
// "remember that the project is called Acme" came back as
// `kind: fact`, and the tool answered with "memory: validate:
// kind must be one of task, file, decision, observation,
// tool_result, model_output; got \"fact\"". The agent was
// right about what it wanted to remember and the tool refused
// it over a spelling.
const kindEnum = `["task","file","decision","observation","tool_result","model_output"]`

// scopeEnum is the JSON Schema enum for the artifact scopes.
//
// The same reasoning as kindEnum: a scope the model guesses
// ("global") is rejected by the same validation.
const scopeEnum = `["session","project","user"]`

// Describe implements tools.Spec for ReadTool.
//
// The memory tools dispatch on an `action` argument, so the
// schema describes the action and the arguments each action
// takes rather than one flat argument list. A planner that
// omits `action` gets the default, which is what the tool
// itself does.
func (ReadTool) Describe() (string, string) {
	return "Read from the agent's memory. " +
			"Args: action (optional: search, recall, or stats; default search). " +
			"For search: kinds, scopes, tags, pattern, since, operator " +
			"(comma-separated strings), limit (int). " +
			"For recall: hash (required). For stats: none. " +
			"Valid kinds: task, file, decision, observation, tool_result, model_output. " +
			"Valid scopes: session, project, user.",
		`{"type":"object","properties":{"action":{"type":"string","enum":["search","recall","stats"]},` +
			`"kinds":{"type":"string","description":"comma-separated artifact kinds"},` +
			`"scopes":{"type":"string","description":"comma-separated artifact scopes"},` +
			`"tags":{"type":"string"},` +
			`"pattern":{"type":"string"},"since":{"type":"string"},"operator":{"type":"string"},` +
			`"limit":{"type":"integer"},"hash":{"type":"string"}}}`
}

// Risk implements tools.RiskTier for ReadTool.
//
// Reading the memory changes nothing.
func (ReadTool) Risk() int { return 0 }

// Describe implements tools.Spec for WriteTool.
func (WriteTool) Describe() (string, string) {
	return "Write to the agent's memory. " +
			"Args: action (optional: remember or forget; default remember). " +
			"For remember: content (required), kind, scope, session_id, task_id, " +
			"tags (comma-separated string). " +
			"For forget: hash (required). " +
			"Valid kinds: task, file, decision, observation, tool_result, model_output. " +
			"Valid scopes: session, project, user.",
		`{"type":"object","properties":{"action":{"type":"string","enum":["remember","forget"]},` +
			`"content":{"type":"string"},` +
			`"kind":{"type":"string","enum":` + kindEnum + `},` +
			`"scope":{"type":"string","enum":` + scopeEnum + `},` +
			`"session_id":{"type":"string"},"task_id":{"type":"string"},"tags":{"type":"string"},` +
			`"hash":{"type":"string"}}}`
}

// Risk implements tools.RiskTier for WriteTool.
//
// Writing to the memory persists state, so it is tier 1.
func (WriteTool) Risk() int { return 1 }
