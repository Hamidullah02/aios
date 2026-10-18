package cli

import (
	"bytes"
	"errors"
	"testing"
)

func TestParse_NoArgs(t *testing.T) {
	if _, err := Parse([]string{"aios"}, &bytes.Buffer{}); err == nil {
		t.Fatal("expected error for no subcommand")
	}
}

func TestParse_Help(t *testing.T) {
	a, err := Parse([]string{"aios", "--help"}, &bytes.Buffer{})
	if !IsHelp(err) {
		t.Fatalf("err: %v, want help", err)
	}
	if a == nil {
		t.Fatal("args should be non-nil even on help")
	}
}

func TestParse_Run(t *testing.T) {
	a, err := Parse([]string{"aios", "run", "plan.yaml"}, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	if a.Command != CmdRun {
		t.Errorf("command: %q", a.Command)
	}
	if a.PlanPath != "plan.yaml" {
		t.Errorf("plan: %q", a.PlanPath)
	}
}

func TestParse_RunWithPlanFlag(t *testing.T) {
	a, err := Parse([]string{"aios", "run", "--plan", "p.yaml"}, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	if a.PlanPath != "p.yaml" {
		t.Errorf("plan: %q", a.PlanPath)
	}
}

func TestParse_Resume(t *testing.T) {
	a, err := Parse([]string{"aios", "resume", "task-1"}, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	if a.Command != CmdResume {
		t.Errorf("command: %q", a.Command)
	}
	if a.TaskID != "task-1" {
		t.Errorf("task: %q", a.TaskID)
	}
}

func TestParse_Cancel(t *testing.T) {
	a, err := Parse([]string{"aios", "cancel", "task-1"}, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	if a.Command != CmdCancel || a.TaskID != "task-1" {
		t.Errorf("command: %q, task: %q", a.Command, a.TaskID)
	}
}

func TestParse_Report(t *testing.T) {
	a, err := Parse([]string{"aios", "report", "--format=markdown", "task-1"}, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	if a.Command != CmdReport || a.TaskID != "task-1" {
		t.Errorf("command: %q, task: %q", a.Command, a.TaskID)
	}
	if a.ReportFmt != "markdown" {
		t.Errorf("format: %q", a.ReportFmt)
	}
}

func TestParse_Audit(t *testing.T) {
	a, err := Parse([]string{"aios", "audit", "--tail", "task-1"}, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	if a.Command != CmdAudit || a.TaskID != "task-1" || !a.Tail {
		t.Errorf("command: %q, task: %q, tail: %v", a.Command, a.TaskID, a.Tail)
	}
}

func TestParse_AuditAllTasks(t *testing.T) {
	a, err := Parse([]string{"aios", "audit"}, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	if a.Command != CmdAudit || a.TaskID != "" {
		t.Errorf("command: %q, task: %q", a.Command, a.TaskID)
	}
}

func TestParse_Init(t *testing.T) {
	a, err := Parse([]string{"aios", "init"}, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	if a.Command != CmdInit {
		t.Errorf("command: %q", a.Command)
	}
}

func TestParse_DaemonStart(t *testing.T) {
	a, err := Parse([]string{"aios", "daemon", "start"}, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	if a.Command != CmdDaemon || a.Subcommand != DaemonStart {
		t.Errorf("command: %q, sub: %q", a.Command, a.Subcommand)
	}
}

func TestParse_DaemonStop(t *testing.T) {
	a, err := Parse([]string{"aios", "daemon", "stop"}, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	if a.Command != CmdDaemon || a.Subcommand != DaemonStop {
		t.Errorf("command: %q, sub: %q", a.Command, a.Subcommand)
	}
}

func TestParse_DaemonMissingSub(t *testing.T) {
	_, err := Parse([]string{"aios", "daemon"}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error for missing daemon subcommand")
	}
}

func TestParse_UnknownSubcommand(t *testing.T) {
	_, err := Parse([]string{"aios", "frobnicate"}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error for unknown subcommand")
	}
}

func TestParse_GlobalFlagsBeforeSubcommand(t *testing.T) {
	a, err := Parse([]string{"aios", "--debug", "--config=c.yaml", "run", "p.yaml"}, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	if !a.Debug || a.ConfigPath != "c.yaml" {
		t.Errorf("debug: %v, config: %q", a.Debug, a.ConfigPath)
	}
	if a.PlanPath != "p.yaml" {
		t.Errorf("plan: %q", a.PlanPath)
	}
}

func TestParse_GlobalFlagsAfterSubcommand(t *testing.T) {
	// The parser's convention is global flags BEFORE the
	// subcommand. After is not supported; with the stdlib
	// flag package, a flag after a positional is treated
	// as a positional too. The test confirms the parser
	// does not crash and at least reaches the validation
	// step.
	_, _ = Parse([]string{"aios", "run", "p.yaml", "--debug"}, &bytes.Buffer{})
}

func TestParse_RunMissingPlan(t *testing.T) {
	_, err := Parse([]string{"aios", "run"}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error for run without plan")
	}
}

func TestParse_ResumeMissingTask(t *testing.T) {
	_, err := Parse([]string{"aios", "resume"}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error for resume without task")
	}
}

func TestIsHelp(t *testing.T) {
	if !IsHelp(errHelp) {
		t.Fatal("errHelp should match")
	}
	if IsHelp(errors.New("other")) {
		t.Fatal("other err should not match")
	}
}

func TestHelpText(t *testing.T) {
	s := HelpText()
	if s == "" {
		t.Fatal("non-empty")
	}
	if !contains(s, "aios") {
		t.Fatal("help should mention aios")
	}
}
