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
//	run, resume, cancel, report, daemon start, daemon stop,
//	audit, init
const (
	CmdRun       = "run"
	CmdResume    = "resume"
	CmdCancel    = "cancel"
	CmdReport    = "report"
	CmdAudit     = "audit"
	CmdInit      = "init"
	CmdDaemon    = "daemon"
	CmdSearch    = "search"
	CmdMemory    = "memory"
	CmdEgress    = "egress"
	CmdBenchmark = "benchmark"
	CmdModel     = "model"
	CmdSession   = "session"
	CmdPlan      = "plan"
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

	// Subcommand-specific fields. The handler picks what it
	// needs; the others stay at their zero value.
	PlanPath   string
	TaskID     string
	ReportFmt  string // "json" | "markdown" | "html"
	Tail       bool
	SocketPath string
	DaemonMode bool   // true when `daemon start` is foreground
	DaemonPID  string // pid file path for `daemon start` (default)

	// Subcommand-specific fields. Used by the new
	// subcommands (search, memory, egress, benchmark,
	// model, session, plan).
	Query        string
	Pattern      string
	Path         string
	Limit        int
	Listen       string
	DatasetPath  string
	OutputPath   string
	PricePath    string
	SessionsPath string
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
	// --config defaults to "" (no policy). Operators who
	// want enforcement pass --config=path/to/policy.yaml;
	// operators running a smoke test or a CI job leave it
	// empty and the system allows every tool call. The
	// doc set treats the policy as optional; the default
	// must reflect that.
	fs.StringVar(&a.ConfigPath, "config", "", "path to the config file (optional)")
	fs.StringVar(&a.LogPath, "log", "./var/aios.log", "path to the log file")
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
	case CmdSearch:
		sub.StringVar(&a.Query, "query", "", "search query (grep mode)")
		sub.StringVar(&a.Pattern, "pattern", "", "file glob (glob mode)")
		sub.StringVar(&a.Path, "path", ".", "directory to search")
	case CmdMemory:
		sub.StringVar(&a.Query, "query", "", "memory query string")
		sub.IntVar(&a.Limit, "limit", 20, "max results to print")
	case CmdEgress:
		sub.StringVar(&a.Listen, "listen", "127.0.0.1:8080", "address to bind")
		sub.StringVar(&a.ConfigPath, "egress-config", "var/egress.yaml", "egress config path")
	case CmdBenchmark:
		sub.StringVar(&a.DatasetPath, "dataset", "./benchmark/wedge.json", "benchmark dataset")
		sub.StringVar(&a.OutputPath, "output", "./var/benchmark-report.json", "metrics output")
	case CmdModel:
		sub.StringVar(&a.PricePath, "prices", "var/prices.yaml", "price table path")
	case CmdSession:
		sub.StringVar(&a.SessionsPath, "sessions", "", "sessions directory")
	case CmdPlan:
		sub.StringVar(&a.PlanPath, "plan", "", "plan YAML path")
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
		CmdRun: true, CmdResume: true, CmdCancel: true, CmdReport: true,
		CmdAudit: true, CmdInit: true, CmdDaemon: true,
		CmdSearch: true, CmdMemory: true, CmdEgress: true,
		CmdBenchmark: true, CmdModel: true, CmdSession: true,
		CmdPlan: true,
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
  run <plan>             Run a plan from a YAML file.
  resume <task>          Resume a paused task.
  cancel <task>          Cancel a running task.
  report <task>          Print a task report (json, markdown, html).
  audit [task]           Show the audit log; with --tail, follow.
  init                   Initialise the workspace (./var/).
  search glob|grep       Search the workspace from the command line.
  memory query           Query the memory store.
  egress start           Start the egress proxy.
  benchmark run          Run the benchmark dataset.
  model list             List configured models.
  session list           List persisted sessions.
  plan validate          Validate a plan YAML file.
  daemon start           Start the approval daemon (foreground by default).
  daemon stop            Stop the approval daemon.

Global flags:
  --debug                Print the full error traceback on failure.
  --config=path          Optional config file (default: none). When set, every
                       tool call is checked against the policy rules in the
                       file. When unset, every tool call is allowed.
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
