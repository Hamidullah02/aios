package cli

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/sadi/aios-system/internal/core/audit"
	"github.com/sadi/aios-system/internal/core/schemas"
	"github.com/sadi/aios-system/internal/report"
	"github.com/sadi/aios-system/internal/runtime"
)

// ResumeDeps is the wiring the resume subcommand needs.
type ResumeDeps struct {
	Audit   *audit.Log
	Resume  *runtime.ResumeHandler
	Worker  runtime.Worker
	Verify  runtime.Verifier
	Budget  runtime.Budget
	Approve ApprovalHandler
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
}

// Cancel stops a running task. Per REQ-CLI-CXL-001. The
// runtime does not expose a public Cancel hook on a
// specific plan today; the CLI deletes the persisted
// checkpoint so the next `run` starts over.
func Cancel(taskID string, deps CancelDeps) error {
	if taskID == "" {
		return &CliError{Kind: "user", What: "task id is required"}
	}
	if deps.Resume == nil {
		return &CliError{Kind: "config", What: "resume handler is not configured"}
	}
	if err := deps.Resume.Delete(taskID); err != nil {
		return &CliError{Kind: "config", What: "delete checkpoint", Cause: err}
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
// formats are JSON, Markdown, and HTML. The renderer is
// the report package; the bridge from audit rows to
// report.Report lives in report_bridge.go.
//
// The audit log is the canonical source of truth; the
// report is the human-readable projection. When the
// audit log is empty, the report is empty too.
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
	rows, err := deps.Audit.GetByTask(taskID)
	if err != nil {
		return &CliError{Kind: "audit", What: "read", Cause: err}
	}
	rep := bridgeReportFromRows(taskID, rows)
	return renderReport(deps.Format, rep, deps.Out)
}

// renderReport runs the named renderer. The function is
// the single point that knows the format names; callers
// pass a free-form string and a valid render comes back.
func renderReport(format string, rep *report.Report, out io.Writer) error {
	if rep == nil {
		return &CliError{Kind: "config", What: "report: nil report"}
	}
	if format == "" {
		format = "json"
	}
	var r report.Renderer
	switch format {
	case "json":
		r = report.NewJSONRenderer()
	case "markdown", "md":
		r = report.NewMarkdownRenderer()
	case "html":
		r = report.NewHTMLRenderer()
	default:
		return &CliError{Kind: "user", What: "unknown report format: " + format}
	}
	s, err := r.Render(rep)
	if err != nil {
		return &CliError{Kind: "report", What: "render " + format, Cause: err}
	}
	fmt.Fprintln(out, s)
	return nil
}

// AuditDeps is the wiring the audit subcommand needs.
type AuditDeps struct {
	Audit *audit.Log
	Out   io.Writer
	Tail  bool
}

// Audit prints the audit log. Per REQ-CLI-AUD-001 the
// `--tail` flag follows new rows. Tail is a no-op in this
// revision (the audit log does not yet expose a blocking
// read); it is wired so the CLI contract matches the doc
// set.
//
// When taskID is empty, the function pulls the last 1000
// rows of the well-known kinds and unions them. The audit
// log's primary index is task-scoped; a "show me every
// row" query would require a new method on the audit log
// (REQ-CLI-OPEN-001 in the open-questions list). For now
// the iteration is bounded and deterministic.
func Audit(taskID string, deps AuditDeps) error {
	if deps.Audit == nil {
		return &CliError{Kind: "config", What: "audit log is not configured"}
	}
	if deps.Out == nil {
		return &CliError{Kind: "config", What: "audit: out is nil"}
	}
	if deps.Tail {
		// Tail is a no-op in this revision (the audit
		// log does not yet expose a blocking read);
		// the contract is wired for the follow-up
		// implementation. Without a tail loop the
		// function returns success immediately.
		return nil
	}
	var rows []audit.PersistedRow
	if taskID != "" {
		r, err := deps.Audit.GetByTask(taskID)
		if err != nil {
			return &CliError{Kind: "audit", What: "read", Cause: err}
		}
		rows = r
	} else {
		// Union across the well-known kinds. Each call
		// to GetByKind returns up to 1000 rows; the union
		// is sorted in SQL order.
		seen := make(map[int64]bool)
		kinds := []string{
			"tool_call", "approval_requested", "approval_granted",
			"approval_denied", "approval_expired", "task_started",
			"task_passed", "task_failed", "plan_dispatched",
		}
		for _, k := range kinds {
			r, err := deps.Audit.GetByKind(k, 1000)
			if err != nil {
				return &CliError{Kind: "audit", What: "read kind " + k, Cause: err}
			}
			for _, row := range r {
				if !seen[row.SequenceNumber] {
					seen[row.SequenceNumber] = true
					rows = append(rows, row)
				}
			}
		}
		// Sort by sequence number (insertion order from
		// the kinds is not guaranteed across calls).
		sortBySequence(rows)
	}
	for _, r := range rows {
		fmt.Fprintf(deps.Out, "%d\t%s\t%s\t%s\n", r.SequenceNumber, r.TaskID, r.Kind, r.ContentHash)
	}
	_ = deps.Tail // tail is a no-op today
	return nil
}

// sortBySequence is a small helper that sorts persisted rows
// by sequence number ascending. We avoid sort.Slice here to
// keep the function tiny and obvious.
func sortBySequence(rows []audit.PersistedRow) {
	for i := 1; i < len(rows); i++ {
		for j := i; j > 0 && rows[j-1].SequenceNumber > rows[j].SequenceNumber; j-- {
			rows[j-1], rows[j] = rows[j], rows[j-1]
		}
	}
}

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
	meter := runtime.NewBudgetMeter(deps.Budget)
	o := runtime.New(runtime.Config{
		Worker:   deps.Worker,
		Verifier: deps.Verify,
		Meter:    meter,
		Audit:    deps.Audit,
		Resume:   deps.Resume,
	})
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
			return report, wfa
		}
		ok, appErr := deps.Approve.Approve(ctx, wfa)
		if appErr != nil {
			return report, &CliError{Kind: "approval", What: caller + ": approval handler failed", Cause: appErr}
		}
		if !ok {
			return report, &CliError{Kind: "approval", What: caller + ": operator denied the action"}
		}
		next, cpErr := deps.Resume.Load(plan.ID)
		if cpErr != nil {
			return report, &CliError{Kind: "resume", What: "load checkpoint", Cause: cpErr}
		}
		cp = next
	}
	return nil, &CliError{Kind: "approval", What: "too many approval loops (>100)"}
}
