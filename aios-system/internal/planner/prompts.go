package planner

import (
	"bytes"
	"fmt"
	"sort"
	"text/template"
)

// PromptData is the data the planner passes to the prompt
// templates. The struct is the single source of truth for the
// variables the templates can use; it is also the public shape
// callers can fill in directly to test the rendering.
type PromptData struct {
	Goal     string
	Context  map[string]any
	Tools    []ToolSpec
	Errors   PlanErrorList
	Previous string // the previous plan JSON, set on repair iterations
	// Platform describes the machine the plan will run on.
	//
	// It is there because a model writing a shell command
	// without it writes a Unix one: `ls`, `date`, `cat`,
	// `echo`. On Windows those programs do not exist, so the
	// plan fails with "exec: \"ls\": executable file not found
	// in %PATH%" -- a failure that looks like a broken
	// installer and is actually a plan written for the wrong
	// operating system.
	Platform string
}

// ToolSpec is the tool map entry the planner hands to the model.
// The planner does not own the full Tool type from aios-tools
// (the design separates aios-tools from aios-planner); it just
// needs the name, the description, the JSON input schema, and the
// declared risk tier so the model can pick the right tool.
type ToolSpec struct {
	Name        string
	Description string
	InputSchema string // raw JSON schema; the model is told to honour it
	RiskTier    int
}

// planSystemTmpl is the system prompt. It is the same on the
// first plan call and on every repair call.
var planSystemTmpl = template.Must(template.New("plan_system").Parse(planSystemText))

const planSystemText = `You are aios-planner. Produce a JSON TaskPlan for the user's goal.

Output rules:

- Respond with JSON only. No prose, no markdown fences, no commentary.
- The JSON must match the TaskPlan schema:
  - "id" (string, required)
  - "goal" (string, required, equal to the user's goal)
  - "tasks" (array, required, at least one)
  - "tools" (array of strings, optional)
  - "model" (string, optional)
  - "budget" (object, optional) with "cost_usd" and "wall_clock_sec"
- Each task has: "id", "description", "tool" (optional, must be a
  registered tool name), "inputs" (array of strings, optional),
  "acceptance" (a testable statement).
- "inputs" holds the tool's arguments as "key=value" strings, one
  per entry, where each key is a property from that tool's input
  schema. For a tool whose schema requires "path", the entry is
  "path=notes.txt". Every required property in the schema must
  appear. A bare value with no "key=" is not an argument and the
  tool will reject the call.
- For every edit-style tool call, the task description must call
  out a "lines_changed" estimate in the acceptance. The verifier
  uses this to enforce a 200-line diff-scope limit by default.
- The plan must respect the tool map: only use tools listed.
- The plan must respect the budget: every "cost_usd" is a ceiling
  in USD; "wall_clock_sec" is a ceiling in seconds.

On a repair call, the previous attempt and the verifier's errors
are included. Fix every error before re-emitting. Do not re-emit
the same plan.`

// planUserTmpl is the first-pass user prompt.
var planUserTmpl = template.Must(template.New("plan_user").Parse(planUserText))

const planUserText = `Goal: {{.Goal}}

{{- if .Platform}}

Platform: {{.Platform}}
The plan runs on this machine. Any shell command must exist on
this platform. {{if eq .Platform "windows"}}On Windows use
"dir" rather than "ls", "type" rather than "cat", and
"date /t" rather than "date". PowerShell built-ins
(Get-ChildItem, Get-Content) are available as
"powershell -Command".{{else}}Unix commands like "ls", "cat",
and "date" are available as written.{{end}}
{{- end}}

{{- if .Context}}

Context:
{{range $k, $v := .Context}}
- {{$k}}: {{$v}}
{{- end}}
{{- end}}

{{- if .Tools}}

Registered tools (use only these):
{{range .Tools}}
- {{.Name}} (risk_tier={{.RiskTier}}): {{.Description}}
  Input schema: {{.InputSchema}}
{{- end}}
{{- end}}

Emit the TaskPlan JSON. No other text.`

// planRepairTmpl is the prompt used on repair iterations. It
// re-states the goal, the previous attempt, and the verifier's
// errors. The model is asked to fix every error.
var planRepairTmpl = template.Must(template.New("plan_repair").Parse(planRepairText))

const planRepairText = `The previous plan failed verification. Produce a fixed plan.

Goal: {{.Goal}}

{{- if .Platform}}

Platform: {{.Platform}}
The plan runs on this machine. Any shell command must exist on
this platform. {{if eq .Platform "windows"}}On Windows use
"dir" rather than "ls", "type" rather than "cat", and
"date /t" rather than "date". PowerShell built-ins
(Get-ChildItem, Get-Content) are available as
"powershell -Command".{{else}}Unix commands like "ls", "cat",
and "date" are available as written.{{end}}
{{- end}}

{{- if .Context}}

Context:
{{range $k, $v := .Context}}
- {{$k}}: {{$v}}
{{- end}}
{{- end}}
{{- if .Tools}}

Registered tools (use only these):
{{range .Tools}}
- {{.Name}} (risk_tier={{.RiskTier}}): {{.Description}}
  Input schema: {{.InputSchema}}
{{- end}}
{{- end}}

Previous plan (invalid):

{{.Previous}}

Verifier errors:
{{- range .Errors}}
- {{.Code}}{{if .Field}} ({{.Field}}){{end}}: {{.Message}}
{{- end}}

Emit the corrected TaskPlan JSON. No other text.`

// RenderSystem returns the system prompt text. The output is
// deterministic for a given package version.
func RenderSystem() string {
	var buf bytes.Buffer
	if err := planSystemTmpl.Execute(&buf, nil); err != nil {
		// The template has no dynamic fields, so the only way this
		// can fail is a programmer error. Panic.
		panic(fmt.Sprintf("planner: render system: %v", err))
	}
	return buf.String()
}

// RenderUser returns the first-pass user prompt.
func RenderUser(d PromptData) (string, error) {
	// Sort the tools so the rendered prompt is deterministic.
	d.Tools = sortedTools(d.Tools)
	return renderTemplate(planUserTmpl, d)
}

// RenderRepair returns the repair prompt.
func RenderRepair(d PromptData) (string, error) {
	d.Tools = sortedTools(d.Tools)
	return renderTemplate(planRepairTmpl, d)
}

func renderTemplate(t *template.Template, d PromptData) (string, error) {
	var buf bytes.Buffer
	if err := t.Execute(&buf, d); err != nil {
		return "", fmt.Errorf("planner: render template %s: %w", t.Name(), err)
	}
	return buf.String(), nil
}

func sortedTools(in []ToolSpec) []ToolSpec {
	if len(in) <= 1 {
		return in
	}
	out := make([]ToolSpec, len(in))
	copy(out, in)
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}
