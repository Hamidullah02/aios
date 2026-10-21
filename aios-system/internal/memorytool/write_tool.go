package memorytool

import (
	"fmt"
	"strings"

	"github.com/sadi/aios-system/internal/core/audit"
	"github.com/sadi/aios-system/internal/memory"
	"github.com/sadi/aios-system/internal/tools"
)

// WriteTool exposes the mutating side of the memory.
//
// Actions:
//
//	remember — store an artifact. Args: content (required),
//	           kind, scope, session_id, task_id, tags.
//	forget   — mark an artifact pruned. Args: hash (required).
//
// The defaults matter for a plan that only says "remember
// this": kind defaults to observation, scope defaults to
// project, and session_id defaults to the bus context's
// TaskID so the artifact is still attributable without the
// plan having to invent a session id.
type WriteTool struct {
	// Mem is the memory facade. Required. Shared with
	// ReadTool so a write is visible to a read in the
	// same process.
	Mem *memory.Memory
	// Audit is the audit log. nil means "skip the row".
	Audit *audit.Log
}

func (WriteTool) Name() string { return "memory_write" }

// Run dispatches on the `action` arg.
func (w WriteTool) Run(args map[string]any, ctx tools.Context) (tools.Result, error) {
	if w.Mem == nil {
		return tools.Result{OK: false, Summary: "no memory"},
			&ToolError{Op: "run", Reason: "memory is not wired"}
	}
	action := strings.ToLower(strings.TrimSpace(stringArg(args, "action")))
	switch action {
	case "", "remember", "put", "add":
		return w.remember(args, ctx)
	case "forget", "prune", "delete":
		return w.forget(args, ctx)
	default:
		return tools.Result{OK: false, Summary: "bad action"},
			&ToolError{Op: "run", Reason: fmt.Sprintf("action must be remember or forget; got %q", action)}
	}
}

// remember stores one artifact and returns its hash.
func (w WriteTool) remember(args map[string]any, ctx tools.Context) (tools.Result, error) {
	content := stringArg(args, "content")
	if content == "" {
		return tools.Result{OK: false, Summary: "missing content"},
			&ToolError{Op: "remember", Reason: "content is required"}
	}

	kind := memory.ArtifactKind(stringArg(args, "kind"))
	if kind == "" {
		kind = memory.KindObservation
	}
	scope := memory.Scope(stringArg(args, "scope"))
	if scope == "" {
		scope = memory.ScopeProject
	}
	sessionID := stringArg(args, "session_id")
	if sessionID == "" {
		// Fall back to the task id so the artifact is
		// attributable even when the plan does not
		// name a session.
		sessionID = ctx.TaskID
	}
	if sessionID == "" {
		sessionID = "default"
	}

	artifact, err := memory.NewArtifactWithMeta(
		kind, scope, sessionID,
		stringArg(args, "task_id"),
		splitList(args, "tags"),
		[]byte(content),
	)
	if err != nil {
		return tools.Result{OK: false, Summary: "bad artifact"},
			&ToolError{Op: "remember", Reason: err.Error()}
	}

	hash, err := w.Mem.Remember(artifact)
	if err != nil {
		return tools.Result{OK: false, Summary: "store failed"},
			&ToolError{Op: "remember", Reason: err.Error()}
	}
	w.writeAudit(ctx, "remember", artifact)
	return tools.Result{
		OK:      true,
		Output:  map[string]any{"hash": hash, "kind": string(kind), "scope": string(scope)},
		Summary: fmt.Sprintf("remembered %s...", truncate(hash, 12)),
	}, nil
}

// forget marks an artifact pruned. The raw payload is
// left in place (soft delete, matching Index.Prune /
// REQ-MEM-CLI-005); a later compaction reclaims it.
func (w WriteTool) forget(args map[string]any, ctx tools.Context) (tools.Result, error) {
	hash := strings.TrimSpace(stringArg(args, "hash"))
	if hash == "" {
		return tools.Result{OK: false, Summary: "missing hash"},
			&ToolError{Op: "forget", Reason: "hash is required"}
	}
	// The Memory facade has no Prune; the per-scope Index
	// does, and is reachable through ScopeView.Index(). We
	// prune every scope that holds the hash rather than
	// making the caller name one: content addressing means
	// the same content in two scopes yields the same hash
	// in both, and "forget this thing" should mean the
	// thing is gone everywhere.
	if err := w.pruneAcrossScopes(hash); err != nil {
		return tools.Result{OK: false, Summary: "not found"},
			&ToolError{Op: "forget", Reason: err.Error()}
	}
	w.writeAuditRow(ctx, map[string]any{
		"action": "forget",
		"hash":   hash,
		"pruned": true,
	})
	return tools.Result{
		OK:      true,
		Output:  map[string]any{"hash": hash, "pruned": true},
		Summary: fmt.Sprintf("forgot %s...", truncate(hash, 12)),
	}, nil
}

// pruneAcrossScopes marks the hash pruned in every scope
// that holds it. A hash is content-addressed, so the same
// content stored in two scopes produces the same hash in
// both; pruning all of them is the honest "forget".
func (w WriteTool) pruneAcrossScopes(hash string) error {
	pruned := false
	for _, view := range []*memory.ScopeView{
		w.Mem.Session(), w.Mem.Project(), w.Mem.User(),
	} {
		if view == nil {
			continue
		}
		idx := view.Index()
		if idx == nil {
			continue
		}
		// Index.Prune returns nil when the hash is
		// already pruned (idempotent re-prune) and
		// ErrNotFound when the scope does not hold the
		// hash. Either way we keep going so a hash
		// stored in two scopes is pruned in both.
		if err := idx.Prune(hash); err == nil {
			pruned = true
		}
	}
	if !pruned {
		return &ToolError{Op: "forget", Reason: fmt.Sprintf("no scope holds %s", truncate(hash, 12))}
	}
	return nil
}

// writeAudit records a mutating memory action. The row
// kind is "memory_write" and the payload carries the hash
// and action so an operator can audit what the agent
// persisted.
func (w WriteTool) writeAudit(ctx tools.Context, action string, a memory.Artifact) {
	w.writeAuditRow(ctx, map[string]any{
		"action":     action,
		"hash":       a.ContentHash,
		"kind":       string(a.Kind),
		"scope":      string(a.Scope),
		"session_id": a.SessionID,
		"tags":       a.Tags,
	})
}

// writeAuditRow writes one "memory_write" audit row. The
// helper is a single sink so every mutating action emits
// the same row shape regardless of whether it carried an
// Artifact or just a hash.
func (w WriteTool) writeAuditRow(ctx tools.Context, payload map[string]any) {
	if w.Audit == nil {
		return
	}
	taskID := ctx.TaskID
	if taskID == "" {
		taskID = "memory_write"
	}
	actor := ctx.Actor
	if actor == "" {
		actor = "memory_write"
	}
	_, _ = w.Audit.Write(&audit.Row{
		TaskID:  taskID,
		Kind:    "memory_write",
		Actor:   actor,
		Payload: payload,
	})
}
