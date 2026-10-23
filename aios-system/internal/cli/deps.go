package cli

import (
	"fmt"
	"io"
	"path/filepath"

	"github.com/sadi/aios-system/internal/approvals"
	"github.com/sadi/aios-system/internal/ask"
	"github.com/sadi/aios-system/internal/core/audit"
	"github.com/sadi/aios-system/internal/egress"
	"github.com/sadi/aios-system/internal/memory"
	"github.com/sadi/aios-system/internal/models"
	"github.com/sadi/aios-system/internal/planner"
	"github.com/sadi/aios-system/internal/policy"
	"github.com/sadi/aios-system/internal/runtime"
	"github.com/sadi/aios-system/internal/sandbox"
	"github.com/sadi/aios-system/internal/tools"
)

// subcommands that bootstrap the workspace. These run
// before the policy file exists, so BuildDeps must not
// require a valid policy for them. Every other subcommand
// fails closed: missing or invalid policy is a hard
// error and the binary exits 2.
var bootstrapSubcommands = map[string]bool{
	CmdInit:   true,
	CmdDaemon: true,
	// `web` serves the operator UI. v0.41.0: it is in
	// this set because the UI is useful before a policy
	// exists -- an operator opens it to see that nothing
	// has run yet -- and because requiring a policy to
	// LOOK at state would make the first-run experience
	// fail with a config error instead of an empty page.
	//
	// The UI is read-only over run state and can post
	// messages; it can never start or stop a plan, so it
	// grants no authority the policy would have denied.
	CmdWeb: true,
	// `tools` lists what the agent can dispatch. It builds
	// the tool set directly rather than reading the policy
	// or the audit log, so demanding either one bought
	// nothing: on a workspace where `aios init` has not
	// run yet, `aios tools` was the exact command a user
	// would reach for to diagnose the failure, and it
	// failed with "run `aios init` first" -- pointing at
	// the very command that would have fixed it, from
	// inside the diagnostic path.
	CmdTools: true,
	// approve / deny speak to a running daemon's socket and
	// resolve what it is already holding. They need no local
	// policy, and requiring one would be actively harmful: a
	// broken policy is exactly when an operator most needs to
	// inspect and clear what is pending.
	CmdApprove: true,
	CmdDeny:    true,
}

// requiresPolicy reports whether the subcommand needs a
// loaded policy gateway. Bootstrap subcommands return
// false; everything else returns true.
func requiresPolicy(command string) bool {
	if bootstrapSubcommands[command] {
		return false
	}
	return true
}

// Deps is the production dependency graph the CLI uses.
// Every subcommand reads what it needs from this struct;
// the production binary calls BuildDeps to construct it.
//
// Tests can build a Deps with only the fields the test
// cares about. The other fields stay nil and the
// subcommand returns a *ConfigError when the test does
// not exercise that code path.
type Deps struct {
	Audit     *audit.Log
	Policy    *policy.Gateway
	Resume    *runtime.ResumeHandler
	Bus       *tools.Bus
	Worker    runtime.Worker
	Verifier  runtime.Verifier
	Budget    runtime.Budget
	Approval  ApprovalHandler
	Approvals *approvals.Service
	// Grants is the per-run approval store the bus consults.
	// It is the reason an operator answering "yes" actually
	// lets the call through: without it the bus re-asks on
	// every retry.
	Grants *grantStore
	// stdin/stdout for the interactive approval prompt.
	//
	// Main fills these so the handler built in BuildDeps can
	// actually ask a human. They are fields rather than
	// parameters because BuildDeps predates the approval
	// prompt and is called from Main and from tests.
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
	// Router, Prices, Egress: wired for the `fix`
	// subcommand only. The router holds the model
	// adapters (OpenAI, Anthropic); the price table
	// computes USD cost; the egress allow list is a
	// future hook. nil for subcommands that do not
	// need a model client.
	Router *models.Router
	Prices *models.PriceTable
	Egress *egress.AllowList
	// Planner: the `fix` subcommand's planner. nil
	// for subcommands that do not call the planner.
	// The planner holds the model router internally;
	// the Router field is also exposed so tests can
	// stub the model client.
	Planner *planner.Planner
}

// newApprovalHandler picks the approval strategy for this
// invocation.
//
// The default is the terminal prompt. That is a change in
// behaviour, not just a new code path: previously no handler
// existed at all and every gated call was an unresolvable
// error. Making the prompt the default is the smallest
// behaviour that lets a plan finish while still putting a
// human in front of every gated action.
func newApprovalHandler(args *Args, deps *Deps) ApprovalHandler {
	if args.AssumeYes {
		return &ApprovalPrompt{
			In:        deps.Stdin,
			Out:       deps.Stderr,
			AssumeYes: true,
		}
	}
	// An explicit --socket means the operator is driving
	// approvals out-of-band (a second window, a separate
	// operator tool), so defer to the daemon.
	if args.SocketExplicit {
		return &DaemonApprovalHandler{
			SocketPath: args.SocketPath,
			Out:        deps.Stderr,
		}
	}
	// Otherwise ask at the terminal. If stdin cannot answer
	// (EOF, a pipe, a CI job), the prompt reads that as a
	// denial rather than hanging or auto-approving.
	return &ApprovalPrompt{
		In:      deps.Stdin,
		Out:     deps.Stderr,
		Timeout: approvalReadTimeout,
	}
}

// Close releases the resources the Deps holds. The
// production binary calls it on exit. The audit log is
// the main resource; the others are pure structs.
//
// Tests that open a real audit log must call Close
// before the temp directory is removed or the cleanup
// will fail on Windows where open files cannot be
// deleted.
func (d *Deps) Close() error {
	if d == nil {
		return nil
	}
	if d.Audit != nil {
		err := d.Audit.Close()
		d.Audit = nil
		return err
	}
	return nil
}

// BuildDeps constructs the production dependency graph
// from a parsed Args. The function returns an error when
// a required dependency cannot be built (the audit log
// path is unreachable, the policy file is invalid, ...).
//
// The function is best-effort: the production binary
// treats a failure here as exit code 2 (system error).
//
// Worker and Verifier are wired to the production
// implementations:
//
//   - Worker is a *BusWorker that dispatches each task to
//     the tools.Bus. The bus, in turn, runs the policy
//     gateway, executes the tool, and writes the audit
//     row. This is the model->tool->result loop the doc
//     set describes; the previous default (PassthroughWorker)
//     did not touch the bus and silently passed every
//     task.
//   - Verifier is a CommandVerifier that inspects the
//     bus's WorkerResult and rejects empty outputs, OK=false
//     results, and non-empty error strings. The previous
//     default (SmokeVerifier) returned VerdictPass for
//     every result and therefore rubber-stamped every
//     tool call, including failed ones.
//
// The bus is constructed with the production tool set:
// read_file, search_files, write_file, run_shell (with
// a SubprocessRunner), and ask_user (with a stdin prompt
// callback). Tests can inject a custom bus by mutating
// deps.Bus after this call returns and rebuilding the
// worker on top of it.
func BuildDeps(args *Args) (*Deps, error) {
	if args == nil {
		return nil, &CliError{Kind: "config", What: "args is nil"}
	}
	deps := &Deps{
		Budget: runtime.Budget{},
	}
	// Audit log: required for run / resume / report /
	// audit / cancel. Optional for init and daemon start:
	// those commands create or operate on the workspace
	// before the audit log exists. A failure to open the
	// audit log is a hard error for non-bootstrap
	// subcommands.
	//
	// The default is DefaultLogPath (see parser.go). Both
	// live in one constant because the parser always
	// populates args.LogPath, so the fallback below is only
	// reachable when a caller builds Args programmatically
	// rather than through Parse.
	auditPath := DefaultLogPath
	if args.LogPath != "" {
		auditPath = args.LogPath
	}
	if abs, err := filepath.Abs(auditPath); err == nil {
		auditPath = abs
	}
	log, err := audit.Open(audit.Config{Path: auditPath})
	if err != nil {
		if requiresPolicy(args.Command) {
			return nil, &CliError{
				Kind:  "config",
				What:  fmt.Sprintf("audit log open failed: %s (run `aios init` first)", auditPath),
				Cause: err,
			}
		}
		// Bootstrap subcommand: log is optional.
	} else {
		deps.Audit = log
	}
	// Policy gateway: required for every non-bootstrap
	// subcommand. A missing or unparseable policy file
	// is a hard error. The init and daemon subcommands
	// are exempt: they create the policy file (init) or
	// wire the daemon before policy validation runs.
	//
	// The policy is loaded from `policy.yaml` next to
	// the `--config` file (the canonical layout `aios
	// init` scaffolds). If `args.ConfigPath` already
	// points at a policy file (test compat / operator
	// override), the policyPath resolver returns it
	// unchanged.
	if !requiresPolicy(args.Command) {
		// Bootstrap path: try to load a policy if one
		// exists, but do not require it.
		if args.ConfigPath != "" {
			if ppath, perr := policyPath(args.ConfigPath); perr == nil {
				if rules, perr := policy.Load(ppath); perr == nil {
					deps.Policy = policy.New(rules, deps.Audit)
				}
			}
		}
	} else {
		if args.ConfigPath == "" {
			return nil, &CliError{
				Kind: "config",
				What: "policy file is required (use --config)",
			}
		}
		ppath, err := policyPath(args.ConfigPath)
		if err != nil {
			return nil, &CliError{
				Kind:  "config",
				What:  "policy path resolution failed",
				Cause: err,
			}
		}
		rules, perr := policy.Load(ppath)
		if perr != nil {
			// Policy load failed; close the audit log so
			// we do not leak an open file handle (which
			// Windows would refuse to delete on temp dir
			// cleanup).
			_ = deps.Close()
			return nil, &CliError{
				Kind:  "config",
				What:  fmt.Sprintf("policy load failed: %s", ppath),
				Cause: perr,
			}
		}
		deps.Policy = policy.New(rules, deps.Audit)
	}
	// Resume handler: required for run / resume / cancel.
	deps.Resume = runtime.NewResumeHandler("./var/checkpoints")
	// Approvals service: optional. The daemon wires it
	// when running; the CLI's run subcommand does not
	// require it.
	deps.Approvals = approvals.New(approvals.Config{Audit: deps.Audit})
	// Tool bus + worker + verifier. These are the agent
	// loop the doc set describes: each task flows from
	// the orchestrator to the BusWorker, the worker
	// dispatches to the bus, the bus consults the policy
	// gateway, runs the tool, and writes the audit row,
	// and the verifier inspects the result.
	//
	// Bootstrap subcommands (init, daemon) skip the
	// wiring: they do not need a worker because they do
	// not run plans.
	if requiresPolicy(args.Command) {
		// Per-run grant store. Created BEFORE the bus is wired
		// because the bus holds a reference to it. One store
		// per process is one store per run: grants never leak
		// between plans.
		deps.Grants = newGrantStore(deps.Approvals)
		if err := wireAgentLoop(deps, args); err != nil {
			_ = deps.Close()
			return nil, err
		}
		// Approval handler.
		//
		// This was NEVER assigned before, so deps.Approval
		// stayed nil for every subcommand. runWithApprovals
		// checks `deps.Approve == nil` and returns the raw
		// WaitingForApproval, so any plan hitting an `ask`
		// policy rule stopped with "runtime: waiting for
		// approval ask:t1" and exit 2. The grant/deny
		// machinery existed (approvals.Service, the daemon's
		// socket protocol) but no path reached it: the gate
		// was a dead end.
		//
		// Precedence, most explicit first:
		//   --yes            auto-approve (loudly)
		//   --socket=<path>  drive the approval daemon
		//   default          prompt the operator at the
		//                     terminal
		//
		// The default is the interactive prompt because the
		// overwhelmingly common case is one human at one
		// terminal running one plan, and the point of an
		// approval gate is that a human sees the request.
		if args.Command == CmdRun || args.Command == CmdFix || args.Command == CmdResume {
			deps.Approval = newApprovalHandler(args, deps)
		}
		// The `fix` subcommand also needs a planner
		// (it turns the objective into a plan) and a
		// model router (the planner calls the model).
		// Other non-bootstrap subcommands (run,
		// resume, cancel, report, audit) do not call
		// the planner; they operate on a plan file
		// that already exists.
		if args.Command == CmdFix {
			if err := wirePlanner(deps, args); err != nil {
				_ = deps.Close()
				return nil, err
			}
			// The `fix` subcommand now has a router.
			// Wire the SelfCheckTool (it needs a Judge)
			// into the bus late. The bus has a Register
			// method for exactly this case.
			if deps.Router != nil {
				// Only add the SelfCheckTool here.
				// RunTestsTool was already wired by
				// wireAgentLoop.
				late := wireSelfCheckOnly(deps.Router, deps.Audit)
				if late != nil {
					deps.Bus.Register("self_check", late)
				}
			}
		}
	}
	return deps, nil
}

// wireAgentLoop constructs the tools.Bus, the BusWorker,
// and the CommandVerifier, and wires them into deps. The
// function is split out of BuildDeps because the wiring
// is non-trivial and reading it in isolation is easier
// than reading it inside the larger BuildDeps body.
//
// The function assumes the policy gateway is already
// loaded (deps.Policy != nil for non-bootstrap
// subcommands) and the audit log is open (deps.Audit
// may be nil; the bus handles that case).
func wireAgentLoop(deps *Deps, args *Args) error {
	// Build the production tool set. The set is the
	// minimum the doc set lists: read_file, search_files,
	// write_file, run_shell, and ask_user. run_shell uses
	// a SubprocessRunner; ask_user uses a stdin prompt
	// callback so the operator can answer interactively.
	askTool, err := ask.New((&StdinPromptCallback{}).StdinPrompt(), nil)
	if err != nil {
		return &CliError{
			Kind:  "config",
			What:  "ask tool init failed",
			Cause: err,
		}
	}
	toolSet := map[string]tools.Tool{
		"read_file":    tools.ReadFileTool{},
		"search_files": tools.SearchFilesTool{},
		"write_file":   tools.WriteFileTool{},
		"run_shell":    tools.RunShellTool{Runner: sandbox.NewSubprocessRunner()},
		"ask_user":     askBusAdapter{tool: askTool},
	}
	// Verify-package tools. The RunTestsTool is always
	// wired (no model dependency). SelfCheckTool is
	// wired below after the router is built (only
	// for `fix`).
	verifyTools := wireVerifyTools(nil, deps.Audit)
	for name, tool := range verifyTools {
		toolSet[name] = tool
	}
	// v0.38.0: the rest of the doc-set tool set. The
	// five tools above are the REQ-DOC-MIN minimum; the
	// edit / git / search / image packages each ship a
	// working tools.Tool implementation that was never
	// registered with the bus, so a plan naming one of
	// them got ToolNotFoundError. wireDocSetTools closes
	// that gap; see internal/cli/docset_wiring.go for the
	// per-package notes and the deliberate exclusions.
	docSetTools := wireDocSetTools()
	for name, tool := range docSetTools {
		// Never shadow a tool that is already wired.
		// The verify set wins on a name collision so a
		// future rename in another package cannot
		// silently change what run_tests resolves to.
		if _, exists := toolSet[name]; exists {
			continue
		}
		toolSet[name] = tool
	}
	// v0.39.0: the memory adapters. Unlike the literal set
	// above these need a constructed *memory.Memory, and
	// both tools share one instance so a write is visible
	// to a read in the same process.
	//
	// Construction is best-effort: a workspace whose data
	// dir is unwritable still starts, the memory tools
	// return a typed error rather than panicking, and every
	// other tool keeps working. Failing the whole CLI over
	// an unwritable memory directory would make the
	// problem strictly worse than not having memory.
	mem, memErr := memory.NewMemory(memory.MemoryConfig{})
	if memErr != nil {
		// Leave mem nil. wireMemoryTools still registers
		// the two tools so `aios tools` shows them and a
		// plan gets a clear "memory is not wired" error
		// instead of ToolNotFoundError, which would send
		// the operator looking for a policy problem.
		mem = nil
	}
	memoryTools := wireMemoryTools(mem, deps.Audit)
	for name, tool := range memoryTools {
		if _, exists := toolSet[name]; exists {
			continue
		}
		toolSet[name] = tool
	}
	// Build the bus. The bus owns the tool map, the
	// policy gateway, and the audit log. The gateway
	// is required for the agent loop: without it the
	// bus would allow every tool call.
	bus := tools.New(tools.Config{
		Tools:  toolSet,
		Policy: deps.Policy,
		Audit:  deps.Audit,
		// Consult the per-run grants so a call the operator
		// already approved proceeds on the retry instead of
		// raising the same `ask` again.
		Grants: deps.Grants,
	})
	deps.Bus = bus
	// Resolve the workspace root. The default is the
	// parent directory of the policy file (the same
	// convention `aios init` uses for the scaffold).
	// Operators can override with --workspace.
	workspace := args.Workspace
	if workspace == "" {
		workspace = filepath.Dir(args.ConfigPath)
	}
	if abs, err := filepath.Abs(workspace); err == nil {
		workspace = abs
	}
	deps.Worker = NewBusWorker(bus, askTool, workspace)
	deps.Verifier = NewCommandVerifier()
	return nil
}

// DBPinger is a small interface the CLI uses to verify the
// database is reachable. It is here so tests can stub the
// ping without touching the real SQLite driver.
type DBPinger interface {
	Ping() error
}

// fmtPath is a tiny helper used by the init printer.
func fmtPath(p string) string {
	if p == "" {
		return "(unset)"
	}
	return fmt.Sprintf("%s", p)
}

// wirePlanner constructs the model router, the egress
// allow list, the price table, and the planner. The
// function is split out of BuildDeps because the
// planner wiring is non-trivial and reading it in
// isolation is easier than reading it inside the
// larger BuildDeps body.
//
// The function assumes wireAgentLoop has already
// run: the audit log is open, the policy gateway
// is loaded, and the resume handler is constructed.
// The function does not depend on the bus or the
// worker; the planner is a separate dependency
// that runs before the loop.
//
// The workspace is taken from args.Workspace (or
// the dirname of args.ConfigPath as a fallback).
// The router reads the egress, prices, and
// capabilities files in that workspace.
func wirePlanner(deps *Deps, args *Args) error {
	workspace := args.Workspace
	if workspace == "" {
		workspace = filepath.Dir(args.ConfigPath)
	}
	workspace = ensureWorkspace(workspace)
	router, prices, allow, err := buildRouter(workspace)
	if err != nil {
		return err
	}
	deps.Router = router
	deps.Prices = prices
	deps.Egress = allow
	// The planner verifies each candidate plan with the
	// same CodeFirstVerifier the planner package's
	// tests use. The verifier is the typed bridge
	// between the planner (which speaks ToolSpec) and
	// the bus (which speaks Tool).
	tools := plannerToolSpecs()
	verifier := planner.NewCodeFirstVerifier(tools)
	pl, err := planner.NewPlanner(planner.Config{
		Router:   router,
		Tools:    tools,
		Verifier: verifier,
	})
	if err != nil {
		return &CliError{
			Kind:  "config",
			What:  "planner init failed",
			Cause: err,
		}
	}
	deps.Planner = pl
	return nil
}
