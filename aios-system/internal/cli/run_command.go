package cli

import (
	"context"
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/Hamidullah02/aios/aios-system/internal/core/audit"
	"github.com/Hamidullah02/aios/aios-system/internal/core/schemas"
	"github.com/Hamidullah02/aios/aios-system/internal/policy"
	"github.com/Hamidullah02/aios/aios-system/internal/runtime"
)

// ApprovalHandler is the hook Run uses when the orchestrator
// raises *runtime.WaitingForApproval. The default handler
// (in NewDaemonApprovalHandler) talks to the Unix-socket
// approval daemon. Tests substitute a fake that records the
// call and returns a scripted decision.
type ApprovalHandler interface {
	Approve(ctx context.Context, wfa *runtime.WaitingForApproval) (approved bool, err error)
}

// RunDeps is the wiring the run subcommand needs. Fields are
// filled by the caller (the Main entry point builds the
// production deps; tests build fakes).
type RunDeps struct {
	Audit    *audit.Log
	Policy   *policy.Gateway
	Resume   *runtime.ResumeHandler
	Worker   runtime.Worker
	Verifier runtime.Verifier
	Budget   runtime.Budget
	// Grants records approvals given during this run and is
	// what the bus consults so a retried task does not
	// re-prompt for a call the operator already answered.
	// nil is safe and reproduces the previous behaviour.
	Grants       *grantStore
	Approval     ApprovalHandler
	ReportWriter func(*runtime.Report) error
	// Observer is optional. v0.41.1: when set, it receives
	// task lifecycle events for the duration of the run.
	// `aios run --web` uses this to feed the live UI.
	Observer       runtime.Observer
	Stdout, Stderr interface {
		Write([]byte) (int, error)
	}
}

// LoadPlan reads the plan YAML and returns a *schemas.TaskPlan.
// The plan file is the doc set's TaskPlan shape (id, goal,
// tasks, tools, model, budget). We accept the same key set
// the schemas package already understands, plus the
// `approval_required` flag the doc set names in
// REQ-CLI-RUN-001 (the flag is informational; the runtime
// raises WaitingForApproval based on the policy, not on the
// flag).
func LoadPlan(path string) (*schemas.TaskPlan, error) {
	if path == "" {
		return nil, &PlanLoadError{Path: path, Err: errors.New("path is empty")}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, &PlanLoadError{Path: path, Err: err}
	}
	var plan schemas.TaskPlan
	if err := yaml.Unmarshal(data, &plan); err != nil {
		return nil, &PlanLoadError{Path: path, Err: err}
	}
	if err := plan.Validate(); err != nil {
		return nil, &PlanLoadError{Path: path, Err: err}
	}
	return &plan, nil
}

// Run executes the plan. The function returns the final
// Report and an error. The error is *runtime.WaitingForApproval
// when the worker raised it AND the approval handler could not
// resolve it; the CLI catches that and forwards the request to
// the daemon.
//
// Per REQ-CLI-RUN-001 the plan is loaded from a file. Per
// REQ-CLI-RUN-002 the orchestrator is started. Per
// REQ-CLI-RUN-003 the WaitingForApproval error is handled
// by asking the approval handler; the orchestrator is then
// resumed with a checkpoint. Per REQ-CLI-RUN-004 the report
// is printed at the end.
func Run(ctx context.Context, args *Args, deps RunDeps) (*runtime.Report, error) {
	if args == nil {
		return nil, &CliError{Kind: "user", What: "args is nil"}
	}
	if args.PlanPath == "" {
		return nil, &CliError{Kind: "user", What: "plan path is required"}
	}
	plan, err := LoadPlan(args.PlanPath)
	if err != nil {
		return nil, err
	}
	if deps.Worker == nil {
		return nil, &CliError{Kind: "config", What: "run: worker is not configured"}
	}
	if deps.Verifier == nil {
		return nil, &CliError{Kind: "config", What: "run: verifier is not configured"}
	}
	if deps.Resume == nil {
		return nil, &CliError{Kind: "config", What: "run: resume handler is not configured"}
	}
	if deps.Audit == nil {
		return nil, &CliError{Kind: "config", What: "run: audit log is not configured"}
	}
	// Delegate to the shared approval loop.
	report, err := runWithApprovals(ctx, plan, nil, ResumeDeps{
		Audit:      deps.Audit,
		Resume:     deps.Resume,
		Worker:     deps.Worker,
		Verify:     deps.Verifier,
		Budget:     deps.Budget,
		Approve:    deps.Approval,
		GrantStore: deps.Grants,
		Observer:   deps.Observer,
	}, "run")
	if err == nil && deps.ReportWriter != nil {
		if wErr := deps.ReportWriter(report); wErr != nil {
			return report, &CliError{Kind: "report", What: "write report", Cause: wErr}
		}
	}
	return report, err
}

// PrintReport writes a one-line summary of the report. The
// full report is also available in JSON; PrintReport is the
// CLI's terse output (the audit log is the canonical record).
func PrintReport(w interface{ Write([]byte) (int, error) }, r *runtime.Report) {
	if w == nil || r == nil {
		return
	}
	fmt.Fprintf(w, "Plan:      %s\n", r.PlanID)
	fmt.Fprintf(w, "Tasks:     %d\n", r.TaskCount)
	fmt.Fprintf(w, "From:      %d\n", r.FromIndex)
	fmt.Fprintf(w, "Started:   %s\n", r.StartedAt)
	fmt.Fprintf(w, "Finished:  %s\n", r.FinishedAt)
	fmt.Fprintf(w, "Stopped:   %s\n", r.StoppedReason)
	fmt.Fprintf(w, "Budget:    cost=%.4f tool=%d model=%d net=%d\n",
		r.Budget.CostUSD, r.Budget.ToolCalls, r.Budget.ModelCalls, r.Budget.NetworkRequests)
}
