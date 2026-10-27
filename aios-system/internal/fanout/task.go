package fanout

import (
	"fmt"
	"time"

	"github.com/Hamidullah02/aios/aios-system/internal/core/schemas"
)

// MergeStrategy is the typed enum for the result
// merger strategies (REQ-FAN-RMG-001).
type MergeStrategy string

const (
	// MergeConcat concatenates the results in
	// order (REQ-FAN-RMG-002). The strategy is the
	// safe default: no information loss.
	MergeConcat MergeStrategy = "concat"
	// MergeSummary summarizes the results through
	// the caller's Summarizer (REQ-FAN-RMG-003).
	// The strategy requires a Summarizer to be
	// supplied to the ResultMerger; otherwise the
	// merger falls back to Concat.
	MergeSummary MergeStrategy = "summary"
	// MergeVote picks the result that matches the
	// majority of the other results (REQ-FAN-RMG-004).
	// The strategy is content-based: the merger
	// scores each result by character-level
	// similarity to the others and picks the
	// highest-scoring one.
	MergeVote MergeStrategy = "vote"
)

// IsValid reports whether s is one of the three
// known strategies.
func (s MergeStrategy) IsValid() bool {
	switch s {
	case MergeConcat, MergeSummary, MergeVote:
		return true
	}
	return false
}

// FanOutTask is the typed object that describes
// one parallel task (REQ-FAN-DAT-001).
//
// The task has an id, a prompt, an agent type, an
// optional isolate flag, and an optional branch
// for the worktree.
type FanOutTask struct {
	// ID is the unique identifier of the task. The
	// field is required.
	ID string `json:"id"`
	// Prompt is the prompt for the sub-agent. The
	// field is required.
	Prompt string `json:"prompt"`
	// AgentType is the agent type the sub-agent
	// should run as. The field is required.
	AgentType string `json:"agent_type"`
	// Isolate requests a per-task git worktree.
	// The worktree is created before the sub-agent
	// runs and removed after the sub-agent returns.
	Isolate bool `json:"isolate,omitempty"`
	// Branch is the worktree branch. The field is
	// optional; the worktree is created on a
	// detached HEAD when empty.
	Branch string `json:"branch,omitempty"`
}

// Validate checks the task is well-formed. The
// function is the gate the FanOutTool calls
// before enqueuing the task.
func (t *FanOutTask) Validate() error {
	if t == nil {
		return &FanOutError{Op: "validate", Reason: "task is nil"}
	}
	if t.ID == "" {
		return &FanOutError{Op: "validate", Reason: "id is required"}
	}
	if t.Prompt == "" {
		return &FanOutError{Op: "validate", TaskID: t.ID, Reason: "prompt is required"}
	}
	if t.AgentType == "" {
		return &FanOutError{Op: "validate", TaskID: t.ID, Reason: "agent_type is required"}
	}
	return nil
}

// FanOutStatus is the typed status of a fan-out
// result (REQ-FAN-DAT-002).
type FanOutStatus string

const (
	// StatusSucceeded is the success status.
	StatusSucceeded FanOutStatus = "succeeded"
	// StatusFailed is the failure status.
	StatusFailed FanOutStatus = "failed"
	// StatusCanceled is the canceled status.
	StatusCanceled FanOutStatus = "canceled"
)

// IsValid reports whether s is one of the three
// known statuses.
func (s FanOutStatus) IsValid() bool {
	switch s {
	case StatusSucceeded, StatusFailed, StatusCanceled:
		return true
	}
	return false
}

// FanOutResult is the typed object that describes
// the merged result of one parallel task
// (REQ-FAN-DAT-002).
//
// The result carries the task id, the output, the
// status, the duration in milliseconds, the cost
// in USD, and the index in the input list.
type FanOutResult struct {
	// ID is the task id the result is for.
	ID string `json:"id"`
	// Index is the position of the task in the
	// input list. The field is informational; the
	// ID is the join key.
	Index int `json:"index"`
	// Output is the output of the sub-agent. The
	// field is the merged output for the task.
	Output string `json:"output"`
	// Status is the result status.
	Status FanOutStatus `json:"status"`
	// DurationMs is the duration in milliseconds.
	DurationMs int64 `json:"duration_ms"`
	// CostUSD is the cost in USD.
	CostUSD float64 `json:"cost_usd"`
	// StartedAt is the ISO 8601 timestamp at
	// which the sub-agent started.
	StartedAt string `json:"started_at,omitempty"`
	// EndedAt is the ISO 8601 timestamp at which
	// the sub-agent ended.
	EndedAt string `json:"ended_at,omitempty"`
	// WorktreePath is the worktree path the
	// sub-agent ran in. The field is empty when
	// the task was not isolated.
	WorktreePath string `json:"worktree_path,omitempty"`
}

// Validate checks the result is well-formed. The
// function is the gate the FanOutResult.Status
// enum check uses.
func (r *FanOutResult) Validate() error {
	if r == nil {
		return &FanOutError{Op: "validate", Reason: "result is nil"}
	}
	if r.ID == "" {
		return &FanOutError{Op: "validate", Reason: "id is required"}
	}
	if !r.Status.IsValid() {
		return &FanOutError{Op: "validate", TaskID: r.ID, Reason: fmt.Sprintf("status must be one of succeeded, failed, canceled; got %q", r.Status)}
	}
	if r.DurationMs < 0 {
		return &FanOutError{Op: "validate", TaskID: r.ID, Reason: "duration_ms must be non-negative"}
	}
	if r.CostUSD < 0 {
		return &FanOutError{Op: "validate", TaskID: r.ID, Reason: "cost_usd must be non-negative"}
	}
	return nil
}

// schemasSchema is the typed schema the result
// uses when it is written to the audit log. The
// function is the helper the audit log calls; the
// schemas.Schema interface is the canonical
// payload contract.
//
// The function is unused by the merger; it lives
// here so the FanOutResult satisfies the
// audit-payload validation contract when the
// FanOutTool writes a fanout_task audit row.
func (r *FanOutResult) ValidateSchema() error { return r.Validate() }

// Compile-time check that FanOutResult satisfies
// the schemas.Schema interface (audit payload
// contract). The check is a no-op at runtime; it
// keeps the import of schemas from going unused
// when the audit log is wired.
var _ schemas.Schema = (*FanOutResult)(nil)

// timer is the small wall-clock helper the result
// uses for the DurationMs field. The struct
// replaces the deprecated time.Now/time.Since
// pair with a single object the test can replace.
type timer struct {
	start time.Time
}

// newTimer returns a fresh timer. The function is
// the constructor the FanOutTool calls at the
// start of every task.
func newTimer() *timer { return &timer{start: time.Now()} }

// elapsed returns the milliseconds since the
// timer was created. The function is the
// DurationMs helper.
func (t *timer) elapsed() int64 {
	if t == nil {
		return 0
	}
	return elapsedMs(t.start)
}
