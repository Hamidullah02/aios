package cli

import (
	"bytes"
	"sort"
	"strings"
	"testing"

	"github.com/sadi/aios-system/internal/tools"
)

// TestBusNames_SortedAndComplete covers the new
// tools.Bus.Names method (v0.38.0). The method exists so
// a wiring regression is testable: before it, nothing
// could ask the bus what it had, which is why the
// never-registered edit / git / search / image tools
// went unnoticed.
func TestBusNames_SortedAndComplete(t *testing.T) {
	bus := tools.New(tools.Config{
		Tools: map[string]tools.Tool{
			"zebra":   tools.ReadFileTool{},
			"alpha":   tools.WriteFileTool{},
			"middle":  tools.SearchFilesTool{},
			"another": tools.ReadFileTool{},
		},
	})
	got := bus.Names()
	if !sort.StringsAreSorted(got) {
		t.Errorf("Names() not sorted: %v", got)
	}
	want := []string{"alpha", "another", "middle", "zebra"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("Names() = %v, want %v", got, want)
	}
}

// TestBusNames_EmptyBus asserts an empty (or
// unregistered) bus returns an empty slice, not nil,
// so callers can range over it unconditionally.
func TestBusNames_EmptyBus(t *testing.T) {
	bus := tools.New(tools.Config{})
	got := bus.Names()
	if got == nil {
		t.Fatal("Names() = nil, want an empty slice")
	}
	if len(got) != 0 {
		t.Errorf("Names() = %v, want empty", got)
	}
}

// TestBusNames_MutatingResultIsSafe asserts Names
// returns a fresh slice: a caller sorting or truncating
// it must not corrupt the bus.
func TestBusNames_MutatingResultIsSafe(t *testing.T) {
	bus := tools.New(tools.Config{
		Tools: map[string]tools.Tool{
			"alpha": tools.ReadFileTool{},
			"beta":  tools.WriteFileTool{},
		},
	})
	first := bus.Names()
	first[0] = "MUTATED"
	second := bus.Names()
	for _, n := range second {
		if n == "MUTATED" {
			t.Fatal("Names() returned an aliased slice; caller mutation leaked into the bus")
		}
	}
}

// TestRunTools_ListsEveryWiredTool is the v0.38.0
// operator-facing gate: `aios tools` must name every tool
// a plan can dispatch.
//
// The exact count is asserted so a tool silently dropped
// from the wiring is a test failure rather than a
// surprise at run time.
func TestRunTools_ListsEveryWiredTool(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runTools(&Args{Command: CmdTools}, &Deps{}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("exit %d, stderr: %s", code, stderr.String())
	}
	var names []string
	for _, line := range strings.Split(strings.TrimSpace(stdout.String()), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			names = append(names, line)
		}
	}
	if !sort.StringsAreSorted(names) {
		t.Errorf("aios tools output not sorted:\n%s", stdout.String())
	}
	// The doc-set set: the REQ-DOC-MIN five, the verify
	// tools, and the 16 wired in v0.38.0.
	want := []string{
		"read_file", "search_files", "write_file", "run_shell", "ask_user",
		"run_tests",
		"edit", "multi_edit", "apply_patch", "write", "notebook_edit",
		"git_status", "git_diff", "git_log", "git_commit", "git_branch", "git_worktree",
		"grep", "glob", "code_search", "diagnostics",
		"read_image",
		"memory", "memory_write",
	}
	got := strings.Join(names, ",")
	for _, name := range want {
		if !containsName(names, name) {
			t.Errorf("aios tools missing %q (got: %s)", name, got)
		}
	}
	if len(names) != len(want) {
		t.Errorf("aios tools listed %d tools, want %d:\n%s", len(names), len(want), got)
	}
}

// TestParse_Tools covers the parser wiring for the new
// subcommand.
func TestParse_Tools(t *testing.T) {
	a, err := Parse([]string{"aios", "tools"}, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	if a.Command != CmdTools {
		t.Errorf("command = %q, want %q", a.Command, CmdTools)
	}
}

func containsName(list []string, want string) bool {
	for _, n := range list {
		if n == want {
			return true
		}
	}
	return false
}
