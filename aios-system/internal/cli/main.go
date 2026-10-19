package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
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
