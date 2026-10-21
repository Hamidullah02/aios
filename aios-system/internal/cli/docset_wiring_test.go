package cli

import (
	"path/filepath"
	"sort"
	"testing"

	"github.com/sadi/aios-system/internal/core/audit"
	"github.com/sadi/aios-system/internal/policy"
	"github.com/sadi/aios-system/internal/tools"
)

// TestWireDocSetTools_ReturnsEveryTool is the v0.38.0
// gate: every tool the doc-set packages already
// implement must appear in the returned map under its
// canonical name.
//
// Before v0.38.0 none of these were registered with the
// bus, so a plan naming `tool: edit` got
// ToolNotFoundError even though edit.EditTool was
// compiled into the same binary. The bug was invisible
// because nothing could ask the bus what it had.
func TestWireDocSetTools_ReturnsEveryTool(t *testing.T) {
	got := wireDocSetTools()
	want := []string{
		// edit package
		"edit", "multi_edit", "apply_patch", "write", "notebook_edit",
		// git package
		"git_status", "git_diff", "git_log", "git_commit", "git_branch", "git_worktree",
		// search package
		"grep", "glob", "code_search", "diagnostics",
		// image package
		"read_image",
	}
	for _, name := range want {
		tool, ok := got[name]
		if !ok {
			t.Errorf("wireDocSetTools: missing %q", name)
			continue
		}
		if tool == nil {
			t.Errorf("wireDocSetTools: %q maps to a nil tool", name)
			continue
		}
		// The map key and the tool's own Name() must
		// agree, otherwise a plan that dispatches by
		// name would reach a tool that reports a
		// different identity in the audit row.
		if reported := tool.Name(); reported != name {
			t.Errorf("wireDocSetTools: key %q but tool.Name() = %q", name, reported)
		}
	}
	if len(got) != len(want) {
		names := make([]string, 0, len(got))
		for n := range got {
			names = append(names, n)
		}
		sort.Strings(names)
		t.Errorf("wireDocSetTools: got %d tools %v, want %d", len(got), names, len(want))
	}
}

// TestWireDocSetTools_NoNilTools guards against a
// constructor that returns a typed-nil interface. A nil
// tool in the map passes `got[name] == nil` only for an
// untyped nil; a typed nil pointer would dispatch and
// panic inside the tool's method.
func TestWireDocSetTools_NoNilTools(t *testing.T) {
	for name, tool := range wireDocSetTools() {
		if tool == nil {
			t.Fatalf("%q is a nil tool", name)
		}
	}
}

// TestBusDispatch_FindsWiredTools proves the wiring
// reaches the bus, not just the helper map: a bus built
// with wireDocSetTools must resolve every name instead of
// returning ToolNotFoundError.
//
// The calls themselves are expected to fail (the tools
// need a real workspace / real arguments); what matters
// is that Dispatch finds the tool and gets past the
// lookup, so the error must NOT be ToolNotFoundError.
func TestBusDispatch_FindsWiredTools(t *testing.T) {
	dir := t.TempDir()
	log, err := audit.Open(audit.Config{Path: filepath.Join(dir, "aios.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()
	// Allow-everything policy so a missing tool is the
	// only thing under test. The gateway is default-deny
	// (REQ-POL-FUN-004), so without this rule every
	// dispatch would stop at the deny branch and never
	// reach the tool. `Match.Tool` is exact equality;
	// the wildcard lives in ToolGlob.
	gw := policy.New([]policy.Rule{
		{ID: "allow-all", Match: policy.Match{ToolGlob: "*"}, Action: policy.ActionAllow},
	}, log)
	bus := tools.New(tools.Config{
		Tools:  wireDocSetTools(),
		Policy: gw,
		Audit:  log,
	})
	names := bus.Names()
	if len(names) != 16 {
		t.Errorf("bus has %d tools %v, want 16", len(names), names)
	}
	// Every name must resolve. Missing args are fine;
	// a tool-not-found is not.
	for _, name := range names {
		_, derr := bus.Dispatch(name, map[string]any{}, tools.Context{
			Actor:     "t",
			TaskID:    "wiring",
			Workspace: dir,
		})
		if derr == nil {
			continue
		}
		var tnf *tools.ToolNotFoundError
		if asToolNotFound(derr, &tnf) {
			t.Errorf("Dispatch(%q): tool not found", name)
		}
	}
}

// asToolNotFound is a tiny errors.As wrapper so the test
// file does not need the errors import for one call.
func asToolNotFound(err error, target **tools.ToolNotFoundError) bool {
	for err != nil {
		if e, ok := err.(*tools.ToolNotFoundError); ok {
			*target = e
			return true
		}
		u, ok := err.(interface{ Unwrap() error })
		if !ok {
			return false
		}
		err = u.Unwrap()
	}
	return false
}
