// Package tools is the tool bus and the built-in tools.
//
// The package exports the Tool interface, the ToolBus, and
// four built-in tools: ReadFileTool, WriteFileTool,
// SearchFilesTool, RunShellTool. The bus consults the policy
// gateway before every call and writes one `tool_call` audit
// row per call. The file tools validate the path against
// the workspace root; the shell tool runs through the
// sandbox runner.
package tools

import (
	"errors"
	"fmt"
)

// Sentinel errors. Per the aios-tools SRS §9.
var (
	// ErrPathNotAllowed is returned when a path resolves
	// outside the workspace root (REQ-TOL-ERR-001).
	ErrPathNotAllowed = errors.New("tools: path not allowed")

	// ErrToolTimeout is returned when a tool exceeds its
	// timeout (REQ-TOL-ERR-002).
	ErrToolTimeout = errors.New("tools: tool timeout")

	// ErrToolNotFound is returned when the bus is asked to
	// dispatch a tool name that is not in the tool map
	// (REQ-TOL-ERR-003).
	ErrToolNotFound = errors.New("tools: tool not found")
)

// PathNotAllowedError is the typed error for path-validation
// failures. The Path is the rejected path; the Workspace is
// the configured root.
type PathNotAllowedError struct {
	Path      string
	Workspace string
}

func (e *PathNotAllowedError) Error() string {
	return fmt.Sprintf("tools: path %q is outside workspace %q", e.Path, e.Workspace)
}

func (e *PathNotAllowedError) Is(target error) bool { return target == ErrPathNotAllowed }
func (e *PathNotAllowedError) Unwrap() error        { return ErrPathNotAllowed }

// ToolNotFoundError is the typed error for missing tools.
type ToolNotFoundError struct {
	Name string
}

func (e *ToolNotFoundError) Error() string {
	return fmt.Sprintf("tools: tool %q not found", e.Name)
}

func (e *ToolNotFoundError) Is(target error) bool { return target == ErrToolNotFound }
func (e *ToolNotFoundError) Unwrap() error        { return ErrToolNotFound }

// ToolTimeoutError is the typed error for tool-level timeouts
// (as opposed to sandbox-level timeouts).
type ToolTimeoutError struct {
	Tool    string
	Timeout string
}

func (e *ToolTimeoutError) Error() string {
	return fmt.Sprintf("tools: tool %q timed out after %s", e.Tool, e.Timeout)
}

func (e *ToolTimeoutError) Is(target error) bool { return target == ErrToolTimeout }
func (e *ToolTimeoutError) Unwrap() error        { return ErrToolTimeout }
