package memorytool

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/sadi/aios-system/internal/core/audit"
	"github.com/sadi/aios-system/internal/memory"
	"github.com/sadi/aios-system/internal/tools"
)

// newTestMemory builds a Memory rooted at a temp dir so
// the project and user scopes are real on-disk stores
// while the session scope stays in memory.
func newTestMemory(t *testing.T) *memory.Memory {
	t.Helper()
	dir := t.TempDir()
	m, err := memory.NewMemory(memory.MemoryConfig{
		DataDir: filepath.Join(dir, "data"),
		Home:    filepath.Join(dir, "home"),
	})
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func newTestAudit(t *testing.T) *audit.Log {
	t.Helper()
	log, err := audit.Open(audit.Config{Path: ":memory:"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { log.Close() })
	return log
}

// TestReadWrite_RoundTrip is the v0.39.0 gate: an
// artifact written through memory_write is readable
// through memory, through a search and through a recall
// by the returned hash.
//
// The two tools share one *memory.Memory, which is the
// property that makes the split into read + write safe:
// a write is visible to a read in the same process.
func TestReadWrite_RoundTrip(t *testing.T) {
	mem := newTestMemory(t)
	log := newTestAudit(t)
	read := ReadTool{Mem: mem, Audit: log}
	write := WriteTool{Mem: mem, Audit: log}
	ctx := tools.Context{Actor: "agent", TaskID: "t1"}

	// Write.
	wres, err := write.Run(map[string]any{
		"action":  "remember",
		"content": "the deploy key is in vault under ops/deploy",
		"kind":    "decision",
		"scope":   "project",
		"tags":    "deploy,secrets",
	}, ctx)
	if err != nil {
		t.Fatalf("remember: %v", err)
	}
	if !wres.OK {
		t.Fatalf("remember OK=false, summary=%s", wres.Summary)
	}
	hash, _ := wres.Output["hash"].(string)
	if hash == "" {
		t.Fatalf("remember returned no hash: %v", wres.Output)
	}

	// Read back by hash.
	rres, err := read.Run(map[string]any{"action": "recall", "hash": hash}, ctx)
	if err != nil {
		t.Fatalf("recall: %v", err)
	}
	if got, _ := rres.Output["content"].(string); !strings.Contains(got, "vault under ops/deploy") {
		t.Errorf("recall content = %q", got)
	}
	if got, _ := rres.Output["kind"].(string); got != "decision" {
		t.Errorf("recall kind = %q, want decision", got)
	}

	// Search by tag.
	sres, err := read.Run(map[string]any{"action": "search", "tags": "deploy"}, ctx)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if n, _ := sres.Output["count"].(int); n != 1 {
		t.Errorf("search count = %v, want 1 (output=%v)", sres.Output["count"], sres.Output)
	}
}

// TestReadWrite_SharedInstance asserts the ReadTool sees
// what the WriteTool stored. This is the regression guard
// for wiring two separate Memory instances by mistake,
// which would silently split the memory in half.
func TestReadWrite_SharedInstance(t *testing.T) {
	shared := newTestMemory(t)
	other := newTestMemory(t)

	write := WriteTool{Mem: shared}
	read := ReadTool{Mem: shared}

	if _, err := write.Run(map[string]any{
		"action": "remember", "content": "shared only",
	}, tools.Context{TaskID: "t"}); err != nil {
		t.Fatal(err)
	}

	res, err := read.Run(map[string]any{"action": "search"}, tools.Context{TaskID: "t"})
	if err != nil {
		t.Fatal(err)
	}
	if n, _ := res.Output["count"].(int); n != 1 {
		t.Errorf("shared instance: count = %v, want 1", res.Output["count"])
	}

	// The other instance must NOT see it.
	res2, err := ReadTool{Mem: other}.Run(map[string]any{"action": "search"}, tools.Context{TaskID: "t"})
	if err != nil {
		t.Fatal(err)
	}
	if n, _ := res2.Output["count"].(int); n != 0 {
		t.Errorf("isolated instance: count = %v, want 0", res2.Output["count"])
	}
}

// TestRemember_Defaults covers the "remember this" plan
// that supplies only content. kind defaults to
// observation, scope to project, and session_id falls back
// to the bus context's TaskID.
func TestRemember_Defaults(t *testing.T) {
	mem := newTestMemory(t)
	write := WriteTool{Mem: mem}
	res, err := write.Run(map[string]any{
		"content": "minimal plan input",
	}, tools.Context{Actor: "agent", TaskID: "task-77"})
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK {
		t.Fatalf("OK=false: %s", res.Summary)
	}
	if got, _ := res.Output["kind"].(string); got != "observation" {
		t.Errorf("default kind = %q, want observation", got)
	}
	if got, _ := res.Output["scope"].(string); got != "project" {
		t.Errorf("default scope = %q, want project", got)
	}
	// The artifact must be attributable to task-77.
	got, err := mem.Get(res.Output["hash"].(string))
	if err != nil {
		t.Fatalf("get back: %v", err)
	}
	if got.SessionID != "task-77" {
		t.Errorf("session_id = %q, want task-77", got.SessionID)
	}
}

// TestRemember_RequiresContent covers the validation path.
func TestRemember_RequiresContent(t *testing.T) {
	write := WriteTool{Mem: newTestMemory(t)}
	res, err := write.Run(map[string]any{"action": "remember"}, tools.Context{TaskID: "t"})
	if err == nil {
		t.Fatal("expected an error for empty content")
	}
	if res.OK {
		t.Error("OK should be false when content is missing")
	}
}

// TestRemember_FallsBackOnAnUnknownKind pins that an unknown
// kind is a near-miss rather than a refusal.
//
// The model was asked to remember a fact and wrote kind
// "fact", which is not one of the six kinds the memory knows.
// The tool used to answer "kind must be one of task, file,
// decision, observation, tool_result, model_output; got
// \"fact\"" and the whole task failed.
//
// The agent understood what it wanted to remember. The kind was
// a label, and "observation" -- the kind that means "the agent
// noticed something" -- is what it actually was. The memory
// package's own validator still rejects an invalid kind; the
// tool is what normalizes before calling it.
func TestRemember_FallsBackOnAnUnknownKind(t *testing.T) {
	mem := newTestMemory(t)
	write := WriteTool{Mem: mem}
	res, err := write.Run(map[string]any{
		"content": "x", "kind": "not-a-kind",
	}, tools.Context{TaskID: "t"})
	if err != nil {
		t.Fatalf("an unknown kind should not fail the task: %v", err)
	}
	if !res.OK {
		t.Fatalf("the write did not happen: %+v", res)
	}
	// The artifact was stored, and it was stored as an
	// observation.
	out := res.Output
	if got, _ := out["kind"].(string); got != string(memory.KindObservation) {
		t.Errorf("kind = %q, want %q", got, memory.KindObservation)
	}
}

// TestForget_PrunesAcrossScopes covers the soft-delete
// path. A hash stored in two scopes (the same content in
// session and project) must be pruned in both, because
// content addressing gives both the same hash.
func TestForget_PrunesAcrossScopes(t *testing.T) {
	mem := newTestMemory(t)
	write := WriteTool{Mem: mem}
	ctx := tools.Context{TaskID: "t"}

	// Same content in two scopes -> same hash in both.
	h1, err := write.Run(map[string]any{
		"content": "duplicated", "scope": "session",
	}, ctx)
	if err != nil {
		t.Fatal(err)
	}
	h2, err := write.Run(map[string]any{
		"content": "duplicated", "scope": "project",
	}, ctx)
	if err != nil {
		t.Fatal(err)
	}
	if h1.Output["hash"] != h2.Output["hash"] {
		t.Fatalf("expected the same content-addressed hash: %v vs %v",
			h1.Output["hash"], h2.Output["hash"])
	}
	hash := h1.Output["hash"].(string)

	// Still visible before forget.
	if !mem.Has(hash) {
		t.Fatal("artifact should exist before forget")
	}
	res, err := write.Run(map[string]any{"action": "forget", "hash": hash}, ctx)
	if err != nil {
		t.Fatalf("forget: %v", err)
	}
	if !res.OK {
		t.Fatalf("forget OK=false: %s", res.Summary)
	}
	// Pruned in every scope: a fresh Query over the index
	// must not return it.
	found, err := mem.Query(memory.Query{})
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range found {
		if a.ContentHash == hash {
			t.Errorf("artifact %s survived forget", hash)
		}
	}
}

// TestForget_UnknownHash covers the not-found path: the
// tool reports an error rather than silently succeeding.
func TestForget_UnknownHash(t *testing.T) {
	write := WriteTool{Mem: newTestMemory(t)}
	res, err := write.Run(map[string]any{
		"action": "forget",
		"hash":   strings.Repeat("a", 64),
	}, tools.Context{TaskID: "t"})
	if err == nil {
		t.Fatal("expected an error for an unknown hash")
	}
	if res.OK {
		t.Error("OK should be false for an unknown hash")
	}
}

// TestForget_RequiresHash covers the validation path.
func TestForget_RequiresHash(t *testing.T) {
	write := WriteTool{Mem: newTestMemory(t)}
	if _, err := write.Run(map[string]any{"action": "forget"}, tools.Context{TaskID: "t"}); err == nil {
		t.Fatal("expected an error for a missing hash")
	}
}

// TestBadAction_IsRejected asserts an unknown action on
// either tool is an error, not a silent no-op. Without
// this a typo like action="serach" would return an empty
// result and read to the model as "no matches".
func TestBadAction_IsRejected(t *testing.T) {
	mem := newTestMemory(t)
	ctx := tools.Context{TaskID: "t"}
	if _, err := (ReadTool{Mem: mem}).Run(map[string]any{"action": "serach"}, ctx); err == nil {
		t.Error("read: expected an error for an unknown action")
	}
	if _, err := (WriteTool{Mem: mem}).Run(map[string]any{"action": "destroy"}, ctx); err == nil {
		t.Error("write: expected an error for an unknown action")
	}
}

// TestNilMemory_IsAnError asserts the tools fail cleanly
// when the bus forgot to wire a Memory, rather than
// panicking on the nil pointer.
func TestNilMemory_IsAnError(t *testing.T) {
	ctx := tools.Context{TaskID: "t"}
	if _, err := (ReadTool{}).Run(map[string]any{}, ctx); err == nil {
		t.Error("read: expected an error for a nil memory")
	}
	if _, err := (WriteTool{}).Run(map[string]any{"content": "x"}, ctx); err == nil {
		t.Error("write: expected an error for a nil memory")
	}
}

// TestStats_CountsPerScope covers the stats action.
func TestStats_CountsPerScope(t *testing.T) {
	mem := newTestMemory(t)
	write := WriteTool{Mem: mem}
	ctx := tools.Context{TaskID: "t"}
	for i := 0; i < 3; i++ {
		if _, err := write.Run(map[string]any{
			"content": "artifact " + string(rune('a'+i)), "scope": "project",
		}, ctx); err != nil {
			t.Fatal(err)
		}
	}
	res, err := (ReadTool{Mem: mem}).Run(map[string]any{"action": "stats"}, ctx)
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	t.Logf("stats output = %#v", res.Output)
	if n, _ := res.Output["total"].(int); n != 3 {
		t.Errorf("total = %v, want 3 (output=%v)", res.Output["total"], res.Output)
	}
	scopes, ok := res.Output["scopes"].(map[string]any)
	if !ok {
		t.Fatalf("scopes is %T, want map[string]any", res.Output["scopes"])
	}
	proj, ok := scopes["project"].(map[string]any)
	if !ok {
		t.Fatalf("scopes[project] is %T (%v), want map[string]any", scopes["project"], scopes["project"])
	}
	if n, _ := proj["count"].(int); n != 3 {
		t.Errorf("project count = %v, want 3", proj["count"])
	}
}

// TestWriteAudit_RowsAreWritten covers the audit
// contract: every mutating action emits a memory_write row.
func TestWriteAudit_RowsAreWritten(t *testing.T) {
	dir := t.TempDir()
	log, err := audit.Open(audit.Config{Path: filepath.Join(dir, "aios.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()

	mem := newTestMemory(t)
	write := WriteTool{Mem: mem, Audit: log}
	ctx := tools.Context{Actor: "agent", TaskID: "t1"}

	res, err := write.Run(map[string]any{"content": "audited"}, ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := write.Run(map[string]any{
		"action": "forget", "hash": res.Output["hash"].(string),
	}, ctx); err != nil {
		t.Fatal(err)
	}

	rows, err := log.GetByKind("memory_write", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 memory_write rows, got %d", len(rows))
	}
}

// TestToolNames pins the two canonical names. A rename
// here is a breaking change for every existing plan.
func TestToolNames(t *testing.T) {
	if got := (ReadTool{}).Name(); got != "memory" {
		t.Errorf("read name = %q, want memory", got)
	}
	if got := (WriteTool{}).Name(); got != "memory_write" {
		t.Errorf("write name = %q, want memory_write", got)
	}
}

// TestSplitList covers the arg coercion helpers, since the
// bus worker hands every plan input over as a string.
func TestSplitList(t *testing.T) {
	args := map[string]any{
		"csv":   "a, b ,c,",
		"slice": []string{"x", " y "},
		"anys":  []any{"p", 1, "q"},
		"bad":   42,
	}
	if got := splitList(args, "csv"); len(got) != 3 || got[2] != "c" {
		t.Errorf("csv = %v", got)
	}
	if got := splitList(args, "slice"); len(got) != 2 || got[1] != "y" {
		t.Errorf("slice = %v", got)
	}
	if got := splitList(args, "anys"); len(got) != 2 || got[1] != "q" {
		t.Errorf("anys = %v", got)
	}
	if got := splitList(args, "bad"); got != nil {
		t.Errorf("bad = %v, want nil", got)
	}
	if got := splitList(args, "missing"); got != nil {
		t.Errorf("missing = %v, want nil", got)
	}
}

// TestIntArg covers the numeric coercion, including the
// float64 a JSON round-trip would produce.
func TestIntArg(t *testing.T) {
	args := map[string]any{"i": 5, "f": float64(7), "s": "x"}
	if got := intArg(args, "i"); got != 5 {
		t.Errorf("i = %d", got)
	}
	if got := intArg(args, "f"); got != 7 {
		t.Errorf("f = %d", got)
	}
	if got := intArg(args, "s"); got != 0 {
		t.Errorf("s = %d", got)
	}
}
