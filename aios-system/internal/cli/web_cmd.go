package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/sadi/aios-system/internal/webui"
)

// runWeb serves the operator UI.
//
// The command is foreground and blocks until the operator
// interrupts it. There is no background mode: the UI shows
// in-process run state, so a daemonised server would show
// an empty page, which is worse than not offering one.
//
// When --workspace is unset the hub is populated from the
// audit log rather than from a live orchestrator. `aios web`
// and `aios run` are separate processes, so the running
// plan is not ours to observe -- what the operator sees is
// what has already happened, read back from the durable
// record. A single process that both runs a plan and serves
// the UI gets live events through the same hub.
func runWeb(args *Args, deps *Deps, stdout, stderr io.Writer) int {
	hub := webui.NewHub()

	// Load the operator's saved panel arrangement. A failure
	// here costs a rearranged dashboard, so it is a note on
	// stderr rather than a fatal error: the UI still serves
	// with the default layout.
	if err := hub.SetStore(webui.NewLayoutStore(webui.DefaultLayoutPath())); err != nil {
		fmt.Fprintf(stderr, "note: could not load your saved layout, using the default: %v\n", err)
	}

	// Seed from the audit log so a freshly-started UI is not
	// blank. Best effort: a UI with no history is still a
	// working UI, so a read failure is a note on stderr
	// rather than a fatal error.
	if deps != nil && deps.Audit != nil {
		if n, err := seedFromAudit(hub, deps); err != nil {
			fmt.Fprintf(stderr, "note: could not read past activity: %v\n", err)
		} else if n > 0 {
			fmt.Fprintf(stderr, "note: %d past task(s) loaded from the audit log\n", n)
		}
	}

	// If an approval daemon is running, this UI can show the
	// pending queue and apply decisions to it.
	//
	// `aios web` is a separate process from the run, so it
	// holds no approvals of its own. Without the daemon it can
	// still show agent history, but the approval panel would be
	// permanently empty -- and an operator looking at a run
	// that is blocked would find a panel that says nothing,
	// which is worse than one that explains it cannot act.
	var stopPoll chan struct{}
	if args.SocketPath != "" {
		res := NewDaemonApprovalResolver(args.SocketPath, stderr)
		hub.SetApprovalResolver(res)
		stopPoll = make(chan struct{})
		go pollDaemonApprovals(res, hub, stopPoll, stderr)
	}

	srv, err := webui.New(webui.Config{Hub: hub, Addr: args.WebAddr})
	if err != nil {
		if stopPoll != nil {
			close(stopPoll)
		}
		fmt.Fprintf(stderr, "error: system: %v\n", err)
		return ExitSystemError
	}

	// Bind before printing the URL. With a port of 0 the
	// configured address is not the address the browser can
	// reach, and a printed URL that does not open is worse than
	// no URL at all.
	ln, err := srv.Listen()
	if err != nil {
		if stopPoll != nil {
			close(stopPoll)
		}
		fmt.Fprintf(stderr, "error: system: %v\n", err)
		return ExitSystemError
	}

	url := srv.SessionURL()
	if args.WebOpen {
		// --open reports the URL without serving. It exists
		// so an operator can check what the command would do
		// without holding a port, which matters in CI and in
		// a scripted smoke test.
		fmt.Fprintln(stdout, url)
		_ = ln.Close()
		return ExitOK
	}

	fmt.Fprintf(stdout, "aios web: serving %s\n", url)
	fmt.Fprintf(stdout, "the link above carries this session's token; treat it as a credential.\n")
	fmt.Fprintf(stdout, "press Ctrl-C to stop\n")
	if err := srv.Serve(ln); err != nil {
		fmt.Fprintf(stderr, "error: system: %v\n", err)
		return ExitSystemError
	}
	return ExitOK
}

// auditSeedLimit is how many rows of each kind the UI reads
// back on start.
//
// It is generous enough to cover any plan an operator is
// likely looking at, and bounded because this is a startup
// cost on every `aios web` launch: the UI is meant to open
// instantly.
const auditSeedLimit = 1000

// seedFromAudit reconstructs agent panels from the audit
// log's task lifecycle rows.
//
// It folds rows into the hub so the panels show the last
// known state of each task rather than a list of every
// transition. A task with a task_started and no terminal row
// is shown as running, which is correct for a plan in flight
// and is the most useful reading for one that was
// interrupted.
func seedFromAudit(hub *webui.Hub, deps *Deps) (int, error) {
	started, err := deps.Audit.GetByKind("task_started", auditSeedLimit)
	if err != nil {
		return 0, err
	}
	passed, err := deps.Audit.GetByKind("task_passed", auditSeedLimit)
	if err != nil {
		return 0, err
	}
	failed, err := deps.Audit.GetByKind("task_failed", auditSeedLimit)
	if err != nil {
		return 0, err
	}

	for _, r := range started {
		if r.TaskID == "" {
			continue
		}
		hub.SetStatus(r.TaskID, webui.StatusRunning, "")
		hub.Upsert(r.TaskID, func(a *webui.Agent) { a.TaskID = r.TaskID })
	}
	// Terminal states overwrite "running", and are applied in
	// the order passed then failed so a retried task that
	// later failed ends up failed rather than passed.
	for _, r := range passed {
		if r.TaskID == "" {
			continue
		}
		hub.SetStatus(r.TaskID, webui.StatusDone, "")
	}
	for _, r := range failed {
		if r.TaskID == "" {
			continue
		}
		hub.SetStatus(r.TaskID, webui.StatusFailed, strings.TrimSpace(r.Kind))
	}
	return len(started), nil
}
