package planner

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/sadi/aios-system/internal/core/schemas"
)

// Verifier is the protocol the planner uses to check a plan. The
// only built-in implementation is CodeFirstVerifier, but tests
// can substitute a fake to drive the repair loop deterministically.
type Verifier interface {
	// Verify returns an empty list when the plan is acceptable, or
	// a non-empty list of PlanError describing every issue found.
	Verify(plan *schemas.TaskPlan) PlanErrorList
}

// CustomRule is the shape of an extra rule the verifier can run.
// A rule returns true to accept the plan, false to reject. A rule
// may also add one or more PlanError values to the output slice.
//
// Rules run in the order they were registered. The default
// CodeFirstVerifier has four built-in rules (schema, tool call,
// tool registry, diff scope) and then runs the custom rules in
// order.
//
// Per REQ-PLN-VRF-005 the verifier supports custom rules.
type CustomRule func(plan *schemas.TaskPlan, out PlanErrorList) PlanErrorList

// CodeFirstVerifier is the default verifier. It validates the
// plan against the TaskPlan schema, then every tool call's
// arguments, then the diff-scope estimate, then any custom
// rules the caller registered.
//
// Per REQ-PLN-VRF-001..005 the verifier is code-first. No model
// call. The model is reserved for the planner's repair step.
type CodeFirstVerifier struct {
	// ToolMap is the name -> ToolSpec registry. The verifier
	// rejects plans that name a tool not in the map.
	ToolMap map[string]ToolSpec

	// DiffScopeLimit is the maximum sum of declared "lines_changed"
	// values across all tasks in a plan. Plans that exceed it
	// are rejected with CodeDiffScope. The default (200) matches
	// the doc set's §6.2.3.
	DiffScopeLimit int

	// CustomRules is the optional chain of extra rules. They run
	// after the built-in rules.
	CustomRules []CustomRule
}

// NewCodeFirstVerifier returns a verifier with the given tool map
// and the default 200-line diff-scope limit.
func NewCodeFirstVerifier(tools map[string]ToolSpec) *CodeFirstVerifier {
	return &CodeFirstVerifier{
		ToolMap:        tools,
		DiffScopeLimit: 200,
	}
}

// WithDiffScopeLimit sets the limit and returns the receiver so
// callers can chain.
func (v *CodeFirstVerifier) WithDiffScopeLimit(n int) *CodeFirstVerifier {
	v.DiffScopeLimit = n
	return v
}

// WithRule appends a custom rule and returns the receiver.
func (v *CodeFirstVerifier) WithRule(r CustomRule) *CodeFirstVerifier {
	v.CustomRules = append(v.CustomRules, r)
	return v
}

// Verify runs the built-in checks then the custom rules. The
// returned slice is non-nil and stable; it is safe to range over.
func (v *CodeFirstVerifier) Verify(plan *schemas.TaskPlan) PlanErrorList {
	if plan == nil {
		return PlanErrorList{{Code: CodeSchema, Field: "", Message: "plan is nil"}}
	}
	var errs PlanErrorList
	errs = v.checkSchema(plan, errs)
	errs = v.checkToolCalls(plan, errs)
	errs = v.checkToolRegistry(plan, errs)
	errs = v.checkDiffScope(plan, errs)
	for _, r := range v.CustomRules {
		errs = r(plan, errs)
	}
	return errs
}

// checkSchema runs schemas.TaskPlan.Validate and converts any
// validation error into a single PlanError. The original error's
// path is preserved as the Field.
func (v *CodeFirstVerifier) checkSchema(plan *schemas.TaskPlan, errs PlanErrorList) PlanErrorList {
	if err := plan.Validate(); err != nil {
		// The schemas package returns a *ValidationError; if not
		// (some JoinValidationErrors results are plain errors),
		// the path is empty.
		if ve, ok := err.(*schemas.ValidationError); ok {
			return append(errs, PlanError{
				Code:    CodeSchema,
				Field:   ve.Path,
				Message: ve.Reason,
			})
		}
		return append(errs, PlanError{
			Code:    CodeSchema,
			Message: err.Error(),
		})
	}
	return errs
}

// checkToolCalls validates the plan's tool-call shape. The
// schemas.TaskPlan type does not embed explicit ToolCall records;
// the model is expected to declare tool calls in the task
// "description" or in the task "tool" + "inputs" fields. The
// verifier checks:
//
//   - the task "tool" name is a non-empty string when present;
//   - the task "inputs" is a list of strings.
//
// The doc set's REQ-PLN-VRF-003 says "check the schema of every
// tool call". The aios-core schema is at the task level (each
// task has a single tool), so that is what we check.
func (v *CodeFirstVerifier) checkToolCalls(plan *schemas.TaskPlan, errs PlanErrorList) PlanErrorList {
	for i, t := range plan.Tasks {
		base := fmt.Sprintf("tasks.%d", i)
		if t.Tool != "" {
			// The tool name must be non-empty. A space-only
			// name is treated as a schema error too.
			if looksBlank(t.Tool) {
				errs = append(errs, PlanError{
					Code:    CodeToolCallSchema,
					Field:   base + ".tool",
					Message: "tool name is blank",
				})
			}
		}
		for j, in := range t.Inputs {
			if looksBlank(in) {
				errs = append(errs, PlanError{
					Code:    CodeToolCallSchema,
					Field:   fmt.Sprintf("%s.inputs.%d", base, j),
					Message: "input is blank",
				})
			}
		}
	}
	return errs
}

// checkToolRegistry rejects plans that name a tool not in the
// verifier's ToolMap. An empty ToolMap means the verifier cannot
// make a registry decision; in that case the rule is permissive
// (the plan is accepted on this dimension).
func (v *CodeFirstVerifier) checkToolRegistry(plan *schemas.TaskPlan, errs PlanErrorList) PlanErrorList {
	if len(v.ToolMap) == 0 {
		return errs
	}
	for i, t := range plan.Tasks {
		if t.Tool == "" {
			continue
		}
		if _, ok := v.ToolMap[t.Tool]; !ok {
			errs = append(errs, PlanError{
				Code:    CodeUnknownTool,
				Field:   fmt.Sprintf("tasks.%d.tool", i),
				Message: fmt.Sprintf("tool %q is not in the registered tool map", t.Tool),
			})
		}
	}
	return errs
}

// checkDiffScope enforces the §9.2 algorithm: walk every task,
// look for a "lines_changed" hint in the acceptance string, parse
// it as an integer, and sum. If the sum exceeds DiffScopeLimit,
// reject with CodeDiffScope.
//
// The "lines_changed" hint is encoded as the token `lines=N` in
// the task's acceptance string, e.g.:
//
//	"tests pass and lines=12"
//
// This keeps the schema unchanged (the doc set does not introduce
// a new field) while still giving the verifier a number to sum.
func (v *CodeFirstVerifier) checkDiffScope(plan *schemas.TaskPlan, errs PlanErrorList) PlanErrorList {
	if v.DiffScopeLimit <= 0 {
		return errs // no limit set
	}
	total := 0
	for i, t := range plan.Tasks {
		// Reject `lines=-N` early; parseLinesHint only matches
		// unsigned integers, so without this check the negative
		// form would silently fall through as "no hint".
		if neg, ok := parseNegativeLinesHint(t.Acceptance); ok {
			errs = append(errs, PlanError{
				Code:    CodeDiffScope,
				Field:   fmt.Sprintf("tasks.%d.acceptance", i),
				Message: fmt.Sprintf("lines_changed hint is negative: %d", neg),
			})
			continue
		}
		n, ok := parseLinesHint(t.Acceptance)
		if !ok {
			continue
		}
		total += n
	}
	if total > v.DiffScopeLimit {
		errs = append(errs, PlanError{
			Code:    CodeDiffScope,
			Message: fmt.Sprintf("declared diff scope %d exceeds limit %d", total, v.DiffScopeLimit),
		})
	}
	return errs
}

// parseNegativeLinesHint looks for a `lines=-N` token. ok is true
// when the token is present and parses cleanly. parseLinesHint
// does not match negative numbers (digits only); this helper
// covers the negative case so the verifier can flag it.
func parseNegativeLinesHint(s string) (int, bool) {
	const key = "lines=-"
	idx := indexOf(s, key)
	if idx < 0 {
		return 0, false
	}
	rest := s[idx+len(key):]
	end := 0
	for end < len(rest) && rest[end] >= '0' && rest[end] <= '9' {
		end++
	}
	if end == 0 {
		return 0, false
	}
	n, err := strconv.Atoi(rest[:end])
	if err != nil {
		return 0, false
	}
	return -n, true
}

// parseLinesHint extracts the `lines=N` token from s. ok is true
// when the token is present and parses cleanly. The function
// tolerates other text around the token.
func parseLinesHint(s string) (int, bool) {
	// Look for "lines=" then a non-negative integer.
	const key = "lines="
	idx := indexOf(s, key)
	if idx < 0 {
		return 0, false
	}
	rest := s[idx+len(key):]
	// Take the longest run of digits.
	end := 0
	for end < len(rest) && rest[end] >= '0' && rest[end] <= '9' {
		end++
	}
	if end == 0 {
		return 0, false
	}
	n, err := strconv.Atoi(rest[:end])
	if err != nil {
		return 0, false
	}
	return n, true
}

func indexOf(s, sub string) int {
	if len(sub) == 0 {
		return 0
	}
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func looksBlank(s string) bool {
	for _, r := range s {
		if r != ' ' && r != '\t' && r != '\n' && r != '\r' {
			return false
		}
	}
	return true
}

// EncodePlan is a helper the planner uses to put the previous
// attempt into the repair prompt as a JSON string.
func EncodePlan(plan *schemas.TaskPlan) (string, error) {
	if plan == nil {
		return "", nil
	}
	raw, err := json.Marshal(plan)
	if err != nil {
		return "", fmt.Errorf("planner: encode plan: %w", err)
	}
	return string(raw), nil
}
