package watch

import (
	"errors"
	"strings"
	"testing"
)

// TestWatchError_ErrorBranches covers the with/without-Err
// branches of WatchError.Error.
func TestWatchError_ErrorBranches(t *testing.T) {
	e1 := &WatchError{Op: "x", Path: "p", Reason: "r"}
	if !strings.Contains(e1.Error(), "watch:") {
		t.Fatalf("msg = %q", e1.Error())
	}
	e2 := &WatchError{Op: "x", Path: "p", Reason: "r", Err: errors.New("inner")}
	if !strings.Contains(e2.Error(), "inner") {
		t.Fatalf("msg = %q", e2.Error())
	}
	if e2.Unwrap() == nil {
		t.Fatal("Unwrap should return Err")
	}
}

// TestSubAgentError_ErrorBranches covers the with/without-Err
// branches of SubAgentError.Error.
func TestSubAgentError_ErrorBranches(t *testing.T) {
	e1 := &SubAgentError{AgentType: "explore", Prompt: "p", Reason: "r"}
	if !strings.Contains(e1.Error(), "explore") {
		t.Fatalf("msg = %q", e1.Error())
	}
	e2 := &SubAgentError{AgentType: "explore", Prompt: "p", Reason: "r", Err: errors.New("inner")}
	if !strings.Contains(e2.Error(), "inner") {
		t.Fatalf("msg = %q", e2.Error())
	}
	if e2.Unwrap() == nil {
		t.Fatal("Unwrap should return Err")
	}
}

// TestToolNames covers the Name() methods on FileWatcherTool
// and TaskTool.
func TestToolNames(t *testing.T) {
	if (FileWatcherTool{}).Name() != "file_watcher" {
		t.Fatal("FileWatcherTool.Name mismatch")
	}
	if (TaskTool{}).Name() != "task" {
		t.Fatal("TaskTool.Name mismatch")
	}
}

// TestTruncate covers the truncation helper.
func TestTruncate(t *testing.T) {
	if truncate("hi", 10) != "hi" {
		t.Fatal("short strings pass through")
	}
	if truncate("hello world", 5) != "hello..." {
		t.Fatalf("truncate = %q", truncate("hello world", 5))
	}
}

// TestSubAgentError_IsTarget covers the errors.Is match when
// target is a *non*-sentinel error.
func TestSubAgentError_IsTarget(t *testing.T) {
	boom := errors.New("boom")
	e := &SubAgentError{Err: boom}
	if !e.Is(boom) {
		t.Fatal("SubAgentError should match its underlying error")
	}
}

// TestWatchError_IsTarget covers the same for WatchError.
func TestWatchError_IsTarget(t *testing.T) {
	boom := errors.New("boom")
	e := &WatchError{Err: boom}
	if !e.Is(boom) {
		t.Fatal("WatchError should match its underlying error")
	}
}
