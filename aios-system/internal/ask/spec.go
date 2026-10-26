package ask

// Describe returns the tool's description and its JSON input
// schema.
//
// The schema lives here, next to the Question type it
// describes and the decodeQuestion function that reads it, so
// the three cannot drift apart. A planner that is told the
// wrong field names produces a question the operator cannot
// answer, and the failure looks like an operator problem
// rather than a planning one.
//
// The schema describes the JSON object form of the question
// because that is the form a model can produce.
func (a *AskUserTool) Describe() (string, string) {
	return "Ask the operator a multiple-choice question and record the answer. " +
			"Args: question (required, an object with id, prompt, and options; " +
			"each option has id and label, plus an optional description). " +
			"The question may also carry timeout_seconds and default_option.",
		`{"type":"object","required":["question"],"properties":{"question":{"type":"object",` +
			`"required":["id","prompt","options"],"properties":{"id":{"type":"string"},` +
			`"prompt":{"type":"string"},"options":{"type":"array","items":{"type":"object",` +
			`"required":["id","label"],"properties":{"id":{"type":"string"},` +
			`"label":{"type":"string"},"description":{"type":"string"}}}}},` +
			`"timeout_seconds":{"type":"integer"},"default_option":{"type":"string"}}}}`
}
