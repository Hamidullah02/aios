package cli

import (
	"fmt"
	"path/filepath"

	"github.com/sadi/aios-system/internal/approvals"
	"github.com/sadi/aios-system/internal/core/audit"
	"github.com/sadi/aios-system/internal/policy"
	"github.com/sadi/aios-system/internal/runtime"
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
// Worker and Verifier are wired to the passthrough
// implementations so `aios run <plan.yaml>` is runnable
// in a smoke-test without a real agent. Production
// deployments replace the worker with a real one (the
// planner-driven worker, a code-generation worker, etc.)
// by mutating deps.Worker / deps.Verifier after this
// call.
func BuildDeps(args *Args) (*Deps, error) {
	if args == nil {
		return nil, &CliError{Kind: "config", What: "args is nil"}
	}
	deps := &Deps{
		Budget:   runtime.Budget{},
		Worker:   PassthroughWorker{},
		Verifier: SmokeVerifier{},
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
	return deps, nil
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
