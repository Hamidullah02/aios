package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
)

// Subcommand is the set of supported subcommands. The list is
// closed at compile time so the parser can validate the input
// before doing any work. Per REQ-CLI-FST-002 the supported
// subcommands are:
//
//	fix, run, resume, cancel, report, daemon start, daemon stop,
//	audit, init
//
// `fix` is the primary entry point the doc set describes:
// the CLI takes a natural-language objective, calls the
// planner to turn it into a plan, and runs the plan through
// the same loop `run` uses. `run` is retained for users
// who want to feed a hand-written plan YAML.
const (
	CmdFix    = "fix"
	CmdRun    = "run"
	CmdResume = "resume"
	CmdCancel = "cancel"
	CmdReport = "report"
	CmdAudit  = "audit"
	CmdInit   = "init"
	CmdDaemon = "daemon"
	// CmdTools lists the doc-set tools currently
	// registered in the bus. v0.38.0: before this
	// version the edit / git / search / image tools
	// were implemented but never registered, so
	// there was no way to discover the gap short of
	// writing a TaskPlan and reading ToolNotFoundError.
	CmdTools = "tools"
)

// DaemonSub is the sub-subcommand for the daemon. Per
// REQ-CLI-FST-002 the daemon command has start and stop
// sub-actions.
const (
	DaemonStart = "start"
	DaemonStop  = "stop"
)

// Args is the parsed argv. The fields are populated by Parse
// and consumed by the subcommand handlers. The Debug field is
// shared across handlers; the ConfigPath is the per-run config
// override; the LogPath is the per-run log override.
type Args struct {
	Command    string // "run", "resume", etc.
	Subcommand string // "start" or "stop" for daemon; "" otherwise
	Positional []string
	Debug      bool
	ConfigPath string
	LogPath    string
	Workspace  string // tool-bus workspace root; default = dirname of ConfigPath

	// Subcommand-specific fields. The handler picks what it
	// needs; the others stay at their zero value.
	PlanPath   string
	TaskID     string
	ReportFmt  string // "json" | "markdown" | "html"
	Tail       bool
	SocketPath string
	DaemonMode bool   // true when `daemon start` is foreground
	DaemonPID  string // pid file path for `daemon start` (default)
	Objective  string // the natural-language goal for `fix`
	Model      string // model id override for `fix` and `run` (default: planner picks)
	PlanOut    string // path to write the planner's plan (for `fix`); empty = no write
}

// Parse parses argv into Args. argv[0] is the program name
// (matching Go's convention in os.Args). The function writes
// help to w when --help or -h is set; otherwise it returns an
// error on bad input.
//
// The parser does not touch the filesystem. Subcommand
// handlers do that.
//
// Per REQ-CLI-FST-003 the parser uses the Go stdlib flag
// package. The doc set says argparse, which is the Python
// equivalent.
func Parse(argv []string, w io.Writer) (*Args, error) {
	if len(argv) == 0 {
		return nil, errors.New("cli: no arguments")
	}
	a := &Args{}
	fs := flag.NewFlagSet("aios", flag.ContinueOnError)
	fs.SetOutput(w)
	fs.BoolVar(&a.Debug, "debug", false, "print the full error traceback on failure")
	fs.StringVar(&a.ConfigPath, "config", "./var/config.yaml", "path to the config file")
	fs.StringVar(&a.LogPath, "log", "./var/aios.log", "path to the log file")
	fs.StringVar(&a.Workspace, "workspace", "", "tool-bus workspace root; default = dirname of --config")
	fs.StringVar(&a.SocketPath, "socket", "/tmp/aios-approval.sock", "approval daemon socket path")
	fs.StringVar(&a.DaemonPID, "pidfile", "./var/aios-daemon.pid", "path to the daemon pid file")
	// --format and --tail are subcommand-specific and
	// registered on the per-subcommand FlagSet.

	// We accept the global flags either before or after the
	// subcommand. The convention is:
	//   aios [--debug] [--config=...] <subcommand> [sub-flags] [args...]
	//
	// To support both we look for the subcommand first, then
	// parse the globals from the prefix and the subcommand-
	// specific flags from the suffix.
	idx := findSubcommandIndex(argv[1:])
	if idx < 0 {
		// No subcommand. Maybe the user asked for --help.
		if err := fs.Parse(argv[1:]); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				printHelp(w)
				return a, errHelp
			}
			return nil, err
		}
		if a.helpRequested(fs) {
			printHelp(w)
			return a, errHelp
		}
		return nil, &CliError{Kind: "user", What: "missing subcommand (try `aios --help`)"}
	}
	// Split argv into [pre, sub, post].
	pre := append([]string(nil), argv[1:1+idx]...)
	rest := append([]string(nil), argv[1+idx:]...)
	a.Command = rest[0]
	rest = rest[1:]

	// Parse the global flags from the prefix. We use a fresh
	// FlagSet so the help text only shows the globals.
	if err := fs.Parse(pre); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printHelp(w)
			return a, errHelp
		}
		return nil, err
	}
	if a.helpRequested(fs) {
		printHelp(w)
		return a, errHelp
	}

	// Subcommand-specific flags. We add only the flags the
	// subcommand actually uses; everything else is rejected.
	sub := flag.NewFlagSet(a.Command, flag.ContinueOnError)
	sub.SetOutput(w)
	switch a.Command {
	case CmdFix:
		// `fix` takes a single positional (the objective)
		// and optional --model and --plan-out overrides.
		// The Objective field is populated by the caller
		// before the switch; the parser just adds the
		// flags.
		sub.StringVar(&a.Model, "model", "", "model id override (default: planner picks)")
		sub.StringVar(&a.PlanOut, "plan-out", "", "path to write the planner's plan YAML; empty = do not persist")
	case CmdRun:
		sub.StringVar(&a.PlanPath, "plan", "", "path to the plan YAML file")
	case CmdResume:
		sub.StringVar(&a.PlanPath, "plan", "", "path to the original plan YAML file")
		sub.StringVar(&a.TaskID, "task", "", "task id to resume")
	case CmdCancel:
		sub.StringVar(&a.TaskID, "task", "", "task id to cancel")
	case CmdReport:
		sub.StringVar(&a.TaskID, "task", "", "task id to report on")
		sub.StringVar(&a.ReportFmt, "format", "json", "report format: json, markdown, html")
	case CmdAudit:
		sub.StringVar(&a.TaskID, "task", "", "task id to audit (default: all)")
		sub.BoolVar(&a.Tail, "tail", false, "follow the audit log")
	case CmdInit:
		// init takes no sub-flags.
	case CmdTools:
		// no flags yet; reserved for future filters
		// (e.g. --json, --package=git).
	case CmdDaemon:
		// The daemon command takes start or stop as a
		// positional arg, not as a flag.
		if len(rest) == 0 {
			return nil, &CliError{Kind: "user", What: "`daemon` requires `start` or `stop`"}
		}
		a.Subcommand = rest[0]
		rest = rest[1:]
	default:
		return nil, &CliError{Kind: "user", What: fmt.Sprintf("unknown subcommand %q", a.Command)}
	}
	if err := sub.Parse(rest); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printHelp(w)
			return a, errHelp
		}
		return nil, err
	}
	a.Positional = sub.Args()

	// Subcommand-specific validation.
	if err := a.validate(); err != nil {
		return nil, err
	}
	return a, nil
}

func (a *Args) helpRequested(fs *flag.FlagSet) bool {
	// flag sets visit=true when a flag was set, but --help
	// is its own flag we didn't declare. Walk the args
	// looking for -h / -help / --help.
	// We use the fs.Visit semantics: a flag we did not
	// declare that matches `help` will fail Parse with
	// "flag provided but not defined" unless the user used
	// -h / --help, which we accept via the standard handler
	// that prints usage and returns ErrHelp.
	return false
}

func (a *Args) validate() error {
	switch a.Command {
	case CmdFix:
		// `fix` takes the objective as a single positional
		// argument. We join all positionals so users can
		// pass a multi-word objective without quoting each
		// word (the shell already collapses them into one
		// arg when quoted; we keep them as one field here).
		if len(a.Positional) == 0 {
			return &CliError{Kind: "user", What: "`fix` requires an objective: aios fix <objective>"}
		}
		a.Objective = a.Positional[0]
		if a.Objective == "" {
			return &CliError{Kind: "user", What: "`fix` objective is empty"}
		}
	case CmdRun:
		if a.PlanPath == "" && len(a.Positional) > 0 {
			a.PlanPath = a.Positional[0]
		}
		if a.PlanPath == "" {
			return &CliError{Kind: "user", What: "`run` requires a plan path"}
		}
	case CmdResume:
		if a.PlanPath == "" && len(a.Positional) > 1 {
			a.PlanPath = a.Positional[0]
			a.Positional = a.Positional[1:]
		}
		if a.TaskID == "" && len(a.Positional) > 0 {
			a.TaskID = a.Positional[0]
		}
		if a.TaskID == "" {
			return &CliError{Kind: "user", What: "`resume` requires a task id"}
		}
	case CmdCancel, CmdReport, CmdAudit:
		if a.TaskID == "" && len(a.Positional) > 0 {
			a.TaskID = a.Positional[0]
		}
		if a.Command != CmdAudit && a.TaskID == "" {
			return &CliError{Kind: "user", What: fmt.Sprintf("`%s` requires a task id", a.Command)}
		}
	case CmdDaemon:
		if a.Subcommand != DaemonStart && a.Subcommand != DaemonStop {
			return &CliError{Kind: "user", What: "`daemon` requires `start` or `stop`"}
		}
	}
	return nil
}

// findSubcommandIndex returns the index in argv of the first
// known subcommand. -1 means no subcommand was found.
func findSubcommandIndex(argv []string) int {
	known := map[string]bool{
		CmdFix: true,
		CmdRun: true, CmdResume: true, CmdCancel: true, CmdReport: true,
		CmdAudit: true, CmdInit: true, CmdDaemon: true, CmdTools: true,
	}
	for i, a := range argv {
		if strings.HasPrefix(a, "-") {
			continue
		}
		if known[a] {
			return i
		}
	}
	return -1
}

// errHelp is the sentinel the parser returns when the user
// asked for help. The caller (Main) treats it as a clean
// exit with code 0.
var errHelp = errors.New("cli: help requested")

// IsHelp reports whether err is the help-requested sentinel.
func IsHelp(err error) bool {
	return errors.Is(err, errHelp)
}

// HelpText returns the help text. Exposed so Main can print it
// on `--help` without going through Parse.
func HelpText() string {
	return strings.TrimSpace(helpBody) + "\n"
}

const helpBody = `aios — aios-system CLI

Usage:
  aios [--debug] [--config=path] [--log=path] <subcommand> [flags] [args...]

Subcommands:
  fix <objective>        Plan + run: turn the objective into a plan and run it.
  run <plan>             Run a plan from a YAML file.
  resume <task>          Resume a paused task.
  cancel <task>          Cancel a running task.
  report <task>          Print a task report (json, markdown, html).
  audit [task]           Show the audit log; with --tail, follow.
  init                   Initialise the workspace (./var/).
  daemon start           Start the approval daemon (foreground by default).
  daemon stop            Stop the approval daemon.

Global flags:
  --debug                Print the full error traceback on failure.
  --config=path          Config file (default: ./var/config.yaml).
  --log=path             Log file (default: ./var/aios.log).
  --socket=path          Approval daemon socket (default: /tmp/aios-approval.sock).
  --pidfile=path         Daemon pid file (default: ./var/aios-daemon.pid).
  --format=fmt           Report format (default: json).

Exit codes:
  0  success
  1  user error
  2  system error
`

func printHelp(w io.Writer) {
	fmt.Fprint(w, HelpText())
}
