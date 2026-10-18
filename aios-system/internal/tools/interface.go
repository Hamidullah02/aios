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
