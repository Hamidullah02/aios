package cli

import (
	"path/filepath"
	"testing"

	"github.com/Hamidullah02/aios/aios-system/internal/core/audit"
	"github.com/Hamidullah02/aios/aios-system/internal/memory"
	"github.com/Hamidullah02/aios/aios-system/internal/tools"
)

// TestWireMemoryTools_SharesOneInstance is the v0.39.0
// wiring gate.
//
// The two memory tools must receive the SAME
// *memory.Memory. If they get separate instances (one
// per tool, or a fresh one per call), a write through
// memory_write is invisible to a read through memory and
// the memory silently appears broken: every write
// "succeeds" and every read comes back empty.
//
// The check is behavioural, not structural — it proves
// the round trip through the wired tools, which is what
// the user actually sees.
func TestWireMemoryTools_SharesOneInstance(t *testing.T) {
	dir := t.TempDir()
	mem, err := memory.NewMemory(memory.MemoryConfig{
		DataDir: filepath.Join(dir, "data"),
		Home:    filepath.Join(dir, "home"),
	})
	if err != nil {
		t.Fatal(err)
	}
	log, err := audit.Open(audit.Config{Path: ":memory:"})
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()

	wired := wireMemoryTools(mem, log)
	if len(wired) != 2 {
		t.Fatalf("wireMemoryTools returned %d tools, want 2", len(wired))
	}
	readTool, hasRead := wired["memory"]
	writeTool, hasWrite := wired["memory_write"]
	if !hasRead || !hasWrite {
		t.Fatalf("missing tools: read=%v write=%v", hasRead, hasWrite)
	}
	ctx := tools.Context{Actor: "agent", TaskID: "t1"}

	// Write through the wired write tool.
	wres, err := writeTool.Run(map[string]any{
		"action":  "remember",
		"content": "the release train leaves Tuesday",
		"kind":    "decision",
		"scope":   "project",
	}, ctx)
	if err != nil {
		t.Fatalf("remember through wired tool: %v", err)
	}
	hash, _ := wres.Output["hash"].(string)
	if hash == "" {
		t.Fatalf("no hash: %v", wres.Output)
	}

	// Read it back through the wired read tool.
	rres, err := readTool.Run(map[string]any{"action": "recall", "hash": hash}, ctx)
	if err != nil {
		t.Fatalf("recall through wired tool: %v", err)
	}
	if got, _ := rres.Output["content"].(string); got != "the release train leaves Tuesday" {
		t.Errorf("recall content = %q", got)
	}
}

// TestWireMemoryTools_NilMemoryStillRegisters asserts the
// tools are registered even when the Memory could not be
// constructed.
//
// This matters for diagnosis: if an unwritable data dir
// dropped the tools from the bus, a plan asking for
// memory would get ToolNotFoundError and the operator
// would go looking for a policy problem instead of a
// filesystem one.
func TestWireMemoryTools_NilMemoryStillRegisters(t *testing.T) {
	wired := wireMemoryTools(nil, nil)
	for _, name := range []string{"memory", "memory_write"} {
		tool, ok := wired[name]
		if !ok {
			t.Errorf("wireMemoryTools(nil): missing %q", name)
			continue
		}
		if tool == nil {
			t.Errorf("wireMemoryTools(nil): %q is a nil tool", name)
		}
	}
}

// TestWireMemoryTools_NamesMatchKeys pins the canonical
// names against the map keys.
func TestWireMemoryTools_NamesMatchKeys(t *testing.T) {
	for name, tool := range wireMemoryTools(nil, nil) {
		if got := tool.Name(); got != name {
			t.Errorf("key %q but tool.Name() = %q", name, got)
		}
	}
}
