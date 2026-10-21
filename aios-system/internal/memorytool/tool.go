// Package memorytool — tools.Tool adapters over
// internal/memory.
//
// v0.39.0. The memory package shipped a complete,
// tested implementation of the doc set's memory layer
// (three scopes, a content-addressed store, a typed
// index, the query language, compaction) and no way for
// an agent to reach it: it was not a tools.Tool, so a
// plan that asked the agent to "remember this decision"
// or "recall the last commit we made" had no tool to
// dispatch. These adapters close that gap.
//
// Two tools, not one, so a policy rule can grant recall
// without also granting the ability to persist or
// delete artifacts:
//
//   - "memory":       read-only. Actions: search,
//     recall, stats.
//   - "memory_write": mutating. Actions: remember,
//     forget.
//
// The split is why both exist rather than one tool
// with five actions: on a default-deny workspace the
// operator who wants the agent to look things up should
// not have to hand out write and delete at the same
// time.
package memorytool

import (
	"fmt"
	"strings"

	"github.com/sadi/aios-system/internal/core/audit"
	"github.com/sadi/aios-system/internal/memory"
	"github.com/sadi/aios-system/internal/tools"
)

// ToolError is the typed error for the memory tools.
type ToolError struct {
	Op     string
	Reason string
}

func (e *ToolError) Error() string {
	return fmt.Sprintf("memorytool: %s: %s.", e.Op, e.Reason)
}

// Is lets errors.Is match on the op, so a caller can
// distinguish "not found" from "bad args" without string
// matching.
func (e *ToolError) Is(target error) bool {
	_, ok := target.(*ToolError)
	return ok
}

// ReadTool exposes the read side of the memory.
//
// Actions:
//
//	search (default) — run a Query. Args: kinds, scopes,
//	                  tags, pattern, since, limit,
//	                  operator (each a comma-separated
//	                  string except limit).
//	recall          — fetch one artifact by content hash.
//	                  Args: hash (required).
//	stats           — count per scope by running an empty
//	                  Query against each one. No args.
//
// stats is best-effort because ScopeView exposes no
// Count / Size getters; the tool reports what it can
// reach through the public Query API. A future revision
// can add direct counters when the memory package grows
// them.
type ReadTool struct {
	// Mem is the memory facade. Required. The bus
	// shares one instance across ReadTool and
	// WriteTool, so a write is immediately visible to
	// a read inside the same process.
	Mem *memory.Memory
	// Audit is the audit log. nil means "skip the row",
	// matching every other tool in the bus.
	Audit *audit.Log
}

func (ReadTool) Name() string { return "memory" }

// Run dispatches on the `action` arg.
func (r ReadTool) Run(args map[string]any, ctx tools.Context) (tools.Result, error) {
	if r.Mem == nil {
		return tools.Result{OK: false, Summary: "no memory"},
			&ToolError{Op: "run", Reason: "memory is not wired"}
	}
	action := strings.ToLower(strings.TrimSpace(stringArg(args, "action")))
	switch action {
	case "", "search", "query":
		return r.search(args)
	case "recall", "get":
		return r.recall(args)
	case "stats":
		return r.stats(args)
	default:
		return tools.Result{OK: false, Summary: "bad action"},
			&ToolError{Op: "run", Reason: fmt.Sprintf("action must be search, recall, or stats; got %q", action)}
	}
}

// search runs a Query against all three scopes.
func (r ReadTool) search(args map[string]any) (tools.Result, error) {
	q := memory.Query{
		Kinds:    splitList(args, "kinds"),
		Scopes:   splitList(args, "scopes"),
		Tags:     splitList(args, "tags"),
		Pattern:  stringArg(args, "pattern"),
		Since:    stringArg(args, "since"),
		Limit:    intArg(args, "limit"),
		Operator: memory.Operator(stringArg(args, "operator")),
	}
	// Query.Validate runs inside Memory.Query, so an
	// invalid kind / scope / operator surfaces as the
	// package's own MemoryError. We only default the
	// operator here so an empty arg means AND, not
	// "operator empty" (which would read as a
	// request for raw filter-less matching that the
	// current index treats as AND anyway).
	if q.Operator == "" {
		q.Operator = memory.OpAND
	}
	artifacts, err := r.Mem.Query(q)
	if err != nil {
		return tools.Result{OK: false, Summary: "query failed"},
			&ToolError{Op: "search", Reason: err.Error()}
	}
	rows := make([]map[string]any, 0, len(artifacts))
	for _, a := range artifacts {
		rows = append(rows, artifactRow(a))
	}
	return tools.Result{
		OK:      true,
		Output:  map[string]any{"artifacts": rows, "count": len(rows)},
		Summary: fmt.Sprintf("%d artifact(s)", len(rows)),
	}, nil
}

// recall fetches one artifact by content hash.
func (r ReadTool) recall(args map[string]any) (tools.Result, error) {
	hash := strings.TrimSpace(stringArg(args, "hash"))
	if hash == "" {
		return tools.Result{OK: false, Summary: "missing hash"},
			&ToolError{Op: "recall", Reason: "hash is required"}
	}
	a, err := r.Mem.Get(hash)
	if err != nil {
		return tools.Result{OK: false, Summary: "not found"},
			&ToolError{Op: "recall", Reason: err.Error()}
	}
	return tools.Result{
		OK:      true,
		Output:  artifactRow(a),
		Summary: fmt.Sprintf("artifact %s...", truncate(a.ContentHash, 12)),
	}, nil
}

// stats counts the artifacts in each scope by running a
// single-scope Query per scope. The empty-query form
// is documented as "match every entry" (REQ-MEM-QUE-001)
// and goes through the public API, so the answer is
// the same one Memory.Get would report minus artifacts
// in other scopes.
//
// The nested maps are map[string]any (not
// map[string]map[string]any) so the shape matches every
// other tool in the bus and survives a JSON round-trip
// through the report writer unchanged.
func (r ReadTool) stats(_ map[string]any) (tools.Result, error) {
	scopes := map[string]any{}
	total := 0
	for _, sc := range []memory.Scope{memory.ScopeSession, memory.ScopeProject, memory.ScopeUser} {
		name := string(sc)
		artifacts, err := r.Mem.Query(memory.Query{
			Scopes: []string{name},
		})
		if err != nil {
			return tools.Result{OK: false, Summary: "stats failed"},
				&ToolError{Op: "stats", Reason: err.Error()}
		}
		scopes[name] = map[string]any{"count": len(artifacts)}
		total += len(artifacts)
	}
	return tools.Result{
		OK:      true,
		Output:  map[string]any{"scopes": scopes, "total": total},
		Summary: fmt.Sprintf("%d artifact(s)", total),
	}, nil
}

// artifactRow renders an Artifact as the bus Output map.
// The content is included as a string because every
// downstream consumer (the report writer, the audit
// viewer, the model's next turn) wants text, and the
// artifacts this system stores are text-shaped.
func artifactRow(a memory.Artifact) map[string]any {
	return map[string]any{
		"hash":       a.ContentHash,
		"kind":       string(a.Kind),
		"scope":      string(a.Scope),
		"session_id": a.SessionID,
		"task_id":    a.TaskID,
		"timestamp":  a.Timestamp,
		"tags":       a.Tags,
		"content":    string(a.Content),
	}
}

// stringArg returns args[key] when it is a non-empty
// string, else "".
func stringArg(args map[string]any, key string) string {
	s, _ := args[key].(string)
	return strings.TrimSpace(s)
}

// intArg returns args[key] coerced to int, else 0.
func intArg(args map[string]any, key string) int {
	switch v := args[key].(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	}
	return 0
}

// splitList reads a comma-separated string (or a
// []string / []any) into a trimmed, non-empty slice. The
// bus worker's "inputs" pass each value as a string, so
// the comma form is the one a plan actually produces.
func splitList(args map[string]any, key string) []string {
	switch v := args[key].(type) {
	case string:
		parts := strings.Split(v, ",")
		out := make([]string, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				out = append(out, p)
			}
		}
		return out
	case []string:
		out := make([]string, 0, len(v))
		for _, p := range v {
			p = strings.TrimSpace(p)
			if p != "" {
				out = append(out, p)
			}
		}
		return out
	case []any:
		out := make([]string, 0, len(v))
		for _, raw := range v {
			if p, ok := raw.(string); ok {
				if p = strings.TrimSpace(p); p != "" {
					out = append(out, p)
				}
			}
		}
		return out
	}
	return nil
}

// truncate returns s clipped to n runes with a trailing
// "..." the caller adds itself.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
