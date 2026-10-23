package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/sadi/aios-system/internal/webui"
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
	// CmdWeb serves the operator UI from the binary.
	// v0.41.0: the UI is embedded via go:embed, so this
	// needs no sidecar assets, no npm, and no network.
	CmdWeb = "web"
	// CmdTools lists the doc-set tools currently
	// registered in the bus. v0.38.0: before this
	// version the edit / git / search / image tools
	// were implemented but never registered, so
	// there was no way to discover the gap short of
	// writing a TaskPlan and reading ToolNotFoundError.
	CmdTools = "tools"
	// CmdApprove and CmdDeny resolve a pending approval over
	// the daemon socket. Without them an operator can only
	// answer a gate at the terminal the run is attached to;
	// these let a second process (a second window, a
	// separate operator tool, a second admin) decide.
	CmdApprove = "approve"
	CmdDeny    = "deny"
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
	WebAddr    string // listen address for `web` (default: webui.DefaultAddr)
	WebOpen    bool   // `web --open`: print the URL and exit without serving
	WebLive    bool   // `run --web`: serve the UI for the duration of the run
	// AssumeYes auto-approves every gated tool call instead of
	// prompting. Off by default: an approval gate exists to put
	// a human in front of a dangerous action, so this must be
	// asked for explicitly and is announced on every use.
	AssumeYes bool
	// SocketExplicit records whether the operator passed
	// --socket themselves. The default socket value is not
	// enough to decide the approval strategy: an operator who
	// names a socket is asking for the daemon, while one who
	// never mentioned it is at a terminal.
	SocketExplicit bool
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
// DefaultLogPath is the audit-log path used when the
// operator passes no --log flag.
//
// v0.40.2: this used to be "./var/aios.log", which was
// misleading twice over. It is a sqlite3 database, not a
// text log, and the BuildDeps fallback in deps.go named a
// DIFFERENT default ("./var/aios.db") for the same file.
// Because the parser always populates LogPath, the deps.go
// fallback could never run, so the two defaults disagreed
// on paper while only the parser's was reachable -- and the
// name an operator saw in `--help` was the wrong one.
//
// The default is relative, so it resolves against the
// working directory like every other default here.
// Explicitly naming it as a constant keeps BuildDeps and
// the parser in agreement and lets tests assert the exact
// path instead of reconstructing it.
const DefaultLogPath = "./var/aios.db"

// Parse turns argv into an *Args. The function writes usage
// and parse errors to w and returns them as errors. An
// equivalent.
func Parse(argv []string, w io.Writer) (*Args, error) {
	if len(argv) == 0 {
		return nil, errors.New("cli: no arguments")
	}
	a := &Args{}
	fs := flag.NewFlagSet("aios", flag.ContinueOnError)
	// Silence the flag package's own printing. It writes a
	// bare error line plus "Usage of aios:" to its output, and
	// we also render an error line plus our own help, so the
	// operator saw every message twice (and `aios --help`
	// printed two different usage blocks). Routing every
	// message through one renderer keeps the output to one
	// coherent block per invocation.
	fs.SetOutput(io.Discard)
	fs.Usage = func() {}
	fs.BoolVar(&a.Debug, "debug", false, "print the full error traceback on failure")
	fs.StringVar(&a.ConfigPath, "config", "./var/config.yaml", "path to the config file")
	fs.StringVar(&a.LogPath, "log", DefaultLogPath, "path to the audit log database")
	fs.StringVar(&a.Workspace, "workspace", "", "tool-bus workspace root; default = dirname of --config")
	fs.StringVar(&a.SocketPath, "socket", DefaultSocketPath(), "approval daemon socket path")
	fs.BoolVar(&a.AssumeYes, "yes", false, "auto-approve gated tool calls instead of prompting (no human sees each request)")
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
			return nil, &CliError{Kind: "user", What: err.Error(), Cause: err}
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
		return nil, &CliError{Kind: "user", What: err.Error(), Cause: err}
	}
	// Record whether the operator named --socket themselves.
	// The flag set distinguishes "left at the default" from
	// "set to this value", and the approval strategy depends
	// on the difference.
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "socket" {
			a.SocketExplicit = true
		}
	})

	// Subcommand-specific flags. We add only the flags the
	// subcommand actually uses; everything else is rejected.
	sub := flag.NewFlagSet(a.Command, flag.ContinueOnError)
	sub.SetOutput(io.Discard)
	sub.Usage = func() {}
	switch a.Command {
	case CmdFix:
		// `fix` takes the objective as positionals
		// (joined in validate) and optional --model and
		// --plan-out overrides.
		sub.StringVar(&a.Model, "model", "", "model id override (default: planner picks)")
		sub.StringVar(&a.PlanOut, "plan-out", "", "path to write the planner's plan YAML; empty = do not persist")
		sub.BoolVar(&a.AssumeYes, "yes", false, "auto-approve gated tool calls instead of prompting")
	case CmdRun:
		sub.StringVar(&a.PlanPath, "plan", "", "path to the plan YAML file")
		// v0.41.1: --web serves the UI from THIS process, so
		// the panels update live while the plan runs. Without
		// it the run is unchanged; `aios web` in another shell
		// still works but only shows history.
		sub.BoolVar(&a.WebLive, "web", false, "also serve the operator UI while the plan runs (see --web-addr)")
		sub.StringVar(&a.WebAddr, "web-addr", "", "listen address for --web (default: "+webui.DefaultAddr+")")
		sub.BoolVar(&a.AssumeYes, "yes", false, "auto-approve gated tool calls instead of prompting")
	case CmdResume:
		sub.StringVar(&a.PlanPath, "plan", "", "path to the original plan YAML file")
		sub.StringVar(&a.TaskID, "task", "", "task id to resume")
		sub.BoolVar(&a.AssumeYes, "yes", false, "auto-approve gated tool calls instead of prompting")
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
	case CmdWeb:
		sub.StringVar(&a.WebAddr, "addr", "", "listen address (default: "+webui.DefaultAddr+"; loopback only)")
		sub.BoolVar(&a.WebOpen, "open", false, "print the URL and exit instead of serving (useful to check the bind)")
	case CmdTools:
		// no flags yet; reserved for future filters
		// (e.g. --json, --package=git).
	case CmdApprove, CmdDeny:
		// No sub-flags: the approval id is positional.
		// The global --socket selects the daemon.
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
	if err := sub.Parse(hoistFlags(rest, valueFlagNames(sub))); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printSubHelp(w, a.Command, sub)
			return a, errHelp
		}
		return nil, &CliError{Kind: "user", What: err.Error(), Cause: err}
	}
	a.Positional = sub.Args()

	// Subcommand-specific validation.
	if err := a.validate(); err != nil {
		return nil, err
	}
	return a, nil
}

func (a *Args) validate() error {
	switch a.Command {
	case CmdFix:
		// `fix` takes the objective as its positionals. We
		// join them with a space so both `aios fix "repair
		// the login page"` and `aios fix repair the login
		// page` produce the same objective.
		//
		// This used to take Positional[0] only, which
		// silently truncated an unquoted multi-word
		// objective to its first word: `aios fix repair the
		// login page` planned on "repair" and reported no
		// error. The comment here claimed the opposite of
		// what the code did.
		if len(a.Positional) == 0 {
			return &CliError{Kind: "user", What: "`fix` requires an objective: aios fix <objective>"}
		}
		a.Objective = strings.Join(a.Positional, " ")
		if strings.TrimSpace(a.Objective) == "" {
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
		CmdWeb: true, CmdApprove: true, CmdDeny: true,
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

// hoistFlags moves flags ahead of positional arguments so
// Go's flag package sees them.
//
// v0.41.2: this fixes a defect that affected EVERY
// subcommand, not just the ones v0.41.0 added. Go's
// flag.Parse stops at the first non-flag argument, so
//
//	aios run plan.yaml --web
//
// left --web sitting in Args() as a positional and never
// set the field: the flag was silently ignored and the
// command ran as if it had not been passed. Every flag on
// every subcommand had to come before its positional, which
// is the opposite of what the usage text advertises
// ("aios [--debug] ... <subcommand> [flags] [args...]")
// and of what an operator types.
//
// The reorder is stable within each group, so
// "run --web plan.yaml --format json" keeps the flags in
// order and the positionals in order.
//
// Only the subcommand's own argument list is reordered. The
// global flags before the subcommand name are parsed by the
// outer FlagSet and are untouched.
//
// A bare "--" is respected: everything after it is
// positional, which is how a caller passes a value that
// begins with a dash.
//
// valueFlags names the flags that take an argument, so the
// separated form "--plan p.yaml" keeps p.yaml attached to
// its flag. Without that, hoisting would split the pair and
// leave p.yaml as a positional: "resume --plan p.yaml
// --task t1" would set PlanPath to "--task".
func hoistFlags(args []string, valueFlags map[string]bool) []string {
	var flags, positional []string
	endOfFlags := false
	// expectValue is set when the previous iteration was a
	// flag that takes its value in the next argument.
	expectValue := false
	for _, arg := range args {
		if endOfFlags {
			positional = append(positional, arg)
			continue
		}
		if arg == "--" {
			endOfFlags = true
			positional = append(positional, arg)
			continue
		}
		if expectValue {
			expectValue = false
			flags = append(flags, arg)
			continue
		}
		if len(arg) > 1 && strings.HasPrefix(arg, "-") {
			flags = append(flags, arg)
			// "--plan=p.yaml" carries its own value; only the
			// bare "--plan" form consumes the next argument.
			if !strings.Contains(arg, "=") && valueFlags[strings.TrimLeft(arg, "-")] {
				expectValue = true
			}
			continue
		}
		positional = append(positional, arg)
	}
	if len(flags) == 0 {
		return args
	}
	out := make([]string, 0, len(args))
	out = append(out, flags...)
	out = append(out, positional...)
	return out
}

// valueFlagNames returns the names of the flags in fs that
// require a value.
//
// It reads the declared flags rather than a hand-maintained
// list, so a flag added later is hoisted correctly without
// anyone remembering to register its arity here.
func valueFlagNames(fs *flag.FlagSet) map[string]bool {
	out := map[string]bool{}
	fs.VisitAll(func(f *flag.Flag) {
		// A boolean flag ("true" / "false") can be written
		// without a value, so it must never swallow the next
		// argument. Everything else in this CLI is a string or
		// int option that takes one.
		if bf, ok := f.Value.(interface{ IsBoolFlag() bool }); ok && bf.IsBoolFlag() {
			return
		}
		out[f.Name] = true
	})
	return out
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

// printHelp writes the help text to w.
//
// The text is pure ASCII on purpose. The em dash that used to
// separate the name from the description rendered as mojibake
// ("aios ΓÇö aios-system CLI") on a Windows console using the
// default OEM code page, and the product ships on Windows first,
// so the punctuation has to survive a console that is not
// UTF-8.
const helpBody = `aios - aios-system CLI

Usage:
  aios [--debug] [--config=path] [--log=path] <subcommand> [flags] [args...]

Subcommands:
  fix <objective>        Plan + run: turn the objective into a plan and run it.
  run <plan>             Run a plan from a YAML file (--web to also serve the UI).
  resume <task>          Resume a paused task.
  cancel <task>          Cancel a paused task.
  report <task>          Print a task report (json, markdown, html).
  audit [task]           Show the audit log; with --tail, follow.
  tools                  List the tools the agent can dispatch.
  init                   Initialise the workspace (./var/).
  web [--addr=host:port]  Serve the operator UI (embedded; loopback only).
  daemon start           Start the approval daemon (foreground by default).
  daemon stop            Stop the approval daemon.
  approve <id>           Grant a pending approval (over the daemon socket).
  deny <id>              Deny a pending approval (over the daemon socket).

Global flags:
  --debug                Print the full error traceback on failure.
  --config=path          Config file (default: ./var/config.yaml).
  --log=path             Audit log database (default: ./var/aios.db).
  --workspace=path       Tool-bus workspace root (default: dirname of --config).
  --socket=path          Approval daemon socket.
  --pidfile=path         Daemon pid file (default: ./var/aios-daemon.pid).

Run flags (run / fix / resume):
  --yes                  Auto-approve gated tool calls instead of prompting.
                          Every auto-approval is announced; no human decides.

Approvals:
  A policy rule with action "ask" pauses the run and shows what it
  wants to authorize. Answer y at the prompt, or run with --yes, or
  leave a daemon running and decide from another terminal with
  "aios approve <id>" / "aios deny <id>".
  A grant covers exactly one call, in one run. It is not remembered
  for the next run, and it does not carry over to another task,
  action, or target.

Run "aios <subcommand> --help" for the flags of one subcommand.

Exit codes:
  0  success
  1  user error
  2  system error
`

func printHelp(w io.Writer) {
	fmt.Fprint(w, HelpText())
}

// subUsage names the positional argument each subcommand
// expects, so its help can show a real invocation.
var subUsage = map[string]string{
	CmdFix:    "aios fix <objective> [flags]",
	CmdRun:    "aios run <plan.yaml> [flags]",
	CmdResume: "aios resume <task-id> --plan <plan.yaml>",
	CmdCancel: "aios cancel <task-id>",
	CmdReport: "aios report <task-id> [--format=json|markdown|html]",
	CmdAudit:  "aios audit [task-id] [--tail]",
	CmdInit:   "aios init [--config=./var/config.yaml]",
	CmdWeb:    "aios web [--addr=127.0.0.1:7755] [--open]",
	CmdTools:  "aios tools",
	CmdDaemon: "aios daemon start|stop [--socket=path] [--pidfile=path]",
}

// printSubHelp writes help for one subcommand: its usage
// line, its own flags, and a pointer back to the global help.
//
// Previously `aios run --help` printed the global help, so an
// operator asking about `--web-addr` had to read a block that
// did not mention it while the flags they asked about scrolled
// past in the flag package's own dump.
func printSubHelp(w io.Writer, command string, sub *flag.FlagSet) {
	usage, ok := subUsage[command]
	if !ok {
		printHelp(w)
		return
	}
	fmt.Fprintf(w, "Usage:\n  %s\n\nFlags:\n", usage)
	var flags []*flag.Flag
	sub.VisitAll(func(f *flag.Flag) {
		flags = append(flags, f)
	})
	if len(flags) == 0 {
		fmt.Fprint(w, "  (none)\n")
	}
	for _, f := range flags {
		def := ""
		if f.DefValue != "" && f.DefValue != "false" {
			def = fmt.Sprintf(" (default %q)", f.DefValue)
		}
		fmt.Fprintf(w, "  --%s%s\n        %s\n", f.Name, def, f.Usage)
	}
	fmt.Fprint(w, "\nRun \"aios --help\" for the global flags and the subcommand list.\n")
}
