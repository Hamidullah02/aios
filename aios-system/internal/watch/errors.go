package watch

import (
	"errors"
	"fmt"
)

// Sentinel errors. Per REQ-WTC-ERR-001 / 002.
var (
	// ErrWatch is the package-level sentinel for file-watcher failures.
	// Wrap with %w; callers can match with errors.Is.
	ErrWatch = errors.New("watch: file watcher error")

	// ErrSubAgent is the package-level sentinel for sub-agent failures.
	ErrSubAgent = errors.New("watch: sub-agent error")
)

// WatchError is the typed error returned by the file watcher.
// Path is the path that triggered the failure; Op is the
// operation in flight when the failure happened.
type WatchError struct {
	Op     string
	Path   string
	Reason string
	Err    error
}

func (e *WatchError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("watch: %s %s: %s: %v", e.Op, e.Path, e.Reason, e.Err)
	}
	return fmt.Sprintf("watch: %s %s: %s", e.Op, e.Path, e.Reason)
}

func (e *WatchError) Unwrap() error { return e.Err }
func (e *WatchError) Is(target error) bool {
	return target == ErrWatch || errors.Is(e.Err, target)
}

// SubAgentError is the typed error returned by the Task tool.
// AgentType is the sub-agent type that was requested; Prompt is
// the prompt that triggered the failure (truncated by callers when
// needed).
type SubAgentError struct {
	AgentType string
	Prompt    string
	Reason    string
	Err       error
}

func (e *SubAgentError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("watch: sub-agent %s: %s: %v", e.AgentType, e.Reason, e.Err)
	}
	return fmt.Sprintf("watch: sub-agent %s: %s", e.AgentType, e.Reason)
}

func (e *SubAgentError) Unwrap() error { return e.Err }
func (e *SubAgentError) Is(target error) bool {
	return target == ErrSubAgent || errors.Is(e.Err, target)
}
