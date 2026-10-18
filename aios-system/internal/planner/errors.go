// Package planner is the planner for aios.
//
// The package turns a goal into a TaskPlan. The flow is:
//
//  1. The caller calls Planner.Plan(goal, ctx).
//  2. The planner renders the system + user prompts and asks the
//     model router to complete.
//  3. The planner parses the response as JSON and unmarshals into
//     a schemas.TaskPlan.
//  4. The planner runs the CodeFirstVerifier; on pass it returns
//     the plan.
//  5. On fail, the planner runs the repair loop: it asks the model
//     again with the errors, then re-runs the verifier. After
//     max repair iterations it raises PlanGenerationError.
//
// Substitutions vs the doc set:
//
//   - Python filenames (.py) and jinja2 templates (.txt) become
//     Go files (.go) and `text/template` (stdlib, REQ-PLN-NFR-002
//     the doc set's "depends only on aios-core + stdlib").
//   - The doc set's `model_router: ModelRouter` is the aios-models
//     router (internal/models/router.go). The planner calls
//     Router.Complete and reads the response text + usage.
//   - The doc set's diff-scope check (§6.2.3, §9.2) is computed
//     from the tool calls in the plan. The plan JSON does not
//     carry a per-task line-count today; the planner infers the
//     count from a `lines_changed` argument on the tool call, and
//     ignores tasks that do not declare it. This is the §9.2 spec
//     without inventing a schema field that the rest of the system
//     does not own.
package planner

import (
	"encoding/json"
	"fmt"
)

// PlanError is a single issue the verifier found on a TaskPlan.
//
// Per REQ-PLN-DAT-002 the error carries the code, the field path,
// and the message. Codes are stable strings the caller can switch
// on; the field path uses dotted notation (e.g. "tasks.2.tool");
// the message is a short human-readable explanation.
type PlanError struct {
	Code    string `json:"code"`
	Field   string `json:"field,omitempty"`
	Message string `json:"message"`
}

// Error renders the error as a single line. The format is stable
// for log/audit consumption.
func (e *PlanError) Error() string {
	if e == nil {
		return "<nil plan error>"
	}
	if e.Field == "" {
		return fmt.Sprintf("%s: %s", e.Code, e.Message)
	}
	return fmt.Sprintf("%s (%s): %s", e.Code, e.Field, e.Message)
}

// PlanErrorList is the verifier's return type. An empty list is a
// pass; a non-empty list is a fail. The slice is the immutable
// shape — callers may range over it but should not mutate it.
type PlanErrorList []PlanError

// Codes is the catalog of error codes the verifier and the planner
// can produce. The set is intentionally small so callers can match
// deterministically. New codes are additive; old codes do not get
// renamed.
const (
	CodeSchema           = "schema"             // plan failed schemas.TaskPlan.Validate
	CodeToolCallSchema   = "tool_call_schema"   // a tool call inside the plan is malformed
	CodeUnknownTool      = "unknown_tool"       // tool_call tool is not in the registered tool map
	CodeDiffScope        = "diff_scope"         // declared lines_changed exceeds the limit
	CodeMalformedJSON    = "malformed_json"     // planner could not parse the model response
	CodeModelCall        = "model_call_failed"  // router/Complete returned an error
	CodeRepairExhausted  = "repair_exhausted"   // model failed to repair after max iterations
	CodeCustomRuleFailed = "custom_rule_failed" // a custom rule returned false
)

// PlanValidationError is raised when the verifier rejects a plan.
//
// Per REQ-PLN-ERR-002 the error wraps the list of issues so the
// caller can inspect each PlanError with errors.As / errors.Is
// matching on the type.
type PlanValidationError struct {
	Errors PlanErrorList
}

func (e *PlanValidationError) Error() string {
	if len(e.Errors) == 0 {
		return "plan rejected (no errors)"
	}
	parts := make([]string, len(e.Errors))
	for i, pe := range e.Errors {
		parts[i] = pe.Error()
	}
	return "plan rejected: " + joinStrings(parts, "; ")
}

// Unwrap is not provided because PlanValidationError is a wrapper
// for several errors at once. Callers iterate the Errors slice.

func joinStrings(parts []string, sep string) string {
	if len(parts) == 0 {
		return ""
	}
	if len(parts) == 1 {
		return parts[0]
	}
	out := parts[0]
	for i := 1; i < len(parts); i++ {
		out += sep + parts[i]
	}
	return out
}

// PlanGenerationError is raised when the planner cannot produce a
// valid plan within the maximum repair iteration count, or when an
// earlier round produced no JSON at all.
//
// Per REQ-PLN-ERR-001 the error includes the iteration count and
// the last-known verifier verdict so the operator can see whether
// the planner ran out of iterations on a schema error or on a
// model call error.
type PlanGenerationError struct {
	Iterations int
	LastErrors PlanErrorList
	Cause      error // may be nil; populated when the failure is from the model itself
}

func (e *PlanGenerationError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("plan generation failed after %d iteration(s): %v", e.Iterations, e.Cause)
	}
	if len(e.LastErrors) > 0 {
		return fmt.Sprintf("plan generation failed after %d iteration(s); last verifier report: %s",
			e.Iterations, formatErrors(e.LastErrors))
	}
	return fmt.Sprintf("plan generation failed after %d iteration(s)", e.Iterations)
}

// Unwrap exposes the underlying cause so errors.Is matches.
func (e *PlanGenerationError) Unwrap() error { return e.Cause }

func formatErrors(errs PlanErrorList) string {
	if len(errs) == 0 {
		return "<none>"
	}
	parts := make([]string, len(errs))
	for i, pe := range errs {
		parts[i] = pe.Error()
	}
	return joinStrings(parts, "; ")
}

// marshalErrors is a tiny helper for the verifier: it marshals a
// PlanErrorList to canonical JSON for inclusion in the audit row
// and the repair prompt.
func marshalErrors(errs PlanErrorList) ([]byte, error) {
	if errs == nil {
		return []byte("[]"), nil
	}
	return json.Marshal(errs)
}
