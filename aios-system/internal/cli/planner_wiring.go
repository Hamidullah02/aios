package cli

import (
	"github.com/sadi/aios-system/internal/planner"
)

// plannerToolSpecs is the planner's view of the production
// tool set. The planner does not import the bus's Tool type
// (it depends on aios-tools only through the prompt
// rendering layer); it consumes a flat {Name, Description,
// InputSchema, RiskTier} map. The descriptions and
// schemas are the strings the model reads when it
// decides which tool to dispatch to.
//
// The tool list mirrors the bus's tool map in
// wireAgentLoop. The two are deliberately kept in sync
// by hand: the bus's runtime contracts (file paths,
// argv, timeouts) match the planner's published
// schemas here. If a new tool is added to the bus,
// add a corresponding entry here. The reviewer
// flagged this duplication; the v0.32.0 refactor
// already noted that the planner and the bus are
// separated by design and the seam is a typed
// bridge.
//
// Risk tiers follow the bus's riskFromArgs default
// of 1, with read-only tools at 0 and shell at 2
// (matching the doc set's risk table).
func plannerToolSpecs() map[string]planner.ToolSpec {
	return map[string]planner.ToolSpec{
		"read_file": {
			Name:        "read_file",
			Description: "Read a file from the workspace. Args: path (required, relative to --workspace or absolute inside the workspace), offset (optional int), limit (optional int).",
			InputSchema: `{"type":"object","required":["path"],"properties":{"path":{"type":"string"},"offset":{"type":"integer"},"limit":{"type":"integer"}}}`,
			RiskTier:    0,
		},
		"search_files": {
			Name:        "search_files",
			Description: "Search the workspace for files whose path matches a glob. Args: pattern (required), include (optional glob), exclude (optional glob), max (optional int default 1000).",
			InputSchema: `{"type":"object","required":["pattern"],"properties":{"pattern":{"type":"string"},"include":{"type":"string"},"exclude":{"type":"string"},"max":{"type":"integer"}}}`,
			RiskTier:    0,
		},
		"write_file": {
			Name:        "write_file",
			Description: "Write a file in the workspace atomically. Args: path (required, inside the workspace), content (required string), mode (optional int, default 0644).",
			InputSchema: `{"type":"object","required":["path","content"],"properties":{"path":{"type":"string"},"content":{"type":"string"},"mode":{"type":"integer"}}}`,
			RiskTier:    1,
		},
		"run_shell": {
			Name:        "run_shell",
			Description: "Run a shell command. Args: command (required argv as a JSON array of strings), timeout (optional int seconds, default 60), cwd (optional), env (optional map).",
			InputSchema: `{"type":"object","required":["command"],"properties":{"command":{"type":"array","items":{"type":"string"}},"timeout":{"type":"integer"},"cwd":{"type":"string"},"env":{"type":"object","additionalProperties":{"type":"string"}}}}`,
			RiskTier:    2,
		},
		"ask_user": {
			Name:        "ask_user",
			Description: "Ask the operator a multiple-choice question. Args: question (required object: {id, prompt, options[]}). The answer is recorded in the audit log.",
			InputSchema: `{"type":"object","required":["question"],"properties":{"question":{"type":"object","required":["id","prompt","options"],"properties":{"id":{"type":"string"},"prompt":{"type":"string"},"options":{"type":"array","items":{"type":"object","required":["id","label"],"properties":{"id":{"type":"string"},"label":{"type":"string"},"description":{"type":"string"}}}}}}}}`,
			RiskTier:    0,
		},
	}
}
