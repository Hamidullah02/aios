package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/sadi/aios-system/internal/core/audit"
)

// TestAuditCmd_UnknownTaskIDExplainsItself is the v0.40.0
// gate for the audit command.
//
// Before this change, `aios audit --task=no-such-task`
// printed absolutely nothing and exited 0. An operator
// running that command has no way to tell "that task does
// not exist" from "the audit log is empty" from "I am
// pointed at the wrong database file". The command now
// says which query came back empty.
func TestAuditCmd_UnknownTaskIDExplainsItself(t *testing.T) {
	log := openTestAudit(t)
	var stdout, stderr bytes.Buffer
	code := runAudit(&Args{Command: CmdAudit, TaskID: "no-such-task-xyz"}, &Deps{Audit: log}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit = %d, want 0 (an unknown task is a question, not an error); stderr=%s",
			code, stderr.String())
	}
	got := stdout.String()
	if !strings.Contains(got, "no rows for task no-such-task-xyz") {
		t.Errorf("stdout = %q, want it to name the task that came back empty", got)
	}
	if strings.Contains(got, "error") {
		t.Errorf("stdout = %q, want no error text on a normal empty answer", got)
	}
}

// TestAuditCmd_EmptyLogExplainsItself covers the
// no-task-id variant: an empty log must also say so
// rather than printing a blank screen.
func TestAuditCmd_EmptyLogExplainsItself(t *testing.T) {
	log := openTestAudit(t)
	var stdout, stderr bytes.Buffer
	code := runAudit(&Args{Command: CmdAudit}, &Deps{Audit: log}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit = %d, want 0; stderr=%s", code, stderr.String())
	}
	got := stdout.String()
	if !strings.Contains(got, "no rows in the audit log") {
		t.Errorf("stdout = %q, want an explicit empty-log message", got)
	}
}

// TestAuditCmd_KnownTaskIDStillListsRows is the
// regression guard: adding the empty-result message must
// not suppress or alter the normal listing.
func TestAuditCmd_KnownTaskIDStillListsRows(t *testing.T) {
	log := openTestAudit(t)
	if _, err := log.Write(&audit.Row{
		TaskID:  "task-abc",
		Kind:    "tool_call",
		Actor:   "agent",
		Payload: map[string]any{"tool": "read_file"},
	}); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runAudit(&Args{Command: CmdAudit, TaskID: "task-abc"}, &Deps{Audit: log}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit = %d, want 0; stderr=%s", code, stderr.String())
	}
	got := stdout.String()
	if !strings.Contains(got, "task-abc") || !strings.Contains(got, "tool_call") {
		t.Errorf("stdout = %q, want the task's row listed", got)
	}
	if strings.Contains(got, "no rows for task") {
		t.Errorf("stdout = %q, want no empty-result message when rows exist", got)
	}
}

// TestAuditCmd_NoTaskIDListsAllKinds asserts the union
// path still works after the change: rows of several kinds
// appear in a single listing.
func TestAuditCmd_NoTaskIDListsAllKinds(t *testing.T) {
	log := openTestAudit(t)
	for _, row := range []*audit.Row{
		{TaskID: "t1", Kind: "task_started", Actor: "orch", Payload: map[string]any{}},
		{TaskID: "t1", Kind: "tool_call", Actor: "agent", Payload: map[string]any{"tool": "glob"}},
		// memory_write is the kind v0.39.0 added to the
		// union list; it must survive this change too.
		{TaskID: "t1", Kind: "memory_write", Actor: "memory_write", Payload: map[string]any{"hash": "abc"}},
	} {
		if _, err := log.Write(row); err != nil {
			t.Fatal(err)
		}
	}
	var stdout, stderr bytes.Buffer
	code := runAudit(&Args{Command: CmdAudit}, &Deps{Audit: log}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit = %d, want 0; stderr=%s", code, stderr.String())
	}
	got := stdout.String()
	for _, want := range []string{"task_started", "tool_call", "memory_write"} {
		if !strings.Contains(got, want) {
			t.Errorf("stdout = %q, want a %s row", got, want)
		}
	}
}

// TestParse_AuditAllowsEmptyTaskID is the parser-level
// guard: `aios audit` with no --task must parse rather than
// being rejected as a user error.
func TestParse_AuditAllowsEmptyTaskID(t *testing.T) {
	a, err := Parse([]string{"aios", "audit"}, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("`aios audit` should parse with no task id: %v", err)
	}
	if a.Command != CmdAudit {
		t.Errorf("command = %q, want audit", a.Command)
	}
	if a.TaskID != "" {
		t.Errorf("task = %q, want empty", a.TaskID)
	}
}

// TestParse_AuditAcceptsPositionalTaskID covers
// `aios audit <task>` as well as the flag form.
func TestParse_AuditAcceptsPositionalTaskID(t *testing.T) {
	a, err := Parse([]string{"aios", "audit", "task-7"}, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	if a.TaskID != "task-7" {
		t.Errorf("task = %q, want task-7", a.TaskID)
	}
}
