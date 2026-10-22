package cli

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// varsPattern matches a `$vars.` reference and its dotted
// path.
//
// The pattern is deliberately strict about the identifier
// characters: a plan input is a `key=value` string, so a
// bare `$` (shell syntax, a currency amount, a regex
// anchor) must not be mistaken for a reference. Only
// `$vars.` followed by at least one identifier starts a
// match.
//
// The path segments may themselves contain underscores
// and digits after the first character, which covers task
// ids like `t1` and hashes.
var varsPattern = regexp.MustCompile(`\$vars\.([A-Za-z_][A-Za-z0-9_]*(?:\.[A-Za-z_][A-Za-z0-9_]*)*)`)

// interpolateVars replaces every `$vars.<path>` reference
// in s with the value found at that path in vars.
//
// Behaviour:
//
//   - A reference that resolves is replaced with the
//     string form of the leaf value.
//   - A reference that does NOT resolve (missing key, a
//     non-map segment mid-path, a nil value) is left
//     verbatim.
//
// Leaving unresolved references verbatim is the
// deliberate choice. Erroring would abort the whole plan
// on a typo in one input; substituting an empty string
// would silently pass "" to a tool, which is worse. Left
// verbatim, the tool receives `$vars.foo` and reports an
// unknown-value problem against that argument, which
// points at the actual mistake.
//
// The caller must hold vars before calling. A nil map is
// fine and means "resolve nothing".
func interpolateVars(s string, vars map[string]any) string {
	if len(vars) == 0 || !strings.Contains(s, "$vars.") {
		return s
	}
	return varsPattern.ReplaceAllStringFunc(s, func(ref string) string {
		m := varsPattern.FindStringSubmatch(ref)
		if len(m) != 2 {
			return ref
		}
		v, ok := lookupVarPath(vars, strings.Split(m[1], "."))
		// A nil leaf is treated as unresolved, not as an
		// empty string. lookupVarPath reports (nil, true)
		// when the key exists but holds nil, which is
		// indistinguishable from "the tool wrote a null
		// field" and tells a later task nothing. Substituting
		// "" there would hand the tool a plausible-looking
		// empty argument that silently does the wrong thing.
		// Leaving it verbatim keeps the failure visible.
		if !ok || v == nil {
			return ref
		}
		return formatVarValue(v)
	})
}

// lookupVarPath walks a dotted path through the vars map.
// The function reports false when any segment is missing
// or is not a map.
func lookupVarPath(vars map[string]any, path []string) (any, bool) {
	if len(path) == 0 {
		return nil, false
	}
	cur, ok := vars[path[0]]
	if !ok {
		return nil, false
	}
	for _, seg := range path[1:] {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil, false
		}
		cur, ok = m[seg]
		if !ok {
			return nil, false
		}
	}
	return cur, true
}

// formatVarValue renders a leaf value as the string a tool
// will receive.
//
// Numbers and bools use their canonical forms (no
// exponent for ints, "true"/"false" for bools). Anything
// else falls back to fmt.Sprint, which is what a
// structured value (a slice, a nested map) degrades to.
//
// A nil value is filtered out by the caller, so it never
// reaches this function; the case is kept only so the
// switch is total for a future caller that forgets the
// check.
func formatVarValue(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case bool:
		return strconv.FormatBool(t)
	case int:
		return strconv.Itoa(t)
	case int32:
		return strconv.FormatInt(int64(t), 10)
	case int64:
		return strconv.FormatInt(t, 10)
	case uint:
		return strconv.FormatUint(uint64(t), 10)
	case uint64:
		return strconv.FormatUint(t, 10)
	case float32:
		return strconv.FormatFloat(float64(t), 'g', -1, 32)
	case float64:
		return strconv.FormatFloat(t, 'g', -1, 64)
	default:
		return fmt.Sprint(t)
	}
}

// The task-id keying and the "last" alias are written by
// (*runtime.TaskContext).recordTaskVars in
// internal/runtime/vars.go. That lives in runtime (not cli)
// because the orchestrator is what publishes a completed
// task's output, and runtime must not import cli.
