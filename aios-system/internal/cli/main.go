package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
)

// Main is the entry point the production binary calls. The
// function returns an exit code: 0 on success, 1 on user
// error, 2 on system error (per REQ-CLI-ERR-003).
//
// argv follows Go's os.Args convention: argv[0] is the
// program name, argv[1:] are the user arguments. The
// function accepts a slice so tests can drive it without
// touching the real process args.
//
// stdout and stderr are the I/O targets. The defaults are
// os.Stdout / os.Stderr; tests pass a bytes.Buffer.
//
// The Deps field carries the production collaborators
// (audit log, policy, etc.). Tests pass a fully-stubbed
// Deps; the production binary calls DefaultDeps() to build
// the real one.
func Main(argv []string, stdout, stderr io.Writer) int {
	if stdout == nil {
		stdout = os.Stdout
	}
	if stderr == nil {
		stderr = os.Stderr
	}
	if len(argv) == 0 {
		fmt.Fprintln(stderr, "cli: no arguments")
		return ExitUserError
	}
	args, err := Parse(argv, stderr)
	if err != nil {
		if IsHelp(err) {
			// Help was printed by Parse; exit clean.
			return ExitOK
		}
		fmt.Fprint(stderr, DisplayError(err, false))
		return ExitCodeFor(err)
	}
	deps, err := BuildDeps(args)
	if err != nil {
		fmt.Fprint(stderr, DisplayError(err, args.Debug))
		return ExitCodeFor(err)
	}
	defer func() {
		if err := deps.Close(); err != nil {
			fmt.Fprintf(stderr, "cli: close deps: %v\n", err)
		}
	}()
	return RunCommand(args, deps, stdout, stderr)
}

// RunCommand dispatches on args.Command. The function is
// exported so the tests can drive a single subcommand
// without going through Main's argv parsing.
func RunCommand(args *Args, deps *Deps, stdout, stderr io.Writer) int {
	if args == nil {
		fmt.Fprintln(stderr, "cli: no arguments")
		return ExitUserError
	}
	ctx := context.Background()
	switch args.Command {
	case CmdInit:
		return runInit(args, deps, stdout, stderr)
	case CmdFix:
		return runFix(ctx, args, deps, stdout, stderr)
	case CmdRun:
		return runRun(ctx, args, deps, stdout, stderr)
	case CmdResume:
		return runResume(ctx, args, deps, stdout, stderr)
	case CmdCancel:
		return runCancel(args, deps, stderr)
	case CmdReport:
		return runReport(args, deps, stdout, stderr)
	case CmdAudit:
		return runAudit(args, deps, stdout, stderr)
	case CmdDaemon:
		return runDaemon(ctx, args, deps, stderr)
	case CmdTools:
		return runTools(args, deps, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "cli: unknown subcommand %q\n", args.Command)
		return ExitUserError
	}
}

func runInit(args *Args, deps *Deps, stdout, stderr io.Writer) int {
	workspace := args.ConfigPath
	// The init subcommand scaffolds the directory named
	// in --config (treating it as a workspace). The
	// default is "./var/config.yaml"; the workspace
	// directory is the parent of that.
	if workspace == "" {
		workspace = "./var/config.yaml"
	}
	dir := filepath.Dir(workspace)
	created, existed, err := Init(dir)
	if err != nil {
		fmt.Fprint(stderr, DisplayError(err, args.Debug))
		return ExitCodeFor(err)
	}
	PrintInitReport(stdout, created, existed)
	return ExitOK
}

func runRun(ctx context.Context, args *Args, deps *Deps, stdout, stderr io.Writer) int {
	report, err := Run(ctx, args, RunDeps{
		Audit:        deps.Audit,
		Policy:       deps.Policy,
		Resume:       deps.Resume,
		Worker:       deps.Worker,
		Verifier:     deps.Verifier,
		Budget:       deps.Budget,
		Approval:     deps.Approval,
		ReportWriter: nil,
		Stdout:       stdout,
		Stderr:       stderr,
	})
	if err != nil {
		fmt.Fprint(stderr, DisplayError(err, args.Debug))
		return ExitCodeFor(err)
	}
	PrintReport(stdout, report)
	return ExitOK
}

func runResume(ctx context.Context, args *Args, deps *Deps, stdout, stderr io.Writer) int {
	report, err := Resume(ctx, args, ResumeDeps{
		Audit:   deps.Audit,
		Resume:  deps.Resume,
		Worker:  deps.Worker,
		Verify:  deps.Verifier,
		Budget:  deps.Budget,
		Approve: deps.Approval,
	})
	if err != nil {
		fmt.Fprint(stderr, DisplayError(err, args.Debug))
		return ExitCodeFor(err)
	}
	PrintReport(stdout, report)
	return ExitOK
}

func runCancel(args *Args, deps *Deps, stderr io.Writer) int {
	if err := Cancel(args.TaskID, CancelDeps{Resume: deps.Resume}); err != nil {
		fmt.Fprint(stderr, DisplayError(err, args.Debug))
		return ExitCodeFor(err)
	}
	return ExitOK
}

func runReport(args *Args, deps *Deps, stdout, stderr io.Writer) int {
	if err := Report(args.TaskID, ReportDeps{
		Audit:  deps.Audit,
		Format: args.ReportFmt,
		Out:    stdout,
	}); err != nil {
		fmt.Fprint(stderr, DisplayError(err, args.Debug))
		return ExitCodeFor(err)
	}
	return ExitOK
}

func runAudit(args *Args, deps *Deps, stdout, stderr io.Writer) int {
	if err := Audit(args.TaskID, AuditDeps{
		Audit: deps.Audit,
		Out:   stdout,
		Tail:  args.Tail,
	}); err != nil {
		fmt.Fprint(stderr, DisplayError(err, args.Debug))
		return ExitCodeFor(err)
	}
	return ExitOK
}

// runTools lists every tool currently registered in the
// bus. v0.38.0.
//
// The command reports the doc-set tool set the agent can
// actually dispatch, which is the first thing to check
// when a plan names a tool and gets ToolNotFoundError.
// The list is sorted and one name per line so it pipes
// into grep / comm without parsing.
func runTools(args *Args, deps *Deps, stdout, stderr io.Writer) int {
	// The tool set does not depend on the policy or the
	// audit log, so we build it directly rather than
	// going through BuildDeps. That keeps `aios tools`
	// working on a workspace with no config.yaml and no
	// audit.db, which is exactly the situation where a
	// user is trying to figure out what went wrong.
	names := make([]string, 0)
	// The doc-set set that needs no construction.
	for name := range wireDocSetTools() {
		names = append(names, name)
	}
	// The memory adapters (v0.39.0). We pass a nil
	// Memory because the listing only needs the tool
	// names, and the tools themselves fail cleanly when
	// the memory is not wired.
	for name := range wireMemoryTools(nil, nil) {
		names = append(names, name)
	}
	// The minimum five from wireAgentLoop plus the
	// verify tools that always wire.
	for name := range wireVerifyTools(nil, nil) {
		names = append(names, name)
	}
	names = append(names,
		"read_file", "search_files", "write_file", "run_shell", "ask_user",
	)
	sort.Strings(names)
	for _, name := range names {
		fmt.Fprintln(stdout, name)
	}
	_ = stderr
	return ExitOK
}

// runFix is the entry point the doc set describes: it takes
// a natural-language objective, calls the planner to turn
// it into a plan, and runs the plan through the same loop
// `run` uses. The chain is:
//
//	objective -> planner.Plan -> TaskPlan -> runWithApprovals
//	         -> BusWorker -> tools.Bus -> policy -> tool -> audit
//	         -> CommandVerifier
//
// This is the loop the v0.31.0 review demanded. Every
// component is the production implementation, not a stub:
// the planner calls a real model (via the router); the
// bus dispatches to a real tool; the verifier inspects
// the result and fails the plan on an empty Output or
// nil Data.
//
// The function fails closed: a missing planner (the
// router could not be wired), a model call error
// (e.g. OPENAI_API_KEY not set), or a verifier reject
// all return a non-zero exit code and a clear error
// message on stderr.
func runFix(ctx context.Context, args *Args, deps *Deps, stdout, stderr io.Writer) int {
	if args.Objective == "" {
		fmt.Fprintln(stderr, "cli: `fix` requires an objective")
		return ExitUserError
	}
	if deps.Planner == nil {
		fmt.Fprintln(stderr, "cli: `fix`: planner is not configured (router or planner init failed)")
		return ExitSystemError
	}
	// The planner reads the model's choice of model id
	// (or the operator's --model override) and calls
	// the model once or twice (initial + maybe one
	// repair). The result is a verified TaskPlan.
	fmt.Fprintf(stdout, "Planning: %s\n", args.Objective)
	plan, err := deps.Planner.Plan(ctx, args.Objective, nil)
	if err != nil {
		fmt.Fprint(stderr, DisplayError(err, args.Debug))
		return ExitCodeFor(err)
	}
	if plan == nil {
		fmt.Fprintln(stderr, "cli: `fix`: planner returned a nil plan")
		return ExitSystemError
	}
	fmt.Fprintf(stdout, "Plan:     %s (tasks=%d)\n", plan.ID, len(plan.Tasks))
	// Persist the plan when --plan-out is set. The file
	// is loadable by `aios run <path>` so a user can
	// re-run the planner's plan after editing it. The
	// write happens BEFORE the loop runs so a write
	// error stops the run (we do not want to claim
	// success and silently drop the plan).
	if args.PlanOut != "" {
		if err := WritePlanYAML(plan, args.PlanOut); err != nil {
			fmt.Fprint(stderr, DisplayError(err, args.Debug))
			return ExitCodeFor(err)
		}
		fmt.Fprintf(stdout, "Wrote:    %s\n", args.PlanOut)
	}
	// Run the plan through the same loop `run` uses.
	// This is the part the v0.32.0 commit landed: the
	// BusWorker dispatches to the bus, the bus consults
	// the policy gateway, runs the tool, writes the
	// audit row, and the verifier inspects the result.
	report, err := runWithApprovals(ctx, plan, nil, ResumeDeps{
		Audit:   deps.Audit,
		Resume:  deps.Resume,
		Worker:  deps.Worker,
		Verify:  deps.Verifier,
		Budget:  deps.Budget,
		Approve: deps.Approval,
	}, "fix")
	if err != nil {
		fmt.Fprint(stderr, DisplayError(err, args.Debug))
		return ExitCodeFor(err)
	}
	PrintReport(stdout, report)
	return ExitOK
}

func runDaemon(ctx context.Context, args *Args, deps *Deps, stderr io.Writer) int {
	switch args.Subcommand {
	case DaemonStart:
		if err := RunDaemonStart(ctx, DaemonConfig{
			SocketPath: args.SocketPath,
			PidPath:    args.DaemonPID,
			Approvals:  deps.Approvals,
			Audit:      deps.Audit,
		}); err != nil {
			fmt.Fprint(stderr, DisplayError(err, args.Debug))
			return ExitCodeFor(err)
		}
		return ExitOK
	case DaemonStop:
		if err := RunDaemonStop(args.DaemonPID); err != nil {
			fmt.Fprint(stderr, DisplayError(err, args.Debug))
			return ExitCodeFor(err)
		}
		return ExitOK
	default:
		fmt.Fprintf(stderr, "cli: unknown daemon subcommand %q\n", args.Subcommand)
		return ExitUserError
	}
}
