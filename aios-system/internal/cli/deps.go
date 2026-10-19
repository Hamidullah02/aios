package cli

import (
	"fmt"
	"path/filepath"

	"github.com/sadi/aios-system/internal/approvals"
	"github.com/sadi/aios-system/internal/ask"
	"github.com/sadi/aios-system/internal/core/audit"
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
	auditPath := "./var/aios.db"
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
	if !requiresPolicy(args.Command) {
		// Bootstrap path: try to load a policy if one
		// exists, but do not require it.
		if args.ConfigPath != "" {
			if rules, perr := policy.Load(args.ConfigPath); perr == nil {
				deps.Policy = policy.New(rules, deps.Audit)
			}
		}
	} else {
		if args.ConfigPath == "" {
			return nil, &CliError{
				Kind: "config",
				What: "policy file is required (use --config)",
			}
		}
		rules, perr := policy.Load(args.ConfigPath)
		if perr != nil {
			// Policy load failed; close the audit log so
			// we do not leak an open file handle (which
			// Windows would refuse to delete on temp dir
			// cleanup).
			_ = deps.Close()
			return nil, &CliError{
				Kind:  "config",
				What:  fmt.Sprintf("policy load failed: %s", args.ConfigPath),
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
		if err := wireAgentLoop(deps, args); err != nil {
			_ = deps.Close()
			return nil, err
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
	// Build the bus. The bus owns the tool map, the
	// policy gateway, and the audit log. The gateway
	// is required for the agent loop: without it the
	// bus would allow every tool call.
	bus := tools.New(tools.Config{
		Tools:  toolSet,
		Policy: deps.Policy,
		Audit:  deps.Audit,
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
