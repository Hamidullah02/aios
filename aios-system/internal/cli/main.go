package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/sadi/aios-system/internal/webui"
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
	// Hand the terminal to the approval handler. BuildDeps
	// runs before the subcommand dispatcher, so this is the
	// first point at which a handler can exist; without it
	// the interactive approval prompt has no stream to read
	// from and every gated call dead-ends.
	//
	// stdin is os.Stdin rather than a parameter because Main
	// is the only place that knows about the real process.
	if deps.Stdin == nil {
		deps.Stdin = os.Stdin
	}
	deps.Stdout = stdout
	deps.Stderr = stderr
	if args.Command == CmdRun || args.Command == CmdFix || args.Command == CmdResume {
		deps.Approval = newApprovalHandler(args, deps)
		// The egress guard installs the network policy on the
		// process-wide default runner. It has to be started
		// before any tool can run a subprocess, which is why
		// it is here rather than inside the subcommand
		// handlers.
		guard, err := StartEgressGuard(workspaceForArgs(args), deps.Audit, args.EgressMode)
		if err != nil {
			fmt.Fprint(stderr, DisplayError(err, args.Debug))
			return ExitCodeFor(err)
		}
		defer func() {
			if err := guard.Close(); err != nil {
				fmt.Fprintf(stderr, "cli: close egress guard: %v\n", err)
			}
		}()
		fmt.Fprintln(stderr, guard.Describe())
	}
	defer func() {
		if err := deps.Close(); err != nil {
			fmt.Fprintf(stderr, "cli: close deps: %v\n", err)
		}
	}()
	return RunCommand(args, deps, stdout, stderr)
}

// workspaceForArgs returns the workspace directory for a run.
//
// The workspace is where var/egress.yaml lives, so the egress
// guard needs the same directory the planner and the router
// read. An explicit --workspace wins; otherwise the config
// file's directory is the workspace.
func workspaceForArgs(args *Args) string {
	if args == nil {
		return "./var"
	}
	if args.Workspace != "" {
		return ensureWorkspace(args.Workspace)
	}
	if args.ConfigPath != "" {
		return ensureWorkspace(filepath.Dir(args.ConfigPath))
	}
	return ensureWorkspace("./var")
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
		return runCancel(args, deps, stdout, stderr)
	case CmdReport:
		return runReport(args, deps, stdout, stderr)
	case CmdAudit:
		return runAudit(args, deps, stdout, stderr)
	case CmdDaemon:
		return runDaemon(ctx, args, deps, stderr)
	case CmdTools:
		return runTools(args, deps, stdout, stderr)
	case CmdWeb:
		return runWeb(args, deps, stdout, stderr)
	case CmdApprove:
		return runApprovalDecision(args, true, stdout, stderr)
	case CmdDeny:
		return runApprovalDecision(args, false, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "cli: unknown subcommand %q\n", args.Command)
		return ExitUserError
	}
}

// runApprovalDecision implements `aios approve <id>` and
// `aios deny <id>`.
//
// The approval gate only ever had one resolution path: the
// run's own process asking its terminal. That works for one
// person at one machine. It does not work for the cases that
// matter in practice: a run on a background host, a second
// administrator who is not sitting at that terminal, or a
// reviewer who wants to decide from somewhere else. Both
// commands talk the daemon's own socket protocol, so the
// daemon stays the single source of truth for what is
// pending.
func runApprovalDecision(args *Args, approve bool, stdout, stderr io.Writer) int {
	verb := "approve"
	if !approve {
		verb = "deny"
	}
	if len(args.Positional) == 0 {
		fmt.Fprintf(stderr, "error: user: `aios %s` requires an approval id: aios %s <approval-id>\n", verb, verb)
		return ExitUserError
	}
	id := strings.TrimSpace(args.Positional[0])
	if id == "" {
		fmt.Fprintf(stderr, "error: user: approval id is empty\n")
		return ExitUserError
	}

	c, err := dialDaemon(args.SocketPath, 5*time.Second)
	if err != nil {
		fmt.Fprintf(stderr, "%s", DisplayError(err, args.Debug))
		return ExitSystemError
	}
	defer c.Close()

	// The wire verb is NOT the CLI verb. The daemon's
	// protocol (REQ-CLI-DAE-003) names the message "grant";
	// the user-facing command is `approve`. Sending "approve"
	// gets `unknown kind: approve` back, which is what the
	// first version of this did.
	wire := verb
	if approve {
		wire = "grant"
	}
	req := map[string]any{
		"kind":  wire,
		"id":    id,
		"actor": "operator",
	}
	if approve {
		// The daemon checks the binding on grant, so the
		// context has to match the call-site. The task,
		// tool, and actor are unknown to this process, so
		// they are sent empty and the daemon treats an empty
		// field as "do not check": the operator already
		// named the exact approval id, which is the
		// capability.
		req["ctx"] = map[string]any{"actor": "operator"}
	}
	resp, err := c.send(req)
	if err != nil {
		fmt.Fprintf(stderr, "%s", DisplayError(err, args.Debug))
		return ExitSystemError
	}
	state, _ := resp["state"].(string)
	if state == "" {
		state = "granted"
		if !approve {
			state = "denied"
		}
	}
	fmt.Fprintf(stdout, "approval %s: %s\n", id, state)
	return ExitOK
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
	// v0.41.1: --web serves the UI from this process so the
	// panels update live while the plan runs.
	//
	// Without it, an operator who opens `aios web` in another
	// shell sees only the audit log's history, because the two
	// commands are separate processes and only this one holds
	// the orchestrator.
	var live *webui.Hub
	// approval is the handler the run uses. With --web it is
	// the UI handler, so a gated call is decided in the
	// browser; without it, it stays whatever Main wired (the
	// terminal prompt or the daemon).
	approval := deps.Approval
	if args.WebLive {
		live = webui.NewHub()
		// Load the operator's saved panel arrangement so
		// --web looks like the UI they already arranged.
		if err := live.SetStore(webui.NewLayoutStore(webui.DefaultLayoutPath())); err != nil {
			fmt.Fprintf(stderr, "note: could not load your saved layout, using the default: %v\n", err)
		}
		srv, err := webui.New(webui.Config{Hub: live, Addr: args.WebAddr})
		if err != nil {
			fmt.Fprint(stderr, DisplayError(&CliError{Kind: "config", What: "run --web: " + err.Error()}, args.Debug))
			return ExitSystemError
		}
		// Bind before printing the URL: with a port of 0 the
		// configured address is not the address the browser can
		// reach.
		//
		// A bind failure must not fail the plan. The plan is the
		// point; the UI is a convenience. The run continues
		// without the UI and says why.
		ln, err := srv.Listen()
		if err != nil {
			fmt.Fprintf(stderr, "note: the operator UI could not start (%v); the run continues without it.\n", err)
			ln = nil
		}
		// The handler is built AFTER the server so it can carry
		// the address the listener actually bound to. Building
		// it first meant the approval prompt printed the default
		// port to an operator whose run was listening elsewhere.
		//
		// Registering the resolver is a field write on the hub,
		// and the server has not started serving yet, so no
		// request can arrive for a run that has nobody waiting
		// on it.
		//
		// The URL is the session URL when the listener bound: it
		// carries the token the browser needs. When the bind
		// failed there is no UI to open, so the address is
		// printed on its own and the prompt says where to go
		// instead.
		url := srv.BaseURL()
		if ln != nil {
			url = srv.SessionURL()
			fmt.Fprintf(stdout, "aios: serving the operator UI at %s\n", url)
			fmt.Fprintf(stdout, "the link above carries this session's token; treat it as a credential.\n")
		}
		approval = NewUIApprovalHandler(live, stderr, url)
		// The server runs on its own goroutine and is never
		// shut down: the process exits when the plan finishes,
		// and a graceful-shutdown handshake for a process that
		// is about to exit would be code with no caller.
		if ln != nil {
			go func() {
				if sErr := srv.Serve(ln); sErr != nil {
					// Not fatal: the plan is still running and the
					// operator has the terminal report. Surface it
					// and carry on.
					fmt.Fprintf(stderr, "aios: web server stopped: %v\n", sErr)
				}
			}()
		}
		// Give the listener a moment so the URL printed above
		// is reachable by the time an operator clicks it. This
		// is a courtesy delay, not a correctness barrier: the
		// UI's SSE client reconnects with backoff regardless.
		time.Sleep(150 * time.Millisecond)
	}

	report, err := Run(ctx, args, RunDeps{
		Audit:        deps.Audit,
		Policy:       deps.Policy,
		Resume:       deps.Resume,
		Worker:       deps.Worker,
		Verifier:     deps.Verifier,
		Budget:       deps.Budget,
		Approval:     approval,
		Grants:       deps.Grants,
		ReportWriter: nil,
		Observer:     webui.NewBridge(live),
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

func runCancel(args *Args, deps *Deps, stdout, stderr io.Writer) int {
	if err := Cancel(args.TaskID, CancelDeps{Resume: deps.Resume, Out: stdout}); err != nil {
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
		Budget:   deps.Budget,
		Approve:  deps.Approval,
		GrantStore: deps.Grants,
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
