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

	srv, err := webui.New(webui.Config{Hub: hub, Addr: args.WebAddr})
	if err != nil {
		fmt.Fprintf(stderr, "error: system: %v\n", err)
		return ExitSystemError
	}

	url := "http://" + srv.Addr() + "/"
	if args.WebOpen {
		// --open reports the URL without binding. It exists
		// so an operator can check what the command would do
		// without holding a port, which matters in CI and in
		// a scripted smoke test.
		fmt.Fprintln(stdout, url)
		return ExitOK
	}

	fmt.Fprintf(stdout, "aios web: serving %s\n", url)
	fmt.Fprintf(stdout, "press Ctrl-C to stop\n")
	if err := srv.ListenAndServe(); err != nil {
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
