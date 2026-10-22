package runtime

import "testing"

// TestRecordTaskVars_NamesAndAlias covers the two keys the
// method writes: the task id and the "last" alias.
func TestRecordTaskVars_NamesAndAlias(t *testing.T) {
	ctx := &TaskContext{Vars: map[string]any{}}
	if !ctx.recordTaskVars("t1", map[string]any{"hash": "abc", "count": 2}) {
		t.Fatal("recordTaskVars returned false for a valid input")
	}
	if ctx.Vars["t1"] == nil {
		t.Error("Vars[t1] not set")
	}
	if ctx.Vars["last"] == nil {
		t.Error("Vars[last] not set")
	}
}

// TestRecordTaskVars_LastMovesOn is the documented sharp
// edge: "last" tracks the most recent task, so a second
// task overwrites it. A plan needing a specific task's
// output must name the task.
func TestRecordTaskVars_LastMovesOn(t *testing.T) {
	ctx := &TaskContext{Vars: map[string]any{}}
	ctx.recordTaskVars("t1", map[string]any{"hash": "first"})
	ctx.recordTaskVars("t2", map[string]any{"hash": "second"})

	first, _ := ctx.Vars["t1"].(map[string]any)
	if first["hash"] != "first" {
		t.Errorf("t1 hash = %v, want first (named tasks must not move)", first["hash"])
	}
	last, _ := ctx.Vars["last"].(map[string]any)
	if last["hash"] != "second" {
		t.Errorf("last hash = %v, want second", last["hash"])
	}
}

// TestRecordTaskVars_SkipsEmpty covers the no-op cases: no
// task id, or no structured output. Writing an empty map
// would make a reference resolve to "" instead of failing to
// resolve, which reads as a deliberate empty value.
func TestRecordTaskVars_SkipsEmpty(t *testing.T) {
	ctx := &TaskContext{Vars: map[string]any{}}
	if ctx.recordTaskVars("", map[string]any{"a": 1}) {
		t.Error("expected false for an empty task id")
	}
	if ctx.recordTaskVars("t1", nil) {
		t.Error("expected false for a nil Data map")
	}
	if ctx.recordTaskVars("t1", map[string]any{}) {
		t.Error("expected false for an empty Data map")
	}
	if len(ctx.Vars) != 0 {
		t.Errorf("Vars = %v, want untouched", ctx.Vars)
	}
}

// TestRecordTaskVars_NilContextIsSafe asserts the nil
// receiver guard holds.
func TestRecordTaskVars_NilContextIsSafe(t *testing.T) {
	var ctx *TaskContext
	if ctx.recordTaskVars("t1", map[string]any{"a": 1}) {
		t.Error("expected false for a nil context")
	}
}

// TestRecordTaskVars_AllocatesBag asserts a context built
// without a Vars map (as a fresh TaskContext literal is)
// gets one on first write rather than panicking.
func TestRecordTaskVars_AllocatesBag(t *testing.T) {
	ctx := &TaskContext{}
	if ctx.recordTaskVars("t1", map[string]any{"a": 1}) != true {
		t.Fatal("expected true")
	}
	if ctx.Vars == nil {
		t.Fatal("Vars was not allocated")
	}
	if ctx.Vars["t1"] == nil {
		t.Error("Vars[t1] not set after allocation")
	}
}
