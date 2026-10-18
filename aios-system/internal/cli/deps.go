package cli

import (
	"fmt"
	"path/filepath"

	"github.com/sadi/aios-system/internal/approvals"
	"github.com/sadi/aios-system/internal/core/audit"
	"github.com/sadi/aios-system/internal/policy"
	"github.com/sadi/aios-system/internal/runtime"
)

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
	// Audit log: default to ./var/aios.db (SQLite). The
	// audit log is required for the run / resume / report
	// / audit subcommands; init / daemon can run without
	// one.
	auditPath := "./var/aios.db"
	if args.LogPath != "" {
		auditPath = args.LogPath
	}
	if abs, err := filepath.Abs(auditPath); err == nil {
		auditPath = abs
	}
	// Audit log: required for run / resume / report /
	// audit / cancel. Optional for init and daemon start:
	// those commands create or operate on the workspace
	// before the audit log exists.
	needAudit := !isInitOrDaemonStart(args)
	if needAudit {
		log, err := audit.Open(audit.Config{Path: auditPath})
		if err != nil {
			return nil, &CliError{Kind: "config", What: "open audit log " + auditPath, Cause: err}
		}
		deps.Audit = log
	}
	// Policy gateway: optional. Without it every tool
	// call is allowed. The init subcommand does not
	// require a policy; the other subcommands do.
	// A policy file was passed but failed to load is a
	// configuration error and must be reported. The
	// already-opened audit log is closed before the
	// function returns the error.
	if args.ConfigPath != "" {
		rules, err := policy.Load(args.ConfigPath)
		if err != nil {
			if deps.Audit != nil {
				_ = deps.Audit.Close()
			}
			return nil, &CliError{Kind: "config", What: "load policy " + args.ConfigPath, Cause: err}
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

// isInitOrDaemonStart reports whether the command does
// not need an audit log. Init creates the workspace;
// `daemon start` boots the daemon (which opens its own
// audit); `daemon stop` only removes a pidfile. None of
// these touch the audit log.
func isInitOrDaemonStart(args *Args) bool {
	if args.Command == CmdInit {
		return true
	}
	if args.Command == CmdDaemon {
		return args.Subcommand == DaemonStart || args.Subcommand == DaemonStop
	}
	return false
}

func fmtPath(p string) string {
	if p == "" {
		return "(unset)"
	}
	return fmt.Sprintf("%s", p)
}
