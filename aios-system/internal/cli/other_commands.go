package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/sadi/aios-system/internal/core/audit"
	"github.com/sadi/aios-system/internal/core/schemas"
	"github.com/sadi/aios-system/internal/runtime"
)

// ResumeDeps is the wiring the resume subcommand needs.
type ResumeDeps struct {
	Audit   *audit.Log
	Resume  *runtime.ResumeHandler
	Worker  runtime.Worker
	Verify  runtime.Verifier
	Budget  runtime.Budget
	// GrantStore records the approvals granted during this
	// run. The bus consults it so a retried task whose call
	// the operator already approved proceeds instead of
	// re-raising the same `ask`.
	//
	// nil is safe: the run then behaves as before, re-asking
	// on every attempt.
	GrantStore *grantStore
	Approve    ApprovalHandler
	// Observer is optional. v0.41.1: when set, it receives
	// task lifecycle events. This is how `aios run --web`
	// feeds the live UI without the runtime knowing the UI
	// exists: the runtime publishes, this command bridges.
	Observer runtime.Observer
}

// Resume restarts a paused plan from the last checkpoint.
// Per REQ-CLI-RSM-001 the checkpoint is loaded, then the
// orchestrator is restarted with the checkpoint.
//
// The plan path must be re-supplied (the checkpoint stores
// the task context, not the plan). The CLI accepts the
// plan via Args.PlanPath, the same flag `run` uses.
func Resume(ctx context.Context, args *Args, deps ResumeDeps) (*runtime.Report, error) {
	if args == nil || args.TaskID == "" {
		return nil, &CliError{Kind: "user", What: "task id is required"}
	}
	if args.PlanPath == "" {
		return nil, &CliError{Kind: "user", What: "resume requires the original plan path (--plan)"}
	}
	if deps.Resume == nil {
		return nil, &CliError{Kind: "config", What: "resume handler is not configured"}
	}
	plan, err := LoadPlan(args.PlanPath)
	if err != nil {
		return nil, err
	}
	cp, err := deps.Resume.Load(args.TaskID)
	if err != nil {
		if errors.Is(err, runtime.ErrCheckpointNotFound) {
			return nil, &CliError{Kind: "user", What: fmt.Sprintf("no checkpoint for task %q", args.TaskID)}
		}
		return nil, &CliError{Kind: "config", What: "load checkpoint", Cause: err}
	}
	return runWithApprovals(ctx, plan, cp, deps, "resume")
}

// CancelDeps is the wiring the cancel subcommand needs.
type CancelDeps struct {
	Resume *runtime.ResumeHandler
	// Out receives the one-line summary of what cancel did.
	// Optional; nil discards it.
	Out io.Writer
}

// Cancel cancels a task per REQ-CLI-CXL-001.
//
// What it actually does today: deletes the persisted
// checkpoint, so the next `run` of that plan starts over
// rather than resuming. There is no signal path from this
// process to a running orchestrator in another one, so it
// cannot stop a run that is in flight right now; the help
// text used to claim it could.
//
// The function returns a *CliError{Kind: "user"} when there
// was no checkpoint to delete. Cancelling something that is
// not cancelled is a user error, and reporting it as success
// is how an operator ends up believing a runaway plan has
// stopped. It used to return nil here, so `aios cancel
// <anything>` always exited 0.
func Cancel(taskID string, deps CancelDeps) error {
	if taskID == "" {
		return &CliError{Kind: "user", What: "task id is required"}
	}
	if deps.Resume == nil {
		return &CliError{Kind: "config", What: "resume handler is not configured"}
	}
	// Check for the checkpoint BEFORE deleting it. Delete is
	// deliberately idempotent and swallows "not exist", so
	// deleting first would make "there was nothing to
	// cancel" indistinguishable from "cancelled
	// successfully" -- which is how `aios cancel
	// <anything>` came to always report success.
	if _, err := deps.Resume.Load(taskID); err != nil {
		if errors.Is(err, runtime.ErrCheckpointNotFound) {
			return &CliError{
				Kind: "user",
				What: fmt.Sprintf("nothing to cancel for %q: no checkpoint (a run in progress is not interrupted by cancel; stop the `aios run` process instead)", taskID),
			}
		}
		return &CliError{Kind: "config", What: "load checkpoint", Cause: err}
	}
	if err := deps.Resume.Delete(taskID); err != nil {
		return &CliError{Kind: "config", What: "delete checkpoint", Cause: err}
	}
	if deps.Out != nil {
		fmt.Fprintf(deps.Out, "cancelled %s: checkpoint deleted\n", taskID)
	}
	return nil
}

// ReportDeps is the wiring the report subcommand needs.
type ReportDeps struct {
	Audit  *audit.Log
	Format string // "json" | "markdown" | "html"
	Out    io.Writer
}

// Report prints a task report. Per REQ-CLI-RPT-001 the
// formats are JSON, Markdown, and HTML.
//
// The CLI reconstructs the report from the audit log. The
// runtime does not persist a separate "report" file; the
// audit log is the canonical record.
func Report(taskID string, deps ReportDeps) error {
	if taskID == "" {
		return &CliError{Kind: "user", What: "task id is required"}
	}
	if deps.Audit == nil {
		return &CliError{Kind: "config", What: "report: audit log is not configured"}
	}
	if deps.Out == nil {
		return &CliError{Kind: "config", What: "report: out is nil"}
	}
	format := deps.Format
	if format == "" {
		format = "json"
	}
	rows, err := deps.Audit.GetByTask(taskID)
	if err != nil {
		return &CliError{Kind: "audit", What: "read", Cause: err}
	}
	switch format {
	case "json":
		out, _ := json.MarshalIndent(rows, "", "  ")
		fmt.Fprintln(deps.Out, string(out))
	case "markdown":
		fmt.Fprintf(deps.Out, "# Report for %s\n\n", taskID)
		fmt.Fprintf(deps.Out, "Rows: %d\n\n", len(rows))
		fmt.Fprintln(deps.Out, "| seq | kind | ts_ns | content_hash |")
		fmt.Fprintln(deps.Out, "|-----|------|-------|--------------|")
		for _, r := range rows {
			fmt.Fprintf(deps.Out, "| %d | %s | %d | %s |\n", r.SequenceNumber, r.Kind, r.TSNano, r.ContentHash)
		}
	case "html":
		fmt.Fprintf(deps.Out, "<!doctype html><html><head><title>Report %s</title></head><body>\n", taskID)
		fmt.Fprintf(deps.Out, "<h1>Report for %s</h1>\n<table border='1'>\n", taskID)
		fmt.Fprintln(deps.Out, "<tr><th>seq</th><th>kind</th><th>ts_ns</th><th>hash</th></tr>")
		for _, r := range rows {
			fmt.Fprintf(deps.Out, "<tr><td>%d</td><td>%s</td><td>%d</td><td>%s</td></tr>\n",
				r.SequenceNumber, r.Kind, r.TSNano, r.ContentHash)
		}
		fmt.Fprintln(deps.Out, "</table></body></html>")
	default:
		return &CliError{Kind: "user", What: "unknown report format: " + format}
	}
	return nil
}

// AuditDeps is the wiring the audit subcommand needs.
type AuditDeps struct {
	Audit *audit.Log
	Out   io.Writer
	Tail  bool
}

// Audit prints the audit log.
//
// With a taskID it shows that task's rows. Without one it
// shows the tail of the whole log.
//
// The no-taskID path used to union a hardcoded list of "well
// known" kinds, which had drifted: `policy_decision` (written
// on every tool call), the plan-level state rows, and
// `task_cancelled` were all missing from it, so the default
// `aios audit` view silently omitted most of what the system
// recorded while `aios audit <task>` showed everything. It
// now calls Log.Recent, which cannot go stale.
//
// --tail is still not implemented (the audit log has no
// blocking read). Rather than exit 0 having printed nothing,
// which reads as "no activity", it says so and points at the
// supported alternative.
func Audit(taskID string, deps AuditDeps) error {
	if deps.Audit == nil {
		return &CliError{Kind: "config", What: "audit log is not configured"}
	}
	if deps.Out == nil {
		return &CliError{Kind: "config", What: "audit: out is nil"}
	}
	if deps.Tail {
		fmt.Fprintln(deps.Out,
			"--tail is not implemented yet; showing the last rows instead")
	}
	var rows []audit.PersistedRow
	if taskID != "" {
		r, err := deps.Audit.GetByTask(taskID)
		if err != nil {
			return &CliError{Kind: "audit", What: "read", Cause: err}
		}
		rows = r
	} else {
		r, err := deps.Audit.Recent(auditTailLimit)
		if err != nil {
			return &CliError{Kind: "audit", What: "read", Cause: err}
		}
		rows = r
	}
	for _, r := range rows {
		fmt.Fprintf(deps.Out, "%d\t%s\t%s\t%s\n", r.SequenceNumber, r.TaskID, r.Kind, r.ContentHash)
	}
	// v0.40.0: an empty result is a normal answer, not a
	// failure, but printing nothing at all reads as a
	// broken command. Say which query came back empty so
	// the operator can tell "no such task" from "the log
	// is empty" from "I am looking at the wrong file".
	//
	// Exit code stays 0: an unknown task is a legitimate
	// question with a legitimate answer.
	if len(rows) == 0 {
		if taskID != "" {
			fmt.Fprintf(deps.Out, "no rows for task %s\n", taskID)
		} else {
			fmt.Fprintln(deps.Out, "no rows in the audit log")
		}
	}
	return nil
}

// auditTailLimit is how many rows a bare `aios audit` shows.
//
// It is a bound so a long-lived log cannot flood a terminal;
// it is not a filter. Rows are never dropped by kind.
const auditTailLimit = 1000

// runWithApprovals is the inner loop shared by Run and
// Resume. It builds the orchestrator, runs it, and on
// *runtime.WaitingForApproval asks the approval handler to
// resolve the request, then re-runs from the next
// checkpoint. The flow runs at most MaxApprovalLoops times
// to bound infinite-loop bugs in the handler.
func runWithApprovals(
	ctx context.Context,
	plan *schemas.TaskPlan,
	cp *runtime.Checkpoint,
	deps ResumeDeps,
	caller string,
) (*runtime.Report, error) {
	if plan == nil {
		return nil, errors.New("plan is nil")
	}
	meter := runtime.NewBudgetMeter(planBudget(plan, deps.Budget))
	o := runtime.New(runtime.Config{
		Worker:   deps.Worker,
		Verifier: deps.Verify,
		Meter:    meter,
		Audit:    deps.Audit,
		Resume:   deps.Resume,
	})
	// v0.41.1: register the observer on the ONE orchestrator
	// instance, before the approval loop. Registering per
	// iteration would replay the task_started row for a task
	// that resumed into, double-counting it in the UI.
	o.AddObserver(deps.Observer)
	for i := 0; i < 100; i++ {
		report, err := o.Run(ctx, plan, cp)
		if err == nil {
			return report, nil
		}
		var wfa *runtime.WaitingForApproval
		if !errors.As(err, &wfa) {
			return report, err
		}
		if deps.Approve == nil {
			// No handler: the run cannot ask anyone. This is
			// a wiring bug in the caller, not a user error,
			// so say what to do rather than leaking the raw
			// "waiting for approval" sentinel that told an
			// operator nothing about how to proceed.
			return report, &CliError{
				Kind: "approval",
				What: fmt.Sprintf("task %s needs approval but no approval handler is configured (this is a bug; run through `aios run`, or pass --yes / --socket)", taskIDOf(wfa)),
			}
		}
		ok, appErr := deps.Approve.Approve(ctx, wfa)
		if appErr != nil {
			return report, &CliError{Kind: "approval", What: caller + ": approval handler failed", Cause: appErr}
		}
		if !ok {
			return report, &CliError{Kind: "approval", What: caller + ": operator denied the action"}
		}
		// Record the grant BEFORE the retry, so the retried
		// task's identical tool call is recognised as
		// already approved by the bus instead of prompting
		// the operator a second time for one action.
		if deps.GrantStore != nil {
			deps.GrantStore.Grant(wfa)
		}
		next, cpErr := deps.Resume.Load(plan.ID)
		if cpErr != nil {
			return report, &CliError{Kind: "resume", What: "load checkpoint", Cause: cpErr}
		}
		cp = next
	}
	return nil, &CliError{Kind: "approval", What: "too many approval loops (>100)"}
}

// planBudget merges the plan's declared budget into the
// caller's budget envelope.
//
// A TaskPlan carries a schemas.Budget with cost_usd and
// wall_clock_sec, but nothing ever mapped it onto the
// runtime.Budget the meter enforces, so every run enforced
// nothing and the report printed cost=0.0000 tool=0 model=0
// net=0 no matter how much work happened. A budget the
// operator wrote into their plan was documentation, not a
// limit.
//
// Precedence: an explicit caller-supplied cap wins. The
// caller (BuildDeps, or a test) may have set a tighter
// operational limit, and a plan file must not be able to
// loosen a limit the operator configured out of band.
func planBudget(plan *schemas.TaskPlan, caller runtime.Budget) runtime.Budget {
	out := caller
	if plan == nil || plan.Budget == nil {
		return out
	}
	if out.MaxCostUSD == 0 && plan.Budget.CostUSD > 0 {
		out.MaxCostUSD = plan.Budget.CostUSD
	}
	if out.MaxWallTimeSec == 0 && plan.Budget.WallClockSec > 0 {
		out.MaxWallTimeSec = plan.Budget.WallClockSec
	}
	return out
}
// taskIDOf extracts a task identifier from an approval
// request for use in an error message.
//
// The runtime request carries the approval id, not the task
// id, and the two differ ("ask:t1" vs "t1"). Returning the
// request's id is enough to let an operator find the task.
func taskIDOf(wfa *runtime.WaitingForApproval) string {
	if wfa == nil || wfa.ApprovalID == "" {
		return "unknown"
	}
	return wfa.ApprovalID
}