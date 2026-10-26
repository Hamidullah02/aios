package tools

// Tool is the interface every tool implements
// (REQ-TOL-IFC-001).
//
// Name returns the tool's name (the same name the bus uses
// to dispatch). Run executes the tool with the given args
// and returns a typed result. The bus wraps the call with
// policy and audit, so the tool itself only needs to do
// the actual work.
type Tool interface {
	// Name is the name the bus uses to dispatch. The name
	// is also the audit row's `tool` field.
	Name() string
	// Run executes the tool. The args are the call's
	// arguments; the context carries the actor, the task
	// id, and the workspace root.
	Run(args map[string]any, ctx Context) (Result, error)
}

// Describer is an OPTIONAL interface a tool may implement to
// describe itself to the planner.
//
// The planner has to tell the model what a tool does and what
// arguments it takes. Until this interface existed, that
// information lived in a hand-written table in the CLI
// package, and the two drifted: the bus registered twenty-five
// tools and the planner knew five. A tool the planner did not
// know about could not be named in a plan, so it was dead
// code however well it worked.
//
// A tool that implements Describer tells the planner about
// itself, which means the description cannot drift from the
// implementation: the two live in the same file.
//
// Describer is optional rather than required because Tool is a
// public interface with implementors outside this repository,
// and breaking it would break them. A tool that does not
// implement Describer is still dispatchable; it just is not
// described to the planner.
type Describer interface {
	// Describe returns the tool's description and its JSON
	// input schema.
	//
	// The schema is the raw JSON Schema object the model is
	// told to honour: a JSON string, not a Go value. It is
	// raw because the planner hands it straight to the model
	// and a model reads JSON Schema, not Go types.
	Describe() (description string, inputSchema string)
}

// RiskTier is an OPTIONAL interface a tool may implement to
// declare how dangerous it is.
//
// The tier drives two things: the policy gateway's default
// treatment of a call, and the planner's prompt. A tool that
// does not declare a tier is treated as tier 1, which is the
// bus's own default.
type RiskTier interface {
	// Risk returns the tool's risk tier, 0 (read-only) to 3.
	Risk() int
}

// Describe returns a tool's description and input schema, or
// a generic placeholder when the tool does not implement Spec.
//
// The placeholder is not silent: it says the tool exists but
// its arguments are unknown. A planner that offers such a tool
// will produce a task whose inputs do not match, and the run
// fails at the tool call with a message that does not explain
// why.
//
// The generic description is deliberately vague. A better
// guess would be a lie the model would plan against.
func Describe(t Tool) (description string, inputSchema string) {
	if t == nil {
		return "", ""
	}
	if s, ok := t.(Describer); ok {
		return s.Describe()
	}
	return "A tool named " + t.Name() + ". Its arguments are not documented, so it cannot be planned reliably.", "{}"
}

// Tier returns a tool's risk tier, defaulting to 1.
//
// The default matches the bus's own riskFromArgs default, so a
// tool that does not declare a tier is treated exactly as the
// bus has always treated it.
func Tier(t Tool) int {
	if t == nil {
		return 1
	}
	if r, ok := t.(RiskTier); ok {
		return r.Risk()
	}
	return 1
}

// Context is what the bus hands to each tool. The workspace
// is the root for path validation; actor and task_id are
// recorded on the audit row.
type Context struct {
	Actor     string
	TaskID    string
	Workspace string
}

// Result is the typed return value of a tool. The Output
// is what the tool produced; the Summary is a short string
// for the audit row's `result` field. Duration is filled in
// by the bus, not by the tool.
type Result struct {
	OK      bool
	Output  map[string]any
	Summary string
}
