package cli

import (
	"testing"

	"github.com/sadi/aios-system/internal/runtime"
	"github.com/sadi/aios-system/internal/tools"
)

// TestInterpolate_ReplacesString is the base case: a
// top-level string reference resolves.
func TestInterpolate_ReplacesString(t *testing.T) {
	vars := map[string]any{"dir": "/srv/app"}
	if got := interpolateVars("cwd=$vars.dir", vars); got != "cwd=/srv/app" {
		t.Errorf("got %q, want cwd=/srv/app", got)
	}
}

// TestInterpolate_ReplacesNestedPath is the case that
// makes multi-step plans possible: a tool's Output map is
// a map[string]any, so a task reaches into it with a dot.
func TestInterpolate_ReplacesNestedPath(t *testing.T) {
	vars := map[string]any{
		"t1": map[string]any{
			"hash": "deadbeef",
			"kind": "decision",
		},
	}
	if got := interpolateVars("hash=$vars.t1.hash", vars); got != "hash=deadbeef" {
		t.Errorf("got %q, want hash=deadbeef", got)
	}
	if got := interpolateVars("k=$vars.t1.kind", vars); got != "k=decision" {
		t.Errorf("got %q, want k=decision", got)
	}
}

// TestInterpolate_LeavesUnresolvedVerbatim is the
// deliberate design decision: an unknown reference is NOT
// an error and NOT an empty string. The tool receives the
// literal and reports the unknown argument, which points
// at the actual mistake.
func TestInterpolate_LeavesUnresolvedVerbatim(t *testing.T) {
	vars := map[string]any{"t1": map[string]any{"hash": "x"}}
	cases := []string{
		"hash=$vars.missing",
		"hash=$vars.t1.nosuchkey",
		"hash=$vars.t1.hash.deeper",
		// $vars.dir is a string, so .subseg cannot walk
		// into it.
		"hash=$vars.dir.sub",
	}
	vars["dir"] = "/srv"
	for _, in := range cases {
		if got := interpolateVars(in, vars); got != in {
			t.Errorf("interpolateVars(%q) = %q, want unchanged", in, got)
		}
	}
}

// TestInterpolate_HandlesNumberAndBool covers the numeric
// and boolean leaf types, which arrive in a tool Output
// map as JSON-decoded float64 or native Go values.
func TestInterpolate_HandlesNumberAndBool(t *testing.T) {
	vars := map[string]any{
		"count": 42,
		"i64":   int64(9007199254740993),
		"ratio": 0.5,
		"ok":    true,
		"no":    false,
	}
	cases := map[string]string{
		"a=$vars.count": "a=42",
		"b=$vars.i64":   "b=9007199254740993",
		"c=$vars.ratio": "c=0.5",
		"d=$vars.ok":    "d=true",
		"e=$vars.no":    "e=false",
	}
	for in, want := range cases {
		if got := interpolateVars(in, vars); got != want {
			t.Errorf("interpolateVars(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestInterpolate_NoPlaceholderPassesThrough guards the
// fast path: a plan input with no `$vars.` is returned
// byte-for-byte.
func TestInterpolate_NoPlaceholderPassesThrough(t *testing.T) {
	vars := map[string]any{"x": "y"}
	for _, in := range []string{
		"",
		"plain text",
		"path=/usr/local/bin",
		// A bare $ is shell syntax, not a reference.
		"cmd=echo $HOME",
		"regex=^$",
		"money=$5",
	} {
		if got := interpolateVars(in, vars); got != in {
			t.Errorf("interpolateVars(%q) = %q, want unchanged", in, got)
		}
	}
}

// TestInterpolate_MultiplePlaceholdersInOneString covers a
// composed argument.
func TestInterpolate_MultiplePlaceholdersInOneString(t *testing.T) {
	vars := map[string]any{"a": "1", "b": "2"}
	in := "before $vars.a middle $vars.b after"
	want := "before 1 middle 2 after"
	if got := interpolateVars(in, vars); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestInterpolate_RepeatedReference covers the same
// placeholder appearing twice.
func TestInterpolate_RepeatedReference(t *testing.T) {
	vars := map[string]any{"p": "x"}
	in := "$vars.p-$vars.p"
	if got := interpolateVars(in, vars); got != "x-x" {
		t.Errorf("got %q, want x-x", got)
	}
}

// TestInterpolate_NilVars asserts a nil bag is safe and
// resolves nothing.
func TestInterpolate_NilVars(t *testing.T) {
	in := "hash=$vars.t1.hash"
	if got := interpolateVars(in, nil); got != in {
		t.Errorf("got %q, want unchanged", got)
	}
}

// TestInterpolate_NilLeafIsLeftVerbatim asserts a key that
// exists with a nil value is treated as unresolved, not
// substituted with "".
//
// v0.40.3: this returned "", which contradicted both this
// test's own name and interpolateVars' documented contract
// that an unresolvable reference is left verbatim. A nil
// leaf is what a tool emits for a null field, and it tells
// a later task nothing. Handing the tool "" instead would
// look like a deliberate empty argument and silently do the
// wrong thing, so it is left verbatim like any other
// unresolved reference.
func TestInterpolate_NilLeafIsLeftVerbatim(t *testing.T) {
	vars := map[string]any{"nothing": nil}
	in := "x=$vars.nothing"
	if got := interpolateVars(in, vars); got != in {
		t.Errorf("got %q, want %q left verbatim", got, in)
	}
}

// TestInterpolate_EmptyStringLeafDoesResolve is the paired
// case that keeps the fix honest: a key present with an
// EMPTY STRING is a real value and must substitute, or the
// nil fix would be too broad and a tool could never receive
// "" on purpose.
func TestInterpolate_EmptyStringLeafDoesResolve(t *testing.T) {
	vars := map[string]any{"empty": ""}
	if got := interpolateVars("x=$vars.empty", vars); got != "x=" {
		t.Errorf("got %q, want %q (an explicit empty string is a value)", got, "x=")
	}
}

// TestInterpolate_NestedNilLeafIsLeftVerbatim covers the
// same case one level down, where the nil is inside a task's
// output map.
func TestInterpolate_NestedNilLeafIsLeftVerbatim(t *testing.T) {
	vars := map[string]any{
		"t1": map[string]any{"hash": nil, "ok": "yes"},
	}
	in := "hash=$vars.t1.hash"
	if got := interpolateVars(in, vars); got != in {
		t.Errorf("got %q, want %q left verbatim", got, in)
	}
	// The sibling key must still resolve, so the fix does
	// not disable the whole task's map.
	if got := interpolateVars("ok=$vars.t1.ok", vars); got != "ok=yes" {
		t.Errorf("got %q, want ok=yes", got)
	}
}

// --- worker integration ---

// echoVarsTool records the args it was called with so the
// worker test can assert what the bus actually saw.
type echoVarsTool struct {
	got map[string]any
}

func (e *echoVarsTool) Name() string { return "echo_vars" }

func (e *echoVarsTool) Run(args map[string]any, _ tools.Context) (tools.Result, error) {
	e.got = args
	return tools.Result{OK: true, Output: map[string]any{"seen": "yes"}, Summary: "ok"}, nil
}

// TestBusWorker_ResolvesVarsInInputs is the end-to-end
// gate: a task input referencing an earlier task's output
// reaches the tool with the resolved value.
func TestBusWorker_ResolvesVarsInInputs(t *testing.T) {
	tool := &echoVarsTool{}
	bus := tools.New(tools.Config{Tools: map[string]tools.Tool{tool.Name(): tool}})
	worker := NewBusWorker(bus, nil, t.TempDir())

	taskCtx := &runtime.TaskContext{Vars: map[string]any{
		"t1": map[string]any{"hash": "deadbeef", "count": 7},
	}}
	res, err := worker.Run(runtime.TaskInput{
		ID:     "t2",
		Tool:   tool.Name(),
		Inputs: []string{"hash=$vars.t1.hash", "n=$vars.t1.count"},
	}, taskCtx, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Output != "ok" {
		t.Errorf("Output = %q, want ok", res.Output)
	}
	if got := tool.got["hash"]; got != "deadbeef" {
		t.Errorf("args[hash] = %v, want deadbeef", got)
	}
	if got := tool.got["n"]; got != "7" {
		t.Errorf("args[n] = %v, want 7", got)
	}
}

// TestBusWorker_UnresolvedReferencePassesThrough asserts
// the literal reaches the tool rather than failing the
// task.
func TestBusWorker_UnresolvedReferencePassesThrough(t *testing.T) {
	tool := &echoVarsTool{}
	bus := tools.New(tools.Config{Tools: map[string]tools.Tool{tool.Name(): tool}})
	worker := NewBusWorker(bus, nil, t.TempDir())

	_, err := worker.Run(runtime.TaskInput{
		ID:     "t2",
		Tool:   tool.Name(),
		Inputs: []string{"hash=$vars.nope.hash"},
	}, &runtime.TaskContext{Vars: map[string]any{}}, nil)
	if err != nil {
		t.Fatalf("Run: %v (an unresolved reference must not fail the task)", err)
	}
	if got := tool.got["hash"]; got != "$vars.nope.hash" {
		t.Errorf("args[hash] = %v, want the literal $vars.nope.hash", got)
	}
}

// TestBusWorker_NilTaskContextIsSafe asserts the worker
// still dispatches when a caller passes no context.
func TestBusWorker_NilTaskContextIsSafe(t *testing.T) {
	tool := &echoVarsTool{}
	bus := tools.New(tools.Config{Tools: map[string]tools.Tool{tool.Name(): tool}})
	worker := NewBusWorker(bus, nil, t.TempDir())

	_, err := worker.Run(runtime.TaskInput{
		ID:     "t1",
		Tool:   tool.Name(),
		Inputs: []string{"a=1"},
	}, nil, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if tool.got["a"] != "1" {
		t.Errorf("args[a] = %v, want 1", tool.got["a"])
	}
}

// TestBusWorker_ResolvedValueWithCommaSplits covers the
// interaction between interpolation and the comma-splitting
// the worker already did for run_shell: a resolved value
// containing a comma becomes a []string.
func TestBusWorker_ResolvedValueWithCommaSplits(t *testing.T) {
	tool := &echoVarsTool{}
	bus := tools.New(tools.Config{Tools: map[string]tools.Tool{tool.Name(): tool}})
	worker := NewBusWorker(bus, nil, t.TempDir())

	_, err := worker.Run(runtime.TaskInput{
		ID:     "t2",
		Tool:   tool.Name(),
		Inputs: []string{"command=$vars.t1.cmd"},
	}, &runtime.TaskContext{Vars: map[string]any{
		"t1": map[string]any{"cmd": "go,test"},
	}}, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	list, ok := tool.got["command"].([]string)
	if !ok {
		t.Fatalf("args[command] is %T, want []string", tool.got["command"])
	}
	if len(list) != 2 || list[0] != "go" || list[1] != "test" {
		t.Errorf("args[command] = %v, want [go test]", list)
	}
}
